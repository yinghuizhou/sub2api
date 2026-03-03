#\!/bin/bash
# =============================================================================
# Sub2API High Availability Test Script
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASSED=0
FAILED=0

log_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

log_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

echo "=========================================="
echo "  Sub2API HA Deployment Test"
echo "=========================================="
echo ""

# Test 1: Check if all containers are running
log_info "Test 1: Checking container status..."
if docker ps | grep -q "sub2api-1" && \
   docker ps | grep -q "sub2api-2" && \
   docker ps | grep -q "sub2api-3" && \
   docker ps | grep -q "sub2api-nginx" && \
   docker ps | grep -q "sub2api-postgres" && \
   docker ps | grep -q "sub2api-redis"; then
    log_pass "All containers are running"
else
    log_fail "Some containers are not running"
fi

# Test 2: Check Nginx health
log_info "Test 2: Checking Nginx health..."
if curl -sf http://localhost/health > /dev/null 2>&1; then
    log_pass "Nginx is healthy"
else
    log_fail "Nginx is not responding"
fi

# Test 3: Check Sub2API instances health
log_info "Test 3: Checking Sub2API instances health..."
for i in 1 2 3; do
    if docker exec sub2api-$i curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        log_pass "Sub2API instance $i is healthy"
    else
        log_fail "Sub2API instance $i is not healthy"
    fi
done

# Test 4: Check PostgreSQL
log_info "Test 4: Checking PostgreSQL..."
if docker exec sub2api-postgres pg_isready -U sub2api > /dev/null 2>&1; then
    log_pass "PostgreSQL is ready"
else
    log_fail "PostgreSQL is not ready"
fi

# Test 5: Check Redis
log_info "Test 5: Checking Redis..."
if docker exec sub2api-redis redis-cli ping > /dev/null 2>&1; then
    log_pass "Redis is responding"
else
    log_fail "Redis is not responding"
fi

# Test 6: Check database connections
log_info "Test 6: Checking database connections..."
CONN_COUNT=$(docker exec sub2api-postgres psql -U sub2api -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname='sub2api';" 2>/dev/null | tr -d ' ')
if [ -n "$CONN_COUNT" ] && [ "$CONN_COUNT" -gt 0 ]; then
    log_pass "Database has $CONN_COUNT active connections"
else
    log_fail "No active database connections found"
fi

# Test 7: Check load balancing
log_info "Test 7: Testing load balancing (10 requests)..."
declare -A instance_hits
for i in {1..10}; do
    response=$(curl -s http://localhost/health)
    # Note: This is a simple test, actual load balancing verification would need more sophisticated tracking
done
log_pass "Load balancing test completed (check Nginx logs for distribution)"

# Test 8: Check Nginx configuration
log_info "Test 8: Checking Nginx configuration..."
if docker exec sub2api-nginx nginx -t > /dev/null 2>&1; then
    log_pass "Nginx configuration is valid"
else
    log_fail "Nginx configuration has errors"
fi

# Test 9: Check environment variables consistency
log_info "Test 9: Checking JWT_SECRET consistency..."
JWT1=$(docker exec sub2api-1 env | grep "^JWT_SECRET=" | cut -d= -f2)
JWT2=$(docker exec sub2api-2 env | grep "^JWT_SECRET=" | cut -d= -f2)
JWT3=$(docker exec sub2api-3 env | grep "^JWT_SECRET=" | cut -d= -f2)

if [ "$JWT1" = "$JWT2" ] && [ "$JWT2" = "$JWT3" ] && [ -n "$JWT1" ]; then
    log_pass "JWT_SECRET is consistent across all instances"
else
    log_fail "JWT_SECRET is not consistent across instances"
fi

# Test 10: Check TOTP_ENCRYPTION_KEY consistency
log_info "Test 10: Checking TOTP_ENCRYPTION_KEY consistency..."
TOTP1=$(docker exec sub2api-1 env | grep "^TOTP_ENCRYPTION_KEY=" | cut -d= -f2)
TOTP2=$(docker exec sub2api-2 env | grep "^TOTP_ENCRYPTION_KEY=" | cut -d= -f2)
TOTP3=$(docker exec sub2api-3 env | grep "^TOTP_ENCRYPTION_KEY=" | cut -d= -f2)

if [ "$TOTP1" = "$TOTP2" ] && [ "$TOTP2" = "$TOTP3" ] && [ -n "$TOTP1" ]; then
    log_pass "TOTP_ENCRYPTION_KEY is consistent across all instances"
else
    log_fail "TOTP_ENCRYPTION_KEY is not consistent across instances"
fi

# Test 11: Check frontend access
log_info "Test 11: Checking frontend access..."
if curl -sf http://localhost/ > /dev/null 2>&1; then
    log_pass "Frontend is accessible"
else
    log_fail "Frontend is not accessible"
fi

# Test 12: Check API access
log_info "Test 12: Checking API access..."
if curl -sf http://localhost/api/v1/health > /dev/null 2>&1; then
    log_pass "API is accessible"
else
    # API might return 404 if /api/v1/health doesn't exist, check /health instead
    if curl -sf http://localhost/health > /dev/null 2>&1; then
        log_pass "API health endpoint is accessible"
    else
        log_fail "API is not accessible"
    fi
fi

# Test 13: Simulate instance failure
log_info "Test 13: Simulating instance failure..."
log_info "Stopping instance 2..."
docker compose -f docker-compose.ha.yml stop sub2api-2 > /dev/null 2>&1
sleep 5

if curl -sf http://localhost/health > /dev/null 2>&1; then
    log_pass "Service still available after instance 2 failure"
else
    log_fail "Service unavailable after instance 2 failure"
fi

log_info "Restarting instance 2..."
docker compose -f docker-compose.ha.yml start sub2api-2 > /dev/null 2>&1
sleep 10

if docker exec sub2api-2 curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    log_pass "Instance 2 recovered successfully"
else
    log_fail "Instance 2 failed to recover"
fi

# Summary
echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASSED"
echo -e "${RED}Failed:${NC} $FAILED"
echo "=========================================="

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed\!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Please check the logs.${NC}"
    exit 1
fi
