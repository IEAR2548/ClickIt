# ClickIt

URL Shortener service written in Go, built for high performance and low-latency redirects using PostgreSQL, Redis caching, and asynchronous click analytics logging.

---

## Features

- **Base62 URL Shortening:** Generates compact, collision-safe short codes from sequence IDs.
- **Fast Redirects with Redis Cache:** Caches frequently accessed links with TTL to minimize database queries.
- **Asynchronous Click Analytics:** Tracks click metadata (IP address, user agent, referrer, timestamp) via a non-blocking buffered background worker (`ClickLogger`).
- **Graceful Shutdown:** Ensures in-flight HTTP requests finish and buffered analytics events are drained safely before closing database connections.
- **Docker Ready:** Includes a multi-stage `Dockerfile` and `docker-compose.yml` for zero-setup local deployment.

---

## Tech Stack

- **Language:** Go 1.25
- **Database:** PostgreSQL (with `pgx/v5` connection pool)
- **Cache:** Redis (`go-redis/v9`)
- **Containerization:** Docker & Docker Compose

---

## Getting Started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/) (if running locally)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)

### 1. Environment Configuration

Copy `.env.example` to `.env` and adjust the variables if needed:

```bash
cp .env.example .env
```

| Variable | Description | Default |
|---|---|---|
| `PORT` | HTTP server listening port | `8080` |
| `BASE_URL` | Base URL returned in short link responses | `http://localhost:8080` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://clickit:clickit@localhost:5432/clickit?sslmode=disable` |
| `REDIS_ADDR` | Redis host:port (optional, runs without cache if unset) | `localhost:6379` |

### 2. Run with Docker Compose (Recommended)

Start the app, PostgreSQL, and Redis in one command:

```bash
docker compose up --build
```

### 3. Run Locally

1. Start database & cache:
   ```bash
   docker compose up -d postgres redis
   ```
2. Run database migrations:
   ```bash
   psql "$DATABASE_URL" -f migrations/001_init_schema.sql
   psql "$DATABASE_URL" -f migrations/002_add_clicks_table.sql
   ```
3. Run the application:
   ```bash
   go run .
   ```

---

## API Reference

### 1. Health Check
```http
GET /health
```
**Response:** `200 OK`

---

### 2. Create Short Link
```http
POST /shorten
Content-Type: application/json

{
  "url": "https://example.com/very/long/url"
}
```

**Response:** `201 Created`
```json
{
  "short_code": "1B",
  "short_url": "http://localhost:8080/1B"
}
```

---

### 3. Redirect Short Link
```http
GET /{code}
```
**Response:** `302 Found` with `Location: <long_url>` header. Automatically logs click analytics asynchronously.

---

## Testing

Run unit tests including race condition detector:

```bash
go test -v -race ./...
```
