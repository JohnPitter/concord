# Phase 1.2 — Void Design System

## Goal

Scaffold the Svelte 5 frontend and implement the Void design system with color tokens, typography, and 9 base UI components.

## Tasks

### 1. Scaffold frontend (Svelte 5 + Vite + TailwindCSS v4 + TypeScript)

- Initialize `frontend/` with `npm create vite` (svelte-ts template)
- Install dependencies: tailwindcss v4, @fontsource/inter, @fontsource/jetbrains-mono
- Configure vite.config.ts, svelte.config.js, tsconfig.json
- Configure TailwindCSS v4 with Void tokens
- Create app.html, main.ts, App.svelte (layout shell with Void background)
- Create app.css with all Void CSS custom properties from ARCHITECTURE.md

### 2. Implement base components

All in `frontend/src/lib/components/ui/`. Each component:
- Uses Svelte 5 runes syntax ($props, snippets)
- Has ARIA attributes and keyboard navigation
- Uses CSS custom properties for theming
- Has CSS transitions (60fps)

Components:
1. **Button.svelte** — variants: solid/outline/ghost/danger, sizes: sm/md/lg, loading state
2. **Input.svelte** — types: text/password/search, error state, icon slot
3. **Modal.svelte** — open/close, title, Escape key, backdrop click, focus trap
4. **Badge.svelte** — variants: default/success/warning/danger
5. **Avatar.svelte** — image/initials fallback, sizes, status indicator (online/idle/dnd/offline)
6. **Tooltip.svelte** — positions: top/bottom/left/right, delay
7. **Toggle.svelte** — checked/disabled, label, accessible
8. **Dropdown.svelte** — items list, keyboard nav, click outside
9. **Card.svelte** — padding, interactive hover variant

### 3. Verify build works with Wails

- Run `wails dev` and verify Void theme renders
- Verify all components render correctly

### 4. Update CHANGELOG.md
