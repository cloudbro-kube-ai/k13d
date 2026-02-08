# k13d Documentation Site

A static documentation website for k13d, styled with the Tokyo Night theme.

## Quick Start

### Run Locally

```bash
cd docs-site
go run serve.go

# Open http://localhost:3000
```

Or with Python:

```bash
cd docs-site
python3 -m http.server 3000

# Open http://localhost:3000
```

### Custom Port

```bash
go run serve.go -port 8080
```

## Structure

```
docs-site/
├── index.html              # Home page (Overview)
├── css/
│   ├── main.css           # Base styles, variables, layout
│   ├── sidebar.css        # Sidebar navigation
│   ├── content.css        # Content area, components
│   └── code.css           # Syntax highlighting
├── js/
│   ├── main.js            # Theme toggle, tabs, navigation
│   └── search.js          # Client-side search
├── pages/
│   ├── installation.html
│   ├── quick-start.html
│   ├── configuration.html
│   ├── architecture.html
│   ├── ai-assistant.html
│   ├── api-reference.html
│   └── ...
├── assets/                 # Images, icons
├── serve.go               # Local development server
└── README.md              # This file
```

## Adding New Pages

1. Create a new HTML file in `pages/`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Page Title - k13d Documentation</title>
    <link rel="stylesheet" href="../css/main.css">
    <link rel="stylesheet" href="../css/sidebar.css">
    <link rel="stylesheet" href="../css/content.css">
    <link rel="stylesheet" href="../css/code.css">
</head>
<body>
    <!-- Copy header and sidebar from another page -->

    <main class="main-content">
        <article class="content">
            <h1>Page Title</h1>
            <!-- Your content here -->
        </article>
    </main>

    <script src="../js/main.js"></script>
    <script src="../js/search.js"></script>
</body>
</html>
```

2. Add the page to the sidebar navigation in all pages.

3. Update the search index in `js/search.js`.

## Design System

### Colors (Tokyo Night)

| Variable | Color | Usage |
|----------|-------|-------|
| `--accent-blue` | #7aa2f7 | Primary actions, links |
| `--accent-purple` | #bb9af7 | Keywords, secondary |
| `--accent-cyan` | #7dcfff | Hover states, code |
| `--accent-green` | #9ece6a | Success, strings |
| `--accent-yellow` | #e0af68 | Warnings |
| `--accent-red` | #f7768e | Errors, danger |

### Components

- **Callouts**: `.callout .callout-info|warning|danger|success`
- **Badges**: `.badge .badge-blue|purple|green|orange|red`
- **Buttons**: `.btn .btn-primary|secondary`
- **Code Tabs**: `.code-tabs .tab-btn .tab-content`
- **Feature Cards**: `.features-grid .feature-card`

## Building for Production

The site is static HTML/CSS/JS and can be deployed to any static hosting:

- GitHub Pages
- Netlify
- Vercel
- CloudFlare Pages
- Any web server

Simply copy the entire `docs-site/` directory to your hosting provider.

## Theme Support

The site supports light/dark themes via a toggle button. Theme preference is saved in localStorage.

To set default theme, modify the initial value in `js/main.js`:

```javascript
const savedTheme = localStorage.getItem('theme') || 'dark';  // or 'light'
```
