# Task Management API

A comprehensive task management system with Redis caching and PostgreSQL persistence.

## Features

- 🚀 **High Performance**: Redis-first architecture with background PostgreSQL sync
- 👥 **Multi-User Support**: Owner, Group Admins, and Regular Users
- 🏢 **Group Management**: Users can belong to multiple groups
- 📋 **Task Management**: Full CRUD operations with group-based organization
- 🔒 **Role-Based Access Control**: Granular permissions system
- 🔄 **Auto-Sync**: Background synchronization every 15 minutes
- 📊 **Statistics & Reporting**: Comprehensive analytics
- 🏥 **Health Monitoring**: Built-in health checks and admin endpoints

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │───▶│   Go API        │───▶│     Redis       │
│                 │    │                 │    │   (Primary)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  Sync Service   │───▶│  PostgreSQL     │
                       │  (15min cycle)  │    │ (Persistent)    │
                       └─────────────────┘    └─────────────────┘
```

## Prerequisites

- Go 1.21+
- Redis 7.2+ (running on localhost:6380)
- PostgreSQL 15+ (running on localhost:5433)

## Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Set Environment Variables (Optional)

```bash
export OWNER_PASSWORD="your_secure_password"
export OWNER_EMAIL="admin@yourcompany.com"
```

### 3. Start the Services

Make sure Redis and PostgreSQL are running:

```bash
# Your existing Docker containers are already running:
# - Redis: localhost:6380
# - PostgreSQL: localhost:5433 (using airflow credentials)
```

### 4. Run the Application

```bash
go run main.go
```

The server will start on `http://localhost:7890`

## API Usage

### Authentication

#### Owner Access (Full Control)
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/users
```

#### User Access (Limited to Own Data)
```bash
curl -u "1:userpassword" http://localhost:7890/users/1
# OR
curl -u "user@email.com:password" http://localhost:7890/users/1
```

### Core Operations

#### 1. Create a Group
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Development Team",
    "admin_id": 1
  }' \
  http://localhost:7890/groups
```

#### 2. Create a User
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "John Doe",
    "email": "john@company.com",
    "password": "secure123",
    "role": "user",
    "group_ids": [1],
    "work_times": {
      "Monday": 8.0,
      "Tuesday": 8.0,
      "Wednesday": 8.0,
      "Thursday": 8.0,
      "Friday": 6.0
    }
  }' \
  http://localhost:8080/users
```

#### 3. Create a Task
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Implement authentication",
    "priority": 1,
    "deadline": "2024-12-31",
    "information": "Add JWT-based authentication to the API",
    "group_id": 1
  }' \
  http://localhost:8080/users/1/tasks
```

#### 4. Search Tasks
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:8080/tasks/search?q=authentication"
```

#### 5. Get Statistics
```bash
curl -H "X-Owner-Password: admin1234" \
  http://localhost:8080/tasks/stats
```

## Endpoints Summary

### User Management
- `GET /users` - List users
- `POST /users` - Create user
- `GET /users/{id}` - Get user
- `PUT /users/{id}` - Update user
- `DELETE /users/{id}` - Delete user
- `GET /users/search?q=query` - Search users

### Task Management
- `GET /users/{id}/tasks` - List user tasks
- `POST /users/{id}/tasks` - Create task
- `GET /users/{id}/tasks/{tid}` - Get task
- `PUT /users/{id}/tasks/{tid}` - Update task
- `DELETE /users/{id}/tasks/{tid}` - Delete task
- `PUT /users/{id}/tasks/{tid}/done` - Mark task done

### Group Management
- `GET /groups` - List groups
- `POST /groups` - Create group
- `GET /groups/{id}` - Get group
- `PUT /groups/{id}` - Update group
- `DELETE /groups/{id}` - Delete group
- `GET /groups/{id}/users` - List group users
- `POST /groups/{id}/users` - Add user to group
- `DELETE /groups/{id}/users/{uid}` - Remove user from group

### Global Operations
- `GET /tasks/search?q=query` - Search all tasks
- `GET /tasks/stats` - Task statistics
- `POST /tasks/batch` - Batch operations
- `GET /tasks/filter` - Advanced filtering

### Admin Operations
- `POST /admin/sync?action=force` - Force sync
- `GET /admin/status` - System status
- `GET /admin/stats` - System statistics
- `GET /health` - Health check

## User Roles & Permissions

### Owner
- Full access to all data and operations
- Can create/modify groups
- Can create group admins
- Can access admin endpoints

### Group Admin
- Full access to users and tasks in administered groups
- Can create regular users in their groups
- Cannot create other admins or groups

### User
- Access only to own data
- Can view group information (read-only)
- Can manage own tasks within assigned groups

## Data Synchronization

### Background Sync (Every 15 minutes)
- Automatically syncs changes from Redis to PostgreSQL
- Handles batch operations efficiently
- Maintains data consistency

### Manual Sync Operations
```bash
# Force immediate sync
curl -X POST -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/sync?action=force

# Restore from PostgreSQL (disaster recovery)
curl -X POST -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/sync?action=restore

# Emergency backup
curl -X POST -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/sync?action=backup
```

## Monitoring & Health

### Health Check
```bash
curl http://localhost:8080/health
```

### System Status (Owner only)
```bash
curl -H "X-Owner-Password: admin1234" \
  http://localhost:8080/admin/status
```

### System Statistics (Owner only)
```bash
curl -H "X-Owner-Password: admin1234" \
  http://localhost:8080/admin/stats
```

## Development

### Project Structure
```
task-manager/
├── main.go              # Application entry point
├── models/
│   └── models.go        # Data structures
├── modules/
│   ├── redis.go         # Redis operations
│   ├── postgres.go      # PostgreSQL operations
│   ├── sync.go          # Background sync service
│   └── auth.go          # Authentication & authorization
├── handlers/
│   ├── users.go         # User-related endpoints
│   ├── tasks.go         # Task-related endpoints
│   └── groups.go        # Group-related endpoints
└── utils/
    └── context.go       # Context utilities
```

### Adding New Features

1. Update models in `models/models.go`
2. Add Redis operations in `modules/redis.go`
3. Add PostgreSQL operations in `modules/postgres.go`
4. Create handlers in appropriate handler file
5. Update sync service in `modules/sync.go` if needed
6. Add routes in `main.go`

## Production Considerations

### Security
- Use strong passwords for owner account
- Enable HTTPS in production
- Consider JWT tokens for better security
- Implement rate limiting
- Use Redis AUTH and PostgreSQL SSL

### Performance
- Monitor Redis memory usage
- Tune PostgreSQL for your workload
- Consider Redis Cluster for high availability
- Implement connection pooling

### Monitoring
- Set up logging aggregation
- Monitor sync service health
- Track API response times
- Monitor database connections

### Backup
- Regular PostgreSQL backups
- Redis persistence configuration
- Consider cross-region replication

## Troubleshooting

### Common Issues

#### 1. Redis Connection Failed
```bash
# Check Redis status
docker ps | grep redis

# Check Redis logs
docker logs airflow-production_redis_1
```

#### 2. PostgreSQL Connection Failed
```bash
# Check PostgreSQL status
docker ps | grep postgres

# Test connection
psql -h localhost -p 5433 -U airflow -d airflow
```

#### 3. Sync Service Issues
```bash
# Check sync status
curl -H "X-Owner-Password: admin1234" \
  http://localhost:8080/admin/status

# Force sync
curl -X POST -H "X-Owner-Password: admin1234" \
  http://localhost:8080/admin/sync?action=force
```

#### 4. Permission Denied
- Verify authentication credentials
- Check user roles and group memberships
- Ensure user belongs to required groups

### Logs
Application logs show:
- Request/response times
- Sync operations
- Error details
- Authentication attempts

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.