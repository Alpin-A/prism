-- Traffic arms for an experiment. Weights must sum to 1.0 (enforced in application layer).
CREATE TABLE variants (
    experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    id             VARCHAR(128) NOT NULL,
    name           TEXT NOT NULL,
    weight         NUMERIC(6,5) NOT NULL CHECK (weight > 0 AND weight <= 1),
    PRIMARY KEY (experiment_id, id)
);

-- Primary key enforces one variant per user per experiment (sticky assignment).
CREATE TABLE exposures (
    experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    user_id        VARCHAR(255) NOT NULL,
    variant_id     VARCHAR(128) NOT NULL,
    first_seen     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (experiment_id, user_id)
);

-- Forces a specific user into a variant, bypassing the hash-based assignment.
-- Used for QA and internal testing without affecting production traffic.
CREATE TABLE assignment_overrides (
    experiment_id  VARCHAR(128) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    user_id        VARCHAR(255) NOT NULL,
    variant_id     VARCHAR(128) NOT NULL,
    PRIMARY KEY (experiment_id, user_id)
);