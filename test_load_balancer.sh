#!/bin/bash

###############################################################################
#                    LOAD BALANCER TEST SCRIPT                                #
#           Hotel Search & Booking Microservices Platform                     #
#                                                                             #
#  Smoke test con asserts: cada check imprime PASS/FAIL, acumula fallos y     #
#  el script sale con exit 1 si hubo alguno (apto para CI).                   #
#  Compatible con bash 3.2 (macOS): sin arrays asociativos.                   #
###############################################################################

set -euo pipefail

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Configuración
BASE_URL="http://localhost"
MONITOR_URL="http://localhost:8090"
TOTAL_REQUESTS=12

# Acumulador de fallos
FAILURES=0

pass() { echo -e "  ${GREEN}✓ PASS${NC} $1"; }
fail() { echo -e "  ${RED}✗ FAIL${NC} $1"; FAILURES=$((FAILURES + 1)); }

# curl que nunca corta el script (devuelve 000 si no conecta)
http_code() {
    curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$@" 2>/dev/null || echo "000"
}

print_header() {
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
}

print_section() {
    echo ""
    echo -e "${BLUE}───────────────────────────────────────────────────────────────${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}───────────────────────────────────────────────────────────────${NC}"
    echo ""
}

# Función para verificar containers
check_containers() {
    print_section "🐳 Docker Containers Status"

    containers="api-gateway users-api-1 users-api-2 users-api-3 hotels-api-container search-api-container users-mysql hotels-mongo hotels-rabbit search-solr"

    running=0
    total=0
    running_names=$(docker ps --format "{{.Names}}" 2>/dev/null || true)

    for container in $containers; do
        total=$((total + 1))
        if echo "$running_names" | grep -q "^${container}$"; then
            echo -e "  ${GREEN}✓${NC} $container ${GREEN}running${NC}"
            running=$((running + 1))
        else
            echo -e "  ${RED}✗${NC} $container ${RED}not running${NC}"
        fi
    done

    echo ""
    if [ "$running" -eq "$total" ]; then
        pass "containers: ${running}/${total} running"
    else
        fail "containers: ${running}/${total} running"
    fi
}

# Función para verificar health endpoints
check_health() {
    print_section "🏥 Health Check Endpoints"

    # API Gateway
    response=$(http_code "$BASE_URL/health")
    if [ "$response" = "200" ]; then
        pass "API Gateway /health → 200"
    else
        fail "API Gateway /health → HTTP $response (esperado 200)"
    fi

    # Users API (a través del gateway; 401 = vivo y protegido)
    response=$(http_code "$BASE_URL/api/v1/users")
    if [ "$response" = "200" ] || [ "$response" = "401" ]; then
        pass "Users API via gateway → HTTP $response (reachable)"
    else
        fail "Users API via gateway → HTTP $response (esperado 200/401)"
    fi

    # Hotels API (listado paginado del catálogo)
    response=$(http_code "$BASE_URL/api/v1/hotels")
    if [ "$response" = "200" ] || [ "$response" = "404" ]; then
        pass "Hotels API via gateway → HTTP $response (reachable)"
    else
        fail "Hotels API via gateway → HTTP $response (esperado 200/404)"
    fi

    # Search API
    response=$(http_code "$BASE_URL/api/v1/search?q=test&offset=0&limit=10")
    if [ "$response" = "200" ]; then
        pass "Search API via gateway → 200"
    else
        fail "Search API via gateway → HTTP $response (esperado 200)"
    fi
}

# Función para probar load balancing
test_load_balancing() {
    print_section "⚖️  Load Balancing Test (Users API)"

    echo "  Sending $TOTAL_REQUESTS requests to /api/v1/users endpoint..."
    echo "  Observing which upstream server handles each request..."
    echo ""

    upstreams_file=$(mktemp)

    i=1
    while [ "$i" -le "$TOTAL_REQUESTS" ]; do
        # Hacer request y capturar el header X-Upstream-Server
        upstream=$(curl -s -I --max-time 10 "$BASE_URL/api/v1/users" 2>/dev/null | grep -i "x-upstream-server" | awk '{print $2}' | tr -d '\r' || true)

        if [ -n "$upstream" ]; then
            echo "$upstream" >> "$upstreams_file"

            # Colorear según el servidor
            case $upstream in
                *"users-api-1"*) color=$GREEN ;;
                *"users-api-2"*) color=$YELLOW ;;
                *"users-api-3"*) color=$MAGENTA ;;
                *) color=$NC ;;
            esac

            printf "  Request %2d: ${color}→ %s${NC}\n" "$i" "$upstream"
        else
            printf "  Request %2d: ${RED}✗ No upstream header${NC}\n" "$i"
        fi

        i=$((i + 1))
        sleep 0.2
    done

    # Mostrar distribución (sin arrays asociativos: bash 3.2 compatible)
    echo ""
    echo -e "  ${BOLD}Load Distribution:${NC}"
    if [ -s "$upstreams_file" ]; then
        sort "$upstreams_file" | uniq -c | while read -r count server; do
            percentage=$((count * 100 / TOTAL_REQUESTS))
            printf "    %-20s: %3d requests (%3d%%)\n" "$server" "$count" "$percentage"
        done
    fi

    answered=$(wc -l < "$upstreams_file" | tr -d ' ')
    distinct=$(sort -u "$upstreams_file" | wc -l | tr -d ' ')
    rm -f "$upstreams_file"

    echo ""
    if [ "$answered" -eq "$TOTAL_REQUESTS" ] && [ "$distinct" -ge 2 ]; then
        pass "load balancing: $answered/$TOTAL_REQUESTS respondidos por $distinct upstreams distintos"
    else
        fail "load balancing: $answered/$TOTAL_REQUESTS respondidos, $distinct upstreams distintos (esperado: todos respondidos, ≥2 upstreams)"
    fi
}

# Función para verificar Nginx status (informativo, sin asserts)
check_nginx_status() {
    print_section "📊 Nginx Monitoring (Port 8090)"

    echo "  Nginx Status:"
    curl -s --max-time 5 "$MONITOR_URL/nginx_status" 2>/dev/null | sed 's/^/    /' || echo "    (no disponible)"

    echo ""
    echo "  Load Balancer Configuration:"
    status_body=$(curl -s --max-time 5 "$MONITOR_URL/status" 2>/dev/null || true)
    if [ -n "$status_body" ]; then
        echo "$status_body" | jq . 2>/dev/null | sed 's/^/    /' || echo "$status_body" | sed 's/^/    /'
    else
        echo "    (no disponible)"
    fi
}

# Función para probar endpoints disponibles (con status esperado por endpoint)
test_all_endpoints() {
    print_section "🔗 API Endpoints Test"

    # formato: método|path|códigos aceptados (separados por coma)|descripción
    endpoints="GET|/health|200|API Gateway Health
GET|/api/v1/users|401|List Users (Auth required)
GET|/api/v1/hotels|200|List Hotels
GET|/api/v1/search?q=test&offset=0&limit=10|200|Search Hotels
GET|/api/v1/admin/microservices|401|Admin (Auth required)
GET|/users|404|Unversioned route removed (A2)"

    while IFS='|' read -r method path expected description; do
        response=$(http_code -X "$method" "$BASE_URL$path")
        if echo ",$expected," | grep -q ",$response,"; then
            pass "$method $path → $response ($description)"
        else
            fail "$method $path → $response (esperado $expected — $description)"
        fi
    done <<< "$endpoints"
}

# Función para probar rate limiting (al final: consume el budget de /login)
test_rate_limiting() {
    print_section "🚦 Rate Limiting Test"

    echo "  Testing /api/v1/login rate limit (5 req/min, burst=3)..."
    echo "  Sending 10 rapid requests..."
    echo ""

    success=0
    limited=0

    i=1
    while [ "$i" -le 10 ]; do
        response=$(http_code -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{}')
        if [ "$response" = "503" ] || [ "$response" = "429" ]; then
            limited=$((limited + 1))
            echo -n -e "${RED}X${NC}"
        else
            success=$((success + 1))
            echo -n -e "${GREEN}.${NC}"
        fi
        i=$((i + 1))
    done

    echo ""
    echo ""
    echo -e "  ${GREEN}Passed through: $success${NC} | ${RED}Rate Limited: $limited${NC}"

    if [ "$limited" -gt 0 ]; then
        pass "rate limiting activo ($limited/10 limitados)"
    else
        fail "rate limiting no se disparó (0/10 limitados con burst=3)"
    fi
}

# Función para mostrar logs recientes de Nginx (informativo)
show_nginx_logs() {
    print_section "📝 Recent Nginx Logs"

    echo "  Last 10 access log entries:"
    docker logs --tail 10 api-gateway 2>/dev/null | grep -v "^\s*$" | sed 's/^/    /' || echo "    No logs available"
}

# Main execution
main() {
    print_header "🚀 LOAD BALANCER TEST SUITE"

    echo -e "  ${BOLD}Platform:${NC} Hotel Search & Booking Microservices"
    echo -e "  ${BOLD}Gateway:${NC} $BASE_URL"
    echo -e "  ${BOLD}Monitor:${NC} $MONITOR_URL"

    # 1. Verificar containers
    check_containers

    # 2. Health checks
    check_health

    # 3. Load balancing test
    test_load_balancing

    # 4. Nginx status (informativo)
    check_nginx_status

    # 5. Test all endpoints
    test_all_endpoints

    # 6. Rate limiting test (último: agota el budget de /login)
    test_rate_limiting

    # 7. Show logs (informativo)
    show_nginx_logs

    if [ "$FAILURES" -eq 0 ]; then
        print_header "✅ TEST SUITE PASSED"
    else
        print_header "❌ TEST SUITE FAILED: $FAILURES check(s)"
    fi

    echo -e "  ${BOLD}Direct Service URLs (for debugging):${NC}"
    echo "  • API Gateway:    http://localhost"
    echo "  • Nginx Monitor:  http://localhost:8090/nginx_status"
    echo "  • RabbitMQ:       http://localhost:15672"
    echo "  • Solr Admin:     http://localhost:8983"
    echo ""

    [ "$FAILURES" -eq 0 ]
}

# Run main function
main "$@"
