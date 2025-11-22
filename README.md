# PR Reviewer Assignment Service

**A microservice for automatically assigning reviewers to Pull Requests**

Automates the process of assigning reviewers to teams, manages users and PRs. Interaction is exclusively via the HTTP API.


##  Quick start


```bash
  docker-compose up --build
```

The service is available at **http://localhost:8080**

```bash
# Check
curl http://localhost:8080/team/get?team_name=backend
```

## Требования

- **Minimum**: Docker & Docker Compose
- **Locally**: Go 1.24, PostgreSQL 15+ (or use Docker)

---


##  API Endpoints 

### Teams
| Method | Endpoint | Description |
|-------|----------|---------|
| POST | `/team/add` | Create a team with members |
| GET | `/team/get?team_name=<name>` | Get a command |

### Users
| Method | Endpoint | Description |
|-------|----------|---------|
| POST | `/users/setIsActive` | Set the activity status |
| POST | `/users/deactivateBatch` |  Massively deactivate + reassign PR |
| GET | `/users/getReview?user_id=<id>` | Get PRs where the reviewer is a user |

### Pull Requests
| Method | Endpoint | Description |
|-------|----------|---------|
| POST | `/pullRequest/create` | Create a PR + auto-assign reviewers |
| POST | `/pullRequest/merge` | Mark PR as merged |
| POST | `/pullRequest/reassign` | Reassign a reviewer |

### Statistics & Health
| Method | Endpoint | Description |
|-------|----------|---------|
| GET | `/stats` | Get appointment statistics |

---

## Development teams

### Makefile

```bash
  make help              # Show all commands
  make build             # Build an app
  make run               # Build and launch
  make test              # Run the tests
  make lint              # Check the code (golangci-lint)
  make fmt               # Format the code
  make tidy              # Update Dependencies
  make ci                # CI pipeline (lint + test)
  make docker-up         # Docker Compose up
  make docker-down       # Docker Compose down
  make clean             # Clear the artifacts
```

### Without build tools

```bash
go build -o bin/pr-reviewer-app ./cmd/api
go test ./...
go run ./cmd/api/main.go
```

## Testing

### All tests

```bash
  make test             
```

### Specific tests

```bash
# Unit tests of services
go test ./internal/service -v

# HTTP tests
go test ./internal/http -v

# Load tests (1000 operations)
go test ./internal/http -run Load -v

# E2E tests (5 scenarios)
go test ./internal/http -run E2E -v
```


##  Project structure

```
├──config/config.yml 
├── cmd/api/
│   └── main.go                
├── internal/
│   ├── api/
│   │   ├── types.gen.go        
│   │   └── server.gen.go       
│   ├── config/
│   │   └── config.go            
│   ├── http/
│   │   ├── server.go           
│   │   ├── server_test.go      
│   │   ├── load_test.go       
│   │   ├── e2e_test.go         
│   │   └── handler/
│   │       └── server_handler.go 
│   ├── repository/
│   │   ├── team_repository.go
│   │   ├── user_repository.go
│   │   ├── pull_request_repository.go     
│   │   ├── inmemory/           
│   │   └── postgres/           
│   │       ├── db.go
│   │       ├── team_repository.go
│   │       ├── user_repository.go
│   │       └── pull_request_repository.go
│   └── service/
│       ├── team_service.go
│       ├── team_service_test.go
│       ├── user_service.go
│       ├── user_service_test.go
│       ├── pull_request_service.go
│       └── pull_request_service_test.go
├── migrations/
│   └── 001_init.sql            
├── .golangci.yml               
├── Makefile                    
├── Dockerfile                 
├── docker-compose.yml          
├── openapi.yml                
├── go.mod & go.sum             
└── README.md                  
```

---

##  Solution Architecture

### Layered Architecture

```
HTTP Layer (handlers)
    ↓
Service Layer (business logic)
    ↓
Repository Layer (data access)
    ↓
PostgreSQL / In-Memory Storage
```

### Key Design Decisions

1. **Repository Pattern** - Two implementations: PostgreSQL (production) and in-memory (tests)
2. **Dependency Injection** - Services receive repositories through constructors
3. **Idempotent Merge** - Merging PR twice does not cause an error
4. **Random Selection** - Reviewers are selected randomly, excluding the author
5. **Batch Operations** - `/users/deactivateBatch` optimized for <100ms

## Business Rules

### Reviewer Assignment

-  When creating PR: up to 2 active reviewers from the author's team
-  Random selection (distributed assignment)
-  Reviewer ≠ PR author
-  If <2 active available: assign available quantity

### Reassignment

-  Selects random active member from current reviewer's team
-  Cannot reassign on merged PR (code: `PR_MERGED`)
-  Cannot reassign someone who is not assigned (code: `NOT_ASSIGNED`)
-  Not possible if no candidates available (code: `NO_CANDIDATE`)

### Deactivation

-  User with `is_active=false` will not receive new PRs
-  During mass deactivation, open PRs are reassigned
-  Reassignment completes in <100ms for 100 users

---

## Configuration

### config/config.yml

```yaml
server:
  port: ":8080"
  env: "local"
database:
  host: "localhost"
  port: "5432"
  dbname: "pr_review_db"
  user: "postgres"
  password: "root"
  sslmode: "disable"
```

**Migration Content** (`001_init.sql`):
- Table `teams` - teams
- Table `users` - team users
- Table `pull_requests` - PRs with status
- Table `pr_reviewers` - PR ↔ reviewers relationship
- Indexes for fast search

## Technology Selection Justification

### go-chi
Chi provides zero memory allocations for routing, which is optimal for our 5 RPS requirements. It works on top of the 
standard net/http library, ensuring full compatibility with the entire Go ecosystem and no vendor lock-in. This is important
for quick assignment verification - the code looks idiomatic and understandable. All our handlers use standard ResponseWriter 
and Request interfaces, making testing straightforward and simple.

### sqlx
Sqlx stays close to the standard database/sql. It solves the main pain point of raw SQL: tedious scanning of large numbers 
of fields through rows.Scan(). Compared to alternatives: pure database/sql would require too much boilerplate code, while 
a full ORM like Gorm would be excessive for our simple queries and would create unnecessary magic.

### viper
Unlike simpler alternatives that only handle basic config file parsing, Viper supports multiple formats like YAML, JSON, 
and TOML out of the box, which is crucial for maintaining clean and readable configuration. It seamlessly integrates with 
environment variables, allowing us to follow Twelve-Factor App principles where environment-specific configs override 
defaults - essential for Docker deployments and different environments.

Compared to manual configuration parsing that would require writing boilerplate code for each config field, Viper provides 
automatic binding and type conversion. For our use case with database connections and server settings, this means less error-prone 
code and faster development. The ability to watch for config file changes in real-time, while not critical for this project, 
demonstrates Viper's production-ready feature set that would benefit future scaling.

Most importantly, Viper strikes the right balance between power and simplicity - it doesn't force complex abstractions 
but provides just enough automation to handle configuration management professionally without the overhead of more 
enterprise-focused solutions.