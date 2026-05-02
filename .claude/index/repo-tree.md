# Repo Tree — trade-buddy

_Auto-reference for agents. Read this before exploring filesystem._
_Last updated: 2026-05-02_

---

## Root

```
trade-buddy/
├── docker-compose.yml      # 3 services: db (postgres:16), api (:8080), ui (:3000)
├── .env.example            # DATABASE_URL, PORT, DATA_DIR
├── AGENTS.md               # AI rules + autoprompt rules
├── CLAUDE.md               # references AGENTS.md
├── docs/
│   ├── plan.md             # ★ source of truth for tasks (check [x] status here)
│   ├── next_go_postgres_migration_plan.md  # architecture + API contracts + Thai labels
│   └── software_requirements.md
├── python/src/trade_buddy/ # Python reference — DO NOT DELETE during migration
│   ├── data/yahoo_finance.py     (121 lines) — Yahoo Finance fetcher
│   ├── analysis/
│   │   ├── engine.py             (69 lines)  — SMA, ATR indicators
│   │   ├── market_structure.py   (96 lines)  — trend, swing detection
│   │   └── top_down.py           (~600 lines) — bias scoring, S/R, weekly plan
│   └── journal/
│       ├── live_chart.py         — SVG weekly plan generator
│       └── store.py              — stats/summary
├── api/                    # Go/Gin/GORM v2 — migration target
│   ├── go.mod              (trade-buddy/api, Go 1.23, gin, gorm)
│   ├── cmd/server/main.go  — entry point, graceful shutdown
│   ├── internal/
│   │   ├── config/config.go      — PORT, DATABASE_URL, DATA_DIR from env
│   │   ├── http/
│   │   │   ├── router.go         — Gin engine, route registration
│   │   │   └── health.go         — GET /health → {"ok":true,"service":"trade-buddy-api"}
│   │   └── db/                   — (T2.2) GORM connect + migrate
│   └── migrations/               — (T2.1) explicit SQL files, run on startup
├── ui/                     # Next.js App Router, TypeScript, Tailwind, dark theme
│   ├── src/app/
│   │   ├── layout.tsx      — dark bg-gray-950
│   │   └── page.tsx        — dashboard page
│   └── next.config.ts      — output: 'standalone'
├── data/
│   ├── postgres/           # docker volume mount (gitignored)
│   └── weekly-plan-maps/   # SVG cache (gitignored)
└── .claude/
    ├── active.md           # ★ task queue + agent status
    ├── orchestrator.md     # orchestration rules for Claude
    ├── index/repo-tree.md  # this file
    ├── tasks/              # agent result files (TXX-result.md)
    ├── topics/             # research notes (shared across agents)
    ├── sessions/           # session checkpoints
    └── private/            # gitignored personal notes
```

---

## Key API Contracts

### GET /health
```json
{"ok": true, "service": "trade-buddy-api"}
// Phase 2+: {"ok": true, "db": "ok", "service": "trade-buddy-api"}
```

### GET /api/chart?symbol=XAUUSD&tf=1h&source=yahoo&limit=200
```json
{
  "symbol": "XAUUSD", "timeframe": "1h", "source": "yahoo",
  "candles": [{"time": 1234567890, "open": 1800, "high": 1820, "low": 1795, "close": 1810, "volume": 1000}],
  "levels": [], "markers": [], "overlays": []
}
```

### GET /api/weekly-plan
```json
{
  "symbol": "XAUUSD",
  "forecast_bias": "long",
  "levels": {"S1": 2300, "S2": 2250, "R1": 2380, "R2": 2420},
  "paths": [{"direction": "long", "type": "primary", "points": []}],
  "swing_trade": {"direction": "long", "action": "รอ Buy", "entry_zone": [2310, 2320], "sl": 2290, "tp": 2370, "tp2": 2410, "rr": 3.0},
  "image_url": "/weekly-plan-maps/<hash>.svg"
}
```

---

## Tech Decisions (quick ref)

- Go router: **Gin** (not stdlib)
- ORM: **GORM v2** with explicit SQL migrations (no AutoMigrate as primary)
- DB: **Postgres 16** via docker volume `./data/postgres`
- Migrations: `api/migrations/*.sql` embedded + run on startup
- Context: `context.Context` everywhere in Go
- Parallel TF loads: `errgroup`
- Next.js: App Router, no large UI framework, dark theme default
- Thai text: UI labels only, never in API JSON field names

---

## Env Vars

| Var          | Default                                         | Used by    |
|--------------|-------------------------------------------------|------------|
| PORT         | 8080                                            | api        |
| DATABASE_URL | postgres://tradebuddy:secret@db:5432/tradebuddy | api        |
| DATA_DIR     | /data                                           | api        |
| NEXT_PUBLIC_API_URL | http://localhost:8080                    | ui         |
