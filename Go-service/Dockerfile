# - build stage -
# Run `go mod tidy` inside the container so the repo does not need to commit a go.sum 
# that was generated against a specific host's Go install.
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache the module download layer when only source changes by copying go.mod first. 
# tidy + build happen after the full copy.
RUN apk add --no-cache git ca-certificates
COPY go.mod ./
RUN go mod download || true

COPY . .
RUN go mod tidy

# Run the unit test suite at build time so a regression breaks the image
# build rather than landing silently in production.
RUN go test ./... -count=1

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/go-service ./cmd/server

# - runtime stage -
FROM alpine:3.20

# libcap gives us setcap so a non-root user can bind to port 80.
RUN apk add --no-cache ca-certificates libcap tzdata && \
    addgroup -S app && adduser -S -G app -u 10001 app && \
    mkdir -p /app/output && chown -R app:app /app

WORKDIR /app
COPY --from=build /out/go-service /app/go-service
RUN setcap 'cap_net_bind_service=+ep' /app/go-service

USER app
EXPOSE 80
VOLUME ["/app/output"]
ENTRYPOINT ["/app/go-service"]
