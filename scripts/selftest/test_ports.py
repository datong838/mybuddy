"""Port / health helper unit tests (≥2)."""
from __future__ import annotations


def is_port_like(n: int) -> bool:
    return 1 <= n <= 65535


def probe_label(ok: bool) -> str:
    return "up" if ok else "down"


def test_port_valid() -> None:
    assert is_port_like(8090) is True
    assert is_port_like(0) is False


def test_probe_label() -> None:
    assert probe_label(True) == "up"
    assert probe_label(False) == "down"
