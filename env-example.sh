# Application Configuration
APP_NAME=TaskMaster
APP_VERSION=1.0.0
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080
APP_HOST=0.0.0.0

# Database Configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=taskmaster
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRES_IN=24h
JWT_REFRESH_EXPIRES_IN=168h

# Security Configuration
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m
PASSWORD_MIN_LENGTH=8

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

# Metrics Configuration
ENABLE_METRICS=true
METRICS_PORT=9090

# Email Configuration (for future features)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_EMAIL=noreply@taskmaster.dev
SMTP_FROM_NAME=TaskMaster

# File Upload Configuration
MAX_UPLOAD_SIZE=10MB
UPLOAD_PATH=./uploads

# Background Jobs Configuration
ENABLE_BACKGROUND_JOBS=true
WORKER_CONCURRENCY=5

# External Integrations
SLACK_WEBHOOK_URL=
TEAMS_WEBHOOK_URL=

# Performance Configuration
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m
DB_CONN_MAX_IDLE_TIME=30s

# Cache Configuration
CACHE_DEFAULT_TTL=1h
CACHE_ENABLED=true

# Development/Testing
ENABLE_SWAGGER=true
ENABLE_PROFILING=false
MIGRATE_ON_STARTUP=false