import sys
from pathlib import Path

# mybuddy-v01 on path
ROOT = Path(__file__).resolve().parents[4]
sys.path.insert(0, str(ROOT))
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
