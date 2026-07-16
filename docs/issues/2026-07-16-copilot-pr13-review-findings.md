# Copilot PR #13 Review Findings — guest cart merge

**PR:** #13 — `feat(cart): guest cart with signed token, 3-day rolling TTL, and login merge`
**Date:** 2026-07-16
**Fix commit:** `12230f5`

Copilot raised two related findings about `MergeGuestCart` behavior when the
authenticated user's cart is already at `MaxCartItems` (100).

---

## Finding 1 — `internal/service/cart_service.go:316` — silent guest item loss at capacity

**What Copilot flagged:** The merge loop broke out entirely once the user cart
reached `MaxCartItems`:

```go
for _, item := range guestCart.Items {
    if len(userCart.Items) >= MaxCartItems {
        break
    }
    userCart.AddItem(item)
}
```

Because `model.Cart.AddItem` merges quantities into an existing product (without
growing `len(Items)`), the blanket `break` had two bad effects:

- **(a)** A guest item for a product **already in the user cart** would only bump
  a quantity, not add a distinct product — yet it was dropped once the cart hit
  the item cap.
- **(b)** Remaining guest items were **silently discarded**, but the code still
  saved the user cart and **deleted the guest cart** — permanent loss of cart
  contents with no error surfaced to the caller.

**Fix:** Only enforce the cap when a guest item would introduce a **new distinct
product**. Existing-product quantity merges always succeed. A genuine overflow
returns `ErrMaxItemsExceeded` *before* saving the user cart or deleting the guest
cart, so nothing is lost and the operation is atomic (the in-memory user-cart
mutation is discarded on the early return).

```go
for _, item := range guestCart.Items {
    if !userCart.ContainsProduct(item.ProductID) && len(userCart.Items) >= MaxCartItems {
        return nil, ErrMaxItemsExceeded
    }
    userCart.AddItem(item)
}
```

Added `model.Cart.ContainsProduct(productID string) bool` helper (`internal/model/cart.go`).

## Finding 2 — `internal/handler/cart_handler.go:260` — overflow returned 500

**What Copilot flagged:** With Finding 1 fixed, `MergeGuestCart` can now return
`service.ErrMaxItemsExceeded`, but the handler mapped every error to a 500.

**Fix:** Mirror the existing `AddItem` handler — translate `ErrMaxItemsExceeded`
to a 400:

```go
if errors.Is(err, service.ErrMaxItemsExceeded) {
    response.BadRequest(c, "Maximum cart items exceeded")
    return
}
```

---

## Root cause

The merge path was written as a straight port of the guest→user loop without
accounting for the difference between the two ways `AddItem` mutates a cart
(quantity-merge vs. append). The capacity guard was expressed in terms of the
loop position rather than the semantics of each individual item.

## Process note

When a helper (`model.Cart.AddItem`) has two distinct mutation modes, any
capacity/limit check built on top of it must branch on which mode applies. Add a
spec rule: **loops that enforce a collection cap must distinguish "grows the
collection" from "updates in place," and must fail loudly (return an error)
rather than truncate-and-commit.** Regression tests added:
`TestCartService_MergeGuestCart_ExistingProductAtCapacity` (merge succeeds) and
`TestCartService_MergeGuestCart_NewProductAtCapacity` (returns error, guest cart
preserved).
