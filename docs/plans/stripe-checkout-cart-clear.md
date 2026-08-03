# Phase D — Basket: stop the premature cart clear

**Repo:** `shopping-cart-basket`  **Branch:** `feat/stripe-checkout-cart-clear`
**Module:** `github.com/user/shopping-cart-basket`
**Design:** `shopping-cart-frontend/docs/plans/stripe-checkout-orchestration-design.md`
**Independent** of Phases A/B/C — branch off `main`, can run in parallel with Phase C.

---

## Objective

`CartService.Checkout()` clears the cart the instant it publishes the `cart.checkout` event —
before any payment happens. That is the root defect the whole Stripe saga resolves: the user's
cart is emptied even if payment later fails. Cart clearing now belongs **exclusively** to the
order-service checkout orchestrator (Phase C), which clears the cart via `DELETE /api/v1/cart`
**only after the order reaches `PAID`**.

This phase removes the premature clear. The `POST /api/v1/cart/checkout` endpoint and the
`cart.checkout` event stay in place (harmless and forward-compatible; the frontend stops calling
the endpoint in Phase E). Every change below was compiled and unit-tested.

**Behavior after this phase:**
- `POST /api/v1/cart/checkout` still validates + publishes the `cart.checkout` event and returns the cart.
- The cart is **no longer emptied** by basket checkout — its items persist until the orchestrator clears them post-`PAID`.
- Empty-cart checkout still returns `ErrCartEmpty` (unchanged).

---

## Before You Start

- `git checkout feat/stripe-checkout-cart-clear && git pull origin feat/stripe-checkout-cart-clear`
- Read `internal/service/cart_service.go` (the `Checkout` method, ~line 257), plus its tests
  `internal/service/cart_service_test.go` (`TestCartService_Checkout`) and
  `internal/service/cart_service_integration_test.go` (`TestCartService_Integration_Checkout`).
- No dependency changes.

---

## Change 1 — `internal/service/cart_service.go`: remove the clear

In `func (s *CartService) Checkout(...)`, replace the trailing clear-and-return block.

**Old:**
```go
	// Clear the cart after successful checkout
	cartCopy := *cart // Keep a copy for the response
	cart.Clear()
	if err := s.repo.Save(ctx, cart); err != nil {
		s.logger.Warn("failed to clear cart after checkout",
			zap.String("cartId", cart.ID),
			zap.Error(err),
		)
	}

	s.logger.Info("cart checkout completed",
		zap.String("cartId", cartCopy.ID),
		zap.String("customerId", customerID),
		zap.Float64("totalAmount", cartCopy.TotalAmount),
	)

	return &cartCopy, nil
}
```
**New:**
```go
	// The cart is intentionally NOT cleared here. Cart clearing is owned by the
	// order-service checkout orchestrator, which clears the cart only after the
	// order reaches PAID (see stripe-checkout-orchestration-design.md). Clearing
	// on the basket checkout event was premature — it emptied the cart before any
	// payment succeeded.
	s.logger.Info("cart checkout event published",
		zap.String("cartId", cart.ID),
		zap.String("customerId", customerID),
		zap.Float64("totalAmount", cart.TotalAmount),
	)

	return cart, nil
}
```

> `cart.Clear()` stays defined and is still used by `ClearCart()` — do not touch that method.

---

## Change 2 — `internal/service/cart_service_test.go`: drop the now-unmet `Save` expectation

In `TestCartService_Checkout`, `Save` is no longer called, so its mock expectation must go or
`mockRepo.AssertExpectations(t)` fails. Remove only that one line.

**Old:**
```go
	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*model.Cart")).Return(nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.CheckoutRequest{
```
**New:**
```go
	mockRepo.On("Get", ctx, "customer-123").Return(existingCart, nil)
	mockPublisher.On("Publish", ctx, mock.AnythingOfType("*model.EventEnvelope")).Return(nil)

	service := NewCartService(mockRepo, mockPublisher, 24*time.Hour, 72*time.Hour, logger)

	req := &model.CheckoutRequest{
```
> The remaining assertions (`cart.TotalAmount == 20.00`, `Len(cart.Items, 1)`) still pass — the
> returned cart is now the live (uncleared) cart, which still holds its item. Leave them as-is.
> Only edit the block inside `TestCartService_Checkout`; other tests use the same three-line
> pattern — do NOT touch those.

---

## Change 3 — `internal/service/cart_service_integration_test.go`: flip the post-checkout assertion

This file is `//go:build integration` (runs only with `-tags integration` + live Redis), but it
must stay correct and compile. In `TestCartService_Integration_Checkout`, the cart now persists.

**Old:**
```go
	// Cart should be cleared after checkout
	postCheckout, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)
	assert.Empty(t, postCheckout.Items)
}
```
**New:**
```go
	// Cart must NOT be cleared by basket checkout — clearing is owned by the
	// order-service orchestrator and happens only after the order is PAID.
	postCheckout, err := service.GetCart(ctx, customerID)
	require.NoError(t, err)
	assert.Len(t, postCheckout.Items, 2)
}
```

---

## Files Changed

| File | Change |
|------|--------|
| `internal/service/cart_service.go` | `Checkout()` no longer clears/saves the cart; returns the live cart |
| `internal/service/cart_service_test.go` | drop the unmet `Save` mock expectation in `TestCartService_Checkout` |
| `internal/service/cart_service_integration_test.go` | assert cart persists after checkout (was: empty) |

No `go.mod`/`go.sum` changes.

---

## Rules

- `gofmt -l internal/service/` → no output
- `go vet ./...` → clean
- `go build ./...` → compiles
- `go test ./...` → all pass (`TestCartService_Checkout` + `TestCartService_Checkout_EmptyCart` included)
- `go vet -tags integration ./internal/service/` → compiles (integration file stays valid)
- No files touched outside the table above. Do NOT remove the endpoint, the handler, the
  `CheckoutRequest` model, the `cart.checkout` event, or `cart.Clear()`.

---

## Definition of Done

- [ ] `Checkout()` no longer empties the cart (unit test green without the `Save` expectation)
- [ ] Empty-cart checkout still returns `ErrCartEmpty` (unchanged test still passes)
- [ ] Integration test asserts the cart persists post-checkout
- [ ] `go build ./...` and `go test ./...` pass; integration file compiles under its tag
- [ ] Committed and pushed to `feat/stripe-checkout-cart-clear`
- [ ] memory-bank updated with commit SHA and task status

**Commit message (exact):**
```
fix(cart): stop clearing cart on basket checkout; orchestrator owns clearing
```

---

## What NOT to Do

- Do NOT create a PR.
- Do NOT skip pre-commit hooks (`--no-verify`).
- Do NOT retire the endpoint / event / `CheckoutRequest` model — only remove the premature clear.
- Do NOT remove or alter `cart.Clear()` or the `ClearCart()` method (still used directly).
- Do NOT edit the other tests that share the `Get`/`Save`/`Publish` mock pattern.
- Do NOT commit to `main` — work on `feat/stripe-checkout-cart-clear`.
