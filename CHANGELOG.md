# Changelog

## [Unreleased]

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
