# Changelog

## [Unreleased]

### Added
- Guest cart support: unauthenticated users can build a cart via a signed HMAC `X-Cart-Token` (`internal/auth/guest.go`), with a 3-day rolling TTL refreshed on every write
- `POST /api/v1/cart/merge`: authenticated endpoint that merges a guest cart into the caller's cart (quantities summed per product), then deletes the guest cart
- `GuestOrAuthMiddleware`: accepts either a Bearer JWT or a guest cart token, keeping checkout auth-gated while allowing anonymous browsing
- `GUEST_TOKEN_SECRET` config for signing guest cart tokens
- `.githooks/pre-push`: pre-push hook to block accidental direct pushes from feature branches to main; bypass with `ALLOW_MAIN_PUSH=1`

### Fixed
- `k8s/base/configmap.yaml`: OAUTH2_ISSUER_URI changed from `keycloak.identity.svc.cluster.local:8080` to `keycloak.shopping-cart.local` to match actual JWT iss claim and remove incorrect port; allows ubuntu-k3s pods to reach Keycloak via cross-cluster DNS resolution

### Changed
- Reduce deployment replicas from 2 to 1 for dev/test environment; HPAs not applicable on single-node cluster (will reintroduce in v1.1.0 EKS)

## [0.1.0] - 2026-03-14

### Added
- Shopping cart CRUD API (add/remove/update items, get cart, clear cart)
- Redis-backed cart persistence
- JWT authentication via Keycloak OAuth2 Resource Server
- Prometheus metrics and health/readiness endpoints
- Dockerfile (multi-stage, non-root user)
- Kubernetes manifests (Deployment, Service, ConfigMap, ServiceAccount)
- GitHub Actions CI: golangci-lint gate + build/test + Trivy security scan + ghcr.io push
- Branch protection (1 required review + CI status check)
