# 🚀 GASK Quick Start Guide

Get GASK up and running in less than 2 minutes!

---

## ⚡ Super Quick Start (One Command)

```bash
git clone <repo-url> && cd gask && chmod +x start-gask.sh && ./start-gask.sh
```

That's it! GASK will automatically:
- ✅ Find available ports
- ✅ Build everything
- ✅ Start services
- ✅ Show you the URL

---

## 📋 Step-by-Step (Recommended)

### 1. Clone Repository
```bash
git clone <your-repo-url>
cd gask
```

### 2. Run Smart Startup
```bash
chmod +x start-gask.sh
./start-gask.sh
```

### 3. Access GASK
```bash
# Check health
curl http://localhost:7890/health

# Get admin token
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/status
```

---

## 🎯 First API Calls

### Create Your First User
```bash
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "John Developer",
    "email": "john@company.com",
    "password": "secure123",
    "role": "user",
    "group_ids": []
  }' \
  http://localhost:7890/users
```

### Create a Group
```bash
# First, get your user ID from the previous response
# Let's say it's 2

curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Engineering",
    "admin_id": 2
  }' \
  http://localhost:7890/groups
```

### Create Your First Task
```bash
# Using user ID 2 and group ID 2
curl -X POST \
  -u "2:secure123" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Task",
    "priority": 1,
    "deadline": "2025-12-31",
    "information": "Getting started with GASK",
    "group_id": 2
  }' \
  http://localhost:7890/users/2/tasks
```

### List Your Tasks
```bash
curl -u "2:secure123" http://localhost:7890/users/2/tasks
```

---

## 🛠️ Using Makefile (Alternative)

If you prefer Makefile commands:

```bash
# Setup
make setup

# Start
make up

# Check health
make health

# View logs
make logs

# Stop
make down
```

---

## 📊 Monitoring

### Quick Health Check
```bash
make health
# or
curl http://localhost:7890/health
```

### Real-Time Monitoring
```bash
./monitor-gask.sh
```

### View Logs
```bash
# All services
make logs

# Just API
make logs-api
```

---

## 🎛️ Configuration

### Change Ports
If ports are busy, GASK auto-finds available ones!

Or manually set:
```bash
# Edit .env
API_PORT=8000
REDIS_PORT=6381
POSTGRES_PORT=5434
```

### Change Password
```bash
# Edit .env
OWNER_PASSWORD=your_secure_password_here
```

Then restart:
```bash
make restart
```

---

## 🧪 Testing

### Run Test Suite
```bash
make test
```

### Manual Tests
```bash
# Health
curl http://localhost:7890/health

# Admin status (requires auth)
curl -H "X-Owner-Password: admin1234" \
  http://localhost:7890/admin/status

# Create test user
curl -X POST \
  -H "X-Owner-Password: admin1234" \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Test","email":"test@test.com","password":"test123"}' \
  http://localhost:7890/users
```

---

## 🔧 Common Commands

```bash
# Start services
make up

# Stop services
make down

# Restart
make restart

# View logs
make logs

# Check health
make health

# Monitor
./monitor-gask.sh

# Backup database
make backup

# Clean everything
make clean-all  # ⚠️ Deletes all data!
```

---

## 📖 Next Steps

1. **Read Full Documentation**: Check `README.md`
2. **Explore API**: See all endpoints in API documentation
3. **Set Up Monitoring**: Use `./monitor-gask.sh`
4. **Configure for Production**: Update `.env` with secure passwords
5. **Set Up Backups**: Add cron job for `make backup`

---

## ❓ Need Help?

- 📖 Full docs: `README.md`
- 🐛 Issues: GitHub Issues
- 💬 Questions: Open a discussion

---

## 🎉 You're Ready!

GASK is now running! Start building your tasks and managing your team.

**API Endpoint**: `http://localhost:7890`

**Owner Password**: Check your `.env` file (default: `admin1234`)

Happy task managing! 🚀