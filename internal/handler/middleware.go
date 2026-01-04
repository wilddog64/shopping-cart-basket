package handler

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/user/shopping-cart-basket/internal/auth"
	"github.com/user/shopping-cart-basket/internal/service"
	"go.uber.org/zap"
)

// RequestLogger returns a middleware that logs requests
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
		}

		if query != "" {
			fields = append(fields, zap.String("query", query))
		}

		if correlationID := c.GetHeader("X-Correlation-ID"); correlationID != "" {
			fields = append(fields, zap.String("correlationId", correlationID))
		}

		if status >= 500 {
			logger.Error("request completed", fields...)
		} else if status >= 400 {
			logger.Warn("request completed", fields...)
		} else {
			logger.Info("request completed", fields...)
		}
	}
}

// CorrelationID returns a middleware that handles correlation IDs
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		c.Set("correlationID", correlationID)
		c.Header("X-Correlation-ID", correlationID)

		// Set correlation ID in context for service layer
		ctx := service.SetCorrelationID(c.Request.Context(), correlationID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// SecurityHeaders returns a middleware that adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS - only in production
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// Recovery returns a middleware that recovers from panics
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An unexpected error occurred",
					},
				})
			}
		}()
		c.Next()
	}
}

// AuthMiddleware returns a middleware that validates JWT tokens
func AuthMiddleware(jwtValidator *auth.JWTValidator, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header required",
				},
			})
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
			})
			return
		}

		token := parts[1]

		// Validate token
		claims, err := jwtValidator.ValidateToken(c.Request.Context(), token)
		if err != nil {
			logger.Warn("token validation failed",
				zap.Error(err),
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			return
		}

		// Set customer ID from claims
		SetCustomerID(c, claims.Subject)
		c.Set("claims", claims)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// MockAuthMiddleware returns a middleware for development without OAuth2
func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use X-User-ID header or default to "dev-user"
		customerID := c.GetHeader("X-User-ID")
		if customerID == "" {
			customerID = "dev-user"
		}

		SetCustomerID(c, customerID)
		c.Set("roles", []string{"cart-user"})

		c.Next()
	}
}

// RequireRole returns a middleware that checks for required roles
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Access denied",
				},
			})
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Invalid roles format",
				},
			})
			return
		}

		// Check if user has any of the required roles
		for _, required := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == required {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Insufficient permissions",
			},
		})
	}
}

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rps      int
	burst    int
}

type visitor struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rps:      rps,
		burst:    burst,
	}

	// Clean up old visitors periodically
	go func() {
		for {
			time.Sleep(time.Minute)
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.visitors {
		if time.Since(v.lastCheck) > time.Minute {
			delete(rl.visitors, ip)
		}
	}
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:    rl.burst - 1,
			lastCheck: now,
		}
		return true
	}

	// Add tokens based on time passed
	elapsed := now.Sub(v.lastCheck)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rps))
	v.tokens += tokensToAdd
	if v.tokens > rl.burst {
		v.tokens = rl.burst
	}
	v.lastCheck = now

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// Middleware returns the rate limiter middleware
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.allow(ip) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests. Please try again later.",
				},
			})
			return
		}

		c.Next()
	}
}
