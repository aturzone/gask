#!/bin/bash

# Task Management API Test Script
# Make sure the server is running on localhost:7890

BASE_URL="http://localhost:7890"
OWNER_PASSWORD="admin1234"

echo "🧪 Testing Task Management API..."
echo "=================================="

# Health Check
echo "1. Health Check"
HEALTH_RESPONSE=$(curl -s "$BASE_URL/health")
if echo "$HEALTH_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$HEALTH_RESPONSE" | jq '.'
else
  echo "Non-JSON response: $HEALTH_RESPONSE"
fi
echo ""

# Test Owner Authentication
echo "2. Testing Owner Access - Get All Users"
curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/users" | jq '.'
echo ""

# Create a Group
echo "3. Creating a Test Group"
GROUP_RESPONSE=$(curl -s -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Team",
    "admin_id": 1
  }' \
  "$BASE_URL/groups")

echo "$GROUP_RESPONSE" | jq '.'
GROUP_ID=$(echo "$GROUP_RESPONSE" | jq -r '.data.group.id // empty')
echo "Created Group ID: $GROUP_ID"
echo ""

# Create a User
echo "4. Creating a Test User"
USER_RESPONSE=$(curl -s -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Test User\",
    \"email\": \"test@example.com\",
    \"password\": \"testpass123\",
    \"role\": \"user\",
    \"group_ids\": [$GROUP_ID],
    \"work_times\": {
      \"Monday\": 8.0,
      \"Tuesday\": 8.0,
      \"Wednesday\": 8.0,
      \"Thursday\": 8.0,
      \"Friday\": 6.0
    }
  }" \
  "$BASE_URL/users")

echo "$USER_RESPONSE" | jq '.'
USER_ID=$(echo "$USER_RESPONSE" | jq -r '.data.user.id // empty')
echo "Created User ID: $USER_ID"
echo ""

# Create a Task
echo "5. Creating a Test Task"
TASK_RESPONSE=$(curl -s -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Test Task\",
    \"priority\": 1,
    \"deadline\": \"2024-12-31\",
    \"information\": \"This is a test task for API testing\",
    \"group_id\": $GROUP_ID
  }" \
  "$BASE_URL/users/$USER_ID/tasks")

echo "$TASK_RESPONSE" | jq '.'
TASK_ID=$(echo "$TASK_RESPONSE" | jq -r '.data.task.id // empty')
echo "Created Task ID: $TASK_ID"
echo ""

# Test User Authentication
echo "6. Testing User Authentication - Get Own Tasks"
curl -s -u "$USER_ID:testpass123" "$BASE_URL/users/$USER_ID/tasks" | jq '.'
echo ""

# Mark Task as Done
echo "7. Marking Task as Done"
curl -s -X PUT \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/users/$USER_ID/tasks/$TASK_ID/done" | jq '.'
echo ""

# Search Tasks
echo "8. Searching Tasks"
curl -s -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/tasks/search?q=test" | jq '.'
echo ""

# Get Task Statistics
echo "9. Getting Task Statistics"
curl -s -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/tasks/stats" | jq '.'
echo ""

# Get Group Information
echo "10. Getting Group Information"
curl -s -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/groups/$GROUP_ID" | jq '.'
echo ""

# Get Admin Status
echo "11. Getting Admin Status"
ADMIN_STATUS_RESPONSE=$(curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/admin/status")
if echo "$ADMIN_STATUS_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$ADMIN_STATUS_RESPONSE" | jq '.'
else
  echo "Non-JSON response: $ADMIN_STATUS_RESPONSE"
fi
echo ""

# Test Sync
echo "12. Testing Manual Sync"
SYNC_RESPONSE=$(curl -s -X POST -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/admin/sync?action=force")
if echo "$SYNC_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$SYNC_RESPONSE" | jq '.'
else
  echo "Non-JSON sync response: $SYNC_RESPONSE"
fi
echo ""

# Clean up - Delete Test Data
echo "13. Cleaning Up Test Data"
echo "Deleting task..."
curl -s -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/users/$USER_ID/tasks/$TASK_ID" | jq '.'

echo "Deleting user..."
curl -s -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/users/$USER_ID" | jq '.'

echo "Deleting group..."
curl -s -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/groups/$GROUP_ID" | jq '.'

echo ""
echo "✅ API Test Complete!"
echo "=================================="

# Test Error Cases
echo "14. Testing Error Cases"
echo "Testing unauthorized access..."
UNAUTH_RESPONSE=$(curl -s "$BASE_URL/users")
if echo "$UNAUTH_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$UNAUTH_RESPONSE" | jq '.'
else
  echo "Expected unauthorized response: $UNAUTH_RESPONSE"
fi
echo ""

echo "Testing invalid credentials..."
INVALID_CRED_RESPONSE=$(curl -s -u "999:wrongpass" "$BASE_URL/users/999")
if echo "$INVALID_CRED_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$INVALID_CRED_RESPONSE" | jq '.'
else
  echo "Expected unauthorized response: $INVALID_CRED_RESPONSE"
fi
echo ""

echo "Testing non-existent resource..."
NOTFOUND_RESPONSE=$(curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/users/99999")
if echo "$NOTFOUND_RESPONSE" | jq . > /dev/null 2>&1; then
  echo "$NOTFOUND_RESPONSE" | jq '.'
else
  echo "Expected not found response: $NOTFOUND_RESPONSE"
fi
echo ""

echo "🔍 Error Testing Complete!"
echo "=================================="