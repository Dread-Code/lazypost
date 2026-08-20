# lazypost Landing Page — Design Spec

- **Date:** 2026-08-19
- **Status:** Approved (Direction A — "Faithful Terminal", with Tailwind amendment)
- **Author:** Dread-Code

## Goal

A single-page landing site for **lazypost** (a terminal API client, Posting-inspired, built with Go + Bubble Tea) published on GitHub Pages at `https://dread-code.github.io/lazypost/`. The page should closely follow the structure and visual language of https://superfile.dev/.

## Context

- Product repo: `Dread-Code/lazypost` (local checkout dir is `postgo`), Go 1.25+, MIT.
- install.sh: `curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh`; binaries on GitHub Releases for macOS/Linux (arm64+amd64); `lazypost -version`; `go build -o lazypost .`
- README contains a real app screenshot (`https://github.com/user-attachments/assets/4a1e695d-ed02-4970-9d5c-162a5f65b6b3`) to reuse on the page.
- The Go codebase is untouched; all work happens in a new `site/` directory plus one workflow file.

## Location & Deployment

- New Astro project at `site/` inside this repo. Nothing else in the repo changes.
- New workflow `.github/workflows/pages.yml`: on push to `main` (paths: `site/**`, `.github/workflows/pages.yml`), checkout → setup-node (node 22, cache npm, working-directory `site`) → `npm ci` → `npm run build` → `actions/upload-pages-artifact` (path `site/dist`) → `actions/deploy-pages`.
- Permissions: `id-token: write`, `pages: write`, `contents: read`.
- Repository settings must set Pages → Source → "GitHub Actions" (user action, noted in README or deployment comments).
- `astro.config.mjs` sets `site: 'https://dread-code.github.io'` and `base: '/lazypost/'` so all asset links work at the subpath. No `output: server` — plain static.

## Tech Stack

- **Astro 5** ("minimal" template, static output). No TypeScript, no framework components.
- **Tailwind CSS v4** via the official Vite plugin (`@tailwindcss/vite` added to `astro.config.mjs`, `@import "tailwindcss";` + `@theme { ... }` tokens in `src/styles/global.css`).
- Fonts: system UI stack for body; `ui-monospace, SF Mono, Menlo, Consolas` stack for terminal/code accents. No webfont downloads.
- Only JS on the page: the install-tab switcher (a few lines of vanilla JS in the component, progressive enhancement — first tab visible without JS).

## Design Tokens (`@theme` in global.css)

- Colors:
  - `--color-bg: #0c0f16` (page background)
  - `--color-panel: #0a0d13` (terminal inverse/sunken)
  - `--color-raised: #10141d` (cards, panes)
  - `--color-line: #1d222c` (hairlines)
  - `--color-line2: #232936` (stronger borders)
  - `--color-text: #c3c9d4` (body)
  - `--color-muted: #8b93a4` (secondary text)
  - `--color-faint: #5f6878` (tertiary)
  - `--color-accent: #7ec699` (green: prompts, `//`, `>`, method labels, active states)
  - `--color-gold: #e5c07b` (code literals/strings)
  - `--color-navbar: #141823` (terminal title bar)
- Font families: `--font-sans` (system-ui stack), `--font-mono` (ui-monospace stack).
- Layout: content max-width 1080px, centered, generous vertical section rhythm.

## Page Structure & Copy

Single page `src/pages/index.astro` composed of components under `src/components/`:

1. **Nav.astro** — left: `lazypost` wordmark (accent terminal block `▮`). Right: `github` link (https://github.com/Dread-Code/lazypost), `star` inline shields.io dynamic badge (stars: `https://img.shields.io/github/stars/Dread-Code/lazypost.svg?style=flat&label=stars&color=7ec699`).
2. **Hero.astro** — giant `lazypost` title, tagline *"An API client that lives in your terminal."*, CTAs `$ get started` (anchor → `#install`) and `view on github`. Below: **TerminalWindow** (see component), then badges row: `★ {stars}` · `macOS · Linux` · `MIT License` · `Go 1.25+` · `v0.1.0`.
3. **TerminalWindow.astro** — hand-built HTML/CSS recreation of the TUI: title bar (`● ● ●` traffic lights + `lazypost — ~/collections`), three panes: left collection tree (`▾ users`, `list.yaml`, `one.yaml`, `▸ posts`, `▸ environments`), center request editor (`POST {{host}}/posts`, headers `Content-Type: application/json`, body `{"title": "hello"}`), right response pane (`200 OK · 142ms · 1.2kb`, JSON with gold strings / green keys), bottom status line `— NORMAL — · ctrl+l url · ctrl+r send · ? help`. Rendered with mono font, `panel` background, `line` borders.
4. **WhatIs.astro** — `// what is lazypost` header; 2–3 sentence description: Posting-inspired TUI built with Go and Bubble Tea; requests are plain YAML files in a directory tree you can version-control, diff, and share; link `see the docs →` (README on GitHub).
5. **Features.astro** — `// features` header; grid of 6 cards, each with `>` prefix + title + one-line description:
   - `> vim editing` — motions, operators, visual selection, clipboard yank
   - `> yaml collections` — a folder you can git diff and share
   - `> lua scripting` — sandboxed pre/post hooks with a session store
   - `> environments` — `{{variable}}` interpolation in any field
   - `> importers` — migrate from Postman and Insomnia
   - `> themes` — dracula, catppuccin, nord, tokyonight + custom YAML
6. **InstallTabs.astro** — `$ install lazypost` header. Tabs (buttons) + one code block visible at a time:
   - **curl | sh**: `$ curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh`
   - **pin a version**: `$ curl -fsSL https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh | sh -s -- v0.1.0` (+ note `PREFIX=/usr/local` override)
   - **go build**: `$ go build -o lazypost .` (Go 1.25+)
   Below tabs: `# then run lazypost` note. Tabs = vanilla JS toggle (class swap + `aria-selected`); first tab visible without JS.
7. **Showcase.astro** — `$ lazypost` header ("a look around"). 3–4 static terminal-styled panels (title bar + body), real content:
   - **vim editing** — keys in a mini keymap grid (`hjkl wbe 0$^ ggG %`, `dd yy dw y$`, `v / V`)
   - **lua chaining** — pre script lines `req.headers["X-Session"] = store.get("token")` and post lines `store.set("last_id", id)`
   - **curl import** — `paste curl …` line + `ctrl+g` export hint
   - **themes** — 8 color swatches (dracula, catppuccin, solarized, gruvbox, nord, tokyonight, one-dark, monokai)
   The real README screenshot (downloaded to `src/assets/lazypost-screenshot.png`) is the first showcase panel.
8. **Community.astro** — centered strip: *"Open source, MIT licensed. Built by developers, for developers. Contributions, themes and feedback are always welcome."* + `star on github` button.
9. **Footer.astro** — `lazypost` wordmark, `MIT License`, `by Dread-Code`, GitHub link.

## Behavior & Edge Cases

- Smooth scrolling via CSS `scroll-behavior: smooth`; `#install` anchor works without JS.
- No-JS: tabs degrade to showing the curl|sh block; all content remains visible in DOM.
- Shields badge requires network — acceptable for a landing page; it is an `<img>`, so it never breaks layout when offline.
- All external links `target="_blank" rel="noopener"` except in-page anchors.
- Hero terminal and showcase panels are static markup — no animation beyond a subtle CSS pulse on the cursor block (optional; must respect `prefers-reduced-motion`).

## Out of Scope

- Search (⌘K), i18n, docs/blog site, dark/light toggle (page is dark by design), GIF demos, analytics.
- Changes to the Go codebase or existing CI (`ci.yml`, `release.yml`) beyond adding `pages.yml`.

## Verification

1. `cd site && npm ci && npm run build` — succeeds, output in `site/dist`.
2. `npm run preview` — page renders, all sections present, terminal window and tabs work; assets resolve relative to `/lazypost/`.
3. Push to `main`; `pages.yml` runs green; `https://dread-code.github.io/lazypost/` shows the page.
4. Quick link audit: GitHub link, install.sh URL, README links, shields badge.
