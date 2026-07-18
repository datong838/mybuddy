"""AOS Local Workstation · buddy ask adapter (127.0.0.1)."""
from __future__ import annotations

import logging
import os
import time
from typing import Any, Literal, Optional

import httpx
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from .mapping import build_ask_response, new_trace_id
from .schema_golden import validate_golden_qa_path
from .demo_mode import match_demo

logger = logging.getLogger("buddy.adapter")
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s [adapter] %(message)s",
)

Status = Literal["ok", "no_hit", "error"]


class UserInfo(BaseModel):
    id: str = "local"
    display_name: str = "业务用户"


class AskMeta(BaseModel):
    object_id: Optional[str] = None


class AskRequest(BaseModel):
    question: str = Field(..., min_length=1)
    channel: str = "desktop"
    user: UserInfo = Field(default_factory=UserInfo)
    session_id: Optional[str] = None
    meta: Optional[AskMeta] = None


class Citation(BaseModel):
    doc_name: str
    page: Optional[int] = None
    snippet: str = ""
    score: Optional[float] = None


class AskResponse(BaseModel):
    trace_id: str
    answer: str
    citations: list[Citation]
    status: Status
    refuse_reason: Optional[str] = None


def create_app() -> FastAPI:
    app = FastAPI(title="AOS Buddy Adapter", version="0.1.0")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/healthz")
    def healthz() -> dict[str, Any]:
        dify = os.getenv("DIFY_BASE_URL", "http://127.0.0.1")
        ok = False
        try:
            r = httpx.get(dify.rstrip("/") + "/", timeout=3.0)
            ok = r.status_code < 500
        except Exception:
            ok = False
        return {
            "status": "ok" if ok else "degraded",
            "dify_reachable": ok,
            "demo_mode": os.getenv("BUDDY_DEMO_MODE", "").strip() in ("1", "true", "TRUE", "yes"),
            "ts": int(time.time()),
        }

    @app.post("/v1/buddy/ask", response_model=AskResponse)
    def buddy_ask(body: AskRequest) -> AskResponse:
        q = (body.question or "").strip()
        if not q:
            raise HTTPException(status_code=422, detail="question required")

        base = os.getenv("DIFY_BASE_URL", "http://127.0.0.1").rstrip("/")
        api_key = os.getenv("DIFY_API_KEY", "")
        timeout = float(os.getenv("DIFY_TIMEOUT_SEC", "60"))
        trace_id = new_trace_id()
        demo = os.getenv("BUDDY_DEMO_MODE", "").strip() in ("1", "true", "TRUE", "yes")

        if demo:
            demo_hit = match_demo(q) or {}
            if demo_hit.get("status") == "ok":
                payload = build_ask_response(
                    trace_id=trace_id,
                    answer=str(demo_hit.get("answer") or ""),
                    raw_citations=list(demo_hit.get("citations") or []),
                )
            elif demo_hit.get("status") == "no_hit":
                payload = build_ask_response(
                    trace_id=trace_id,
                    answer=str(demo_hit.get("answer") or ""),
                    raw_citations=[],
                    force_status="no_hit",
                )
            else:
                payload = build_ask_response(
                    trace_id=trace_id,
                    answer="",
                    raw_citations=[],
                    dify_error="demo",
                    force_status="error",
                )
            return AskResponse(**payload)

        if not api_key:
            # Dev-friendly: allow synthetic responses when key missing (unit path uses mocks)
            payload = build_ask_response(
                trace_id=trace_id,
                answer="",
                raw_citations=[],
                dify_error="DIFY_API_KEY not set",
                force_status="error",
            )
            return AskResponse(**payload)

        # Optional object context prepended for retrieval focus
        query = q
        if body.meta and body.meta.object_id:
            query = f"[object_id={body.meta.object_id}] {q}"

        url = f"{base}/v1/chat-messages"
        headers = {
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        }
        req_body = {
            "inputs": {},
            "query": query,
            "response_mode": "blocking",
            "user": body.user.id or "local",
        }
        try:
            with httpx.Client(timeout=timeout) as client:
                r = client.post(url, headers=headers, json=req_body)
                r.raise_for_status()
                data = r.json()
        except httpx.HTTPStatusError as exc:
            body_snip = ""
            try:
                body_snip = (exc.response.text or "")[:1200]
            except Exception:  # noqa: BLE001
                body_snip = ""
            detail = body_snip
            try:
                j = exc.response.json()
                detail = j.get("message") or j.get("code") or j.get("error") or body_snip
                if isinstance(detail, dict):
                    detail = str(detail)
            except Exception:  # noqa: BLE001
                pass
            err = f"HTTP {exc.response.status_code}: {detail}"
            if exc.response.status_code == 400 and "openai" not in err.lower():
                err = (
                    f"{err} | Provider openai does not exist "
                    "(check Dify LLM node; worker log often has this exact text)"
                )
            logger.error(
                "dify chat-messages failed trace=%s detail=%s",
                trace_id,
                err[:800],
            )
            payload = build_ask_response(
                trace_id=trace_id,
                answer="",
                raw_citations=[],
                dify_error=err,
                force_status="error",
            )
            return AskResponse(**payload)
        except httpx.TimeoutException:
            logger.error("dify timeout trace=%s", trace_id)
            payload = build_ask_response(
                trace_id=trace_id,
                answer="",
                raw_citations=[],
                dify_error="timeout",
                force_status="error",
            )
            return AskResponse(**payload)
        except Exception as exc:  # noqa: BLE001
            logger.exception("dify ask failed trace=%s", trace_id)
            payload = build_ask_response(
                trace_id=trace_id,
                answer="",
                raw_citations=[],
                dify_error=str(exc),
                force_status="error",
            )
            return AskResponse(**payload)

        answer = data.get("answer") or data.get("text") or ""
        raw_cites = data.get("metadata", {}).get("retriever_resources") or data.get(
            "retriever_resources"
        ) or []
        logger.info(
            "ask ok trace=%s answer_len=%s cites=%s",
            trace_id,
            len(answer or ""),
            len(raw_cites or []),
        )
        payload = build_ask_response(
            trace_id=trace_id,
            answer=answer,
            raw_citations=raw_cites,
            dify_error=None,
            force_status=None,
        )
        return AskResponse(**payload)

    @app.get("/v1/golden/validate")
    def golden_validate() -> dict[str, Any]:
        root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "docs", "golden-qa.yaml"))
        # also allow env override
        path = os.getenv("GOLDEN_QA_PATH", root)
        return validate_golden_qa_path(path)

    return app


app = create_app()
