# TaskMaster Core - Quick Start Guide

## 🚀 5-Minute Setup

### 1. Clone and Start
```bash
# Clone the repository
git clone https://github.com/yourusername/taskmaster-core.git
cd taskmaster-core

# Start with Docker Compose (easiest)
docker-compose up -d

# Wait for services to start (about 30 seconds)
docker-compose logs -f api
```

### 2. Run Database Migrations
```bash
# Run migrations
docker-compose exec api ./taskmaster migrate up

# Create admin user
docker-compose exec api ./taskmaster user create \
  --email admin@taskmaster.dev \
  --password admin123 \
  --role admin \
  --first-name Admin \
  --last-name User
```

### 3. Test the API
```bash
# Health check
curl http://localhost:8080/health

# API documentation
open http://localhost:8080/docs
```

## 📖 API Usage Examples

### Authentication

#### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@taskmaster.dev",
    "password": "admin123"
  }'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a1b2c3d4e5f6...",
  "expires_in": 86400,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "admin@taskmaster.dev",
    "role": "admin"
  }
}
```

#### Set Token for Subsequent Requests
```bash
export TOKEN="your_access_token_here"
```

### User Management

#### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "username": "johndoe",
    "password": "securepass123",
    "first_name": "John",
    "last_name": "Doe",
    "role": "developer",
    "working_hours_start": "09:00:00",
    "working_hours_end": "17:00:00",
    "working_days": [1, 2, 3, 4, 5],
    "timezone": "UTC",
    "hourly_rate": 75.00
  }'
```

#### Get Current User
```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

#### List Users
```bash
curl -X GET "http://localhost:8080/api/v1/users?limit=10&offset=0&role=developer" \
  -H "Authorization: Bearer $TOKEN"
```

### Project Management

#### Create a Project
```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Website Redesign",
    "project_code": "WEB-2024",
    "description": "Complete redesign of company website",
    "priority": "high",
    "start_date": "2024-02-01T00:00:00Z",
    "end_date": "2024-04-30T00:00:00Z",
    "budget": 50000.00,
    "currency_code": "USD",
    "client_name": "Internal"
  }'
```

#### Get Project Details
```bash
curl -X GET http://localhost:8080/api/v1/projects/1 \
  -H "Authorization: Bearer $TOKEN"
```

#### List Projects
```bash
curl -X GET "http://localhost:8080/api/v1/projects?status=active&limit=20" \
  -H "Authorization: Bearer $TOKEN"
```

#### Add Team Member to Project
```bash
curl -X POST http://localhost:8080/api/v1/projects/1/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440001",
    "role": "developer",
    "allocation_percentage": 80.0
  }'
```

### Task Management

#### Create a Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": 1,
    "title": "Design homepage mockup",
    "description": "Create wireframes and mockups for the new homepage design",
    "priority": "high",
    "estimated_hours": 16.0,
    "due_date": "2024-02-15T17:00:00Z",
    "tags": ["design", "frontend", "urgent"]
  }'
```

#### Assign Task to User
```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/assign \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "assignee_id": "550e8400-e29b-41d4-a716-446655440001"
  }'
```

#### Start Working on Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/start \
  -H "Authorization: Bearer $TOKEN"
```

#### Complete Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks/1/complete \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_hours": 14.5
  }'
```

#### List Tasks
```bash
curl -X GET "http://localhost:8080/api/v1/tasks?project_id=1&status=in_progress" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Upcoming Deadlines
```bash
curl -X GET "http://localhost:8080/api/v1/tasks/deadlines?days=7" \
  -H "Authorization: Bearer $TOKEN"
```

### Time Tracking

#### Start Time Tracking
```bash
curl -X POST http://localhost:8080/api/v1/time-entries/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 1,
    "project_id": 1,
    "description": "Working on homepage design"
  }'
```

#### Stop Time Tracking
```bash
curl -X POST http://localhost:8080/api/v1/time-entries/stop \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Active Time Entry
```bash
curl -X GET http://localhost:8080/api/v1/time-entries/active \
  -H "Authorization: Bearer $TOKEN"
```

#### Create Manual Time Entry
```bash
curl -X POST http://localhost:8080/api/v1/time-entries \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": 1,
    "project_id": 1,
    "start_time": "2024-02-01T09:00:00Z",
    "end_time": "2024-02-01T12:00:00Z",
    "description": "Initial research and planning",
    "billable": true
  }'
```

#### Get Time Report
```bash
curl -X GET "http://localhost:8080/api/v1/time-entries/report?start_date=2024-02-01&end_date=2024-02-28" \
  -H "Authorization: Bearer $TOKEN"
```

#### List Time Entries
```bash
curl -X GET "http://localhost:8080/api/v1/time-entries?project_id=1&billable=true&limit=50" \
  -H "Authorization: Bearer $TOKEN"
```

## 🔧 Development Setup

### Manual Installation

#### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Redis 6+

#### Setup Steps
```bash
# Clone repository
git clone https://github.com/yourusername/taskmaster-core.git
cd taskmaster-core

# Install dependencies
go mod download

# Copy environment file
cp .env.example .env
# Edit .env with your database credentials

# Install development tools
make dev-setup

# Start database and Redis manually
# Configure your local PostgreSQL and Redis instances

# Run migrations
make migrate-up

# Create admin user
make create-admin

# Start the server
make run
```

### Testing

#### Run All Tests
```bash
# With Docker
make test

# Manual setup
go test ./...
```

#### Run with Coverage
```bash
make test-coverage
open coverage.html
```

### Linting
```bash
make lint
```

## 📊 Monitoring and Health Checks

### Health Endpoints
```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health with database status
curl http://localhost:8080/health/detailed

# Kubernetes readiness probe
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:8080/metrics
```

## 🐳 Docker Commands

### Basic Operations
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down

# Rebuild and start
docker-compose up --build -d

# Run migrations
docker-compose exec api ./taskmaster migrate up

# Access database
docker-compose exec db psql -U postgres -d taskmaster
```

### Production Deployment
```bash
# Production build
docker build -t taskmaster-core:latest .

# Production compose
docker-compose -f docker-compose.prod.yml up -d
```

## ☸️ Kubernetes Deployment

### Deploy to Kubernetes
```bash
# Apply all manifests
kubectl apply -f deployments/kubernetes/

# Check status
kubectl get pods -n taskmaster

# View logs
kubectl logs -f deployment/taskmaster-api -n taskmaster

# Port forward for testing
kubectl port-forward service/taskmaster-api-service 8080:80 -n taskmaster
```

### Delete from Kubernetes
```bash
kubectl delete -f deployments/kubernetes/
```

## 🔑 API Authentication Flow

### 1. Login and Get Tokens
```javascript
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password'
  })
});

const { access_token, refresh_token } = await response.json();
```

### 2. Use Access Token
```javascript
const response = await fetch('/api/v1/projects', {
  headers: {
    'Authorization': `Bearer ${access_token}`,
    'Content-Type': 'application/json'
  }
});
```

### 3. Refresh Token When Expired
```javascript
const response = await fetch('/api/v1/auth/refresh', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    refresh_token: refresh_token
  })
});

const { access_token: newToken } = await response.json();
```

## 🎯 Common Use Cases

### Scenario 1: New Project Setup
1. Create project
2. Add team members
3. Create tasks
4. Assign tasks to team members

### Scenario 2: Daily Time Tracking
1. Start time tracking when beginning work
2. Switch between tasks as needed
3. Stop time tracking at end of day
4. Review time entries

### Scenario 3: Project Reporting
1. Get project statistics
2. Generate time reports for date range
3. Check upcoming deadlines
4. Review team workload

## 🛠 Troubleshooting

### Common Issues

#### Database Connection Failed
```bash
# Check if PostgreSQL is running
docker-compose ps

# Check logs
docker-compose logs db

# Restart database
docker-compose restart db
```

#### Migration Errors
```bash
# Check migration status
docker-compose exec api ./taskmaster migrate version

# Reset migrations (⚠️ destructive)
docker-compose exec api ./taskmaster migrate down
docker-compose exec api ./taskmaster migrate up
```

#### Authentication Issues
```bash
# Verify JWT secret is set
echo $JWT_SECRET

# Check user exists
docker-compose exec db psql -U postgres -d taskmaster -c "SELECT * FROM users;"
```

#### Permission Denied
- Ensure user has correct role for the operation
- Check API endpoint requires specific permissions
- Verify JWT token is valid and not expired

### Getting Help

1. Check the [API Documentation](http://localhost:8080/docs)
2. Review server logs: `docker-compose logs -f api`
3. Check database logs: `docker-compose logs -f db`
4. Verify environment configuration in `.env`
5. Open an issue on GitHub with logs and steps to reproduce

## 📈 Performance Tips

1. **Use appropriate pagination** for large datasets
2. **Filter requests** to reduce data transfer
3. **Cache frequently accessed data** on the client side
4. **Use batch operations** when possible
5. **Monitor API response times** with `/metrics` endpoint

---

**🎉 You're ready to start building with TaskMaster Core!**

For more detailed information, check the full API documentation at `/docs` when running the server.