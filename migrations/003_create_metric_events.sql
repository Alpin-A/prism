CREATE TABLE metric_events (
    id            BIGSERIAL PRIMARY KEY,
    experiment_id VARCHAR(128) NOT NULL,
    user_id       VARCHAR(255) NOT NULL,
    variant_id    VARCHAR(128) NOT NULL,
    event_type    VARCHAR(64)  NOT NULL,
    value         DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_one_event_per_user
        UNIQUE (experiment_id, user_id, event_type)
);

CREATE INDEX idx_metric_events_exp_variant
    ON metric_events(experiment_id, variant_id);