# Go Service - Student Report Generation

A small Go microservice that generates a PDF report for a single student by
calling the school-mgmt backend API.

## 🚀 Quick Start

## Run with Docker Compose

### Prerequisites
- Docker (29.5 or higher)
- Docker Compose (5.1 or higher)

### Installation & Setup
The service is wired into the repository-level [`docker-compose.yml`](../docker-compose.yml)

From the repository root:

```bash
docker compose up --build go-service
```
The container listens on `80`, the compose file maps it to host `8080`.

```bash
# Health check
curl http://localhost:8080/healthz

# Generate the PDF for student id 6 and save it locally
curl -OJ http://localhost:8080/api/v1/students/6/report
# -> student_6.pdf saved in the current directory

# You could also inspect the same file inside the container's volume
docker run --rm -v school-mgmt-go-pdf:/data alpine ls -la /data
```

If the backend is empty (no demo data), reset the database first:

```bash
docker compose down -v && docker compose up --build
```

That re-runs `seed_db/demo-data.sql`, which creates eight students with ids
starting at the lowest free `users.id` value.

## Run locally without Docker

### Prerequisites

- Go 1.23+.

### Installation & Setup

```bash
cd Go-service
go mod tidy
BACKEND_BASE_URL=http://localhost:5007 \
BACKEND_ADMIN_USERNAME=admin@school-admin.com \
BACKEND_ADMIN_PASSWORD=3OU4zn3q6Zh9 \
PORT=8080 OUTPUT_DIR=./output \
go run ./cmd/server
```

Then in another shell:

```bash
curl -OJ http://localhost:8080/api/v1/students/6/report
```

## Environment Configuration

All settings are configured in environment variables.

| Variable                   | Default                  | Description                                                 |
| -------------------------- | ------------------------ | ----------------------------------------------------------- |
| `PORT`                     | `80`                     | HTTP listen port (binds in container via `cap_net_bind_service`) |
| `BACKEND_BASE_URL`         | `http://backend:5007`    | Base URL for the school-mgmt backend                        |
| `BACKEND_ADMIN_USERNAME`   | *(required)*             | Username used for backend login                             |
| `BACKEND_ADMIN_PASSWORD`   | *(required)*             | Password used for backend login                             |
| `REQUEST_TIMEOUT`          | `5s`                     | Per-request timeout for `GET /students/:id`                 |
| `LOGIN_TIMEOUT`            | `5s`                     | Per-request timeout for `POST /auth/login`                  |
| `OUTPUT_DIR`               | `./output`               | Where `student_{id}.pdf` files are written                  |
| `BREAKER_MAX_REQUESTS`     | `3`                      | gobreaker half-open probe count                             |
| `BREAKER_INTERVAL`         | `30s`                    | gobreaker counter reset interval                            |
| `BREAKER_TIMEOUT`          | `20s`                    | gobreaker open -> half-open delay                           |
| `SHUTDOWN_TIMEOUT`         | `15s`                    | Max time to wait for in-flight requests on SIGTERM          |
| `LOG_LEVEL`                | `info`                   | `debug` / `info` / `warn` / `error`                         |

## 🛠️ Technology Stack

### Core Flow

When `GET /api/v1/students/{id}/report` is called, then

```
   ─► login to school-mgmt backend (cached bearer token)
   ─► GET http://backend:5007/api/v1/students/{id}
   ─► render PDF with maroto/v2
   ─► write OUTPUT_DIR/student_{id}.pdf atomically
   ─► stream the same bytes back as application/pdf
```

### Key Dependencies:

- **chi** - routing
- **maroto/v2** - PDF rendering
- **sony/gobreaker** - circuit breaker
- **golang.org/x/sync/singleflight** - coalesce concurrent requests for the
  same student id (only one fetch + one render runs)
- **log/slog** - structured JSON logging with per-request `request_id`

### Others
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Token refresh on `401` (re-logs in once, retries once)
- The service `NEVER` connect to database directly; every request fetch data through the backend API per skill test requirement.

## 📁 Project Structure

```
Go-service/
├── cmd/server/main.go         # entry point + graceful shutdown
├── internal/
│   ├── config/                # env-var configuration
│   ├── logger/                # slog JSON logger + ctx helpers
│   ├── httpapi/               # chi router, middleware, handler, error mapping
│   ├── service/               # ReportService (singleflight + atomic write)
│   ├── backend/               # backend HTTP client, token cache, breaker
│   └── pdf/                   # Generator interface + maroto implementation
├── api/openapi.yaml           # OpenAPI 3.1 spec
├── output/                    # PDF output dir (mounted as a Docker volume)
├── Dockerfile
└── go.mod
```

## 📚 API Documentation

### Base URL
```
http://localhost:8080
```

## Endpoints

| Method | Path                              | Description                          |
| ------ | --------------------------------- | ------------------------------------ |
| GET    | `/healthz`                        | Liveness probe                       |
| GET    | `/api/v1/students/{id}/report`    | Generate and download student PDF      |

See [`api/openapi.yaml`](api/openapi.yaml) for the full schema.

### Error codes

| HTTP | `code`                  | Meaning                                                      |
| ---- | ----------------------- | ------------------------------------------------------------ |
| 400  | `INVALID_ID`            | Path id is not a positive integer                            |
| 404  | `STUDENT_NOT_FOUND`     | Backend returned 404 for this student id                     |
| 502  | `BACKEND_UNAVAILABLE`   | Network error, non-2xx upstream, or decode failure           |
| 502  | `BACKEND_AUTH_FAILED`   | Could not log in / re-login to the backend                   |
| 503  | `BACKEND_BREAKER_OPEN`  | Circuit breaker tripped; backend is unhealthy                |
| 504  | `BACKEND_TIMEOUT`       | Backend exceeded `REQUEST_TIMEOUT`                           |
| 500  | `INTERNAL_ERROR`        | Unhandled panic or unexpected failure                        |



## 🧪 Testing

```bash
cd Go-service
go test ./...

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

The Dockerfile also runs `go test ./... -count=1` during build, so a broken
test fails the image build.



## Known limitations

- The Bearer token cache lives in process memory. Restarting the service
  forces one extra login round trip on the first request; this is intentional
  for simplicity.
- `singleflight` coalesces concurrent requests for the **same** id while a
  render is in flight. Requests that arrive after the prior render has
  completed will trigger a fresh fetch + render and overwrite the file
  atomically.
- The token refresh path retries a request exactly once on `401`. If the
  second attempt also fails with `401`, the failure surfaces as
  `BACKEND_AUTH_FAILED` instead of an infinite refresh loop.
