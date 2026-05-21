CREATE TABLE feature_flags (
    id          VARCHAR(128) PRIMARY KEY,
    name        TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_pct NUMERIC(5,2) NOT NULL DEFAULT 0.0
                CHECK (rollout_pct BETWEEN 0 AND 100),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE flag_overrides (
    flag_id  VARCHAR(128) NOT NULL REFERENCES feature_flags(id) ON DELETE CASCADE,
    user_id  VARCHAR(255) NOT NULL,
    enabled  BOOLEAN NOT NULL,
    PRIMARY KEY (flag_id, user_id)
);