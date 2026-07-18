"""Overlay SearchKnowledgeDoc L1 tests (≥8)."""
from __future__ import annotations

import json

from logic import compose_query, map_dify_to_contract, refuse_copy, skill_result_json


def test_compose_without_object() -> None:
    assert compose_query("hello") == "hello"


def test_compose_with_object() -> None:
    assert compose_query("参数？", "OBJ-1001") == "[object_id=OBJ-1001] 参数？"


def test_compose_strips() -> None:
    assert compose_query("  x  ") == "x"


def test_map_ok() -> None:
    p = map_dify_to_contract(
        answer="ok ans",
        retriever_resources=[{"document_name": "a.pdf", "content": "s", "score": 0.5}],
        trace_id="T",
    )
    assert p["status"] == "ok"
    assert p["citations"][0]["doc_name"] == "a.pdf"


def test_map_no_hit() -> None:
    p = map_dify_to_contract(answer="知识库中没有相关内容", retriever_resources=[], trace_id="T")
    assert p["status"] == "no_hit"


def test_map_error() -> None:
    p = map_dify_to_contract(answer="", retriever_resources=[], error="boom", trace_id="T")
    assert p["status"] == "error"
    assert p["refuse_reason"] == "boom"


def test_refuse_copy() -> None:
    assert "未命中" in refuse_copy("no_hit")
    assert "Docker" in refuse_copy("error")
    assert refuse_copy("ok") == ""


def test_skill_result_json_roundtrip() -> None:
    p = map_dify_to_contract(
        answer="a",
        retriever_resources=[{"document_name": "d.pdf"}],
        trace_id="TID",
    )
    s = skill_result_json(p)
    data = json.loads(s)
    assert data["trace_id"] == "TID"
    assert data["citations"][0]["doc_name"] == "d.pdf"


def test_normalize_missing_fields_via_map() -> None:
    p = map_dify_to_contract(answer="有", retriever_resources=[{}], trace_id="T")
    assert p["status"] == "ok"
    assert p["citations"][0]["doc_name"] == "unknown"
