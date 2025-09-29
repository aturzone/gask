# Task Management API - Complete Guide

A comprehensive, production-ready task management system with Redis caching, PostgreSQL persistence, and role-based access control.

## 📋 Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [API Reference](#api-reference)
  - [User Management](#user-management-endpoints)
  - [Group Management](#group-management-endpoints)
  - [Task Management](#task-management-endpoints)
  - [Admin Operations](#admin-operations-endpoints)
- [Role-Based Access Control](#role-based-access-control)
- [Data Synchronization](#data-synchronization)
- [Monitoring & Health](#monitoring--health)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)

---

## 🚀 Features

### Core Features
- **High Performance**: Redis-first architecture with background PostgreSQL sync
- **Multi-User Support**: Three-tier role system (Owner, Group Admins, Regular Users)
- **Group Management**: Organize users and tasks by departments/teams
- **Full CRUD Operations**: Complete task lifecycle management
- **Role-Based Access Control**: Granular permissions at every level
- **Work Time Tracking**: Flexible per-user weekly schedules
- **Auto-Sync**: Background synchronization every 15 minutes
- **Search & Filtering**: Advanced search across users and tasks
- **Batch Operations**: Update multiple tasks simultaneously
- **Health Monitoring**: Built-in health checks and system status

### Advanced Features
- **Concurrent Operations**: Thread-safe Redis operations
- **Data Consistency**: Automatic conflict resolution
- **Emergency Recovery**: Restore from PostgreSQL backup
- **Real-time Statistics**: Comprehensive analytics and reporting
- **Flexible Authentication**: Owner password header + Basic Auth

---

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │───▶│   Go API        │───▶│     Redis       │
│   / CLI Tool    │    │   Port: 7890    │    │   (Primary)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  Sync Service   │───▶│  PostgreSQL     │
                       │  (15min cycle)  │    │ (Persistent)    │
                       └─────────────────┘    └─────────────────┘
```

**Data Flow:**
1. All API operations hit Redis (fast, in-memory)
2. Changes are marked as "dirty"
3. Sync service runs every 15 minutes
4. Dirty data is synchronized to PostgreSQL
5. PostgreSQL serves as the source of truth for recovery

---

## 📦 Prerequisites

- **Go**: 1.21 or higher
- **Redis**: 7.2+ running on `localhost:6380`
- **PostgreSQL**: 15+ running on `localhost:5433`

---

## 🎯 Quick Start

### 1. Install Dependencies

```bash
go mod download
```

### 2. Set Environment Variables (Optional)

```bash
export OWNER_PASSWORD="your_secure_password"
export OWNER_EMAIL="admin@yourcompany.com"
export POSTGRES_PASSWORD="your_postgres_password"
```

**Defaults:**
- `OWNER_PASSWORD`: `admin1234`
- `OWNER_EMAIL`: `admin@gmail.com`
- `POSTGRES_PASSWORD`: `EKQH9jQX7gAfV7pLwVmsbLbF3XfY6n4S`

### 3. Ensure Services Are Running

```bash
# Redis on port 6380
redis-server --port 6380

# PostgreSQL on port 5433
# Or use Docker containers
```

### 4. Run the Application

```bash
go run main.go
```

The server will start on **`http://localhost:7890`**

### 5. Verify Health

```bash
curl http://localhost:7890/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T10:00:00Z"
}
```

---

## 🔐 Authentication

### Three Authentication Methods

#### 1. Owner Access (Full Control)
Use the `X-Owner-Password` header for system-wide operations:

```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/users
```

**Owner Capabilities:**
- Create/delete groups
- Create group admins
- Access all data across all groups
- Use admin endpoints
- Override any permission

---

#### 2. Basic Authentication - User ID
Regular users and group admins use Basic Auth with their user ID:

```bash
curl -u "USER_ID:password" http://localhost:7890/users/USER_ID
```

Example:
```bash
curl -u "3:mypassword" http://localhost:7890/users/3
```

---

#### 3. Basic Authentication - Email
Alternatively, authenticate using email:

```bash
curl -u "user@email.com:password" http://localhost:7890/users/1
```

Example:
```bash
curl -u "alice@company.com:alice123" http://localhost:7890/users/3
```

---

## 📚 API Reference

### Base URL
```
http://localhost:7890
```

---

## 👥 User Management Endpoints

### List All Users
**Endpoint:** `GET /users`

**Authentication:** Owner, Group Admins (see their groups only)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/users
```

**Response:**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "full_name": "System Owner",
        "role": "owner",
        "group_ids": [],
        "email": "admin@gmail.com",
        "work_times": {}
      }
    ],
    "count": 1
  }
}
```

---

### Create User
**Endpoint:** `POST /users`

**Authentication:** Owner (any role), Group Admin (only 'user' role in their groups)

**Request Body:**
```json
{
  "full_name": "John Doe",
  "email": "john@company.com",
  "password": "secure123",
  "role": "user",
  "group_ids": [1],
  "number": "+1234567890",
  "work_times": {
    "Monday": 8.0,
    "Tuesday": 8.0,
    "Wednesday": 8.0,
    "Thursday": 8.0,
    "Friday": 6.0
  }
}
```

**Valid Roles:**
- `user` - Regular user (default)
- `group_admin` - Group administrator
- `owner` - System owner (only owner can create)

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Alice Admin",
    "email": "alice@company.com",
    "password": "alice123",
    "role": "group_admin",
    "group_ids": [1],
    "work_times": {
      "Monday": 8.0,
      "Tuesday": 8.0,
      "Wednesday": 8.0,
      "Thursday": 8.0,
      "Friday": 8.0
    }
  }' \
  http://localhost:7890/users
```

---

### Get User Details
**Endpoint:** `GET /users/{id}`

**Authentication:** Owner, Group Admin (in same groups), User (own profile only)

**Example:**
```bash
curl -u "3:mypassword" http://localhost:7890/users/3
```

---

### Update User
**Endpoint:** `PUT /users/{id}`

**Authentication:** Owner (any field), Group Admin (limited), User (own profile, limited fields)

**Request Body (partial update):**
```json
{
  "full_name": "John Updated",
  "number": "+9876543210",
  "work_times": {
    "Monday": 9.0,
    "Tuesday": 9.0
  }
}
```

**Example:**
```bash
curl -X PUT \
  -u "3:mypassword" \
  -H "Content-Type: application/json" \
  -d '{"full_name": "John Updated"}' \
  http://localhost:7890/users/3
```

---

### Delete User
**Endpoint:** `DELETE /users/{id}`

**Authentication:** Owner only

**Example:**
```bash
curl -X DELETE \
  -H "X-Owner-Password: admin1234" \
  http://localhost:7890/users/5
```

**Note:** Deletes all user's tasks automatically.

---

### Search Users
**Endpoint:** `GET /users/search?q={query}`

**Authentication:** Owner, Group Admins (see their groups)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/users/search?q=john"
```

---

### Get User's Work Times
**Endpoint:** `GET /users/{id}/worktimes`

**Authentication:** Owner, Group Admin (same groups), User (own times)

**Example:**
```bash
curl -u "3:mypassword" http://localhost:7890/users/3/worktimes
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 3,
    "work_times": {
      "Monday": 8.0,
      "Tuesday": 8.0,
      "Wednesday": 8.0,
      "Thursday": 8.0,
      "Friday": 6.0
    }
  }
}
```

---

### Update User's Work Times
**Endpoint:** `PUT /users/{id}/worktimes`

**Authentication:** Owner, Group Admin (same groups), User (own times)

**Request Body:**
```json
{
  "work_times": {
    "Monday": 9.0,
    "Tuesday": 9.0,
    "Wednesday": 8.0,
    "Thursday": 8.0,
    "Friday": 6.0,
    "Saturday": 4.0,
    "Sunday": 0.0
  }
}
```

**Example:**
```bash
curl -X PUT \
  -u "3:mypassword" \
  -H "Content-Type: application/json" \
  -d '{
    "work_times": {
      "Monday": 9.0,
      "Tuesday": 9.0,
      "Wednesday": 8.0,
      "Thursday": 8.0,
      "Friday": 6.0
    }
  }' \
  http://localhost:7890/users/3/worktimes
```

---

## 👔 Group Management Endpoints

### List All Groups
**Endpoint:** `GET /groups`

**Authentication:** Owner (all groups), Group Admin (their groups), User (their groups, read-only)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/groups
```

**Response:**
```json
{
  "success": true,
  "data": {
    "groups": [
      {
        "id": 1,
        "name": "Engineering",
        "admin_id": 2,
        "created_at": "2025-09-29T10:00:00Z",
        "updated_at": "2025-09-29T10:00:00Z"
      }
    ],
    "count": 1
  }
}
```

---

### Create Group
**Endpoint:** `POST /groups`

**Authentication:** Owner only

**Request Body:**
```json
{
  "name": "Engineering",
  "admin_id": 2
}
```

**Requirements:**
- Admin must have role `group_admin` or `owner`
- Group name must be unique

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Engineering",
    "admin_id": 2
  }' \
  http://localhost:7890/groups
```

---

### Get Group Details
**Endpoint:** `GET /groups/{id}`

**Authentication:** Owner, Group Admin (their groups), User (their groups)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/groups/1
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "Engineering",
    "admin_id": 2,
    "admin": {
      "id": 2,
      "full_name": "Alice Admin",
      "email": "alice@company.com"
    },
    "users_count": 3,
    "tasks_count": 12,
    "created_at": "2025-09-29T10:00:00Z",
    "updated_at": "2025-09-29T10:00:00Z"
  }
}
```

---

### Update Group
**Endpoint:** `PUT /groups/{id}`

**Authentication:** Owner only

**Request Body (partial update):**
```json
{
  "name": "Engineering Team",
  "admin_id": 3
}
```

**Example:**
```bash
curl -X PUT \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{"name": "Engineering Team"}' \
  http://localhost:7890/groups/1
```

---

### Delete Group
**Endpoint:** `DELETE /groups/{id}`

**Authentication:** Owner only

**Example:**
```bash
curl -X DELETE \
  -H "X-Owner-Password: admin1234" \
  http://localhost:7890/groups/1
```

**Note:** Automatically removes group from all users and deletes all group tasks.

---

### List Group Users
**Endpoint:** `GET /groups/{id}/users`

**Authentication:** Owner, Group Admin (their groups), User (their groups)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/groups/1/users
```

---

### Add User to Group
**Endpoint:** `POST /groups/{id}/users`

**Authentication:** Owner, Group Admin (their groups)

**Request Body:**
```json
{
  "user_id": 3
}
```

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 3}' \
  http://localhost:7890/groups/1/users
```

---

### Remove User from Group
**Endpoint:** `DELETE /groups/{id}/users/{user_id}`

**Authentication:** Owner, Group Admin (their groups)

**Example:**
```bash
curl -X DELETE \
  -H "X-Owner-Password: admin1234" \
  http://localhost:7890/groups/1/users/3
```

**Note:** User's tasks in this group are moved to another group or deleted if no other group exists.

---

### Get Group Tasks
**Endpoint:** `GET /groups/{id}/tasks`

**Authentication:** Owner, Group Admin (their groups), User (their groups, own tasks only)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/groups/1/tasks
```

---

### Get Group Statistics
**Endpoint:** `GET /groups/{id}/stats`

**Authentication:** Owner, Group Admin (their groups), User (their groups)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/groups/1/stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "group": {
      "id": 1,
      "name": "Engineering"
    },
    "users_count": 3,
    "total_tasks": 12,
    "completed_tasks": 5,
    "pending_tasks": 7,
    "completion_rate": 41.67,
    "user_task_counts": {
      "Alice Admin": 4,
      "Charlie Developer": 5,
      "Diana Coder": 3
    }
  }
}
```

---

## 📋 Task Management Endpoints

### Get User's Tasks
**Endpoint:** `GET /users/{id}/tasks`

**Authentication:** Owner, Group Admin (same groups), User (own tasks only)

**Example:**
```bash
curl -u "3:mypassword" http://localhost:7890/users/3/tasks
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 3,
    "tasks": [
      {
        "id": 1,
        "title": "Implement Login API",
        "status": false,
        "priority": 1,
        "deadline": "2025-10-15",
        "information": "Create REST API for authentication",
        "user_id": 3,
        "group_id": 1,
        "created_at": "2025-09-29T10:00:00Z",
        "updated_at": "2025-09-29T10:00:00Z"
      }
    ],
    "count": 1
  }
}
```

---

### Create Task
**Endpoint:** `POST /users/{id}/tasks`

**Authentication:** Owner, Group Admin (for users in their groups), User (own tasks)

**Request Body:**
```json
{
  "title": "Implement authentication",
  "priority": 1,
  "deadline": "2025-12-31",
  "information": "Add JWT-based authentication",
  "group_id": 1
}
```

**Priority Levels:**
- `1` - High priority
- `2` - Medium priority
- `3` - Low priority

**Example:**
```bash
curl -X POST \
  -u "3:mypassword" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Implement Login API",
    "priority": 1,
    "deadline": "2025-10-15",
    "information": "Create REST API for authentication",
    "group_id": 1
  }' \
  http://localhost:7890/users/3/tasks
```

---

### Get Specific Task
**Endpoint:** `GET /users/{id}/tasks/{task_id}`

**Authentication:** Owner, Group Admin (same group), User (own task)

**Example:**
```bash
curl -u "3:mypassword" http://localhost:7890/users/3/tasks/1
```

---

### Update Task
**Endpoint:** `PUT /users/{id}/tasks/{task_id}`

**Authentication:** Owner, Group Admin (same group), User (own task)

**Request Body (partial update):**
```json
{
  "title": "Updated Title",
  "priority": 2,
  "deadline": "2025-11-01",
  "information": "Updated information",
  "status": true,
  "group_id": 2
}
```

**Example:**
```bash
curl -X PUT \
  -u "3:mypassword" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": 2,
    "information": "Updated: Critical bug"
  }' \
  http://localhost:7890/users/3/tasks/1
```

---

### Mark Task as Done
**Endpoint:** `PUT /users/{id}/tasks/{task_id}/done`

**Authentication:** Owner, Group Admin (same group), User (own task)

**Example:**
```bash
curl -X PUT \
  -u "3:mypassword" \
  http://localhost:7890/users/3/tasks/1/done
```

---

### Delete Task
**Endpoint:** `DELETE /users/{id}/tasks/{task_id}`

**Authentication:** Owner, Group Admin (same group), User (own task)

**Example:**
```bash
curl -X DELETE \
  -u "3:mypassword" \
  http://localhost:7890/users/3/tasks/1
```

---

### Global Task Search
**Endpoint:** `GET /tasks/search?q={query}`

**Authentication:** Owner (all tasks), Group Admin (their groups), User (cannot use)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/tasks/search?q=authentication"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "query": "authentication",
    "results": [
      {
        "user_id": 3,
        "task": {
          "id": 1,
          "title": "Implement Login API",
          "status": false,
          "priority": 1,
          "deadline": "2025-10-15",
          "information": "Create REST API for authentication",
          "user_id": 3,
          "group_id": 1
        }
      }
    ],
    "count": 1
  }
}
```

---

### Get Task Statistics
**Endpoint:** `GET /tasks/stats`

**Authentication:** Owner (global), Group Admin (their groups), User (own stats)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/tasks/stats
```

**Response (Owner view):**
```json
{
  "success": true,
  "data": {
    "total_tasks": 25,
    "completed_tasks": 10,
    "pending_tasks": 15,
    "completion_rate": 40.0,
    "user_task_counts": {
      "Alice Admin": 8,
      "Charlie Developer": 10,
      "Diana Coder": 7
    },
    "group_task_counts": {
      "1": 15,
      "2": 10
    },
    "total_users": 5
  }
}
```

---

### Batch Update Tasks
**Endpoint:** `POST /tasks/batch`

**Authentication:** Owner (any tasks), Group Admin (their groups), User (own tasks)

**Request Body:**
```json
{
  "task_ids": [1, 2, 3],
  "updates": {
    "priority": 2,
    "deadline": "2025-11-01"
  },
  "action": "update"
}
```

**Actions:**
- `update` - Update specified fields
- `mark_done` - Mark all as completed
- `delete` - Delete all tasks

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "task_ids": [1, 2, 3],
    "updates": {"priority": 2},
    "action": "update"
  }' \
  http://localhost:7890/tasks/batch
```

---

### Advanced Task Filtering
**Endpoint:** `GET /tasks/filter`

**Authentication:** Owner (all tasks), Group Admin (their groups), User (own tasks)

**Query Parameters:**
- `status` - `completed`, `pending`, `all`
- `priority` - `1`, `2`, `3`
- `group_id` - Filter by group
- `user_id` - Filter by user (if permitted)

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/tasks/filter?status=pending&priority=1&group_id=1"
```

---

## 🔧 Admin Operations Endpoints

### Force Sync to PostgreSQL
**Endpoint:** `POST /admin/sync?action=force`

**Authentication:** Owner only

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=force"
```

---

### Restore from PostgreSQL
**Endpoint:** `POST /admin/sync?action=restore`

**Authentication:** Owner only

**Use Case:** Emergency recovery when Redis data is corrupted

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=restore"
```

---

### Emergency Backup
**Endpoint:** `POST /admin/sync?action=backup`

**Authentication:** Owner only

**Example:**
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=backup"
```

---

### Get System Status
**Endpoint:** `GET /admin/status`

**Authentication:** Owner only

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/admin/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "running": true,
    "interval": "15m0s",
    "last_sync": "2025-09-29T10:00:00Z",
    "time_since_last_sync": "5m30s",
    "healthy": true,
    "dirty_types": ["tasks"],
    "pending_changes": true,
    "redis_status": "connected",
    "postgres_status": "connected"
  }
}
```

---

### Get System Statistics
**Endpoint:** `GET /admin/stats`

**Authentication:** Owner only

**Example:**
```bash
curl -H "X-Owner-Password: admin1234" http://localhost:7890/admin/stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "postgresql": {
      "users": 10,
      "groups": 3,
      "tasks": 45
    },
    "redis": {
      "users": 10,
      "groups": 3,
      "tasks": 47
    },
    "sync": {
      "running": true,
      "healthy": true,
      "pending_changes": true
    }
  }
}
```

---

### Health Check
**Endpoint:** `GET /health`

**Authentication:** None (public)

**Example:**
```bash
curl http://localhost:7890/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-09-29T10:00:00Z"
}
```

---

## 🔐 Role-Based Access Control

### User Roles

| Role | Can Do | Cannot Do |
|------|--------|-----------|
| **Owner** | Everything: Create groups, create admins, access all data, admin endpoints | N/A |
| **Group Admin** | Manage users in their groups, create regular users, view group data | Create groups, create other admins, access other groups |
| **User** | Manage own tasks, view own profile, view group info (read-only) | Access other users' data, create users, manage groups |

### Permission Matrix

| Endpoint | Owner | Group Admin | User |
|----------|-------|-------------|------|
| GET /users | ✅ All | ✅ Their groups | ❌ |
| POST /users | ✅ Any role | ✅ Only 'user' in their groups | ❌ |
| PUT /users/{id} | ✅ Any field | ✅ Limited fields, their groups | ✅ Own profile, limited |
| DELETE /users/{id} | ✅ | ❌ | ❌ |
| POST /groups | ✅ | ❌ | ❌ |
| PUT /groups/{id} | ✅ | ❌ | ❌ |
| DELETE /groups/{id} | ✅ | ❌ | ❌ |
| GET /groups/{id}/tasks | ✅ All tasks | ✅ Group tasks | ✅ Own tasks only |
| POST /users/{id}/tasks | ✅ | ✅ For their group users | ✅ Own tasks |
| GET /tasks/search | ✅ | ✅ Their groups | ❌ |
| GET /admin/* | ✅ | ❌ | ❌ |

---

## 🔄 Data Synchronization

### Automatic Sync (Every 15 Minutes)

The sync service automatically:
1. Checks for dirty data types in Redis
2. Syncs only changed data to PostgreSQL
3. Updates sync counters
4. Clears dirty flags
5. Records last sync time

### Manual Sync Operations

#### Force Sync
```bash
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=force"
```

#### Restore from PostgreSQL
```bash
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=restore"
```

#### Emergency Backup
```bash
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=backup"
```

### Sync Status Check

```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/status"
```

---

## 🏥 Monitoring & Health

### Health Checks

**Basic Health:**
```bash
curl http://localhost:7890/health
```

**Detailed Status (Owner):**
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/status"
```

**System Statistics (Owner):**
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/stats"
```

### Monitoring Checklist

- ✅ `/health` returns 200
- ✅ Redis status: connected
- ✅ PostgreSQL status: connected
- ✅ Sync healthy: true
- ✅ Time since last sync < 30 minutes

---

## 🚀 Production Deployment

### Environment Variables

Create `.env` file:
```bash
OWNER_PASSWORD="your_strong_password_here"
OWNER_EMAIL="admin@yourcompany.com"
POSTGRES_PASSWORD="your_postgres_password"
SERVER_PORT=7890
REDIS_HOST=localhost
REDIS_PORT=6380
DB_HOST=localhost
DB_PORT=5433
```

### Security Hardening

1. **Use Strong Passwords**
   ```bash
   export OWNER_PASSWORD=$(openssl rand -base64 32)
   ```

2. **Enable HTTPS** - Use nginx/Caddy as reverse proxy

3. **Redis Security**
   ```bash
   # redis.conf
   requirepass your_redis_password
   bind 127.0.0.1
   ```

4. **PostgreSQL Security**
   - Enable SSL connections
   - Use strong passwords
   - Restrict network access

### Performance Tuning

1. **Redis Memory**
   ```bash
   # redis.conf
   maxmemory 2gb
   maxmemory-policy allkeys-lru
   ```

2. **PostgreSQL**
   ```sql
   -- postgresql.conf
   shared_buffers = 256MB
   effective_cache_size = 1GB
   work_mem = 16MB
   ```

3. **Go Application**
   ```bash
   export GOMAXPROCS=4
   ```

### Docker Deployment

```yaml
version: '3.8'
services:
  api:
    build: .
    ports:
      - "7890:7890"
    environment:
      - OWNER_PASSWORD=${OWNER_PASSWORD}
      - REDIS_HOST=redis
      - DB_HOST=postgres
    depends_on:
      - redis
      - postgres
  
  redis:
    image: redis:7.2-alpine
    ports:
      - "6380:6379"
  
  postgres:
    image: postgres:15-alpine
    ports:
      - "5433:5432"
    environment:
      - POSTGRES_DB=taskmaster
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
```

---

## 🐛 Troubleshooting

### Common Issues

#### 1. Redis Connection Failed

**Symptoms:** API returns 503, logs show Redis connection errors

**Solutions:**
```bash
# Check if Redis is running
redis-cli -p 6380 ping

# Check Redis logs
tail -f /var/log/redis/redis.log

# Restart Redis
redis-server --port 6380
```

---

#### 2. PostgreSQL Connection Failed

**Symptoms:** Sync fails, `/admin/status` shows postgres_status: "error"

**Solutions:**
```bash
# Check PostgreSQL status
pg_isready -h localhost -p 5433

# Test connection
psql -h localhost -p 5433 -U airflow -d airflow

# Check logs
tail -f /var/log/postgresql/postgresql-15-main.log
```

---

#### 3. Sync Service Not Running

**Symptoms:** `time_since_last_sync` > 30 minutes, pending_changes: true

**Solutions:**
```bash
# Check sync status
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/status

# Force manual sync
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=force"

# Restart application
```

---

#### 4. Permission Denied Errors

**Symptoms:** 403 Forbidden responses

**Solutions:**
1. Verify authentication credentials
2. Check user role and group membership
3. Ensure user belongs to required groups
4. Verify group admin assignments

```bash
# Check user details
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/users/{id}

# Check group membership
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/groups/{id}/users
```

---

#### 5. Data Inconsistency

**Symptoms:** Redis and PostgreSQL show different data

**Solutions:**
```bash
# Check system stats
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/stats

# Force sync if minor differences
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=force"

# Restore from PostgreSQL if Redis is corrupted
curl -X POST -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/admin/sync?action=restore"
```

---

### Logs

Application logs show:
- HTTP request/response details
- Authentication attempts
- Sync operations
- Error traces

**View logs:**
```bash
# If running with stdout
go run main.go | tee app.log

# Check specific operations
grep "Sync" app.log
grep "Error" app.log
```

---

## 📞 Support

For issues and questions:
1. Check this documentation
2. Review application logs
3. Check `/health` and `/admin/status` endpoints
4. Verify Redis and PostgreSQL connectivity

---

## 📄 License

MIT License - See LICENSE file for details.

---