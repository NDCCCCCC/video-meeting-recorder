#!/usr/bin/env python3
"""
Check Python dependencies for Record V2.

This script verifies that all required Python packages are installed
and outputs JSON for Go integration.

Usage:
    python scripts/check_python_deps.py

Output:
    JSON: {"ok": true, "python_version": "3.13.2", "packages": {"pptx": "1.0.2", ...}}
    On error: {"ok": false, "error": "..."}
"""

import sys
import json


def check_dependencies():
    """Check if all required packages are installed."""
    try:
        import pptx
        import PIL

        packages = {
            "pptx": pptx.__version__ if hasattr(pptx, "__version__") else "unknown",
            "pillow": PIL.__version__ if hasattr(PIL, "__version__") else "unknown",
        }

        result = {
            "ok": True,
            "python_version": f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
            "packages": packages,
        }
        return result

    except ImportError as e:
        return {
            "ok": False,
            "error": f"Missing package: {e}",
            "python_version": f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
        }
    except Exception as e:
        return {
            "ok": False,
            "error": str(e),
        }


def main():
    result = check_dependencies()
    print(json.dumps(result))
    sys.exit(0 if result["ok"] else 1)


if __name__ == "__main__":
    main()
