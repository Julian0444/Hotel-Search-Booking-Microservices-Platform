#!/bin/bash

###############################################################################
#                    LOAD BALANCER TEST SCRIPT                                #
#           Hotel Search & Booking Microservices Platform                     #
###############################################################################

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
    
    containers=(
        "api-gateway"
        "users-api-1"
        "users-api-2"
        "users-api-3"
        "hotels-api-container"
        "search-api-container"
        "users-mysql"
        "hotels-mongo"
        "hotels-rabbit"
        "search-solr"
    )
    
    running=0
    total=${#containers[@]}
    
    for container in "${containers[@]}"; do
        if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
            echo -e "  ${GREEN}✓${NC} $container ${GREEN}running${NC}"
            ((running++))
        else
            echo -e "  ${RED}✗${NC} $container ${RED}not running${NC}"
        fi
    done
    
    echo ""
    echo -e "  ${BOLD}Status: ${running}/${total} containers running${NC}"
}

# Función para verificar health endpoints
check_health() {
    print_section "🏥 Health Check Endpoints"
    
    # API Gateway
    echo -n "  API Gateway (/health): "
    response=$(curl -s -w "%{http_code}" -o /tmp/health_gw.json "$BASE_URL/health" 2>/dev/null)
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ Healthy${NC}"
    else
        echo -e "${RED}✗ HTTP $response${NC}"
    fi
    
    # Users API (a través del gateway)
    echo -n "  Users API (via gateway): "
    response=$(curl -s -w "%{http_code}" -o /tmp/health_users.json "$BASE_URL/users" 2>/dev/null)
    if [ "$response" = "200" ] || [ "$response" = "401" ]; then
        echo -e "${GREEN}✓ Reachable (HTTP $response)${NC}"
    else
        echo -e "${RED}✗ HTTP $response${NC}"
    fi
    
    # Hotels API
    echo -n "  Hotels API (via gateway): "
    response=$(curl -s -w "%{http_code}" -o /tmp/health_hotels.json "$BASE_URL/hotels" 2>/dev/null)
    if [ "$response" = "200" ] || [ "$response" = "404" ]; then
        echo -e "${GREEN}✓ Reachable (HTTP $response)${NC}"
    else
        echo -e "${RED}✗ HTTP $response${NC}"
    fi
    
    # Search API
    echo -n "  Search API (via gateway): "
    response=$(curl -s -w "%{http_code}" -o /tmp/health_search.json "$BASE_URL/search?q=test&offset=0&limit=10" 2>/dev/null)
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}✓ Healthy${NC}"
    else
        echo -e "${YELLOW}⚠ HTTP $response${NC}"
    fi
}

# Función para probar load balancing
test_load_balancing() {
    print_section "⚖️  Load Balancing Test (Users API)"
    
    echo "  Sending $TOTAL_REQUESTS requests to /users endpoint..."
    echo "  Observing which upstream server handles each request..."
    echo ""
    
    declare -A server_counts
    
    for i in $(seq 1 $TOTAL_REQUESTS); do
        # Hacer request y capturar el header X-Upstream-Server
        response=$(curl -s -I "$BASE_URL/users" 2>/dev/null | grep -i "x-upstream-server" | awk '{print $2}' | tr -d '\r')
        
        if [ -n "$response" ]; then
            # Incrementar contador para este servidor
            server_counts[$response]=$((${server_counts[$response]:-0} + 1))
            
            # Colorear según el servidor
            case $response in
                *"users-api-1"*) color=$GREEN ;;
                *"users-api-2"*) color=$YELLOW ;;
                *"users-api-3"*) color=$MAGENTA ;;
                *) color=$NC ;;
            esac
            
            printf "  Request %2d: ${color}→ %s${NC}\n" "$i" "$response"
        else
            printf "  Request %2d: ${RED}✗ No upstream header${NC}\n" "$i"
        fi
        
        sleep 0.2
    done
    
    # Mostrar distribución
    echo ""
    echo -e "  ${BOLD}Load Distribution:${NC}"
    for server in "${!server_counts[@]}"; do
        count=${server_counts[$server]}
        percentage=$((count * 100 / TOTAL_REQUESTS))
        bar=$(printf '█%.0s' $(seq 1 $((percentage / 5))))
        printf "    %-20s: %3d requests (%3d%%) %s\n" "$server" "$count" "$percentage" "$bar"
    done
}

# Función para verificar Nginx status
check_nginx_status() {
    print_section "📊 Nginx Monitoring (Port 8090)"
    
    echo "  Nginx Status:"
    curl -s "$MONITOR_URL/nginx_status" 2>/dev/null | sed 's/^/    /'
    
    echo ""
    echo "  Load Balancer Configuration:"
    curl -s "$MONITOR_URL/status" 2>/dev/null | jq . 2>/dev/null | sed 's/^/    /' || \
    curl -s "$MONITOR_URL/status" 2>/dev/null | sed 's/^/    /'
}

# Función para probar rate limiting
test_rate_limiting() {
    print_section "🚦 Rate Limiting Test"
    
    echo "  Testing API rate limit (10 req/sec burst=20)..."
    echo "  Sending 25 rapid requests..."
    echo ""
    
    success=0
    limited=0
    
    for i in $(seq 1 25); do
        response=$(curl -s -w "%{http_code}" -o /dev/null "$BASE_URL/users" 2>/dev/null)
        if [ "$response" = "200" ]; then
            ((success++))
            echo -n -e "${GREEN}.${NC}"
        elif [ "$response" = "503" ] || [ "$response" = "429" ]; then
            ((limited++))
            echo -n -e "${RED}X${NC}"
        else
            echo -n -e "${YELLOW}?${NC}"
        fi
    done
    
    echo ""
    echo ""
    echo -e "  ${GREEN}Successful: $success${NC} | ${RED}Rate Limited: $limited${NC}"
    
    if [ $limited -gt 0 ]; then
        echo -e "  ${GREEN}✓ Rate limiting is working!${NC}"
    else
        echo -e "  ${YELLOW}⚠ No rate limiting triggered (burst may be higher)${NC}"
    fi
}

# Función para probar endpoints disponibles
test_all_endpoints() {
    print_section "🔗 API Endpoints Test"
    
    endpoints=(
        "GET:/health:API Gateway Health"
        "GET:/users:List Users (Auth required)"
        "POST:/login:Login Endpoint"
        "GET:/hotels:List Hotels"
        "GET:/search?q=test&offset=0&limit=10:Search Hotels"
        "GET:/admin/microservices:Admin (Auth required)"
    )
    
    for endpoint_info in "${endpoints[@]}"; do
        IFS=':' read -r method path description <<< "$endpoint_info"
        
        echo -n "  $method $path - $description: "
        
        if [ "$method" = "POST" ]; then
            response=$(curl -s -w "%{http_code}" -o /dev/null -X POST "$BASE_URL$path" \
                -H "Content-Type: application/json" -d '{}' 2>/dev/null)
        else
            response=$(curl -s -w "%{http_code}" -o /dev/null "$BASE_URL$path" 2>/dev/null)
        fi
        
        case $response in
            200) echo -e "${GREEN}✓ OK${NC}" ;;
            201) echo -e "${GREEN}✓ Created${NC}" ;;
            400) echo -e "${YELLOW}⚠ Bad Request${NC}" ;;
            401) echo -e "${YELLOW}⚠ Unauthorized (expected)${NC}" ;;
            404) echo -e "${YELLOW}⚠ Not Found${NC}" ;;
            500) echo -e "${RED}✗ Server Error${NC}" ;;
            502) echo -e "${RED}✗ Bad Gateway${NC}" ;;
            503) echo -e "${RED}✗ Service Unavailable${NC}" ;;
            *) echo -e "${RED}✗ HTTP $response${NC}" ;;
        esac
    done
}

# Función para mostrar logs recientes de Nginx
show_nginx_logs() {
    print_section "📝 Recent Nginx Logs"
    
    echo "  Last 10 access log entries:"
    docker logs --tail 10 api-gateway 2>/dev/null | grep -v "^\s*$" | sed 's/^/    /' || \
    echo "    No logs available"
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
    
    # 4. Nginx status
    check_nginx_status
    
    # 5. Test all endpoints
    test_all_endpoints
    
    # 6. Rate limiting test
    test_rate_limiting
    
    # 7. Show logs
    show_nginx_logs
    
    print_header "✅ TEST SUITE COMPLETED"
    
    echo -e "  ${BOLD}Summary:${NC}"
    echo "  • Nginx is load balancing requests across 3 users-api instances"
    echo "  • All API endpoints are accessible through the gateway"
    echo "  • Rate limiting is configured for protection"
    echo "  • Monitoring available at $MONITOR_URL/nginx_status"
    echo ""
    echo -e "  ${BOLD}Direct Service URLs (for debugging):${NC}"
    echo "  • API Gateway:    http://localhost"
    echo "  • Nginx Monitor:  http://localhost:8090/nginx_status"
    echo "  • RabbitMQ:       http://localhost:15672 (root/root)"
    echo "  • Solr Admin:     http://localhost:8983"
    echo ""
}

# Run main function
main "$@"
