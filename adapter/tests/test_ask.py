"""Adapter L1 unit tests (≥12 cases aligned with 10b U-ASK-*)."""
from __future__ import annotations

import os
from unittest.mock import MagicMock, patch

import httpx
import pytest
from fastapi.testclient import TestClient

from app.main import create_app
from app.mapping import (
    build_ask_response,
    detect_no_hit,
    normalize_citation,
    new_trace_id,
)
from app.schema_golden import validate_golden_qa_doc, validate_golden_qa_path


@pytest.fixture()
def client(monkeypatch: pytest.MonkeyPatch) -> TestClient:
    monkeypatch.delenv("BUDDY_DEMO_MODE", raising=False)
    monkeypatch.setenv("DIFY_API_KEY", "app-test-key")
    monkeypatch.setenv("DIFY_BASE_URL", "http://dify.test")
    monkeypatch.setenv("DIFY_TIMEOUT_SEC", "5")
    return TestClient(create_app())


def test_u_ask_02_empty_question_rejected(client: TestClient) -> None:
    r = client.post("/v1/buddy/ask", json={"question": "   "})
    assert r.status_code == 422


def test_trace_id_non_empty() -> None:
    t = new_trace_id()
    assert t.startswith("01J")
    assert len(t) >= 10


def test_normalize_citation_full() -> None:
    c = normalize_citation(
        {
            "document_name": "手册.pdf",
            "page": 5,
            "content": "snippet",
            "score": 0.82,
        }
    )
    assert c["doc_name"] == "手册.pdf"
    assert c["page"] == 5
    assert c["score"] == pytest.approx(0.82)


def test_normalize_citation_partial_fields() -> None:
    c = normalize_citation({"dataset_name": "kb"})
    assert c["doc_name"] == "kb"
    assert c["page"] is None
    assert c["snippet"] == ""


def test_normalize_citation_non_dict() -> None:
    c = normalize_citation("raw")
    assert c["doc_name"] == "unknown"
    assert "raw" in c["snippet"]


def test_detect_no_hit_empty_answer() -> None:
    assert detect_no_hit("", []) is True


def test_detect_no_hit_marker() -> None:
    assert detect_no_hit("抱歉，知识库中没有相关信息。", []) is True


def test_detect_no_hit_with_citations() -> None:
    assert detect_no_hit("anything", [{"doc_name": "a.pdf"}]) is False


def test_build_ok() -> None:
    p = build_ask_response(
        trace_id="T1",
        answer="参数 -20",
        raw_citations=[{"document_name": "a.pdf", "content": "x", "score": 0.9}],
    )
    assert p["status"] == "ok"
    assert p["citations"][0]["doc_name"] == "a.pdf"
    assert p["refuse_reason"] is None


def test_build_no_hit() -> None:
    p = build_ask_response(trace_id="T2", answer="", raw_citations=[])
    assert p["status"] == "no_hit"
    assert p["answer"]


def test_build_error() -> None:
    p = build_ask_response(
        trace_id="T3",
        answer="",
        raw_citations=[],
        dify_error="timeout",
        force_status="error",
    )
    assert p["status"] == "error"
    assert p["refuse_reason"] == "timeout"


def test_u_ask_01_ok_via_mock_dify(client: TestClient) -> None:
    mock_resp = MagicMock()
    mock_resp.raise_for_status = MagicMock()
    mock_resp.json.return_value = {
        "answer": "OBJ-1001 冷运行 -20℃，时长 2 小时。",
        "metadata": {
            "retriever_resources": [
                {
                    "document_name": "OBJ-1001-操作手册_V3.2.pdf",
                    "page": 5,
                    "content": "-20",
                    "score": 0.91,
                }
            ]
        },
    }

    with patch("httpx.Client") as Cls:
        inst = Cls.return_value.__enter__.return_value
        inst.post.return_value = mock_resp
        r = client.post(
            "/v1/buddy/ask",
            json={
                "question": "OBJ-1001 的冷运行参数是什么？",
                "meta": {"object_id": "OBJ-1001"},
            },
        )
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert isinstance(body["citations"], list)
    assert body["citations"][0]["doc_name"].endswith(".pdf")
    _args, kwargs = inst.post.call_args
    assert "OBJ-1001" in kwargs["json"]["query"]


def test_u_ask_03_no_hit(client: TestClient) -> None:
    mock_resp = MagicMock()
    mock_resp.raise_for_status = MagicMock()
    mock_resp.json.return_value = {
        "answer": "知识库中没有相关内容。",
        "metadata": {"retriever_resources": []},
    }
    with patch("httpx.Client") as Cls:
        inst = Cls.return_value.__enter__.return_value
        inst.post.return_value = mock_resp
        r = client.post("/v1/buddy/ask", json={"question": "XYZ-999 参数？"})
    assert r.status_code == 200
    assert r.json()["status"] == "no_hit"


def test_u_ask_04_timeout(client: TestClient) -> None:
    with patch("httpx.Client") as Cls:
        inst = Cls.return_value.__enter__.return_value
        inst.post.side_effect = httpx.TimeoutException("slow")
        r = client.post("/v1/buddy/ask", json={"question": "会超时吗"})
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "error"
    assert body["refuse_reason"] == "timeout"


def test_u_ask_05_partial_citation_serializable(client: TestClient) -> None:
    mock_resp = MagicMock()
    mock_resp.raise_for_status = MagicMock()
    mock_resp.json.return_value = {
        "answer": "有答案",
        "metadata": {"retriever_resources": [{"score": "bad"}]},
    }
    with patch("httpx.Client") as Cls:
        inst = Cls.return_value.__enter__.return_value
        inst.post.return_value = mock_resp
        r = client.post("/v1/buddy/ask", json={"question": "残缺引用"})
    assert r.status_code == 200
    c = r.json()["citations"][0]
    assert "doc_name" in c
    assert c["score"] is None


def test_missing_api_key_error(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("BUDDY_DEMO_MODE", raising=False)
    monkeypatch.delenv("DIFY_API_KEY", raising=False)
    c = TestClient(create_app())
    r = c.post("/v1/buddy/ask", json={"question": "hello"})
    assert r.status_code == 200
    assert r.json()["status"] == "error"


def test_healthz(client: TestClient) -> None:
    with patch("httpx.get") as g:
        g.return_value = MagicMock(status_code=200)
        r = client.get("/healthz")
    assert r.status_code == 200
    assert "dify_reachable" in r.json()


def test_golden_schema_file() -> None:
    path = os.path.abspath(
        os.path.join(os.path.dirname(__file__), "..", "..", "docs", "golden-qa.yaml")
    )
    result = validate_golden_qa_path(path)
    assert result["ok"] is True, result
    assert result["count"] >= 10


def test_golden_schema_invalid() -> None:
    bad = validate_golden_qa_doc({"cases": [{"id": "x"}]})
    assert bad["ok"] is False
