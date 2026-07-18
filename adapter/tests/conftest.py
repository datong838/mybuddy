from pathlib import Path
import sys
import os

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

# Unit tests must not inherit shell BUDDY_DEMO_MODE
os.environ.pop("BUDDY_DEMO_MODE", None)
