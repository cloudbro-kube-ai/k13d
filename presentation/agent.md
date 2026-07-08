# Presentation Page Agent Guide

> **File:** `presentation/index.html` (1791 lines)
> **Type:** Self-contained HTML slideshow presentation
> **Purpose:** k13d Renewal Project feature branch review presentation (15 slides)

## Architecture Overview

```
presentation/
├── index.html              # Main presentation (single-file HTML)
├── agent.md                # This guide
├── k13d 리뉴얼 프로젝트 발표 자료.pdf  # Generated PDF output
├── screenshot-web1.png     # Web UI dashboard
├── screenshot-web2.png     # AI Assistant panel
├── screenshot-web3.png     # Cluster Visualizer
├── screenshot-web4.png     # Web UI Settings
├── screenshot-web5.png     # Additional screenshot
├── screenshot-cli.png      # CLI interface
└── screenshot-tui.png      # TUI interface
```

## File Structure

The entire presentation is a **single HTML file** with embedded CSS and JavaScript — no build tools, no external CSS/JS dependencies. Self-contained for portability.

### Document Sections

| # | Lines | Section | Purpose |
|---|-------|---------|---------|
| 1 | 8–1046 | `<style>` | All CSS (embedded) |
| 2 | 1048–1710 | 15 Slides | Presentation content (HTML) |
| 3 | 1712 | Nav hint | Keyboard navigation hint |
| 4 | 1714–1723 | PDF button | Download-as-PDF floating button |
| 5 | 1725–1789 | `<script>` | JS: slide navigation + PDF print |

---

## Slide Type Catalog

Each slide follows one of these layout patterns:

---

### Type 1: Title Slide

**CSS class:** `.slide-1`

**Purpose:** Opening/closing splash, high-impact branding.

**Structure:**
```
.slide.slide-1
├── .title-logo           # Big brand text (72px)
│   └── span              # Accent-colored portion (e94560)
├── .title-subtitle        # Tagline (28px, light weight)
├── .title-meta            # Metadata: dates, event info
├── .title-badge           # Round badge with stats
├── .slide-number          # "1 / 15"
└── .contributors          # Contributor avatars grid
    ├── .contributors-label
    └── .contributor * N
        ├── img            # GitHub avatar
        └── .contributor-name
```

**Key CSS:**
- Dark gradient background (`#1a1a2e → #16213e → #0f3460`)
- Radial gradient background pseudo-element with `pulse` animation
- White text, centered layout
- Slide number offset from bottom (`bottom: 80px`)

**Used in:** Slide 1 (Title), Slide 15 (Summary/Closing — custom structure)

---

### Type 2: Content + Header Slide

**CSS class:** `.slide-2`, `.slide-3`, `.slide-5`–`.slide-14`

**Purpose:** Standard content slide with header. Most common type.

**Structure:**
```
.slide.slide-N
├── .slide-header
│   ├── h2                # Title text (36px, bold)
│   └── .accent-line       # Decorative bar (80×4px, gradient)
├── [content-grid]         # Varies by type (see below)
└── .slide-number          # Page number
```

**Key CSS:**
- Light gradient background (`#ffffff → #f8f9fa`)
- `.slide-header` has 50px bottom margin
- `.accent-line` is the brand accent bar: `#e94560 → #0f3460` gradient

---

### Type 3: Overview Grid (2×2 cards)

**CSS class:** `.slide-2`

**Purpose:** Four-item feature/overview summary with statistics bar.

**Structure:**
```
.slide-2
├── .slide-header
├── .overview-grid         # 2×2 grid, gap 30px
│   └── .overview-card * 4  # White card with colored left border
│       ├── h3
│       └── p
├── .stat-row              # Horizontal stats bar
│   └── .stat-item * N
│       ├── .stat-number   # Large number (36px, accent color)
│       └── .stat-label
└── .slide-number
```

**Card variants:** Each `.overview-card:nth-child(N)` has a distinct border color (`#e94560`, `#0f3460`, `#16a085`, `#9b59b6`).

---

### Type 4: Feature Grid (3-column cards)

**CSS class:** `.slide-3`

**Purpose:** Feature showcase with SVG icons.

**Structure:**
```
.slide-3
├── .slide-header
├── .feature-grid           # 3-column grid, gap 24px
│   └── .feature-card * 6   # White card with colored icon area
│       ├── .feature-icon   # 48×48px rounded container
│       │   └── svg         # Lucide-style SVG icon
│       ├── h4              # Feature name (16px)
│       └── p               # Description (13px)
└── .slide-number
```

**Icon variants:** Each card's icon area has a distinct colored background (`#ffe8ec`, `#e8f4fd`, `#e8f8f5`, `#f4ecf7`, `#fef5e7`, `#fde8e8`).

---

### Type 5: Block Diagram (pure CSS)

**CSS class:** `.slide-4` (custom padding: `40px 50px`)

**Purpose:** Complex UI architecture/flow visualization without external tools. Entirely CSS + HTML.

**Structure:**
```
.slide-4
├── .slide-header
├── .block-diagram          # Vertical flex container
│   ├── .block-topbar       # TOP BAR row (dark bg)
│   │   ├── .block-topbar-label
│   │   └── .block-topbar-items
│   │       └── .block-topbar-item * N  # (some .key for accent)
│   ├── .block-main          # 3-column grid: 200px / 1fr / 260px
│   │   ├── .block-sidebar   # Dark sidebar with groups
│   │   │   ├── .block-sidebar-title
│   │   │   ├── .block-sidebar-group * N
│   │   │   └── .block-sidebar-items
│   │   │       └── .block-sidebar-item * N
│   │   ├── .block-content   # White content area
│   │   │   ├── .block-content-title
│   │   │   └── .block-content-grid  # 2×2 inner grid
│   │   │       └── .block-content-section * 4
│   │   │           ├── h5
│   │   │           │   └── .count (badge)
│   │   │           └── ul.block-content-list
│   │   │               └── li * N
│   │   └── .block-ai        # Dark AI panel (gradient bg)
│   │       ├── .block-ai-title
│   │       │   └── .badge
│   │       └── .block-ai-section * N
│   │           ├── h6
│   │           └── .block-ai-items
│   │               └── .block-ai-item * N
│   └── .block-settings      # Settings bar (white, bottom)
│       ├── .block-settings-label
│       └── .block-settings-tabs
│           └── .block-settings-tab * N  # (some .key for accent)
└── .slide-number
```

**Key CSS:**
- Nested grids: top-level `.block-diagram` is vertical flex, `.block-main` is 3-column grid
- Sidebar uses 8px font for group labels, 9px for items
- AI panel uses gradient: `#16213e → #1a1a2e`
- Settings tabs at bottom: horizontal flex wrap

---

### Type 6: Screenshot (full image)

**CSS class:** `.slide-5`, `.slide-7`, `.slide-8`, `.slide-10`, `.slide-12`

**Purpose:** Full-slide screenshot display.

**Structure:**
```
.slide-N
├── .slide-header
├── .cli-screenshot          # Centered flex container
│   └── img                  # Screenshot image
└── .slide-number
```

**Key CSS:**
- `.cli-screenshot`: `flex: 1; display: flex; align-items: center; justify-content: center; padding: 0 20px;`
- `img`: `max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 12px;`
- Slides 5–8 use `.slide-5`–`.slide-8` classes but share `.cli-screenshot` layout

---

### Type 7: Two-Column Split Layout

**CSS class:** `.slide-11`

**Purpose:** Side-by-side comparison (TUI improvements vs Config system).

**Structure:**
```
.slide-11
├── .slide-header
├── .split-layout            # 2-column grid, gap 40px
│   └── .split-section * 2   # White card sections
│       ├── h3
│       └── ul.improvement-list
│           └── li * N       # ✓ prefixed items
└── .slide-number
```

**Key CSS:**
- `.split-layout`: `grid-template-columns: 1fr 1fr; gap: 40px;`
- `.improvement-list li::before { content: '✓'; color: #16a085; }`

---

### Type 8: Tech Grid (2×2 cards)

**CSS class:** `.slide-13`, `.slide-14`

**Purpose:** Technical challenges / strategic direction cards.

**Structure:**
```
.slide-N
├── .slide-header
├── .tech-grid               # 2-column grid, gap 24px
│   └── .tech-card * 4       # White rounded card
│       ├── h4               # Title (20px)
│       ├── p                # Description
│       └── .tech-tag * N    # Tag badges (#f0f2f5 bg)
└── .slide-number
```

**Variant:** `.slide-14 .tech-grid` uses `gap: 20px` (narrower) for direction cards.

---

### Type 9: Summary Slide (custom structure)

**CSS class:** `.slide-15`

**Purpose:** Final summary with stats grid + roadmap timeline.

**Structure:**
```
.slide-15
├── .summary-content          # 2-column grid, gap 50px
│   ├── .summary-left
│   │   ├── h3
│   │   └── .summary-stats    # 2×2 grid
│   │       └── .summary-stat * 4
│   │           ├── .number   # 32px, accent color
│   │           └── .label
│   └── .summary-right
│       ├── h3
│       └── ul.roadmap
│           └── li * N
│               ├── .roadmap-phase  # Phase badge
│               └── span            # Description
├── .closing-message
│   └── p                    # Tagline
└── .slide-number
```

**Key CSS:**
- Same dark gradient background as title slide
- `.summary-stat`: semi-transparent white background (`rgba(255,255,255,0.1)`)
- `.roadmap-phase`: accent-colored badge with border
- `.closing-message`: top border separator

---

## Design System

### Color Palette

| Token | Hex | Usage |
|-------|-----|-------|
| Dark bg | `#1a1a2e` | Title/Summary slides, sidebars |
| Darker | `#16213e` | Gradient midpoint |
| Navy | `#0f3460` | Gradient endpoint, tab highlight |
| Accent | `#e94560` | Logo "13", badges, buttons, borders |
| White | `#ffffff` | Content backgrounds |
| Light bg | `#f8f9fa` | Slide backgrounds |
| Green | `#16a085` | Checkmarks, confirmation |
| Purple | `#9b59b6` | Accent variant |
| Yellow | `#f39c12` | Accent variant |
| Red | `#c0392b` | Accent variant |
| Terminal | `#00ff88` | Code block text |

### Typography

- **Font stack:** `'Segoe UI', -apple-system, BlinkMacSystemFont, sans-serif`
- **Monospace:** `'Monaco', 'Menlo', monospace` (for CLI commands/code)
- **Slide titles:** 36px, 700 weight
- **Card titles:** 16–20px weight
- **Body text:** 13–15px, `#555` or `#666`
- **Small print:** 9–11px (block diagrams, tags)

### Animation

- **Title slide glow:** `.slide-1::before` radial gradient with 8s pulse keyframe (scale 1 → 1.1)
- **Feature card hover:** `transition: transform 0.2s` (defined but no hover transform set in CSS)

### Print / PDF

- `@page { size: A4 landscape; margin: 0; }`
- Print hides `.nav-hint` and `.pdf-download-btn`
- `page-break-after: always` on each slide
- `print-color-adjust: exact` for full color fidelity
- PDF download triggers `window.print()` via button

---

## Interactive JavaScript

### Slide Navigation

```javascript
let currentSlide = 0;
const slides = document.querySelectorAll('.slide');

function showSlide(index) {
    slides.forEach((slide, i) => {
        slide.style.display = i === index ? 'flex' : 'none';
    });
}

// Arrow keys: Right/Down → next, Left/Up → prev
document.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') { ... }
    else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') { ... }
});
```

### PDF Download

```javascript
function downloadPDF() {
    // 1. Show all slides
    // 2. Hide nav hint + button
    // 3. Call window.print()
    // 4. Restore single-slide view after print dialog closes
}
```

---

## Slide Content Map

| Slide | Class | Type | Content |
|-------|-------|------|---------|
| 1 | `.slide-1` | Title | k13d branding, subtitle, dates, contributors |
| 2 | `.slide-2` | Overview Grid | 4 improvement categories + stats row |
| 3 | `.slide-3` | Feature Grid | 6 Web UI features with SVG icons |
| 4 | `.slide-4` | Block Diagram | Full Web UI menu/feature architecture |
| 5 | `.slide-5` | Screenshot | Web UI dashboard |
| 6 | `.slide-6` | Screenshot | AI Assistant panel |
| 7 | `.slide-7` | Screenshot | Cluster Visualizer |
| 8 | `.slide-8` | Screenshot | Web UI Settings |
| 9 | `.slide-9` | CLI (custom) | CLI features, code example, command tables, shortcuts |
| 10 | `.slide-10` | Screenshot | CLI interface |
| 11 | `.slide-11` | Split Layout | TUI improvements + Config system |
| 12 | `.slide-12` | Screenshot | TUI interface |
| 13 | `.slide-13` | Tech Grid | Technical challenges & solutions |
| 14 | `.slide-14` | Tech Grid | Strategic direction & market |
| 15 | `.slide-15` | Summary | Stats summary + 4-phase roadmap |

---

## Adding / Modifying Slides

### To add a new slide:

1. Copy an existing slide's HTML block
2. Give it a unique `.slide-N` class
3. Add corresponding CSS if new layout is needed
4. Update slide numbers (`.slide-number` and comments)

### Architecture rules:
- **All CSS** goes inside the single `<style>` block (lines 8–1046)
- **All JS** goes inside the single `<script>` block (lines 1725–1789)
- **Slide visibility** is controlled by JS `display: none/flex`
- **Images** are referenced as relative paths in the same `presentation/` directory
- **No external dependencies** — everything is self-contained

### Naming conventions:
- Slide classes: `.slide-N` where N = slide number
- Block diagram classes: `.block-*`
- CLI section classes: `.cli-*`
- Generic component patterns: `.feature-*`, `.tech-*`, `.overview-*`, `.split-*`, `.screenshot-*`

---

## PDF Generation

The presentation uses **browser-native print-to-PDF**:

1. Click "PDF 다운로드" button
2. All slides become visible simultaneously
3. Browser print dialog opens with A4 landscape preset
4. User selects "Save as PDF" in the print dialog
5. After closing, single-slide view is restored

Prerequisites: `@page { size: A4 landscape; margin: 0; }` CSS ensures proper output.
