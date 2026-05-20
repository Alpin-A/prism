"""Tests for the statistical significance engine."""

import math
import pytest

from engine import VariantStats, compute, _wilson_ci, _two_prop_z_test


# --- Wilson CI ---

def test_wilson_ci_basic():
    lo, hi = _wilson_ci(50, 100)
    assert 0.40 < lo < 0.50
    assert 0.50 < hi < 0.62


def test_wilson_ci_zero_users():
    lo, hi = _wilson_ci(0, 0)
    assert lo == 0.0 and hi == 0.0


def test_wilson_ci_all_converted():
    lo, hi = _wilson_ci(100, 100)
    assert math.isclose(hi, 1.0, abs_tol=1e-9)
    assert lo > 0.95


def test_wilson_ci_none_converted():
    lo, hi = _wilson_ci(0, 100)
    assert lo == 0.0
    assert hi < 0.05


# --- Two-proportion z-test ---

def test_z_test_clearly_significant():
    # 5% vs 10% with large samples
    p = _two_prop_z_test(500, 10_000, 1000, 10_000)
    assert p < 0.001


def test_z_test_clearly_not_significant():
    # identical rates
    p = _two_prop_z_test(100, 1000, 100, 1000)
    assert p == 1.0 or p > 0.99


def test_z_test_zero_users():
    p = _two_prop_z_test(0, 0, 10, 100)
    assert p == 1.0


def test_z_test_zero_variance():
    # both zero events — p_pool=0, variance=0
    p = _two_prop_z_test(0, 100, 0, 100)
    assert p == 1.0


# --- compute ---

def _make_variant(vid, n_users, n_events):
    return VariantStats(variant_id=vid, n_users=n_users, n_events=n_events)


def test_compute_significant():
    variants = [
        _make_variant("control", 10_000, 500),    # 5%
        _make_variant("treatment", 10_000, 1000),  # 10%
    ]
    result = compute(variants, "control")
    assert result.control_id == "control"
    assert result.is_significant
    assert result.p_value < 0.05

    ctrl = next(v for v in result.variants if v.variant_id == "control")
    assert math.isclose(ctrl.rate, 0.05)
    assert ctrl.ci_lower < 0.05 < ctrl.ci_upper


def test_compute_not_significant():
    variants = [
        _make_variant("control", 100, 10),
        _make_variant("treatment", 100, 11),
    ]
    result = compute(variants, "control")
    assert not result.is_significant
    assert result.p_value > 0.05


def test_compute_control_fallback_to_first_variant():
    variants = [
        _make_variant("a", 1000, 100),
        _make_variant("b", 1000, 200),
    ]
    # control_id not in list — should fall back to first variant
    result = compute(variants, "missing")
    assert result.control_id == "a"


def test_compute_single_variant():
    variants = [_make_variant("control", 500, 50)]
    result = compute(variants, "control")
    assert result.p_value == 1.0
    assert not result.is_significant


def test_compute_three_variants_bonferroni():
    variants = [
        _make_variant("control",    10_000, 500),
        _make_variant("treatment1", 10_000, 1000),
        _make_variant("treatment2", 10_000, 1000),
    ]
    result = compute(variants, "control")
    # With Bonferroni correction p_value is still significant for 10% vs 5%
    assert result.is_significant
    assert len(result.variants) == 3


def test_compute_zero_exposure():
    variants = [
        _make_variant("control", 0, 0),
        _make_variant("treatment", 0, 0),
    ]
    result = compute(variants, "control")
    assert result.p_value == 1.0
    for v in result.variants:
        assert v.rate == 0.0
        assert v.ci_lower == 0.0
        assert v.ci_upper == 0.0


def test_compute_raises_on_empty():
    with pytest.raises(ValueError, match="no variants"):
        compute([], "control")