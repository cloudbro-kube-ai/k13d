# Presentation Page Agent Guide

> **File:** `presentation/index.html` (3496 lines)
> **Type:** Self-contained HTML slideshow presentation
> **Purpose:** k13d Renewal Project feature branch review presentation (20 slides)

## Architecture Overview

```
presentation/
├── index.html                    # Main presentation (single-file HTML)
├── agent.md                      # This guide
├── generate_pptx.py              # PPTX generation script (python-pptx)
├── capture_slides.py             # Slide capture script
├── screenshot-web1.png           # Web UI dashboard
├── screenshot-web2.png           # AI Assistant panel
├── screenshot-web3.png           # Cluster Visualizer
├── screenshot-web4.png           # Web UI Settings
├── screenshot-web5.png           # Additional screenshot
├── screenshot-cli.png            # CLI interface
├── screenshot-tui.png            # TUI interface
├── hermes.png                    # CLI Reference: Hermes
├── ibmbob.png                    # CLI Reference: IBM BOP
├── mimo.png                      # CLI Reference: Mimo
├── opencode.png                  # CLI Reference: OpenCode
├── *.pptx                        # Generated PPTX outputs
└── *.pdf                         # Generated PDF outputs
```

## File Structure

The entire presentation is a **single HTML file** with embedded CSS and JavaScript — no build tools, no external CSS/JS dependencies. Self-contained for portability.

### Document Sections

| # | Lines | Section | Purpose |
|---|-------|---------|---------|
| 1 | 8–2284 | `<style>` | All CSS (embedded, ~2277 lines) |
| 2 | 2287–3416 | 20 Slides | Presentation content (HTML) |
| 3 | 3417 | Nav hint | Keyboard navigation hint |
| 4 | 3419–3428 | PDF button | Download-as-PDF floating button |
| 5 | 3430–3493 | `<script>` | JS: slide navigation + PDF print |

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
├── .slide-number          # "1 / 20"
└── .contributors          # Contributor avatars grid
    ├── .contributors-label
    └── .contributor * N
        ├── img            # GitHub avatar
        └── .contributor-name
```

**Key CSS:**
- Dark gradient background (`#1a1a2e → #16213e → #0f3460`)
- Radial gradient background pseudo-element with `pulse` animation (8s)
- White text, centered layout
- Slide number offset from bottom (`bottom: 80px`)

**Used in:** Slide 1 (Title)

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

### Type 3: About / Introduction Slide (Dual Screen)

**CSS class:** `.slide-about`

**Purpose:** K13D project introduction with dual-screen mockup and value proposition.

**Structure:**
```
.slide-about
├── .slide-header
├── .about-visual              # 2-column grid (1.1fr 1fr)
│   ├── .about-value           # Left: Value proposition
│   │   ├── .about-value-title
│   │   │   ├── .about-namederiv   # Name derivation (k + 13 letters + d)
│   │   │   ├── span.paradigm      # "K8s 운영의 새로운 패러다임"
│   │   │   └── span.sub           # "Dashboard + AI Assistant"
│   │   ├── .about-value-desc      # Description paragraph
│   │   └── .about-compare         # 2×2 comparison grid (vs k9s, K8s Dashboard, etc.)
│   │       └── .about-compare-item * 4
│   │           ├── span.vs        # "vs k9s"
│   │           └── span.vs-label  # Comparison text
│   └── .about-dual            # Right: Dual screen mockup
│       ├── .dual-screens      # 2-column grid
│       │   ├── .dual-screen   # Dashboard mockup
│       │   │   ├── .dual-screen-header
│       │   │   └── .dual-screen-content
│       │   │       └── .dual-row * N (with .status .name .ns)
│       │   │           └── .dual-row.highlight  # Error state
│       │   └── .dual-screen   # AI Chat mockup
│       │       ├── .dual-screen-header
│       │       └── .dual-chat
│       │           └── .chat-bubble * N (.user / .ai)
│       └── .about-connect     # Connection description bar
└── .slide-number
```

**Key CSS:**
- `.about-visual`: grid with 1.1fr / 1fr columns
- `.dual-screen`: light gray bg with border-radius 14px
- `.dual-row.highlight`: red-tinted background for error states
- `.chat-bubble.user`: accent-colored, right-aligned
- `.chat-bubble.ai`: gray bg, left-aligned

**Used in:** Slide 2 (K13D Introduction)

---

### Type 4: Architecture Diagram (Layered Boxes)

**CSS class:** `.slide-arch`

**Purpose:** System architecture visualization with layered UI → Core → External services.

**Structure:**
```
.slide-arch
├── .slide-header
├── .arch-content              # Vertical flex, centered
│   ├── .arch-layer-label      # Uppercase layer title
│   ├── .arch-row              # Horizontal row of boxes
│   │   └── .arch-box * N      # White card box
│   │       ├── .name          # Component name
│   │       ├── .sub           # Subtitle
│   │       └── .badge         # Optional tech badge
│   ├── .arch-arrow-down       # Vertical arrow between layers
│   ├── .arch-core             # Dark gradient core layer
│   │   ├── .arch-core-item * N
│   │   │   ├── .name
│   │   │   └── .sub
│   │   └── .arch-core-divider # Vertical separator
│   ├── .arch-arrow-down
│   ├── .arch-layer-label
│   └── .arch-external         # External services row
│       └── .arch-external-item * N
│           ├── .name
│           ├── .sub
│           └── .badge
└── .slide-number
```

**Key CSS:**
- `.arch-box-highlight`: red border + glow shadow for emphasized boxes
- `.arch-core`: dark gradient background with white text, horizontal flex
- `.arch-arrow-down`: 2px vertical line with CSS triangle arrow
- `.arch-external-item`: white card with optional colored badge

**Used in:** Slide 3 (Architecture), Slide 7 (AI Answering Flow — reused with different content)

---

### Type 5: Feature Map (6-Group Grid)

**CSS class:** `.slide-featmap`

**Purpose:** Comprehensive feature overview in 6 categorized groups.

**Structure:**
```
.slide-featmap
├── .slide-header
├── .feat-grid                  # 3-column grid, gap 16px
│   └── .feat-group * 6         # White card with colored header
│       ├── .feat-header        # Colored gradient header
│       │   ├── svg             # Icon
│       │   ├── [text]          # Group name
│       │   └── .badge          # Count badge
│       └── .feat-body
│           └── .feat-item * N  # Feature line items
│               └── [strong + text]
```

**Key CSS:**
- 6 color variants: `.feat-group-1` (navy) through `.feat-group-6` (red)
- Each group header has a unique gradient
- `.feat-item` uses bullet point pseudo-element

**Used in:** Slide 4 (K13D Feature Map)

---

### Type 6: API Spec Grid

**CSS class:** `.slide-2` (reused)

**Purpose:** API endpoint specification organized by category.

**Structure:**
```
.slide-2
├── .slide-header
├── .apispec-grid                # 3-column grid
│   └── .apispec-group * N       # White card
│       ├── .apispec-header      # Icon + title + count
│       │   ├── .apispec-icon    # Colored letter badge
│       │   ├── .apispec-title
│       │   └── .apispec-count   # Endpoint count
│       └── .apispec-item * N    # Monospace endpoint text
└── .slide-number
```

**Key CSS:**
- `.apispec-item`: monospace font (`Monaco, Menlo`), 10px size
- `.apispec-icon`: 22×22px colored letter badge
- 9 categories: System, Auth, K8s Resources, Workload Ops, AI & LLM, Cluster, Helm, Metrics & Audit, Security

**Used in:** Slide 5 (K13D API Spec)

---

### Type 7: LLM Provider Table

**CSS class:** `.slide-llm`

**Purpose:** Detailed comparison table of supported LLM providers.

**Structure:**
```
.slide-llm
├── .slide-header
├── .llm-table-wrapper          # Flex container with overflow auto
│   ├── table.llm-table         # Full-width table
│   │   ├── thead               # Dark gradient header row
│   │   │   └── th * 7          # Provider, 유형, Tool Calling, Streaming, etc.
│   │   └── tbody
│   │       └── tr * N          # One per provider
│   │           ├── .provider-name + .provider-type
│   │           ├── td (API type)
│   │           ├── td (.check / .cross)
│   │           ├── td (.tag .tag-cloud/.tag-local/.tag-enterprise/.tag-proxy)
│   │           └── td (비고)
│   └── .llm-note               # Tip box below table
└── .slide-number
```

**Key CSS:**
- `.llm-table thead th`: dark gradient background with red bottom border
- `.tag-cloud`: blue, `.tag-local`: green, `.tag-enterprise`: purple, `.tag-proxy`: orange
- `.check`: green checkmark, `.cross`: gray X
- Row hover: `rgba(233, 69, 96, 0.06)` highlight

**Used in:** Slide 6 (LLM Provider Table)

---

### Type 8: AI Flow Diagram

**CSS class:** `.slide-arch` (reused)

**Purpose:** Step-by-step AI answering process visualization.

**Structure:**
```
.slide-arch
├── .slide-header
├── .aiflow-content             # Vertical flex
│   ├── .aiflow-row             # Horizontal flow row
│   │   └── .aiflow-node * N    # Colored bordered node
│   │       ├── .aiflow-icon    # SVG icon
│   │       ├── .aiflow-label   # Node title
│   │       └── .aiflow-sub     # Description
│   ├── .aiflow-subrow          # Tool execution detail row
│   │   └── .aiflow-group * N   # Sub-groups (tools, safety, API)
│   │       ├── .aiflow-group-title
│   │       └── .aiflow-tool * N
│   │           ├── .aiflow-tool-badge  (.kb/.bs/.mcp/.ok/.warn)
│   │           └── [text]
│   └── .aiflow-info            # Dark info box with final result
│       └── .aiflow-info-item * N
│           └── strong + text
└── .slide-number
```

**Key CSS:**
- `.aiflow-node` color variants: `.aiflow-user` (navy), `.aiflow-context` (blue), `.aiflow-llm` (purple), `.aiflow-synth` (green), `.aiflow-answer` (red)
- `.aiflow-tool-badge` variants: `.kb` (blue), `.bs` (green), `.mcp` (purple), `.ok` (green), `.warn` (orange)
- `.aiflow-info`: dark background with monospace font

**Used in:** Slide 7 (AI Answering Flow)

---

### Type 9: Overview Grid (2×2 cards with stats)

**CSS class:** `.slide-2` (reused)

**Purpose:** Summary of major improvement categories with statistics bar.

**Structure:**
```
.slide-2
├── .slide-header
├── .overview-grid              # 2×2 grid, gap 20px
│   └── .overview-card * 4      # White card with colored left border
│       ├── .card-icon          # 44×44px colored icon container
│       │   └── svg
│       ├── h3
│       └── p
├── .stat-row                   # Horizontal stats bar
│   └── .stat-item * N
│       ├── .stat-number        # Large number (32px, accent color)
│       └── .stat-label
└── .slide-number
```

**Card variants:** Each `.overview-card:nth-child(N)` has a distinct border color (`#e94560`, `#0f3460`, `#16a085`, `#9b59b6`).

**Used in:** Slide 8 (Overview)

---

### Type 10: Feature Grid (3-column cards with SVG icons)

**CSS class:** `.slide-3`

**Purpose:** Feature showcase with icons.

**Structure:**
```
.slide-3
├── .slide-header
├── .feature-grid              # 3-column grid, gap 24px
│   └── .feature-card * 6      # White card with colored icon area
│       ├── .feature-icon      # 48×48px rounded container
│       │   └── svg
│       ├── h4
│       └── p
└── .slide-number
```

**Icon variants:** Each card's icon area has a distinct colored background (`#ffe8ec`, `#e8f4fd`, `#e8f8f5`, `#f4ecf7`, `#fef5e7`, `#fde8e8`).

**Used in:** Slide 9 (Web UI Features)

---

### Type 11: Screenshot (full image)

**CSS class:** `.slide-6`, `.slide-7`, `.slide-8`, `.slide-10`, `.slide-12`

**Purpose:** Full-slide screenshot display.

**Structure:**
```
.slide-N
├── .slide-header
├── .cli-screenshot             # Centered flex container
│   └── img                     # Screenshot image
└── .slide-number
```

**Key CSS:**
- `.cli-screenshot`: `flex: 1; display: flex; align-items: center; justify-content: center; padding: 0 20px;`
- `img`: `max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 12px;`

**Used in:** Slides 10–14 (Web UI screenshots + TUI screenshot)

---

### Type 12: CLI Layout (Two-column)

**CSS class:** `.slide-9`

**Purpose:** CLI features, command tables, keyboard shortcuts.

**Structure:**
```
.slide-9
├── .slide-header
├── .cli-layout                 # 2-column grid (1fr 1.4fr)
│   ├── [left column]
│   │   ├── .cli-main           # Feature list card
│   │   │   ├── h3
│   │   │   └── ul.cli-list
│   │   │       └── li * N (with .cli-badge)
│   │   └── .cli-code           # Dark code block
│   │       ├── .comment
│   │       ├── [commands]
│   │       └── .highlight
│   └── .cli-commands-panel     # Right panel with command tables
│       ├── h3
│       ├── .cli-cmd-group * N  # Command group
│       │   ├── .cli-cmd-group-title
│       │   └── table.cli-cmd-table
│       │       └── tr * N
│       └── .cli-keys-panel     # Dark keyboard shortcuts panel
│           ├── .cli-keys-title
│           └── .cli-keys-grid  # 2-column grid
│               └── .cli-key-item * N
│                   └── kbd + text
└── .slide-number
```

**Key CSS:**
- `.cli-keys-panel`: dark background (`#1a1a2e`), green kbd text (`#00ff88`)
- `.cli-badge`: red pill badge
- `.cli-code`: dark background, green monospace text

**Used in:** Slide 15 (CLI Features)

---

### Type 13: CLI Screenshot

**CSS class:** `.slide-10`

**Purpose:** CLI interface screenshot display.

**Structure:** Same as Type 11 (Screenshot).

**Used in:** Slide 16 (CLI Screenshot)

---

### Type 14: Two-Column Split Layout

**CSS class:** `.slide-11`

**Purpose:** Side-by-side comparison.

**Structure:**
```
.slide-11
├── .slide-header
├── .split-layout               # 2-column grid, gap 40px
│   └── .split-section * 2      # White card sections
│       ├── h3
│       └── ul.improvement-list
│           └── li * N          # ✓ prefixed items
└── .slide-number
```

**Key CSS:**
- `.split-layout`: `grid-template-columns: 1fr 1fr; gap: 40px;`
- `.improvement-list li::before { content: '✓'; color: #16a085; }`

**Used in:** (Previously used; currently not in slide rotation but CSS preserved)

---

### Type 15: Tech Grid (2×2 cards)

**CSS class:** `.slide-13`, `.slide-14`

**Purpose:** Technical challenges / strategic direction cards.

**Structure:**
```
.slide-N
├── .slide-header
├── .tech-grid                  # 2-column grid, gap 24px
│   └── .tech-card * 4          # White rounded card
│       ├── h4                  # Title (26px)
│       ├── p                   # Description (18px)
│       └── .tech-tag * N       # Tag badges (#f0f2f5 bg)
└── .slide-number
```

**Variant:** `.slide-14 .tech-grid` uses `gap: 20px` (narrower) for direction cards.

**Used in:** Slides 17 (Technical), 18 (Direction)

---

### Type 16: Summary Slide (custom structure)

**CSS class:** `.slide-15`

**Purpose:** Final summary with stats grid + roadmap timeline.

**Structure:**
```
.slide-15
├── .summary-content            # 2-column grid, gap 50px
│   ├── .summary-left
│   │   ├── h3
│   │   └── .summary-stats      # 2×2 grid
│   │       └── .summary-stat * 4
│   │           ├── .number     # 32px, accent color
│   │           └── .label
│   └── .summary-right
│       ├── h3
│       └── ul.roadmap
│           └── li * N
│               ├── .roadmap-phase  # Phase badge
│               └── span            # Description
├── .closing-message
│   └── p
└── .slide-number
```

**Key CSS:**
- Same dark gradient background as title slide
- `.summary-stat`: semi-transparent white background (`rgba(255,255,255,0.1)`)
- `.roadmap-phase`: accent-colored badge with border

**Used in:** Slide 19 (Summary)

---

### Type 17: CLI Reference Grid (2×2 images)

**CSS class:** `.slide-cli-ref`

**Purpose:** Reference UI screenshots from other CLI tools.

**Structure:**
```
.slide-cli-ref
├── .slide-header
├── .cli-ref-grid               # 2×2 grid
│   └── .cli-ref-card * 4       # White rounded card
│       └── img                 # Reference screenshot
└── .slide-number
```

**Key CSS:**
- `.cli-ref-grid`: `grid-template-columns: repeat(2, 1fr); gap: 24px;`
- `.cli-ref-card`: white bg, border-radius 16px, centered image
- `img`: `width: 100%; height: 100%; object-fit: contain;`

**Used in:** Slide 20 (CLI Reference)

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
- **Card titles:** 16–26px weight
- **Body text:** 13–18px, `#555` or `#666`
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
| 2 | `.slide-about` | About | K13D introduction, dual-screen mockup, value proposition |
| 3 | `.slide-arch` | Architecture | k13D system architecture (UI → Core → External) |
| 4 | `.slide-featmap` | Feature Map | 6-group feature grid (K8s, AI, TUI, Web, Monitor, Ops) |
| 5 | `.slide-2` (reused) | API Spec | 9-category API endpoint grid |
| 6 | `.slide-llm` | LLM Table | 9 LLM provider comparison table |
| 7 | `.slide-arch` (reused) | AI Flow | AI answering process visualization |
| 8 | `.slide-2` (reused) | Overview | 4 improvement categories + stats row |
| 9 | `.slide-3` | Feature Grid | 6 Web UI features with SVG icons |
| 10 | `.slide-6` | Screenshot | Web UI dashboard screenshot |
| 11 | `.slide-6` | Screenshot | Web UI AI Assistant screenshot |
| 12 | `.slide-7` | Screenshot | Cluster Visualizer screenshot |
| 13 | `.slide-8` | Screenshot | Web UI Settings screenshot |
| 14 | `.slide-12` | Screenshot | TUI interface screenshot |
| 15 | `.slide-9` | CLI Layout | CLI features, commands, shortcuts |
| 16 | `.slide-10` | Screenshot | CLI interface screenshot |
| 17 | `.slide-13` | Tech Grid | Technical challenges & solutions |
| 18 | `.slide-14` | Tech Grid | Strategic direction & market |
| 19 | `.slide-15` | Summary | Stats summary + 4-phase roadmap |
| 20 | `.slide-cli-ref` | CLI Reference | 4 reference CLI UI screenshots |

---

## Adding / Modifying Slides

### To add a new slide:

1. Copy an existing slide's HTML block
2. Give it a unique class (use existing type if layout matches, or create new)
3. Add corresponding CSS if new layout is needed
4. Update slide numbers (`.slide-number` and comments)
5. Update the Slide Content Map above

### Architecture rules:
- **All CSS** goes inside the single `<style>` block (lines 8–2284)
- **All JS** goes inside the single `<script>` block (lines 3430–3493)
- **Slide visibility** is controlled by JS `display: none/flex`
- **Images** are referenced as relative paths in the same `presentation/` directory
- **No external dependencies** — everything is self-contained
- **Slide numbers** must be manually updated in both the HTML comments and `.slide-number` divs

### Naming conventions:
- Slide classes: `.slide-N` where N = slide number (or descriptive like `.slide-about`, `.slide-arch`, `.slide-featmap`, `.slide-llm`, `.slide-cli-ref`)
- Block diagram classes: `.block-*`
- CLI section classes: `.cli-*`
- About section classes: `.about-*`, `.dual-*`
- Architecture classes: `.arch-*`
- Feature map classes: `.feat-*`
- AI flow classes: `.aiflow-*`
- API spec classes: `.apispec-*`
- LLM table classes: `.llm-*`
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

### PPTX Generation

Alternatively, `generate_pptx.py` uses `python-pptx` to generate PPTX files programmatically from the slide content. Verify availability with `python3 -c "import pptx"`.
