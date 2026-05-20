"""Statistical significance engine for two-proportion z-test."""

from __future__ import annotations

import math
from dataclasses import dataclass

from scipy.stats import norm


@dataclass
class VariantStats:
    variant_id: str
    n_users: int
    n_events: int

    @property
    def rate(self) -> float:
        if self.n_users == 0:
            return 0.0
        return self.n_events / self.n_users


@dataclass
class VariantResult:
    variant_id: str
    n_users: int
    n_events: int
    rate: float
    ci_lower: float
    ci_upper: float


@dataclass
class ExperimentResult:
    variants: list[VariantResult]
    p_value: float
    is_significant: bool
    control_id: str


def _wilson_ci(n_events: int, n_users: int, z: float = 1.96) -> tuple[float, float]:
    """Wilson score interval for a proportion."""
    if n_users == 0:
        return 0.0, 0.0
    p = n_events / n_users
    denominator = 1 + z * z / n_users
    centre = (p + z * z / (2 * n_users)) / denominator
    margin = (z * math.sqrt(p * (1 - p) / n_users + z * z / (4 * n_users * n_users))) / denominator
    return max(0.0, centre - margin), min(1.0, centre + margin)


def _two_prop_z_test(
    n_events_a: int,
    n_users_a: int,
    n_events_b: int,
    n_users_b: int,
) -> float:
    """Two-sided p-value for H0: rate_a == rate_b."""
    if n_users_a == 0 or n_users_b == 0:
        return 1.0

    p_a = n_events_a / n_users_a
    p_b = n_events_b / n_users_b
    p_pool = (n_events_a + n_events_b) / (n_users_a + n_users_b)

    variance = p_pool * (1 - p_pool) * (1 / n_users_a + 1 / n_users_b)
    if variance == 0:
        return 1.0 if p_a == p_b else 0.0

    z = (p_b - p_a) / math.sqrt(variance)
    # two-sided p-value from standard normal
    p_value = 2 * norm.sf(abs(z))
    return float(p_value)


def compute(variants: list[VariantStats], control_id: str) -> ExperimentResult:
    """Compute per-variant CIs and a two-sided p-value vs the control variant."""
    if not variants:
        raise ValueError("no variants provided")

    control = next((v for v in variants if v.variant_id == control_id), variants[0])
    actual_control_id = control.variant_id

    results: list[VariantResult] = []
    for v in variants:
        lo, hi = _wilson_ci(v.n_events, v.n_users)
        results.append(VariantResult(
            variant_id=v.variant_id,
            n_users=v.n_users,
            n_events=v.n_events,
            rate=v.rate,
            ci_lower=lo,
            ci_upper=hi,
        ))

    treatments = [v for v in variants if v.variant_id != actual_control_id]
    if not treatments:
        p_value = 1.0
    elif len(treatments) == 1:
        t = treatments[0]
        p_value = _two_prop_z_test(control.n_events, control.n_users, t.n_events, t.n_users)
    else:
        # Bonferroni-corrected minimum p-value across all treatment arms
        p_values = [
            _two_prop_z_test(control.n_events, control.n_users, t.n_events, t.n_users)
            for t in treatments
        ]
        p_value = min(p_values) * len(p_values)

    return ExperimentResult(
        variants=results,
        p_value=min(p_value, 1.0),
        is_significant=p_value < 0.05,
        control_id=actual_control_id,
    )