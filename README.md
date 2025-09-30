# GASK - Go-based Advanced taSK Management System

<div align="center">

```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║    ██████╗  █████╗ ███████╗██╗  ██╗                     ║
║   ██╔════╝ ██╔══██╗██╔════╝██║ ██╔╝                     ║
║   ██║  ███╗███████║███████╗█████╔╝                      ║
║   ██║   ██║██╔══██║╚════██║██╔═██╗                      ║
║   ╚██████╔╝██║  ██║███████║██║  ██╗                     ║
║    ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝                     ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
```

**Production-Ready Task Management System**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[Features](#-features) • [Quick Start](#-quick-start) • [Architecture](#-architecture) • [API Docs](#-api-documentation) • [Deployment](#-deployment)

</div>

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Architecture](#-architecture)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [API Documentation](#-api-documentation)
- [Development](#-development)
- [Deployment](#-deployment)
- [Monitoring](#-monitoring)
- [Troubleshooting](#-troubleshooting)
- [Contributing](#-contributing)

---

## 🌟 Overview

**GASK** is a production-ready task management system built with Go, featuring:
- **High Performance**: Redis-first architecture with background PostgreSQL sync
- **Flexible Deployment**: Automatic port detection and configuration
- **Enterprise-Ready**: Role-based access control, health monitoring, and comprehensive API
- **Developer-Friendly**: Complete Docker setup with one-command deployment

---

## ✨ Features

### Core Features
- ✅ **Multi-User Support** with three-tier role system (Owner, Group Admin, User)
- ✅ **Group Management** for organizing teams and departments
- ✅ **Task Management** with priorities, deadlines, and status tracking
- ✅ **Role-Based Access Control** with granular permissions
- ✅ **Work Time Tracking** for flexible scheduling
- ✅ **Auto-Sync** between Redis and PostgreSQL (configurable interval)
- ✅ **Advanced Search** across users and tasks
- ✅ **Batch Operations** for bulk task updates
- ✅ **Health Monitoring** with built-in health checks

### Infrastructure Features
- 🔄 **Automatic Port Detection** - finds available ports automatically
- 🐳 **Docker-Ready** - complete containerization with docker-compose
- 📊 **Real-Time Monitoring** - advanced monitoring dashboard
- 💾 **Backup & Restore** - automated backup scripts
- 🔒 **Security** - secure authentication with owner and user roles
- 📈 **Scalable** - designed for horizontal scaling

---

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client        │───▶│   GASK API      │───▶│   gaskRedis     │
│   (HTTP/REST)   │    │   (gaskMain)    │    │   (Primary)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │  Sync Service   │───▶│  gaskPostgres   │
                       │  (15min cycle)  │    │  (Persistent)   │
                       └─────────────────┘    └─────────────────┘
```

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| **gaskMain** | API Server | Go 1.21, Gorilla Mux |
| **gaskRedis** | Primary Data Store | Redis 7.2 |
| **gaskPostgres** | Persistent Storage | PostgreSQL 15 |
| **Sync Service** | Data Synchronization | Background Worker |

---

## 🚀 Quick Start

### Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- 2GB RAM minimum
- 5GB disk space

### One-Command Setup

```bash
# Clone repository
git clone <your-repo-url>
cd gask

# Make startup script executable
chmod +x start-gask.sh

# Start GASK (automatically finds available ports)
./start-gask.sh
```

That's it! GASK will:
1. ✅ Check prerequisites
2. ✅ Find available ports automatically
3. ✅ Build Docker images
4. ✅ Start all services
5. ✅ Verify health
6. ✅ Show connection details

### Alternative: Using Makefile

```bash
# Setup environment
make setup

# Deploy for production
make prod

# View logs
make logs

# Check health
make health

# Stop services
make down
```

### Manual Setup

```bash
# 1. Create environment file
cp .env.example .env
nano .env  # Edit with your values

# 2. Build and start
docker-compose up -d

# 3. Check health
curl http://localhost:7890/health
```

---

## ⚙️ Configuration

### Environment Variables

All configuration is centralized in `.env`:

```env
# Application
APP_NAME=gask
ENVIRONMENT=production
LOG_LEVEL=info

# API Server (auto port detection enabled)
API_PORT=7890
AUTO_PORT_FIND=true

# Redis
REDIS_HOST=localhost
REDIS_PORT=6380

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_USER=airflow
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=airflow

# Authentication
OWNER_PASSWORD=your_owner_password
OWNER_EMAIL=admin@company.com

# Sync
SYNC_INTERVAL=15m

# System
TZ=Asia/Tehran
```

### Port Configuration

GASK automatically finds available ports if configured ports are busy:

- **API_PORT**: Starting from 7890, tries up to 7990
- **REDIS_PORT**: Starting from 6380, tries up to 6480
- **POSTGRES_PORT**: Starting from 5433, tries up to 5533

Disable auto-detection:
```env
AUTO_PORT_FIND=false
```

---

## 📚 API Documentation

### Base URL
```
http://localhost:7890
```

### Authentication

**Owner Access** (Full Control):
```bash
curl -H "X-Owner-Password: your_password" http://localhost:7890/users
```

**User Access** (Basic Auth):
```bash
# Using User ID
curl -u "USER_ID:password" http://localhost:7890/users/USER_ID

# Using Email
curl -u "user@email.com:password" http://localhost:7890/users/1
```

### Quick API Examples

#### Create User
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "John Doe",
    "email": "john@company.com",
    "password": "secure123",
    "role": "user",
    "group_ids": [1]
  }' \
  http://localhost:7890/users
```

#### Create Task
```bash
curl -X POST \
  -u "USER_ID:password" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Complete project",
    "priority": 1,
    "deadline": "2025-12-31",
    "group_id": 1
  }' \
  http://localhost:7890/users/USER_ID/tasks
```

#### Search Tasks
```bash
curl -H "X-Owner-Password: admin1234" \
  "http://localhost:7890/tasks/search?q=project"
```

### Complete API Reference

📖 **Full API documentation**: See [API_REFERENCE.md](docs/API_REFERENCE.md)

Key endpoints:
- 👥 **Users**: `/users`, `/users/{id}`
- 👔 **Groups**: `/groups`, `/groups/{id}`
- 📋 **Tasks**: `/users/{id}/tasks`, `/tasks/search`
- 📊 **Stats**: `/tasks/stats`, `/groups/{id}/stats`
- 🔧 **Admin**: `/admin/sync`, `/admin/status`
- 🏥 **Health**: `/health`

---

## 🛠️ Development

### Local Development

```bash
# Install Go dependencies
go mod download

# Set environment variables
export OWNER_PASSWORD=admin1234
export API_PORT=7890

# Run locally (requires Redis and PostgreSQL)
go run main.go
```

### Development with Docker

```bash
# Start with live logs
make dev

# Or
docker-compose up
```

### Hot Reload (Optional)

Install Air for hot reload:
```bash
go install github.com/cosmtrek/air@latest
air
```

### Running Tests

```bash
# Run test suite
make test

# Or manually
chmod +x test_api.sh
./test_api.sh
```

---

## 🚢 Deployment

### Production Deployment

#### 1. Security Setup

```bash
# Generate strong passwords
openssl rand -base64 32  # For OWNER_PASSWORD
openssl rand -base64 32  # For POSTGRES_PASSWORD

# Update .env
nano .env
```

#### 2. Deploy

```bash
# Option 1: Using smart startup script
./start-gask.sh

# Option 2: Using Makefile
make prod

# Option 3: Manual
docker-compose up -d
```

#### 3. Verify

```bash
# Check health
make health

# View logs
make logs

# Monitor
./monitor-gask.sh
```

### Scaling for Production

#### Horizontal Scaling

```yaml
# docker-compose.yml
services:
  gaskMain:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1'
          memory: 512M
```

#### Resource Limits

Edit `docker-compose.yml` to set resource limits:

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 1G
    reservations:
      cpus: '1'
      memory: 512M
```

### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:7890;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 📊 Monitoring

### Real-Time Monitoring

```bash
# Start monitoring dashboard
./monitor-gask.sh

# Custom refresh interval
./monitor-gask.sh -i 10  # Refresh every 10 seconds
```

### Health Checks

```bash
# Quick health check
make health

# Detailed status
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/status
```

### Log Management

```bash
# View all logs
make logs

# View specific service
make logs-api
make logs-redis
make logs-postgres

# Last 50 lines
docker-compose logs --tail=50 gaskMain
```

### Metrics

```bash
# System statistics
make stats

# Admin statistics
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/stats
```

---

## 🔧 Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# GASK automatically finds available ports!
# If you want to force a specific port:
API_PORT=8000 ./start-gask.sh
```

#### Services Not Starting
```bash
# Check logs
make logs

# Check Docker
docker ps -a

# Restart services
make restart
```

#### Redis Connection Failed
```bash
# Check Redis health
docker exec gaskRedis redis-cli ping

# View Redis logs
make logs-redis

# Restart Redis
docker-compose restart gaskRedis
```

#### PostgreSQL Connection Failed
```bash
# Check PostgreSQL
docker exec gaskPostgres pg_isready -U airflow

# View PostgreSQL logs
make logs-postgres

# Restart PostgreSQL
docker-compose restart gaskPostgres
```

#### Data Not Syncing
```bash
# Check sync status
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/status

# Force sync
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/sync?action=force
```

### Debug Mode

Enable debug logging:
```bash
# In .env
LOG_LEVEL=debug

# Restart
make restart
```

---

## 💾 Backup & Restore

### Backup

```bash
# Backup PostgreSQL
make backup

# Backup Redis
make backup-redis

# Manual backup
docker exec gaskPostgres pg_dump -U airflow airflow > backup.sql
```

### Restore

```bash
# Restore from backup
make restore FILE=backups/backup.sql

# Manual restore
docker exec -i gaskPostgres psql -U airflow airflow < backup.sql
```

### Automated Backups

Add to crontab:
```bash
# Daily backup at 2 AM
0 2 * * * cd /path/to/gask && make backup
```

---

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices
- Add tests for new features
- Update documentation
- Use meaningful commit messages

---

## 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Go community for excellent tools and libraries
- Redis and PostgreSQL teams
- Docker for containerization platform

---

## 📞 Support

- 📧 Email: support@gask.io
- 💬 Issues: [GitHub Issues](https://github.com/your-repo/issues)
- 📖 Documentation: [docs.gask.io](https://docs.gask.io)

---

<div align="center">

**Made with ❤️ by GASK Team**

⭐ Star us on GitHub if you find this useful!

</div>