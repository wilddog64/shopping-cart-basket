# Copilot Instructions — Basket Service

## Service Overview

Go 1.21+ microservice managing shopping basket sessions.
Redis for state, RabbitMQ for events, Keycloak for OAuth2/OIDC.

---

## Architecture Guardrails

### Layer Boundaries — Never Cross These
- **Handler** layer: HTTP only — no business logic, no direct Redis calls
- **Service** layer: business logic only — no HTTP concerns, no raw Redis commands
- **Repository** layer: Redis access only — no business logic
- A handler must never call the repository directly. Always go through the service layer.

### Redis Key Isolation
- All keys must follow the pattern `cart:{customerId}` — no exceptions
- Never read or write keys owned by another service
- Never use `KEYS *` or `SCAN` without a prefix filter in production paths
- TTL must always be set on write — no persistent keys

### RabbitMQ Ownership
- This service publishes to exchange `events` with routing keys: `cart.created`, `cart.updated`, `cart.cleared`, `cart.checkout`
- This service does NOT consume any queues — it is a publisher only
- Never add a consumer in this repo without an explicit spec and Claude review
- Never publish to a routing key owned by another service (e.g. `order.*`, `inventory.*`)

### Authentication
- JWT validation is optional — controlled by `OAUTH2_ENABLED`
- `X-User-ID` header fallback is **dev-only** — never remove the `OAUTH2_ENABLED` gate
- Never bypass JWT validation in production code paths
- Never log JWT token values — log only claims (user ID, roles)

---

## Security Rules (treat violations as bugs)

### Secrets (OWASP A02)
- Never hardcode credentials, API keys, or passwords in source code or config files
- All secrets come from environment variables injected by ESO from Vault
- Never log `REDIS_PASSWORD` or any credential value — not even partially
- `kubectl get secret` output must never appear in code comments or tests

### Injection (OWASP A03)
- All Redis keys must be constructed from validated inputs — never raw user input
- All HTTP inputs must be validated before use in service or repository calls
- Use `binding:"required"` on Gin handler structs; reject on validation failure

### Access Control (OWASP A01)
- Cart operations must always be scoped to the authenticated customer ID
- Never allow one customer to read or modify another customer's cart
- The customer ID comes from JWT claims or `X-User-ID` — never from the request body

### Cryptographic Failures (OWASP A02)
- Never disable TLS verification in RabbitMQ or Redis connections in non-test code
- Never store sensitive cart data unencrypted if it contains PII

---

## Code Quality Rules

### Testing
- All new service and handler logic requires unit tests
- Never delete or comment out existing tests
- Never weaken an assertion (e.g. changing `assert.Equal` to `assert.NotNil`)
- Integration tests require real Redis — do not mock Redis in integration tests
- Run `make test` before every commit; must pass clean

### Code Style
- Follow standard Go conventions (`gofmt`, `goimports`)
- Error wrapping: always include context (`fmt.Errorf("cart.GetCart: %w", err)`)
- Use interfaces for all repository and external dependencies — enables mocking
- Context propagation: every function that calls Redis or RabbitMQ must accept `context.Context`
- Structured logging with zap — never use `fmt.Println` or `log.Printf` in production paths

### Dependencies
- Never add a new dependency without justification in the PR description
- Prefer stdlib over third-party for simple operations
- Pin all dependency versions in `go.mod`

---

## Completion Report Requirements

Before marking any task complete, the agent must provide:
- `make test` output (must be clean)
- `make lint` output (must be clean)
- Confirmation that no test was deleted or weakened
- Confirmation that no credential appears in any changed file
- List of exact files modified

---

## What NOT To Do

- Do not refactor code outside the scope of the current task
- Do not add logging statements beyond what is needed for the task
- Do not change Redis key patterns without an explicit spec
- Do not add new RabbitMQ routing keys without updating `shopping-cart-infra` event contracts
- Do not change the `OAUTH2_ENABLED` default value
