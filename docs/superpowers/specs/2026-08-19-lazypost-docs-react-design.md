# lazypost Docs — React-Markdown Architecture Design Spec

- **Date:** 2026-08-19
- **Status:** Approved (supersedes 2026-08-19-lazypost-docs-design.md for the docs section)
- **Author:** Dread-Code

## Goal

Rebuild the lazypost docs section: author every guide as a plain Markdown file, render it with React + `react-markdown`, and serve real per-doc URLs (`/docs/quickstart/`, `/docs/faq/`, …) via an Astro dynamic route + client-side `history.pushState` navigation. This replaces the 10 hand-written `.astro` doc pages from the previous build.

## Context

- Current state (branch `docs-section`): 10 `.astro` pages under `site/src/pages/docs/`, `DocsLayout.astro` shell with sidebar + theme picker, `docs.js` sidebar data, bundled `ThemeScript.astro` (localStorage `lazypost-theme`, applies 12 CSS tokens → works page-to-page), `.docs-table` styles in `global.css`, 39-test smoke suite.
- Content of all 10 pages is final-approved (README-verbatim + vault facts) — the rework changes the RENDERING MECHANISM, not the copy.
- Dependencies to add: `@astrojs/react`, `react`, `react-dom`, `react-markdown`, `remark-gfm`, `@tailwindcss/typography` (Tailwind v4 `@plugin`).

## Architecture

```
site/src/
  docs/*.md                          authoring (10 files: index, quickstart, installation,
                                     keybindings, collections, environments, scripting,
                                     importers, themes, faq)
  data/docs.js                       [{slug, title}] in sidebar order
  components/DocsApp.jsx             React island: sidebar + react-markdown renderer +
                                     pushState router
  components/DocsLayout.astro        chrome shell: head/meta, Nav, Footer, stylesheet +
                                     prose css, ThemeScript (no sidebar, no picker)
  pages/docs/index.astro             /docs/  → DocsApp slug="index"
  pages/docs/[slug].astro            /docs/{slug}/ → DocsApp slug={slug}; getStaticPaths
                                     from docs.js; unknown slug → Astro 404
  styles/global.css                  + typography plugin + prose token overrides
tests/smoke.test.mjs                 reworked docs assertions
```

- **Markdown bundling:** `DocsApp.jsx` uses `import.meta.glob('../docs/*.md', { query: '?raw', import: 'default' })` — all markdown ships compiled into the JS bundle at build time; zero runtime fetching.
- **SSR first:** `[slug].astro` passes the raw markdown + slug to `<DocsApp client:load>`; react-markdown is rendered to HTML at build time, so every `/docs/{slug}/` page's content is in its static HTML (works without JS; SEO-friendly).
- **Client routing:** sidebar buttons call `history.pushState` with the target path (`/docs/{slug}/`, relative to `import.meta.env.BASE_URL`); a `popstate` listener re-syncs state; no page reloads, no hashes.
- **Styling:** `@plugin "@tailwindcss/typography"` in `global.css`; a `prose prose-invert` wrapper tuned to tokens: `--tw-prose-*` overrides using the site palette (headings `--color-text`, links `--color-accent`, code `bg-raised`/`text-gold`, tables: borders `--color-line` incl. thead `bg-navbar`, blockquote accent left border). `.docs-table` rules in global.css are removed (typography owns tables now).
- **Internal links inside markdown:** absolute site paths `(/lazypost/docs/…/)` (base is fixed for this deployment; the E2E link audit enforces they resolve).
- **Sidebar data:** `docs.js` becomes `[{ slug, title }]` (order: quickstart, installation, keybindings, collections, environments, scripting, importers, themes, faq; `index` is the landing of /docs/ and not listed in the sidebar — the logo/docs link covers navigation).
- **Deleted:** `site/src/pages/docs/*.astro` (all 10), old sidebar/picker markup in any layout, `.docs-table` CSS, and the `themes-example` route (its content becomes a fenced code block inside `themes.md`).

## Content mapping (copy is frozen from the existing pages)

Each `.md` file ports the visible content of the corresponding `site/src/pages/docs/*.astro`:
- headings as `#`/`##`, GFM tables for keybindings (2-col `Key | Action`), flags (2-col), globals (2-col); code fences ```yaml / ```bash / ```lua / ```text for the terminal blocks (colors are lost in markdown — plain code blocks are fine); FAQ as `##` Q headings + answer paragraphs; index.md as a `#` intro + link list to the other guides; themes.md ends with the `docs/themes/example.yaml` content in a ```yaml fence.
- `{{variable}}`-style text is plain in markdown (no escaping needed).
- `$` prompts stay in code blocks.
- Internal cross-links use absolute `/lazypost/docs/<slug>/` URLs.

## Behavior & Edge Cases

- JS off: markdown still renders (SSR HTML); sidebar is unusable (links render as buttons) — acceptable; provide the sidebar buttons with real `<a href>` fallbacks (clicking them = full page load to the target doc) — sidebar items are anchors, not buttons, so no-JS navigation still works; DocsApp intercepts clicks and pushStates.
- Unknown slug: Astro's static 404.
- Theme persistence unchanged (ThemeScript stays on all docs pages; docs render in the saved theme).
- Active sidebar item derived from current slug.

## Out of Scope

- Search, i18n, MDX plugins beyond remark-gfm, syntax highlighting libraries (code blocks stay uncolored in docs), migration of the landing page to React, changing the deploy workflow.

## Testing

- Smoke suite rework: `page()` helper unchanged.
- Assertions: each doc route exists in `dist/docs/{slug}/index.html`; each contains its signature string (now SSR'd into HTML); the old per-page routes are gone; landing `index.html` nav unchanged; the compiled CSS contains prose classes (`--tw-prose` or `.prose-invert`); source `site/src/docs/*.md` files each exist with a signature string; internal link audit (every `/lazypost/docs/…` href in the built docs HTML resolves in `dist`).
- E2E: full build + `npm run preview` → `/lazypost/docs/`, `/lazypost/docs/quickstart/`, `/lazypost/docs/faq/` all 200.

## Verification

1. `cd site && npm run build && npm test` all green.
2. `npm run preview`: `/docs/` loads index; clicking sidebar items changes the URL to real paths with no reload; back/forward work; direct `/docs/keybindings/` loads its content; theme picked on the landing persists on docs.
3. Push → Pages workflow green → `dread-code.github.io/lazypost/docs/…` live.
