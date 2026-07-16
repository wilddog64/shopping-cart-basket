# Feature: Guest cart (Amazon-style) — persist 3 days, validate identity at checkout

**Primary repo:** `shopping-cart-basket` (Go backend) — the bulk of this work.
**Second repo:** `shopping-cart-frontend` (React) — persist the guest token + merge on login.
**Branch (both repos):** `feat/guest-cart`

---

## Problem (verified live on `ubuntu-hostinger`, 2026-07-15)

Two real defects behind "items added to the cart do not persist":

1. **Guests cannot use the cart at all.** Every `/api/v1/cart` route sits behind
   `AuthMiddleware` (`cmd/server/main.go:200`), which returns **401** without a valid
   Keycloak JWT. An anonymous shopper can never store a cart, so nothing persists.
   *(Confirmed live: `GET /api/v1/cart` → 401 without a bearer token.)*

2. **Cart expiry is fixed, not rolling.** `Cart.ExpiresAt` is set once in
   `model.NewCart` (`internal/model/cart.go:42`) and **never extended** on later writes.
   `AddItem`/`UpdateItemQuantity`/`RemoveItem` bump `UpdatedAt` but not `ExpiresAt`, and the
   Redis TTL is derived from `ExpiresAt` (`internal/repository/cart_repository.go:90`). A cart
   created 7 days ago still dies at day 7 even if the shopper touched it a minute ago.

**Goal (Amazon behavior):** an anonymous guest can add to the cart; the cart persists on a
**rolling 3-day** window; identity is required only at **checkout**; and on login the guest cart
**merges** into the user's cart.

---

## Design

- **Guest identity = signed opaque token.** A guest is `guest-<uuid>`. The server hands the
  client a token `"<guestID>.<base64(HMAC-SHA256(guestID, secret))>"` in the `X-Cart-Token`
  response header. The client stores it and echoes it on later requests. HMAC makes it
  unforgeable — a guest cannot read another guest's cart by guessing an id.
- **One middleware, two identities.** `GuestOrAuthMiddleware` replaces `AuthMiddleware` on the
  cart group: a valid JWT → authenticated user (`isGuest=false`); otherwise a validated/newly
  minted guest (`isGuest=true`). Redis keys are already namespaced by customerID
  (`cart:<id>`), so `cart:guest-<uuid>` vs `cart:<sub>` needs no key changes.
- **Rolling TTL, per identity.** Every mutating save sets `ExpiresAt = now + ttlFor(id)`.
  Guests get `GUEST_CART_TTL` (default **72h / 3 days**); authenticated users keep
  `CART_TTL` (168h / 7 days).
- **Checkout requires auth.** `Checkout` rejects guests with 401 — identity is validated here.
- **Merge on login.** `POST /api/v1/cart/merge` (auth required, `X-Cart-Token` carries the guest
  token): guest items are folded into the user cart by productID, the guest cart is deleted.

---

## Before You Start

- `hostname && uname -n`
- In `shopping-cart-basket`: `git checkout feat/guest-cart && git pull origin feat/guest-cart`
- Read `CLAUDE.md`, `memory-bank/activeContext.md`, `memory-bank/progress.md`.
- Read the files you will edit: `cmd/server/main.go`, `internal/config/config.go`,
  `internal/handler/middleware.go`, `internal/handler/cart_handler.go`,
  `internal/service/cart_service.go`, `internal/auth/jwt.go` (for the `auth` package shape).
- Baseline: `go build ./... && go vet ./... && go test ./...` before you start.

---

## Backend changes — `shopping-cart-basket`

### 1. `internal/config/config.go` — add guest settings

**Old block (struct, the `// Cart settings` region):**
```go
	// Cart settings
	CartTTL time.Duration
```
**New block:**
```go
	// Cart settings
	CartTTL          time.Duration
	GuestCartTTL     time.Duration
	GuestTokenSecret string
```

**Old block (loader, the `// Cart` region):**
```go
		// Cart
		CartTTL: getEnvAsDuration("CART_TTL", 168*time.Hour), // 7 days
```
**New block:**
```go
		// Cart
		CartTTL:          getEnvAsDuration("CART_TTL", 168*time.Hour),    // 7 days
		GuestCartTTL:     getEnvAsDuration("GUEST_CART_TTL", 72*time.Hour), // 3 days
		GuestTokenSecret: getEnv("GUEST_TOKEN_SECRET", ""),
```

### 2. NEW `internal/auth/guest.go` — mint/verify guest tokens

```go
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GuestTokenManager mints and validates opaque, HMAC-signed guest cart tokens.
type GuestTokenManager struct {
	secret []byte
}

// NewGuestTokenManager creates a manager with the given signing secret.
func NewGuestTokenManager(secret string) *GuestTokenManager {
	return &GuestTokenManager{secret: []byte(secret)}
}

// NewGuestID returns a new opaque guest identity, e.g. "guest-<uuid>".
func (m *GuestTokenManager) NewGuestID() string {
	return "guest-" + uuid.New().String()
}

// Sign returns "<guestID>.<base64url(hmac)>".
func (m *GuestTokenManager) Sign(guestID string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(guestID))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return guestID + "." + sig
}

// Verify parses a token and returns the guestID only if the signature is valid.
func (m *GuestTokenManager) Verify(token string) (string, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("malformed guest token")
	}
	guestID := parts[0]
	if !strings.HasPrefix(guestID, "guest-") {
		return "", fmt.Errorf("invalid guest id")
	}
	expected := m.Sign(guestID)
	if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
		return "", fmt.Errorf("invalid guest token signature")
	}
	return guestID, nil
}
```

Add `internal/auth/guest_test.go` covering: `Sign`→`Verify` round-trips; a tampered
signature fails; a token signed with a different secret fails; a non-`guest-` id fails.

### 3. `internal/handler/middleware.go` — add `GuestOrAuthMiddleware` + `isGuest`

Add these to the file (leave the existing `AuthMiddleware`/`MockAuthMiddleware` in place):

```go
// Context key for the guest flag
const isGuestKey = "isGuest"

// isGuest reports whether the current request is an anonymous guest.
func isGuest(c *gin.Context) bool {
	if v, ok := c.Get(isGuestKey); ok {
		if b, ok2 := v.(bool); ok2 {
			return b
		}
	}
	return false
}

// GuestOrAuthMiddleware admits either an authenticated user (valid JWT) or an
// anonymous guest identified by a signed guest token. Guests arriving without a
// valid token are issued a fresh one via the X-Cart-Token response header.
func GuestOrAuthMiddleware(jwtValidator *auth.JWTValidator, guestTokens *auth.GuestTokenManager, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization header format"},
				})
				return
			}
			claims, err := jwtValidator.ValidateToken(c.Request.Context(), parts[1])
			if err != nil {
				logger.Warn("token validation failed", zap.Error(err), zap.String("ip", c.ClientIP()))
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"},
				})
				return
			}
			SetCustomerID(c, claims.Subject)
			c.Set("claims", claims)
			c.Set("roles", claims.Roles)
			c.Set(isGuestKey, false)
			c.Next()
			return
		}

		// No Authorization header -> guest path.
		var guestID string
		if token := c.GetHeader("X-Cart-Token"); token != "" {
			if id, err := guestTokens.Verify(token); err == nil {
				guestID = id
			}
		}
		if guestID == "" {
			guestID = guestTokens.NewGuestID()
		}
		c.Header("X-Cart-Token", guestTokens.Sign(guestID))
		SetCustomerID(c, guestID)
		c.Set("roles", []string{"cart-guest"})
		c.Set(isGuestKey, true)
		c.Next()
	}
}
```

### 4. `internal/handler/cart_handler.go` — hold the token manager, guard checkout, add merge

**Old block (struct + constructor):**
```go
// CartHandler handles cart HTTP requests
type CartHandler struct {
	service *service.CartService
	logger  *zap.Logger
}

// NewCartHandler creates a new cart handler
func NewCartHandler(service *service.CartService, logger *zap.Logger) *CartHandler {
	return &CartHandler{
		service: service,
		logger:  logger,
	}
}
```
**New block:**
```go
// CartHandler handles cart HTTP requests
type CartHandler struct {
	service     *service.CartService
	guestTokens *auth.GuestTokenManager
	logger      *zap.Logger
}

// NewCartHandler creates a new cart handler
func NewCartHandler(service *service.CartService, guestTokens *auth.GuestTokenManager, logger *zap.Logger) *CartHandler {
	return &CartHandler{
		service:     service,
		guestTokens: guestTokens,
		logger:      logger,
	}
}
```
Add the `auth` import: `"github.com/user/shopping-cart-basket/internal/auth"`.

**Guard checkout — old block (top of `Checkout`):**
```go
func (h *CartHandler) Checkout(c *gin.Context) {
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}
```
**New block:**
```go
func (h *CartHandler) Checkout(c *gin.Context) {
	if isGuest(c) {
		response.Unauthorized(c, "Authentication required to checkout")
		return
	}
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}
```

**Add the merge handler** (place after `Checkout`):
```go
// MergeGuestCart handles POST /api/v1/cart/merge — folds the guest cart identified
// by the X-Cart-Token header into the authenticated user's cart, then deletes it.
func (h *CartHandler) MergeGuestCart(c *gin.Context) {
	if isGuest(c) {
		response.Unauthorized(c, "Authentication required to merge cart")
		return
	}
	customerID := getCustomerID(c)
	if customerID == "" {
		response.Unauthorized(c, "Customer ID not found")
		return
	}

	guestToken := c.GetHeader("X-Cart-Token")
	if guestToken == "" {
		cart, err := h.service.GetCart(c.Request.Context(), customerID)
		if err != nil {
			response.InternalError(c, "Failed to get cart")
			return
		}
		response.Success(c, cart.ToResponse())
		return
	}

	guestID, err := h.guestTokens.Verify(guestToken)
	if err != nil {
		response.BadRequest(c, "Invalid guest token")
		return
	}

	cart, err := h.service.MergeGuestCart(c.Request.Context(), guestID, customerID)
	if err != nil {
		h.logger.Error("failed to merge guest cart",
			zap.String("customerId", customerID),
			zap.String("guestId", guestID),
			zap.Error(err),
		)
		response.InternalError(c, "Failed to merge cart")
		return
	}

	response.Success(c, cart.ToResponse())
}
```

### 5. `internal/service/cart_service.go` — guest TTL, rolling save, merge

**Old block (struct + constructor):**
```go
// CartService handles cart business logic
type CartService struct {
	repo      repository.CartRepository
	publisher EventPublisher
	cartTTL   time.Duration
	logger    *zap.Logger
}

// NewCartService creates a new cart service
func NewCartService(repo repository.CartRepository, publisher EventPublisher, cartTTL time.Duration, logger *zap.Logger) *CartService {
	return &CartService{
		repo:      repo,
		publisher: publisher,
		cartTTL:   cartTTL,
		logger:    logger,
	}
}
```
**New block:**
```go
// CartService handles cart business logic
type CartService struct {
	repo         repository.CartRepository
	publisher    EventPublisher
	cartTTL      time.Duration
	guestCartTTL time.Duration
	logger       *zap.Logger
}

// NewCartService creates a new cart service
func NewCartService(repo repository.CartRepository, publisher EventPublisher, cartTTL, guestCartTTL time.Duration, logger *zap.Logger) *CartService {
	return &CartService{
		repo:         repo,
		publisher:    publisher,
		cartTTL:      cartTTL,
		guestCartTTL: guestCartTTL,
		logger:       logger,
	}
}

// ttlFor returns the cart TTL for an identity: guests get the shorter guest TTL,
// authenticated users the standard TTL.
func (s *CartService) ttlFor(customerID string) time.Duration {
	if strings.HasPrefix(customerID, "guest-") {
		return s.guestCartTTL
	}
	return s.cartTTL
}

// saveRolling persists the cart and resets its expiry to now + the identity's TTL,
// giving a rolling window on every write.
func (s *CartService) saveRolling(ctx context.Context, cart *model.Cart, customerID string) error {
	cart.ExpiresAt = time.Now().Add(s.ttlFor(customerID))
	return s.repo.Save(ctx, cart)
}
```
Add `"strings"` to the import block.

**Rolling expiry on create — old block (inside `GetCart`):**
```go
			// Create a new cart
			cart = model.NewCart(customerID, s.cartTTL)
			if err := s.repo.Save(ctx, cart); err != nil {
				return nil, err
			}
```
**New block:**
```go
			// Create a new cart
			cart = model.NewCart(customerID, s.ttlFor(customerID))
			if err := s.repo.Save(ctx, cart); err != nil {
				return nil, err
			}
```

**Rolling save on mutation** — in `AddItem`, `UpdateItemQuantity`, `RemoveItem`, and
`ClearCart`, replace the single line `if err := s.repo.Save(ctx, cart); err != nil {`
(the save that follows the mutation) with `if err := s.saveRolling(ctx, cart, customerID); err != nil {`.
Do **not** change the `Checkout` save (the cart is emptied there) or the Redis fallback in the
repository. Four call sites total.

**Add the merge method** (place after `Checkout`):
```go
// MergeGuestCart folds a guest cart into the authenticated user's cart and deletes
// the guest cart. It is a no-op (returns the user cart) if no guest cart exists.
func (s *CartService) MergeGuestCart(ctx context.Context, guestID, customerID string) (*model.Cart, error) {
	userCart, err := s.GetCart(ctx, customerID)
	if err != nil {
		return nil, err
	}

	guestCart, err := s.repo.Get(ctx, guestID)
	if err != nil {
		if errors.Is(err, repository.ErrCartNotFound) {
			return userCart, nil
		}
		return nil, err
	}

	for _, item := range guestCart.Items {
		if len(userCart.Items) >= MaxCartItems {
			break
		}
		userCart.AddItem(item)
	}

	if err := s.saveRolling(ctx, userCart, customerID); err != nil {
		return nil, err
	}
	if err := s.repo.Delete(ctx, guestID); err != nil && !errors.Is(err, repository.ErrCartNotFound) {
		s.logger.Warn("failed to delete guest cart after merge",
			zap.String("guestId", guestID),
			zap.Error(err),
		)
	}

	if s.publisher != nil {
		correlationID := getCorrelationID(ctx)
		event := model.NewCartUpdatedEvent(userCart, "guest_merged", correlationID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.Warn("failed to publish cart merged event",
				zap.String("cartId", userCart.ID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("guest cart merged",
		zap.String("guestId", guestID),
		zap.String("customerId", customerID),
	)
	return userCart, nil
}
```

### 6. `cmd/server/main.go` — wire it together

**Old block (service + handler init, lines ~58–63):**
```go
	// Initialize service
	cartService := service.NewCartService(repo, publisher, cfg.CartTTL, logger)

	// Initialize handlers
	cartHandler := handler.NewCartHandler(cartService, logger)
	healthHandler := handler.NewHealthHandler(repo.Client(), version)
```
**New block:**
```go
	// Initialize service
	cartService := service.NewCartService(repo, publisher, cfg.CartTTL, cfg.GuestCartTTL, logger)

	// Initialize guest token manager (falls back to an ephemeral secret in dev)
	guestSecret := cfg.GuestTokenSecret
	if guestSecret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			logger.Fatal("failed to generate guest token secret", zap.Error(err))
		}
		guestSecret = base64.RawURLEncoding.EncodeToString(buf)
		logger.Warn("GUEST_TOKEN_SECRET not set - using an ephemeral secret; guest carts will not survive a restart or scale-out")
	}
	guestTokens := auth.NewGuestTokenManager(guestSecret)

	// Initialize handlers
	cartHandler := handler.NewCartHandler(cartService, guestTokens, logger)
	healthHandler := handler.NewHealthHandler(repo.Client(), version)
```
Add imports `"crypto/rand"` and `"encoding/base64"` to `main.go`.

**Swap the auth middleware — old block (in `setupRouter`):**
```go
	// Apply authentication middleware
	if cfg.OAuth2Enabled && jwtValidator != nil {
		api.Use(handler.AuthMiddleware(jwtValidator, logger))
	} else {
		api.Use(handler.MockAuthMiddleware())
	}
```
**New block:**
```go
	// Apply authentication middleware (guests allowed on cart routes; checkout/merge
	// enforce authentication in the handler)
	if cfg.OAuth2Enabled && jwtValidator != nil {
		api.Use(handler.GuestOrAuthMiddleware(jwtValidator, guestTokens, logger))
	} else {
		api.Use(handler.MockAuthMiddleware())
	}
```
`setupRouter` must now receive `guestTokens`. Add a `guestTokens *auth.GuestTokenManager`
parameter to `setupRouter` and pass it at the call site (line ~80). Keep the parameter order
consistent (add it right after `jwtValidator`).

**Add the merge route — old block:**
```go
		cart.POST("/checkout", cartHandler.Checkout)
	}
```
**New block:**
```go
		cart.POST("/checkout", cartHandler.Checkout)
		cart.POST("/merge", cartHandler.MergeGuestCart)
	}
```

### 7. Backend tests

- `internal/auth/guest_test.go` — as described in §2.
- `internal/service/cart_service_test.go` — add: `MergeGuestCart` sums quantities by
  productID and deletes the guest cart; merge with no guest cart returns the user cart
  unchanged; `ttlFor` returns the guest TTL for a `guest-` id and the standard TTL otherwise;
  a second `AddItem` pushes `ExpiresAt` forward (rolling window).
- Update any existing `NewCartService(...)`/`NewCartHandler(...)` call sites in tests to the new
  signatures.

---

## Frontend changes — `shopping-cart-frontend` (branch `feat/guest-cart`)

> Create this branch from `origin/main` **after** `feat/checkout-payment` has merged, so the
> two carts of frontend work don't collide in `api.ts`/`useCart.ts`. If checkout has not merged
> yet, stop and say so — do not branch from `feat/checkout-payment`.

1. **`src/services/api.ts`** — on the shared axios instance:
   - **Response interceptor:** if a response carries an `x-cart-token` header, write it to
     `localStorage['guestCartToken']`.
   - **Request interceptor:** if `localStorage['guestCartToken']` exists **and** there is no
     authenticated bearer token, attach it as the `X-Cart-Token` request header. (When the user
     is authenticated, send the bearer token as today; the guest token is only for anonymous
     requests.)

2. **`src/services/cartService.ts`** — add
   `mergeGuestCart(): Promise<Cart>` → `api.post('/api/cart/merge', null, { headers: { 'X-Cart-Token': localStorage.getItem('guestCartToken') ?? '' } })`.
   On success, `localStorage.removeItem('guestCartToken')`.

3. **Merge on login** — wherever the OIDC sign-in completes (the `react-oidc-context`
   `onSigninCallback` / an effect on `auth.isAuthenticated` becoming true): if a
   `guestCartToken` is present, call `cartService.mergeGuestCart()`, then invalidate the
   `['cart']` query. Ignore/merge-not-found is non-fatal.

4. **No nginx change needed** — `X-Cart-Token` is a hyphenated header and passes through the
   existing `/api/cart` proxy unchanged. Confirm by reading `nginx.conf`; do not add a rule
   unless a header is actually being stripped.

---

## Infra follow-up (out of scope — file, do not implement here)

`GUEST_TOKEN_SECRET` should be sourced from Vault via ESO into `basket-service-config` for the
live cluster, so guest tokens survive restarts and any future scale-out. Until then the service
generates an ephemeral secret and logs a warning. File this as a `docs/issues/` note in
`shopping-cart-infra`; do not change infra in this task.

---

## Definition of Done

- [ ] `go build ./... && go vet ./...` clean
- [ ] `go test ./...` passes, including the new guest-token, merge, and rolling-TTL tests
- [ ] Guest flow works without a JWT: `POST /api/v1/cart/items` returns 201 and an
      `X-Cart-Token` header; re-sending that token returns the same cart
- [ ] `POST /api/v1/cart/checkout` as a guest returns 401 (`Authentication required to checkout`)
- [ ] Frontend `npm run lint && npm run test && npm run build` pass (frontend branch)
- [ ] Committed and pushed to `feat/guest-cart` in each repo touched
- [ ] memory-bank updated with the commit SHA(s) and task status

**Commit message (exact, both repos):**
```
feat(cart): guest cart with signed token, 3-day rolling TTL, and login merge

Admit anonymous guests on cart routes via a signed X-Cart-Token, persist
their cart on a rolling 3-day window, require auth at checkout, and merge
the guest cart into the user cart on login.
```

---

## What NOT to Do

- Do NOT create a PR.
- Do NOT skip pre-commit hooks (`--no-verify`).
- Do NOT commit to `main` — work on `feat/guest-cart`.
- Do NOT modify files outside the listed targets in each repo.
- Do NOT change the backend cart contracts consumed by the frontend checkout flow
  (`createOrder`/`processPayment` are order/payment services, untouched here).
- Do NOT branch the frontend work off `feat/checkout-payment`; use `origin/main` after it merges.
- Do NOT weaken auth: checkout and merge MUST reject guests.
