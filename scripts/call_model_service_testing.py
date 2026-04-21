#!/usr/bin/env python3
import runpy
import sys
from pathlib import Path


def main():
    repo_root = Path(__file__).resolve().parents[1]
    script = repo_root / "model" / "call_model_service_testing.py"
    sys.path.insert(0, str(script.parent))
    runpy.run_path(str(script), run_name="__main__")


if __name__ == "__main__":
    main()
