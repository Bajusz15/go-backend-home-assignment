# Food Ordering API

A RESTful food ordering system built with Go. Customers can register, browse restaurants, and place orders. Restaurants can manage incoming orders and update their status.

## Tech Stack

- **Go** with [Chi](https://github.com/go-chi/chi) router (stdlib `http.Handler` compatible)
- **PostgreSQL** with `database/sql` and [pgx](https://github.com/jackc/pgx) driver (no ORM — raw SQL)
- **JWT** authentication via [golang-jwt](https://github.com/golang-jwt/jwt)
- **Swagger/OpenAPI** docs via [swaggo/swag](https://github.com/swaggo/swag)
- **Docker Compose** for local development

## Quick Start

### Using Docker Compose (recommended)

```bash
docker compose up --build
```

This starts PostgreSQL and the API server. Migrations and seed data are applied automatically.

The API is available at `http://localhost:8080`.

### Running Locally

Prerequisites: Go 1.26+, PostgreSQL running locally.

1. Create the database:
   ```bash
   createdb foodorder
   ```

2. Set environment variables (or copy `.env.example` to `.env` and source it):
   ```bash
   export JWT_SECRET=your-secret-key
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_USER=postgres
   export DB_PASSWORD=postgres
   export DB_NAME=foodorder
   export DB_SSLMODE=disable
   ```

3. Run the server:
   ```bash
   go run ./cmd/api
   ```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | — | Secret key for signing JWT tokens |
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | No | — | Full PostgreSQL connection string (overrides individual DB_* vars) |
| `DB_HOST` | No | `localhost` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | No | `postgres` | Database user |
| `DB_PASSWORD` | No | `postgres` | Database password |
| `DB_NAME` | No | `foodorder` | Database name |
| `DB_SSLMODE` | No | `disable` | PostgreSQL SSL mode |

## API Documentation

Swagger UI is available at: **http://localhost:8080/swagger/index.html**

To regenerate docs after modifying handler annotations:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go -o docs
```

## Running Tests

```bash
go test ./...
```

## API Overview

### Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register a new customer |
| POST | `/auth/login` | No | Login and receive a JWT token |
| GET | `/auth/who-am-i` | Yes | Get current user info |

### Restaurants (public)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/restaurants` | List all restaurants |
| GET | `/restaurants/{id}` | Get restaurant details with menu |
| GET | `/restaurants/{id}/menu` | Get restaurant menu |

### Orders

| Method | Path | Role | Description |
|--------|------|------|-------------|
| POST | `/orders` | Customer | Place a new order |
| GET | `/orders` | Restaurant | List all orders for the restaurant |
| GET | `/orders/{id}` | Restaurant | Get order details with customer and items |
| PATCH | `/orders/{id}` | Restaurant | Update order status |

Order statuses follow a forward-only progression: `received` → `preparing` → `ready` → `delivered`.

## Seed Data

Docker Compose seeds three restaurants with menus. To log in as a restaurant:

| Email | Password |
|-------|----------|
| `mario@restaurant.com` | `restaurant123` |
| `sakura@restaurant.com` | `restaurant123` |
| `ali@restaurant.com` | `restaurant123` |

## Project Structure

```
cmd/api/              Entry point, routing, server setup
internal/
  config/             Environment-based configuration
  database/           Connection pool setup
  handler/            HTTP handlers (request parsing, validation, response)
  middleware/         JWT authentication and role authorization
  model/              Data types
  repository/         Data access layer (raw SQL)
  service/            Business logic
migrations/           SQL migration and seed scripts
docs/                 Generated Swagger/OpenAPI files
```

## Design Decisions

- **Raw SQL over ORM**: Chose `database/sql` with pgx for full control over queries. No hidden behavior, easy to reason about, and idiomatic Go.
- **Chi router**: Uses stdlib `http.Handler`/`http.HandlerFunc` interfaces. No framework lock-in.
- **Unified auth**: Single `users` table with a `role` field. Customers register via API; restaurant accounts are seeded. Both use the same login endpoint.
- **Server-side price calculation**: Order total is computed from current menu prices at order time. `price_at_time` is stored per order item so price changes don't affect historical orders.
- **Forward-only status transitions**: Prevents invalid state changes (e.g., cannot go from `delivered` back to `preparing`).
- **Repository interfaces in service layer**: Services depend on interfaces, not concrete repository types, enabling unit testing without a database.

## What I'd Improve With More Time

- **Integration tests**: Add end-to-end tests using `httptest` with a real test database.
- **Real-time updates**: SSE endpoint for customers to receive live order status changes.
- **Pagination**: Add cursor-based pagination to list endpoints.
- **Request logging**: Structured request/response logging with correlation IDs.
- **Graceful migration handling**: Separate migration CLI command instead of running on startup.
- **CI pipeline**: GitHub Actions for lint, test, and build.
