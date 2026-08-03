# Changelog

## [Unreleased]

### Added
- `.github/workflows/dependabot-automerge.yml`: auto-merge Dependabot minor/patch version updates and all security updates (any semver, via `alert-lookup`) with `gh pr merge --auto --squash` once required CI checks pass; **non-security** major bumps stay open for review (`dependabot/fetch-metadata` pinned to v2.3.0; `pull_request_target` scoped to `main`, job-level least-privilege permissions, gated on the PR author, no PR-head checkout)
- `.github/dependabot.yml`: Dependabot scheduled version updates for Go modules, Docker base images, and GitHub Actions (weekly; minor/patch grouped, majors separate). Repository-level Dependabot security updates (immediate advisory-triggered PRs) are enabled separately as a repo setting — together they close the first-mile CVE gap so a flagged app dependency opens an update PR that CI builds into a clean image
- Guest cart support: unauthenticated users can build a cart via a signed HMAC `X-Cart-Token` (`internal/auth/guest.go`), with a 3-day rolling TTL refreshed on every write
- `POST /api/v1/cart/merge`: authenticated endpoint that merges a guest cart into the caller's cart (quantities summed per product), then deletes the guest cart
- `GuestOrAuthMiddleware`: accepts either a Bearer JWT or a guest cart token, keeping checkout auth-gated while allowing anonymous browsing
- `GUEST_TOKEN_SECRET` config for signing guest cart tokens
- `.githooks/pre-push`: pre-push hook to block accidental direct pushes from feature branches to main; bypass with `ALLOW_MAIN_PUSH=1`

### Fixed
- Stop clearing the cart on basket checkout (Stripe checkout Phase D): the basket no longer empties the cart when checkout is initiated. The order orchestrator now owns cart clearing and clears only after the order reaches PAID, so a failed or abandoned payment no longer loses the shopper's cart. Spec: `docs/plans/` Phase D basket stop premature cart clear.
- `k8s/base/configmap.yaml`: OAUTH2_ISSUER_URI changed from `keycloak.identity.svc.cluster.local:8080` to `keycloak.shopping-cart.local` to match actual JWT iss claim and remove incorrect port; allows ubuntu-k3s pods to reach Keycloak via cross-cluster DNS resolution

### Changed
- `.github/dependabot.yml`: defer the `golangci/golangci-lint-action` semver-major (v9 defaults to golangci-lint v2, whose config schema breaks this repo's v1-style `.golangci.yml`); other GitHub Actions majors still flow. Stops Dependabot recreating the failing v9 PR each week. Spec: `docs/plans/dependabot-defer-golangci-action-major.md`
- Reduce deployment replicas from 2 to 1 for dev/test environment; HPAs not applicable on single-node cluster (will reintroduce in v1.1.0 EKS)
- `k8s/base/deployment.yaml`: set rolling update to `maxSurge: 0` / `maxUnavailable: 1` (recreate-style) so rollouts complete on the single-node hostinger cluster instead of wedging with an unschedulable surge pod

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
