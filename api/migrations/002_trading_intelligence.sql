CREATE TABLE top_down_contexts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  symbol TEXT NOT NULL,
  captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  detector_version TEXT NOT NULL DEFAULT '1.0',
  monthly_trend TEXT NOT NULL DEFAULT 'neutral',
  weekly_trend TEXT NOT NULL DEFAULT 'neutral',
  daily_trend TEXT NOT NULL DEFAULT 'neutral',
  h4_trend TEXT NOT NULL DEFAULT 'neutral',
  h1_trend TEXT NOT NULL DEFAULT 'neutral',
  m15_trend TEXT NOT NULL DEFAULT 'neutral',
  swing_data JSONB NOT NULL DEFAULT '{}',
  levels_data JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE trade_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  symbol TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  bias TEXT NOT NULL,
  entry DOUBLE PRECISION,
  sl DOUBLE PRECISION,
  tp DOUBLE PRECISION,
  tp2 DOUBLE PRECISION,
  rr DOUBLE PRECISION,
  notes TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft'
);

CREATE TABLE order_simulations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  symbol TEXT NOT NULL,
  timeframe TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  direction TEXT NOT NULL,
  order_type TEXT NOT NULL,
  entry DOUBLE PRECISION NOT NULL,
  sl DOUBLE PRECISION NOT NULL,
  tp DOUBLE PRECISION NOT NULL,
  expiry_bars INTEGER
);

CREATE TABLE simulation_outcomes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  simulation_id UUID NOT NULL REFERENCES order_simulations(id),
  triggered_at TIMESTAMPTZ,
  outcome TEXT NOT NULL,
  mae DOUBLE PRECISION,
  mfe DOUBLE PRECISION,
  r_multiple DOUBLE PRECISION,
  duration_bars INTEGER,
  bars_to_outcome INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trade_features (
  id BIGSERIAL PRIMARY KEY,
  simulation_id UUID NOT NULL REFERENCES order_simulations(id),
  feature_key TEXT NOT NULL,
  feature_value TEXT NOT NULL
);

CREATE TABLE behavior_tags (
  id BIGSERIAL PRIMARY KEY,
  simulation_id UUID NOT NULL REFERENCES order_simulations(id),
  tag TEXT NOT NULL
);

CREATE TABLE rule_calibrations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_name TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1,
  params JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(rule_name, version)
);

CREATE TABLE notification_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  sent_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_top_down_contexts_lookup ON top_down_contexts(symbol, captured_at DESC);
CREATE INDEX idx_trade_plans_symbol ON trade_plans(symbol, created_at DESC);
CREATE INDEX idx_order_simulations_symbol ON order_simulations(symbol, created_at DESC);
CREATE INDEX idx_simulation_outcomes_sim ON simulation_outcomes(simulation_id);
CREATE INDEX idx_trade_features_sim ON trade_features(simulation_id);
CREATE INDEX idx_behavior_tags_sim ON behavior_tags(simulation_id);
CREATE INDEX idx_notification_events_status ON notification_events(status, created_at DESC);
