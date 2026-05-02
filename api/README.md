# Trade Buddy API

Go backend for the Trade Buddy migration.

Planned stack:

- Gin for HTTP routing and middleware
- GORM v2 for Postgres access
- explicit SQL migration files in `api/migrations`

## Structure

```text
api/
  cmd/server/          # Application entrypoint
  internal/config/     # Environment config and defaults
  internal/http/       # Gin router, handlers, response helpers
  internal/db/         # GORM connection, migrations, repositories
```

Implementation rules:

- Keep route wiring explicit in `internal/http`
- Keep GORM behind repositories, not scattered through handlers
- Use SQL migration files as the schema source of truth
- Do not use GORM `AutoMigrate` as the main migration flow
- Keep graceful shutdown explicit in `cmd/server`

## Run

```bash
go run ./cmd/server
```

Environment variables:

- `PORT`, default `8080`
- `DATABASE_URL`, optional until Postgres is wired
- `DATA_DIR`, default `./data`

Health check:

```bash
curl http://localhost:8080/health
```
