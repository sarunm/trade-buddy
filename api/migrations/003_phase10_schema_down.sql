DROP INDEX IF EXISTS idx_alerts_session;
DROP INDEX IF EXISTS idx_alerts_pattern;
DROP INDEX IF EXISTS idx_signal_events_dedup_key;
ALTER TABLE signal_events DROP COLUMN IF EXISTS confidence;
ALTER TABLE signal_events DROP COLUMN IF EXISTS dedup_key;
