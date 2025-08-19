# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a server management system built with Go backend and vanilla HTML/CSS/JavaScript frontend. It provides a web interface for managing directories, Docker scripts, port usage, and system services.

## Project Structure

```
server-manage/
├── backend/
│   ├── go.mod          # Go module definition
│   └── main.go         # Main application with all API handlers
└── frontend/
    ├── static/
    │   ├── css/
    │   │   └── style.css    # Application styles
    │   └── js/
    │       └── app.js        # Frontend JavaScript logic
    └── templates/
        └── index.html        # Main HTML template
```

## Common Development Commands

### Backend (Go)
```bash
cd backend
go mod tidy                    # Install dependencies
go run main.go                 # Start development server on port 8080
go build -o server main.go     # Build binary
```

### Frontend
The frontend is served statically by the Go backend. No separate build process is required.

### Running the Application
```bash
cd backend
go run main.go
# Access at http://localhost:8080
```

## Architecture Overview

### Backend Architecture
- **Framework**: Uses Gorilla Mux for routing
- **Structure**: Single-file application (`main.go`) containing all business logic
- **API Design**: RESTful JSON API with standard HTTP methods
- **System Integration**: Direct execution of system commands (netstat, systemctl, ps, bash)

### Frontend Architecture
- **Framework**: Vanilla JavaScript (no framework dependencies)
- **Structure**: Single-page application with tab-based navigation
- **Communication**: Fetch API for AJAX calls to backend
- **UI**: Custom CSS with responsive design

### Key Features
1. **Directory Management**: Browse, create, delete, rename files and directories
2. **Docker Script Management**: Create, view, execute, and delete Docker-related scripts
3. **Container Management**: Start, stop, restart, remove Docker containers and view logs
4. **Image Management**: Pull, remove, and prune Docker images
5. **Port Monitoring**: View system port usage with process information
6. **Service Management**: Start, stop, restart system services (requires systemctl)

### API Endpoints

#### Directory Management
- `GET /api/directory?path=<path>` - List directory contents
- `POST /api/directory` - Create new directory
- `DELETE /api/directory` - Delete file/directory
- `PUT /api/directory/rename` - Rename file/directory

#### Docker Scripts
- `GET /api/docker-scripts?path=<path>` - List Docker scripts
- `POST /api/docker-scripts` - Create new script
- `DELETE /api/docker-scripts` - Delete script
- `POST /api/docker-scripts/execute` - Execute script

#### Container Management
- `GET /api/containers` - List all containers
- `POST /api/containers/start` - Start container
- `POST /api/containers/stop` - Stop container
- `POST /api/containers/restart` - Restart container
- `POST /api/containers/remove` - Remove container
- `POST /api/containers/logs` - Get container logs

#### Image Management
- `GET /api/images` - List all images
- `POST /api/images/remove` - Remove image
- `POST /api/images/pull` - Pull image from registry
- `POST /api/images/prune` - Remove unused images

#### System Information
- `GET /api/ports` - Get port usage information
- `GET /api/services` - Get service status
- `POST /api/services/start` - Start service
- `POST /api/services/stop` - Stop service
- `POST /api/services/restart` - Restart service

## Development Guidelines

### Code Style
- **Go**: Follow standard Go conventions, use meaningful variable names
- **JavaScript**: Use ES6+ features, camelCase naming, avoid global variables
- **HTML/CSS**: Use semantic HTML, BEM-style class naming, responsive design

### Error Handling
- Backend returns appropriate HTTP status codes (200, 400, 500)
- Frontend displays user-friendly error messages via notifications
- System command errors are captured and returned in API responses

### Security Considerations
- Application executes system commands - ensure proper input validation
- Currently no authentication - consider adding for production use
- File operations are restricted to server's accessible directories
- Scripts are executed with server's user permissions
- Docker operations require appropriate Docker daemon permissions
- Container and image operations can affect system resources significantly

### Testing
- No automated tests currently implemented
- Manual testing through web interface
- Test system commands functionality on target platform

## System Requirements

- Go 1.21+
- Docker (for container and image management)
- Linux system with systemctl (for service management)
- netstat or ss command (for port monitoring)
- Modern web browser with ES6+ support

## Dependencies

### Backend
- `github.com/gorilla/mux v1.8.1` - HTTP router

### Frontend
- No external dependencies - uses vanilla JavaScript and CSS

## File Locations

- **Main Application**: `backend/main.go`
- **Frontend Logic**: `frontend/static/js/app.js`
- **Styling**: `frontend/static/css/style.css`
- **HTML Template**: `frontend/templates/index.html`
- **Go Module**: `backend/go.mod`

## Important Notes

- The application requires appropriate system permissions to execute commands
- Service management features require systemctl and appropriate privileges
- Port monitoring uses netstat/ss commands
- File operations use standard Go os package functions
- The frontend uses relative paths for API calls - ensure backend is running on the same domain