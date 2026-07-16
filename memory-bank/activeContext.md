# Active Context: Basket Service

## Current Status (2026-07-15)

Guest cart backend **merged** — PR #13 → `main` (`d79e5753`, 2026-07-16). Main synced; work
continues on `docs/next-improvements`. Backend was verified locally end-to-end (containerized
`golang:1.21` build/vet/unit + integration, live guest smoke, and authenticated login→merge in
vCluster + dev Keycloak with a real RS256 JWT). Copilot's 2 merge-at-capacity findings were
fixed pre-merge in `25c3d5b` (see `docs/issues/2026-07-16-copilot-pr13-review-findings.md`) with
regression tests; threads resolved. Retro: `docs/retro/2026-07-15-guest-cart-retrospective.md`.
Branch protection unchanged at baseline (1 review + CI required, `enforce_admins: false`).

## What's Implemented

- Shopping cart CRUD (add/remove/update items, get cart, clear cart)
- Redis-backed persistence
- JWT auth via Keycloak OAuth2 Resource Server
- Prometheus metrics, health endpoints
- GitHub Actions CI: golangci-lint gate + build/test + Trivy scan + ghcr.io push

## CI History

- **fix/ci-stabilization PR #1** — merged 2026-03-14. Fixed: Dockerfile security upgrades, Trivy.
- **feature/p4-linter PR #1** — merged 2026-03-14. Added golangci-lint (govet, errcheck, staticcheck, gofmt, goimports).
- **Branch protection** — 1 review + CI required, enforce_admins: false

## Active Task / Next Up

- **Redis circuit breaker** — spec `docs/issues/2026-03-18-redis-circuit-breaker.md` (carried on
  `docs/next-improvements` as commit `42fd44a`). Wrap Redis calls in a `gobreaker` circuit breaker
  so cart reads/writes fail fast and recover cleanly when Redis is degraded. Not yet started.
- **Frontend guest-cart (deferred)** — token-persistence + merge-on-login is a paired
  `feat/guest-cart` branch in `shopping-cart-frontend`, to be cut from `origin/main` AFTER
  `feat/checkout-payment` merges. The backend contract (`POST /api/v1/cart/merge`, signed
  `X-Cart-Token`) is now live for it to build on.

**Shipped (do not re-do):** guest cart backend (PR #13, merged `d79e5753`); multi-arch workflow
pin (merged 2026-03-17, `build-push-deploy.yml@999f8d7`); v0.1.0 release (tagged `v0.1.0`,
2026-03-14).

## Agent Instructions

Rules that apply to ALL agents working in this repo:

1. **CI only** — do NOT run `go test` or `golangci-lint` locally without Go 1.21+ installed.
2. **Memory-bank discipline** — do NOT update `memory-bank/activeContext.md` until CI shows `completed success`.
3. **SHA verification** — verify commit SHA with `gh api repos/wilddog64/shopping-cart-basket/commits/<sha>` before reporting.
4. **Do NOT merge PRs** — open the PR and stop.
5. **No unsolicited changes** — only touch files listed in the task spec.

## Key Notes

- Redis required for local dev — use `docker run -p 6379:6379 redis:7-alpine`
- `.[rabbitmq]` optional dep installs pika + hvac + tenacity
