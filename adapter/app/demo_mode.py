"""Offline demo answers derived from golden-qa intents (BUDDY_DEMO_MODE=1)."""
from __future__ import annotations

from typing import Any, Optional


def match_demo(question: str) -> Optional[dict[str, Any]]:
    q = (question or "").strip()
    rules: list[tuple[tuple[str, ...], dict[str, Any]]] = [
        (
            ("XYZ-999", "火星", "ABC-000", "红烧肉"),
            {
                "answer": "知识库中没有相关内容。",
                "citations": [],
                "status": "no_hit",
            },
        ),
        (
            ("OBJ-1001", "冷运行"),
            {
                "answer": "OBJ-1001 冷运行参数：设定温度 -20℃，连续运行时长上限 2 小时。",
                "citations": [
                    {
                        "doc_name": "OBJ-1001-操作手册_V3.2.pdf",
                        "page": 5,
                        "snippet": "设定温度 -20℃，连续运行时长上限 2 小时",
                        "score": 0.92,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("OBJ-1002", "保养"),
            {
                "answer": "OBJ-1002 保养周期为每 90 天一次全面保养。",
                "citations": [
                    {
                        "doc_name": "OBJ-1002-维保规范.pdf",
                        "page": 2,
                        "snippet": "每 90 天进行一次全面保养",
                        "score": 0.88,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("紧急停机", "急停"),
            {
                "answer": "紧急停机后应先检查电源断开确认与急停按钮状态。",
                "citations": [
                    {
                        "doc_name": "安全操作总则_V1.pdf",
                        "page": 1,
                        "snippet": "电源断开确认、急停按钮状态",
                        "score": 0.85,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("关联", "OBJ-1001 与 OBJ-1002"),
            {
                "answer": "二者同属 A 产线：OBJ-1001 为上游主机，OBJ-1002 为维保单元。",
                "citations": [
                    {
                        "doc_name": "产线对象关系说明.md",
                        "page": 1,
                        "snippet": "上游主机与维保单元",
                        "score": 0.8,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("液压", "油温"),
            {
                "answer": "液压站正常工作油温范围为 35℃～55℃。",
                "citations": [
                    {
                        "doc_name": "液压站技术规格.pdf",
                        "page": 3,
                        "snippet": "油温范围：35℃～55℃",
                        "score": 0.9,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("冷却水", "压力"),
            {
                "answer": "冷却水进水压力下限为 0.2 MPa。",
                "citations": [
                    {
                        "doc_name": "公用工程参数表.xlsx",
                        "page": 1,
                        "snippet": "进水压力下限：0.2 MPa",
                        "score": 0.87,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("交接", "班次"),
            {
                "answer": "换班交接需填写班次、交接人、异常事项与待办。",
                "citations": [
                    {
                        "doc_name": "交接班制度_V2.pdf",
                        "page": 2,
                        "snippet": "班次、异常事项",
                        "score": 0.84,
                    }
                ],
                "status": "ok",
            },
        ),
        (
            ("OBJ-1003", "标定"),
            {
                "answer": "标定摘要：先做零点标定，再做满量程标定。",
                "citations": [
                    {
                        "doc_name": "OBJ-1003-标定手册.pdf",
                        "page": 4,
                        "snippet": "零点标定",
                        "score": 0.86,
                    }
                ],
                "status": "ok",
            },
        ),
    ]
    for keys, payload in rules:
        if any(k in q for k in keys):
            return payload
    return {
        "answer": "知识库中没有相关内容。",
        "citations": [],
        "status": "no_hit",
    }
