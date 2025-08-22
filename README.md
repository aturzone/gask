# TaskMaster - Enterprise Multi-Project Task Management Core

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://docker.com)

## Overview

TaskMaster is a powerful, enterprise-grade task management system core built with Go, designed for multi-project environments. It provides comprehensive project management, resource allocation, task tracking, and team collaboration features through a robust REST API.

### ✨ Key Features

- **Multi-Project Management**: Handle multiple projects with resource allocation
- **Team Collaboration**: User management with skills, working hours, and capacity tracking
- **Advanced Task Management**: Hierarchical tasks with dependencies and time tracking
- **Financial Tracking**: Budget management and cost allocation per project
- **Real-time Notifications**: Deadline alerts and status updates
- **Comprehensive API**: RESTful endpoints with OpenAPI documentation
- **Enterprise Security**: JWT authentication with role-based access control
- **High Performance**: Redis caching and optimized database queries
- **Production Ready**: Docker containerization and Kubernetes deployment

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** - [Download Here](https://golang.org/dl/)
- **PostgreSQL 14+** - [Installation Guide](https://www.postgresql.org/download/)
- **Redis 6+** - [Installation Guide](https://redis.io/download)
- **Docker & Docker Compose** (Optional) - [Get Docker](https://docs.docker.com/get-docker/)

### Installation Options

#### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/taskmaster-core.git
cd taskmaster-core

# Start all services
docker-compose up -d

# Run database migrations
docker-compose exec api ./taskmaster migrate up

# Create admin user
docker-compose exec api ./taskmaster user create --email admin@example.com --password admin123 --role admin
```

API will be available at: `http://localhost:8080`

#### Option 2: Manual Setup

1. **Clone and Setup**
```bash
git clone https://github.com/yourusername/taskmaster-core.git
cd taskmaster-core
go mod download
```

2. **Database Setup**
```bash
# Create PostgreSQL database
createdb taskmaster

# Create .env file from template
cp .env.example .env
# Edit .env with your database credentials
```

3. **Run Migrations**
```bash
go build -o taskmaster cmd/api/main.go
./taskmaster migrate up
```

4. **Start Services**
```bash
# Start Redis (if not running)
redis-server

# Start the API
./taskmaster serve
```

## 📁 Project Structure

```
taskmaster-core/
├── cmd/
│   └── api/                    # Application entry point
│       ├── main.go
│       └── commands/           # CLI commands
├── internal/
│   ├── adapters/              # External interfaces
│   │   ├── http/              # HTTP handlers
│   │   ├── repository/        # Data access layer
│   │   └── cache/             # Caching implementation
│   ├── application/           # Application services
│   │   ├── commands/          # Write operations (CQRS)
│   │   ├── queries/           # Read operations (CQRS)
│   │   └── services/          # Business logic
│   ├── domain/                # Core business logic
│   │   ├── entities/          # Domain entities
│   │   ├── valueobjects/      # Value objects
│   │   └── services/          # Domain services
│   └── infrastructure/        # Technical concerns
│       ├── database/          # Database configuration
│       ├── auth/              # Authentication
│       └── config/            # Configuration
├── pkg/                       # Shared utilities
├── migrations/                # Database migrations
├── docs/                      # Documentation
├── deployments/               # Deployment configurations
│   ├── docker/
│   └── kubernetes/
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## 🔧 Configuration

All configuration is managed through environment variables. Create a `.env` file:

```bash
# Database Configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=taskmaster
DATABASE_USER=your_username
DATABASE_PASSWORD=your_password
DATABASE_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Application Configuration
APP_PORT=8080
APP_ENV=development
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRES_IN=24h

# Security
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Monitoring
LOG_LEVEL=info
ENABLE_METRICS=true
METRICS_PORT=9090
```

## 📊 Database Schema

### Core Tables

#### Users & Authentication
```sql
-- Users with working schedule
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role user_role DEFAULT 'developer',
    is_active BOOLEAN DEFAULT TRUE,
    working_hours_start TIME DEFAULT '09:00:00',
    working_hours_end TIME DEFAULT '17:00:00',
    working_days INTEGER[] DEFAULT '{1,2,3,4,5}', -- Monday to Friday
    timezone VARCHAR(50) DEFAULT 'UTC',
    hourly_rate DECIMAL(8,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- User skills and proficiency
CREATE TABLE user_skills (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    skill_name VARCHAR(100) NOT NULL,
    proficiency_level INTEGER CHECK (proficiency_level BETWEEN 1 AND 5),
    years_of_experience DECIMAL(3,1),
    is_certified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, skill_name)
);
```

#### Projects & Resources
```sql
-- Projects with financial tracking
CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    project_code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    status project_status DEFAULT 'planning',
    priority priority_level DEFAULT 'medium',
    start_date DATE,
    end_date DATE,
    budget DECIMAL(12,2),
    spent_budget DECIMAL(12,2) DEFAULT 0,
    currency_code VARCHAR(3) DEFAULT 'USD',
    owner_id INTEGER REFERENCES users(id),
    client_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    version INTEGER DEFAULT 1
);

-- Financial allocations and expenses
CREATE TABLE financial_allocations (
    id SERIAL PRIMARY KEY,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    category allocation_category NOT NULL,
    allocated_amount DECIMAL(12,2) NOT NULL,
    spent_amount DECIMAL(12,2) DEFAULT 0,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Tasks & Time Tracking
```sql
-- Comprehensive task management
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    parent_task_id INTEGER REFERENCES tasks(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status task_status DEFAULT 'todo',
    priority priority_level DEFAULT 'medium',
    assignee_id INTEGER REFERENCES users(id),
    reporter_id INTEGER REFERENCES users(id),
    estimated_hours DECIMAL(8,2),
    actual_hours DECIMAL(8,2) DEFAULT 0,
    start_date DATE,
    due_date DATE,
    completed_at TIMESTAMP,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    version INTEGER DEFAULT 1
);

-- Time tracking with detailed information
CREATE TABLE time_entries (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) NOT NULL,
    task_id INTEGER REFERENCES tasks(id),
    project_id INTEGER REFERENCES projects(id) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_minutes INTEGER,
    description TEXT,
    entry_date DATE NOT NULL,
    billable BOOLEAN DEFAULT TRUE,
    hourly_rate DECIMAL(8,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Enums and Types
```sql
CREATE TYPE user_role AS ENUM ('admin', 'project_manager', 'team_lead', 'developer', 'viewer');
CREATE TYPE project_status AS ENUM ('planning', 'active', 'on_hold', 'completed', 'cancelled');
CREATE TYPE task_status AS ENUM ('todo', 'in_progress', 'review', 'testing', 'completed', 'cancelled');
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE allocation_category AS ENUM ('personnel', 'equipment', 'software', 'infrastructure', 'other');
```

## 🔌 API Endpoints

### Authentication
```
POST   /api/v1/auth/login           # User login
POST   /api/v1/auth/register        # User registration
POST   /api/v1/auth/refresh         # Refresh token
POST   /api/v1/auth/logout          # User logout
```

### Projects
```
GET    /api/v1/projects             # List projects
POST   /api/v1/projects             # Create project
GET    /api/v1/projects/{id}        # Get project details
PUT    /api/v1/projects/{id}        # Update project
DELETE /api/v1/projects/{id}        # Delete project
GET    /api/v1/projects/{id}/tasks  # Get project tasks
GET    /api/v1/projects/{id}/team   # Get project team
POST   /api/v1/projects/{id}/team   # Add team member
```

### Tasks
```
GET    /api/v1/tasks               # List tasks (with filters)
POST   /api/v1/tasks               # Create task
GET    /api/v1/tasks/{id}          # Get task details
PUT    /api/v1/tasks/{id}          # Update task
DELETE /api/v1/tasks/{id}          # Delete task
POST   /api/v1/tasks/{id}/assign   # Assign task
POST   /api/v1/tasks/{id}/start    # Start working on task
POST   /api/v1/tasks/{id}/complete # Complete task
```

### Users & Team
```
GET    /api/v1/users               # List users
GET    /api/v1/users/me            # Get current user
PUT    /api/v1/users/me            # Update current user
GET    /api/v1/users/{id}          # Get user details
GET    /api/v1/users/{id}/workload # Get user workload
GET    /api/v1/users/{id}/tasks    # Get user tasks
```

### Time Tracking
```
GET    /api/v1/time-entries        # List time entries
POST   /api/v1/time-entries        # Create time entry
PUT    /api/v1/time-entries/{id}   # Update time entry
DELETE /api/v1/time-entries/{id}   # Delete time entry
POST   /api/v1/time-entries/start  # Start time tracking
POST   /api/v1/time-entries/stop   # Stop time tracking
```

### Reports & Analytics
```
GET    /api/v1/reports/dashboard   # Dashboard data
GET    /api/v1/reports/projects    # Project reports
GET    /api/v1/reports/team        # Team performance
GET    /api/v1/reports/time        # Time tracking reports
GET    /api/v1/reports/deadlines   # Upcoming deadlines
```

## 🧪 Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📈 Performance & Monitoring

### Health Checks
```
GET /health          # Basic health check
GET /health/detailed # Detailed system status
GET /ready          # Readiness probe
GET /metrics        # Prometheus metrics
```

### Monitoring Features
- **Prometheus Metrics**: Request duration, error rates, active connections
- **Structured Logging**: JSON formatted logs with correlation IDs
- **Database Monitoring**: Connection pool metrics and query performance
- **Redis Monitoring**: Cache hit rates and connection status

## 🚀 Deployment

### Docker Deployment
```bash
# Build image
docker build -t taskmaster-core .

# Run with environment variables
docker run -d \
  --name taskmaster \
  -p 8080:8080 \
  --env-file .env \
  taskmaster-core
```

### Kubernetes Deployment
```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/kubernetes/

# Check deployment status
kubectl get pods -l app=taskmaster-core

# View logs
kubectl logs -f deployment/taskmaster-core
```

### Environment-Specific Configurations

#### Development
```bash
export APP_ENV=development
export LOG_LEVEL=debug
export ENABLE_METRICS=true
```

#### Production
```bash
export APP_ENV=production
export LOG_LEVEL=info
export ENABLE_METRICS=true
export CORS_ALLOWED_ORIGINS=https://yourdomain.com
```

## 🔒 Security Features

- **JWT Authentication** with refresh tokens
- **Role-Based Access Control (RBAC)**
- **API Rate Limiting**
- **CORS Protection**
- **SQL Injection Prevention**
- **Password Hashing** with bcrypt
- **Security Headers** (HSTS, CSP, etc.)
- **Input Validation** with comprehensive sanitization

## 📝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup
```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/swaggo/swag/cmd/swag@latest

# Run linting
golangci-lint run

# Generate API documentation
swag init -g cmd/api/main.go
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🤝 Support

- **Documentation**: [docs/](./docs/)
- **API Reference**: Available at `/docs` endpoint when running
- **Issues**: [GitHub Issues](https://github.com/yourusername/taskmaster-core/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/taskmaster-core/discussions)

## 🎯 Roadmap

- [ ] **Real-time Notifications** with WebSockets
- [ ] **Advanced Reporting** with charts and exports
- [ ] **Mobile API** optimizations
- [ ] **Plugin System** for extensibility
- [ ] **Multi-tenant Support**
- [ ] **Advanced Workflows** with automation
- [ ] **Integration APIs** (Slack, Teams, etc.)

---

**Built with ❤️ by the TaskMaster Team**