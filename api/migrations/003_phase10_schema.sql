ALTER TABLE signal_events ADD COLUMN IF NOT EXISTS dedup_key TEXT;
ALTER TABLE signal_events ADD COLUMN IF NOT EXISTS confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
UPDATE signal_events SET dedup_key = id::text WHERE dedup_key IS NULL;
ALTER TABLE signal_events ALTER COLUMN dedup_key SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_events_dedup_key ON signal_events(dedup_key);
CREATE INDEX IF NOT EXISTS idx_alerts_pattern ON alerts ((context->>'pattern'));
CREATE INDEX IF NOT EXISTS idx_alerts_session ON alerts ((context->>'session'));
