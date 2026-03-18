# Issue: Redis Has No Circuit Breaker

**Date:** 2026-03-18
**Status:** Open
**Branch:** `docs/next-improvements`

## Problem

Redis is in the hot path of every cart operation. When Redis is down or degraded:

- Every request fails with `500 Internal Server Error` immediately
- If Redis is _hanging_ (not refusing), requests block for up to 15s (HTTP write timeout) before failing
- Under load, a hanging Redis causes thread exhaustion — cascading failure
- The checkout flow has a partial-failure risk: RabbitMQ event published, Redis save fails → duplicate order on retry
- `/health/live` always returns 200 — Kubernetes never restarts the pod even when all requests are failing

## Fix: Add Circuit Breaker around Redis operations

Use `github.com/sony/gobreaker` — wraps any `func() (interface{}, error)` call.

### 1. Add dependency

```bash
go get github.com/sony/gobreaker
```

### 2. Wrap RedisCartRepository

In `internal/repository/cart_repository.go`:

```go
import "github.com/sony/gobreaker"

type RedisCartRepository struct {
    client  *redis.Client
    breaker *gobreaker.CircuitBreaker
}

func NewRedisCartRepository(...) (*RedisCartRepository, error) {
    // ... existing client setup ...

    cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        "redis",
        MaxRequests: 3,           // half-open: allow 3 probes before closing
        Interval:    10 * time.Second, // reset counts every 10s in closed state
        Timeout:     30 * time.Second, // stay open for 30s before half-open probe
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures >= 5
        },
        OnStateChange: func(name string, from, to gobreaker.State) {
            logger.Warn("redis circuit breaker state change",
                zap.String("from", from.String()),
                zap.String("to", to.String()))
        },
    })

    return &RedisCartRepository{client: client, breaker: cb}, nil
}
```

### 3. Wrap each repository operation

```go
func (r *RedisCartRepository) Get(ctx context.Context, customerID string) (*model.Cart, error) {
    result, err := r.breaker.Execute(func() (interface{}, error) {
        data, err := r.client.Get(ctx, cartKey(customerID)).Bytes()
        if err == redis.Nil {
            return nil, ErrCartNotFound
        }
        return data, err
    })
    if err != nil {
        if errors.Is(err, gobreaker.ErrOpenState) {
            return nil, ErrRedisUnavailable // caller returns 503, not 500
        }
        return nil, fmt.Errorf("failed to get cart: %w", err)
    }
    // unmarshal result.([]byte) ...
}
```

Repeat for `Save`, `Delete`, `Exists`.

### 4. Add ErrRedisUnavailable sentinel

In `internal/repository/errors.go` (or alongside `ErrCartNotFound`):

```go
var ErrRedisUnavailable = errors.New("redis unavailable")
```

### 5. Return 503 in handler when circuit is open

In `internal/handler/cart_handler.go`:

```go
if errors.Is(err, repository.ErrRedisUnavailable) {
    response.ServiceUnavailable(c, "Cart service temporarily unavailable")
    return
}
```

### 6. Fix /health/live to reflect circuit state

In `internal/handler/health_handler.go`, expose circuit state:

```go
func (h *HealthHandler) Live(c *gin.Context) {
    if h.repo.CircuitOpen() {
        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "circuit open"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "alive"})
}
```

## Definition of Done

- [ ] `gobreaker` added to `go.mod`
- [ ] `RedisCartRepository` wraps all 4 operations (`Get`, `Save`, `Delete`, `Exists`) in circuit breaker
- [ ] `ErrRedisUnavailable` sentinel defined
- [ ] Handlers return `503` (not `500`) when circuit is open
- [ ] `/health/live` returns `503` when circuit is open
- [ ] Unit tests: circuit opens after 5 consecutive failures; returns `ErrRedisUnavailable` while open; closes after probe succeeds
- [ ] No changes to Dockerfile, k8s manifests, or other services

## What NOT to Do

- Do NOT add retry logic inside the circuit breaker — retries belong at the caller (e.g. frontend), not here
- Do NOT change checkout atomicity — that is a separate saga/outbox issue
- Do NOT modify RabbitMQ integration
