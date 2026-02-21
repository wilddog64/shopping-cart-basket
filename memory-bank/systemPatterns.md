# System Patterns: Basket Service

## Architectural Pattern: Layered / Hexagonal

The service enforces a strict dependency direction: Handler → Service → Repository. No layer reaches backwards. External concerns (Redis, RabbitMQ) are abstracted behind interfaces.

```
HTTP Client
    │
    ▼
┌─────────────────────────────────────┐
│  Handler Layer (internal/handler/)  │
│  - Auth middleware                  │
│  - Request validation (Gin binding) │
│  - Response serialization           │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│  Service Layer (internal/service/)  │
│  - CartService                      │
│  - Business rules                   │
│  - Event publishing coordination    │
└──────────┬──────────────┬───────────┘
           │              │
           ▼              ▼
┌──────────────┐  ┌───────────────────┐
│  Repository  │  │  EventPublisher   │
│  (Redis)     │  │  (RabbitMQ)       │
└──────────────┘  └───────────────────┘
```

## Key Abstractions (Interfaces)

### CartRepository (`internal/repository/`)
```go
type CartRepository interface {
    Get(ctx context.Context, customerID string) (*model.Cart, error)
    Save(ctx context.Context, cart *model.Cart) error
    Delete(ctx context.Context, customerID string) error
}
```
Sentinel error: `repository.ErrCartNotFound`

### EventPublisher (`internal/service/`)
```go
type EventPublisher interface {
    Publish(ctx context.Context, event *model.EventEnvelope) error
}
```

Both interfaces allow unit testing without external infrastructure.

## Domain Model Design

### Cart
```go
type Cart struct {
    ID          string     // UUID
    CustomerID  string     // JWT sub claim — one cart per customer
    Items       []CartItem
    TotalAmount float64    // always recalculated, never stored stale
    Currency    string     // default "USD"
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ExpiresAt   time.Time  // CreatedAt + CartTTL
}
```

### CartItem
```go
type CartItem struct {
    ID        string  // UUID assigned on add
    ProductID string  // reference only, no FK validation
    Name      string
    Quantity  int
    UnitPrice float64
    SubTotal  float64 // = Quantity * UnitPrice, always recalculated
}
```

Business rules enforced in model methods:
- `Cart.AddItem()` — merges if same ProductID already exists
- `Cart.UpdateItemQuantity()` — removes item if quantity <= 0
- `Cart.recalculateTotal()` — called after every mutation (private)
- `Cart.IsEmpty()` — used by service to block checkout on empty cart

## Event-Driven Design

Events are published to RabbitMQ exchange `events` with routing keys prefixed `cart.*`.

### Event Lifecycle
- **Cart Created**: On first `GetCart` when no cart exists in Redis
- **Cart Updated**: On `AddItem`, `UpdateItemQuantity`, `RemoveItem` (sub-type in data payload)
- **Cart Cleared**: On `ClearCart`
- **Cart Checkout**: On `Checkout` — this is the critical event; failure blocks the operation

### Event Envelope (common across all platform services)
```json
{
  "id": "uuid",
  "type": "cart.checkout",
  "version": "1.0",
  "timestamp": "ISO8601",
  "source": "cart-service",
  "correlationId": "uuid",
  "data": { ... }
}
```

### Failure Handling Strategy
- Non-critical events (created, updated, cleared): log warning, continue successfully
- Critical event (checkout): return error to client on publish failure

## Configuration Pattern

All configuration loaded at startup from environment variables via `config.Load()`. The `Config` struct is injected into handlers and services via constructors. No global config singletons. Sensible defaults for all values allow running locally without any environment setup beyond Redis.

```go
cfg := config.Load()
// cfg.RedisAddr() → "host:port"
// cfg.RabbitMQAddr() → "host:port"
```

## Authentication Pattern

Two modes controlled by `OAUTH2_ENABLED`:
1. **Production** (`OAUTH2_ENABLED=true`): JWT Bearer token validated via Keycloak JWKS endpoint; `sub` claim used as `customerID`
2. **Development** (`OAUTH2_ENABLED=false`): `X-User-ID` header used directly as `customerID`

## Middleware Stack (applied in order)

1. Correlation ID injection (generates UUID if not provided via `X-Correlation-ID`)
2. Structured request logging (zap)
3. Auth middleware (JWT or X-User-ID)
4. Rate limiting (per-IP token bucket, default 100 RPS / burst 50)
5. Security headers

## Redis Storage Pattern

- Key: `cart:{customerID}` (one cart per customer)
- Value: JSON-serialized `Cart` struct
- TTL: set on every `Save()` call; resets on any mutation (sliding expiration)
- No secondary indexes; cart is always accessed by customer ID from JWT

## Observability

- **Logging**: JSON-structured via zap; includes `cartId`, `customerId`, `productId`, operation context
- **Metrics**: Prometheus at `/metrics`; `http_requests_total`, `http_request_duration_seconds` plus custom cart operation counters
- **Health**: `/health` (Redis ping), `/health/live` (process alive), `/health/ready` (Redis ready)
- **Correlation**: `X-Correlation-ID` threaded through all log lines and events
