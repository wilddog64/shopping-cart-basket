# Active Context: Basket Service

## Current Status (2026-03-14)

CI green. All PRs merged to main. Branch protection active.

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

## Active Task

- **v0.1.0 release** — cut `release/v0.1.0` from main, add CHANGELOG, open PR, tag after merge.

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
