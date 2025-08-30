#!/bin/bash

# setup.sh - Complete Setup Script for Gask TaskMaster

set -e

echo "🚀 Setting up Gask TaskMaster API..."

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 1. Environment Setup
echo -e "${YELLOW}📝 Setting up environment...${NC}"
if [ ! -f .env ]; then
    cp .env.example .env
    echo -e "${GREEN}✅ .env file created${NC}"
else
    echo -e "${BLUE}ℹ️  .env file already exists${NC}"
fi

# 2. Go Dependencies
echo -e "${YELLOW}📦 Installing Go dependencies...${NC}"
go mod tidy
echo -e "${GREEN}✅ Dependencies installed${NC}"

# 3. Database Setup (Docker)
echo -e "${YELLOW}🐳 Setting up database with Docker...${NC}"
docker-compose up -d db redis
echo -e "${GREEN}✅ Database containers started${NC}"

# Wait for database to be ready
echo -e "${YELLOW}⏳ Waiting for database to be ready...${NC}"
sleep 10

# 4. Run Migrations
echo -e "${YELLOW}🗄️  Running database migrations...${NC}"
go run cmd/api/main.go migrate up
echo -e "${GREEN}✅ Migrations completed${NC}"

# 5. Create Admin User
echo -e "${YELLOW}👤 Creating admin user...${NC}"
go run cmd/api/main.go user create
echo -e "${GREEN}✅ Admin user created${NC}"

# 6. Build Application
echo -e "${YELLOW}🔨 Building application...${NC}"
go build -o bin/taskmaster cmd/api/main.go
echo -e "${GREEN}✅ Application built${NC}"

# 7. Make test script executable
echo -e "${YELLOW}🧪 Setting up test script...${NC}"
chmod +x scripts/test-api.sh
echo -e "${GREEN}✅ Test script ready${NC}"

echo -e "\n${GREEN}🎉 Setup completed successfully!${NC}"
echo -e "\n${BLUE}Next steps:${NC}"
echo -e "1. Start the server: ${YELLOW}./bin/taskmaster serve${NC}"
echo -e "2. Or run with: ${YELLOW}go run cmd/api/main.go serve${NC}"
echo -e "3. Test API: ${YELLOW}./scripts/test-api.sh${NC}"
echo -e "4. Visit: ${YELLOW}http://localhost:8080${NC}"
echo -e "5. API Docs: ${YELLOW}http://localhost:8080/docs/simple-swagger.html${NC}"

echo -e "\n${BLUE}Database Info:${NC}"
echo -e "- Host: localhost:5432"
echo -e "- Database: taskmaster"
echo -e "- User: postgres"
echo -e "- Password: postgres"

echo -e "\n${BLUE}Admin User:${NC}"
echo -e "- Email: admin@taskmaster.dev"
echo -e "- Password: admin123"
echo -e "- Role: admin"
