"""Map Dify chat response → frozen /v1/buddy/ask contract."""
from __future__ import annotations

import re
import uuid
from typing import Any, Literal, Optional

Status = Literal["ok", "no_hit", "error"]

NO_HIT_MARKERS = (
    "没有找到",
    "未找到相关",
    "无法从知识库",
    "知识库中没有",
    "I don't know",
    "no relevant",
    "抱歉，我无法",
)

ERROR_ANSWER = "服务暂时不可用，请检查本机 Docker / 模型配置后重试。"
NO_HIT_ANSWER = "知识库中未命中相关内容，请换一种问法或补充文档后再试。"


def humanize_dify_error(raw: str | None) -> tuple[str, str]:
    """Return (user_answer, refuse_reason) for upstream failures."""
    text = (raw or "").strip() or "upstream_error"
    low = text.lower()
    if "provider openai does not exist" in low or (
        "openai" in low and "does not exist" in low
    ):
        return (
            "知识引擎的检索编排仍引用未安装的 OpenAI 模型（单路召回里的「系统推理模型」），"
            "与回答模型无关。请打开工作室 → 编排 → 知识检索 → 召回设置，"
            "将系统推理模型改为已配置模型，或切到 N 路召回并关闭 Rerank，保存发布后再试。",
            "kb: Provider openai does not exist（知识检索 single_retrieval_config 未改）",
        )
    if "401" in low or "unauthorized" in low or "invalid" in low and "api" in low:
        return (
            "应用访问密钥无效或无权访问。请核对本机 .env 中的应用密钥后重启接入服务。",
            f"auth: {text[:400]}",
        )
    if "timeout" in low:
        return ("调用知识引擎超时，请确认本机 Docker/模型响应后重试。", "timeout")
    if "dify_api_key not set" in low:
        return (
            "未配置应用访问密钥。请编辑本机 .env 后重启接入服务。",
            "DIFY_API_KEY not set",
        )
    # Keep generic shell + put detail in refuse_reason for UI/logs
    return (ERROR_ANSWER, text[:800])


def new_trace_id() -> str:
    # ULID-like sortable prefix + uuid (contract requires opaque non-empty id)
    return "01J" + uuid.uuid4().hex[:22].upper()


def normalize_citation(item: Any) -> dict[str, Any]:
    if not isinstance(item, dict):
        return {
            "doc_name": "unknown",
            "page": None,
            "snippet": str(item)[:500],
            "score": None,
        }
    doc = (
        item.get("document_name")
        or item.get("doc_name")
        or item.get("document_id")
        or item.get("dataset_name")
        or "unknown"
    )
    page = item.get("page") or item.get("page_number")
    try:
        page = int(page) if page is not None else None
    except (TypeError, ValueError):
        page = None
    snippet = item.get("content") or item.get("snippet") or item.get("segment") or ""
    if not isinstance(snippet, str):
        snippet = str(snippet)
    score = item.get("score")
    try:
        score = float(score) if score is not None else None
    except (TypeError, ValueError):
        score = None
    return {
        "doc_name": str(doc),
        "page": page,
        "snippet": snippet[:800],
        "score": score,
    }


def detect_no_hit(answer: str, citations: list[dict[str, Any]]) -> bool:
    if citations:
        return False
    text = (answer or "").strip()
    if not text:
        return True
    low = text.lower()
    for m in NO_HIT_MARKERS:
        if m.lower() in low:
            return True
    # very short refusals
    if len(text) < 8 and re.search(r"不知道|无|无结果", text):
        return True
    return False


def build_ask_response(
    *,
    trace_id: str,
    answer: str,
    raw_citations: list[Any] | None,
    dify_error: Optional[str] = None,
    force_status: Optional[Status] = None,
) -> dict[str, Any]:
    cites = [normalize_citation(c) for c in (raw_citations or [])]

    if force_status == "error" or dify_error:
        answer, reason = humanize_dify_error(dify_error)
        return {
            "trace_id": trace_id,
            "answer": answer,
            "citations": [],
            "status": "error",
            "refuse_reason": reason,
        }

    if force_status == "no_hit" or detect_no_hit(answer, cites):
        return {
            "trace_id": trace_id,
            "answer": NO_HIT_ANSWER if not answer.strip() else answer.strip(),
            "citations": [],
            "status": "no_hit",
            "refuse_reason": "no_hit",
        }

    return {
        "trace_id": trace_id,
        "answer": answer.strip(),
        "citations": cites,
        "status": "ok",
        "refuse_reason": None,
    }
