# Trade Buddy

Local-first trading analysis and journaling toolkit for gold/XAUUSD setups.

The primary app is now the Dockerized Go/Next.js stack:

- Go/Gin API in `api/` with explicit SQL migrations and Postgres persistence
- Next.js dashboard in `ui/` with TradingView Lightweight Charts
- Yahoo Finance market data for XAUUSD/GC=F candles
- weekly forecast JSON plus cached SVG plan maps
- saved alerts, alert outcomes, and learning stats in Postgres
- Python source in `python/` retained as the behavior reference

## Quick Start

From the repo root:

```bash
docker compose up --build
```

Then open:

```text
http://localhost:3000
```

Useful API checks:

```bash
curl http://localhost:8080/health
curl 'http://localhost:8080/api/chart?symbol=XAUUSD&tf=1h&source=yahoo&limit=200'
curl http://localhost:8080/api/weekly-plan
curl http://localhost:8080/api/journal/stats
```

The original Python app remains available under `python/`. From the repo root,
run Python commands with `PYTHONPATH=python/src`, or install the package with
`python3 -m pip install -e python`.

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli analyze XAUUSD --tf 15m --csv examples/xauusd_15m.csv
PYTHONPATH=python/src python3 -m trade_buddy.cli evaluate XAUUSD --tf 15m --csv examples/xauusd_15m.csv --history-bars 21 --spread 0.2 --slippage 0.1
PYTHONPATH=python/src python3 -m trade_buddy.cli backtest XAUUSD --tf 15m --csv examples/xauusd_15m.csv --spread 0.2 --slippage 0.1
PYTHONPATH=python/src python3 -m trade_buddy.cli backtest XAUUSD --tf 15m --csv examples/xauusd_15m.csv --save
PYTHONPATH=python/src python3 -m trade_buddy.cli bulk-backtest --datasets XAUUSD:15m:examples/xauusd_15m.csv --save
PYTHONPATH=python/src python3 -m trade_buddy.cli optimize XAUUSD --tf 15m --csv examples/xauusd_15m.csv
PYTHONPATH=python/src python3 -m trade_buddy.cli chart XAUUSD --tf 15m --csv examples/xauusd_15m.csv --output reports/chart.html
PYTHONPATH=python/src python3 -m trade_buddy.cli serve-chart XAUUSD --tf 15m --source yahoo --refresh-seconds 10
PYTHONPATH=python/src python3 -m trade_buddy.cli serve
PYTHONPATH=python/src python3 -m trade_buddy.cli backfill XAUUSD --tf 15m --csv examples/xauusd_15m.csv
PYTHONPATH=python/src python3 -m trade_buddy.cli review --last 10
PYTHONPATH=python/src python3 -m trade_buddy.cli explain <setup-id>
PYTHONPATH=python/src python3 -m trade_buddy.cli insights --group-by pattern
PYTHONPATH=python/src python3 -m trade_buddy.cli recommend --min-count 3
PYTHONPATH=python/src python3 -m trade_buddy.cli outcome <setup-id> --result win --r 1.8 --notes "Clean breakout follow-through"
```

TradingView MCP can be connected through a command bridge until a direct MCP tool
is available in the runtime:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli analyze XAUUSD --tf 15m --source tradingview --mcp-command ./scripts/tradingview_bridge
```

The bridge receives one JSON argument:

```json
{"symbol": "XAUUSD", "timeframe": "15m", "limit": 200}
```

It should print JSON containing candle rows under a key such as `candles`,
`ohlcv`, `bars`, `data`, or `result`.

See [docs/tradingview_mcp.md](docs/tradingview_mcp.md) for the full data contract.

For local package usage:

```bash
python3 -m pip install -e python
trade-buddy analyze XAUUSD --tf 15m --csv examples/xauusd_15m.csv
```

## Local Workflow

Create a signal from local CSV data:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli alert XAUUSD --tf 15m --csv examples/xauusd_15m.csv --min-confidence 0.5
```

Resolve open signals from newer local candles:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli resolve XAUUSD --tf 15m --csv examples/xauusd_15m.csv
```

Backfill any unresolved setup without an outcome:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli backfill XAUUSD --tf 15m --csv examples/xauusd_15m.csv --max-bars 20
```

Save historical backtest trades into the journal so `insights` and `playbook`
can learn from them:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli backtest XAUUSD --tf 15m --csv examples/xauusd_15m.csv --save --spread 0.2 --slippage 0.1
```

Backtest output includes win rate, average R, cumulative R, max drawdown in R,
and profit factor.

Run several historical CSV datasets as one portfolio-style test:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli bulk-backtest \
  --datasets XAUUSD:15m:examples/xauusd_15m.csv \
  --history-bars 21 \
  --max-bars 20 \
  --save \
  --export reports/bulk-backtest.json \
  --format json
```

Grid-search confidence and hold-window settings:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli optimize XAUUSD \
  --tf 15m \
  --csv examples/xauusd_15m.csv \
  --confidence-grid 0.0,0.55,0.65,0.75,0.85 \
  --max-bars-grid 10,20,30 \
  --export reports/optimization.csv
```

Generate a local dashboard:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli dashboard --output reports/dashboard.html
```

Generate a standalone local chart page:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli chart XAUUSD --tf 15m --csv examples/xauusd_15m.csv --output reports/chart.html
```

Serve an auto-refreshing local chart. This polls the selected data source from
the local machine and updates the browser every few seconds. `serve-web` also
shows alert status, saves active alerts to SQLite, marks active alerts on the
chart, and lets you click saved alerts to review the original snapshot. The live
chart uses TradingView Lightweight Charts in the browser for pan, zoom,
crosshair, markers, price lines, and timeframe tabs:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli serve
```

Then open:

```text
http://127.0.0.1:8765
```

Alert history is stored in the local journal database configured by `--db`.
Each saved alert keeps the setup reasoning and market snapshot so the web UI can
draw the SVG chart from the moment the alert was created.

The web page also exposes common CLI workflows as buttons:

- Select symbol, source, CSV path, and timeframe from the browser
- Load a larger candle history via the Bars setting; web default is 1000 bars
- Apply runtime config for alert threshold and refresh interval
- Resolve open alerts for the selected timeframe
- Run strategy recommendations from completed outcomes
- Review a multi-timeframe alert board for 15m, 30m, 1h, and 1d

Run the daily local artifact bundle:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli daily --output-dir reports --backup-dir backups
```

Export learning summaries:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli insights --group-by setup_type --export reports/insights.csv
PYTHONPATH=python/src python3 -m trade_buddy.cli playbook --min-count 3 --export reports/playbook.json --format json
PYTHONPATH=python/src python3 -m trade_buddy.cli playbook --min-count 3 --export-rules config/learned-rules.json
PYTHONPATH=python/src python3 -m trade_buddy.cli recommend --min-count 3
```

## Next Steps

- Keep Python as the behavior reference while hardening the Go/Next.js stack.
- Add richer pattern detection and more explicit strategy explanations.
- Add bulk historical dataset import for multi-month backfills.
- Re-enable third-party integrations when credentials/runtime are ready.

## Data Sources

Current sources:

- `csv`: local OHLCV CSV files
- `yahoo`: live Yahoo Finance chart API via the built-in adapter
- `tradingview`: command bridge for MCP/client integrations

Gold aliases:

- `XAUUSD` -> `GC=F`
- `GOLD` -> `GC=F`

See [docs/atila_tradingview_mcp.md](docs/atila_tradingview_mcp.md) for notes on
the `atilaahmettaner/tradingview-mcp` server.

## LINE Alerts

LINE Notify is no longer available. Trade Buddy uses the LINE Messaging API.

Set these environment variables:

```bash
export LINE_CHANNEL_ACCESS_TOKEN="..."
export LINE_TO_ID="..."
```

Then send alerts from signal lifecycle commands:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor \
  --symbols XAUUSD \
  --timeframes 15m,1h \
  --source yahoo \
  --exclude-open-candle \
  --notify line
```

Or run from a JSON config:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli run --config config/trade-buddy.example.json
```

Use the journal playbook to calibrate alert confidence:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor \
  --symbols XAUUSD \
  --timeframes 15m,1h \
  --source yahoo \
  --exclude-open-candle \
  --use-playbook \
  --playbook-min-count 5
```

Use a strategy profile:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor \
  --symbols XAUUSD \
  --source yahoo \
  --exclude-open-candle \
  --profile conservative
```

Manual strategy rules example:
[strategy-rules.example.json](config/strategy-rules.example.json)

Resolve ambiguous TP/SL hits with lower timeframe candles:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli monitor \
  --symbols XAUUSD \
  --timeframes 15m,1h \
  --source yahoo \
  --exclude-open-candle \
  --lower-timeframe 1m
```

Check setup before running:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli doctor --config config/trade-buddy.example.json --check-market --check-line
```

Test LINE delivery:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli notify-test --message "Trade Buddy test"
```

Operational runbook: [docs/operations.md](docs/operations.md)

Generate a local dashboard:

```bash
PYTHONPATH=python/src python3 -m trade_buddy.cli dashboard --output reports/dashboard.html
```
# trade-buddy
# trade-buddy
