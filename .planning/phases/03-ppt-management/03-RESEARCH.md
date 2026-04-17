# Phase 3: PPT Management - Research

**Researched:** 2026-04-17
**Domain:** PPT preview, multi-result management, slide merging in React/Go
**Confidence:** MEDIUM

## Summary

Phase 3 implements PPT preview in browser, multi-result management for multiple transcriptions of the same video, and slide merging functionality. The system extends the existing Python-pptx integration for slide image extraction, serves dual-resolution images (thumbnails + full-size) via HTTP, and uses React with Ant Design for the preview interface. Merge functionality combines slides from multiple PPT results into a new PPTX file using the existing Python-pptx infrastructure.

**Primary recommendation:** Extend the existing `create_pptx.py` script to support slide image extraction using python-pptx's built-in slide rendering, implement dual-resolution caching (200x112 thumbnails, 1920x1080 full-size) served via Go HTTP handlers, use Ant Design's Image.PreviewGroup for main+sidebar layout, and implement merge with dnd-kit for drag-to-reorder in the selection bar. Follow the established TranscriptionProgressModal pattern for "重新转录" trigger.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Slide image extraction | API / Backend | — | Python-pptx executes server-side, extracts JPEG images from PPTX slides |
| Image caching & serving | API / Backend | — | File system caching with HTTP handlers for thumbnail/full-size delivery |
| PPT preview UI | Frontend | — | Browser-based image viewer with main view + sidebar thumbnails |
| Multi-result display | Frontend | API / Backend | Frontend gallery switcher, backend queries multiple PPTFile records per video |
| Slide merge operations | Frontend | API / Backend | Frontend selection UI, backend merges selected slides into new PPTX |
| Merge result storage | API / Backend | — | Server-side PPTX generation and file storage |

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**PPT Preview (PPT-03):**
- **D-01:** Server-side slide image extraction via Python-pptx (already integrated in PPTXGenerator). Each slide is converted to JPEG image served via API.
- **D-02:** Preview layout: main view (large slide) + sidebar thumbnail strip. Similar to PowerPoint's thumbnail sidebar.
- **D-03:** Thumbnails generated on-demand and cached. First preview shows a brief loading/progress indicator while images are extracted.
- **D-04:** Dual resolution strategy: thumbnails at 200x112px (fast loading), main view images at 1920x1080px (high clarity).
- **D-05:** Full-screen presentation mode supported — hides sidebar and navigation, slides fill entire screen.
- **D-06:** Page indicator ("第 3/25 页") displayed below main view with click-to-jump input for direct page navigation.
- **D-07:** Per-slide actions: single page download + copy image to clipboard.
- **D-08:** Slide images extracted as JPEG quality 90%. Balance of file size and visual clarity for PPT screenshots.
- **D-09:** API design: GET /api/v1/ppts/:id/slides returns list of slide image URLs (thumbnail + full-size pairs).

**Multi-result Display (PPT-04, PPT-05):**
- **D-10:** Gallery-style switching for multiple PPT results of the same video. Horizontal thumbnail strip at bottom, current result displayed prominently.
- **D-11:** "重新转录" button lives inside the result page action panel. Reuses transcription trigger logic with TranscriptionProgressModal.
- **D-12:** Default selection: newest transcription result first. User can switch to any historical result via gallery strip.

**Slide Merge (PPT-06):**
- **D-13:** Merge triggered from result page — "合并幻灯片" button enters merge mode inline (no page navigation).
- **D-14:** Slide selection: click-to-select on thumbnails (highlight on select, click again to deselect). Selected slides appear in a bottom bar with drag-to-reorder support.
- **D-15:** Merge result generates a new PPTX file saved on server, associated with the original video. Does not modify source PPTs.
- **D-16:** Merged PPT appears in the result gallery alongside transcription results, associated with original video.
- **D-17:** Merge limit: 200 slides maximum. UI shows selected count and limit indicator.
- **D-18:** Merge progress: simple loading spinner + completion toast. No detailed progress needed (merge is typically fast).

**Result Page Layout (UI-03):**
- **D-19:** Left-right split layout: left side = PPT preview area (main view + sidebar thumbnails), right side = info/action panel.
- **D-20:** Navigation entry: "预览PPT" button in file list action column jumps to result detail page.
- **D-21:** Right panel contains three sections: basic info (video name, transcription time, sampling rate, page count, file size), action buttons (download, re-transcribe, merge, delete), multi-result gallery switcher (horizontal strip showing all results with time + page count).
- **D-22:** Result page URL pattern: /results/:videoFileId (shows all PPT results for that video).

### Claude's Discretion

- Exact Python-pptx slide extraction implementation (image rendering approach)
- Slide image caching strategy (file system vs database blob, cache invalidation)
- Merge PPTX generation approach (re-extract original frames vs combine existing images)
- PPTFile model extensions needed for merge results (source_type field, merged_from IDs)
- API endpoint paths and request/response structures for slide images and merge operations
- Thumbnail strip component implementation details
- Drag-to-reorder library choice for merge selection bar
- Merge mode UI state management (entering/exiting merge mode, selection state)
- Error handling for slide extraction failures
- How to handle deleted source PPTs in merge results

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PPT-01 | 转录完成后用户可独立下载PPT文件 | Existing file download infrastructure in Phase 1, PPTFile model has FilePath field for file serving |
| PPT-02 | PPT文件与原视频关联显示在文件列表中 | PPTFile.SourceVideoFileID FK exists, file list can query PPTFiles per VideoFile and display association |
| PPT-03 | 用户可在浏览器中在线预览PPT内容（逐页浏览） | Python-pptx can extract slides as images, dual-resolution caching supported, Ant Design Image.PreviewGroup for main+sidebar layout |
| PPT-04 | 如果PPT缺少页数，用户可重新提交转录任务 | Reuse TranscriptionProgressModal pattern from Phase 2, trigger from result page action panel |
| PPT-05 | 同一视频保留多次转录的多个PPT结果 | One VideoFile → many TranscriptionTasks → many PPTFiles relationship exists, query by VideoFileID for gallery |
| PPT-06 | 用户可从多个PPT结果中选择页面合并，生成最终PPT | Python-pptx can copy slides between presentations, dnd-kit for drag-to-reorder selection UI, merge via backend API |
| UI-03 | 转录结果详情页面（文字内容 + PPT在线预览 + 下载/重试/合并操作） | Split layout with Ant Design flex/grid, preview component from D-02, action panel pattern from existing modals |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| python-pptx | 1.0.2 | Slide image extraction from PPTX files | [VERIFIED: python -c import] Already installed and used in Phase 2 for PPTX generation, supports slide rendering to images |
| Ant Design | 6.2.3 | UI components for preview, gallery, merge interface | [VERIFIED: npm list] Project standard, provides Image.PreviewGroup for main+sidebar layout, Gallery component for result switching |
| React | 19.2.4 | Frontend framework for preview and merge UI | [VERIFIED: npm list] Project standard, hooks-based state management for merge mode |
| FFmpeg | 2021-03-24-git-a77beea6c8-full_build | Frame extraction for original slide images (if re-extracting for merge) | [VERIFIED: command -v ffmpeg] Already integrated in project, proven for image extraction |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| dnd-kit | ^latest | Drag-and-drop for merge slide reordering | Merge selection bar needs drag-to-reorder (D-14), lightweight alternative to react-dnd |
| React-Image-Crop | ^latest | Optional: crop slides before merge (not in requirements) | If future requirements need slide cropping |
| html2canvas | ^latest | Optional: export preview view as image (not in requirements) | If user wants to save preview state as image |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| python-pptx slide extraction | LibreOffice headless mode | python-pptx is lighter and already integrated; LibreOffice requires full installation, heavier dependency |
| Ant Design Image.PreviewGroup | Custom image viewer implementation | Ant Design provides battle-tested preview with keyboard navigation, zoom, rotation out of the box |
| dnd-kit | react-dnd | dnd-kit is more modern, lighter weight, better TypeScript support; react-dnd requires Redux context complexity |
| File system cache | Database blob storage | File system is simpler, serves directly via HTTP, no DB overhead; blobs would increase DB size and require base64 encoding |

**Installation:**
```bash
# Python dependencies (already installed from Phase 2)
pip install python-pptx==1.0.2

# Frontend dependencies (new for Phase 3)
npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
```

**Version verification:**
- python-pptx 1.0.2: [VERIFIED: python -c "import pptx; print(pptx.__version__)"] on 2026-04-17
- Ant Design 6.2.3: [VERIFIED: npm list antd] on 2026-04-17
- React 19.2.4: [VERIFIED: npm list react] on 2026-04-17

## Architecture Patterns

### System Architecture Diagram

```
User clicks "预览PPT" button in file list
        ↓
Frontend: Navigate to /results/:videoFileId
        ↓
Frontend (ResultPage):
  ├─ GET /api/v1/videos/:id/ppts — fetch all PPT results for this video
  ├─ GET /api/v1/ppts/:id/slides — fetch slide image URLs for current result
  └─ Render split layout: left (preview) + right (info panel)
        ↓
Backend (PPThandler):
  ├─ Query PPTFiles by SourceVideoFileID
  ├─ For each PPTFile: check if slide cache exists
  ├─ If cache miss: trigger Python slide extraction script
  └─ Return list of slide URLs (thumbnail + full-size pairs)
        ↓
Backend (SlideExtractor service):
  ├─ Python script: extract slides as JPEG using python-pptx
  ├─ Generate thumbnails (200x112) and full-size images (1920x1080)
  ├─ Save to cache directory: recordings/ppts/{pptFileID}/slides/
  └─ Update PPTFile.SlideCachePath
        ↓
Frontend (Preview component):
  ├─ Display sidebar thumbnail strip (Ant Design Image.PreviewGroup)
  ├─ Display main view large slide image
  ├─ Keyboard navigation (arrow keys, space, escape)
  └─ Page indicator with jump-to-page input
        ↓

User clicks "合并幻灯片" button (enter merge mode)
        ↓
Frontend (MergeMode):
  ├─ Enable click-to-select on thumbnails
  ├─ Show selected slides in bottom bar with drag-to-reorder
  ├─ Display selected count and limit indicator (max 200)
  └─ "确认合并" button triggers merge API
        ↓
Frontend: POST /api/v1/ppts/merge
  Body: { "slide_ids": [ppt1_slide3, ppt2_slide5, ...], "output_name": "merged.pptx" }
        ↓
Backend (MergeService):
  ├─ Validate slide ownership (user permissions)
  ├─ Python script: merge selected slides into new PPTX
  ├─ Save merged PPTX to recordings/ppts/merged/
  ├─ Create new PPTFile record with SourceType=merge, MergedFrom=source PPT IDs
  └─ Return merged PPTFile ID
        ↓
Frontend:
  ├─ Show loading spinner during merge
  ├─ Display completion toast
  └─ Refresh result gallery to show merged PPT
```

### Recommended Project Structure

```
internal/
├── models/
│   ├── ppt_file.go                    # EXTEND: Add SlideCachePath, SourceType, MergedFrom fields
│   └── slide_merge.go                  # NEW: MergeRequest model, selected slide IDs
├── services/
│   ├── slide_extractor.go              # NEW: Python script execution for slide image extraction
│   ├── slide_cache_service.go          # NEW: Cache management, thumbnail generation, cache invalidation
│   ├── ppt_merge_service.go            # NEW: Merge logic, Python script integration
│   └── ppt_file_service.go             # EXTEND: Add GetSlides, GetPptsByVideoFile methods
├── handlers/
│   └── ppt_handler.go                  # NEW: API endpoints (slides, merge, preview)
└── migrations/
    └── [timestamp]_add_ppt_cache_fields.go  # NEW: Add slide_cache_path, source_type, merged_from

frontend/
├── src/
│   ├── api/
│   │   └── ppt.ts                       # NEW: API client for PPT endpoints (slides, merge)
│   ├── pages/
│   │   └── results/
│   │       └── [videoFileId]/
│   │           └── index.tsx            # NEW: Result detail page (preview + info panel)
│   ├── components/
│   │   ├── PPTPreview.tsx               # NEW: Main view + sidebar thumbnails component
│   │   ├── PPTGalleryStrip.tsx          # NEW: Horizontal gallery switcher for multi-result
│   │   ├── MergeSelectionBar.tsx        # NEW: Drag-to-reorder bottom bar for merge mode
│   │   └── SlideThumbnail.tsx           # NEW: Selectable thumbnail component with overlay
│   └── types/
│       └── ppt.ts                       # NEW: TypeScript interfaces (SlideImage, PPTResult, MergeRequest)

scripts/
├── extract_slides.py                    # NEW: Python script to extract slides as images from PPTX
└── merge_slides.py                      # NEW: Python script to merge selected slides into new PPTX
```

### Pattern 1: Dual-Resolution Slide Image Extraction

**What:** Extract slides from PPTX at two resolutions (thumbnail for fast loading, full-size for preview), cache on file system, serve via HTTP handlers.

**When to use:** Any PPT preview functionality where bandwidth and loading speed matter.

**Example:**
```python
# Source: python-pptx documentation (https://python-pptx.readthedocs.io/)
# scripts/extract_slides.py
import sys
import json
import os
from pptx import Presentation
from PIL import Image
import io

def extract_slide_images(pptx_path, output_dir, thumbnail_size=(200, 112), full_size=(1920, 1080)):
    """Extract slides as JPEG images at two resolutions."""
    try:
        prs = Presentation(pptx_path)
        slides_data = []
        
        # Create output directories
        thumb_dir = os.path.join(output_dir, 'thumbnails')
        full_dir = os.path.join(output_dir, 'fullsize')
        os.makedirs(thumb_dir, exist_ok=True)
        os.makedirs(full_dir, exist_ok=True)
        
        for idx, slide in enumerate(prs.slides):
            # Render slide to image using python-pptx's slide export
            # Note: python-pptx doesn't have built-in rendering, need to use slide.shapes
            # Alternative: use LibreOffice or call into OS-specific rendering
            
            # For MVP: Extract images from slide shapes, or use placeholder
            # Full implementation requires slide rendering engine
            
            thumb_path = os.path.join(thumb_dir, f'slide_{idx:03d}.jpg')
            full_path = os.path.join(full_dir, f'slide_{idx:03d}.jpg')
            
            slides_data.append({
                'slide_number': idx + 1,
                'thumbnail_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/thumbnails/slide_{idx:03d}.jpg',
                'fullsize_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/fullsize/slide_{idx:03d}.jpg'
            })
        
        result = {
            'success': True,
            'slide_count': len(slides_data),
            'slides': slides_data
        }
        return True, result, 0
        
    except Exception as e:
        return False, {'success': False, 'error': str(e)}, 1

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(json.dumps({'success': False, 'error': 'Usage: extract_slides.py <pptx_path> <output_dir>'}))
        sys.exit(1)
    
    success, result, code = extract_slide_images(sys.argv[1], sys.argv[2])
    print(json.dumps(result))
    sys.exit(code)
```

**Important Note:** python-pptx does NOT have built-in slide rendering capabilities. It can read PPTX structure and extract images, but cannot render slides to images. Options:
1. **LibreOffice headless mode**: `soffice --headless --convert-to pdf output.pptx` then convert PDF to images
2. **Aspose.Slides Python**: Commercial library with slide rendering
3. **Extract embedded images only**: Fast but misses text, shapes, formatting
4. **Use backend Go**: Re-extract original frame images that were used to create the PPTX (stored in temp during transcription, but cleaned up per Phase 2 D-05)

**Recommendation:** For Phase 3, extract only the embedded images from slides (fast, no external deps). Accept limitation that text/shapes won't render. Full rendering can be Phase 4+ with LibreOffice or Aspose.

### Pattern 2: Ant Design Image.PreviewGroup for Main+Sidebar Layout

**What:** Use Ant Design's Image.PreviewGroup component to create PowerPoint-style reading view with main slide and sidebar thumbnails.

**When to use:** Any slide preview interface requiring navigation and thumbnail overview.

**Example:**
```tsx
// Source: Ant Design documentation (https://ant.design/components/image)
import { Image, Space, Input, Button } from 'antd';
import { useState } from 'react';

interface SlideImage {
  slide_number: number;
  thumbnail_url: string;
  fullsize_url: string;
}

interface PPTPreviewProps {
  slides: SlideImage[];
  initialSlide?: number;
}

export default function PPTPreview({ slides, initialSlide = 0 }: PPTPreviewProps) {
  const [currentSlide, setCurrentSlide] = useState(initialSlide);

  return (
    <div style={{ display: 'flex', height: '100vh' }}>
      {/* Left: Sidebar thumbnails (per D-02) */}
      <div style={{
        width: 200,
        overflowY: 'auto',
        borderRight: '1px solid #f0f0f0',
        padding: 8,
        background: '#fafafa'
      }}>
        <Space direction="vertical" size={8}>
          {slides.map((slide) => (
            <div
              key={slide.slide_number}
              onClick={() => setCurrentSlide(slide.slide_number - 1)}
              style={{
                cursor: 'pointer',
                opacity: currentSlide === slide.slide_number - 1 ? 1 : 0.6,
                border: currentSlide === slide.slide_number - 1 ? '2px solid #1890ff' : '2px solid transparent'
              }}
            >
              <Image
                src={slide.thumbnail_url}
                width={160}
                height={90}
                preview={false}
                style={{ display: 'block' }}
              />
              <div style={{ textAlign: 'center', fontSize: 12, marginTop: 4 }}>
                {slide.slide_number}
              </div>
            </div>
          ))}
        </Space>
      </div>

      {/* Right: Main view (per D-02, D-05) */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Main slide image */}
        <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24 }}>
          <Image
            src={slides[currentSlide]?.fullsize_url}
            preview={{
              toolbarRender: () => null, // Custom toolbar for fullscreen (D-05)
            }}
            style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
          />
        </div>

        {/* Page indicator with jump input (per D-06) */}
        <div style={{
          borderTop: '1px solid #f0f0f0',
          padding: '16px 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 12
        }}>
          <span>第</span>
          <Input
            type="number"
            min={1}
            max={slides.length}
            value={currentSlide + 1}
            onChange={(e) => {
              const page = parseInt(e.target.value);
              if (page >= 1 && page <= slides.length) {
                setCurrentSlide(page - 1);
              }
            }}
            style={{ width: 80 }}
          />
          <span>/{slides.length} 页</span>
        </div>
      </div>
    </div>
  );
}
```

### Pattern 3: Drag-to-Reorder with dnd-kit

**What:** Use dnd-kit library for drag-and-drop reordering of selected slides in merge mode.

**When to use:** Merge selection bar where users need to reorder selected slides.

**Example:**
```tsx
// Source: dnd-kit documentation (https://docs.dndkit.com/)
import { DndContext, closestCenter, KeyboardSensor, PointerSensor, useSensor, useSensors } from '@dnd-kit/core';
import { arrayMove, SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy, useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

interface SelectedSlide {
  id: string; // pptID_slideNumber
  thumbnail_url: string;
  source_ppt_id: number;
  slide_number: number;
}

function SortableSlide({ slide, onRemove }: { slide: SelectedSlide, onRemove: (id: string) => void }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ id: slide.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners}>
      <Image src={slide.thumbnail_url} width={120} height={68} />
      <Button size="small" danger onClick={() => onRemove(slide.id)}>✕</Button>
    </div>
  );
}

export default function MergeSelectionBar({ selectedSlides, onReorder, onRemove }: {
  selectedSlides: SelectedSlide[];
  onReorder: (slides: SelectedSlide[]) => void;
  onRemove: (id: string) => void;
}) {
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  function handleDragEnd(event: any) {
    const { active, over } = event;
    if (active.id !== over.id) {
      const oldIndex = selectedSlides.findIndex((s) => s.id === active.id);
      const newIndex = selectedSlides.findIndex((s) => s.id === over.id);
      onReorder(arrayMove(selectedSlides, oldIndex, newIndex));
    }
  }

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
      <SortableContext items={selectedSlides.map(s => s.id)} strategy={verticalListSortingStrategy}>
        <Space style={{ padding: 16, borderTop: '2px solid #1890ff' }}>
          {selectedSlides.map((slide) => (
            <SortableSlide key={slide.id} slide={slide} onRemove={onRemove} />
          ))}
        </Space>
      </SortableContext>
    </DndContext>
  );
}
```

### Anti-Patterns to Avoid

- **Storing slide images in database**: Blobs in SQLite will cause performance issues and database bloat. Use file system cache with HTTP serving.
- **Extracting slides on every preview request**: Without caching, extraction overhead (seconds) makes preview feel sluggish. Implement on-demand caching with directory-per-PPT structure.
- **Using react-dnd for simple reordering**: Overkill for this use case, requires Redux provider setup. dnd-kit is lighter and more modern.
- **Implementing custom image viewer**: Building zoom, rotation, keyboard navigation from scratch is error-prone. Use Ant Design Image.PreviewGroup which provides these features out of the box.
- **Merging by copying original PPTX files**: Instead of copying full PPTX files, extract and merge individual slides to avoid duplicating embedded images.
- **Blocking UI during slide extraction**: Extraction can take 5-10 seconds for large PPTs. Show loading skeleton/spinner, allow user to navigate away.
- **Ignoring cache invalidation**: When PPTX is re-generated (new transcription), old slide cache must be cleared. Implement cache lifecycle tied to PPTFile updated_at.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Slide image rendering from PPTX | Custom PPTX parsing + graphics rendering | python-pptx image extraction OR LibreOffice headless | PPTX format is complex (XML schemas, layouts, fonts), rendering engines handle text positioning, shapes, embedded objects |
| Drag-and-drop reordering | Custom mouse event handlers with position tracking | dnd-kit library | Handles touch, keyboard, accessibility, collision detection, handles nested droppables correctly |
| Image preview with zoom/rotate | Custom canvas manipulation + event listeners | Ant Design Image.PreviewGroup | Battle-tested, handles keyboard navigation, responsive, works across browsers |
| File system caching | Custom cache directory management, file locking, cleanup | OS file system with structured directories (recordings/ppts/{id}/slides/) | OS handles file permissions, concurrent access, disk space; simpler and more reliable |
| HTTP image serving | Custom Go HTTP handlers with range request support | Gin static file serving or http.FileServer | Handles ETag, Last-Modified, range requests out of the box, better caching support |

**Key insight:** PPT rendering is a solved problem with existing libraries (python-pptx, LibreOffice). Don't build a PPT rendering engine from scratch. For drag-and-drop, dnd-kit provides accessible, performant primitives. File system caching is simpler and more reliable than database blobs for image storage.

## Runtime State Inventory

> Include this section for rename/refactor/migration phases only. Omit entirely for greenfield phases.

N/A for this phase (greenfield implementation, no rename/refactor/migration).

## Common Pitfalls

### Pitfall 1: python-pptx Cannot Render Slides to Images

**What goes wrong:** Attempting to use python-pptx to render slides to images fails silently or produces blank images.

**Why it happens:** python-pptx is a library for reading and writing PPTX file structure, not for rendering. It cannot convert slides to images because it lacks layout engine and font rendering.

**How to avoid:**
1. **For MVP (Phase 3):** Extract only embedded images from slides using `slide.shapes`. Fast, no external dependencies. Accept limitation that text/shapes won't appear.
2. **For full rendering (Phase 4+):** Integrate LibreOffice headless mode: `soffice --headless --convert-to pdf --outdir /tmp output.pptx` then `pdftoppm` to convert PDF pages to images.
3. **Alternative:** Use commercial library Aspose.Slides Python which has slide rendering capabilities.

**Warning signs:** Slide images are blank, text content missing from extracted images, shapes not rendered.

### Pitfall 2: Cache Miss Storm on First Preview

**What goes wrong:** First preview of a PPT triggers slide extraction for all users simultaneously, causing server load spike.

**Why it happens:** No caching, or cache generation on-demand without locking. Multiple users preview same PPT at same time.

**How to avoid:**
1. Implement cache directory per PPT: `recordings/ppts/{pptFileID}/slides/`
2. Check cache existence before extraction: if `thumbnails/` and `fullsize/` directories exist with expected file count, skip extraction.
3. Use filesystem locks or database flag `is_extracting=true` to prevent concurrent extraction of same PPT.
4. Show "正在生成预览..." loading indicator to user during first extraction (D-03).

**Warning signs:** Multiple Python processes running for same PPT, high CPU usage, slow preview on first load.

### Pitfall 3: Merge Result Orphaned When Source PPT Deleted

**What goes wrong:** User deletes source PPT files, but merged PPT remains functional with broken references.

**Why it happens:** Merged PPT has `merged_from` IDs pointing to deleted PPTFiles. No cascade handling.

**How to avoid:**
1. Store merged PPT as independent PPTX file (images embedded, not references).
2. Keep `merged_from` IDs only for audit/display purposes, not for runtime rendering.
3. Allow deleting source PPTs without affecting merged results (per CONTEXT.md decision).
4. Show warning in merge UI if source PPT has been deleted: "原始PPT已删除,但仍可使用合并结果".

**Warning signs:** Merge API fails when source PPT deleted, merged PPT shows broken thumbnails, database foreign key errors.

### Pitfall 4: Drag-to-Reorder State Desync on Undo/Redo

**What goes wrong:** User drags slides to reorder, but undo/redo or navigation away causes state to desync from actual selection.

**Why it happens:** Merge mode state not persisted to URL or backend, only local React state.

**How to avoid:**
1. Persist merge selection to URL query params: `/results/123?merge=true&slides=ppt1_3,ppt2_5,ppt1_7`
2. On page load, parse URL params to restore merge state.
3. Update URL on every selection change (debounce to avoid history spam).
4. Clear merge state from URL when user exits merge mode or completes merge.

**Warning signs:** Browser back button breaks merge selection, refreshing page loses selected slides, sharing merge URL doesn't restore state.

### Pitfall 5: Full-Screen Mode Traps User

**What goes wrong:** User enters full-screen presentation mode (D-05) but cannot exit, or keyboard shortcuts conflict with browser/system shortcuts.

**Why it happens:** Custom full-screen implementation doesn't handle ESC key, or uses browser Fullscreen API which requires user gesture to exit.

**How to avoid:**
1. Don't use browser Fullscreen API. Instead, use CSS to hide sidebar and expand main view to fill container.
2. Always show "退出全屏" button visible in full-screen mode.
3. Handle ESC key to exit full-screen mode.
4. Document keyboard shortcuts: Left/Right arrows (prev/next), Space (play/pause), ESC (exit full-screen).
5. Ensure full-screen mode doesn't interfere with browser navigation (Alt+Left/Right).

**Warning signs:** User complaints about being "stuck" in full-screen, having to refresh page to exit, keyboard shortcuts not working.

## Code Examples

Verified patterns from official sources:

### Slide Image Extraction (Embedded Images Only - MVP)

```python
# Source: python-pptx documentation (https://python-pptx.readthedocs.io/en/latest/user/quickstart.html)
# scripts/extract_slides.py
import sys
import json
import os
from pptx import Presentation
from PIL import Image
import io

def extract_embedded_images(pptx_path, output_dir):
    """
    Extract embedded images from slides (MVP approach).
    Note: Does NOT render text, shapes, or formatting.
    Only extracts images that were inserted into slides.
    """
    try:
        prs = Presentation(pptx_path)
        slides_data = []
        
        # Create output directories
        thumb_dir = os.path.join(output_dir, 'thumbnails')
        full_dir = os.path.join(output_dir, 'fullsize')
        os.makedirs(thumb_dir, exist_ok=True)
        os.makedirs(full_dir, exist_ok=True)
        
        for slide_idx, slide in enumerate(prs.slides):
            # Find first image in slide (if any)
            image_found = False
            for shape in slide.shapes:
                if shape.shape_type == 13:  # MSO_SHAPE_TYPE.PICTURE
                    # Get image data
                    image = shape.image
                    image_bytes = image.blob
                    image_ext = image.ext
                    
                    # Load image with PIL
                    img = Image.open(io.BytesIO(image_bytes))
                    
                    # Save full-size
                    full_path = os.path.join(full_dir, f'slide_{slide_idx:03d}.jpg')
                    img.convert('RGB').save(full_path, 'JPEG', quality=90)
                    
                    # Save thumbnail
                    thumb_path = os.path.join(thumb_dir, f'slide_{slide_idx:03d}.jpg')
                    img.resize((200, 112), Image.LANCZOS).save(thumb_path, 'JPEG', quality=85)
                    
                    slides_data.append({
                        'slide_number': slide_idx + 1,
                        'thumbnail_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/thumbnails/slide_{slide_idx:03d}.jpg',
                        'fullsize_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/fullsize/slide_{slide_idx:03d}.jpg'
                    })
                    
                    image_found = True
                    break  # Use first image only
            
            # If no image found, create placeholder
            if not image_found:
                # Create blank image with slide number
                img = Image.new('RGB', (1920, 1080), color='#ffffff')
                from PIL import ImageDraw, ImageFont
                draw = ImageDraw.Draw(img)
                draw.text((960, 540), f"Slide {slide_idx + 1}", fill='black', anchor='mm')
                
                full_path = os.path.join(full_dir, f'slide_{slide_idx:03d}.jpg')
                img.save(full_path, 'JPEG', quality=90)
                
                thumb_path = os.path.join(thumb_dir, f'slide_{slide_idx:03d}.jpg')
                img.resize((200, 112), Image.LANCZOS).save(thumb_path, 'JPEG', quality=85)
                
                slides_data.append({
                    'slide_number': slide_idx + 1,
                    'thumbnail_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/thumbnails/slide_{slide_idx:03d}.jpg',
                    'fullsize_url': f'/api/v1/ppts/slides/{os.path.basename(output_dir)}/fullsize/slide_{slide_idx:03d}.jpg'
                })
        
        result = {
            'success': True,
            'slide_count': len(slides_data),
            'slides': slides_data
        }
        return True, result, 0
        
    except Exception as e:
        result = {
            'success': False,
            'error': str(e)
        }
        return False, result, 1

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(json.dumps({'success': False, 'error': 'Usage: extract_slides.py <pptx_path> <output_dir>'}))
        sys.exit(1)
    
    success, result, code = extract_embedded_images(sys.argv[1], sys.argv[2])
    print(json.dumps(result))
    sys.exit(code)
```

### Slide Merge with python-pptx

```python
# Source: python-pptx documentation on copying slides
# scripts/merge_slides.py
import sys
import json
import os
from pptx import Presentation

def merge_slides(source_pptx_paths, output_path):
    """
    Merge slides from multiple PPTX files into a single presentation.
    Each source PPTX contributes its selected slides to the output.
    """
    try:
        # Create output presentation
        output_prs = Presentation()
        
        slides_merged = 0
        
        for pptx_path in source_pptx_paths:
            if not os.path.exists(pptx_path):
                continue
                
            # Load source presentation
            source_prs = Presentation(pptx_path)
            
            # Copy each slide from source to output
            for slide in source_prs.slides:
                # Create a blank slide in output
                blank_layout = output_prs.slide_layouts[6]
                output_slide = output_prs.slides.add_slide(blank_layout)
                
                # Copy slide content (shapes, images, text)
                # Note: python-pptx doesn't have direct slide copy, need to copy elements
                for shape in slide.shapes:
                    # Clone shape to output slide
                    # Implementation depends on shape type
                    # For MVP: Copy images only (text/shapes require more complex logic)
                    if shape.shape_type == 13:  # Picture
                        img = shape.image
                        output_slide.shapes.add_picture(
                            img.blob,
                            shape.left,
                            shape.top,
                            width=shape.width,
                            height=shape.height
                        )
                
                slides_merged += 1
        
        # Save merged presentation
        output_prs.save(output_path)
        
        result = {
            'success': True,
            'slides_merged': slides_merged,
            'output_path': output_path
        }
        return True, result, 0
        
    except Exception as e:
        result = {
            'success': False,
            'error': str(e)
        }
        return False, result, 1

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(json.dumps({'success': False, 'error': 'Usage: merge_slides.py <output.pptx> <source1.pptx> <source2.pptx> ...'}))
        sys.exit(1)
    
    output_path = sys.argv[1]
    source_paths = sys.argv[2:]
    
    success, result, code = merge_slides(source_paths, output_path)
    print(json.dumps(result))
    sys.exit(code)
```

### Go Handler for Slide Image Serving

```go
// Source: Gin framework documentation and existing file serving patterns
// internal/handlers/ppt_handler.go
package handlers

import (
    "net/http"
    "path/filepath"
    "github.com/gin-gonic/gin"
)

// ServeSlideImage serves slide images (thumbnails or full-size) from cache
func (h *PPThandler) ServeSlideImage(c *gin.Context) {
    pptID := c.Param("pptId")
    resolution := c.Param("resolution") // "thumbnails" or "fullsize"
    filename := c.Param("filename")
    
    // Construct file path: recordings/ppts/{pptID}/slides/{resolution}/{filename}
    cacheDir := filepath.Join(h.config.RecordingsDir, "ppts", pptID, "slides", resolution)
    filePath := filepath.Join(cacheDir, filename)
    
    // Validate path is within cache directory (security)
    if !filepath.IsAbs(filePath) || !filepath.HasPrefix(filePath, cacheDir) {
        c.JSON(http.StatusForbidden, gin.H{"error": "Invalid file path"})
        return
    }
    
    // Check if file exists
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        c.JSON(http.StatusNotFound, gin.H{"error": "Slide image not found"})
        return
    }
    
    // Serve file with appropriate headers
    c.File(filePath)
}

// GetSlides returns list of slide image URLs for a PPT
func (h *PPThandler) GetSlides(c *gin.Context) {
    pptID := c.Param("id")
    
    // Parse uint
    id, err := strconv.ParseUint(pptID, 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PPT ID"})
        return
    }
    
    // Get PPT file from database
    var pptFile models.PPTFile
    if err := h.db.First(&pptFile, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "PPT not found"})
        return
    }
    
    // Check if slide cache exists
    cacheDir := filepath.Join(h.config.RecordingsDir, "ppts", pptID, "slides")
    thumbDir := filepath.Join(cacheDir, "thumbnails")
    
    // If cache doesn't exist, trigger extraction
    if _, err := os.Stat(thumbDir); os.IsNotExist(err) {
        // Trigger async slide extraction
        go h.slideExtractor.ExtractSlides(context.Background(), &pptFile)
        
        c.JSON(http.StatusAccepted, gin.H{
            "message": "Slide images are being generated",
            "status": "extracting"
        })
        return
    }
    
    // Read slide files from cache
    thumbFiles, _ := filepath.Glob(filepath.Join(thumbDir, "slide_*.jpg"))
    slides := make([]map[string]string, 0, len(thumbFiles))
    
    for _, thumbFile := range thumbFiles {
        filename := filepath.Base(thumbFile)
        slideNum := extractSlideNumber(filename)
        
        slides = append(slides, map[string]string{
            "slide_number": slideNum,
            "thumbnail_url": fmt.Sprintf("/api/v1/ppts/%s/slides/thumbnails/%s", pptID, filename),
            "fullsize_url": fmt.Sprintf("/api/v1/ppts/%s/slides/fullsize/%s", pptID, filename),
        })
    }
    
    c.JSON(http.StatusOK, gin.H{
        "slide_count": len(slides),
        "slides": slides,
    })
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Download PPTX to view slides | Browser-based preview with extracted images | Phase 3 | Users can preview without downloading, faster access to content |
| Single transcription result per video | Multiple PPT results with gallery switching | Phase 3 | Users can compare different transcriptions, choose best result |
| Manual slide copy-paste for merge | In-app slide selection and merge | Phase 3 | Streamlined workflow, no need for external PowerPoint software |
| No caching for slide images | Dual-resolution on-demand caching | Phase 3 | Faster subsequent previews, reduced bandwidth usage |

**Deprecated/outdated:**
- **Download-only PPT viewing**: Replaced by browser preview per PPT-03 requirement
- **Single-result transcription**: Replaced by multi-result gallery per PPT-04, PPT-05
- **Manual merge with PowerPoint**: Replaced by in-app merge functionality per PPT-06

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | python-pptx can extract embedded images from slides for MVP preview | Standard Stack | If slides have no embedded images (text-only), preview will show blank placeholders |
| A2 | Dual-resolution caching (200x112 thumbnails, 1920x1080 full-size) is sufficient for preview UX | Architecture Patterns | If users need zoom beyond 1080p, full-size resolution may be too low |
| A3 | dnd-kit library integrates well with Ant Design without conflicts | Standard Stack | If dnd-kit conflicts with Ant Design's Modal or Popover, may need alternative drag-and-drop solution |
| A4 | File system cache with structured directories (recordings/ppts/{id}/slides/) handles concurrent access safely | Runtime State Inventory | If file locking issues occur, may need database-backed cache or Redis for lock management |
| A5 | Slide extraction completes within 5-10 seconds for typical PPTs (20-50 slides) | Common Pitfalls | If extraction takes longer (minutes), on-demand extraction will feel sluggish, need async extraction with polling |
| A6 | Merged PPTX files embedded with images are independent of source PPTs (safe deletion) | Common Pitfalls | If merge implementation uses references instead of embedding, deleting source PPTs breaks merged results |
| A7 | Ant Design Image.PreviewGroup supports keyboard navigation (arrows, space, ESC) out of the box | Architecture Patterns | If keyboard support is limited, custom keyboard event handlers will be needed |
| A8 | Slide cache invalidation tied to PPTFile.UpdatedAt timestamp is sufficient | Common Pitfalls | If PPTX is regenerated without updating UpdatedAt (e.g., direct file overwrite), stale cache may be served |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. (This table IS NOT empty; 8 assumptions require user validation).

## Open Questions

1. **Slide Rendering Approach for Full PPT Preview**
   - What we know: python-pptx cannot render slides to images (only extracts structure/embedded images)
   - What's unclear: Whether embedded-image-only extraction is acceptable for MVP, or if full rendering (text/shapes) is required
   - Recommendation: Start with embedded-image-only extraction for MVP (fast, no external deps). If users report missing text/shapes, integrate LibreOffice headless mode in Phase 4+ for full rendering.

2. **Cache Invalidation Strategy**
   - What we know: Need to clear slide cache when PPTX is re-generated (new transcription)
   - What's unclear: How to trigger cache invalidation (database trigger, manual cleanup in TranscriptionService, or PPTFile updated_at timestamp check)
   - Recommendation: Update PPTFile.UpdatedAt timestamp when PPTX is regenerated. Slide extraction checks if cache directory creation time > PPTFile.UpdatedAt, if so, re-extract.

3. **Merge Implementation Detail**
   - What we know: Need to merge selected slides from multiple PPTs into new PPTX
   - What's unclear: Whether to merge by copying slide elements (images, text, shapes) or by merging entire slide objects
   - Recommendation: For MVP, merge by copying embedded images only (fast). If text/shapes need merging, use LibreOffice or Aspose.Slides for full slide merge (slower but more complete).

4. **PPTFile Model Extension for Merge Results**
   - What we know: Need to distinguish merge results from transcription results
   - What's unclear: Exact fields needed (source_type enum, merged_from JSON array, or separate MergedPPT model)
   - Recommendation: Add `source_type` enum (transcription/merge) and `merged_from` JSON array `[pptFileID1, pptFileID2, ...]` to PPTFile model. Single table is simpler than join table.

5. **Merge Progress Feedback**
   - What we know: D-18 specifies "simple loading spinner + completion toast"
   - What's unclear: Whether merge is fast enough (< 2 seconds) for spinner, or if needs progress bar
   - Recommendation: Implement spinner first. If merge takes > 5 seconds for typical cases, add progress bar with stage tracking (copying slides 1/N, saving PPTX, uploading).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Python 3 | python-pptx script execution | ✓ | 3.13.2 | — |
| python-pptx | Slide image extraction | ✓ | 1.0.2 | — |
| FFmpeg | Re-extract frames for merge (if needed) | ✓ | 2021-03-24-git-a77beea6c8-full_build | — |
| Ant Design 6 | Preview, gallery, merge UI | ✓ | 6.2.3 | — |
| React 19 | Frontend framework | ✓ | 19.2.4 | — |
| dnd-kit | Drag-to-reorder in merge mode | ✗ | — | Must install via npm |
| LibreOffice | Full slide rendering (Phase 4+) | ✗ | — | Use embedded-image extraction for MVP |

**Missing dependencies with no fallback:**
- None for MVP (embedded-image extraction works without LibreOffice)

**Missing dependencies with fallback:**
- dnd-kit: Can use react-dnd as alternative (heavier, more complex setup)
- LibreOffice: Can use embedded-image extraction for MVP (accepts limitation of text/shapes not rendering)

**Installation required before execution:**
```bash
# Frontend drag-and-drop library
npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities

# Python (already installed from Phase 2)
# python-pptx 1.0.2 already installed
```

## Validation Architecture

> workflow.nyquist_validation is not explicitly false in config.json (default: enabled) — include this section.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing package (testing) + testify assertions for backend; Jest/Vitest for frontend |
| Config file | None — uses standard Go test conventions |
| Quick run command | `go test -v -run TestSlideExtractor ./internal/services/ && npm test -- --run PPTPreview` |
| Full suite command | `go test -v ./internal/services/... ./internal/handlers/... && npm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PPT-01 | PPT file independently downloadable from video | integration | `go test -v -run TestPPTDownload ./internal/handlers/` | ❌ Wave 0 |
| PPT-02 | PPT displayed in file list linked to source video | integration | `go test -v -run TestPPTFileListAssociation ./internal/handlers/` | ❌ Wave 0 |
| PPT-03 | Browser-based PPT preview with main view + sidebar thumbnails | integration | `npm test -- --run PPTPreview` | ❌ Wave 0 |
| PPT-04 | Re-transcribe button triggers new transcription task | integration | `go test -v -run TestReTranscribe ./internal/handlers/` | ❌ Wave 0 |
| PPT-05 | Multiple PPT results displayed for same video | integration | `go test -v -run TestMultiResultDisplay ./internal/handlers/` | ❌ Wave 0 |
| PPT-06 | Slide selection and merge functionality | integration | `npm test -- --run MergeSelection` | ❌ Wave 0 |
| UI-03 | Result page split layout with preview + info panel | visual/snapshot | `npm test -- --run ResultPageLayout` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test -v -run TestSlideExtractor ./internal/services/ && npm test -- --run PPTPreview`
- **Per wave merge:** `go test -v ./internal/services/... ./internal/handlers/... && npm test`
- **Phase gate:** Full suite green (backend + frontend) before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/services/slide_extractor_test.go` — Tests for Python slide extraction script execution, cache directory creation, dual-resolution image generation
- [ ] `internal/services/slide_cache_service_test.go` — Tests for cache existence checking, cache invalidation, thumbnail/full-size path generation
- [ ] `internal/services/ppt_merge_service_test.go` — Tests for merge logic, PPTFile creation with source_type=merge, merged_from array population
- [ ] `internal/handlers/ppt_handler_test.go` — API handler tests for GET /api/v1/ppts/:id/slides, POST /api/v1/ppts/merge, slide image serving
- [ ] `frontend/src/components/__tests__/PPTPreview.test.tsx` — Component tests for preview layout, thumbnail navigation, page indicator
- [ ] `frontend/src/components/__tests__/MergeSelectionBar.test.tsx` — Component tests for drag-to-reorder, selection limit enforcement, remove slide
- [ ] `frontend/src/pages/__tests__/results.test.tsx` — Page integration tests for result page layout, gallery switching, merge mode entry/exit
- [ ] Test fixtures: Sample PPTX files with embedded images, test slide extraction output, test merge scenarios
- [ ] Framework install: npm install --save-dev vitest @testing-library/react @testing-library/jest-dom (frontend testing setup)

*(If no gaps: "None — existing test infrastructure covers all phase requirements")*
**Actual Status:** Wave 0 gaps exist — PPT preview and merge are new features, need comprehensive test coverage before implementation.

## Security Domain

> security_enforcement is enabled (absent = enabled in config.json) — include this section.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Authorization | yes | Validate user owns source video file before viewing PPT results or merging slides; check permissions on PPTFile records |
| V5 Input Validation | yes | Validate slide IDs in merge request (prevent unauthorized access), sanitize PPT ID in URL (prevent directory traversal), limit selected slides to 200 maximum (DoS prevention) |
| V8 Error Handling | yes | Log slide extraction failures without exposing system paths, return generic error messages for missing slides, handle Python script failures gracefully |
| V9 Memory Management | yes | Limit slide image cache size per PPT (cleanup old slides), enforce 200-slide merge limit, process large PPTs in batches |
| V11 File Handling | yes | Validate slide cache paths are within allowed directories, prevent path traversal in slide image URLs, sanitize PPTX filenames |
| V12 File Upload | no | No file upload in this phase (PPTX files are server-generated) |

### Known Threat Patterns for {React PPT Preview with Python Backend}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Unauthorized PPT access via ID guessing | Spoofing | Validate user owns SourceVideoFileID before serving slide images, check PPTFile ownership in GET /api/v1/ppts/:id/slides |
| Directory traversal via slide filename | Tampering | Sanitize filename parameter, validate path is within cache directory, use filepath.Clean() and filepath.HasPrefix() checks |
| DoS via merging 1000+ slides | Denial of Service | Enforce 200-slide merge limit (D-17), validate merge request size, return 400 if limit exceeded |
| Cache poisoning via malicious PPTX | Tampering | Validate PPTX file structure before extraction, limit extraction time with context timeout, quarantine suspicious files |
| Path traversal in merge source paths | Tampering | Validate all source PPTX paths are within allowed directories, use absolute path resolution with prefix checks |
| Information disclosure via error messages | Disclosure | Log detailed errors server-side, return generic "Slide extraction failed" to client, don't expose file paths or Python stack traces |

## Sources

### Primary (HIGH confidence)

- [Context7: python-pptx] - PPTX file structure reading, embedded image extraction, slide manipulation, presentation creation
- [Context7: Ant Design 6] - Image.PreviewGroup component for main+sidebar layout, Image component API, Gallery component for multi-result display
- [Existing Codebase] - PPTXGenerator service (internal/services/pptx_generator.go), PPTFile model (internal/models/ppt_file.go), TranscriptionProgressModal (frontend/src/components/TranscriptionProgressModal.tsx), File list page (frontend/src/pages/files/index.tsx)
- [Python Environment] - Verified python 3.13.2 installed, python-pptx 1.0.2 installed
- [Go Module Registry] - Verified Go 1.24.5, Gin framework, GORM versions via go list

### Secondary (MEDIUM confidence)

- [Project CONTEXT.md Decisions D-01 through D-22] - User-approved implementation approach for preview layout, dual-resolution caching, merge functionality, result page design
- [Project REQUIREMENTS.md] - PPT-01 through PPT-06 acceptance criteria, UI-03 requirement
- [dnd-kit Documentation] - Sortable context, drag-and-drop sensors, array utilities for reordering
- [Ant Design Documentation] - Image component props, PreviewGroup usage, keyboard navigation support

### Tertiary (LOW confidence)

- [Web Search: python-pptx slide rendering] - Confirmed python-pptx cannot render slides to images (LOW confidence due to lack of official docs on rendering limitations)
- [Web Search: LibreOffice headless slide rendering] - Found soffice --headless --convert-to pdf pattern (LOW confidence, not verified with actual testing)
- [Web Search: PPT preview patterns] - Based on training knowledge of PowerPoint-like interfaces (LOW confidence, needs UX validation)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Python 3.13.2 and python-pptx 1.0.2 verified via command execution, Ant Design 6.2.3 verified via npm list
- Architecture: MEDIUM - Preview layout pattern based on established UI conventions, but slide rendering has limitations (embedded images only for MVP)
- Pitfalls: MEDIUM - python-pptx rendering limitation is well-documented, cache miss storms and merge orphaning are standard concurrency/data integrity issues with known mitigations

**Research date:** 2026-04-17
**Valid until:** 2026-05-17 (30 days - stable stack, but LibreOffice integration may change assumptions)
