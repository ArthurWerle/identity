# Identity Service

A Go-based identity service for managing users and feature flags. This service handles user data and feature flag assignments without authentication or authorization logic.

## Features

- **User Management**: Full CRUD operations for users
- **Feature Flag Management**: Create, read, update, and delete feature flags
- **Feature Flag Assignments**: Assign and unassign feature flags to users
- **RESTful API**: Clean REST endpoints with proper HTTP methods
- **Swagger Documentation**: Auto-generated API documentation
- **Database Migrations**: Schema versioning with Atlas
- **Structured Logging**: JSON logging with slog
- **Layered Architecture**: Clean separation of concerns (handler → service → repository)
- **Docker Support**: Easy deployment with Docker and docker-compose
- **Health Checks**: Built-in health check endpoint

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin
- **ORM**: GORM
- **Database**: PostgreSQL 16
- **Migrations**: Atlas
- **Logging**: slog (standard library)
- **Documentation**: Swagger/OpenAPI (swaggo)

## Project Structure

```
identity/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── handler/                 # HTTP handlers
│   ├── middleware/              # HTTP middleware
│   ├── model/                   # Database models
│   ├── repository/              # Data access layer
│   └── service/                 # Business logic layer
│       └── dto/                 # Data transfer objects
├── db/
│   └── migrations/              # Database migration files
├── docs/                        # Swagger documentation (generated)
├── docker-compose.yml           # Docker compose configuration
├── Dockerfile                   # Docker image definition
├── Makefile                     # Build and development commands
├── atlas.hcl                    # Atlas configuration
└── README.md                    # This file
```

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Make (optional, but recommended)
- Atlas CLI (for migrations)
- swag CLI (for Swagger generation)

## Quick Start

### 1. Clone the repository

```bash
git clone <repository-url>
cd identity
```

### 2. Install dependencies

```bash
make deps
```

This will:
- Download Go dependencies
- Install swag CLI for Swagger generation
- Tidy up go.mod

### 3. Generate Swagger documentation

```bash
make swagger
```

### 4. Start the database

```bash
make db-start
```

Or start all services with docker-compose:

```bash
make docker-up
```

### 5. Run migrations

```bash
make migrate-up
```

### 6. Run the application

```bash
make run
```

The service will start on `http://localhost:8080`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `8080` |
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `5432` |
| `DB_USER` | Database user | `identity` |
| `DB_PASSWORD` | Database password | `identity_dev_password` |
| `DB_NAME` | Database name | `identity_db` |
| `DB_SSLMODE` | Database SSL mode | `disable` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## API Endpoints

### Health Check

- `GET /health` - Service health check

### Users

- `POST /api/v1/users` - Create a new user
- `GET /api/v1/users` - Get all users (paginated)
- `GET /api/v1/users/:id` - Get a specific user
- `PUT /api/v1/users/:id` - Update a user
- `DELETE /api/v1/users/:id` - Delete a user (soft delete)

### Feature Flags

- `POST /api/v1/feature-flags` - Create a new feature flag
- `GET /api/v1/feature-flags` - Get all feature flags (paginated)
- `GET /api/v1/feature-flags/:id` - Get a specific feature flag
- `PUT /api/v1/feature-flags/:id` - Update a feature flag
- `DELETE /api/v1/feature-flags/:id` - Delete a feature flag

### User Feature Flag Assignments

- `GET /api/v1/users/:id/feature-flags` - Get all feature flags for a user
- `POST /api/v1/users/:id/feature-flags/:key` - Assign a feature flag to a user
- `DELETE /api/v1/users/:id/feature-flags/:key` - Unassign a feature flag from a user

## API Documentation

Once the service is running, you can access the Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

## Database Schema

### Users Table

| Column | Type | Description |
|--------|------|-------------|
| id | serial | Primary key |
| name | varchar(255) | User's name |
| email | varchar(255) | User's email (unique) |
| enabled | boolean | Whether user is active |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last update timestamp |
| deleted_at | timestamp | Soft delete timestamp |
| last_login | timestamp | Last login timestamp (nullable) |

### Feature Flags Table

| Column | Type | Description |
|--------|------|-------------|
| id | serial | Primary key |
| key | varchar(255) | Feature flag key (unique) |
| description | text | Feature description |
| enabled | boolean | Default enabled state |
| created_at | timestamp | Creation timestamp |
| updated_at | timestamp | Last update timestamp |
| deleted_at | timestamp | Soft delete timestamp |

### User Feature Flags Table

| Column | Type | Description |
|--------|------|-------------|
| user_id | integer | User ID (FK) |
| feature_flag_id | integer | Feature flag ID (FK) |
| created_at | timestamp | Assignment timestamp |

## Makefile Commands

```bash
make help           # Display all available commands
make build          # Build the application
make run            # Run the application locally
make test           # Run tests
make test-coverage  # Run tests with coverage report
make lint           # Run linter
make fmt            # Format code
make clean          # Clean build artifacts

# Docker
make docker-build   # Build Docker image
make docker-up      # Start all services
make docker-down    # Stop all services
make docker-logs    # View logs

# Database
make db-start       # Start only the database
make db-stop        # Stop the database
make db-reset       # Reset database (WARNING: deletes all data)
make migrate-up     # Run migrations
make migrate-status # Check migration status

# Development
make swagger        # Generate Swagger docs
make deps           # Install dependencies
make dev            # Start development environment
make setup          # Full project setup
```

## Development Workflow

1. **Initial setup**:
   ```bash
   make setup
   ```

2. **Start developing**:
   ```bash
   make dev
   ```

3. **Make changes** to code

4. **Run tests**:
   ```bash
   make test
   ```

5. **Format code**:
   ```bash
   make fmt
   ```

6. **Update Swagger docs** (if you changed API):
   ```bash
   make swagger
   ```

## Testing

### Run all tests
```bash
make test
```

### Run tests with coverage
```bash
make test-coverage
```

This generates a `coverage.html` file that you can open in your browser.

### Run specific tests
```bash
go test -v ./internal/service/... -run TestUserService_CreateUser
```

## Docker Deployment

### Build the image
```bash
make docker-build
```

### Run with docker-compose
```bash
make docker-up
```

This starts:
- PostgreSQL database on port 5432
- Identity service on port 8080

### View logs
```bash
make docker-logs
```

### Stop services
```bash
make docker-down
```

## Architecture

The service follows a layered architecture pattern:

```
HTTP Request
     ↓
[Handler Layer]      - HTTP handling, request validation
     ↓
[Service Layer]      - Business logic, validation
     ↓
[Repository Layer]   - Data access
     ↓
[ORM (GORM)]        - Object-relational mapping
     ↓
[Database]          - PostgreSQL
```

### Design Principles

- **Separation of Concerns**: Each layer has a single responsibility
- **Dependency Injection**: Dependencies are injected, making testing easier
- **Interface-Based**: Repositories use interfaces for better testability
- **Context Propagation**: context.Context is used throughout for cancellation
- **Error Handling**: Errors are properly wrapped and logged
- **Validation**: Input validation at the handler and service layers

## Future Enhancements

- [ ] Add authentication (JWT, OAuth2)
- [ ] Add authorization (RBAC, ABAC)
- [ ] Add caching layer (Redis)
- [ ] Add rate limiting
- [ ] Add metrics (Prometheus)
- [ ] Add distributed tracing (OpenTelemetry)
- [ ] Add audit logging
- [ ] Add GraphQL API
- [ ] Add event sourcing
- [ ] Add message queue integration

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For issues and questions, please open an issue in the repository.
