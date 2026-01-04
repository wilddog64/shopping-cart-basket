package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/user/shopping-cart-basket/internal/auth"
	"github.com/user/shopping-cart-basket/internal/config"
	"github.com/user/shopping-cart-basket/internal/event"
	"github.com/user/shopping-cart-basket/internal/handler"
	"github.com/user/shopping-cart-basket/internal/repository"
	"github.com/user/shopping-cart-basket/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting cart service",
		zap.String("version", version),
		zap.String("port", cfg.ServerPort),
	)

	// Initialize Redis repository
	repo, err := repository.NewRedisCartRepository(
		cfg.RedisAddr(),
		cfg.RedisPassword,
		cfg.RedisDB,
		cfg.CartTTL,
	)
	if err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	defer repo.Close()

	logger.Info("connected to Redis", zap.String("addr", cfg.RedisAddr()))

	// Initialize event publisher
	publisher := event.NewPublisher("events", logger, cfg.RabbitMQHost != "")

	// Initialize service
	cartService := service.NewCartService(repo, publisher, cfg.CartTTL, logger)

	// Initialize handlers
	cartHandler := handler.NewCartHandler(cartService, logger)
	healthHandler := handler.NewHealthHandler(repo.Client(), version)

	// Initialize rate limiter
	rateLimiter := handler.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Initialize JWT validator (if OAuth2 is enabled)
	var jwtValidator *auth.JWTValidator
	if cfg.OAuth2Enabled {
		jwtValidator = auth.NewJWTValidator(cfg.OAuth2IssuerURI, cfg.OAuth2ClientID, logger)
		logger.Info("OAuth2 authentication enabled",
			zap.String("issuer", cfg.OAuth2IssuerURI),
		)
	} else {
		logger.Warn("OAuth2 authentication disabled - using mock auth")
	}

	// Setup router
	router := setupRouter(cfg, cartHandler, healthHandler, jwtValidator, rateLimiter, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	// Close publisher
	if err := publisher.Close(); err != nil {
		logger.Error("failed to close publisher", zap.Error(err))
	}

	logger.Info("server stopped")
}

func initLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	return logger
}

func setupRouter(
	cfg *config.Config,
	cartHandler *handler.CartHandler,
	healthHandler *handler.HealthHandler,
	jwtValidator *auth.JWTValidator,
	rateLimiter *handler.RateLimiter,
	logger *zap.Logger,
) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(handler.Recovery(logger))
	router.Use(handler.RequestLogger(logger))
	router.Use(handler.CorrelationID())
	router.Use(handler.SecurityHeaders())

	// Health endpoints (no auth, no rate limit)
	router.GET("/health", healthHandler.Health)
	router.GET("/health/live", healthHandler.Liveness)
	router.GET("/health/ready", healthHandler.Readiness)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes with rate limiting
	api := router.Group("/api/v1")
	api.Use(rateLimiter.Middleware())

	// Apply authentication middleware
	if cfg.OAuth2Enabled && jwtValidator != nil {
		api.Use(handler.AuthMiddleware(jwtValidator, logger))
	} else {
		api.Use(handler.MockAuthMiddleware())
	}

	// Cart endpoints
	cart := api.Group("/cart")
	{
		cart.GET("", cartHandler.GetCart)
		cart.POST("/items", cartHandler.AddItem)
		cart.PUT("/items/:itemId", cartHandler.UpdateItem)
		cart.DELETE("/items/:itemId", cartHandler.RemoveItem)
		cart.DELETE("", cartHandler.ClearCart)
		cart.POST("/checkout", cartHandler.Checkout)
	}

	return router
}
