from app.demo_mode import match_demo
from app.main import create_app
from fastapi.testclient import TestClient


def test_demo_ok_obj1001(monkeypatch):
    monkeypatch.setenv("BUDDY_DEMO_MODE", "1")
    c = TestClient(create_app())
    r = c.post("/v1/buddy/ask", json={"question": "OBJ-1001 的冷运行参数是什么？"})
    assert r.status_code == 200
    body = r.json()
    assert body["status"] == "ok"
    assert "-20" in body["answer"]
    assert body["citations"]


def test_demo_no_hit(monkeypatch):
    monkeypatch.setenv("BUDDY_DEMO_MODE", "1")
    c = TestClient(create_app())
    r = c.post("/v1/buddy/ask", json={"question": "红烧肉怎么做？"})
    assert r.json()["status"] == "no_hit"


def test_match_demo_direct():
    assert match_demo("液压站油温")["status"] == "ok"
