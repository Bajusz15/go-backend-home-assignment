# Food Ordering API

This project is a small food ordering API written in Go. Customers can create an account, browse restaurant menus, and place orders. Restaurant users can read the orders placed with their restaurant and move them through the supported status flow.

## Implementation

- Go with Chi on top of the standard `net/http` handler model
- PostgreSQL through `database/sql` and the pgx driver
- JWT bearer tokens for authentication
- SQL migrations run at API startup
- Swagger docs generated with `swaggo/swag`

The code is split into HTTP handlers, services, and repositories. Services hold the order/authentication rules and depend on narrow repository interfaces so the business rules can be tested without a database.

## Setup

### Database

Docker Compose starts PostgreSQL and creates the configured database from `DB_NAME` on first startup. The API applies the schema migration when it starts.

### Application

1. Create a local environment file:
   ```bash
   cp .env.example .env
   ```

2. Start PostgreSQL and the API:
   ```bash
   docker compose up --build
   ```

3. In another terminal, seed restaurants and menus:
   ```bash
   docker compose exec api /seed
   ```

The API listens on `http://localhost:8080` by default.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | Yes | - | Secret used to sign JWT tokens |
| `PORT` | No | `8080` | HTTP server port |
| `DATABASE_URL` | No | - | Full PostgreSQL connection string; overrides the individual `DB_*` variables |
| `DB_HOST` | No | `localhost` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | No | `postgres` | Database user |
| `DB_PASSWORD` | No | `postgres` | Database password |
| `DB_NAME` | No | `foodorder` | Database name |
| `DB_SSLMODE` | No | `disable` | PostgreSQL SSL mode |

## API Docs

Swagger UI is served at `http://localhost:8080/swagger/index.html`.

Generated Swagger files are committed under `docs/`. Regenerate them after changing handler annotations with:
```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/api/main.go -o docs
```

## Tests

The default test suite has no external service dependency:
```bash
go test ./...
```

The tagged API integration suite uses a real PostgreSQL database and truncates its tables between tests. For the local Compose database:
```bash
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/foodorder?sslmode=disable \
  go test -tags=integration ./tests/integration/... -v
```

There is also a small smoke script for the running Dockerized API:
```bash
docker compose up --build -d
docker compose exec api /seed
./scripts/smoke.sh
```

## API

### Authentication

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register a new customer |
| POST | `/auth/login` | No | Login and receive a JWT token |
| GET | `/auth/who-am-i` | Yes | Return the authenticated user |

### Restaurants

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

Order statuses move forward only: `received` -> `preparing` -> `ready` -> `delivered`.

## Seed Data

Customer accounts are created through `/auth/register`. Restaurant accounts are seeded so the restaurant-side endpoints can be exercised without adding account-management endpoints to the assignment scope.

After running the seed command, these restaurant credentials are available:

| Email | Password |
|-------|----------|
| `mario@restaurant.com` | `restaurant123` |
| `sakura@restaurant.com` | `restaurant123` |
| `ali@restaurant.com` | `restaurant123` |

## Layout

```
cmd/api/              API entry point and server setup
cmd/seed/             Local seed command for restaurants and menus
internal/
  handler/            Routing, HTTP handlers, request validation
  service/            Authentication and ordering rules
  repository/         SQL data access
  middleware/         JWT authentication and role checks
  model/              API and persistence-facing data types
  config/, database/  Environment config and database connection setup
migrations/           Schema migration
docs/                 Generated Swagger files
```

## Notes

- Authentication uses one `users` table with a `role` column. Customers register through the API; seeded restaurant users log in through the same `/auth/login` endpoint.
- Order totals are calculated on the server from menu prices. Each order item stores `price_at_time` so later menu price changes do not rewrite order history.
- Order creation is transactional: the order and its items are inserted together.
- Restaurant order endpoints derive the restaurant from the authenticated restaurant user, so a restaurant cannot read or update another restaurant's orders.
- Status updates are conditional on the previously read status so concurrent requests cannot overwrite a newer order state with a stale transition.

## Tradeoffs

This implementation focuses on the required ordering and restaurant order-management flows. The real-time status update bonus is intentionally left out so the core API, authorization rules, documentation, and tests stay complete and easy to review.

For a larger production service, the first changes I would make are a non-floating-point money representation and pagination on list endpoints. Migrations run at startup here to keep setup short; in a deployed service I would run them as a separate release step. The request logging is intentionally small for this API, while richer structured logs and trace correlation would become more useful once there are downstream services to follow.
