from hypothesis import given, strategies as st

from discount import apply_discount


@given(
    subtotal=st.integers(min_value=0, max_value=10_000),
    percent=st.integers(min_value=0, max_value=100),
)
def test_discount_total(subtotal: int, percent: int) -> None:
    assert apply_discount(subtotal, percent) == subtotal * (100 - percent) // 100
