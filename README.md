# Splitwise Backend (Go)

A backend service for expense sharing and group expense management — users, groups, expenses, settlements, and balance tracking, with JWT authentication and a SQLite datastore.

## Features

- **JWT authentication** — register/login/refresh with 30-minute access tokens and 7-day refresh tokens; bcrypt password hashing
- **Users** — full CRUD with soft-delete/deactivate support
- **Groups** — create groups, add/remove members
- **Categories** — per-group expense categories
- **Expenses** — split four ways: `EQUAL`, `EXACT`, `PERCENTAGE`, `SHARES`
- **Settlements** — record direct payments between group members
- **Balances** — per-group and personal ("me") balance views, with debt simplification to minimize the number of repayments

## Tech Stack

- **Go 1.27** with the standard library `net/http` router (method-based patterns, no web framework)
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO required)
- **JWT** via `github.com/golang-jwt/jwt/v5`
- **bcrypt** via `golang.org/x/crypto`

## Getting Started

### Prerequisites

- Go 1.27+

### Run

```bash
go run ./cmd/server
```

The server starts on `http://localhost:8080`. The SQLite database and schema (tables + indexes) are created automatically on startup.

### Configuration

| Env variable | Default | Description |
|--------------|---------|-------------|
| `DB_PATH` | `./splitwise.db` | Path to the SQLite database file |
| `JWT_SECRET` | `dev-secret-change-me` (with startup warning) | Secret used to sign JWTs — **set this in production** |

Example:

```bash
JWT_SECRET=$(openssl rand -hex 32) DB_PATH=./splitwise.db go run ./cmd/server
```

## Project Layout

```
cmd/server/          # entry point, router setup, DB bootstrap
internal/
  user/              # users + auth handlers/services/repositories
  group/             # groups and group membership
  category/          # expense categories
  expense/           # expenses and split computation
  settlement/        # member-to-member payments
  balance/           # balance computation and debt simplification
  pkg/
    auth/            # JWT manager + auth middleware
    split/           # split types and share calculation
    response/        # JSON response helpers
    util/            # shared utilities
```

Each domain module follows a `handler` → `service` → `repository` layering, with plain `database/sql` queries against SQLite.

## API

Full endpoint reference with request/response examples: [API_DOCUMENTATION.md](API_DOCUMENTATION.md)

Quick overview:

| Area | Endpoints |
|------|-----------|
| Auth | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me` |
| Users | `GET/POST /users`, `GET/PUT/DELETE /users/{id}` |
| Groups | `GET/POST /groups`, `GET/PUT/DELETE /groups/{id}`, `POST /groups/{group_id}/members`, `DELETE /groups/{group_id}/members/{user_id}` |
| Categories | `POST /categories`, `GET /categories/{group_id}`, `GET/PUT/DELETE /categories/{group_id}/{id}` |
| Expenses | `POST /expenses`, `GET /expenses/group/{group_id}`, `GET/PUT/DELETE /expenses/{id}` |
| Settlements | `POST /settlements`, `GET /settlements/group/{group_id}`, `DELETE /settlements/{id}` |
| Balances | `GET /balances/group/{group_id}`, `GET /balances/me` |
| Misc | `GET /health` |

All protected routes require `Authorization: Bearer <access_token>`.

## Health Check

```bash
curl http://localhost:8080/health
```
