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
# Gateway con TLS local (plan 10): el cert es self-signed, así que TODOS los
# curl contra BASE_URL llevan -k (controlado: solo apunta a localhost).
# El monitoreo :8090 sigue siendo HTTP, publicado solo en 127.0.0.1 (RV26).
BASE_URL="https://localhost"
HTTP_URL="http://localhost"
MONITOR_URL="http://localhost:8090"
TOTAL_REQUESTS=12

# Acumulador de fallos
FAILURES=0

pass() { echo -e "  ${GREEN}✓ PASS${NC} $1"; }
fail() { echo -e "  ${RED}✗ FAIL${NC} $1"; FAILURES=$((FAILURES + 1)); }

# curl que nunca corta el script (devuelve 000 si no conecta).
# -k: solo por el cert self-signed local de nginx/certs (ver arriba).
http_code() {
    curl -sk -o /dev/null -w "%{http_code}" --max-time 10 "$@" 2>/dev/null || echo "000"
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

    containers="nginx users-api-1 users-api-2 users-api-3 hotels-api search-api mysql mongo rabbitmq solr"

    running=0
    total=0
    running_names=$(docker compose ps --status running --services 2>/dev/null || true)

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

# TLS + redirect (plan 10: SD4/SD5)
check_tls() {
    print_section "🔐 TLS / Redirect / Security Headers"

    # HTTP → 301 https (curl sin -L: la respuesta ES el redirect)
    response=$(http_code "$HTTP_URL/api/v1/hotels")
    if [ "$response" = "301" ]; then
        pass "HTTP → 301 redirect a https"
    else
        fail "HTTP /api/v1/hotels → $response (esperado 301)"
    fi

    # Security headers presentes en una ruta de API real (SD5)
    headers=$(curl -sk -I --max-time 10 "$BASE_URL/api/v1/hotels" 2>/dev/null || true)
    for h in "x-frame-options" "x-content-type-options" "strict-transport-security"; do
        if echo "$headers" | grep -qi "^$h"; then
            pass "header $h presente en /api/v1/hotels"
        else
            fail "header $h AUSENTE en /api/v1/hotels"
        fi
    done
}

# La misma URI debe llegar siempre al índice; no hay caché HTTP de búsqueda.
check_search_cache() {
    print_section "Search freshness headers"
    for attempt in 1 2; do
        headers=$(curl -sk -D - -o /dev/null --max-time 10 "$BASE_URL/api/v1/search?q=cordoba&limit=5")
        if echo "$headers" | grep -qi '^cache-control:.*no-store' && ! echo "$headers" | grep -qi '^x-cache-status:.*HIT'; then
            pass "misma URI intento $attempt: no-store sin cache HIT"
        else
            fail "búsqueda conserva una caché HTTP inesperada"
        fi
    done
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
        upstream=$(curl -sk -I --max-time 10 "$BASE_URL/api/v1/users" 2>/dev/null | grep -i "x-upstream-server" | awk '{print $2}' | tr -d '\r' || true)

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

    echo "  Testing /api/v1/login rate limit (30 req/min, burst=10)..."
    echo "  Sending 20 rapid requests..."
    echo ""

    success=0
    limited=0
    wrong_status=0

    i=1
    while [ "$i" -le 20 ]; do
        response=$(http_code -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{}')
        if [ "$response" = "429" ]; then
            limited=$((limited + 1))
            echo -n -e "${RED}X${NC}"
        elif [ "$response" = "503" ]; then
            # I7 (plan 10): el límite debe ser 429, nunca más 503
            wrong_status=$((wrong_status + 1))
            echo -n -e "${YELLOW}?${NC}"
        else
            success=$((success + 1))
            echo -n -e "${GREEN}.${NC}"
        fi
        i=$((i + 1))
    done

    echo ""
    echo ""
    echo -e "  ${GREEN}Passed through: $success${NC} | ${RED}Rate Limited (429): $limited${NC} | 503s: $wrong_status"

    if [ "$limited" -gt 0 ] && [ "$wrong_status" -eq 0 ]; then
        pass "rate limiting activo con 429 ($limited/20 limitados, cero 503)"
    elif [ "$wrong_status" -gt 0 ]; then
        fail "el rate limit devolvió $wrong_status respuestas 503 (esperado solo 429 — I7)"
    else
        fail "rate limiting no se disparó (0/20 limitados con burst=10)"
    fi

    # El 429 debe traer el envelope JSON estándar (a esta altura el budget de
    # /login está agotado, así que este request extra viene limitado)
    limited_response=$(curl -sk -D - --max-time 10 -X POST "$BASE_URL/api/v1/login" -H "Content-Type: application/json" -d '{}' 2>/dev/null || true)
    if echo "$limited_response" | grep -qi "^content-type:.*application/json" \
        && echo "$limited_response" | grep -q '"code":"rate_limited"' \
        && echo "$limited_response" | grep -q '"trace_id"'; then
        pass "429 con Content-Type JSON, envelope estándar y trace_id"
    else
        fail "el 429 no trae el envelope JSON estándar con trace_id"
    fi

    preflight=$(http_code -X OPTIONS "$BASE_URL/api/v1/login" \
        -H 'Origin: http://localhost:5173' -H 'Access-Control-Request-Method: POST' \
        -H 'Access-Control-Request-Headers: content-type')
    if [ "$preflight" = "204" ]; then
        pass "OPTIONS no consume el presupuesto de login agotado"
    else
        fail "OPTIONS penalizado por login_limit: $preflight"
    fi
    # Prueba dedicada de recuperación; no es un retry oculto del login normal.
    sleep 3
    recovered=$(http_code -X POST "$BASE_URL/api/v1/login" -H 'Content-Type: application/json' -d '{}')
    if [ "$recovered" = "400" ]; then
        pass "el presupuesto se recupera y vuelve a validar el body"
    else
        fail "login_limit no se recuperó: $recovered"
    fi
}

# Función para mostrar logs recientes de Nginx (informativo)
show_nginx_logs() {
    print_section "📝 Recent Nginx Logs"

    echo "  Last 10 access log entries:"
    docker compose logs --tail 10 nginx 2>/dev/null | grep -v '^$' | sed 's/^/    /' || echo "    No logs available"
}

# Main execution
main() {
    print_header "🚀 LOAD BALANCER TEST SUITE"

    echo -e "  ${BOLD}Platform:${NC} Hotel Search & Booking Microservices"
    echo -e "  ${BOLD}Gateway:${NC} $BASE_URL"
    echo -e "  ${BOLD}Monitor:${NC} $MONITOR_URL"

    # 1. Verificar containers
    check_containers

    # 2. TLS, redirect y security headers (plan 10)
    check_tls

    # 3. Health checks
    check_health

    # 4. Cache de /search (plan 10)
    check_search_cache

    # 5. Load balancing test
    test_load_balancing

    # 6. Nginx status (informativo)
    check_nginx_status

    # 7. Test all endpoints
    test_all_endpoints

    # 8. Rate limiting test (último: agota el budget de /login)
    test_rate_limiting

    # 9. Show logs (informativo)
    show_nginx_logs

    if [ "$FAILURES" -eq 0 ]; then
        print_header "✅ TEST SUITE PASSED"
    else
        print_header "❌ TEST SUITE FAILED: $FAILURES check(s)"
    fi

    echo -e "  ${BOLD}Direct Service URLs (for debugging):${NC}"
    echo "  • API Gateway:    https://localhost (self-signed: curl -k / aceptar en el browser)"
    echo "  • Nginx Monitor:  http://localhost:8090/nginx_status (solo loopback — RV26)"
    echo "  • RabbitMQ:       http://localhost:15672"
    echo "  • Solr Admin:     http://localhost:8983"
    echo ""

    [ "$FAILURES" -eq 0 ]
}

# Run main function
main "$@"
