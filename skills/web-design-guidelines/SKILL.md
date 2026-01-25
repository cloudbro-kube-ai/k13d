---
name: web-design-guidelines
description: Review UI code for compliance with web design best practices. Use this skill when auditing interfaces for accessibility, performance, UX, or when improving web UI code quality. Based on Vercel's Web Interface Guidelines.
version: 1.0.0
---

# Web Design Guidelines Skill

This skill provides comprehensive guidelines for reviewing and improving web UI implementations.

## When to Use

- Auditing UI components for accessibility compliance
- Reviewing web interfaces for UX best practices
- Optimizing performance of web applications
- Ensuring consistent design patterns

## Guidelines Categories

### 1. Accessibility

- **Icon buttons**: Must have `aria-label` attribute
- **Form controls**: Need associated labels or ARIA labels
- **Interactive elements**: Require keyboard event handlers (`onKeyDown`/`onKeyUp`)
- **Semantic HTML**: Prefer `<button>` over `<div onClick>`
- **Images**: Include descriptive `alt` text
- **Decorative icons**: Use `aria-hidden="true"`
- **Color contrast**: Ensure WCAG 2.1 AA compliance (4.5:1 for text)

### 2. Focus States

- **Visible indicators**: All interactive elements need visible focus states
- **Never remove outlines**: Avoid `outline: none` without providing alternatives
- **Use `:focus-visible`**: Prefer over `:focus` for keyboard-only indication
- **Group focus**: Use `:focus-within` for compound components

### 3. Forms

- **Autocomplete**: All inputs should have `autocomplete` attribute
- **Name attribute**: Required for form submission and accessibility
- **Input types**: Use semantic types (`email`, `tel`, `url`, etc.)
- **No paste blocking**: Never prevent pasting in password fields
- **Clickable labels**: Ensure labels trigger input focus
- **Spellcheck**: Disable for sensitive fields (`spellcheck="false"`)
- **Submit buttons**: Keep enabled until request starts

### 4. Animation & Motion

- **Reduced motion**: Respect `prefers-reduced-motion` media query
- **Performant properties**: Only animate `transform` and `opacity`
- **Avoid `transition: all`**: Specify exact properties
- **Interruptible**: Allow users to stop/skip animations

### 5. Typography

- **Proper ellipsis**: Use `…` not `...`
- **Smart quotes**: Use curly quotes (`"` `"`) not straight quotes
- **Non-breaking spaces**: Use `&nbsp;` for measurements ("100&nbsp;GB")
- **Tabular numbers**: Use `font-variant-numeric: tabular-nums` for number columns
- **Line height**: Minimum 1.5 for body text

### 6. Content Handling

- **Text overflow**: Support truncation with `text-overflow: ellipsis`
- **Empty states**: Design and implement empty/zero-data states
- **Variable lengths**: Test with both short and long content

### 7. Images

- **Explicit dimensions**: Always specify `width` and `height`
- **Lazy loading**: Use `loading="lazy"` for below-fold images
- **Priority loading**: Use `loading="eager"` for above-fold critical images
- **Responsive**: Use `srcset` for different device sizes

### 8. Performance

- **List virtualization**: Required for lists exceeding 50 items
- **Layout thrashing**: Avoid reading layout during render
- **DOM batching**: Batch multiple DOM operations
- **Resource hints**: Use `preconnect` for external domains
- **Font loading**: Preload critical fonts

### 9. Navigation & State

- **URL sync**: Reflect application state in URL
- **Deep linking**: Enable direct access to stateful views
- **Destructive actions**: Require confirmation dialogs
- **Back button**: Support browser navigation

### 10. Touch & Mobile

- **Touch action**: Use `touch-action: manipulation` to remove tap delay
- **Tap highlight**: Set appropriate `-webkit-tap-highlight-color`
- **Modal scroll**: Use `overscroll-behavior: contain` for modals
- **Selection**: Manage text selection during drag operations
- **Hit targets**: Minimum 44x44px for touch targets

### 11. Dark Mode

- **System preference**: Respect `prefers-color-scheme`
- **Color variables**: Use CSS custom properties for theming
- **Consistent contrast**: Maintain readability in both modes
- **Image handling**: Consider dark mode variants for images

### 12. Internationalization (i18n)

- **Text direction**: Support RTL with `dir` attribute
- **Language attribute**: Set `lang` on `<html>`
- **Date/number formatting**: Use `Intl` API
- **Avoid text in images**: Keep text separate for translation

### 13. Anti-Patterns to Avoid

- Using `div` for interactive elements instead of semantic elements
- Removing focus outlines without replacement
- Blocking paste in password fields
- Using `transition: all` instead of specific properties
- Hardcoding text instead of using translation keys
- Not testing with screen readers

## Review Checklist

When reviewing UI code, check for:

1. [ ] All interactive elements are keyboard accessible
2. [ ] Form inputs have proper labels and autocomplete
3. [ ] Focus states are visible and clear
4. [ ] Animations respect reduced motion preferences
5. [ ] Images have alt text and dimensions
6. [ ] Lists are virtualized if large
7. [ ] Empty states are handled
8. [ ] Dark mode is supported
9. [ ] Touch targets are adequately sized

## References

- [Vercel Web Interface Guidelines](https://github.com/vercel-labs/web-interface-guidelines)
- [WCAG 2.1 Guidelines](https://www.w3.org/TR/WCAG21/)
- [MDN Accessibility Guide](https://developer.mozilla.org/en-US/docs/Web/Accessibility)
