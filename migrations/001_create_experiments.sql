CREATE TABLE experiments (
    id          VARCHAR(128) PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    status      VARCHAR(32) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft', 'active', 'paused', 'concluded')),
    metric_type VARCHAR(64) NOT NULL DEFAULT 'conversion',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);