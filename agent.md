# Agent Notes

This file is project context, not orchestration state. For agent rules, read
`AGENTS.md`. For orchestration state, use `.agents/`.

## Project Summary
Trade Buddy is a local trading analysis and journaling app for XAUUSD/gold.
It started CLI-first and now includes a local web chart, SQLite journal, alert
history modal, and simple learning loops from resolved setups.

## Current State
- Main entrypoint: `trade-buddy serve`
- Default data source: `yahoo`
- Default symbol/timeframe in web UI: `XAUUSD` / `15m`
- SQLite journal stores setups, outcomes, open alerts, snapshots, and notes
- Web chart uses TradingView Lightweight Charts in the browser
- Alerts can be resolved from the web UI and saved alerts can be opened in a modal
- Ambiguous TP/SL hits can be resolved using a lower timeframe

## Performance
- Bar data is cached in-memory per (symbol, timeframe, source, limit) with a 30s TTL
- `/api/multiframe` and `/api/topdown` fetch timeframes in parallel via `ThreadPoolExecutor`
- Frontend calls `loadMultiTf()`, `loadTopDown()`, and `loadAlertHistory()` in parallel via `Promise.all`

## Important Behavior
- `loss` means stop loss was hit first
- `win` means take profit was hit first
- `ambiguous` means one candle touched both TP and SL
- `timeout` means neither side was hit before expiry
- Lower-timeframe resolution should use the full original expiry window, not the
  lower TF's own bar count

## Implemented UI
- Dark local dashboard
- Timeframe tabs including `1w` and `1mo`
- Settings panel behind the gear icon
- Toggle buttons for FIB and support/resistance
- Multi-TF alert board
- Saved alert history with modal detail view
- Resolution TF selector in Settings

## Key Files
- `python/src/trade_buddy/cli.py`
- `python/src/trade_buddy/journal/live_chart.py`
- `python/src/trade_buddy/journal/signals.py`
- `python/src/trade_buddy/journal/store.py`
- `python/src/trade_buddy/data/source.py`
- `python/src/trade_buddy/data/yahoo_finance.py`
- `python/src/trade_buddy/data/tradingview_mcp.py`

## Data Sources
- `csv`: local OHLCV files
- `yahoo`: live Yahoo Finance candles
- `tradingview`: command bridge for TradingView MCP-style data

## Important Constraints
- Yahoo interval support is limited; `4h` is intentionally disabled in the web
  UI for now
- `1m` history may be short depending on source limits
- Do not assume TradingView MCP is directly available as a native tool; the
  project currently uses a bridge / adapter boundary

## Useful Commands
```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli serve
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor --symbols XAUUSD --timeframes 15m,1h --source yahoo --exclude-open-candle --lower-timeframe 1m
PYTHONPATH=python/src python3 -m unittest discover -s python/tests
PYTHONPYCACHEPREFIX=/private/tmp/trade_buddy_pycache PYTHONPATH=python/src python3 -m compileall python/src
```

## Verification
- Last known test status: `88 tests OK`
- `compileall` passed on `python/src`

## Next Steps
1. Keep refining alert resolution and traceback quality.
2. Add better learning from wins/losses in SQLite.
3. Improve chart usability if more overlay toggles or alert drill-down is needed.
