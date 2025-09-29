# 🐳 Docker Setup Guide - Task Manager API

Complete guide for deploying Task Manager API using Docker Compose.

---

## 📋 Prerequisites

Before starting, ensure you have:

- **Docker**: Version 20.10 or higher
- **Docker Compose**: Version 2.0 or higher
- **Git**: To clone the repository

### Install Docker

**Linux:**
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
```

**macOS:**
```bash
brew install docker docker-compose
```

**Windows:**
Download and install [Docker Desktop](https://www.docker.com/products/docker-desktop)

### Verify Installation

```bash
docker --version
docker-compose --version
```

---

## 🚀 Quick Start (5 Minutes)

### Step 1: Clone Repository

```bash
git clone <your-repo-url>
cd task-manager
```

### Step 2: Create Environment File

```bash
# Copy example to .env
cp .env.example .env

# Edit with your preferred editor
nano .env  # or vim, code, etc.
```

**Important:** Change these values in `.env`:
```bash
OWNER_PASSWORD=your_secure_password_here
POSTGRES_PASSWORD=another_secure_password_here
```

### Step 3: Start Services

```bash
# Build and start all services
docker-compose up -d

# Or use Makefile (if available)
make up
```

### Step 4: Verify Health

```bash
# Check if API is running
curl http://localhost:7890/health

# Expected response:
# {"status":"healthy","timestamp":"2025-09-29T10:00:00Z"}
```

### Step 5: Test Authentication

```bash
# Test owner access
curl -H "X-Owner-Password: your_password" \
  http://localhost:7890/users
```

**✅ Done!** Your API is now running at `http://localhost:7890`

---

## 📁 File Structure

After setup, your directory should look like:

```
task-manager/
├── docker-compose.yml      # Docker Compose configuration
├── Dockerfile              # Go application image
├── .env                    # Your environment variables (gitignored)
├── .env.example            # Environment template
├── .dockerignore           # Files to ignore in build
├── Makefile               # Optional: convenient commands
├── DOCKER_SETUP.md        # This file
├── go.mod
├── go.sum
├── main.go
├── handlers/
├── models/
├── modules/
└── logs/                  # Created automatically
```

---

## 🎯 Detailed Configuration

### docker-compose.yml Overview

The setup includes three services:

#### 1. **Redis Service**
- **Image:** redis:7.2-alpine
- **Port:** 6380 → 6379 (host → container)
- **Purpose:** Primary data storage (in-memory)
- **Persistence:** Enabled with AOF (Append-Only File)

#### 2. **PostgreSQL Service**
- **Image:** postgres:15-alpine
- **Port:** 5433 → 5432 (host → container)
- **Purpose:** Persistent storage and backup
- **Database:** Configurable via `POSTGRES_DB`

#### 3. **API Service**
- **Build:** From local Dockerfile
- **Port:** 7890
- **Purpose:** Task Manager API
- **Dependencies:** Waits for Redis and PostgreSQL to be healthy

### Volume Persistence

Data is persisted in Docker volumes:
- `redis_data`: Redis database files
- `postgres_data`: PostgreSQL database files

**Location:** `/var/lib/docker/volumes/` (Linux/Mac)

---

## 🎛️ Environment Variables

### Required Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OWNER_PASSWORD` | admin1234 | System owner password |
| `OWNER_EMAIL` | admin@gmail.com | Owner email address |
| `POSTGRES_PASSWORD` | (long string) | PostgreSQL password |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | airflow | PostgreSQL username |
| `POSTGRES_DB` | airflow | Database name |
| `SERVER_PORT` | 7890 | API server port |
| `LOG_LEVEL` | info | Logging level |
| `TZ` | Asia/Tehran | Timezone |

### Changing Environment Variables

1. Edit `.env` file
2. Restart services:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

---

## 🔧 Using Makefile Commands

If you have `make` installed, these shortcuts are available:

### Essential Commands

```bash
make help           # Show all available commands
make setup          # Create .env from example
make build          # Build Docker images
make up             # Start all services
make down           # Stop all services
make restart        # Restart all services
make logs           # View logs (all services)
```

### Monitoring Commands

```bash
make ps             # Show running containers
make health         # Check service health
make stats          # Show resource usage
make logs-api       # View API logs only
make logs-redis     # View Redis logs only
make logs-postgres  # View PostgreSQL logs only
```

### Debugging Commands

```bash
make shell-api      # Open shell in API container
make shell-redis    # Open Redis CLI
make shell-postgres # Open PostgreSQL shell
```

### Maintenance Commands

```bash
make backup-postgres        # Backup database
make restore-postgres FILE= # Restore from backup
make clean                  # Remove all data (!)
make update                 # Pull latest images
```

---

## 📊 Managing Containers

### View Running Containers

```bash
docker-compose ps
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f redis
docker-compose logs -f postgres

# Last 100 lines
docker-compose logs --tail=100 api
```

### Restart Specific Service

```bash
docker-compose restart api
```

### Stop and Remove Containers

```bash
# Stop (keep data)
docker-compose down

# Stop and remove volumes (DELETE ALL DATA!)
docker-compose down -v
```

---

## 🐛 Troubleshooting

### Problem: Port Already in Use

**Symptoms:** Error: `port is already allocated`

**Solution:**
```bash
# Find what's using the port
lsof -i :7890  # or :6380, :5433

# Kill the process
kill -9 <PID>

# Or change port in docker-compose.yml
ports:
  - "8890:7890"  # Use different host port
```

---

### Problem: Container Keeps Restarting

**Check logs:**
```bash
docker-compose logs api
```

**Common causes:**
1. **Database not ready:** Wait 30 seconds after first start
2. **Wrong password:** Check `.env` file
3. **Port conflict:** See above

**Solution:**
```bash
# Check health
docker-compose ps

# Restart with fresh logs
docker-compose restart api
docker-compose logs -f api
```

---

### Problem: Permission Denied

**Symptoms:** Cannot write to volume

**Solution (Linux):**
```bash
# Fix volume permissions
sudo chown -R $USER:$USER logs/

# Or run with sudo
sudo docker-compose up -d
```

---

### Problem: Cannot Connect to PostgreSQL

**Check PostgreSQL is running:**
```bash
docker-compose exec postgres pg_isready -U airflow
```

**Test connection:**
```bash
docker-compose exec postgres psql -U airflow -d airflow -c "SELECT 1;"
```

**View PostgreSQL logs:**
```bash
docker-compose logs postgres
```

---

### Problem: Redis Connection Failed

**Check Redis is running:**
```bash
docker-compose exec redis redis-cli ping
# Should return: PONG
```

**Test connection from API container:**
```bash
docker-compose exec api sh
# Inside container:
wget -qO- http://redis:6379
```

---

### Problem: API Returns 500 Error

**Steps to debug:**

1. **Check API logs:**
   ```bash
   docker-compose logs -f api
   ```

2. **Verify services are healthy:**
   ```bash
   docker-compose ps
   # All should show "Up (healthy)"
   ```

3. **Check API health:**
   ```bash
   curl http://localhost:7890/health
   ```

4. **Check admin status (if owner):**
   ```bash
   curl -H "X-Owner-Password: your_password" \
     http://localhost:7890/admin/status
   ```

---

### Problem: Data Not Persisting

**Check volumes exist:**
```bash
docker volume ls | grep task
```

**Inspect volume:**
```bash
docker volume inspect taskmanager_postgres_data
```

**Backup before fixing:**
```bash
make backup-postgres
# or
docker-compose exec postgres pg_dump -U airflow airflow > backup.sql
```

---

## 💾 Backup and Restore

### Backup PostgreSQL

**Using Makefile:**
```bash
make backup-postgres
# Saves to: backups/backup_YYYYMMDD_HHMMSS.sql
```

**Using Docker directly:**
```bash
mkdir -p backups
docker-compose exec -T postgres pg_dump -U airflow airflow > backups/backup.sql
```

### Restore PostgreSQL

**Using Makefile:**
```bash
make restore-postgres FILE=backups/backup.sql
```

**Using Docker directly:**
```bash
docker-compose exec -T postgres psql -U airflow airflow < backups/backup.sql
```

### Backup Redis

```bash
# Redis automatically saves to disk with AOF
# To manually save:
docker-compose exec redis redis-cli BGSAVE

# To backup the AOF file:
docker cp taskmanager_redis:/data/appendonly.aof backups/redis_backup.aof
```

---

## 🔒 Production Best Practices

### 1. Security Hardening

**Change default passwords:**
```bash
# Generate strong passwords
openssl rand -base64 32

# Update .env file
OWNER_PASSWORD=<generated_password>
POSTGRES_PASSWORD=<generated_password>
```

**Restrict network access:**
```yaml
# In docker-compose.yml, remove port mappings for internal services:
services:
  redis:
    # Remove this for production:
    # ports:
    #   - "6380:6379"
    
  postgres:
    # Remove this for production:
    # ports:
    #   - "5433:5432"
```

**Add Redis password:**
```yaml
services:
  redis:
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD}
```

---

### 2. Resource Limits

Add resource constraints:

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
  
  redis:
    deploy:
      resources:
        limits:
          memory: 256M
  
  postgres:
    deploy:
      resources:
        limits:
          memory: 512M
```

---

### 3. Logging Configuration

**Limit log file sizes:**

```yaml
services:
  api:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

---

### 4. Health Checks

Already configured! Verify with:

```bash
docker-compose ps
# Should show "(healthy)" status
```

---

### 5. Use Secrets (Docker Swarm)

For Docker Swarm deployments:

```yaml
secrets:
  owner_password:
    external: true
  postgres_password:
    external: true

services:
  api:
    secrets:
      - owner_password
      - postgres_password
    environment:
      OWNER_PASSWORD_FILE: /run/secrets/owner_password
      POSTGRES_PASSWORD_FILE: /run/secrets/postgres_password
```

---

## 🚀 Deployment Scenarios

### Development

```bash
# Start with live logs
docker-compose up

# Or in background
docker-compose up -d
docker-compose logs -f
```

---

### Staging

```bash
# Use separate .env file
cp .env.example .env.staging
nano .env.staging

# Start with staging config
docker-compose --env-file .env.staging up -d
```

---

### Production

**1. Prepare environment:**
```bash
cp .env.example .env.production
# Edit with strong passwords and production values
nano .env.production
```

**2. Build production images:**
```bash
docker-compose --env-file .env.production build --no-cache
```

**3. Deploy:**
```bash
docker-compose --env-file .env.production up -d
```

**4. Verify:**
```bash
curl http://your-server:7890/health
```

**5. Setup monitoring:**
```bash
# Add to crontab:
*/5 * * * * curl -f http://localhost:7890/health || systemctl restart docker-compose
```

---

### Production with Reverse Proxy (Nginx)

**nginx.conf:**
```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:7890;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**With SSL (Let's Encrypt):**
```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d api.yourdomain.com
```

---

## 📈 Monitoring

### Container Stats

```bash
# Real-time stats
docker stats

# Or with Makefile
make stats
```

### Application Metrics

```bash
# System health
curl http://localhost:7890/health

# Admin status (owner only)
curl -H "X-Owner-Password: your_password" \
  http://localhost:7890/admin/status

# System statistics (owner only)
curl -H "X-Owner-Password: your_password" \
  http://localhost:7890/admin/stats
```

### External Monitoring

**Prometheus + Grafana setup:**

```yaml
# Add to docker-compose.yml
services:
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
```

---

## 🔄 Updates and Maintenance

### Update Docker Images

```bash
# Pull latest images
docker-compose pull

# Rebuild and restart
docker-compose up -d --build

# Or with Makefile
make update
```

### Update Application Code

```bash
# Pull latest code
git pull origin main

# Rebuild
docker-compose build api

# Restart
docker-compose up -d

# Or with Makefile
make clean-build
```

### Scheduled Maintenance

**Create backup script:** `backup.sh`
```bash
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup PostgreSQL
docker-compose exec -T postgres pg_dump -U airflow airflow > \
  ${BACKUP_DIR}/postgres_${DATE}.sql

# Backup Redis
docker cp taskmanager_redis:/data/appendonly.aof \
  ${BACKUP_DIR}/redis_${DATE}.aof

# Keep only last 7 days
find ${BACKUP_DIR} -name "*.sql" -mtime +7 -delete
find ${BACKUP_DIR} -name "*.aof" -mtime +7 -delete

echo "Backup completed: ${DATE}"
```

**Add to crontab:**
```bash
# Daily backup at 2 AM
0 2 * * * /path/to/backup.sh >> /var/log/backup.log 2>&1
```

---

## 🧪 Testing

### Run Test Suite

```bash
# Make test script executable
chmod +x test_api.sh

# Run tests
./test_api.sh

# Or with Makefile
make test
```

### Manual Testing

```bash
# Health check
curl http://localhost:7890/health

# Create test user
curl -X POST \
  -H "X-Owner-Password: your_password" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Test User",
    "email": "test@example.com",
    "password": "test123",
    "role": "user",
    "group_ids": [1]
  }' \
  http://localhost:7890/users
```

---

## 🆘 Getting Help

### Check Logs First

```bash
# All services
docker-compose logs -f

# API only (most common issues)
docker-compose logs -f api

# Last 50 lines
docker-compose logs --tail=50
```

### Common Log Locations

Inside containers:
- API: `/root/logs/` (if mounted)
- PostgreSQL: `/var/log/postgresql/`
- Redis: Logs to stdout (use `docker-compose logs redis`)

### Debug Mode

Enable debug logging:

```bash
# Edit .env
LOG_LEVEL=debug

# Restart
docker-compose restart api
```

---

## 📚 Additional Resources

- **Docker Documentation**: https://docs.docker.com
- **Docker Compose Reference**: https://docs.docker.com/compose/
- **Redis Documentation**: https://redis.io/documentation
- **PostgreSQL Documentation**: https://www.postgresql.org/docs/

---

## 🎯 Quick Command Reference

```bash
# Setup
cp .env.example .env && nano .env
docker-compose up -d

# Monitor
docker-compose ps
docker-compose logs -f
curl http://localhost:7890/health

# Maintain
docker-compose restart
docker-compose down
docker-compose up -d --build

# Backup
make backup-postgres

# Clean
docker-compose down -v  # ⚠️ Deletes all data!

# Debug
docker-compose exec api sh
docker-compose logs -f api
```

---

## ✅ Deployment Checklist

Before going to production:

- [ ] Changed `OWNER_PASSWORD` to strong password
- [ ] Changed `POSTGRES_PASSWORD` to strong password
- [ ] Removed unnecessary port mappings
- [ ] Set up SSL/HTTPS
- [ ] Configured firewall rules
- [ ] Set up automated backups
- [ ] Configured monitoring
- [ ] Tested restore procedure
- [ ] Set up log rotation
- [ ] Documented admin procedures
- [ ] Configured resource limits
- [ ] Tested health checks
- [ ] Set up alerting

---

**🐳 Happy Dockerizing!**