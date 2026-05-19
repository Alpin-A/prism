CREATE TABLE agg_metrics (
    experiment_id VARCHAR(128)     NOT NULL,
    variant_id    VARCHAR(128)     NOT NULL,
    event_type    VARCHAR(64)      NOT NULL,
    n_events      BIGINT           NOT NULL DEFAULT 0,
    sum_value     DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    sum_sq_value  DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    last_updated  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (experiment_id, variant_id, event_type)
);