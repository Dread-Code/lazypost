# lazypost Landing Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a single-page landing site for lazypost on GitHub Pages at `https://dread-code.github.io/lazypost/`, styled after superfile.dev (Direction A — "Faithful Terminal"), built with Astro 5 + Tailwind v4, deployed by GitHub Actions from `site/` in this repo.

**Architecture:** A static Astro 5 project in `site/` — one page composed of eight .astro components, a Tailwind v4 `@theme` token system in `global.css`, one inline vanilla-JS tab switcher, and a GitHub Actions workflow that builds and deploys `site/dist` to Pages. All content is hand-written HTML/CSS replicating the lazypost TUI as a terminal window; the only fetched asset is a shields.io star badge.

**Tech Stack:** Astro 5 (static), Tailwind CSS v4 via `@tailwindcss/vite`, Node 22 + built-in `node --test` for smoke tests, GitHub Actions Pages deployment.

**Spec:** `docs/superpowers/specs/2026-08-19-lazypost-landing-page-design.md` (approved). Where the plan and spec disagree the plan is wrong.

## Global Constraints

Fixed values every task must respect (from the spec):

- **Node 22+ and npm.** All npm commands run inside `site/` (use `workdir`/`cd site`). Node built-in test runner (`node --test tests/`) — no test dependencies.
- **Dependencies:** `astro@^5.0.0`, `tailwindcss@^4.0.0`, `@tailwindcss/vite@^4.0.0`. Nothing else.
- **astro.config.mjs:** `site: 'https://dread-code.github.io'`, `base: '/lazypost/'`, vite plugin `tailwindcss()`.
- **Design tokens** (`@theme` in `src/styles/global.css`): `--color-bg: #0c0f16`, `--color-panel: #0a0d13`, `--color-raised: #10141d`, `--color-line: #1d222c`, `--color-line2: #232936`, `--color-text: #c3c9d4`, `--color-muted: #8b93a4`, `--color-faint: #5f6878`, `--color-accent: #7ec699`, `--color-gold: #e5c07b`, `--color-navbar: #141823`; `--font-sans`: system-ui stack; `--font-mono: ui-monospace, "SF Mono", Menlo, Consolas, monospace`.
- **Copy is fixed.** Title `lazypost`; tagline `An API client that lives in your terminal.`; CTAs `$ get started` (href `#install`) and `view on github`; section headers `// what is lazypost`, `// features`, `$ install lazypost`, `$ lazypost`; community line `Open source, MIT licensed. Built by developers, for developers. Contributions, themes and feedback are always welcome.`; CTA `★ star on github`; footer `lazypost · MIT License · by Dread-Code`.
- **Links:** github `https://github.com/Dread-Code/lazypost`; stars badge `https://img.shields.io/github/stars/Dread-Code/lazypost.svg?style=flat&label=stars&color=7ec699`; install.sh `https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh`; screenshot `https://github.com/user-attachments/assets/4a1e695d-ed02-4970-9d5c-162a5f65b6b3`.
- **Every external link has `target="_blank" rel="noopener"`.** In-page anchors (`#install`, `#top`) do not.
- No TypeScript, no framework components, no webfonts, no JS beyond the install-tab switcher.
- Tests live in `site/tests/smoke.test.mjs`. Shared helpers: `html()` (returns `dist/index.html` content or `''`), `css()` (returns the single `dist/_astro/*.css` content or `''`), `read(p)` (returns content of any file relative to the repo root, or `''` if missing), constants `DIST`, `ROOT`.
- `src/pages/index.astro` is wired up progressively: import only components that exist at that point.
- Commit after every task. Repo root is `/Users/dread-code/Dev/postgo`; git worktree not required.

## File Structure

```
.github/workflows/pages.yml                        T9  GitHub Pages deploy workflow
site/
  package.json                                     T1  astro + tailwind deps, scripts
  astro.config.mjs                                 T1  site/base/tailwind plugin
  tsconfig.json                                    T1  astro base config
  .gitignore                                       T1  node_modules/dist/.astro
  src/
    styles/global.css                              T2  tailwind import + @theme tokens + base
    data/site.js                                   T3  exported constants (github, starsBadge, installUrl)
    assets/favicon.svg                             T1  terminal-block icon
    assets/lazypost-screenshot.png                 T7  downloaded from GitHub user-attachments
    pages/index.astro                              T1  skeleton → grows each task
    components/
      Nav.astro            Nav                      T3
      Hero.astro           Hero + terminal + badges T3 (T4 adds TerminalWindow)
      TerminalWindow.astro TUI recreation           T4
      WhatIs.astro         // what is lazypost      T5
      Features.astro       6 > cards               T5
      InstallTabs.astro    $ install + tabs JS      T6
      Showcase.astro       $ lazypost panels        T7
      Community.astro      star CTA strip           T8
      Footer.astro                                 T8
  tests/smoke.test.mjs                             T1  node:test harness, grows each task
```

---

### Task 1: Scaffold the Astro + Tailwind project

**Files:**
- Create: `site/package.json`, `site/astro.config.mjs`, `site/tsconfig.json`, `site/.gitignore`, `site/src/assets/favicon.svg`, `site/src/pages/index.astro`, `site/tests/smoke.test.mjs`

**Interfaces:**
- Consumes: nothing (repo root only, per Global Constraints).
- Produces: `package.json` with scripts `dev` / `build` / `preview` / `test`; `astro.config.mjs` with `base: '/lazypost/'`; `src/pages/index.astro` (title `lazypost`, favicon import); test harness helpers `html()`, `css()`, `read(p)`, constants `DIST`, `ROOT`.

- [ ] **Step 1: Write the failing tests**

Create `site/tests/smoke.test.mjs`:

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

export const ROOT = join(import.meta.dirname, '..', '..');
export const DIST = join(import.meta.dirname, '..', 'dist');

export const html = () => {
  const p = join(DIST, 'index.html');
  return existsSync(p) ? readFileSync(p, 'utf8') : '';
};

export const css = () => {
  const dir = join(DIST, '_astro');
  if (!existsSync(dir)) return '';
  const f = readdirSync(dir).find((n) => n.endsWith('.css'));
  return f ? readFileSync(join(dir, f), 'utf8') : '';
};

export const read = (p) => {
  const full = join(ROOT, p);
  return existsSync(full) ? readFileSync(full, 'utf8') : '';
};

test('build output exists', () => {
  assert.ok(existsSync(join(DIST, 'index.html')), 'dist/index.html must exist — run npm run build first');
});

test('page has lazypost title', () => {
  assert.match(html(), /<title>lazypost<\/title>/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `node --test tests/` (in `site/`)
Expected: FAIL — `dist/index.html` does not exist.

- [ ] **Step 3: Scaffold the project**

`site/package.json`:

```json
{
  "name": "lazypost-site",
  "type": "module",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "astro dev",
    "build": "astro build",
    "preview": "astro preview",
    "test": "node --test tests/"
  },
  "dependencies": {
    "astro": "^5.0.0",
    "tailwindcss": "^4.0.0",
    "@tailwindcss/vite": "^4.0.0"
  }
}
```

`site/astro.config.mjs`:

```js
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  site: 'https://dread-code.github.io',
  base: '/lazypost/',
  vite: {
    plugins: [tailwindcss()],
  },
});
```

`site/tsconfig.json`:

```json
{
  "extends": "astro/tsconfigs/base",
  "include": [".astro/types.d.ts", "**/*"],
  "exclude": ["dist"]
}
```

`site/.gitignore`:

```
node_modules/
dist/
.astro/
```

`site/src/assets/favicon.svg`:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect width="32" height="32" rx="6" fill="#0c0f16"/><text x="16" y="23" font-family="monospace" font-size="19" font-weight="bold" fill="#7ec699" text-anchor="middle">▮</text></svg>
```

`site/src/pages/index.astro`:

```astro
---
import favicon from '../assets/favicon.svg';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
  </body>
</html>
```

Run `npm install` in `site/`.

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — dist/index.html exists, contains `<title>lazypost</title>`.

- [ ] **Step 5: Commit**

```bash
git add site/package.json site/package-lock.json site/astro.config.mjs site/tsconfig.json site/.gitignore site/src/assets/favicon.svg site/src/pages/index.astro site/tests/smoke.test.mjs
git commit -m "feat(site): scaffold astro + tailwind v4 project"
```

---

### Task 2: Global styles and design tokens

**Files:**
- Create: `site/src/styles/global.css`
- Modify: `site/src/pages/index.astro` (import css, body classes)

**Interfaces:**
- Consumes: Task 1 `index.astro` skeleton.
- Produces: `global.css` with `@import "tailwindcss"` + `@theme` tokens (exact hex values in Global Constraints) enabling utility classes `bg-bg`, `bg-panel`, `bg-raised`, `bg-navbar`, `border-line`, `border-line2`, `text-text`, `text-muted`, `text-faint`, `text-accent`, `text-gold`, `font-sans`, `font-mono`.

- [ ] **Step 1: Add the failing test**

Append to `site/tests/smoke.test.mjs`:

```js
test('css emits design tokens', () => {
  assert.match(css(), /--color-accent:/, 'compiled css must contain --color-accent');
});

test('css sets smooth scrolling', () => {
  assert.match(css(), /scroll-behavior:smooth/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — `css()` returns `''` (no `dist/_astro/*.css` yet).

- [ ] **Step 3: Implement**

Create `site/src/styles/global.css`:

```css
@import "tailwindcss";

@theme {
  --color-bg: #0c0f16;
  --color-panel: #0a0d13;
  --color-raised: #10141d;
  --color-line: #1d222c;
  --color-line2: #232936;
  --color-text: #c3c9d4;
  --color-muted: #8b93a4;
  --color-faint: #5f6878;
  --color-accent: #7ec699;
  --color-gold: #e5c07b;
  --color-navbar: #141823;

  --font-sans: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  --font-mono: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
}

html {
  scroll-behavior: smooth;
}

body {
  @apply bg-bg text-text font-sans antialiased;
}

::selection {
  background: #7ec699;
  color: #0c0f16;
}
```

Replace `site/src/pages/index.astro` with:

```astro
---
import favicon from '../assets/favicon.svg';
import '../styles/global.css';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
  </body>
</html>
```

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — compiled css contains `--color-accent:` and `scroll-behavior:smooth`.

- [ ] **Step 5: Commit**

```bash
git add site/src/styles/global.css site/src/pages/index.astro
git commit -m "feat(site): global tokens and base layout"
```

---

### Task 3: Nav, hero and badges

**Files:**
- Create: `site/src/data/site.js`, `site/src/components/Nav.astro`, `site/src/components/Hero.astro`
- Modify: `site/src/pages/index.astro`

**Interfaces:**
- Consumes: Task 2 tokens/utilities (`text-accent`, `bg-accent`, `border-line`, `border-line2`, `text-text`, `text-muted`, `text-faint`, `font-mono`).
- Produces: `site.js` exporting `github`, `starsBadge`, `installUrl` (used by Nav, Hero in this task; InstallTabs T6; Community/Footer T8); Nav.astro (`header` with wordmark, `github` link, star pill with badge `<img>`); Hero.astro (`h1` lazypost, tagline, two CTAs — `$ get started` → `#install`, `view on github`; empty spot for `<TerminalWindow />`; badges `<ul>` with 5 `<li>`s).

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('hero copy, ctas and badges render', () => {
  const h = html();
  assert.match(h, /An API client that lives in your terminal\./);
  assert.match(h, /\$ get started/);
  assert.match(h, /view on github/);
  assert.match(h, /macOS · Linux/);
  assert.match(h, /MIT License/);
  assert.match(h, /Go 1\.25\+/);
  assert.match(h, /v0\.1\.0/);
  assert.match(h, /img\.shields\.io/);
  assert.match(h, /github\.com\/Dread-Code\/lazypost/);
  assert.match(h, /href="#install"/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — hero copy absent from built page.

- [ ] **Step 3: Implement**

Create `site/src/data/site.js`:

```js
export const github = 'https://github.com/Dread-Code/lazypost';
export const starsBadge =
  'https://img.shields.io/github/stars/Dread-Code/lazypost.svg?style=flat&label=stars&color=7ec699';
export const installUrl = 'https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh';
```

Create `site/src/components/Nav.astro`:

```astro
---
import { github, starsBadge } from '../data/site.js';
---
<header id="top" class="mx-auto flex max-w-6xl items-center justify-between px-6 py-5">
  <a href="#top" class="font-mono text-sm font-bold text-text">lazypost<span class="text-accent">▮</span></a>
  <nav class="flex items-center gap-5 font-mono text-xs">
    <a href={github} target="_blank" rel="noopener" class="text-muted transition-colors hover:text-text">github</a>
    <a href={github} target="_blank" rel="noopener" class="flex items-center gap-2 rounded-full border border-line2 px-3 py-1.5 text-text transition-colors hover:border-accent">
      <span>★</span><span>star</span>
      <img src={starsBadge} alt="GitHub stars" class="h-4" />
    </a>
  </nav>
</header>
```

Create `site/src/components/Hero.astro`:

```astro
---
import { github, starsBadge } from '../data/site.js';
---
<section class="mx-auto max-w-6xl px-6 pb-8 pt-20 text-center">
  <h1 class="font-mono text-6xl font-extrabold tracking-tight text-text md:text-7xl">lazypost</h1>
  <p class="mx-auto mt-5 max-w-xl text-lg text-muted">An API client that lives in your terminal.</p>
  <div class="mt-9 flex flex-wrap items-center justify-center gap-4 font-mono text-sm">
    <a href="#install" class="rounded-lg bg-accent px-5 py-2.5 font-bold text-bg transition-opacity hover:opacity-85">$ get started</a>
    <a href={github} target="_blank" rel="noopener" class="rounded-lg border border-line2 px-5 py-2.5 text-text transition-colors hover:border-faint">view on github</a>
  </div>
  <ul class="mt-12 flex flex-wrap items-center justify-center gap-3 font-mono text-xs text-muted">
    <li class="rounded-full border border-line px-4 py-1.5">
      <img src={starsBadge} alt="GitHub stars" class="h-4" />
    </li>
    <li class="rounded-full border border-line px-4 py-1.5">macOS · Linux</li>
    <li class="rounded-full border border-line px-4 py-1.5">MIT License</li>
    <li class="rounded-full border border-line px-4 py-1.5">Go 1.25+</li>
    <li class="rounded-full border border-line px-4 py-1.5">v0.1.0</li>
  </ul>
</section>
```

Replace `site/src/pages/index.astro` with:

```astro
---
import favicon from '../assets/favicon.svg';
import '../styles/global.css';
import Nav from '../components/Nav.astro';
import Hero from '../components/Hero.astro';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
    <Nav />
    <main>
      <Hero />
    </main>
  </body>
</html>
```

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — all hero/badge assertions hold.

- [ ] **Step 5: Commit**

```bash
git add site/src/data/site.js site/src/components/Nav.astro site/src/components/Hero.astro site/src/pages/index.astro
git commit -m "feat(site): nav, hero and badges"
```

---

### Task 4: Terminal window (TUI recreation)

**Files:**
- Create: `site/src/components/TerminalWindow.astro`
- Modify: `site/src/components/Hero.astro` (mount `<TerminalWindow />`)

**Interfaces:**
- Consumes: Task 3 `Hero.astro`; tokens `bg-panel`, `bg-raised`, `bg-navbar`, `border-line`, `border-line2`, `text-accent`, `text-gold`, `text-faint`, `text-muted`, `text-text`, `font-mono`; arbitrary color utilities for the three traffic lights.
- Produces: `TerminalWindow.astro` — static HTML recreation of the lazypost TUI (title bar, three panes, status line), mounted between the CTAs and the badges row.

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('terminal window renders the TUI', () => {
  const h = html();
  assert.match(h, /lazypost — ~\/collections/);
  assert.match(h, /\{\{host\}\}\/posts/);
  assert.match(h, /Content-Type: application\/json/);
  assert.match(h, /200 OK · 142ms · 1\.2kb/);
  assert.match(h, /NORMAL/);
  assert.match(h, /ctrl\+r send/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — no terminal markup yet.

- [ ] **Step 3: Implement**

Create `site/src/components/TerminalWindow.astro`:

```astro
---
---
<div class="mx-auto mt-14 max-w-4xl text-left">
  <div class="overflow-hidden rounded-xl border border-line2 bg-panel shadow-2xl shadow-black/40">
    <div class="flex items-center gap-2 border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">
      <span class="h-3 w-3 rounded-full bg-[#ff5f57]"></span>
      <span class="h-3 w-3 rounded-full bg-[#febc2e]"></span>
      <span class="h-3 w-3 rounded-full bg-[#28c840]"></span>
      <span class="ml-3">lazypost — ~/collections</span>
    </div>
    <div class="grid grid-cols-1 gap-3 p-4 font-mono text-[13px] leading-6 sm:grid-cols-[9rem_1fr] lg:grid-cols-[9rem_1fr_11rem]">
      <div class="rounded-lg border border-line bg-raised p-3 text-xs leading-6 text-faint">
        <div>▾ <span class="text-text">users</span></div>
        <div class="pl-3">list.yaml</div>
        <div class="pl-3">one.yaml</div>
        <div class="text-accent">▸ posts</div>
        <div>▸ environments</div>
      </div>
      <div class="rounded-lg border border-line bg-raised p-3">
        <div><span class="font-bold text-accent">POST</span> <span class="text-muted">{{host}}/posts</span></div>
        <div class="mt-3 text-xs text-faint">headers: Content-Type: application/json</div>
        <div class="mt-1 text-xs text-faint">body: <span class="text-gold">{"title": "hello"}</span></div>
      </div>
      <div class="rounded-lg border border-line bg-raised p-3 text-xs leading-6 lg:block hidden">
        <div class="text-muted">200 OK · 142ms · 1.2kb</div>
        <div class="mt-2 text-faint">&nbsp;</div>
        <div class="text-faint"><span class="text-accent">"title"</span>: <span class="text-gold">"hello"</span>,</div>
        <div class="text-faint"><span class="text-accent">"id"</span>: <span class="text-gold">42</span>,</div>
        <div class="text-faint"><span class="text-accent">"created"</span>: <span class="text-gold">"2026-08-19"</span></div>
      </div>
    </div>
    <div class="flex flex-wrap items-center justify-between gap-2 border-t border-line bg-navbar px-4 py-2 font-mono text-xs text-faint">
      <span>— <span class="text-accent">NORMAL</span> —</span>
      <span class="hidden sm:inline">ctrl+l url · ctrl+r send · ? help</span>
    </div>
  </div>
</div>
```

Modify `site/src/components/Hero.astro`: add the import to the frontmatter and mount the component. The frontmatter becomes:

```astro
---
import { github, starsBadge } from '../data/site.js';
import TerminalWindow from './TerminalWindow.astro';
---
```

and insert `<TerminalWindow />` between the CTA `<div>` and the badges `<ul>` (i.e. directly after the closing `</div>` of the CTA block).

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — all six terminal assertions hold. Note: `NORMAL` and `ctrl+r send` must both be present in the built `index.html` (they are — status line markup), so the two `assert.match` calls pass even though the status line is hidden below `sm` breakpoints; hidden via CSS (`hidden sm:inline`) still ships the text in the HTML.

- [ ] **Step 5: Commit**

```bash
git add site/src/components/TerminalWindow.astro site/src/components/Hero.astro
git commit -m "feat(site): terminal window hero visual"
```

### Task 5: What-is and features sections

**Files:**
- Create: `site/src/components/WhatIs.astro`, `site/src/components/Features.astro`
- Modify: `site/src/pages/index.astro`

**Interfaces:**
- Consumes: Task 2 tokens; Task 3 `site.js` (`github`).
- Produces: `WhatIs.astro` (section with `<h2>` `// what is lazypost`, description paragraph, `see the docs →` link to github README); `Features.astro` (section with `<h2>` `// features` and a grid of exactly 6 cards rendered from a frontmatter array). Card titles: `vim editing`, `yaml collections`, `lua scripting`, `environments`, `importers`, `themes`.

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('what-is section renders', () => {
  const h = html();
  assert.match(h, /\/\/ what is lazypost/);
  assert.match(h, /see the docs/);
  assert.match(h, /Posting-inspired/);
  assert.match(h, /Bubble Tea/);
});

test('features grid renders six cards', () => {
  const h = html();
  for (const title of ['vim editing', 'yaml collections', 'lua scripting', 'environments', 'importers', 'themes']) {
    assert.match(h, new RegExp(`> ${title}`), `missing feature: ${title}`);
  }
  assert.match(h, /\{\{variable\}\} interpolation/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — sections not yet in the page.

- [ ] **Step 3: Implement**

Create `site/src/components/WhatIs.astro`:

```astro
---
import { github } from '../data/site.js';
---
<section class="mx-auto max-w-6xl px-6 py-20">
  <h2 class="font-mono text-base font-bold text-accent">// what is lazypost</h2>
  <p class="mt-5 max-w-2xl text-lg leading-relaxed text-text">
    Posting-inspired TUI built with <span class="font-mono text-accent">Go</span> and
    <span class="font-mono text-accent">Bubble Tea</span>. Requests are plain YAML files in a
    directory tree — version-control, diff, and share your API collection.
  </p>
  <a href={github} target="_blank" rel="noopener" class="mt-5 inline-block font-mono text-sm text-muted transition-colors hover:text-accent">see the docs →</a>
</section>
```

Create `site/src/components/Features.astro`:

```astro
---
const features = [
  ['vim editing', 'motions, operators, visual selection, clipboard yank'],
  ['yaml collections', 'a folder you can git diff and share'],
  ['lua scripting', 'sandboxed pre/post hooks with a session store'],
  ['environments', '{{variable}} interpolation in any field'],
  ['importers', 'migrate from Postman and Insomnia'],
  ['themes', 'dracula, catppuccin, nord, tokyonight + custom YAML'],
];
---
<section class="mx-auto max-w-6xl px-6 py-20">
  <h2 class="font-mono text-base font-bold text-accent">// features</h2>
  <div class="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
    {features.map(([title, desc]) => (
      <div class="rounded-lg border border-line bg-raised p-5 transition-colors hover:border-line2">
        <div class="font-mono text-sm">
          <span class="text-accent">{'>'}</span> <span class="font-bold text-text">{title}</span>
        </div>
        <p class="mt-2 font-mono text-xs leading-relaxed text-faint">{desc}</p>
      </div>
    ))}
  </div>
</section>
```

Replace `site/src/pages/index.astro` with:

```astro
---
import favicon from '../assets/favicon.svg';
import '../styles/global.css';
import Nav from '../components/Nav.astro';
import Hero from '../components/Hero.astro';
import WhatIs from '../components/WhatIs.astro';
import Features from '../components/Features.astro';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
    <Nav />
    <main>
      <Hero />
      <WhatIs />
      <Features />
    </main>
  </body>
</html>
```

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — what-is copy and all six feature titles present (the feature cards render a literal `>` via the `{'>'}` expression, so the raw HTML matches the test regexes).

- [ ] **Step 5: Commit**

```bash
git add site/src/components/WhatIs.astro site/src/components/Features.astro site/src/pages/index.astro
git commit -m "feat(site): what-is and features sections"
```

---

### Task 6: Install section with tabs

**Files:**
- Create: `site/src/components/InstallTabs.astro`
- Modify: `site/src/pages/index.astro`

**Interfaces:**
- Consumes: Task 3 `site.js` (`installUrl`); Task 2 tokens.
- Produces: `InstallTabs.astro` — `<section id="install">` with `<h2>` `$ install lazypost`, a tablist (3 buttons: `curl | sh`, `pin a version`, `go build`) and 3 code panes; a `# then run lazypost` hint below; an inline `<script>` implementing the tab switch (class swap + `aria-selected`, activate the matching pane; first pane visible without JS).

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('install section renders tabs and commands', () => {
  const h = html();
  assert.match(h, /\$ install lazypost/);
  assert.match(h, /id="install"/);
  assert.match(h, /curl \| sh/);
  assert.match(h, /pin a version/);
  assert.match(h, /go build/);
  assert.match(h, /install\.sh \| sh/);
  assert.match(h, /sh -s -- v0\.1\.0/);
  assert.match(h, /PREFIX=\/usr\/local/);
  assert.match(h, /then run/);
  assert.match(h, /aria-selected="true"/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — no install section yet.

- [ ] **Step 3: Implement**

Create `site/src/components/InstallTabs.astro`:

```astro
---
import { installUrl } from '../data/site.js';
const tabs = [
  { id: 'curl', label: 'curl | sh', lines: [`$ ${installUrl} | sh`] },
  {
    id: 'pin',
    label: 'pin a version',
    lines: [`$ ${installUrl} | sh -s -- v0.1.0`, '# PREFIX=/usr/local overrides the install directory'],
  },
  { id: 'build', label: 'go build', lines: ['$ go build -o lazypost .', '# requires Go 1.25+'] },
];
---
<section id="install" class="mx-auto max-w-6xl scroll-mt-8 px-6 py-20">
  <h2 class="font-mono text-base font-bold text-accent">$ install lazypost</h2>
  <div class="mt-8 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div id="install-tabs" role="tablist" class="flex gap-1 border-b border-line bg-navbar px-3 pt-2">
      {tabs.map((t, i) => (
        <button
          type="button"
          role="tab"
          id={`tab-${t.id}`}
          data-tab={t.id}
          aria-controls={`panel-${t.id}`}
          aria-selected={i === 0}
          class={'rounded-t-md border-b-2 px-4 py-2 font-mono text-xs transition-colors ' + (i === 0 ? 'border-line bg-raised text-text' : 'border-transparent text-faint hover:text-text')}
        >
          {t.label}
        </button>
      ))}
    </div>
    {tabs.map((t, i) => (
      <div id={`panel-${t.id}`} role="tabpanel" aria-labelledby={`tab-${t.id}`} class={i === 0 ? '' : 'hidden'}>
        {t.lines.map((l) => (
          <div class="border-b border-line px-5 py-4 font-mono text-sm last:border-0">
            {l.startsWith('#')
              ? <span class="text-faint">{l}</span>
              : <span><span class="text-accent">&dollar;</span> <span class="text-text">{l.slice(2)}</span></span>}
          </div>
        ))}
      </div>
    ))}
  </div>
  <p class="mt-4 font-mono text-sm text-faint"># then run <span class="text-accent">lazypost</span> in any directory</p>
</section>
<script>
  const tablist = document.getElementById('install-tabs');
  if (tablist) {
    const buttons = [...tablist.querySelectorAll('[role="tab"]')];
    const onCls = ['border-line', 'bg-raised', 'text-text'];
    const offCls = ['border-transparent', 'text-faint'];
    const set = (btn) => {
      buttons.forEach((b) => {
        const active = b === btn;
        b.setAttribute('aria-selected', String(active));
        onCls.forEach((c) => b.classList.toggle(c, active));
        offCls.forEach((c) => b.classList.toggle(c, !active));
        document.getElementById(`panel-${b.dataset.tab}`).classList.toggle('hidden', !active);
      });
    };
    buttons.forEach((btn) => btn.addEventListener('click', () => set(btn)));
  }
</script>
```

Replace `site/src/pages/index.astro`: add `import InstallTabs from '../components/InstallTabs.astro';` to the frontmatter and mount the section in `<main>` after `<Features />` and before the closing `</main>`:

```astro
---
import favicon from '../assets/favicon.svg';
import '../styles/global.css';
import Nav from '../components/Nav.astro';
import Hero from '../components/Hero.astro';
import WhatIs from '../components/WhatIs.astro';
import Features from '../components/Features.astro';
import InstallTabs from '../components/InstallTabs.astro';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
    <Nav />
    <main>
      <Hero />
      <WhatIs />
      <Features />
      <InstallTabs />
    </main>
  </body>
</html>
```

Note: `&dollar;` renders as `$` in the browser; the raw HTML contains `&dollar;`, and no test assertion depends on a literal `$` inside the code panes.

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — all install assertions hold; `aria-selected="true"` present (Astro renders boolean attribute as `aria-selected="true"`/`"false"`).

- [ ] **Step 5: Commit**

```bash
git add site/src/components/InstallTabs.astro site/src/pages/index.astro
git commit -m "feat(site): install tabs"
```

---

### Task 7: Showcase panels and screenshot

**Files:**
- Create: `site/src/components/Showcase.astro`, `site/src/assets/lazypost-screenshot.png` (downloaded)
- Modify: `site/src/pages/index.astro`

**Interfaces:**
- Consumes: Task 2 tokens.
- Produces: `Showcase.astro` — section with `<h2>` `$ lazypost` and hint `a look around`; grid of 5 terminal-styled panels: 1) screenshot `<img>` (imported asset), 2) `vim editing` keymap grid, 3) `lua chaining` code block, 4) `curl import` paste/export hints, 5) `themes` swatches (8 items).

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('showcase section renders panels', () => {
  const h = html();
  assert.match(h, /\$ lazypost/);
  assert.match(h, /a look around/);
  for (const label of ['vim editing', 'lua chaining', 'curl import', 'themes']) {
    assert.match(h, new RegExp(label), `missing showcase panel: ${label}`);
  }
  assert.match(h, /X-Session/);
  assert.match(h, /ctrl\+g/);
  assert.match(h, /store\.set/);
test('showcase screenshot asset emitted', () => {
  const dir = join(DIST, '_astro');
  assert.ok(existsSync(dir) && readdirSync(dir).some((n) => n.endsWith('.png')), 'dist/_astro must contain a hashed png');
});
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — showcase not present yet.

- [ ] **Step 3: Implement**

First download the screenshot (in `site/`):

```bash
curl -fsSL "https://github.com/user-attachments/assets/4a1e695d-ed02-4970-9d5c-162a5f65b6b3" -o src/assets/lazypost-screenshot.png
file src/assets/lazypost-screenshot.png   # must report PNG image data
```

Create `site/src/components/Showcase.astro`:

```astro
---
import screenshot from '../assets/lazypost-screenshot.png';

const vimKeys = [
  ['h j k l', 'motions'],
  ['w b e', 'words'],
  ['0 $ ^', 'line edges'],
  ['g g G', 'jump'],
  ['dd dw d$', 'delete'],
  ['yy yw y$', 'yank'],
  ['v / V', 'visual'],
  ['p / P', 'paste'],
  ['%', 'match'],
];

const themes = [
  ['dracula', '#44475a'],
  ['catppuccin', '#1e1e2e'],
  ['solarized', '#073642'],
  ['gruvbox', '#3c3836'],
  ['nord', '#2e3440'],
  ['tokyonight', '#1a1b26'],
  ['one-dark', '#282c34'],
  ['monokai', '#272822'],
];
---
<section class="mx-auto max-w-6xl px-6 py-20">
  <h2 class="font-mono text-base font-bold text-accent">$ lazypost</h2>
  <p class="mt-3 font-mono text-sm text-faint">a look around</p>
  <div class="mt-8 grid gap-4 md:grid-cols-2">
    <figure class="overflow-hidden rounded-xl border border-line2 bg-panel">
      <figcaption class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">screenshot</figcaption>
      <img src={screenshot} alt="lazypost TUI screenshot" class="w-full rounded-b-xl" />
    </figure>
    <article class="overflow-hidden rounded-xl border border-line2 bg-panel">
      <header class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">vim editing</header>
      <div class="grid grid-cols-3 gap-2 p-4 font-mono text-xs">
        {vimKeys.map(([keys, action]) => (
          <div class="rounded-md border border-line bg-raised px-2 py-1.5 text-center">
            <div class="text-text">{keys}</div>
            <div class="mt-0.5 text-faint">{action}</div>
          </div>
        ))}
      </div>
    </article>
    <article class="overflow-hidden rounded-xl border border-line2 bg-panel">
      <header class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">lua chaining</header>
      <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-faint"><span class="text-accent">-- pre</span>
req.headers[<span class="text-gold">"X-Session"</span>] = store.get(<span class="text-gold">"token"</span>)
req.query[<span class="text-gold">"page"</span>] = <span class="text-gold">"2"</span>
<span class="text-accent">-- post</span>
local id = string.match(response.body, <span class="text-gold">'"id": (%d+)'</span>)
store.set(<span class="text-gold">"last_id"</span>, id)</pre>
    </article>
    <article class="overflow-hidden rounded-xl border border-line2 bg-panel">
      <header class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">curl import</header>
      <div class="p-4 font-mono text-xs leading-6">
        <div><span class="text-accent">$</span> <span class="text-text">paste curl …</span></div>
        <div class="mt-1 text-faint">any pane accepts a pasted curl command</div>
        <div class="mt-4"><span class="text-accent">ctrl+g</span> <span class="text-text">export the current request as curl</span></div>
      </div>
    </article>
    <article class="overflow-hidden rounded-xl border border-line2 bg-panel">
      <header class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">themes</header>
      <ul class="flex flex-wrap gap-2 p-4">
        {themes.map(([name, hex]) => (
          <li class="flex items-center gap-2 rounded-full border border-line bg-raised px-3 py-1.5 font-mono text-xs">
            <span class="h-3 w-3 rounded-full" style={`background:${hex}`}></span>
            <span class="text-muted">{name}</span>
          </li>
        ))}
      </ul>
    </article>
  </div>
</section>
```

Replace `site/src/pages/index.astro`: add `import Showcase from '../components/Showcase.astro';` and render it in `<main>` after `<InstallTabs />`.

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — all showcase assertions hold, plus `dist/_astro/` contains a hashed `.png` (Astro emits the imported asset; the `<img src=...>` in built html references it).

- [ ] **Step 5: Commit**

```bash
git add site/src/components/Showcase.astro site/src/assets/lazypost-screenshot.png site/src/pages/index.astro
git commit -m "feat(site): showcase panels and screenshot"
```

---

### Task 8: Community strip and footer

**Files:**
- Create: `site/src/components/Community.astro`, `site/src/components/Footer.astro`
- Modify: `site/src/pages/index.astro`

**Interfaces:**
- Consumes: Task 3 `site.js` (`github`); Task 2 tokens.
- Produces: `Community.astro` (centered `<p>` with the exact community sentence + `★ star on github` button linking to `github`); `Footer.astro` (`lazypost · MIT License · by Dread-Code`, with `Dread-Code` and a github link, both external with `target="_blank" rel="noopener"`).

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js
test('community strip and footer render', () => {
  const h = html();
  assert.match(h, /Open source, MIT licensed\./);
  assert.match(h, /★ star on github/);
  assert.match(h, /lazypost · MIT License · by /);
  assert.match(h, /target="_blank" rel="noopener"/);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — community/footer absent.

- [ ] **Step 3: Implement**

Create `site/src/components/Community.astro`:

```astro
---
import { github } from '../data/site.js';
---
<section class="mx-auto max-w-6xl px-6 py-24 text-center">
  <p class="mx-auto max-w-xl text-lg leading-relaxed text-muted">
    Open source, MIT licensed. Built by developers, for developers. Contributions, themes and feedback are always welcome.
  </p>
  <a href={github} target="_blank" rel="noopener" class="mt-8 inline-block rounded-lg bg-accent px-6 py-3 font-mono text-sm font-bold text-bg transition-opacity hover:opacity-85">★ star on github</a>
</section>
```

Create `site/src/components/Footer.astro`:

```astro
---
import { github } from '../data/site.js';
---
<footer class="mx-auto max-w-6xl px-6 pb-12 pt-4">
  <div class="border-t border-line pt-6 text-center font-mono text-xs text-faint">
    <span class="font-bold text-muted">lazypost</span> · MIT License · by
    <a href="https://github.com/Dread-Code" target="_blank" rel="noopener" class="text-muted transition-colors hover:text-accent">Dread-Code</a>
    <span class="mx-2">·</span>
    <a href={github} target="_blank" rel="noopener" class="text-muted transition-colors hover:text-accent">github</a>
  </div>
</footer>
```

Replace `site/src/pages/index.astro` with:

```astro
---
import favicon from '../assets/favicon.svg';
import '../styles/global.css';
import Nav from '../components/Nav.astro';
import Hero from '../components/Hero.astro';
import WhatIs from '../components/WhatIs.astro';
import Features from '../components/Features.astro';
import InstallTabs from '../components/InstallTabs.astro';
import Showcase from '../components/Showcase.astro';
import Community from '../components/Community.astro';
import Footer from '../components/Footer.astro';
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="lazypost — a terminal API client. Posting-inspired TUI built with Go and Bubble Tea. Requests are plain YAML files in a directory tree." />
    <title>lazypost</title>
    <link rel="icon" type="image/svg+xml" href={favicon} />
  </head>
  <body>
    <Nav />
    <main>
      <Hero />
      <WhatIs />
      <Features />
      <InstallTabs />
      <Showcase />
      <Community />
    </main>
    <Footer />
  </body>
</html>
```

- [ ] **Step 4: Build and run the tests**

Run: `npm run build && npm test`
Expected: PASS — community copy, star CTA, footer line, and the `target="_blank" rel="noopener"` attribute pattern present.

- [ ] **Step 5: Commit**

```bash
git add site/src/components/Community.astro site/src/components/Footer.astro site/src/pages/index.astro
git commit -m "feat(site): community strip and footer"
```

---

### Task 9: GitHub Pages workflow and end-to-end verification

**Files:**
- Create: `.github/workflows/pages.yml`

**Interfaces:**
- Consumes: everything from Tasks 1–8 (the complete `site/` project).
- Produces: workflow that on push to `main` (paths `site/**` and `.github/workflows/pages.yml`) checks out, installs with npm ci, builds `site/dist`, and deploys via `actions/deploy-pages`. Note: repo Pages settings must be set to Source "GitHub Actions" (user action — document it in the task's final summary).

- [ ] **Step 1: Add the failing test**

Append to `site/tests/smoke.test.mjs`:

```js
test('pages workflow exists and deploys site/dist', () => {
  const wf = read('.github/workflows/pages.yml');
  assert.match(wf, /actions\/deploy-pages@v4/);
  assert.match(wf, /path: site\/dist/);
  assert.match(wf, /npm run build/);
  assert.match(wf, /paths:\s*\[\s*site\/\*\*/s);
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm run build && npm test` (in `site/`)
Expected: FAIL — workflow file absent.

- [ ] **Step 3: Implement**

Create `.github/workflows/pages.yml`:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
    paths: ['site/**', '.github/workflows/pages.yml']

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: site
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: site/package-lock.json
      - run: npm ci
      - run: npm run build
      - uses: actions/configure-pages@v5
      - uses: actions/upload-pages-artifact@v3
        with:
          path: site/dist

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

- [ ] **Step 4: Run the tests**

Run: `npm run build && npm test` (in `site/`)
Expected: PASS — workflow assertions hold.

- [ ] **Step 5: Final end-to-end verification**

Run from repo root:

```bash
cd site && npm run build && npm test
```

then start the production preview:

```bash
cd site && (npm run preview -- --port 4321 >/dev/null 2>&1 &) && sleep 3 && curl -s http://localhost:4321/lazypost/ | grep -c "An API client that lives in your terminal" && pkill -f "astro preview" || true
```

Expected: `1` printed (the tagline served at the `/lazypost/` base path).

Manual link audit (browser or curl): `https://github.com/Dread-Code/lazypost` (nav github, hero view on github, CTAs, footer), `https://github.com/Dread-Code` (footer), `https://raw.githubusercontent.com/Dread-Code/lazypost/main/install.sh` (installs), shields.io badge URL (renders a badge image), and `#install` (smooth-scrolls; `curl http://localhost:4321/lazypost/` should contain `id="install"`).

Missing: only the deployment step is out of your hands — after push to `main`, the workflow must show green, and `https://dread-code.github.io/lazypost/` must render the page. Repo settings: Settings → Pages → Build and deployment → Source: "GitHub Actions".

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/pages.yml site/tests/smoke.test.mjs
git commit -m "ci: add github pages deploy workflow"
```
