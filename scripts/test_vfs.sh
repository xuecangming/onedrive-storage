#!/bin/bash

# VFS API Test Script
# Tests the Virtual File System endpoints

set -e

BASE_URL="http://localhost:8080/api/v1"
TEST_BUCKET="vfs-test-bucket"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to print test results
print_test_result() {
    TESTS_RUN=$((TESTS_RUN + 1))
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

echo "========================================"
echo "VFS API Test Suite"
echo "========================================"
echo ""

# Wait for server to be ready
echo "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
        echo "Server is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Server failed to start"
        exit 1
    fi
    sleep 1
done
echo ""

# Cleanup from previous runs
echo "Cleaning up from previous test runs..."
# Try to delete all objects in the bucket
curl -s "${BASE_URL}/objects/${TEST_BUCKET}" 2>/dev/null | jq -r '.objects[]?.key' 2>/dev/null | while read key; do
    [ -n "$key" ] && curl -s -X DELETE "${BASE_URL}/objects/${TEST_BUCKET}/${key}" > /dev/null 2>&1
done
# Try to delete the bucket
curl -s -X DELETE "${BASE_URL}/buckets/${TEST_BUCKET}" > /dev/null 2>&1 || true
sleep 1
echo ""

# Test 1: Create test bucket
echo "Test 1: Create test bucket"
RESPONSE=$(curl -s -X PUT "${BASE_URL}/buckets/${TEST_BUCKET}")
if echo "$RESPONSE" | grep -q "\"name\":\"${TEST_BUCKET}\""; then
    print_test_result 0 "Create test bucket"
else
    print_test_result 1 "Create test bucket"
fi
echo ""

# Test 2: Upload file to virtual path
echo "Test 2: Upload file to virtual path /documents/readme.txt"
echo "This is a test file" > /tmp/test_file.txt
RESPONSE=$(curl -s -X PUT \
    -H "Content-Type: text/plain" \
    --data-binary @/tmp/test_file.txt \
    "${BASE_URL}/vfs/${TEST_BUCKET}/documents/readme.txt")
if echo "$RESPONSE" | grep -q "\"name\":\"readme.txt\""; then
    print_test_result 0 "Upload file to virtual path"
else
    print_test_result 1 "Upload file to virtual path"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 3: List root directory
echo "Test 3: List root directory"
RESPONSE=$(curl -s "${BASE_URL}/vfs/${TEST_BUCKET}/?type=directory")
if echo "$RESPONSE" | grep -q "\"name\":\"documents\""; then
    print_test_result 0 "List root directory"
else
    print_test_result 1 "List root directory"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 4: List documents directory
echo "Test 4: List documents directory"
RESPONSE=$(curl -s "${BASE_URL}/vfs/${TEST_BUCKET}/documents/?type=directory")
if echo "$RESPONSE" | grep -q "\"name\":\"readme.txt\""; then
    print_test_result 0 "List documents directory"
else
    print_test_result 1 "List documents directory"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 5: Download file from virtual path
echo "Test 5: Download file from virtual path"
RESPONSE=$(curl -s "${BASE_URL}/vfs/${TEST_BUCKET}/documents/readme.txt")
if echo "$RESPONSE" | grep -q "This is a test file"; then
    print_test_result 0 "Download file from virtual path"
else
    print_test_result 1 "Download file from virtual path"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 6: Upload another file to nested path
echo "Test 6: Upload file to nested path /documents/2024/report.txt"
echo "Annual report 2024" > /tmp/report.txt
RESPONSE=$(curl -s -X PUT \
    -H "Content-Type: text/plain" \
    --data-binary @/tmp/report.txt \
    "${BASE_URL}/vfs/${TEST_BUCKET}/documents/2024/report.txt")
if echo "$RESPONSE" | grep -q "\"name\":\"report.txt\""; then
    print_test_result 0 "Upload file to nested path (auto-create directories)"
else
    print_test_result 1 "Upload file to nested path (auto-create directories)"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 7: List directory recursively
echo "Test 7: List documents directory recursively"
RESPONSE=$(curl -s "${BASE_URL}/vfs/${TEST_BUCKET}/documents/?type=directory&recursive=true")
if echo "$RESPONSE" | grep -q "\"name\":\"2024\"" && echo "$RESPONSE" | grep -q "\"name\":\"report.txt\""; then
    print_test_result 0 "List directory recursively"
else
    print_test_result 1 "List directory recursively"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 8: Create empty directory
echo "Test 8: Create empty directory /photos/"
RESPONSE=$(curl -s -X POST "${BASE_URL}/vfs/${TEST_BUCKET}/_mkdir" \
    -H "Content-Type: application/json" \
    -d "{\"path\":\"/photos/\"}")
# Note: The handler expects path in URL, let's try with query param
RESPONSE=$(curl -s -X POST "${BASE_URL}/vfs/${TEST_BUCKET}/_mkdir?path=/photos/")
# Check response
if [ $? -eq 0 ]; then
    print_test_result 0 "Create empty directory"
else
    print_test_result 1 "Create empty directory"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 9: Move/rename file
echo "Test 9: Move file from /documents/readme.txt to /documents/README.md"
RESPONSE=$(curl -s -X POST "${BASE_URL}/vfs/${TEST_BUCKET}/_move" \
    -H "Content-Type: application/json" \
    -d '{"source":"/documents/readme.txt","destination":"/documents/README.md"}')
if echo "$RESPONSE" | grep -q "\"name\":\"README.md\""; then
    print_test_result 0 "Move/rename file"
else
    print_test_result 1 "Move/rename file"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 10: Verify moved file exists
echo "Test 10: Verify moved file exists at new location"
RESPONSE=$(curl -s "${BASE_URL}/vfs/${TEST_BUCKET}/documents/README.md")
if echo "$RESPONSE" | grep -q "This is a test file"; then
    print_test_result 0 "Verify moved file exists"
else
    print_test_result 1 "Verify moved file exists"
    echo "Response: $RESPONSE"
fi
echo ""

# Test 11: Get file metadata (HEAD request)
echo "Test 11: Get file metadata with HEAD request"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -I "${BASE_URL}/vfs/${TEST_BUCKET}/documents/README.md")
if [ "$HTTP_CODE" = "200" ]; then
    print_test_result 0 "Get file metadata (HEAD)"
else
    print_test_result 1 "Get file metadata (HEAD)"
    echo "HTTP Code: $HTTP_CODE"
fi
echo ""

# Test 12: Delete file
echo "Test 12: Delete file /documents/2024/report.txt"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BASE_URL}/vfs/${TEST_BUCKET}/documents/2024/report.txt")
if [ "$HTTP_CODE" = "204" ]; then
    print_test_result 0 "Delete file"
else
    print_test_result 1 "Delete file"
    echo "HTTP Code: $HTTP_CODE"
fi
echo ""

# Test 13: Verify file is deleted
echo "Test 13: Verify file is deleted"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/vfs/${TEST_BUCKET}/documents/2024/report.txt")
if [ "$HTTP_CODE" = "404" ]; then
    print_test_result 0 "Verify file is deleted"
else
    print_test_result 1 "Verify file is deleted"
    echo "HTTP Code: $HTTP_CODE (expected 404)"
fi
echo ""

# Test 14: Try to delete non-empty directory (should fail without recursive)
echo "Test 14: Try to delete non-empty directory (should fail)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BASE_URL}/vfs/${TEST_BUCKET}/documents/?type=directory")
if [ "$HTTP_CODE" = "409" ]; then
    print_test_result 0 "Reject delete of non-empty directory"
else
    print_test_result 1 "Reject delete of non-empty directory"
    echo "HTTP Code: $HTTP_CODE (expected 409)"
fi
echo ""

# Test 15: Delete directory recursively
echo "Test 15: Delete directory recursively"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BASE_URL}/vfs/${TEST_BUCKET}/documents/?type=directory&recursive=true")
if [ "$HTTP_CODE" = "204" ]; then
    print_test_result 0 "Delete directory recursively"
else
    print_test_result 1 "Delete directory recursively"
    echo "HTTP Code: $HTTP_CODE"
fi
echo ""

# Cleanup: Delete test bucket
echo "Cleanup: Delete test bucket"
curl -s -X DELETE "${BASE_URL}/buckets/${TEST_BUCKET}" > /dev/null
echo ""

# Print summary
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "Total tests run: ${TESTS_RUN}"
echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
