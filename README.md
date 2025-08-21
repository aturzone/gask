# Gask

# Gask Task Manager - Setup Guide

## Fixed Issues

1. **Missing import statements** - Added proper imports in utils.go
2. **Incomplete handlers** - Created full CRUD operations for tasks, notes, and boxes
3. **Missing routes** - Added all necessary API endpoints
4. **Frontend connectivity** - Fixed API calls and error handling
5. **CORS issues** - Properly configured CORS middleware
6. **Go module version** - Updated to Go 1.21 for compatibility
7. **UI/UX improvements** - Modern, responsive design with proper feedback

## Directory Structure
```
gask/
├── backend/
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── models.go
│   ├── handlers.go
│   ├── server.go
│   ├── storage.go
│   └── utils.go
└── frontend/
    └── index.html
```

## Backend Setup

1. **Navigate to backend directory:**
   ```bash
   cd backend
   ```

2. **Initialize Go modules and download dependencies:**
   ```bash
   go mod tidy
   ```

3. **Run the server:**
   ```bash
   go run .
   ```

The server will start on port 7887 and you should see:
```
Server configured with routes:
  GET    /api/tasks
  POST   /api/tasks
  PUT    /api/tasks/{id}
  DELETE /api/tasks/{id}
  GET    /api/notes
  POST   /api/notes
  DELETE /api/notes/{id}
  GET    /api/boxes
  POST   /api/boxes
  DELETE /api/boxes/{id}
  GET    /health
Server starting on :7887
```

## Frontend Setup

1. **Navigate to frontend directory:**
   ```bash
   cd frontend
   ```

2. **Serve the HTML file using a local server:**

   **Option 1 - Python:**
   ```bash
   python3 -m http.server 8080
   ```

   **Option 2 - Node.js (if you have npx):**
   ```bash
   npx serve .
   ```

   **Option 3 - PHP:**
   ```bash
   php -S localhost:8080
   ```

3. **Open your browser and visit:**
   ```
   http://localhost:8080
   ```

## Authentication

The API uses Basic Authentication:
- **Username:** `admin`
- **Password:** `securepass`

This is automatically handled by the frontend.

## Features

### Tasks
- ✅ Create tasks with title, description, and due date
- ✅ View tasks with both Gregorian and Shamsi dates
- ✅ Delete tasks
- ✅ Auto-save to JSON file

### Notes
- ✅ Create notes with title and content
- ✅ View creation dates
- ✅ Delete notes

### Boxes
- ✅ Create organizational boxes
- ✅ Track number of tasks per box
- ✅ Delete boxes

## API Endpoints

### Tasks
- `GET /api/tasks` - Get all tasks
- `POST /api/tasks` - Create new task
- `PUT /api/tasks/{id}` - Update task
- `DELETE /api/tasks/{id}` - Delete task

### Notes
- `GET /api/notes` - Get all notes
- `POST /api/notes` - Create new note
- `DELETE /api/notes/{id}` - Delete note

### Boxes
- `GET /api/boxes` - Get all boxes
- `POST /api/boxes` - Create new box
- `DELETE /api/boxes/{id}` - Delete box

### Health Check
- `GET /health` - Check server status

## Data Storage

- Data is stored in `backend/data.json`
- Automatic backup on every change
- Thread-safe operations with mutex locks

## Production Considerations

1. **Security:**
   - Change default credentials
   - Use environment variables for sensitive data
   - Implement proper JWT authentication
   - Add rate limiting

2. **CORS:**
   - Update CORS settings to specific domains
   - Remove wildcard (*) for production

3. **Database:**
   - Consider using PostgreSQL or MySQL
   - Implement proper database migrations

4. **Deployment:**
   - Use process managers like PM2 or systemd
   - Set up reverse proxy with Nginx
   - Configure HTTPS with SSL certificates

5. **Monitoring:**
   - Add logging middleware
   - Implement health checks
   - Set up error tracking

## Troubleshooting

### Backend Issues
- **Port 7887 already in use:** Change port in `server.go`
- **Permission denied:** Check file permissions for data.json
- **Dependencies not found:** Run `go mod tidy`

### Frontend Issues
- **CORS errors:** Ensure backend is running and CORS is configured
- **API connection failed:** Check backend URL in frontend
- **Authentication errors:** Verify credentials are correct

### Testing the Setup

1. **Health Check:**
   ```bash
   curl http://localhost:7887/health
   ```
   Should return: `{"status":"ok"}`

2. **Create a task via API:**
   ```bash
   curl -X POST http://localhost:7887/api/tasks \
     -H "Content-Type: application/json" \
     -H "Authorization: Basic YWRtaW46c2VjdXJlcGFzcw==" \
     -d '{"title":"Test Task","description":"Test Description"}'
   ```

3. **Get tasks:**
   ```bash
   curl -H "Authorization: Basic YWRtaW46c2VjdXJlcGFzcw==" \
     http://localhost:7887/api/tasks
   ```

Your Gask Task Manager should now be fully functional! 🚀

## License
This project is licensed under the GNU General Public License v3.0 (GPL-3.0)  
See the [LICENSE](LICENSE) file for details.
