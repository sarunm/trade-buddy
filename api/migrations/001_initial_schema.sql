CREATE TABLE market_bars (
  id BIGSERIAL PRIMARY KEY,
  symbol TEXT NOT NULL,
  timeframe TEXT NOT NULL,
  source TEXT NOT NULL,
  ts TIMESTAMPTZ NOT NULL,
  open DOUBLE PRECISION NOT NULL,
  high DOUBLE PRECISION NOT NULL,
  low DOUBLE PRECISION NOT NULL,
  close DOUBLE PRECISION NOT NULL,
  volume DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(symbol, timeframe, source, ts)
);

CREATE TABLE alerts (
  id UUID PRIMARY KEY,
  symbol TEXT NOT NULL,
  timeframe TEXT NOT NULL,
  direction TEXT NOT NULL,
  entry DOUBLE PRECISION,
  stop_loss DOUBLE PRECISION,
  take_profit DOUBLE PRECISION,
  risk_reward DOUBLE PRECISION,
  confidence DOUBLE PRECISION,
  reason JSONB NOT NULL DEFAULT '[]',
  context JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'open',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at TIMESTAMPTZ
);

CREATE TABLE alert_outcomes (
  id BIGSERIAL PRIMARY KEY,
  alert_id UUID NOT NULL REFERENCES alerts(id),
  outcome TEXT NOT NULL,
  resolved_price DOUBLE PRECISION,
  bars_elapsed INTEGER,
  mfe DOUBLE PRECISION,
  mae DOUBLE PRECISION,
  details JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE weekly_plans (
  id UUID PRIMARY KEY,
  symbol TEXT NOT NULL,
  week_start DATE NOT NULL,
  source TEXT NOT NULL,
  forecast_bias TEXT NOT NULL,
  payload JSONB NOT NULL,
  image_path TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(symbol, week_start, source)
);

CREATE TABLE signal_events (
  id UUID PRIMARY KEY,
  symbol TEXT NOT NULL,
  timeframe TEXT NOT NULL,
  signal_type TEXT NOT NULL,
  name TEXT NOT NULL,
  direction TEXT NOT NULL,
  price DOUBLE PRECISION,
  ts TIMESTAMPTZ NOT NULL,
  geometry JSONB NOT NULL DEFAULT '{}',
  reason JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_config (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_market_bars_lookup ON market_bars(symbol, timeframe, source, ts DESC);
CREATE INDEX idx_alerts_symbol_time ON alerts(symbol, timeframe, created_at DESC);
CREATE INDEX idx_weekly_plans_lookup ON weekly_plans(symbol, week_start, source);
CREATE INDEX idx_signal_events_lookup ON signal_events(symbol, timeframe, ts DESC);
