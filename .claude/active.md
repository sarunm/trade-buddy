# Active Task Queue

_Orchestrator: Claude | Updated: 2026-05-02 | Project: trade-buddy_

---

## Current Phase: 3 — Market Data

Gate: `GET /api/chart?symbol=XAUUSD&tf=1h&source=yahoo&limit=200` returns real candles

### Ready Queue (pending, dependencies met)

| ID   | Title                          | Agent  | Status      | Depends On |
|------|--------------------------------|--------|-------------|------------|
| T3.2 | Yahoo Finance adapter (Go)     | Gemini | dispatched  | T3.1 ✓     |
| T3.3 | Market bars DB repository      | Codex  | dispatched  | T3.1 ✓     |

### Blocked (waiting on dependency)

| ID   | Title                          | Agent  | Status  | Depends On  |
|------|--------------------------------|--------|---------|-------------|
| T3.4 | /api/chart handler             | Codex  | pending | T3.2, T3.3  |

## Phase 2 — Database: COMPLETE ✓

Gate passed: `{"ok":true,"service":"trade-buddy-api","db":"ok"}`

### Done (this phase)

| ID   | Title                              | Agent  | Status | Result file           |
|------|------------------------------------|--------|--------|-----------------------|
| T2.1 | Postgres schema migration file     | Codex  | done ✓ | tasks/T2.1-result.md  |
| T2.2 | Go DB package + migration runner   | Codex  | done ✓ | tasks/T2.2-result.md  |
| T2.3 | Health check with DB ping          | Codex  | done ✓ | tasks/T2.3-result.md  |
| T3.1 | Candle model + source interface    | Codex  | done ✓ | tasks/T3.1-result.md  |

---

## Results Pending Review

_empty — fill when agent completes_

---

## Dispatch Log

_append-only_

| Time        | Task | Agent  | Action     |
|-------------|------|--------|------------|
| 2026-05-02  | T2.1 | Codex  | dispatched |
| 2026-05-02  | T2.1 | Codex  | done ✓     |
| 2026-05-02  | T2.2 | Codex  | dispatched |
| 2026-05-02  | T2.2 | Codex  | done ✓     |
| 2026-05-02  | T2.3 | Codex  | done ✓     |
| 2026-05-02  | T3.1 | Codex  | dispatched |
| 2026-05-02  | T3.1 | Codex  | done ✓     |
| 2026-05-02  | T3.2 | Gemini | dispatched |
| 2026-05-02  | T3.3 | Codex  | dispatched |

---

## Backlog (future phases)

| ID   | Phase | Title                           | Agent    | Depends On |
|------|-------|---------------------------------|----------|------------|
| T3.4 | 3     | /api/chart handler              | Codex    | T3.2,T3.3  |
| T4.1 | 4     | Indicators (SMA, ATR)           | Gemini   | T3.3       |
| T4.2 | 4     | Trend + swing detection         | Gemini   | T4.1       |
| T4.3 | 4     | Support/Resistance levels       | Gemini   | T4.2       |
| T4.4 | 4     | Forecast bias + routes          | Gemini   | T4.3       |
| T4.5 | 4     | Weekly SVG generator + cache    | Gemini   | T4.4       |
| T4.6 | 4     | /api/weekly-plan handler        | Codex    | T4.5       |
| T5.1 | 5     | API types + fetch helpers       | Opencode | T4.6       |
| T5.2 | 5     | TradingChart component          | Opencode | T5.1       |
| T5.3 | 5     | Timeframe tabs + Multi-TF board | Opencode | T5.2       |
| T5.4 | 5     | Weekly Plan + Swing Bias UI     | Opencode | T5.3       |
| T5.5 | 5     | Page layout wiring              | Opencode | T5.4       |
| T6.1 | 6     | Alerts DB repository            | Codex    | T5.5       |
| T6.2 | 6     | Alerts HTTP handlers            | Codex    | T6.1       |
| T6.3 | 6     | Alerts UI                       | Opencode | T6.2       |
| T6.4 | 6     | Persistence test                | Codex    | T6.3       |
| T7.1 | 7     | Stats endpoint                  | Codex    | T6.4       |
| T7.2 | 7     | Stats UI                        | Opencode | T7.1       |

---

## Research Topics

| File                          | Subject                    | Created By | Used By |
|-------------------------------|----------------------------|------------|---------|
| _empty_                       |                            |            |         |

---

## Status Legend

| Status      | Meaning                                      |
|-------------|----------------------------------------------|
| `pending`   | Not started, ready when deps are met         |
| `dispatched`| Task tool spawned, agent running             |
| `review`    | Agent finished, waiting Claude review        |
| `done`      | Reviewed + accepted, marked [x] in plan.md  |
| `blocked`   | Waiting on dependency or missing info        |
| `failed`    | Agent failed — needs retry or reassign       |
| `research`  | Blocked on knowledge → Gemini dispatched     |
