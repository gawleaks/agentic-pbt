def apply_discount(subtotal: int, percent: int) -> int:
    return subtotal * (100 - percent) // 100
