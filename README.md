<div align="center">
  <h1>shrt</h1>
  <img src="ui/static/img/hero.webp" width="720" alt="shrt" />
  <p>A URL shortening service with analytics, authentication, and a server-rendered dashboard.</p>
</div>

---

## How it works

**shrt** takes long URLs and turns them into short, shareable links. Users can create links anonymously or sign up to manage them from a dashboard with click analytics, top referers, and expiration controls.

The service exposes both a REST API and a server-rendered web UI built with Go templates and HTMX.

### Features

- Shorten URLs with auto-generated or custom short codes
- Click tracking with daily breakdown and top referers
- Link expiration with automatic invalidation
- User dashboard with CRUD operations
- Local auth (email/password) and OAuth (Google, GitHub)
- Redis cache for fast redirects

## Tech stack

| Layer     | Technology                   |
| --------- | ---------------------------- |
| Language  | Go 1.25                     |
| Router    | chi                          |
| Database  | PostgreSQL 17                |
| Cache     | Redis 7                      |
| Auth      | JWT (access + refresh tokens)|
| OAuth     | Google, GitHub               |
| Frontend  | Go templates + HTMX          |
| Migrations| golang-migrate               |

## Getting started

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### Setup

**1. Clone and configure:**

```bash
git clone https://github.com/juanpeyrot/shrt.git
cd shrt
cp .env.example .env
# Fill in the .env values (DB credentials, JWT secret, OAuth keys, etc.)
```

**2. Start the infrastructure:**

```bash
docker compose up -d
```

This starts PostgreSQL and Redis.

**3. Run migrations:**

```bash
make migrate-up
```

**4. Start the server:**

```bash
make run
```

The app will be available at `http://localhost:3000`.

### Available commands

```bash
make run             # Start the server
make build           # Build the project
make test            # Run tests
make migrate-up      # Apply all pending migrations
make migrate-down    # Roll back the last migration
make migrate-create name=<name>  # Create a new migration
```

---

## Architecture

```
                    ┌──────────────────────┐
                    │       Client         │
                    │  (Browser / cURL)    │
                    └──────────┬───────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │     Go HTTP Server   │
                    │       (chi)          │
                    ├──────────┬───────────┤
                    │  API     │  Web UI   │
                    │ /api/*   │  / + HTMX │
                    └────┬─────┴─────┬─────┘
                         │           │
              ┌──────────┤           │
              │          │           │
              ▼          ▼           ▼
       ┌───────────┐  ┌───────┐  ┌───────────────┐
       │   Auth    │  │ Link  │  │  Middleware    │
       │ Service   │  │Service│  │ (JWT, OAuth)   │
       └─────┬─────┘  └───┬───┘  └───────────────┘
             │             │
             │         ┌───┴────────┐
             │         │            │
             ▼         ▼            ▼
       ┌──────────┐  ┌──────┐  ┌────────┐
       │PostgreSQL│  │ Redis│  │PostgreSQL│
       │ (users,  │  │(link │  │ (links, │
       │  auth)   │  │cache)│  │ clicks) │
       └──────────┘  └──────┘  └─────────┘
```

### Request flow for redirects

1. `GET /{shortCode}` hits the web handler
2. The service checks **Redis** first (`link:<shortCode>`)
3. On cache miss, it queries **PostgreSQL** and populates the cache (TTL: 10 min)
4. A click is recorded asynchronously in a goroutine
5. The user is redirected via `302 Found`

---

## Design decisions

### Short code generation

Codes are **7 characters** drawn from a **base-62 alphabet** (`0-9`, `a-z`, `A-Z`), yielding **62⁷ ≈ 3.5 trillion** possible combinations.

Generation uses `crypto/rand` for cryptographic randomness. To avoid **modulo bias**, a rejection sampling technique is used: random bytes ≥ 248 are discarded so that `byte % 62` maps uniformly across all 62 characters (`248 / 62 = 4`, exact division).

### Collision handling

Short codes have a `UNIQUE` constraint in PostgreSQL. When an auto-generated code collides, the service **retries up to 5 times** with a freshly generated code. At the current keyspace size (3.5T codes), collisions are extremely unlikely until billions of links exist.

Custom (user-provided) codes return a `409 Conflict` immediately on collision with no retry.

### Caching strategy

Redis is used as a **read-through cache** for the redirect hot path:

- On **cache hit**: return the original URL directly, skip the database
- On **cache miss**: query PostgreSQL, populate the cache with a **10-minute TTL**
- On **update/delete**: the cache entry is **invalidated** to prevent stale redirects
- On **cache failure**: the service **falls back to the database** gracefully

Expired links are checked at the application level even when served from cache.

### Soft deletes

Links are never physically removed from the database. A `deleted_at` timestamp is set instead, which all queries filter on. This preserves click history for analytics.

### Click tracking

Each redirect records a click in the `link_clicks` table with the referer header and timestamp. The `click_count` column on `links` is also incremented atomically in the same transaction. Both writes happen **asynchronously** in a goroutine to avoid adding latency to the redirect response.

### Authentication

The service supports **multi-provider authentication**:

- **Local**: email/password with bcrypt hashing
- **OAuth**: Google and GitHub via standard OAuth2 flows

A user can have multiple auth methods linked to the same account. If an OAuth provider returns a verified email that matches an existing user, the new auth method is automatically linked.

JWT tokens are issued as access + refresh pairs.

---

## ERD

```
┌───────────────────────┐       ┌──────────────────────────────┐
│         users         │       │        auth_methods           │
├───────────────────────┤       ├──────────────────────────────┤
│ id          UUID  PK  │◄──┐  │ id                UUID  PK   │
│ display_name TEXT     │   └──│ user_id            UUID  FK   │
│ email        TEXT  UQ │      │ provider           TEXT       │
│ created_at   TIMESTAMPTZ│    │ provider_user_id   TEXT       │
└───────────┬───────────┘      │ password_hash      TEXT       │
            │                  │ refresh_token_hash TEXT       │
            │                  │ refresh_token_jwt_id TEXT     │
            │                  │ created_at  TIMESTAMPTZ      │
            │                  │ updated_at  TIMESTAMPTZ      │
            │                  │ UNIQUE(user_id, provider)    │
            │                  └──────────────────────────────┘
            │
            │ 1:N (optional)
            ▼
┌───────────────────────────┐       ┌──────────────────────────┐
│          links            │       │       link_clicks         │
├───────────────────────────┤       ├──────────────────────────┤
│ id           UUID  PK     │◄──┐  │ id         UUID  PK      │
│ user_id      UUID  FK     │   └──│ link_id    UUID  FK      │
│ short_code   TEXT  UQ     │      │ referer    TEXT           │
│ original_url TEXT         │      │ clicked_at TIMESTAMPTZ   │
│ created_at   TIMESTAMPTZ  │      └──────────────────────────┘
│ expires_at   TIMESTAMPTZ  │
│ deleted_at   TIMESTAMPTZ  │
│ click_count  BIGINT       │
└───────────────────────────┘
```

---

## ⭐ Give it a star

If you found this project useful, consider giving it a star on GitHub — it helps a lot!

## Based on

This project was built following the [URL Shortening Service](https://roadmap.sh/projects/url-shortening-service) project from [roadmap.sh](https://roadmap.sh), extended with authentication, OAuth, a web UI, caching, and click analytics.

## License

MIT — see [LICENSE](LICENSE) for details.
