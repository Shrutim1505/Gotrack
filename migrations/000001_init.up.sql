CREATE TABLE IF NOT EXISTS events (
    id         VARCHAR(36) PRIMARY KEY,
    source_id  VARCHAR(255) NOT NULL,
    type       VARCHAR(255) NOT NULL,
    payload    TEXT NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL,
    valid      BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_source_id ON events(source_id);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_valid ON events(valid);

CREATE TABLE IF NOT EXISTS quarantined_events (
    id         VARCHAR(36) PRIMARY KEY,
    source_id  VARCHAR(255) NOT NULL,
    type       VARCHAR(255) NOT NULL,
    payload    TEXT NOT NULL,
    timestamp  TIMESTAMPTZ NOT NULL,
    reason     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    key       VARCHAR(64) PRIMARY KEY,
    source_id VARCHAR(255) NOT NULL,
    active    BOOLEAN NOT NULL DEFAULT true
);

-- Seed a default API key for testing
INSERT INTO api_keys (key, source_id, active) VALUES
    ('test-api-key-1', 'source-alpha', true),
    ('test-api-key-2', 'source-beta', true)
ON CONFLICT DO NOTHING;
