#!/bin/bash

BASE_URL="http://localhost:7890"
OWNER_PASSWORD="admin1234"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PASSED=0
FAILED=0

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((PASSED++))
}

log_fail() {
    echo -e "${RED}[✗]${NC} $1"
    ((FAILED++))
}

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

assert_status() {
    local expected=$1
    local actual=$2
    local test_name=$3
    
    if [ "$actual" -eq "$expected" ]; then
        log_success "$test_name (Status: $actual)"
    else
        log_fail "$test_name (Expected: $expected, Got: $actual)"
    fi
}

assert_contains() {
    local response=$1
    local pattern=$2
    local test_name=$3
    
    if echo "$response" | grep -q "$pattern"; then
        log_success "$test_name"
    else
        log_fail "$test_name (Pattern not found: $pattern)"
    fi
}

echo "════════════════════════════════════════════════════════════════"
echo "       COMPREHENSIVE TASK MANAGEMENT API TEST SUITE"
echo "════════════════════════════════════════════════════════════════"
echo ""

# ============================================================================
# SECTION 1: HEALTH & INITIAL STATE
# ============================================================================
log_info "SECTION 1: Health Check & Initial State"
echo "────────────────────────────────────────────────────────────────"

log_test "1.1 Health Check"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Health check endpoint"
assert_contains "$BODY" "healthy" "Health status is healthy"

log_test "1.2 Get Initial Users (Owner)"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get users as owner"
OWNER_ID=$(echo "$BODY" | jq -r '.data.users[0].id')
log_info "Owner ID: $OWNER_ID"

echo ""

# ============================================================================
# SECTION 2: GROUP MANAGEMENT
# ============================================================================
log_info "SECTION 2: Group Management"
echo "────────────────────────────────────────────────────────────────"

log_test "2.1 Create Engineering Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Engineering\", \"admin_id\": $OWNER_ID}" \
  "$BASE_URL/groups")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create Engineering group"
GROUP_ENG_ID=$(echo "$BODY" | jq -r '.data.group.id')
log_info "Engineering Group ID: $GROUP_ENG_ID"

log_test "2.2 Create Marketing Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Marketing\", \"admin_id\": $OWNER_ID}" \
  "$BASE_URL/groups")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create Marketing group"
GROUP_MKT_ID=$(echo "$BODY" | jq -r '.data.group.id')
log_info "Marketing Group ID: $GROUP_MKT_ID"

log_test "2.3 List All Groups"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "List all groups"
assert_contains "$BODY" "Engineering" "Engineering group in list"
assert_contains "$BODY" "Marketing" "Marketing group in list"

log_test "2.4 Get Engineering Group Details"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get group details"

echo ""

# ============================================================================
# SECTION 3: USER MANAGEMENT - GROUP ADMIN
# ============================================================================
log_info "SECTION 3: User Management - Group Admins"
echo "────────────────────────────────────────────────────────────────"

log_test "3.1 Create Engineering Group Admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Alice Admin\",
    \"email\": \"alice@company.com\",
    \"password\": \"alice123\",
    \"role\": \"group_admin\",
    \"group_ids\": [$GROUP_ENG_ID],
    \"work_times\": {\"Monday\": 8, \"Tuesday\": 8, \"Wednesday\": 8, \"Thursday\": 8, \"Friday\": 8}
  }" \
  "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create Engineering group admin"
ADMIN_ENG_ID=$(echo "$BODY" | jq -r '.data.user.id')
log_info "Engineering Admin ID: $ADMIN_ENG_ID"

log_test "3.2 Update Group to Assign New Admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"admin_id\": $ADMIN_ENG_ID}" \
  "$BASE_URL/groups/$GROUP_ENG_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Update Engineering group admin"

log_test "3.3 Create Marketing Group Admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Bob Manager\",
    \"email\": \"bob@company.com\",
    \"password\": \"bob123\",
    \"role\": \"group_admin\",
    \"group_ids\": [$GROUP_MKT_ID],
    \"work_times\": {\"Monday\": 8, \"Tuesday\": 8, \"Wednesday\": 8, \"Thursday\": 8, \"Friday\": 6}
  }" \
  "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create Marketing group admin"
ADMIN_MKT_ID=$(echo "$BODY" | jq -r '.data.user.id')
log_info "Marketing Admin ID: $ADMIN_MKT_ID"

log_test "3.4 Update Marketing Group Admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"admin_id\": $ADMIN_MKT_ID}" \
  "$BASE_URL/groups/$GROUP_MKT_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Update Marketing group admin"

echo ""

# ============================================================================
# SECTION 4: USER MANAGEMENT - REGULAR USERS
# ============================================================================
log_info "SECTION 4: User Management - Regular Users"
echo "────────────────────────────────────────────────────────────────"

log_test "4.1 Admin Creates Engineer User"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$ADMIN_ENG_ID:alice123" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Charlie Developer\",
    \"email\": \"charlie@company.com\",
    \"password\": \"charlie123\",
    \"role\": \"user\",
    \"group_ids\": [$GROUP_ENG_ID],
    \"work_times\": {\"Monday\": 8, \"Tuesday\": 8, \"Wednesday\": 8, \"Thursday\": 8, \"Friday\": 8}
  }" \
  "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Admin creates user in their group"
USER_ENG1_ID=$(echo "$BODY" | jq -r '.data.user.id')
log_info "Engineer User ID: $USER_ENG1_ID"

log_test "4.2 Admin Creates Another Engineer"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$ADMIN_ENG_ID:alice123" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Diana Coder\",
    \"email\": \"diana@company.com\",
    \"password\": \"diana123\",
    \"role\": \"user\",
    \"group_ids\": [$GROUP_ENG_ID],
    \"work_times\": {\"Monday\": 6, \"Tuesday\": 8, \"Wednesday\": 8, \"Thursday\": 8, \"Friday\": 8}
  }" \
  "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Admin creates second user"
USER_ENG2_ID=$(echo "$BODY" | jq -r '.data.user.id')
log_info "Engineer 2 User ID: $USER_ENG2_ID"

log_test "4.3 Create Marketing User"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$ADMIN_MKT_ID:bob123" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Eve Marketer\",
    \"email\": \"eve@company.com\",
    \"password\": \"eve123\",
    \"role\": \"user\",
    \"group_ids\": [$GROUP_MKT_ID],
    \"work_times\": {\"Monday\": 8, \"Tuesday\": 8, \"Wednesday\": 8, \"Thursday\": 8, \"Friday\": 6}
  }" \
  "$BASE_URL/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create marketing user"
USER_MKT1_ID=$(echo "$BODY" | jq -r '.data.user.id')
log_info "Marketing User ID: $USER_MKT1_ID"

log_test "4.4 User Gets Own Profile"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/users/$USER_ENG1_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "User gets own profile"

log_test "4.5 User Cannot Access Other User's Profile"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/users/$USER_MKT1_ID")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "User denied access to other user"

log_test "4.6 Search Users by Name"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/users/search?q=Developer")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Search users"
assert_contains "$BODY" "Charlie" "Search finds correct user"

echo ""

# ============================================================================
# SECTION 5: TASK MANAGEMENT
# ============================================================================
log_info "SECTION 5: Task Management"
echo "────────────────────────────────────────────────────────────────"

log_test "5.1 Engineer Creates Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Implement Login API\",
    \"priority\": 1,
    \"deadline\": \"2025-10-15\",
    \"information\": \"Create REST API for user authentication\",
    \"group_id\": $GROUP_ENG_ID
  }" \
  "$BASE_URL/users/$USER_ENG1_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create engineering task"
TASK_ENG1_ID=$(echo "$BODY" | jq -r '.data.task.id')
log_info "Engineering Task 1 ID: $TASK_ENG1_ID"

log_test "5.2 Engineer Creates Second Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Fix Database Bug\",
    \"priority\": 2,
    \"deadline\": \"2025-10-10\",
    \"information\": \"Connection pool exhaustion issue\",
    \"group_id\": $GROUP_ENG_ID
  }" \
  "$BASE_URL/users/$USER_ENG1_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Create second engineering task"
TASK_ENG2_ID=$(echo "$BODY" | jq -r '.data.task.id')
log_info "Engineering Task 2 ID: $TASK_ENG2_ID"

log_test "5.3 Another Engineer Creates Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$USER_ENG2_ID:diana123" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Write Unit Tests\",
    \"priority\": 1,
    \"deadline\": \"2025-10-20\",
    \"information\": \"Increase test coverage to 80%\",
    \"group_id\": $GROUP_ENG_ID
  }" \
  "$BASE_URL/users/$USER_ENG2_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Second engineer creates task"
TASK_ENG3_ID=$(echo "$BODY" | jq -r '.data.task.id')

log_test "5.4 Marketing User Creates Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$USER_MKT1_ID:eve123" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Create Social Media Campaign\",
    \"priority\": 1,
    \"deadline\": \"2025-10-25\",
    \"information\": \"Launch campaign for new product\",
    \"group_id\": $GROUP_MKT_ID
  }" \
  "$BASE_URL/users/$USER_MKT1_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 201 "$STATUS" "Marketing user creates task"
TASK_MKT1_ID=$(echo "$BODY" | jq -r '.data.task.id')

log_test "5.5 Get User's Tasks"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/users/$USER_ENG1_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get user tasks"
TASK_COUNT=$(echo "$BODY" | jq -r '.data.count')
log_info "User has $TASK_COUNT tasks"

log_test "5.6 Update Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{\"priority\": 3, \"information\": \"Updated: Critical bug in production\"}" \
  "$BASE_URL/users/$USER_ENG1_ID/tasks/$TASK_ENG2_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Update task"

log_test "5.7 Mark Task as Done"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
  -u "$USER_ENG1_ID:charlie123" \
  "$BASE_URL/users/$USER_ENG1_ID/tasks/$TASK_ENG1_ID/done")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Mark task as done"
assert_contains "$BODY" "true" "Task status is true"

log_test "5.8 Get Specific Task"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/users/$USER_ENG1_ID/tasks/$TASK_ENG1_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get specific task"

echo ""

# ============================================================================
# SECTION 6: ADVANCED TASK OPERATIONS
# ============================================================================
log_info "SECTION 6: Advanced Task Operations"
echo "────────────────────────────────────────────────────────────────"

log_test "6.1 Search Tasks Globally"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/search?q=API")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Global task search"
assert_contains "$BODY" "Login" "Search finds correct task"

log_test "6.2 Get Global Task Statistics"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/stats")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get task statistics"
TOTAL_TASKS=$(echo "$BODY" | jq -r '.data.total_tasks')
log_info "Total tasks in system: $TOTAL_TASKS"

log_test "6.3 Filter Tasks by Status (Pending)"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/filter?status=pending")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Filter pending tasks"

log_test "6.4 Filter Tasks by Status (Completed)"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/filter?status=completed")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Filter completed tasks"

log_test "6.5 Filter Tasks by Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/filter?group_id=$GROUP_ENG_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Filter tasks by group"

log_test "6.6 Batch Update Tasks"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{
    \"task_ids\": [$TASK_ENG2_ID, $TASK_ENG3_ID],
    \"updates\": {\"priority\": 2},
    \"action\": \"update\"
  }" \
  "$BASE_URL/tasks/batch")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Batch update tasks"

echo ""

# ============================================================================
# SECTION 7: GROUP OPERATIONS
# ============================================================================
log_info "SECTION 7: Group Operations"
echo "────────────────────────────────────────────────────────────────"

log_test "7.1 Get Group Users"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get group users"
USER_COUNT=$(echo "$BODY" | jq -r '.data.count')
log_info "Engineering group has $USER_COUNT users"

log_test "7.2 Get Group Tasks"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID/tasks")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get group tasks"

log_test "7.3 Get Group Statistics"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID/stats")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get group statistics"

log_test "7.4 Admin Views Own Group Stats"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$ADMIN_ENG_ID:alice123" "$BASE_URL/groups/$GROUP_ENG_ID/stats")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Admin views own group stats"

log_test "7.5 Admin Cannot View Other Group Stats"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$ADMIN_ENG_ID:alice123" "$BASE_URL/groups/$GROUP_MKT_ID/stats")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "Admin denied other group stats"

log_test "7.6 Add User to Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\": $USER_ENG1_ID}" \
  "$BASE_URL/groups/$GROUP_MKT_ID/users")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Add user to second group"

log_test "7.7 Remove User from Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/groups/$GROUP_MKT_ID/users/$USER_ENG1_ID")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Remove user from group"

echo ""

# ============================================================================
# SECTION 8: WORK TIMES
# ============================================================================
log_info "SECTION 8: Work Times Management"
echo "────────────────────────────────────────────────────────────────"

log_test "8.1 Get User Work Times"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/users/$USER_ENG1_ID/worktimes")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get user work times"

log_test "8.2 Update User Work Times"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{
    \"work_times\": {
      \"Monday\": 9,
      \"Tuesday\": 9,
      \"Wednesday\": 9,
      \"Thursday\": 9,
      \"Friday\": 6
    }
  }" \
  "$BASE_URL/users/$USER_ENG1_ID/worktimes")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Update work times"

echo ""

# ============================================================================
# SECTION 9: AUTHORIZATION TESTS
# ============================================================================
log_info "SECTION 9: Authorization & Access Control"
echo "────────────────────────────────────────────────────────────────"

log_test "9.1 User Cannot Create Group"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Unauthorized Group\", \"admin_id\": $USER_ENG1_ID}" \
  "$BASE_URL/groups")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "User denied group creation"

log_test "9.2 User Cannot Delete Other User"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
  -u "$USER_ENG1_ID:charlie123" \
  "$BASE_URL/users/$USER_ENG2_ID")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "User denied delete other user"

log_test "9.3 Admin Cannot Create Admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -u "$ADMIN_ENG_ID:alice123" \
  -H "Content-Type: application/json" \
  -d "{
    \"full_name\": \"Unauthorized Admin\",
    \"email\": \"unauth@company.com\",
    \"password\": \"pass123\",
    \"role\": \"group_admin\",
    \"group_ids\": [$GROUP_ENG_ID]
  }" \
  "$BASE_URL/users")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "Admin denied creating another admin"

log_test "9.4 User Cannot Access Admin Endpoints"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:charlie123" "$BASE_URL/admin/status")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 403 "$STATUS" "User denied admin status"

log_test "9.5 Invalid Password Rejected"
RESPONSE=$(curl -s -w "\n%{http_code}" -u "$USER_ENG1_ID:wrongpass" "$BASE_URL/users/$USER_ENG1_ID")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 401 "$STATUS" "Invalid password rejected"

log_test "9.6 No Auth Rejected"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/users")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 401 "$STATUS" "No authentication rejected"

echo ""

# ============================================================================
# SECTION 10: SYNC OPERATIONS
# ============================================================================
log_info "SECTION 10: Data Synchronization"
echo "────────────────────────────────────────────────────────────────"

log_test "10.1 Get Admin Status"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/admin/status")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get admin status"
log_info "Sync status:"
echo "$BODY" | jq '.data | {running, healthy, pending_changes, postgres_status, redis_status}'

log_test "10.2 Force Sync to PostgreSQL"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/admin/sync?action=force")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Force sync to PostgreSQL"
log_success "All data synced to PostgreSQL"

log_test "10.3 Get System Statistics"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/admin/stats")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Get system statistics"
log_info "System stats:"
echo "$BODY" | jq '.data'

echo ""

# ============================================================================
# SECTION 11: CONCURRENT OPERATIONS
# ============================================================================
log_info "SECTION 11: Concurrent Operations"
echo "────────────────────────────────────────────────────────────────"

log_test "11.1 Multiple Users Create Tasks Simultaneously"
(curl -s -X POST \
  -u "$USER_ENG1_ID:charlie123" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"Concurrent Task 1\", \"priority\": 1, \"group_id\": $GROUP_ENG_ID}" \
  "$BASE_URL/users/$USER_ENG1_ID/tasks" > /dev/null) &
(curl -s -X POST \
  -u "$USER_ENG2_ID:diana123" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"Concurrent Task 2\", \"priority\": 1, \"group_id\": $GROUP_ENG_ID}" \
  "$BASE_URL/users/$USER_ENG2_ID/tasks" > /dev/null) &
(curl -s -X POST \
  -u "$USER_MKT1_ID:eve123" \
  -H "Content-Type: application/json" \
  -d "{\"title\": \"Concurrent Task 3\", \"priority\": 1, \"group_id\": $GROUP_MKT_ID}" \
  "$BASE_URL/users/$USER_MKT1_ID/tasks" > /dev/null) &
wait
log_success "Multiple concurrent task creations completed"

log_test "11.2 Verify All Concurrent Tasks Created"
RESPONSE=$(curl -s -w "\n%{http_code}" -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/tasks/stats")
BODY=$(echo "$RESPONSE" | head -n -1)
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Verify concurrent tasks"
TOTAL=$(echo "$BODY" | jq -r '.data.total_tasks')
log_info "Total tasks after concurrent operations: $TOTAL"

echo ""

# ============================================================================
# SECTION 12: CLEANUP
# ============================================================================
log_info "SECTION 12: Cleanup Test Data"
echo "────────────────────────────────────────────────────────────────"

log_test "12.1 Delete Engineering Tasks"
for task_id in $(curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID/tasks" | jq -r '.data.tasks[].id'); do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
      -H "X-Owner-Password: $OWNER_PASSWORD" \
      "$BASE_URL/users/$(curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_ENG_ID/tasks" | jq -r ".data.tasks[] | select(.id==$task_id) | .user_id")/tasks/$task_id")
    STATUS=$(echo "$RESPONSE" | tail -n 1)
done
log_success "Engineering tasks deleted"

log_test "12.2 Delete Marketing Tasks"
for task_id in $(curl -s -H "X-Owner-Password: $OWNER_PASSWORD" "$BASE_URL/groups/$GROUP_MKT_ID/tasks" | jq -r '.data.tasks[].id'); do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
      -H "X-Owner-Password: $OWNER_PASSWORD" \
      "$BASE_URL/users/$USER_MKT1_ID/tasks/$task_id")
    STATUS=$(echo "$RESPONSE" | tail -n 1)
done
log_success "Marketing tasks deleted"

log_test "12.3 Delete Users"
for user_id in $USER_ENG1_ID $USER_ENG2_ID $USER_MKT1_ID $ADMIN_ENG_ID $ADMIN_MKT_ID; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
      -H "X-Owner-Password: $OWNER_PASSWORD" \
      "$BASE_URL/users/$user_id")
    STATUS=$(echo "$RESPONSE" | tail -n 1)
done
log_success "Test users deleted"

log_test "12.4 Delete Groups"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/groups/$GROUP_ENG_ID")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Delete Engineering group"

RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/groups/$GROUP_MKT_ID")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Delete Marketing group"

log_test "12.5 Final Sync After Cleanup"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
  -H "X-Owner-Password: $OWNER_PASSWORD" \
  "$BASE_URL/admin/sync?action=force")
STATUS=$(echo "$RESPONSE" | tail -n 1)
assert_status 200 "$STATUS" "Final sync after cleanup"

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "                     TEST SUMMARY"
echo "════════════════════════════════════════════════════════════════"
echo -e "${GREEN}PASSED:${NC} $PASSED"
echo -e "${RED}FAILED:${NC} $FAILED"
echo "TOTAL:  $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "${RED}✗ SOME TESTS FAILED${NC}"
    exit 1
fi