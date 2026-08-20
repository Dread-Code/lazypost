# lazypost Documentation Section — Design Spec

- **Date:** 2026-08-19
- **Status:** Approved (multi-page `/docs/` in the existing Astro site; fresh branch implementation)
- **Author:** Dread-Code

## Goal

Add a documentation section to the lazypost landing site (`site/`), served at `https://dread-code.github.io/lazypost/docs/…`, covering everything a user needs beyond the landing page: installation, quickstart, keybindings, collections, environments, scripting, importers, themes, and a FAQ. Content sourced from the repo README (authoritative) plus the project vault (`/Users/dread-code/Documents/Vault/lazy-post` — ADRs, gotchas, postmortems) for fact-level details and troubleshooting.

## Context

- Landing site is Astro 5 + Tailwind v4 in `site/`, deployed by `.github/workflows/pages.yml` (push to main, paths `site/**`) to `dread-code.github.io/lazypost/` (`base: '/lazypost/'`).
- The site's theme switcher (localStorage `lazypost-theme`, 11 CSS-variable tokens per theme) currently exists only as an inline script inside `Showcase.astro` (landing page) — docs pages need the same switcher, so the script is extracted into a shared component.
- README (`README.md`:277 lines) is the single source of user docs today; vault adds: body JSON auto-format (ADR-0016/17), executed URL shown in response Headers tab (ADR-0014), unresolved-placeholder loud failure (ADR-0008), vim yank/paste asymmetry, and troubleshooting material (08 Gotchas/, 12 Postmortems/).

## Architecture

New multi-page ASTRO section, plain `.astro` pages (no MDX, no content collections — YAGNI):

```
site/src/
  layouts/DocsLayout.astro          shared shell (nav + sidebar + footer + ThemeScript)
  components/ThemeScript.astro      extracted theme-switcher inline script (landing + docs)
  data/docs.js                      sidebar entries [{href, title}] — single source
  pages/docs/index.astro            overview + page map
  pages/docs/installation.astro
  pages/docs/quickstart.astro
  pages/docs/keybindings.astro
  pages/docs/collections.astro
  pages/docs/environments.astro
  pages/docs/scripting.astro
  pages/docs/importers.astro
  pages/docs/themes.astro
  pages/docs/faq.astro
```

- `DocsLayout.astro`: full-page shell reusing landing tokens/utilities; top bar = existing `Nav.astro` plus a `docs` link pointing to `import.meta.env.BASE_URL + 'docs/'`; left sidebar (collapsible on small screens via `<details>`) listing `data/docs.js` entries with the current page highlighted; main content column with a sticky "on this page" TOC built from the page's `##` headings (custom `Code`-block-free component reading `Astro.props` headings passed from each page frontmatter); footer identical to landing `Footer.astro`.
- `Nav.astro` gains the `docs` link site-wide (landing too).
- `ThemeScript.astro`: move the inline script currently in `Showcase.astro` verbatim, wrapped in an IIFE. Both `showcase` themes-panel buttons and a compact swatch row in the DocsLayout sidebar footer drive it; `localStorage` key unchanged (`lazypost-theme`), so selection follows the visitor across landing and docs.
- Table styling (keybindings/importers flags): add `.docs-table` rules to `src/styles/global.css` (bordered rows using `--color-line`, header row `bg-navbar`, mono font, `overflow-x-auto` wrapper) — no new dependencies.

## Pages & Content (user-facing; internals only where observable)

1. **docs/ (index)** — `// docs` header; short intro; "why terminal" vision line from vault 02 Vision/Project Vision.md; card grid linking the 9 guides (style mirrors Features cards).
2. **installation** — three tab blocks (reuse InstallTabs markup pattern, no JS needed: static `<details>` or keep tabs via the existing pattern) for `curl | sh`, pin a version (`-s -- v0.4.0` from `site.js` `version`), `go build`; binaries note (macOS/linux arm64+amd64, releases); `lazypost -version`.
3. **quickstart** — the four numbered steps (lazypost in any dir; `n`, `ctrl+l`, `ctrl+t`; `ctrl+s`, `ctrl+r`; environments with `ctrl+e`), run examples with `-dir`, implicit collection folders.
4. **keybindings** — five tables verbatim from README (Global, Collection sidebar, URL bar, Editor, Response) in `.docs-table` style.
5. **collections** — directory layout example, `config/config.yaml`, request YAML format + 3 auth variants (verbatim README), ".lazypost legacy migration" one-liner (ADR-0019), `-dir`/current-dir behavior.
6. **environments** — YAML file shape, `{{var}}` substitution semantics (unknown → left as-is; unresolved URL placeholder → loud error pointing at `ctrl+e` per ADR-0008), env manager (`ctrl+/` → Environments, `ctrl+e` cycle, `a`/`r`/`d`, leading `/` creates env).
7. **scripting** — sandbox note (no fs/network, `os.time` only), pre/post semantics, `req`/`env`/`store`/`response` globals table, two examples from README verbatim.
8. **importers** — usage lines + flags table + behavior bullets (verbatim README Import section incl. workspaces → folders, env namespacing `<workspace>--<environment>`, collision suffixes, `--strict`, `--dry-run`).
9. **themes** — presets list, switch via palette; custom YAML theme file shape (`light`/`dark` hex pairs, all optional, `~/.config/lazypost/themes/`), link to `docs/themes/example.yaml`.
10. **faq** — troubleshooting Q&A: unresolved placeholder error; first send "unsupported protocol scheme" (Postmortem - first send fails); vim selection reverse-video gotcha (fix: behaves correctly in app); yank → system clipboard, paste internal register; body JSON auto-format on blur/save incl. placeholders (ADR-0016/17); response Headers tab shows executed URL; `--dry-run` never writes; `ctrl+arrows` macOS collision → `alt+←/→`; any other gotcha phrased user-facing. Each answer 1-3 lines, honest "by design" where applicable.

## Behavior & Edge Cases

- All internal links absolute under `import.meta.env.BASE_URL` (e.g. `/lazypost/docs/`), matching existing base handling.
- Theme switcher degraded without JS: default dracula (same as today); swatches render but do nothing.
- Small screens: sidebar collapses (`<details>`); tables overflow-x scroll (no content loss).
- `docs` link in Nav.astro present on landing page too; no active-state logic beyond docs pages themselves.
- Navigation is fully static; no search on docs pages (out of scope).

## Out of Scope

- MDX/content-collections/Starlight or similar docs frameworks; i18n; search; blog; API reference auto-generation; contribution/development pages (README-linked); changes to Go codebase or the deploy workflow.

## Testing

- smoke harness (`site/tests/smoke.test.mjs`) gains `page(path)` helper reading `dist/<path>`.
- Tests: every docs route exists in `dist/docs/*.html`; each page contains one signature string (e.g. keybindings page includes `ctrl+h` row text, scripting page includes `store.set`, faq page includes `unsupported protocol`); landing `index.html` contains a link to the docs section; ThemeScript extraction keeps `lazypost-theme` in the landing page (existing assertion) and adds it to docs pages.

## Verification

1. `cd site && npm run build && npm test` — all green.
2. `npm run preview` — walk landing → docs → each page; sidebar highlights current; theme picked on one page persists on the next; tables scroll on mobile width.
3. Push to main → Pages workflow green → `dread-code.github.io/lazypost/docs/` serves each page.
