#!/usr/bin/env python3
"""Replace hardcoded dark colors in CSS files with CSS variables."""

import re
import sys

def fix_views_css(content):
    """Replace hardcoded dark colors in views.css with CSS variables."""

    # ========== GRADIENT REPLACEMENTS (order matters - longer patterns first) ==========

    # Login container background
    content = content.replace(
        'linear-gradient(135deg, #0d1117 0%, #161b22 50%, #0d1117 100%)',
        'linear-gradient(135deg, var(--bg-deep) 0%, var(--bg-primary) 50%, var(--bg-deep) 100%)'
    )

    # App container gradient
    content = content.replace(
        "linear-gradient(180deg, var(--bg-primary) 0%, #12131a 100%)",
        "linear-gradient(180deg, var(--bg-primary) 0%, var(--bg-deep) 100%)"
    )

    # Panel gradients (most common pattern)
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Panel subtle gradients
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.8) 0%, rgba(26, 27, 38, 0.9) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Header gradients (horizontal)
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.8) 0%, rgba(26, 27, 38, 0.9) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Tab gradients
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.6) 0%, rgba(26, 27, 38, 0.8) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # AI input container gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.5) 0%, rgba(26, 27, 38, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Top bar gradient
    content = content.replace(
        'linear-gradient(135deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(135deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Main panel gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(26, 27, 38, 0.5) 0%, rgba(18, 19, 26, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-secondary) 0%, var(--bg-deep) 100%)'
    )

    # Table header gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(65, 72, 104, 0.6) 0%, rgba(36, 40, 59, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-tertiary) 0%, var(--surface-primary) 100%)'
    )

    # Table header hover gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, rgba(36, 40, 59, 0.9) 100%)',
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, var(--surface-primary) 100%)'
    )

    # AI messages subtle gradient (keep but fix)
    content = content.replace(
        'linear-gradient(180deg, transparent 0%, rgba(0, 0, 0, 0.1) 100%)',
        'linear-gradient(180deg, transparent 0%, var(--surface-code) 100%)'
    )

    # ========== DIRECT BACKGROUND REPLACEMENTS ==========

    # Login box background
    content = content.replace(
        'background: rgba(22, 27, 34, 0.9);',
        'background: var(--surface-primary);'
    )

    # Input backgrounds (rgba(13, 17, 23, ...))
    content = content.replace(
        'background: rgba(13, 17, 23, 0.8);',
        'background: var(--surface-input);'
    )
    content = content.replace(
        'background: rgba(13, 17, 23, 1);',
        'background: var(--surface-input-solid);'
    )
    content = content.replace(
        'background: rgba(13, 17, 23, 0.5);',
        'background: var(--surface-input);'
    )
    content = content.replace(
        'background: rgba(13, 17, 23, 0.6);',
        'background: var(--surface-input);'
    )

    # Input backgrounds (rgba(26, 27, 38, ...))
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 1);',
        'background: var(--surface-input-solid);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 0.9);',
        'background: var(--surface-secondary);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 0.6);',
        'background: var(--surface-tertiary);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 0.5);',
        'background: var(--surface-tertiary);'
    )

    # Tertiary backgrounds (rgba(65, 72, 104, ...))
    content = content.replace(
        'background: rgba(65, 72, 104, 0.6);',
        'background: var(--surface-tertiary);'
    )
    content = content.replace(
        'background: rgba(65, 72, 104, 0.8);',
        'background: var(--surface-elevated);'
    )
    content = content.replace(
        'background: rgba(65, 72, 104, 0.5);',
        'background: var(--bg-hover);'
    )

    # Table backgrounds
    content = content.replace(
        'background: rgba(36, 40, 59, 0.4);',
        'background: var(--surface-tertiary);'
    )
    content = content.replace(
        'background: rgba(36, 40, 59, 0.3);',
        'background: var(--surface-tertiary);'
    )

    # Terminal/log body backgrounds
    content = content.replace(
        'background: #1a1b26;',
        'background: var(--bg-primary);'
    )
    content = content.replace(
        'background: #0d1117;',
        'background: var(--bg-deep);'
    )

    # Code block backgrounds
    content = content.replace(
        'background: rgba(0, 0, 0, 0.3);',
        'background: var(--surface-code);'
    )

    # Scrollbar track
    content = content.replace(
        'background: rgba(26, 27, 38, 0.5);',
        'background: var(--bg-primary);'
    )

    # ========== BORDER REPLACEMENTS ==========

    # Borders using rgba(65, 72, 104, ...)
    content = content.replace(
        'border-bottom: 1px solid rgba(65, 72, 104, 0.3);',
        'border-bottom: 1px solid var(--border-color);'
    )
    content = content.replace(
        'border-right: 1px solid rgba(65, 72, 104, 0.4);',
        'border-right: 1px solid var(--border-color);'
    )

    # ========== TEXT COLOR REPLACEMENTS ==========

    # Dark text on accent-colored badges
    content = content.replace(
        'color: #1a1b26;',
        'color: var(--text-on-accent);'
    )

    # About version row border (hardcoded white opacity)
    content = content.replace(
        'border-bottom: 1px solid rgba(255, 255, 255, 0.05);',
        'border-bottom: 1px solid var(--border-color);'
    )

    # ========== REMOVE OUTDATED LIGHT THEME OVERRIDES ==========
    # The existing light theme overrides in views.css are now redundant
    # since we use CSS variables. Remove them.
    # Lines 4902-4967 contain [data-theme="light"] overrides
    # Lines 5694-5707 contain @media (prefers-color-scheme: light) overrides

    # Remove the light theme overrides section
    content = re.sub(
        r'/\* =+\s*\n\s*\* Light Theme Overrides\s*\n\s*\* Override hardcoded dark colors for light mode\s*\n\s*\* =+ \*/\s*\n'
        r'.*?'
        r'\[data-theme="light"\] \.command-bar \{[^}]*\}',
        '/* Light Theme Overrides removed - now using CSS variables */',
        content,
        flags=re.DOTALL
    )

    # Remove the prefers-color-scheme light section at the end
    content = re.sub(
        r'@media \(prefers-color-scheme: light\) \{\s*\n'
        r'\s*:root:not\(\[data-theme\]\) \.login-container \{[^}]*\}\s*\n'
        r'\s*:root:not\(\[data-theme\]\) \.login-box \{[^}]*\}\s*\n'
        r'\s*:root:not\(\[data-theme\]\) \.sidebar \{[^}]*\}\s*\n'
        r'\s*\}',
        '/* System light preference overrides removed - now using CSS variables */',
        content,
        flags=re.DOTALL
    )

    return content


def fix_layout_css(content):
    """Replace hardcoded dark colors in layout.css with CSS variables."""

    # App container gradient
    content = content.replace(
        "linear-gradient(180deg, var(--bg-primary) 0%, #12131a 100%)",
        "linear-gradient(180deg, var(--bg-primary) 0%, var(--bg-deep) 100%)"
    )

    # Top bar gradient
    content = content.replace(
        'linear-gradient(135deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(135deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Sidebar gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.8) 0%, rgba(26, 27, 38, 0.9) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Nav item hover
    content = content.replace(
        'background: rgba(65, 72, 104, 0.5);',
        'background: var(--bg-hover);'
    )

    # Nav item count
    content = content.replace(
        'background: rgba(65, 72, 104, 0.8);',
        'background: var(--surface-elevated);'
    )

    # Main panel gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(26, 27, 38, 0.5) 0%, rgba(18, 19, 26, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-secondary) 0%, var(--bg-deep) 100%)'
    )

    # AI panel gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Header gradients
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.8) 0%, rgba(26, 27, 38, 0.9) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # AI input container gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.5) 0%, rgba(26, 27, 38, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # AI input background
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 1);',
        'background: var(--surface-input-solid);'
    )

    # Message assistant background
    content = content.replace(
        'background: rgba(65, 72, 104, 0.6);',
        'background: var(--surface-tertiary);'
    )

    # Message pre background
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )

    # Summary item background
    content = content.replace(
        'background: rgba(26, 27, 38, 0.6);',
        'background: var(--surface-tertiary);'
    )

    # Scrollbar backgrounds
    content = content.replace(
        'background: rgba(26, 27, 38, 0.5);',
        'background: var(--bg-primary);'
    )

    # Table backgrounds
    content = content.replace(
        'background: rgba(36, 40, 59, 0.4);',
        'background: var(--surface-tertiary);'
    )

    # Table header
    content = content.replace(
        'linear-gradient(180deg, rgba(65, 72, 104, 0.6) 0%, rgba(36, 40, 59, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-tertiary) 0%, var(--surface-primary) 100%)'
    )
    content = content.replace(
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, rgba(36, 40, 59, 0.9) 100%)',
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, var(--surface-primary) 100%)'
    )

    # td border
    content = content.replace(
        'border-bottom: 1px solid rgba(65, 72, 104, 0.3);',
        'border-bottom: 1px solid var(--border-color);'
    )

    # Pagination backgrounds
    content = content.replace(
        'background: rgba(65, 72, 104, 0.5);',
        'background: var(--bg-hover);'
    )

    # Page size select
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )

    # Tab gradient
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.6) 0%, rgba(26, 27, 38, 0.8) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Modal gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Remove the Light Theme Overrides section at the bottom of layout.css
    # since the CSS variables now handle theme switching automatically
    content = re.sub(
        r'/\* =+\s*\n\s*\* Light Theme Overrides\s*\n\s*\* Override hardcoded dark backgrounds for light mode\s*\n\s*\* =+ \*/\s*\n'
        r'.*$',
        '/* Light Theme Overrides removed - now using CSS variables */\n',
        content,
        flags=re.DOTALL
    )

    return content


def fix_components_css(content):
    """Replace hardcoded dark colors in components.css with CSS variables."""

    # Scrollbar track
    content = content.replace(
        'background: rgba(26, 27, 38, 0.5);',
        'background: var(--bg-primary);'
    )

    # Table background
    content = content.replace(
        'background: rgba(36, 40, 59, 0.4);',
        'background: var(--surface-tertiary);'
    )

    # Table header gradient
    content = content.replace(
        'linear-gradient(180deg, rgba(65, 72, 104, 0.6) 0%, rgba(36, 40, 59, 0.8) 100%)',
        'linear-gradient(180deg, var(--surface-tertiary) 0%, var(--surface-primary) 100%)'
    )

    # Table header hover
    content = content.replace(
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, rgba(36, 40, 59, 0.9) 100%)',
        'linear-gradient(180deg, rgba(122, 162, 247, 0.2) 0%, var(--surface-primary) 100%)'
    )

    # td border
    content = content.replace(
        'border-bottom: 1px solid rgba(65, 72, 104, 0.3);',
        'border-bottom: 1px solid var(--border-color);'
    )

    # Pagination backgrounds
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.8) 0%, rgba(26, 27, 38, 0.9) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )
    content = content.replace(
        'background: rgba(65, 72, 104, 0.5);',
        'background: var(--bg-hover);'
    )

    # Page size select
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )

    # Tab gradients
    content = content.replace(
        'linear-gradient(90deg, rgba(36, 40, 59, 0.6) 0%, rgba(26, 27, 38, 0.8) 100%)',
        'linear-gradient(90deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Modal
    content = content.replace(
        'linear-gradient(180deg, rgba(36, 40, 59, 0.95) 0%, rgba(26, 27, 38, 0.98) 100%)',
        'linear-gradient(180deg, var(--surface-primary) 0%, var(--surface-secondary) 100%)'
    )

    # Form inputs
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )
    content = content.replace(
        'background: rgba(26, 27, 38, 1);',
        'background: var(--surface-input-solid);'
    )

    # Message content pre
    content = content.replace(
        'background: rgba(26, 27, 38, 0.8);',
        'background: var(--surface-input);'
    )

    # Assistant message
    content = content.replace(
        'background: rgba(65, 72, 104, 0.6);',
        'background: var(--surface-tertiary);'
    )

    # color on accent badges
    content = content.replace(
        'color: #1a1b26;',
        'color: var(--text-on-accent);'
    )

    return content


def main():
    files = {
        '/Users/youngjukim/Desktop/k13d/pkg/web/static/css/views.css': fix_views_css,
        '/Users/youngjukim/Desktop/k13d/pkg/web/static/css/layout.css': fix_layout_css,
        '/Users/youngjukim/Desktop/k13d/pkg/web/static/css/components.css': fix_components_css,
    }

    for filepath, fix_func in files.items():
        print(f"Processing {filepath}...")
        with open(filepath, 'r') as f:
            content = f.read()

        original = content
        content = fix_func(content)

        if content != original:
            with open(filepath, 'w') as f:
                f.write(content)
            print(f"  Updated {filepath}")
        else:
            print(f"  No changes needed in {filepath}")

    print("\nDone! Checking for remaining hardcoded dark colors...")

    # Check for remaining hardcoded colors
    dark_patterns = [
        '#0d1117', '#161b22', '#12131a',
        'rgba(13, 17, 23',
        'rgba(22, 27, 34',
        'rgba(26, 27, 38',
        'rgba(36, 40, 59',
        'rgba(65, 72, 104',
        'rgba(18, 19, 26',
    ]

    for filepath in files:
        with open(filepath, 'r') as f:
            content = f.read()
        for pattern in dark_patterns:
            lines = [i+1 for i, line in enumerate(content.split('\n')) if pattern in line]
            if lines:
                print(f"  REMAINING in {filepath.split('/')[-1]}: '{pattern}' at lines {lines}")


if __name__ == '__main__':
    main()
