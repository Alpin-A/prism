"""A/A simulation: measure the false-positive rate of the stats engine.

In an A/A test both arms have the same true conversion rate, so every
"significant" result from engine.compute() is a false positive. Over many
trials, the fraction of significant results should land close to alpha (0.05).

Run from the stats/ directory (needs numpy and scipy from requirements.txt):
    python aa_simulation.py
"""

from __future__ import annotations

import math

import numpy as np

from engine import VariantStats, compute

N_TRIALS = 10_000
USERS_PER_ARM = 5_000
TRUE_RATE = 0.10
# Must match the threshold hard-coded in engine.compute (p_value < 0.05).
ALPHA = 0.05
SEED = 0


def run() -> None:
    # One seeded generator for the whole run, so the result is reproducible.
    rng = np.random.default_rng(SEED)
    false_positives = 0

    for _ in range(N_TRIALS):
        # Conversions in each arm ~ Binomial(users, rate); both arms share the
        # same true rate, so any difference between them is pure chance.
        control = VariantStats(
            variant_id="control",
            n_users=USERS_PER_ARM,
            # rng.binomial returns a numpy integer; cast to the plain int the dataclass declares.
            n_events=int(rng.binomial(USERS_PER_ARM, TRUE_RATE)),
        )
        treatment = VariantStats(
            variant_id="treatment",
            n_users=USERS_PER_ARM,
            n_events=int(rng.binomial(USERS_PER_ARM, TRUE_RATE)),
        )

        # Two arms -> compute() runs a single two-sided z-test (no Bonferroni).
        result = compute([control, treatment], control_id="control")
        if result.is_significant:
            false_positives += 1

    fpr = false_positives / N_TRIALS

    # The observed rate is itself a binomial proportion over N_TRIALS trials,
    # so its standard error is sqrt(alpha * (1 - alpha) / N_TRIALS).
    std_err = math.sqrt(ALPHA * (1 - ALPHA) / N_TRIALS)
    # 1.96 standard errors either side covers ~95% of runs of a correct engine.
    band_low, band_high = ALPHA - 1.96 * std_err, ALPHA + 1.96 * std_err

    print(f"Trials:              {N_TRIALS:,}")
    print(f"Users per arm:       {USERS_PER_ARM:,}")
    print(f"True conversion:     {TRUE_RATE:.2f} (both arms)")
    print(f"False positives:     {false_positives:,}")
    print(f"False-positive rate: {fpr:.4f} (alpha = {ALPHA})")
    print(f"Expected 95% band:   [{band_low:.4f}, {band_high:.4f}]")
    print(f"Within band:         {'yes' if band_low <= fpr <= band_high else 'no'}")


if __name__ == "__main__":
    run()
