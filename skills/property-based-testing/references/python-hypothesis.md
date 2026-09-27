# Hypothesis for PBT v1

Official documentation: https://hypothesis.readthedocs.io/en/latest/

Use `@given` with explicit strategies for the contract's valid domain. Keep the assertion independent of the system under test.

```python
from hypothesis import given, strategies as st

from discount import apply_discount


@given(
    subtotal=st.integers(min_value=0, max_value=10_000),
    percent=st.integers(min_value=0, max_value=100),
)
def test_discount_total(subtotal: int, percent: int) -> None:
    expected = subtotal * (100 - percent) // 100
    assert apply_discount(subtotal, percent) == expected
```

## Boundaries and invalid inputs

Include boundaries in valid strategies (`min_value=0`, `max_value=100`) and state them in the contract. Test invalid values separately when the public behavior rejects them:

```python
import pytest
from hypothesis import given, strategies as st


@given(percent=st.integers().filter(lambda value: value < 0 or value > 100))
def test_rejects_invalid_percent(percent: int) -> None:
    with pytest.raises(ValueError):
        apply_discount(100, percent)
```

Do not generate invalid input for a property whose contract promises only valid input unless rejection is itself the property.

## Failures, shrinking, and replay

Hypothesis shrinks a failing generated case to a small counterexample. Preserve that case while investigating; do not loosen the assertion to hide it. Re-run the exact contract node through `pbt verify --task <id> --json` after repair. For a durable regression, add the minimized case with `@example`:

```python
from hypothesis import example, given, strategies as st


@example(subtotal=1, percent=100)
@given(subtotal=st.integers(min_value=0, max_value=10_000), percent=st.integers(min_value=0, max_value=100))
def test_discount_total(subtotal: int, percent: int) -> None:
    assert apply_discount(subtotal, percent) == subtotal * (100 - percent) // 100
```

If Hypothesis prints an `@reproduce_failure` decorator, copy its exact version and payload above the failing `@given` test to replay that failure. Remove the temporary replay decorator only after the repaired property passes.

The URL is for agent reference. In v1, `pbt` does not download, cache, or validate online documentation.
