#!/bin/bash
# =============================================================================
# ERP API Smoke Test Script
# =============================================================================
# This script performs comprehensive API smoke tests against the backend.
# It tests all major modules: Auth, Catalog, Partner, Inventory, Trade, Finance
#
# Usage:
#   ./docker/test-api.sh [BASE_URL]
#
# Arguments:
#   BASE_URL - API base URL (default: http://localhost:8081/api/v1)
#
# Exit codes:
#   0 - All tests passed
#   1 - One or more tests failed
# =============================================================================

set -e

# Configuration
BASE_URL="${1:-http://localhost:8081/api/v1}"
TENANT_ID="00000000-0000-0000-0000-000000000001"
TOKEN=""
PASSED=0
FAILED=0
TOTAL=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
    ((TOTAL++))
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
    ((TOTAL++))
}

log_section() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}  $1${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Make API request and check response
# Usage: api_test "Test Name" "METHOD" "endpoint" "expected_status" ["json_body"]
api_test() {
    local test_name="$1"
    local method="$2"
    local endpoint="$3"
    local expected_status="$4"
    local body="$5"

    local url="${BASE_URL}${endpoint}"
    local response
    local status_code

    # Build curl command
    local curl_args=(-s -w "\n%{http_code}" -X "$method")
    curl_args+=(-H "Content-Type: application/json")
    curl_args+=(-H "X-Tenant-ID: ${TENANT_ID}")

    if [ -n "$TOKEN" ]; then
        curl_args+=(-H "Authorization: Bearer ${TOKEN}")
    fi

    if [ -n "$body" ]; then
        curl_args+=(-d "$body")
    fi

    # Make request
    response=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo -e "\n000")

    # Extract status code (last line)
    status_code=$(echo "$response" | tail -n1)

    # Check result
    if [ "$status_code" = "$expected_status" ]; then
        log_success "$test_name (${method} ${endpoint}) - Status: ${status_code}"
        return 0
    else
        log_failure "$test_name (${method} ${endpoint}) - Expected: ${expected_status}, Got: ${status_code}"
        return 1
    fi
}

# Make API request and save token from response
login_and_get_token() {
    local url="${BASE_URL}/auth/login"
    local body='{"username":"admin","password":"admin123"}'

    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "X-Tenant-ID: ${TENANT_ID}" \
        -d "$body" \
        "$url" 2>/dev/null)

    # Extract token (basic parsing - works if token is in response)
    TOKEN=$(echo "$response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$TOKEN" ]; then
        log_success "Login successful - Token obtained"
        return 0
    else
        log_failure "Login failed - Could not obtain token"
        echo "Response: $response"
        return 1
    fi
}

# =============================================================================
# Test Suites
# =============================================================================

test_health_endpoints() {
    log_section "Health & System Endpoints"

    api_test "Health check" "GET" "/../health" "200" || true
    api_test "System info" "GET" "/system/info" "200" || true
    api_test "System ping" "GET" "/system/ping" "200" || true
}

test_auth_module() {
    log_section "Authentication Module"

    # Login and get token
    login_and_get_token || return 1

    api_test "Get current user" "GET" "/auth/me" "200" || true
}

test_catalog_module() {
    log_section "Catalog Module (Categories & Products)"

    # Categories
    api_test "List categories" "GET" "/catalog/categories" "200" || true
    api_test "Get category tree" "GET" "/catalog/categories/tree" "200" || true
    api_test "Get root categories" "GET" "/catalog/categories/roots" "200" || true
    api_test "Get category by ID" "GET" "/catalog/categories/30000000-0000-0000-0000-000000000001" "200" || true

    # Products
    api_test "List products" "GET" "/catalog/products" "200" || true
    api_test "List products with pagination" "GET" "/catalog/products?page=1&page_size=10" "200" || true
    api_test "Get product by ID" "GET" "/catalog/products/40000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get product by code" "GET" "/catalog/products/code/IPHONE15" "200" || true
    api_test "Get products by category" "GET" "/catalog/categories/30000000-0000-0000-0000-000000000011/products" "200" || true
    api_test "Get product stats" "GET" "/catalog/products/stats/count" "200" || true

    # Product units
    api_test "List product units" "GET" "/catalog/products/40000000-0000-0000-0000-000000000010/units" "200" || true
}

test_partner_module() {
    log_section "Partner Module (Customers, Suppliers, Warehouses)"

    # Customers
    api_test "List customers" "GET" "/partner/customers" "200" || true
    api_test "List customers with pagination" "GET" "/partner/customers?page=1&page_size=10" "200" || true
    api_test "Get customer by ID" "GET" "/partner/customers/50000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get customer by code" "GET" "/partner/customers/code/CUST001" "200" || true
    api_test "Get customer stats" "GET" "/partner/customers/stats/count" "200" || true

    # Customer balance
    api_test "Get customer balance" "GET" "/partner/customers/50000000-0000-0000-0000-000000000001/balance" "200" || true
    api_test "Get customer balance summary" "GET" "/partner/customers/50000000-0000-0000-0000-000000000001/balance/summary" "200" || true
    api_test "Get customer balance transactions" "GET" "/partner/customers/50000000-0000-0000-0000-000000000001/balance/transactions" "200" || true

    # Suppliers
    api_test "List suppliers" "GET" "/partner/suppliers" "200" || true
    api_test "Get supplier by ID" "GET" "/partner/suppliers/51000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get supplier by code" "GET" "/partner/suppliers/code/SUP001" "200" || true
    api_test "Get supplier stats" "GET" "/partner/suppliers/stats/count" "200" || true

    # Warehouses
    api_test "List warehouses" "GET" "/partner/warehouses" "200" || true
    api_test "Get warehouse by ID" "GET" "/partner/warehouses/52000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get warehouse by code" "GET" "/partner/warehouses/code/WH001" "200" || true
    api_test "Get default warehouse" "GET" "/partner/warehouses/default" "200" || true
    api_test "Get warehouse stats" "GET" "/partner/warehouses/stats/count" "200" || true
}

test_inventory_module() {
    log_section "Inventory Module"

    # Inventory items
    api_test "List inventory items" "GET" "/inventory/items" "200" || true
    api_test "List inventory items with pagination" "GET" "/inventory/items?page=1&page_size=10" "200" || true
    api_test "Get inventory item by ID" "GET" "/inventory/items/60000000-0000-0000-0000-000000000001" "200" || true
    api_test "Lookup inventory item" "GET" "/inventory/items/lookup?warehouse_id=52000000-0000-0000-0000-000000000001&product_id=40000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get warehouse inventory" "GET" "/inventory/warehouses/52000000-0000-0000-0000-000000000001/items" "200" || true
    api_test "Get product inventory" "GET" "/inventory/products/40000000-0000-0000-0000-000000000001/items" "200" || true
    api_test "Get low stock alerts" "GET" "/inventory/items/alerts/low-stock" "200" || true

    # Stock locks
    api_test "List stock locks" "GET" "/inventory/locks" "200" || true

    # Inventory transactions
    api_test "List inventory transactions" "GET" "/inventory/transactions" "200" || true
    api_test "Get item transactions" "GET" "/inventory/items/60000000-0000-0000-0000-000000000001/transactions" "200" || true

    # Stock takings
    api_test "List stock takings" "GET" "/inventory/stock-takings" "200" || true
    api_test "Get pending approval stock takings" "GET" "/inventory/stock-takings/pending-approval" "200" || true
}

test_trade_module() {
    log_section "Trade Module (Sales & Purchase Orders)"

    # Sales orders
    api_test "List sales orders" "GET" "/trade/sales-orders" "200" || true
    api_test "List sales orders with pagination" "GET" "/trade/sales-orders?page=1&page_size=10" "200" || true
    api_test "Get sales order by ID" "GET" "/trade/sales-orders/70000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get sales order by number" "GET" "/trade/sales-orders/number/SO-2026-0001" "200" || true
    api_test "Get sales order stats" "GET" "/trade/sales-orders/stats/summary" "200" || true

    # Purchase orders
    api_test "List purchase orders" "GET" "/trade/purchase-orders" "200" || true
    api_test "Get purchase order by ID" "GET" "/trade/purchase-orders/72000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get purchase order by number" "GET" "/trade/purchase-orders/number/PO-2026-0001" "200" || true
    api_test "Get pending receipt orders" "GET" "/trade/purchase-orders/pending-receipt" "200" || true
    api_test "Get purchase order stats" "GET" "/trade/purchase-orders/stats/summary" "200" || true

    # Sales returns
    api_test "List sales returns" "GET" "/trade/sales-returns" "200" || true
    api_test "Get sales return stats" "GET" "/trade/sales-returns/stats/summary" "200" || true

    # Purchase returns
    api_test "List purchase returns" "GET" "/trade/purchase-returns" "200" || true
    api_test "Get purchase return stats" "GET" "/trade/purchase-returns/stats/summary" "200" || true
}

test_finance_module() {
    log_section "Finance Module"

    # Account receivables
    api_test "List receivables" "GET" "/finance/receivables" "200" || true
    api_test "Get receivable by ID" "GET" "/finance/receivables/80000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get receivables summary" "GET" "/finance/receivables/summary" "200" || true

    # Account payables
    api_test "List payables" "GET" "/finance/payables" "200" || true
    api_test "Get payable by ID" "GET" "/finance/payables/81000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get payables summary" "GET" "/finance/payables/summary" "200" || true

    # Receipt vouchers
    api_test "List receipts" "GET" "/finance/receipts" "200" || true
    api_test "Get receipt by ID" "GET" "/finance/receipts/82000000-0000-0000-0000-000000000001" "200" || true

    # Payment vouchers
    api_test "List payments" "GET" "/finance/payments" "200" || true
    api_test "Get payment by ID" "GET" "/finance/payments/83000000-0000-0000-0000-000000000001" "200" || true
}

test_report_module() {
    log_section "Report Module"

    # Sales reports
    api_test "Get sales summary" "GET" "/reports/sales/summary" "200" || true
    api_test "Get sales daily trend" "GET" "/reports/sales/daily-trend" "200" || true
    api_test "Get product ranking" "GET" "/reports/sales/products/ranking" "200" || true
    api_test "Get customer ranking" "GET" "/reports/sales/customers/ranking" "200" || true

    # Inventory reports
    api_test "Get inventory summary" "GET" "/reports/inventory/summary" "200" || true
    api_test "Get inventory turnover" "GET" "/reports/inventory/turnover" "200" || true
    api_test "Get inventory value by category" "GET" "/reports/inventory/value-by-category" "200" || true
    api_test "Get inventory value by warehouse" "GET" "/reports/inventory/value-by-warehouse" "200" || true
    api_test "Get slow moving items" "GET" "/reports/inventory/slow-moving" "200" || true

    # Finance reports
    api_test "Get profit loss" "GET" "/reports/finance/profit-loss" "200" || true
    api_test "Get monthly trend" "GET" "/reports/finance/monthly-trend" "200" || true
    api_test "Get profit by product" "GET" "/reports/finance/profit-by-product" "200" || true
    api_test "Get cash flow" "GET" "/reports/finance/cash-flow" "200" || true
}

test_identity_module() {
    log_section "Identity Module (Users, Roles, Tenants)"

    # Users
    api_test "List users" "GET" "/identity/users" "200" || true
    api_test "Get user by ID" "GET" "/identity/users/20000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get user stats" "GET" "/identity/users/stats/count" "200" || true

    # Roles
    api_test "List roles" "GET" "/identity/roles" "200" || true
    api_test "Get role by ID" "GET" "/identity/roles/10000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get system roles" "GET" "/identity/roles/system" "200" || true
    api_test "Get permissions list" "GET" "/identity/permissions" "200" || true
    api_test "Get role stats" "GET" "/identity/roles/stats/count" "200" || true

    # Tenants
    api_test "List tenants" "GET" "/identity/tenants" "200" || true
    api_test "Get tenant by ID" "GET" "/identity/tenants/00000000-0000-0000-0000-000000000001" "200" || true
    api_test "Get tenant stats" "GET" "/identity/tenants/stats" "200" || true
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║           ERP API Smoke Test Suite                        ║${NC}"
    echo -e "${BLUE}╠════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BLUE}║  Base URL: ${BASE_URL}${NC}"
    echo -e "${BLUE}║  Tenant:   ${TENANT_ID}${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"

    # Run all test suites
    test_health_endpoints
    test_auth_module
    test_catalog_module
    test_partner_module
    test_inventory_module
    test_trade_module
    test_finance_module
    test_report_module
    test_identity_module

    # Print summary
    log_section "Test Summary"
    echo ""
    echo -e "  Total tests:  ${TOTAL}"
    echo -e "  ${GREEN}Passed:       ${PASSED}${NC}"
    echo -e "  ${RED}Failed:       ${FAILED}${NC}"
    echo ""

    if [ "$FAILED" -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed!${NC}"
        exit 1
    fi
}

# Check if curl is available
if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed.${NC}"
    exit 1
fi

# Run main
main
