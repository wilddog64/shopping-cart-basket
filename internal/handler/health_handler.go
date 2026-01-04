package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	redisClient *redis.Client
	version     string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(redisClient *redis.Client, version string) *HealthHandler {
	return &HealthHandler{
		redisClient: redisClient,
		version:     version,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version,omitempty"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Health handles GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	checks := make(map[string]string)
	status := "healthy"

	// Check Redis
	if h.redisClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "error: " + err.Error()
			status = "unhealthy"
		} else {
			checks["redis"] = "ok"
		}
	}

	statusCode := http.StatusOK
	if status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:  status,
		Version: h.version,
		Checks:  checks,
	})
}

// Liveness handles GET /health/live
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "alive",
	})
}

// Readiness handles GET /health/ready
func (h *HealthHandler) Readiness(c *gin.Context) {
	checks := make(map[string]string)
	ready := true

	// Check Redis
	if h.redisClient != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "not ready: " + err.Error()
			ready = false
		} else {
			checks["redis"] = "ready"
		}
	} else {
		checks["redis"] = "not configured"
		ready = false
	}

	status := "ready"
	statusCode := http.StatusOK
	if !ready {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status: status,
		Checks: checks,
	})
}
