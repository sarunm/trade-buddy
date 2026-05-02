# Trade Buddy — AI Rules

## Project Context

Trade Buddy is a local-first trading assistant for XAUUSD/gold.
- Current stack: Python CLI + local web dashboard (TradingView Lightweight Charts, SQLite)
- Migration target: Next.js (`ui/`) + Go/Gin/GORM v2 (`api/`) + Postgres + Docker Compose
- Python source in `python/src/` is the behavior reference — do not delete it during migration
- Primary command after migration: `docker compose up --build`

Key docs:
- `docs/plan.md` — current task checklist and phase status (source of truth for what to do next)
- `docs/next_go_postgres_migration_plan.md` — architecture decisions and API contracts
- `docs/software_requirements.md` — feature requirements
- `agent.md` — current Python implementation notes

## Autoprompt Rules

These rules apply automatically. Do not ask the user before following them.

### Rule 1 — Done → Continue

When a task is completed:
1. Mark it `[x]` in `docs/plan.md`
2. Read the next unchecked `[ ]` task in the current phase
3. Start it immediately
4. If the current phase is fully checked, confirm the phase gate (see `docs/plan.md`), then move to the next phase

Do not stop and ask "what should I do next?" — check the plan.

### Rule 2 — Stuck → Research First

When stuck on a technical problem (build error, library API, Go/Next.js pattern):
1. Use WebSearch to find the specific error or pattern
2. Try the solution
3. If still stuck after 2 attempts, summarize what was tried and ask the user

When stuck on a UX/feature decision (what should traders see, how to display data):
1. Use WebSearch with queries like:
   - `"trading dashboard UX" what traders want to see gold`
   - `TradingView chart UI best practices`
   - `"price action dashboard" trader workflow`
   - `XAUUSD swing trade checklist trader`
2. Apply findings directly — do not ask for approval on UX decisions unless they change scope

### Rule 3 — Phase Gate Before Moving On

Before starting the next phase, confirm the gate passes:
- Phase 1 gate: `docker compose up --build` starts all 3 services (ui, api, db)
- Phase 2 gate: `GET /health` returns `{"ok":true}` AND API connects to Postgres
- Phase 3 gate: `GET /api/chart` returns real XAUUSD candles
- Phase 4 gate: `GET /api/weekly-plan` returns forecast JSON + cached SVG path
- Phase 5 gate: browser at `http://localhost:3000` shows chart + weekly plan from Go API
- Phase 6 gate: saved alert survives `docker compose down && docker compose up`

If a gate fails, fix it before marking the phase done.

### Rule 4 — Scope Control

- Do not refactor Python code during migration
- Do not add features not in the plan
- Do not change the plan's phase order without user approval
- Keep commits scoped to one phase at a time

### Rule 5 — Thai UX Text

The UI uses Thai for trader-facing labels. When generating UI text:
- Use simple, direct Thai phrasing (see `docs/next_go_postgres_migration_plan.md` section 12)
- Do not translate Thai wording to English in the UI
- Numeric values (entry, SL, TP, RR) must stay as separate fields — never embed them in text only

## Code Conventions

- Go: Gin for HTTP, GORM v2 for DB, `internal/` packages, `context.Context` everywhere, `errgroup` for parallel TF loads
- DB schema source of truth: explicit SQL migrations in `api/migrations`; do not use GORM AutoMigrate as the primary migration mechanism
- Next.js: App Router, TypeScript, dark theme default, no large UI framework unless needed
- API responses: flat JSON, no Thai text in machine-readable fields
- Docker: map `./data/postgres` and `./data/weekly-plan-maps` as local volumes

## Current Phase

See `docs/plan.md` for current phase and next unchecked task.
