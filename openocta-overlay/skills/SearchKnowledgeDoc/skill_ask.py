#!/usr/bin/env python3
"""CLI wrapper: SearchKnowledgeDoc → adapter /v1/buddy/ask (or Demo)."""
from __future__ import annotations

import argparse
import json
import os
import sys
import urllib.request

ASK = os.getenv("BUDDY_ASK_URL", "http://127.0.0.1:8090/v1/buddy/ask")


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("question")
    p.add_argument("--object", default=None)
    args = p.parse_args()
    body: dict = {
        "question": args.question,
        "channel": "desktop",
        "user": {"id": "skill", "display_name": "SearchKnowledgeDoc"},
    }
    if args.object:
        body["meta"] = {"object_id": args.object}
    req = urllib.request.Request(
        ASK,
        data=json.dumps(body).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        out = resp.read().decode("utf-8")
    print(out)
    return 0


if __name__ == "__main__":
    sys.exit(main())
