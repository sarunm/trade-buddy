# Agent Notes

This file is project context, not orchestration state. For agent rules, read
`AGENTS.md`. For orchestration state, use `.agents/`.

## Project Summary
Trade Buddy is a local trading analysis and journaling app for XAUUSD/gold.
The primary implementation is now a Dockerized Go/Next.js/Postgres stack, with
the original Python CLI/web app kept under `python/` as the behavior reference.

## Current State
- Main entrypoint: `docker compose up --build`
- Default data source: `yahoo`
- Default symbol/timeframe in web UI: `XAUUSD` / `1h`
- Go API runs on `localhost:8080`
- Next.js UI runs on `localhost:3000`
- Postgres stores market bars, alerts, alert outcomes, weekly plans, signal events, and app config
- Web chart uses TradingView Lightweight Charts in the browser
- Weekly plan returns forecast JSON plus a cached SVG plan map
- Saved alerts can be listed and opened in a modal
- Journal stats report total outcomes, win rate, average RR, and grouped stats

## Performance
- Market bars are persisted in Postgres and reused by `/api/chart`
- Weekly SVG files are cached under `data/weekly-plan-maps`
- Weekly plan loads weekly and monthly candles concurrently in Go

## Important Behavior
- `loss` means stop loss was hit first
- `win` means take profit was hit first
- `ambiguous` means one candle touched both TP and SL
- `timeout` means neither side was hit before expiry
- The Python source remains the reference for behavior that has not yet been
  ported or hardened in Go

## Implemented UI
- Dark local dashboard
- Timeframe tabs for chart switching
- Multi-TF trend board
- Weekly plan SVG with forecast route cards
- Swing trade bias with Thai trader-facing labels
- Saved alert history with modal detail view
- Learning stats cards

## Key Files
- `docker-compose.yml`
- `api/cmd/server/main.go`
- `api/internal/http/router.go`
- `api/internal/http/chart.go`
- `api/internal/http/weekly_plan.go`
- `api/internal/http/alerts.go`
- `api/internal/http/journal.go`
- `api/internal/db/`
- `api/internal/analysis/`
- `api/internal/forecast/`
- `ui/app/page.tsx`
- `ui/components/`
- `python/src/trade_buddy/cli.py`
- `python/src/trade_buddy/journal/live_chart.py`
- `python/src/trade_buddy/data/yahoo_finance.py`
- `python/src/trade_buddy/data/tradingview_mcp.py`

## Data Sources
- `csv`: local OHLCV files
- `yahoo`: live Yahoo Finance candles
- `tradingview`: command bridge for TradingView MCP-style data

## Important Constraints
- SQL migrations in `api/migrations` are the schema source of truth
- Python source must not be deleted during migration
- Trader-facing UI labels should use simple Thai
- Machine-readable API fields should stay flat and should not contain Thai text
- Numeric values such as entry, SL, TP, and RR should stay as separate fields
- Yahoo interval support is limited; `4h` can be unavailable depending on source
- `1m` history may be short depending on source limits
- Do not assume TradingView MCP is directly available as a native tool; the
  project currently uses a bridge / adapter boundary

## Useful Commands
```bash
docker compose up --build
curl http://localhost:8080/health
curl 'http://localhost:8080/api/chart?symbol=XAUUSD&tf=1h&source=yahoo&limit=200'
curl http://localhost:8080/api/weekly-plan
curl http://localhost:8080/api/journal/stats
cd api && go test ./...
cd ui && npm run build
PYTHONPATH=python/src python3 -m trade_buddy.cli serve
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor --symbols XAUUSD --timeframes 15m,1h --source yahoo --exclude-open-candle --lower-timeframe 1m
PYTHONPATH=python/src python3 -m unittest discover -s python/tests
PYTHONPYCACHEPREFIX=/private/tmp/trade_buddy_pycache PYTHONPATH=python/src python3 -m compileall python/src
```

## Verification
- `cd api && go test ./...` passes
- `cd ui && npm run build` passes
- `docker compose up --build` starts db, api, and ui
- Phase 8 parity check passed for XAUUSD 1h candle timestamps/OHLCV and weekly forecast direction

## Next Steps
1. Keep hardening Go/Next.js behavior against the Python reference.
2. Improve chart usability if more overlays or alert drill-down are needed.
3. Revisit TradingView integration once direct runtime support is available.
