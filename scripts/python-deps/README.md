# Python Dependencies Management

This directory contains Python scripts used by the Record V2 system. Dependencies are managed using [uv](https://github.com/astral-sh/uv).

## Prerequisites

Install `uv` (fast Python package manager):

```bash
# On Linux/macOS
curl -LsSf https://astral.sh/uv/install.sh | sh

# On Windows
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"

# Or via pip
pip install uv
```

## Installation

Run from project root:

```bash
# Sync dependencies (creates .venv if needed)
uv sync

# Or install to system Python
uv pip install -e .
```

## Usage

Run Python scripts with uv:

```bash
# Using uv run (recommended - uses project venv)
uv run python scripts/create_pptx.py output.pptx image1.jpg image2.jpg

# Or activate venv manually
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
python scripts/create_pptx.py output.pptx image1.jpg image2.jpg
```

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| python-pptx | ^1.0.2 | PowerPoint file generation and manipulation |
| Pillow | ^10.0.0 | Image processing for slide extraction |

## Scripts

| Script | Purpose | Go Service |
|--------|---------|------------|
| `create_pptx.py` | Create PPTX from frame images | PPTXGenerator |
| `extract_slides.py` | Extract slide images from PPTX | SlideExtractor |
| `merge_slides.py` | Merge slides from multiple PPTX | PPTMergeService |

## Troubleshooting

**Missing dependencies:**
```bash
uv sync --reinstall
```

**Wrong Python version:**
```bash
uv python install 3.13  # or 3.11, 3.12
uv python pin 3.13
```

**Check installation:**
```bash
uv run python -c "import pptx; print(pptx.__version__)"
```
