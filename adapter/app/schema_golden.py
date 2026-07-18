"""Validate golden-qa.yaml schema (L1)."""
from __future__ import annotations

from pathlib import Path
from typing import Any

import yaml

REQUIRED_CASE_KEYS = {"id", "question"}
VALID_STATUS = {"ok", "no_hit", "error"}


def validate_golden_qa_doc(doc: Any) -> dict[str, Any]:
    errors: list[str] = []
    if not isinstance(doc, dict):
        return {"ok": False, "errors": ["root must be mapping"], "count": 0}

    if "version" not in doc:
        errors.append("missing version")
    cases = doc.get("cases")
    if not isinstance(cases, list) or not cases:
        errors.append("cases must be non-empty list")
        return {"ok": False, "errors": errors, "count": 0}

    ids: set[str] = set()
    for i, case in enumerate(cases):
        if not isinstance(case, dict):
            errors.append(f"cases[{i}] not object")
            continue
        missing = REQUIRED_CASE_KEYS - set(case.keys())
        if missing:
            errors.append(f"cases[{i}] missing {sorted(missing)}")
        cid = case.get("id")
        if isinstance(cid, str):
            if cid in ids:
                errors.append(f"duplicate id {cid}")
            ids.add(cid)
        st = case.get("expect_status")
        if st is not None and st not in VALID_STATUS:
            errors.append(f"cases[{i}] bad expect_status={st}")
        if "expect_contains" in case and not isinstance(case["expect_contains"], list):
            errors.append(f"cases[{i}] expect_contains must be list")

    return {
        "ok": len(errors) == 0,
        "errors": errors,
        "count": len(cases),
        "ids": sorted(ids),
    }


def validate_golden_qa_path(path: str | Path) -> dict[str, Any]:
    p = Path(path)
    if not p.is_file():
        return {"ok": False, "errors": [f"file not found: {p}"], "count": 0}
    data = yaml.safe_load(p.read_text(encoding="utf-8"))
    return validate_golden_qa_doc(data)
