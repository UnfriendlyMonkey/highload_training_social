CREATE TABLE IF NOT EXISTS load_test_events (
  id BIGSERIAL PRIMARY KEY,
  run_id TEXT NOT NULL,
  seq BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (run_id, seq)
);