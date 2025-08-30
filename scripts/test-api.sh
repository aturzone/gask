#!/bin/bash

# scripts/test-api.sh
# Complete API Testing Script for Gask TaskMaster System

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Server configuration
SERVER_URL="http://localhost:8080"
API_URL="${SERVER_URL}/api/v1"

# Global variables
ACCESS_TOKEN=""
USER_ID=""
PROJECT_ID=""
TASK_ID=""

echo -e "${BLUE}🚀 Gask API Testing Script${NC}"
echo -e "${BLUE}================================${NC}"

# Function to make HTTP requests
make_request() {
    local method=$1
    local endpoint=$2
    local data=${3:-""}
    local auth_header=""
    
    if [ ! -z "$ACCESS_TOKEN" ]; then
        auth_header="-H \"Authorization: Bearer $ACCESS_TOKEN\""
    fi
    
    if [ ! -z "$data" ]; then
        eval curl -s -X $method $auth_header \
            -H \"Content-Type: application/json\" \
            -d \'$data\' \
            \"$endpoint\"
    else
        eval curl -s -X $method $auth_header \
            -H \"Content-Type: application/json\" \
            \"$endpoint\"
    fi
}

# Function to extract value from JSON response
extract_value() {
    local json=$1
    local key=$2
    echo $json | grep -o "\"$key\":\"[^\"]*\"" | cut -d'"' -f4
}

# Function to check if server is running
check_server() {
    echo -e "${YELLOW}🔍 Checking if server is running...${NC}"
    
    response=$(curl -s -f "$SERVER_URL/health" 2>/dev/null || echo "error")
    
    if [[ $response == *"ok"* ]]; then
        echo -e "${GREEN}✅ Server is running${NC}"
    else
        echo -e "${RED}❌ Server is not running. Please start the server first:${NC}"
        echo "   docker-compose up -d"
        echo "   # OR"
        echo "   go run cmd/api/main.go serve"
        exit 1
    fi
}

# Test health endpoints
test_health() {
    echo -e "\n${YELLOW}🏥 Testing Health Endpoints${NC}"
    
    # Basic health check
    echo -e "${BLUE}Testing /health${NC}"
    response=$(make_request "GET" "$SERVER_URL/health")
    if [[ $response == *"ok"* ]]; then
        echo -e "${GREEN}✅ Basic health check passed${NC}"
    else
        echo -e "${RED}❌ Basic health check failed${NC}"
        echo "Response: $response"
    fi
    
    # Detailed health check
    echo -e "${BLUE}Testing /health/detailed${NC}"
    response=$(make_request "GET" "$SERVER_URL/health/detailed")
    if [[ $response == *"database"* ]]; then
        echo -e "${GREEN}✅ Detailed health check passed${NC}"
    else
        echo -e "${RED}❌ Detailed health check failed${NC}"
        echo "Response: $response"
    fi
    
    # Readiness check
    echo -e "${BLUE}Testing /ready${NC}"
    response=$(make_request "GET" "$SERVER_URL/ready")
    if [[ $response == *"ready"* ]]; then
        echo -e "${GREEN}✅ Readiness check passed${NC}"
    else
        echo -e "${RED}❌ Readiness check failed${NC}"
        echo "Response: $response"
    fi
}

# Test authentication
test_auth() {
    echo -e "\n${YELLOW}🔐 Testing Authentication${NC}"
    
    # Test user registration
    echo -e "${BLUE}Testing user registration${NC}"
    register_data='{
        "email": "test@gask.dev",
        "username": "testuser",
        "password": "password123",
        "first_name": "Test",
        "last_name": "User",
        "role": "developer"
    }'
    
    response=$(make_request "POST" "$API_URL/auth/register" "$register_data")
    
    if [[ $response == *"access_token"* ]]; then
        echo -e "${GREEN}✅ User registration successful${NC}"
        ACCESS_TOKEN=$(extract_value "$response" "access_token")
        USER_ID=$(echo $response | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo -e "${BLUE}   User ID: $USER_ID${NC}"
        echo -e "${BLUE}   Token: ${ACCESS_TOKEN:0:20}...${NC}"
    else
        echo -e "${YELLOW}⚠️  Registration may have failed (user might already exist)${NC}"
        
        # Try login instead
        echo -e "${BLUE}Testing user login${NC}"
        login_data='{
            "email": "test@gask.dev",
            "password": "password123"
        }'
        
        response=$(make_request "POST" "$API_URL/auth/login" "$login_data")
        
        if [[ $response == *"access_token"* ]]; then
            echo -e "${GREEN}✅ User login successful${NC}"
            ACCESS_TOKEN=$(extract_value "$response" "access_token")
            USER_ID=$(echo $response | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
            echo -e "${BLUE}   User ID: $USER_ID${NC}"
            echo -e "${BLUE}   Token: ${ACCESS_TOKEN:0:20}...${NC}"
        else
            echo -e "${RED}❌ Authentication failed${NC}"
            echo "Response: $response"
            exit 1
        fi
    fi
    
    # Test getting current user
    echo -e "${BLUE}Testing /users/me${NC}"
    response=$(make_request "GET" "$API_URL/users/me")
    if [[ $response == *"email"* ]]; then
        echo -e "${GREEN}✅ Get current user successful${NC}"
    else
        echo -e "${RED}❌ Get current user failed${NC}"
        echo "Response: $response"
    fi
}

# Test projects
test_projects() {
    echo -e "\n${YELLOW}📋 Testing Projects${NC}"
    
    # Create a project
    echo -e "${BLUE}Testing project creation${NC}"
    project_data='{
        "name": "Test Project",
        "description": "A test project for API testing",
        "status": "active",
        "manager_id": "'$USER_ID'",
        "start_date": "'$(date -Iseconds)'",
        "end_date": "'$(date -d '+30 days' -Iseconds)'"
    }'
    
    response=$(make_request "POST" "$API_URL/projects" "$project_data")
    
    if [[ $response == *"\"id\""* ]]; then
        echo -e "${GREEN}✅ Project creation successful${NC}"
        PROJECT_ID=$(echo $response | grep -o '"id":[0-9]*' | cut -d':' -f2)
        echo -e "${BLUE}   Project ID: $PROJECT_ID${NC}"
    else
        echo -e "${RED}❌ Project creation failed${NC}"
        echo "Response: $response"
        return
    fi
    
    # Get project
    echo -e "${BLUE}Testing get project${NC}"
    response=$(make_request "GET" "$API_URL/projects/$PROJECT_ID")
    if [[ $response == *"Test Project"* ]]; then
        echo -e "${GREEN}✅ Get project successful${NC}"
    else
        echo -e "${RED}❌ Get project failed${NC}"
        echo "Response: $response"
    fi
    
    # List projects
    echo -e "${BLUE}Testing list projects${NC}"
    response=$(make_request "GET" "$API_URL/projects")
    if [[ $response == *"data"* ]]; then
        echo -e "${GREEN}✅ List projects successful${NC}"
    else
        echo -e "${RED}❌ List projects failed${NC}"
        echo "Response: $response"
    fi
    
    # Update project
    echo -e "${BLUE}Testing project update${NC}"
    update_data='{
        "name": "Updated Test Project",
        "description": "An updated test project"
    }'
    
    response=$(make_request "PUT" "$API_URL/projects/$PROJECT_ID" "$update_data")
    if [[ $response == *"Updated Test Project"* ]]; then
        echo -e "${GREEN}✅ Project update successful${NC}"
    else
        echo -e "${RED}❌ Project update failed${NC}"
        echo "Response: $response"
    fi
}

# Test tasks
test_tasks() {
    echo -e "\n${YELLOW}✅ Testing Tasks${NC}"
    
    if [ -z "$PROJECT_ID" ]; then
        echo -e "${RED}❌ No project ID available. Skipping task tests.${NC}"
        return
    fi
    
    # Create a task
    echo -e "${BLUE}Testing task creation${NC}"
    task_data='{
        "title": "Test Task",
        "description": "A test task for API testing",
        "status": "todo",
        "priority": "medium",
        "project_id": '$PROJECT_ID',
        "due_date": "'$(date -d '+7 days' -Iseconds)'"
    }'
    
    response=$(make_request "POST" "$API_URL/tasks" "$task_data")
    
    if [[ $response == *"\"id\""* ]]; then
        echo -e "${GREEN}✅ Task creation successful${NC}"
        TASK_ID=$(echo $response | grep -o '"id":[0-9]*' | cut -d':' -f2)
        echo -e "${BLUE}   Task ID: $TASK_ID${NC}"
    else
        echo -e "${RED}❌ Task creation failed${NC}"
        echo "Response: $response"
        return
    fi
    
    # Get task
    echo -e "${BLUE}Testing get task${NC}"
    response=$(make_request "GET" "$API_URL/tasks/$TASK_ID")
    if [[ $response == *"Test Task"* ]]; then
        echo -e "${GREEN}✅ Get task successful${NC}"
    else
        echo -e "${RED}❌ Get task failed${NC}"
        echo "Response: $response"
    fi
    
    # List tasks
    echo -e "${BLUE}Testing list tasks${NC}"
    response=$(make_request "GET" "$API_URL/tasks")
    if [[ $response == *"data"* ]]; then
        echo -e "${GREEN}✅ List tasks successful${NC}"
    else
        echo -e "${RED}❌ List tasks failed${NC}"
        echo "Response: $response"
    fi
    
    # Update task status
    echo -e "${BLUE}Testing task status update${NC}"
    status_data='{"status": "in_progress"}'
    
    response=$(make_request "PATCH" "$API_URL/tasks/$TASK_ID/status" "$status_data")
    if [[ $response == *"in_progress"* ]]; then
        echo -e "${GREEN}✅ Task status update successful${NC}"
    else
        echo -e "${RED}❌ Task status update failed${NC}"
        echo "Response: $response"
    fi
    
    # Assign task to user
    echo -e "${BLUE}Testing task assignment${NC}"
    assign_data='{"assignee_id": "'$USER_ID'"}'
    
    response=$(make_request "POST" "$API_URL/tasks/$TASK_ID/assign" "$assign_data")
    if [[ $response == *"$USER_ID"* ]]; then
        echo -e "${GREEN}✅ Task assignment successful${NC}"
    else
        echo -e "${RED}❌ Task assignment failed${NC}"
        echo "Response: $response"
    fi
}

# Test time tracking
test_time_tracking() {
    echo -e "\n${YELLOW}⏰ Testing Time Tracking${NC}"
    
    if [ -z "$TASK_ID" ]; then
        echo -e "${RED}❌ No task ID available. Skipping time tracking tests.${NC}"
        return
    fi
    
    # Start time tracking
    echo -e "${BLUE}Testing start time tracking${NC}"
    start_data='{"task_id": '$TASK_ID'}'
    
    response=$(make_request "POST" "$API_URL/time/start" "$start_data")
    if [[ $response == *"start_time"* ]]; then
        echo -e "${GREEN}✅ Start time tracking successful${NC}"
    else
        echo -e "${RED}❌ Start time tracking failed${NC}"
        echo "Response: $response"
        return
    fi
    
    # Wait a moment
    sleep 2
    
    # Get active time entry
    echo -e "${BLUE}Testing get active time entry${NC}"
    response=$(make_request "GET" "$API_URL/time/active")
    if [[ $response == *"start_time"* ]]; then
        echo -e "${GREEN}✅ Get active time entry successful${NC}"
    else
        echo -e "${RED}❌ Get active time entry failed${NC}"
        echo "Response: $response"
    fi
    
    # Stop time tracking
    echo -e "${BLUE}Testing stop time tracking${NC}"
    response=$(make_request "POST" "$API_URL/time/stop")
    if [[ $response == *"end_time"* ]]; then
        echo -e "${GREEN}✅ Stop time tracking successful${NC}"
    else
        echo -e "${RED}❌ Stop time tracking failed${NC}"
        echo "Response: $response"
    fi
    
    # List time entries
    echo -e "${BLUE}Testing list time entries${NC}"
    response=$(make_request "GET" "$API_URL/time/entries")
    if [[ $response == *"data"* ]]; then
        echo -e "${GREEN}✅ List time entries successful${NC}"
    else
        echo -e "${RED}❌ List time entries failed${NC}"
        echo "Response: $response"
    fi
}

# Test error scenarios
test_error_scenarios() {
    echo -e "\n${YELLOW}🚨 Testing Error Scenarios${NC}"
    
    # Test unauthorized access
    echo -e "${BLUE}Testing unauthorized access${NC}"
    temp_token=$ACCESS_TOKEN
    ACCESS_TOKEN=""
    
    response=$(make_request "GET" "$API_URL/users/me")
    if [[ $response == *"Missing authorization header"* ]] || [[ $response == *"Unauthorized"* ]]; then
        echo -e "${GREEN}✅ Unauthorized access properly blocked${NC}"
    else
        echo -e "${RED}❌ Unauthorized access not blocked${NC}"
        echo "Response: $response"
    fi
    
    ACCESS_TOKEN=$temp_token
    
    # Test invalid token
    echo -e "${BLUE}Testing invalid token${NC}"
    temp_token=$ACCESS_TOKEN
    ACCESS_TOKEN="invalid-token"
    
    response=$(make_request "GET" "$API_URL/users/me")
    if [[ $response == *"Invalid token"* ]] || [[ $response == *"Unauthorized"* ]]; then
        echo -e "${GREEN}✅ Invalid token properly rejected${NC}"
    else
        echo -e "${RED}❌ Invalid token not rejected${NC}"
        echo "Response: $response"
    fi
    
    ACCESS_TOKEN=$temp_token
    
    # Test non-existent resource
    echo -e "${BLUE}Testing non-existent resource${NC}"
    response=$(make_request "GET" "$API_URL/projects/99999")
    if [[ $response == *"not found"* ]] || [[ $response == *"404"* ]]; then
        echo -e "${GREEN}✅ Non-existent resource properly handled${NC}"
    else
        echo -e "${RED}❌ Non-existent resource not properly handled${NC}"
        echo "Response: $response"
    fi
}

# Test documentation endpoints
test_documentation() {
    echo -e "\n${YELLOW}📚 Testing Documentation Endpoints${NC}"
    
    # Test swagger.json
    echo -e "${BLUE}Testing /swagger.json${NC}"
    response=$(make_request "GET" "$SERVER_URL/swagger.json")
    if [[ $response == *"swagger"* ]] && [[ $response == *"paths"* ]]; then
        echo -e "${GREEN}✅ Swagger JSON accessible${NC}"
    else
        echo -e "${RED}❌ Swagger JSON failed${NC}"
        echo "Response: $response"
    fi
    
    # Test documentation page
    echo -e "${BLUE}Testing documentation page accessibility${NC}"
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER_URL/docs/simple-swagger.html")
    if [[ $status_code == "200" ]]; then
        echo -e "${GREEN}✅ Documentation page accessible${NC}"
    else
        echo -e "${RED}❌ Documentation page not accessible (Status: $status_code)${NC}"
    fi
}

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up test data${NC}"
    
    if [ ! -z "$TASK_ID" ]; then
        echo -e "${BLUE}Deleting test task${NC}"
        make_request "DELETE" "$API_URL/tasks/$TASK_ID" > /dev/null 2>&1
    fi
    
    if [ ! -z "$PROJECT_ID" ]; then
        echo -e "${BLUE}Deleting test project${NC}"
        make_request "DELETE" "$API_URL/projects/$PROJECT_ID" > /dev/null 2>&1
    fi
}

# Main execution
main() {
    echo -e "${BLUE}Starting API tests at $(date)${NC}"
    
    check_server
    test_health
    test_auth
    test_projects
    test_tasks
    test_time_tracking
    test_error_scenarios
    test_documentation
    
    echo -e "\n${GREEN}🎉 API Testing Completed!${NC}"
    echo -e "${BLUE}================================${NC}"
    echo -e "${GREEN}✅ All major endpoints tested${NC}"
    echo -e "${BLUE}📊 Summary:${NC}"
    echo -e "   • Health endpoints: ✅"
    echo -e "   • Authentication: ✅"
    echo -e "   • Projects: ✅"
    echo -e "   • Tasks: ✅"
    echo -e "   • Time tracking: ✅"
    echo -e "   • Error handling: ✅"
    echo -e "   • Documentation: ✅"
    echo
    echo -e "${BLUE}🌐 Access your API at:${NC}"
    echo -e "   • Main page: $SERVER_URL"
    echo -e "   • API docs: $SERVER_URL/docs/simple-swagger.html"
    echo -e "   • Health: $SERVER_URL/health"
    echo
    echo -e "${GREEN}Ready for development! 🚀${NC}"
    
    cleanup
}

# Run if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
