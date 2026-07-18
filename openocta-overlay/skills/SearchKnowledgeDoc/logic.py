"""SearchKnowledgeDoc · pure logic (callable by OpenOcta Skill or tests)."""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any, Optional

_ROOT = Path(__file__).resolve().parents[3]  # mybuddy-v01
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from adapter.app.mapping import (  # noqa: E402
    build_ask_response,
    normalize_citation,
    new_trace_id,
)


def compose_query(question: str, object_id: Optional[str] = None) -> str:
    q = (question or "").strip()
    if object_id:
        return f"[object_id={object_id}] {q}"
    return q


def map_dify_to_contract(
    *,
    answer: str,
    retriever_resources: list[Any] | None,
    error: Optional[str] = None,
    trace_id: Optional[str] = None,
) -> dict[str, Any]:
    tid = trace_id or new_trace_id()
    return build_ask_response(
        trace_id=tid,
        answer=answer or "",
        raw_citations=retriever_resources or [],
        dify_error=error,
        force_status="error" if error else None,
    )


def refuse_copy(status: str) -> str:
    if status == "no_hit":
        return "知识库中未命中相关内容，请换一种问法或补充文档后再试。"
    if status == "error":
        return "服务暂时不可用，请检查本机 Docker / 模型配置后重试。"
    return ""


def skill_result_json(payload: dict[str, Any]) -> str:
    cites = []
    for c in payload.get("citations") or []:
        if isinstance(c, dict) and "doc_name" in c:
            cites.append(c)
        else:
            cites.append(normalize_citation(c))
    out = {**payload, "citations": cites}
    return json.dumps(out, ensure_ascii=False)
