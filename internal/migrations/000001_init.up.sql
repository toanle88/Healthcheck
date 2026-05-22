CREATE TABLE IF NOT EXISTS targets (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	url TEXT UNIQUE NOT NULL,
	is_active BOOLEAN DEFAULT TRUE,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE targets ADD COLUMN IF NOT EXISTS method TEXT NOT NULL DEFAULT 'GET';
ALTER TABLE targets ADD COLUMN IF NOT EXISTS headers TEXT;
ALTER TABLE targets ADD COLUMN IF NOT EXISTS expected_status INT NOT NULL DEFAULT 200;
ALTER TABLE targets ADD COLUMN IF NOT EXISTS response_contains TEXT;
ALTER TABLE targets ADD COLUMN IF NOT EXISTS failure_threshold INT NOT NULL DEFAULT 3;
ALTER TABLE targets ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0;
ALTER TABLE targets ADD COLUMN IF NOT EXISTS last_alert_status TEXT NOT NULL DEFAULT 'up';

INSERT INTO targets (name, url) VALUES
	('Httpbin', 'http://httpbin.org/get'),
	('GitHub', 'https://github.com'),
	('Azure Status', 'https://azure.microsoft.com/en-us/status/')
ON CONFLICT (url) DO NOTHING;

CREATE TABLE IF NOT EXISTS checks (
	id SERIAL PRIMARY KEY,
	target TEXT NOT NULL,
	status TEXT NOT NULL,
	latency_ms INT NOT NULL,
	checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checks_target_checked_at ON checks(target, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_checks_checked_at_target ON checks(checked_at DESC, target);
