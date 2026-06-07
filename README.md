# Skill Test Project - PDF Report Generation Microservice

This is a prototype of a skill test assignment, which is to implement a pdf report generation microservice via API Integration.

It is built based on a full-stack student management app and implemented a standalone 
microservice in `Golang 1.23` to generate PDF reports for students by calling the backend API of student management app.

## 🚀 Quick Start

To avoid installing various dependencies locally, I implemented it in docker.

### Prerequisites
- Docker (29.5 or higher)
- Docker Compose (5.1 or higher)

### Installation & Setup

- Run `docker compose up --build` from the repository root.

#### Confirm the original app works well (optional)
- You could open `http://localhost:5173` and log in as `admin@school-admin.com` / `3OU4zn3q6Zh9` to confirm the student management app works well

#### Test PDF generation service
Once the app is started, 13 dummy students (id from `1` to `13`) will be added to the database automatically.

- You can open the browser and go to `http://localhost:8080/api/v1/students/5/report`
- You can open the terminal and run `curl -OJ http://localhost:8080/api/v1/students/6/report`

Either can trigger the PDF generation and download it to your local.

### Clean up
- Run `docker compose down -v` from the repository root to stop the service and remove the data

## 🟢 My Testing Result
### Service Status
```
nan.so@mypc go_skill_accessment % docker compose ps   
NAME                     IMAGE                           COMMAND                   SERVICE      CREATED          STATUS                    PORTS
school-mgmt-backend      school-mgmt-backend:latest      "docker-entrypoint.s…"   backend      23 minutes ago   Up 11 seconds             0.0.0.0:5007->5007/tcp, [::]:5007->5007/tcp
school-mgmt-db           postgres:16-alpine              "docker-entrypoint.s…"   db           4 hours ago      Up 16 seconds (healthy)   0.0.0.0:5432->5432/tcp, [::]:5432->5432/tcp
school-mgmt-frontend     school-mgmt-frontend:latest     "/docker-entrypoint.…"   frontend     23 minutes ago   Up 11 seconds             0.0.0.0:5173->80/tcp, [::]:5173->80/tcp
school-mgmt-go-service   school-mgmt-go-service:latest   "/app/go-service"         go-service   23 minutes ago   Up 11 seconds             0.0.0.0:8080->80/tcp, [::]:8080->80/tcp
```
### Generated PDF
[`student_4.pdf`](./student_report_4.pdf)

## 📁 Project Structure

```
go_skill_accessment/
├── frontend/             # React + TypeScript + Material-UI
├── backend/              # Node.js + Express + PostgreSQL
├── Go-service/           # Go microservice for PDF reports
├── seed_db/              # Database schema and seed data
├── docker-compose.yml    # docker compose file for orchestrating the services
├── README.md
```

## 📝 What I Did On the Top Of The Original Codebase

### PR
Please see the PR for all changes
- https://github.com/NolanZONG/go_skill_accessment/pull/1/changes

### Fix Issues
#### Implemented missing CRUD operations for student management backend API (Known Issue #1)
All five handlers in `backend/src/modules/students/students-controller.js` were empty. Implemented them following the same pattern as `staffs-controller.js`.

#### Fixed Notice Description Not Saving (Known Issue #2)
The root cause was on the frontend:
- `frontend/src/domains/notice/components/notice-form.tsx` registered the textarea against `content`
- `frontend/src/domains/notice/pages/add-notice-page.tsx` initialized `initialState` with `content: ''`

but the zod schema and the backend repository expected `description`.

Renamed both occurrences to `description`. 
As a side benefit, `npm run build` (which runs `tsc` before `vite build`) now passes inside the Docker image.

### Authentication for service-to-service integration
#### Problem
In the original codebase, the backend `ONLY` supports a cookie-based auth flow built for a browser frontend: login sets three cookies (`accessToken`, `refreshToken`, `csrfToken`) and every protected route requires both `authenticateToken` (which reads access + refresh cookies) and `csrfProtection` (which compares the `x-csrf-token` header against an HMAC embedded inside the access token JWT). 

For Go-service implementation, even if it is a backend-to-backend integration, I have to parse `Set-Cookie` headers out of the login response, save three cookies, attach an `x-csrf-token` header on every request.

The `README` claimed "JWT authentication" but in practice the JWT was never exposed as a Bearer token.

#### Solution
To make the API usable for the Go-service without breaking the browser frontend, I updated several parts: 

- `backend/src/middlewares/authenticate-token.js`: accepts an `Authorization: Bearer <jwt>` header as an alternative to the cookie pair. When there is a bearer token in the request header, then it sets `req.isBearerAuth = true` so downstream middlewares can detect it. The original cookie-based branch is preserved unchanged for the browser.
- `backend/src/middlewares/csrf-protection.js`: short-circuits and calls `next()` when `req.isBearerAuth` is true.
- `backend/src/modules/auth/auth-controller.js`: `handleLogin` now also returns `accessToken` and `csrfToken` in the JSON response body alongside `accountBasic`. The cookie path is unchanged (browser ignores the extra fields), but a service caller can now grab the token without parsing `Set-Cookie` headers.

After these changes, the Go-service can authenticate in two HTTP calls without any cookie machinery.

```bash
TOKEN=$(curl -s -X POST http://localhost:5007/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin@school-admin.com","password":"3OU4zn3q6Zh9"}' \
  | jq -r .accessToken)

curl http://localhost:5007/api/v1/students/6 -H "Authorization: Bearer $TOKEN"
```

#### Notes
- Bearer requests verify the JWT with the same `JWT_ACCESS_TOKEN_SECRET` as cookie requests, so the trust model is unchanged.
- The CSRF bypass relies heavily on the presence of the `Authorization` request header. The two paths will never interfere with each other.

### Demo data

The original `seed_db/seed-db.sql` only provided the admin account and a few lookup tables. Most of pages and items were empty. 

To make the setup and test simpler, I

- added `seed_db/demo-data.sql`: a small dummy dataset.
- added `/docker-entrypoint-initdb.d/03-demo.sql` in `docker-compose.yml`, so it runs strictly after `01-tables.sql` and `02-seed.sql`.

### Docker env

To avoid installing various dependencies locally and make the dev and setup simpler, I created a container-based development environment.

- Added `Dockerfile` for `backend` and `frontend`.
- For local Docker dev we just need a tiny SPA static server, so I introducted `nginx` and added `frontend/nginx.conf`.
- Added `docker-compose.yml` to orchestrates the three services with PostgreSQL.

### Go-service Implementation
After fixed all the issues and completed the preparation, I implemented the `Go-service` according to the requirements.
Please see the [`Go-service/README.md`](./Go-service/README.md) for detail.

## 🗺️ Improvement Point In The Future
### API Doc Visualization
Currently, I only write an openapi.yaml in the Go-service, we can consider to introduce Swagger UI and make the API specification visible and easily accessible to callers.

### Authentication and Authorization
The original backend is designed for browser-only access, the added bearer token is a work around rather than a best pratice. We should consider that

- Add OAuth2 to backend service and register Go-service as an OAuth2 client for better permission control
- Add authentication to Go-service if needed

### Secret Management
Currently, all the secrets write in plaintext. We can consider
- Use docker secret
- Use AWS/GCP Secret Manager

### Observability
- Collect metrics to monitoring tool for distributed tracing
- Log aggregation
- Proper alerts

### Reliability
- Perf test for optimizing container resource limits
- Move PDFs from local volume: write them to cloud storage

### Others
- Student reports contain PII, so if required, we may need to consider things such as audit log, field-level redaction, PDF retention policy, compliance regime.
- CI/CD pipelines for code quality and deployment stability.
