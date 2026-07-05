# Identity Service

Go-based identity service that owns **authentication (login/sessions)**, **feature flags**, and the **audit log** for the financer ecosystem. It also serves an htmx-powered admin web UI for managing users and flags.

## Features

- **Login / sessions**: cookie-based sessions stored in Postgres, bcrypt password hashing, 30-day sliding expiration (configurable)
- **Session validation for other services**: `POST /api/v1/auth/validate` with `X-Session-ID` header — used by the BFF to authenticate requests
- **Feature flags**: global flags with per-user overrides, public `GET /api/v1/feature-flags/check` for service-to-service checks
- **Audit log**: every auth and flag action is written to an `audit_logs` table and logged as structured JSON (slog)
- **Admin web UI** (`/admin`): user CRUD, set password, force-logout ("log people out" button), flag management, audit log viewer
- **Migrations**: embedded SQL files applied automatically on boot (same pattern as the transactions service)

## Architecture

```
financer (Next.js) ──► bff (Express) ──► identity (this service)
                          │                  ▲
                          └──► transactions  │ session validation,
                                             │ feature flags, audit
```

- The frontend never talks to identity directly; the BFF proxies `/api/bff/auth/*` and validates the `session_id` cookie against `POST /api/v1/auth/validate` for every protected route.
- Identity lives on the shared docker network (`bff_back-end` in prod, `staging_bff_back-end` in staging). Only the admin UI port is meant to be reached from the host.

## API

Public (within the docker network):

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/auth/login` | Login (`{email, password}`), sets `session_id` cookie, returns `session_id` in body |
| POST | `/api/v1/auth/logout` | Invalidate session (cookie or `X-Session-ID`) |
| GET | `/api/v1/auth/me` | Current user (cookie or `X-Session-ID`) |
| POST | `/api/v1/auth/validate` | Validate a session (`X-Session-ID` header or JSON body), returns the user |
| GET | `/api/v1/feature-flags/check?key=&user_id=` | Is a flag enabled (globally or for a user)? |

Protected (require a valid session via cookie or `X-Session-ID`): `/api/v1/users*` CRUD + per-user flag assignment, `/api/v1/feature-flags` CRUD.

There is **no public registration endpoint** — users are created via the admin UI (or seeded, see below).

## Configuration

All configuration is via environment variables (see `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | HTTP port |
| `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/DB_SSLMODE` | — | Postgres connection |
| `LOG_LEVEL` | `info` | slog level |
| `SESSION_DURATION_HOURS` | `720` | Session lifetime (sliding: each validation pushes expiry forward) |
| `COOKIE_SECURE` | `false` | Set `true` when behind HTTPS |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | — | First-boot admin seed: created only when the users table is empty |

## Deployment

Two compose files, following the fleet convention:

- `docker-compose.yml` (**prod**) — container names default to `identity` / `identity-postgres`, joins external network `bff_back-end`, service port default `8083`
- `docker-compose.staging.yml` (**staging**) — container names default to `identity-staging` / `identity-postgres-staging`, joins external network `staging_bff_back-end`, service port default `9083`, separate `postgres_data_staging` volume

Both read an uncommitted `stack.env` for secrets/overrides. Required keys at deploy time:

```
POSTGRES_CONTAINER_NAME, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, POSTGRES_PORT
SERVICE_CONTAINER_NAME, SERVICE_PORT
DB_HOST (=postgres service name), DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
ADMIN_EMAIL, ADMIN_PASSWORD   # first boot only
SESSION_DURATION_HOURS, COOKIE_SECURE, LOG_LEVEL   # optional
```

The BFF needs `IDENTITY_SERVICE_URL=http://identity:8080/api/v1` (prod) or `http://identity-staging:8080/api/v1` (staging) in its own stack.env.

```bash
make docker-up      # prod compose
make staging-up     # staging compose
make docker-logs / staging-logs / docker-down / staging-down
```

## Development

```bash
make deps       # install dependencies
make run        # run locally (needs a Postgres, see db-start)
make test       # run tests
make build      # build binary
```

Migrations live in `internal/migrations/*.sql`, are embedded into the binary, and run automatically on boot (tracked in `schema_migrations`). Create a new one with `make migrate-create NAME=my_change`.

## Admin UI

Browse to `http://<host>:<SERVICE_PORT>/admin`, log in with an admin account. Tabs:

- **Feature Flags** — create/toggle/delete global flags
- **Users** — create/edit/delete users, set passwords, manage per-user flags, and **Log out** (kills all of a user's sessions)
- **Audit Log** — latest auth/flag events (also available in `audit_logs` table and container logs)
