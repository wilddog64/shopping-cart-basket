# Retrospective — Guest Cart (backend)

**Date:** 2026-07-15
**Milestone:** Guest cart backend — anonymous carts, rolling TTL, login merge
**PR:** #13 — merged to main (`d79e5753`)
**Participants:** Claude, Codex, Copilot

## What Went Well

- **End-to-end local verification without host Go.** Build/vet/unit/integration ran in a
  `golang:1.21` container (`go build ./... && go vet ./... && go test ./...`) with shared
  mod/build cache volumes, plus a live guest smoke and a full authenticated login→merge flow
  in vCluster + dev Keycloak using a real RS256 JWT (13/13). No reliance on CI for build/test.
- **Copilot caught a genuine data-loss bug that local tests missed.** The merge-at-capacity
  edge case was not covered by the original suite; review surfaced it before merge.
- **Fixes landed with root-cause analysis + regression tests, not just patches.** Both findings
  were documented in `docs/issues/2026-07-16-copilot-pr13-review-findings.md`, and two targeted
  tests (`..._ExistingProductAtCapacity`, `..._NewProductAtCapacity`) now lock the behavior.
- **Clean `/create-pr` flow.** CI green, all Copilot threads replied-to and resolved via GraphQL,
  issue doc + README Issue Logs updated, memory-bank kept current.

## What Went Wrong

- **`MergeGuestCart` used a position-based cap.** The loop did `if len(userCart.Items) >= MaxCartItems { break }`,
  which (a) dropped quantity-merges for products already in the user cart and (b) silently
  discarded remaining guest items *while still deleting the guest cart* — permanent loss with no
  error surfaced. Root cause: `model.Cart.AddItem` has two mutation modes (quantity-merge vs.
  append) and the cap ignored the distinction. Fixed in `25c3d5b`.
- **Overflow returned 500, not 400.** The handler mapped every merge error to InternalError.
  Fixed alongside the above.
- **`activeContext.md` carried stale "Active Task" rows.** The multi-arch workflow pin
  (shipped 2026-03-17) and the v0.1.0 release (tagged 2026-03-14) were never cleared from the
  active-task list, so they read as pending work this session. Cleaned up in this close-out.

## Process Rules Added

| Rule | Where | Why |
|------|-------|-----|
| Loops that enforce a collection cap must distinguish "grows the collection" from "updates in place," and must fail loudly (return an error) rather than truncate-and-commit. | `docs/issues/2026-07-16-copilot-pr13-review-findings.md` | Prevents the silent-drop class of bug when a helper has multiple mutation modes. |
| Clear an Active-Task row at its close-out, not "later." | this retro | Two shipped tasks lingered as stale rows and cost a verification cycle to disprove. |

## Decisions Made

- Guest identity is a **signed HMAC `X-Cart-Token`** (`internal/auth/guest.go`), not a server
  session. Guest carts persist on a **3-day rolling TTL** refreshed on every write (`saveRolling`).
- **Checkout stays auth-gated**; `POST /api/v1/cart/merge` folds the guest cart into the user
  cart on login (quantities summed per product) and then deletes the guest cart.
- On merge overflow, **fail atomically**: return `ErrMaxItemsExceeded` *before* saving the user
  cart or deleting the guest cart, so nothing is lost.
- **Frontend guest-cart work stays deferred** — the paired token-persistence + merge-on-login
  branch in `shopping-cart-frontend` is cut from `origin/main` only *after* `feat/checkout-payment`
  merges. The backend contract is now live and stable for that work to build on.

## Theme

The guest-cart backend shipped with strong verification discipline — a full containerized test
run plus a live authenticated merge against real Keycloak — and still, Copilot caught a real
data-loss edge case the local suite didn't cover. The lesson isn't that local testing was weak;
it's that a helper with two mutation modes needs its callers' invariants checked explicitly, and
that review remains valuable precisely at the seams local tests don't reach. Close-out also
surfaced two stale active-task rows, a reminder that bookkeeping deferred is bookkeeping that
quietly rots.
