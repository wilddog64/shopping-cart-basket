# Cart Service Troubleshooting Guide

## Common Issues

### Connection Issues

#### Redis Connection Failed

**Symptoms:**
```
failed to connect to Redis: dial tcp: connection refused
```

**Causes:**
- Redis not running
- Incorrect REDIS_HOST or REDIS_PORT
- Network connectivity issues
- Redis authentication failure

**Solutions:**

1. Verify Redis is running:
   ```bash
   redis-cli ping
   # Should return: PONG
   ```

2. Check environment variables:
   ```bash
   echo $REDIS_HOST $REDIS_PORT
   ```

3. Test connectivity:
   ```bash
   nc -zv $REDIS_HOST $REDIS_PORT
   ```

4. Check Redis authentication:
   ```bash
   redis-cli -a $REDIS_PASSWORD ping
   ```

### Authentication Issues

#### 401 Unauthorized

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authorization header required"
  }
}
```

**Causes:**
- Missing Authorization header
- Invalid JWT token
- Expired token
- JWKS fetch failed

**Solutions:**

1. Check if OAuth2 is enabled:
   ```bash
   echo $OAUTH2_ENABLED
   ```

2. For development, use mock auth:
   ```bash
   export OAUTH2_ENABLED=false
   curl -H "X-User-ID: test-user" http://localhost:8083/api/v1/cart
   ```

3. Verify token format:
   ```bash
   curl -H "Authorization: Bearer <token>" http://localhost:8083/api/v1/cart
   ```

4. Check Keycloak connectivity:
   ```bash
   curl $OAUTH2_ISSUER_URI/.well-known/openid-configuration
   ```

### Data Issues

#### Cart Not Found

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Cart not found"
  }
}
```

**Solutions:**

1. Cart may have expired. Check TTL:
   ```bash
   redis-cli TTL "cart:user-123"
   ```

2. Verify cart exists:
   ```bash
   redis-cli GET "cart:user-123"
   ```

3. Check if using correct user ID:
   ```bash
   redis-cli KEYS "cart:*"
   ```

#### Item Not Found

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "Item not found in cart"
  }
}
```

**Solutions:**

1. Get current cart to see item IDs:
   ```bash
   curl http://localhost:8083/api/v1/cart -H "X-User-ID: user-123"
   ```

2. Use correct item ID from the response

### Performance Issues

#### Slow Response Times

**Symptoms:**
- API latency > 100ms
- Request timeouts

**Diagnosis:**

1. Check Redis latency:
   ```bash
   redis-cli --latency
   ```

2. Check service logs:
   ```bash
   kubectl logs cart-service-xxx | grep -i slow
   ```

**Solutions:**

1. Check Redis connection pool
2. Review rate limiting settings
3. Check for memory pressure

#### Rate Limited

**Symptoms:**
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later."
  }
}
```

**Solutions:**

1. Wait for the Retry-After duration
2. Reduce request frequency
3. Increase RATE_LIMIT_RPS if needed

### Startup Issues

#### Service Won't Start

**Symptoms:**
```
failed to connect to Redis
```

**Solutions:**

1. Ensure Redis is accessible before starting
2. Check Docker Compose dependencies:
   ```yaml
   depends_on:
     redis:
       condition: service_healthy
   ```

3. Use startup probe for Kubernetes

## Logging

### Enable Debug Logging

```bash
export LOG_LEVEL=debug
./bin/run-local.sh
```

### Useful Log Patterns

```bash
# Authentication errors
kubectl logs cart-service-xxx | grep -i "auth\|jwt\|token"

# Redis operations
kubectl logs cart-service-xxx | grep -i "redis"

# All errors
kubectl logs cart-service-xxx | grep -i "error"
```

## Health Checks

### Verify Service Health

```bash
# Overall health
curl http://localhost:8083/health

# Liveness (is service alive?)
curl http://localhost:8083/health/live

# Readiness (is service ready for traffic?)
curl http://localhost:8083/health/ready
```

### Kubernetes Health

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=cart-service

# Check events
kubectl describe pod cart-service-xxx

# Check probes
kubectl get pod cart-service-xxx -o jsonpath='{.status.conditions}'
```

## Metrics

### Key Metrics to Monitor

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `http_requests_total` | Request count | - |
| `http_request_duration_seconds` | Latency | p99 > 100ms |
| Redis connection status | Health check | any failure |

### Check Prometheus Metrics

```bash
curl http://localhost:8083/metrics
```

## Testing

### Run Local Tests

```bash
# Unit tests
make test-unit

# All tests
make test

# With coverage
make coverage
```

### Test API Manually

```bash
# Use provided test script
./bin/test-api.sh
```

## Environment Variables

Required for debugging:

```bash
# Enable debug mode
export LOG_LEVEL=debug

# Disable OAuth2 for local testing
export OAUTH2_ENABLED=false
```

## Support

For issues not covered here:
1. Check application logs with DEBUG level
2. Review [Architecture docs](../architecture/README.md)
3. Check [API docs](../api/README.md) for correct usage
4. Open an issue in GitHub repository
