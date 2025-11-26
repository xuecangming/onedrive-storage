#!/bin/bash

# OneDrive Storage Middleware - API Test Script
# This script tests all implemented API endpoints

set -o pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"

echo "=========================================="
echo "OneDrive Storage Middleware - API Tests"
echo "=========================================="
echo ""
echo "Testing against: $BASE_URL"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to check HTTP status
check_status() {
    local expected=$1
    local actual=$2
    local description=$3
    
    if [ "$expected" = "$actual" ]; then
        echo -e "${GREEN}✓${NC} $description (HTTP $actual)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $description (Expected: $expected, Got: $actual)"
        ((TESTS_FAILED++))
        return 1
    fi
}

echo "========================================"
echo "1. System Health Tests"
echo "========================================"

# Test 1: Health check
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "Health check endpoint"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

# Test 2: Info endpoint
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/info")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "Info endpoint"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

echo "========================================"
echo "2. Bucket Management Tests"
echo "========================================"

# Test 3: List buckets (should be empty initially)
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/buckets")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "200" "$HTTP_CODE" "List buckets (initial)"
echo ""

# Test 4: Create a bucket
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/buckets/test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "201" "$HTTP_CODE" "Create bucket 'test-bucket'"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

# Test 5: Try to create duplicate bucket (should fail)
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/buckets/test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "409" "$HTTP_CODE" "Create duplicate bucket (should fail)"
echo ""

# Test 6: Create another bucket
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/buckets/images")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "201" "$HTTP_CODE" "Create bucket 'images'"
echo ""

# Test 7: List buckets (should show 2 buckets)
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/buckets")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "List buckets (should show 2)"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

echo "========================================"
echo "3. Object Storage Tests"
echo "========================================"

# Test 8: Upload text object
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/objects/test-bucket/hello.txt" \
    -H "Content-Type: text/plain" \
    -d "Hello, OneDrive Storage!")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "Upload text object"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

# Test 9: Upload JSON object
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/objects/test-bucket/data.json" \
    -H "Content-Type: application/json" \
    -d '{"name": "test", "value": 123}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "200" "$HTTP_CODE" "Upload JSON object"
echo ""

# Test 10: Upload image to images bucket
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/objects/images/photo.jpg" \
    -H "Content-Type: image/jpeg" \
    --data-binary @<(head -c 1024 /dev/urandom))
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "200" "$HTTP_CODE" "Upload binary object (image)"
echo ""

# Test 11: List objects in test-bucket
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/objects/test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "List objects in bucket"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

# Test 12: Get object metadata (HEAD)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "$BASE_URL/objects/test-bucket/hello.txt")
check_status "200" "$HTTP_CODE" "Get object metadata (HEAD)"
echo ""

# Test 13: Download object
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/objects/test-bucket/hello.txt")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "Download object"
echo "Content: $BODY"
echo ""

# Test 14: Try to access non-existent object
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/objects/test-bucket/nonexistent.txt")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "404" "$HTTP_CODE" "Access non-existent object (should fail)"
echo ""

echo "========================================"
echo "4. Object Deletion Tests"
echo "========================================"

# Test 15: Delete an object
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/objects/test-bucket/hello.txt")
check_status "204" "$HTTP_CODE" "Delete object"
echo ""

# Test 16: Try to delete already deleted object
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/objects/test-bucket/hello.txt")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "404" "$HTTP_CODE" "Delete non-existent object (should fail)"
echo ""

# Test 17: List objects after deletion
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/objects/test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)
check_status "200" "$HTTP_CODE" "List objects after deletion"
if command -v jq > /dev/null 2>&1; then
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
    echo "$BODY"
fi
echo ""

echo "========================================"
echo "5. Bucket Deletion Tests"
echo "========================================"

# Test 18: Try to delete non-empty bucket
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/buckets/test-bucket")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
check_status "409" "$HTTP_CODE" "Delete non-empty bucket (should fail)"
echo ""

# Test 19: Delete remaining object
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/objects/test-bucket/data.json")
check_status "204" "$HTTP_CODE" "Delete remaining object"
echo ""

# Test 20: Delete empty bucket
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/buckets/test-bucket")
check_status "204" "$HTTP_CODE" "Delete empty bucket"
echo ""

# Cleanup
echo "Cleaning up..."
curl -s -X DELETE "$BASE_URL/objects/images/photo.jpg" > /dev/null
curl -s -X DELETE "$BASE_URL/buckets/images" > /dev/null

echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
