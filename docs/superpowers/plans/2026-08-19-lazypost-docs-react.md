# lazypost Docs — React-Markdown Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 10 hand-written `.astro` doc pages with markdown-authored guides rendered by `react-markdown`, served at real per-doc URLs via an Astro dynamic route + client-side `history.pushState` navigation.

**Architecture:** Markdown sources in `site/src/docs/*.md`; a React island `DocsApp.jsx` (sidebar + renderer + pushState router) mounted by `pages/docs/index.astro` and `pages/docs/[slug].astro` (server-rendered at build → content always in the static HTML); prose styling via `@tailwindcss/typography` tuned to the site tokens. Old pages, sidebar-as-layout, and `.docs-table` CSS are deleted.

**Tech Stack:** Adds `@astrojs/react`, `react`, `react-dom`, `react-markdown`, `remark-gfm`, `@tailwindcss/typography`. Existing: Astro 5, Tailwind v4, node:test smoke suite.

**Spec:** `docs/superpowers/specs/2026-08-19-lazypost-docs-react-design.md` (approved). Where plan and spec disagree the plan is wrong.

## Global Constraints

- npm commands inside `site/`; test command `npm test`.
- Copy is frozen: markdown ports the VISIBLE content of the existing `site/src/pages/docs/*.astro` pages verbatim (those pages are the source of truth; the README is the underlying authority). Signatures per file are enforced by tests.
- No hash routing anywhere. Sidebar navigation uses `<a href>` + `history.pushState`; `popstate` handles back/forward.
- Internal markdown links are absolute site paths (`/lazypost/docs/<slug>/`).
- No new components/pages beyond those listed. Landing page is untouched.
- Astro config gains the React integration; `global.css` gains `@plugin "@tailwindcss/typography";` and the prose overrides; the `.docs-table` block is removed.
- Branch: `docs-section`. Commit after every task.

## File Structure

```
site/src/
  docs/*.md                       T2  (index, quickstart, installation, keybindings,
                                     collections, environments, scripting, importers,
                                     themes, faq)
  data/docs.js                    T2  [{slug, title}]
  components/DocsApp.jsx          T3  React island (sidebar + react-markdown + router)
  components/DocsLayout.astro     T4  chrome-only shell (Nav/Footer/css/ThemeScript)
  pages/docs/index.astro          T4
  pages/docs/[slug].astro         T4  getStaticPaths from docs.js
  pages/docs/*.astro OLD 10       T4  DELETED
  styles/global.css               T1+T4  typography plugin; prose overrides; drop docs-table
tests/smoke.test.mjs              T1..T4  reworked
```

---

### Task 1: Dependencies, React integration, typography plugin

**Files:** Modify `site/package.json` (via npm i), `site/astro.config.mjs`, `site/src/styles/global.css`, `site/tests/smoke.test.mjs`

- [ ] **Step 1: Add the failing test** — append to `site/tests/smoke.test.mjs`:

```js

test('css ships typography prose styles', () => {
  assert.match(css(), /\.prose/);
});
```

- [ ] **Step 2: Verify failure** — `npm run build && npm test` (site/) — the new test FAILS (`.prose` absent from css()).

- [ ] **Step 3: Implement**

1. Install (inside `site/`): `npm install @astrojs/react react react-dom react-markdown remark-gfm @tailwindcss/typography`
2. `site/astro.config.mjs`: add `import react from '@astrojs/react';` and put `react()` in the `integrations` array (output stays static; vite tailwindcss plugin untouched).
3. `site/src/styles/global.css`: add the line `@plugin "@tailwindcss/typography";` directly under `@import "tailwindcss";`.

- [ ] **Step 4: Verify** — `npm run build && npm test` — all green.
- [ ] **Step 5: Commit** — `git add site/package.json site/package-lock.json site/astro.config.mjs site/src/styles/global.css site/tests/smoke.test.mjs && git commit -m "feat(site): react + react-markdown + typography deps"`

---

### Task 2: Markdown sources and sidebar data

**Files:** Create `site/src/docs/*.md` (10), modify `site/src/data/docs.js`, modify `site/tests/smoke.test.mjs`

**Interfaces:** Produces the markdown strings `DocsApp.jsx` will glob (T3) and the `[{slug, title}]` list `[slug].astro` will enumerate (T4).

- [ ] **Step 1: Add the failing tests** — append:

```js

const DOC_SLUGS = ['index', 'quickstart', 'installation', 'keybindings', 'collections', 'environments', 'scripting', 'importers', 'themes', 'faq'];

const DOC_SIGNATURES = {
  index: 'Quick start',
  quickstart: 'ctrl+l edits the URL',
  installation: 'sh -s -- v0.4.0',
  keybindings: 'cycle auth type',
  collections: 'version: 1',
  environments: 'api.dev.example.com',
  scripting: 'expected 200',
  importers: 'workspace--environment',
  themes: 'XDG_CONFIG_HOME',
  faq: 'unsupported protocol scheme',
};

test('markdown sources exist with signatures', () => {
  for (const slug of DOC_SLUGS) {
    assert.match(read(`site/src/docs/${slug}.md`), new RegExp(DOC_SIGNATURES[slug]), `missing signature in docs/${slug}.md`);
  }
});
```

- [ ] **Step 2: Verify failure** — `npm run build && npm test` — the new test FAILS (files absent).

- [ ] **Step 3: Implement**

1. Create `site/src/data/docs.js` with EXACTLY:

```js
export const docs = [
  { slug: 'quickstart', title: 'Quick start' },
  { slug: 'installation', title: 'Installation' },
  { slug: 'keybindings', title: 'Keybindings' },
  { slug: 'collections', title: 'Collections' },
  { slug: 'environments', title: 'Environments' },
  { slug: 'scripting', title: 'Scripting' },
  { slug: 'importers', title: 'Importers' },
  { slug: 'themes', title: 'Themes' },
  { slug: 'faq', title: 'FAQ' },
];
```

2. Create the 10 markdown files. CONVERSION RULES (apply to each source page; never invent copy):
   - Source = the existing page `site/src/pages/docs/<slug-or-map>.astro`, whose `h1`-content becomes the file's `#` heading and whose copy/text/tables/code transfer 1:1. Mapping: index→index.md (intro + link list card content, as a markdown link list to `/lazypost/docs/<slug>/`), quickstart→quickstart.md, installation→installation.md, keybindings→keybindings.md, collections→collections.md, environments→environments.md, scripting→scripting.md, importers→importers.md, themes→themes.md, faq→faq.md.
   - Headings: `##` for section headings. Code blocks: fenced with the right info string — bash for `$` prompts, yaml for config/request/env/theme files, lua for scripts, text for keymap grids.
   - The keybindings tables → GFM tables `| Key | Action |` with the exact rows from the page; importers flags table → `| Flag | Meaning |`; scripting globals → `| Global | Meaning |`.
   - FAQ: ten `##` question headings, each followed by the answer paragraph.
   - `index.md`: `# // docs` heading is NOT allowed — use `# docs`; then the intro paragraph verbatim, then:

     ```markdown
     ## Guides
     - [Quick start](/lazypost/docs/quickstart/)
     - [Installation](/lazypost/docs/installation/)
     - [Keybindings](/lazypost/docs/keybindings/)
     - [Collections](/lazypost/docs/collections/)
     - [Environments](/lazypost/docs/environments/)
     - [Scripting](/lazypost/docs/scripting/)
     - [Importers](/lazypost/docs/importers/)
     - [Themes](/lazypost/docs/themes/)
     - [FAQ](/lazypost/docs/faq/)
     ```
   - `themes.md` ends with a `## example.yaml` section containing the full `docs/themes/example.yaml` file in a yaml fence.
   - `{{variable}}`, `{{host}}` etc. are written plainly (no escaping in markdown).

- [ ] **Step 4: Verify** — `npm run build && npm test` — all green.
- [ ] **Step 5: Commit** — `git add site/src/docs site/src/data/docs.js site/tests/smoke.test.mjs && git commit -m "feat(site): markdown docs sources"`

---

### Task 3: DocsApp React island

**Files:** Create `site/src/components/DocsApp.jsx`

**Interfaces:** Consumes `docs.js` and the globbed markdown; `client:load`-mountable with a `slug` prop; server-renderable (react-markdown stringifies at build). Produces: sidebar `<a>` list + mobile `<select>`, article with `.prose` classes and the `data-docs-content` marker; pushState navigation; `popstate` listener.

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs store anchors to real urls', () => {
  const src = read('site/src/components/DocsApp.jsx');
  assert.match(src, /pushState/);
  assert.match(src, /react-markdown/);
  assert.match(src, /popstate/);
});
```

- [ ] **Step 2: Verify failure** — `npm run build && npm test` — FAILS (file absent).

- [ ] **Step 3: Implement** — create with EXACTLY (NOTE: React requires `className`, not `class`):

```jsx
import { useCallback, useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { docs } from '../data/docs.js';

const raw = import.meta.glob('../docs/*.md', { query: '?raw', import: 'default' });

export default function DocsApp({ slug }) {
  const [current, setCurrent] = useState(slug);
  const hrefFor = (s) => `${import.meta.env.BASE_URL}docs/${s}/`;

  useEffect(() => {
    const onPop = () => {
      const m = window.location.pathname.match(/\/docs\/([^/]+)\//);
      setCurrent(m ? m[1] : 'index');
    };
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  const navigate = useCallback((e, s) => {
    e.preventDefault();
    window.history.pushState({}, '', hrefFor(s));
    setCurrent(s);
    window.scrollTo({ top: 0 });
  }, []);

  const loadRaw = raw[`../docs/${current}.md`];
  const markdown = typeof loadRaw === 'function' ? loadRaw() : '';

  return (
    <div className="flex flex-col gap-8 lg:flex-row lg:items-start">
      <select
        className="w-full rounded-lg border border-line bg-raised p-2 font-mono text-xs text-text lg:hidden"
        value={current}
        onChange={(e) => {
          const s = e.target.value;
          window.history.pushState({}, '', hrefFor(s));
          setCurrent(s);
        }}
      >
        {docs.map((d) => (
          <option key={d.slug} value={d.slug}>
            {d.title}
          </option>
        ))}
      </select>
      <aside className="hidden w-52 shrink-0 lg:block">
        <nav className="sticky top-6 space-y-1">
          <p className="mb-2 font-mono text-xs font-bold text-faint">// docs</p>
          {docs.map((d) => (
            <a
              key={d.slug}
              href={hrefFor(d.slug)}
              onClick={(e) => navigate(e, d.slug)}
              aria-current={d.slug === current ? 'page' : undefined}
              className={
                d.slug === current
                  ? 'block rounded-md bg-raised px-3 py-1.5 font-mono text-xs text-accent transition-colors'
                  : 'block rounded-md px-3 py-1.5 font-mono text-xs text-muted transition-colors hover:bg-raised hover:text-text'
              }
            >
              {d.title}
            </a>
          ))}
        </nav>
      </aside>
      <article className="prose prose-invert min-w-0 max-w-none flex-1 pb-16" data-docs-content>
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{markdown}</ReactMarkdown>
      </article>
    </div>
  );
}
```

(NOTE: if Astro processes the file with an .astro-style frontmatter accidentally, keep it a plain .jsx with no `---` block.)

- [ ] **Step 4: Verify** — `npm run build && npm test` — all green.
- [ ] **Step 5: Commit** — `git add site/src/components/DocsApp.jsx && git commit -m "feat(site): DocsApp react island"`

---

### Task 4: Routes, chrome layout, deletions, prose overrides, test rework

**Files:** Create `site/src/pages/docs/index.astro`, `site/src/pages/docs/[slug].astro`; rewrite `site/src/components/DocsLayout.astro`; delete `site/src/pages/docs/*.astro` old pages (all files directly in that dir except the two new ones) and `site/src/pages/docs/themes-example.astro` if it exists; modify `site/src/styles/global.css` (prose overrides, remove `.docs-table` block); rework `site/tests/smoke.test.mjs`.

- [ ] **Step 1: Rework the tests**

DELETE these test blocks entirely: `docs index content`, `docs quickstart page`, `docs installation page`, `docs keybindings page`, `docs collections page`, `docs environments page`, `docs scripting page`, `docs importers page`, `docs themes page`, `docs themes example page`, `docs faq page`, `docs pages carry the docs nav link`.

APPEND exactly:

```js

test('docs routes pre-render with content', () => {
  for (const slug of DOC_SLUGS) {
    const p = slug === 'index' ? 'docs/index.html' : `docs/${slug}/index.html`;
    assert.match(page(p), new RegExp(DOC_SIGNATURES[slug]), `missing content on ${p}`);
  }
});

test('docs use real urls, no hash routing', () => {
  const d = page('docs/keybindings/index.html');
  assert.match(d, /\/lazypost\/docs\/faq\//);
  assert.doesNotMatch(html(), /#\/|href="#/);
});

test('old docs routes are gone', () => {
  const dir = join(DIST, 'docs');
  assert.ok(existsSync(dir), 'docs dir missing');
  const names = readdirSync(dir);
  for (const n of names) {
    assert.ok(DOC_SLUGS.includes(n), `unexpected docs entry: ${n}`);
  }
});

test('internal docs links resolve', () => {
  const files = ['index.html', ...DOC_SLUGS.map((s) => (s === 'index' ? 'docs/index.html' : `docs/${s}/index.html`))];
  const needs = new Set();
  for (const f of files) {
    const h = f === 'index.html' ? html() : page(f);
    for (const m of h.matchAll(/href="(\/lazypost\/docs\/[^"]+)"/g)) {
      needs.add(m[1]);
    }
  }
  for (const n of needs) {
    const rel = n.replace('/lazypost/', '');
    const ok = existsSync(join(DIST, rel)) || existsSync(join(DIST, rel, 'index.html'));
    assert.ok(ok, `unresolved docs link: ${n}`);
  }
});
```

- [ ] **Step 2: Verify failure** — `npm run build && npm test` — expect multiple failures (old pages still 200-happy but the reworked assertions hit stale content; e.g. keybindings page still static with no sidebar links, `docs use real urls` fails). Capture 2 representative failing lines.

- [ ] **Step 3: Implement**

1. Rewrite `site/src/components/DocsLayout.astro` with EXACTLY:

```astro
---
import Nav from '../components/Nav.astro';
import Footer from '../components/Footer.astro';
import ThemeScript from '../components/ThemeScript.astro';
import '../styles/global.css';

const { title, slug } = Astro.props;
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content={`lazypost docs — ${title}`} />
    <title>{`lazypost — ${title}`}</title>
    <link rel="icon" type="image/svg+xml" href={`${import.meta.env.BASE_URL}favicon.svg`} />
  </head>
  <body>
    <Nav />
    <main class="mx-auto max-w-6xl px-6 py-10">
      <slot slug={slug} />
    </main>
    <Footer />
    <ThemeScript />
  </body>
</html>
```

2. Create `site/src/pages/docs/index.astro` with EXACTLY:

```astro
---
import DocsLayout from '../../components/DocsLayout.astro';
import DocsApp from '../../components/DocsApp.jsx';
---
<DocsLayout title="Docs" slug="index">
  <DocsApp slug="index" client:load />
</DocsLayout>
```

3. Create `site/src/pages/docs/[slug].astro` with EXACTLY:

```astro
---
import DocsLayout from '../../components/DocsLayout.astro';
import DocsApp from '../../components/DocsApp.jsx';
import { docs } from '../../data/docs.js';

export function getStaticPaths() {
  return docs.map((d) => ({ params: { slug: d.slug } }));
}

const { slug } = Astro.params;
const meta = docs.find((d) => d.slug === slug);
---
<DocsLayout title={meta?.title ?? 'Docs'} slug={slug}>
  <DocsApp slug={slug} client:load />
</DocsLayout>
```

4. Delete the old pages: `rm site/src/pages/docs/{index,quickstart,installation,keybindings,collections,environments,scripting,importers,themes,themes-example,faq}.astro` (keep only the two new files).

5. `site/src/styles/global.css`:
   a. REMOVE the whole `@layer components { .docs-table ... }` block.
   b. APPEND exactly:

```css
.prose {
  --tw-prose-body: var(--color-text);
  --tw-prose-headings: var(--color-text);
  --tw-prose-links: var(--color-accent);
  --tw-prose-code: var(--color-gold);
  --tw-prose-pre-bg: var(--color-panel);
  --tw-prose-pre-code: var(--color-text);
  --tw-prose-quotes: var(--color-text);
  --tw-prose-th-borders: var(--color-line2);
  --tw-prose-td-borders: var(--color-line);
  --tw-prose-counters: var(--color-muted);
  --tw-prose-bullets: var(--color-accent);
  --tw-prose-hr: var(--color-line);
}
.prose code {
  background: var(--color-raised);
  border: 1px solid var(--color-line);
  border-radius: 0.25rem;
  padding: 0.125rem 0.3rem;
  font-weight: 400;
}
.prose pre {
  border: 1px solid var(--color-line);
  border-radius: 0.5rem;
}
.prose pre code {
  background: transparent;
  border: 0;
  padding: 0;
}
.prose thead th {
  background: var(--color-navbar);
  color: var(--color-muted);
}
.prose blockquote {
  border-left-color: var(--color-accent);
}
```

- [ ] **Step 4: Verify** — `npm run build && npm test` — all green. Confirm signatures landed in SSR HTML on a sample: `grep -o 'unsupported protocol scheme' dist/docs/faq/index.html | wc -l` → 1 and `grep -o 'cycle auth type' dist/docs/keybindings/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add -A site && git commit -m "feat(site): dynamic react-markdown docs routes"`

---

### Task 5: End-to-end verification and final commit

- [ ] **Step 1: Full suite** — `cd site && npm run build && npm test` — all green.
- [ ] **Step 2: Preview audit** — start preview on 4321 and curl every route:

```bash
(cd site && (npm run preview -- --port 4321 >/dev/null 2>&1 &)) && sleep 4
for p in docs docs/quickstart docs/installation docs/keybindings docs/collections docs/environments docs/scripting docs/importers docs/themes docs/faq docs/nonexistent; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:4321/lazypost/$p/")
  echo "$p -> $code"
done
pkill -f "astro preview" || true
```

`docs/nonexistent` may return 200 via the SPA-fallback or 404 — report which; the 9 real routes + `docs` MUST be 200.

- [ ] **Step 3: Manual browser check (report what you can)** — if a headless browser is available (playwright/puppeteer installed) load `/lazypost/docs/` and click a sidebar link, confirm the URL changes without reload; otherwise note "manual browser check pending".
- [ ] **Step 4: Final commit** — if anything is uncommitted: `git add -A site && git commit -m "chore(site): react docs final polish"` (skip if clean).
