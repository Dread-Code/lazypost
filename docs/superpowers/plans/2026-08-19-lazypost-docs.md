# lazypost Documentation Section Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a 10-page `/docs/` section to the lazypost landing site (`site/`), styled identically to the landing page, with a shared docs layout, theme switcher on docs pages, and smoke-tested content.

**Architecture:** Plain `.astro` routes under `site/src/pages/docs/`, wrapped in a `DocsLayout` shell (site nav + sidebar from a `data/docs.js` array + footer + `ThemeScript`). The theme-switch inline script is extracted from `Showcase.astro` into `ThemeScript.astro` (used by landing + docs); swatch buttons are factored into `Swatches.astro` (used by the Showcase themes panel + a compact picker in the docs sidebar footer). Content is ported verbatim from `README.md` plus user-facing facts from the vault (`08 Gotchas/`, `12 Postmortems/`, ADRs).

**Tech Stack:** Astro 5 (static output), Tailwind v4 tokens as already used, Node 22 built-in test runner.

**Spec:** `docs/superpowers/specs/2026-08-19-lazypost-docs-design.md` (approved). Where the plan and spec disagree the plan is wrong.

## Global Constraints

- Working dir for npm commands is `site/`; test command `npm test` (= `node --test "tests/*.test.mjs"`).
- No new dependencies. No MDX, no content collections.
- All internal links are `{import.meta.env.BASE_URL}docs/...`-style (BASE_URL is `/lazypost/`). Never hardcode `/lazypost/` in source.
- Copy marked "verbatim from README" must match `README.md` character-for-character (it is the source of truth); vault facts are phrased user-facing, 1-3 sentences.
- Tests append to `site/tests/smoke.test.mjs`. Shared helpers gain `page(p)` reading `dist/<p>` (see Task 0). New-per-page signature assertions only (exist + one content string).
- Astro emits `src/pages/docs/x.astro` as `dist/docs/x/index.html` (directory format default).
- Design tokens/utilities from `global.css` are available (`text-accent`, `bg-panel`, `border-line`, `font-mono`, etc.).
- Commit after every task with the listed message.

## File Structure

```
site/src/
  components/
    Nav.astro                  T1  + docs link
    ThemeScript.astro          T1  extracted inline script (IIFE)
    Swatches.astro             T2  theme swatch buttons (landing + docs picker)
    Showcase.astro             T1/T2  drop inline script; embed <Swatches />
  layouts/
    DocsLayout.astro           T2  shell: Nav + sidebar + picker + Footer + ThemeScript
  data/
    docs.js                    T2  [{href, title}]
  pages/
    index.astro                T1  mount <ThemeScript />
    docs/{index,installation,quickstart,keybindings,collections,environments,scripting,importers,themes,faq}.astro   T3..T12
  styles/global.css            T2  .docs-table rules
tests/smoke.test.mjs           T0  + page() helper; grows per task
```

---

### Task 0: Test helper `page()`

**Files:** Modify `site/tests/smoke.test.mjs`

- [ ] **Step 1: Add the failing test**

Append to `site/tests/smoke.test.mjs`:

```js

test('page() helper reads built docs files', () => {
  assert.equal(page('index.html'), html());
  assert.equal(page('nope/index.html'), '');
});
```

And add the helper next to the existing `html` helper (after the `html` definition):

```js
export const page = (p) => {
  const full = join(DIST, p);
  return existsSync(full) ? readFileSync(full, 'utf8') : '';
};
```

- [ ] **Step 2: Run to verify failure**

Run: `npm test` (in `site/`)
Expected: FAIL — `page is not defined`.

- [ ] **Step 3: Make it pass by adding the helper**

(The helper is written in Step 1; re-run the test.)

- [ ] **Step 4: Verify**

Run: `npm run build && npm test`
Expected: PASS — all existing tests + new one green.

- [ ] **Step 5: Commit**

```bash
git add site/tests/smoke.test.mjs && git commit -m "test(site): page() helper for docs files"
```

---

### Task 1: Extract ThemeScript, add nav docs link

**Files:**
- Create: `site/src/components/ThemeScript.astro`
- Modify: `site/src/components/Showcase.astro` (delete inline script), `site/src/components/Nav.astro` (docs link), `site/src/pages/index.astro` (mount `<ThemeScript />`)

**Interfaces:**
- Consumes: nothing new; the inline script currently inside `Showcase.astro` (its whole `<script is:inline>...</script>` block).
- Produces: `ThemeScript.astro` — the same IIFE script driving `[data-theme]` buttons + localStorage key `lazypost-theme`; `Nav.astro` gains a `docs` link `{base}docs/` where `const base = import.meta.env.BASE_URL`.

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js

test('theme script component parses', () => {
  const src = read('site/src/components/ThemeScript.astro');
  const m = src.match(/<script is:inline>([\s\S]*?)<\/script>/);
  assert.ok(m, 'inline script not found in ThemeScript.astro');
  assert.doesNotThrow(() => new vm.Script(m[1]));
});

test('landing links to docs', () => {
  assert.match(html(), /\/lazypost\/docs\//);
});
```

- [ ] **Step 2: Run to verify failure**

Run: `npm run build && npm test` — Expected: both new tests FAIL.

- [ ] **Step 3: Implement**

1. Create `site/src/components/ThemeScript.astro` with EXACTLY:

```astro
<script is:inline>
  (() => {
    const KEY = 'lazypost-theme';
    const TOKENS = ['bg', 'panel', 'raised', 'line', 'line2', 'navbar', 'text', 'muted', 'faint', 'accent', 'gold', 'cyan'];
    const buttons = [...document.querySelectorAll('[data-theme]')];
    if (!buttons.length) return;
    const apply = (name) => {
      const btn = buttons.find((b) => b.dataset.theme === name) || buttons[0];
      const root = document.documentElement.style;
      for (const key of TOKENS) {
        root.setProperty(`--color-${key}`, btn.dataset[key]);
      }
      buttons.forEach((b) => {
        const on = b === btn;
        b.setAttribute('aria-pressed', String(on));
        b.classList.toggle('border-accent', on);
        b.classList.toggle('text-text', on);
        b.classList.toggle('border-line', !on);
        b.classList.toggle('text-muted', !on);
      });
    };
    let saved = 'dracula';
    try {
      saved = localStorage.getItem(KEY) || 'dracula';
    } catch (e) {}
    apply(saved);
    buttons.forEach((b) => {
      b.addEventListener('click', () => {
        try {
          localStorage.setItem(KEY, b.dataset.theme);
        } catch (e) {}
        apply(b.dataset.theme);
      });
    });
  })();
</script>
```

2. In `site/src/components/Showcase.astro`: delete the ENTIRE `<script is:inline>...</script>` block (from `<script is:inline>` to `</script>`); everything else unchanged.

3. In `site/src/pages/index.astro`: add `import ThemeScript from '../components/ThemeScript.astro';` and render `<ThemeScript />` just before `</body>`.

4. In `site/src/components/Nav.astro`: add `const base = import.meta.env.BASE_URL;` to the frontmatter and a `docs` link in the `<nav>` BEFORE the star pill:

```astro
    <a href={`${base}docs/`} class="text-muted transition-colors hover:text-text">docs</a>
```

- [ ] **Step 4: Verify**

Run: `npm run build && npm test` — Expected: all tests green (existing `lazypost-theme` landing assertion should still hold — the script is now in the page via `<ThemeScript />`).

- [ ] **Step 5: Commit**

```bash
git add site/src/components/ThemeScript.astro site/src/components/Showcase.astro site/src/components/Nav.astro site/src/pages/index.astro && git commit -m "refactor(site): shared theme script, docs nav link"
```

---

### Task 2: DocsLayout, docs data, swatches, docs table styles

**Files:**
- Create: `site/src/components/Swatches.astro`, `site/src/layouts/DocsLayout.astro`, `site/src/data/docs.js`
- Modify: `site/src/components/Showcase.astro` (embed `<Swatches />`), `site/src/styles/global.css` (`.docs-table`)

**Interfaces:**
- Consumes: Task 1 `ThemeScript.astro`, `Nav.astro`; existing `Footer.astro`; `data/themes.js`.
- Produces: `Swatches.astro` (the 8 theme buttons, exactly the current Showcase markup); `docs.js` exporting `docs = [{href, title}]` in sidebar order; `DocsLayout.astro` accepting `title` and `current` props and rendering: `<Nav />`, a left sidebar (sticky; on small screens a `<details>`), a `<Swatches />` picker under the sidebar nav, the slot content, `<Footer />`, `<ThemeScript />`; `.docs-table` CSS in `global.css`.

- [ ] **Step 1: Add the failing tests**

Append to `site/tests/smoke.test.mjs`:

```js

test('docs index page builds', () => {
  assert.ok(existsSync(join(DIST, 'docs/index.html')), 'dist/docs/index.html must exist');
});

test('docs pages carry theme script and docs nav', () => {
  const d = page('docs/index.html');
  assert.match(d, /lazypost-theme/);
  assert.match(d, /\/lazypost\/docs\//);
});
```

- [ ] **Step 2: Run to verify failure** — build+test; expected FAIL (no docs/index.html yet).

- [ ] **Step 3: Implement**

1. Create `site/src/data/docs.js` with EXACTLY:

```js
export const docs = [
  { href: 'docs/quickstart', title: 'Quick start' },
  { href: 'docs/installation', title: 'Installation' },
  { href: 'docs/keybindings', title: 'Keybindings' },
  { href: 'docs/collections', title: 'Collections' },
  { href: 'docs/environments', title: 'Environments' },
  { href: 'docs/scripting', title: 'Scripting' },
  { href: 'docs/importers', title: 'Importers' },
  { href: 'docs/themes', title: 'Themes' },
  { href: 'docs/faq', title: 'FAQ' },
];
```

2. Create `site/src/components/Swatches.astro` — move the `<ul class="flex flex-wrap gap-2 p-4">…</ul>` themes block (the 8 buttons incl. their `data-theme/data-bg/.../data-cyan` attributes and aria-pressed) verbatim from `Showcase.astro` into this component (frontmatter: `import { themes } from '../data/themes.js';`). In `Showcase.astro` replace that `<ul>…</ul>` block with `<Swatches />` and add the import. Visual output identical.

3. Create `site/src/layouts/DocsLayout.astro` with EXACTLY:

```astro
---
import Nav from '../components/Nav.astro';
import Footer from '../components/Footer.astro';
import Swatches from '../components/Swatches.astro';
import ThemeScript from '../components/ThemeScript.astro';
import { docs } from '../data/docs.js';

const base = import.meta.env.BASE_URL;
const { title, current } = Astro.props;
---
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content={`lazypost docs — ${title}`} />
    <title>{`lazypost — ${title}`}</title>
    <link rel="icon" type="image/svg+xml" href={`${base}${'favicon.svg'}`} />
    <style>
      :global(body) { @apply bg-bg text-text font-sans antialiased; }
      .docs-table { width: 100%; border-collapse: collapse; }
      .docs-table th, .docs-table td {
        @apply border-b border-line px-3 py-2 text-left align-top;
      }
      .docs-table th { @apply bg-navbar text-muted font-mono text-xs font-bold; }
      .docs-table td { @apply font-mono text-xs text-text; }
      .docs-table tr:hover td { @apply bg-raised; }
    </style>
  </head>
  <body>
    <Nav />
    <div class="mx-auto flex max-w-6xl gap-10 px-6 py-10">
      <details class="lg:hidden"><summary class="font-mono text-xs text-muted">index</summary></details>
      <aside class="hidden w-52 shrink-0 lg:block">
        <nav class="sticky top-6 space-y-1">
          <p class="mb-2 font-mono text-xs font-bold text-faint">// docs</p>
          {docs.map((d) => (
            <a
              href={`${base}${d.href}/`}
              class:list={[
                'block rounded-md px-3 py-1.5 font-mono text-xs transition-colors',
                d.href === current
                  ? 'bg-raised text-accent'
                  : 'text-muted hover:bg-raised hover:text-text',
              ]}
            >
              {d.title}
            </a>
          ))}
          <p class="mb-2 mt-6 font-mono text-xs font-bold text-faint">// theme</p>
          <div class="flex flex-wrap gap-1.5"><Swatches /></div>
        </nav>
      </aside>
      <main class="min-w-0 flex-1 pb-16">
        <slot />
      </main>
    </div>
    <Footer />
    <ThemeScript />
  </body>
</html>
```

NOTE: the favicon link uses `{`base``}&#123;`${...}` patterns — write it exactly as above: `<link rel="icon" type="image/svg+xml" href={`${base}favicon.svg`} />` — the heredoc content above renders it literally; keep the EXACT template literal.

4. Append to `site/src/styles/global.css` (end of file):

```css
@layer components {
  .docs-table { width: 100%; border-collapse: collapse; }
  .docs-table th, .docs-table td { border-bottom: 1px solid var(--color-line); padding: 0.5rem 0.75rem; text-align: left; vertical-align: top; font-family: var(--font-mono); font-size: 0.75rem; }
  .docs-table th { background: var(--color-navbar); color: var(--color-muted); font-weight: 700; }
  .docs-table td { color: var(--color-text); }
  .docs-table tr:hover td { background: var(--color-raised); }
}
```
(If Tailwind v4 rejects `@layer components` there, use plain rules without the @layer wrapper — same selectors — and note it.)

5. Create `site/src/pages/docs/index.astro` (minimal, full version in Task 3) with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
---
<DocsLayout title="Docs" current="none">
  <h1 class="font-mono text-3xl font-extrabold text-text">docs</h1>
</DocsLayout>
```

- [ ] **Step 4: Verify**

Run: `npm run build && npm test` — all green (docs/index.html exists; theme script + nav link assertions pass).

- [ ] **Step 5: Commit**

```bash
git add site/src/components/Swatches.astro site/src/layouts/DocsLayout.astro site/src/data/docs.js site/src/components/Showcase.astro site/src/styles/global.css site/src/pages/docs/index.astro && git commit -m "feat(site): docs layout, swatches, table styles"
```

---

### Task 3: Docs index page (full)

**Files:** Modify `site/src/pages/docs/index.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs index content', () => {
  const d = page('docs/index.html');
  assert.match(d, /an API client that lives in your terminal/);
  assert.match(d, /Quick start/);
  assert.match(d, /Troubleshooting/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — replace the whole file with:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
---
<DocsLayout title="Docs" current="none">
  <h1 class="font-mono text-3xl font-extrabold text-text">// docs</h1>
  <p class="mt-4 max-w-2xl text-lg leading-relaxed text-muted">
    lazypost is <span class="font-mono text-accent">an API client that lives in your terminal</span> —
    Posting-inspired, built with Go and Bubble Tea. Requests are plain YAML files in a directory
    tree, so a collection is a folder you can version-control, diff, and share.
  </p>
  <div class="mt-10 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
    <a href={`${import.meta.env.BASE_URL}docs/quickstart/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Quick start</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">First request in under a minute.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/installation/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Installation</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">One command, a pinned version, or go build.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/keybindings/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Keybindings</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Every binding, grouped: global, sidebar, URL bar, editor, response.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/collections/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Collections</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Request YAML format, auth variants, folder structure.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/environments/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Environments</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">{{variable}} interpolation and the environment manager.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/scripting/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Scripting</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Sandboxed Lua pre/post hooks with a session store.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/importers/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Importers</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Postman and Insomnia into a lazypost collection, one command.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/themes/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">Themes</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Eight presets and custom YAML themes.</p>
    </a>
    <a href={`${import.meta.env.BASE_URL}docs/faq/`} class="rounded-lg border border-line bg-raised p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-accent">
      <div class="font-mono text-sm"><span class="text-accent">&gt;</span> <span class="font-bold text-text">FAQ</span></div>
      <p class="mt-2 font-mono text-xs leading-relaxed text-faint">Troubleshooting and behavior notes.</p>
    </a>
  </div>
</DocsLayout>
```

(NOTE: use `{'{'}{variable}'}` — no; the CI line `{{variable}} interpolation and the environment manager.` contains braces — write it as a frontmatter string: add `const envBlurb = '{{variable}} interpolation and the environment manager.';` to the frontmatter and render `{envBlurb}`.)

- [ ] **Step 4: Verify** — build+test green; `grep -o 'Quick start' dist/docs/index.html | wc -l` ≥ 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/index.astro && git commit -m "feat(site): docs index page"`

### Task 4: Installation page

**Files:** Create `site/src/pages/docs/installation.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs installation page', () => {
  const d = page('docs/installation/index.html');
  assert.match(d, /install\.sh \| sh/);
  assert.match(d, /sh -s -- v0\.4\.0/);
  assert.match(d, /go build -o lazypost \./);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL (page absent).

- [ ] **Step 3: Implement** — create the page with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
import Footer from '../../components/Footer.astro';
import { installUrl, version } from '../../data/site.js';
---
<DocsLayout title="Installation" current="docs/installation">
  <h1 class="font-mono text-3xl font-extrabold text-text">$ install lazypost</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    One command (checksum-verified; installs to <span class="font-mono text-xs text-text">~/.local/bin</span>, override with <span class="font-mono text-xs text-text">PREFIX</span>):
  </p>
  <div class="mt-6 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">lazypost — install.sh</div>
    <div class="border-b border-line p-4 font-mono text-xs">
      <div><span class="text-accent">$</span> <span class="text-muted">curl -fsSL {installUrl} | sh</span></div>
      <div class="mt-2"><span class="text-faint"># resolves the latest release, verifies checksums, installs to ~/.local/bin</span></div>
    </div>
    <div class="border-b border-line p-4 font-mono text-xs">
      <div><span class="text-accent">$</span> <span class="text-text">curl -fsSL {installUrl} | sh -s -- {version}</span></div>
      <div class="mt-2"><span class="text-faint"># pin a version; install location override: PREFIX=/usr/local sh install.sh</span></div>
    </div>
    <div class="p-4 font-mono text-xs">
      <div><span class="text-accent">$</span> <span class="text-text">go build -o lazypost .</span></div>
      <div class="mt-2"><span class="text-faint"># or build from source — Go 1.25+</span></div>
    </div>
  </div>
  <p class="mt-6 max-w-2xl leading-relaxed text-muted">
    Pre-built binaries for macOS and Linux (arm64 + amd64) are attached to every GitHub release.
    Re-running the install command updates to the latest release. Check what you are running with
    <span class="font-mono text-xs text-text">lazypost -version</span>.
  </p>
</DocsLayout>
```

NOTE: the heredoc line containing `curl -fsSL `{`...`}` for the first command is written exactly as: `<span class="text-muted">curl -fsSL {installUrl} | sh</span>` — i.e., a normal Astro expression for installUrl inside the span, and if the executor hits an Astro-esbuild parsing issue with `{installUrl}` inside the template, use the frontmatter string `const oneLiner = `curl -fsSL ${installUrl} | sh``; and render `{oneLiner}`. Pick the working variant; both must render the same visible text.

- [ ] **Step 4: Verify** — build+test green; `grep -o 'install.sh | sh' dist/docs/installation/index.html | wc -l` → 2.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/installation.astro && git commit -m "feat(site): docs installation page"`

---

### Task 5: Quick start page

**Files:** Create `site/src/pages/docs/quickstart.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs quickstart page', () => {
  const d = page('docs/quickstart/index.html');
  assert.match(d, /ctrl\+l edits the URL/);
  assert.match(d, /-dir my-collection/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
const base = import.meta.env.BASE_URL;
const steps = [
  ['run it anywhere', `lazypost`],
  ['create a request', 'n — then ctrl+l edits the URL, ctrl+t cycles the method'],
  ['save and send', 'ctrl+s saves, ctrl+r sends'],
  ['add environments', 'drop YAML files in environments/ and switch with ctrl+e'],
];
---
<DocsLayout title="Quick start" current="docs/quickstart">
  <h1 class="font-mono text-3xl font-extrabold text-text">quick start</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Run <span class="font-mono text-xs text-text">lazypost</span> in any directory — the current
    directory is initialized as a collection (with <span class="font-mono text-xs text-text">config/config.yaml</span>)
    when needed.
  </p>
  <ol class="mt-8 space-y-4">
    {steps.map(([title, body], i) => (
      <li class="rounded-lg border border-line bg-raised p-5">
        <div class="font-mono text-sm"><span class="text-accent">{i + 1}.</span> <span class="font-bold text-text">{title}</span></div>
        <p class="mt-2 font-mono text-xs leading-relaxed text-faint">{body}</p>
      </li>
    ))}
  </ol>
  <h2 class="mt-10 font-mono text-lg font-bold text-text">$ run</h2>
  <div class="mt-4 overflow-hidden rounded-xl border border-line2 bg-panel font-mono text-xs">
    <div class="border-b border-line bg-navbar px-4 py-2.5 text-faint"># from the repo, with the sample collections</div>
    <div class="border-b border-line p-4 text-text"><span class="text-accent">$</span> ./lazypost</div>
    <div class="p-4 text-text"><span class="text-accent">$</span> ./lazypost<span class="text-muted"> -dir my-collection</span><span class="text-faint">  # or point at your own collection directory</span></div>
  </div>
</DocsLayout>
```

- [ ] **Step 4: Verify** — build+test green; `grep -o 'my-collection' dist/docs/quickstart/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/quickstart.astro && git commit -m "feat(site): docs quickstart page"`

---

### Task 6: Keybindings page

**Files:** Create `site/src/pages/docs/keybindings.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs keybindings page', () => {
  const d = page('docs/keybindings/index.html');
  assert.match(d, /switch panes/);
  assert.match(d, /export the current request as curl/);
  assert.match(d, /NORMAL/);
  assert.match(d, /cycle auth type/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with the five tables below (verbatim from README). Table helper inline: each section is `<h2>` + `<div class="overflow-x-auto"><table class="docs-table">…</table></div>`. Use these exact rows:

Global:
| `tab` | switch panes |
| `ctrl+/` | command palette |
| `?` | keybindings panel |
| `ctrl+h` | request history |
| `ctrl+r` | send request |
| `ctrl+e` | cycle environment |
| `ctrl+s` | save request |
| `ctrl+l` | jump to the URL bar |
| `ctrl+g` | export the current request as curl |
| `q` | quit (collection / response / editor normal & visual modes) |
| `ctrl+c` | quit |

Collection · sidebar:
| `↑`/`↓`, `ctrl+n`/`ctrl+p` | navigate (loads the request) |
| `enter` | focus the URL bar / toggle folder (collection root toggles all) |
| `n` | new request |
| `a` | add request in folder; lead with `/` for a folder |
| `d` | delete (confirm with `y`) |
| `r` | rename |

URL bar:
| `ctrl+t` | cycle method |
| `enter` | send |
| `esc` | back to previous pane |
| paste `curl …` | import a curl command |

Editor (NORMAL-mode field; mode shown in a colored footer row):
| `i` / `a` / `A` / `I` / `o` / `O` | enter insert mode (editing) |
| `esc` | back to NORMAL |
| `hjkl` / `wbe` / `0$^` / `ggG` / `%` | motions |
| `x` / `dd` / `dw` / `d$` / `d0` | delete (with counts: `d2w`, `3dd`) |
| `yy` / `yw` / `y$` | yank (copied to the system clipboard) |
| `p` / `P` | paste the last yank/delete |
| `v` / `V` | visual selection (char/line); `y` yanks, `d` deletes |
| `q` | quit (normal/visual mode only; in insert it types) |
| `ctrl+n`/`ctrl+p` | move between sections |
| `alt+←`/`alt+→` | switch tabs |
| `ctrl+t` | cycle auth type |
| `ctrl+s` | save |

Response:
| `←`/`→`, `b`/`h` | switch tabs |
| `↑`/`↓` | scroll |
| `q` | quit |

Page shell: `<DocsLayout title="Keybindings" current="docs/keybindings">`, h1 `$ keybindings`, a lead `<p>`: "Every binding, grouped by pane. Keys render literally — `ctrl+l` means Ctrl+L." followed by the five sections. Table markup per row: `<tr><td class="whitespace-nowrap text-accent">…</td><td>…</td></tr>` for the key column.

- [ ] **Step 4: Verify** — build+test green; `grep -o 'cycle auth type' dist/docs/keybindings/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/keybindings.astro && git commit -m "feat(site): docs keybindings page"`

---

### Task 7: Collections page

**Files:** Create `site/src/pages/docs/collections.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs collections page', () => {
  const d = page('docs/collections/index.html');
  assert.match(d, /version: 1/);
  assert.match(d, /type: bear/);
  assert.match(d, /keyIn: query/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
const base = import.meta.env.BASE_URL;
---
<DocsLayout title="Collections" current="docs/collections">
  <h1 class="font-mono text-3xl font-extrabold text-text">collections</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    A collection is a directory of YAML files; subdirectories become folders. A root-level
    <span class="font-mono text-xs text-text">config/config.yaml</span> marks a directory as a collection:
  </p>
  <div class="mt-6 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">config/config.yaml</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text">version: 1</pre>
  </div>
  <p class="mt-6 max-w-2xl leading-relaxed text-muted">
    Open any directory with <span class="font-mono text-xs text-text">-dir</span> (or the current directory) and lazypost
    initializes it automatically. Legacy <span class="font-mono text-xs text-text">.lazypost</span> markers remain readable for the
    current session and migrate to <span class="font-mono text-xs text-text">config/config.yaml</span> on the first write.
    <span class="font-mono text-xs text-text">./sample-collections</span> / <span class="font-mono text-xs text-text">./collections</span>
    are implicit collections without a marker.
  </p>

  <h2 class="mt-10 font-mono text-lg font-bold text-text">request format</h2>
  <div class="mt-4 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">posts/list.yaml</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text">name: create post
method: POST
url: "&#123;{host}&#125;/posts"
headers:
  - name: Content-Type
    value: application/json
auth:
  type: bearer        # none | basic | bearer | apikey
  token: "&#123;{api_token}&#125;"
body: |
  {"title": "hello"}</pre>
  </div>

  <h2 class="mt-10 font-mono text-lg font-bold text-text">auth variants</h2>
  <div class="mt-4 overflow-x-auto">
    <table class="docs-table">
      <thead><tr><th>kind</th><th>yaml</th></tr></thead>
      <tbody>
        <tr><td class="text-accent">basic</td><td>auth: { type: basic, username: u, password: p }</td></tr>
        <tr><td class="text-accent">bearer</td><td>auth: { type: bearer, token: t }</td></tr>
        <tr><td class="text-accent">apikey</td><td>auth: { type: apikey, keyName: X-Api-Key, keyValue: k, keyIn: header } # or keyIn: query</td></tr>
      </tbody>
    </table>
  </div>
</DocsLayout>
```

NOTE: heredoc writers: the `{{host}}`/`{{api_token}}` inside the pre must be HTML-escaped as `&#123;{host}&#125;` (renders as `{{host}}`). Keep the request file byte-identical to README otherwise.

- [ ] **Step 4: Verify** — build+test green; `grep -o 'version: 1' dist/docs/collections/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/collections.astro && git commit -m "feat(site): docs collections page"`

---

### Task 8: Environments page

**Files:** Create `site/src/pages/docs/environments.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs environments page', () => {
  const d = page('docs/environments/index.html');
  assert.match(d, /api\.dev\.example\.com/);
  assert.match(d, /unknown placeholders are left as-is/);
  assert.match(d, /ctrl\+e/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
---
<DocsLayout title="Environments" current="docs/environments">
  <h1 class="font-mono text-3xl font-extrabold text-text">environments</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Put YAML files in <span class="font-mono text-xs text-text">&lt;collection&gt;/environments/</span>:
  </p>
  <div class="mt-6 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">environments/dev.yaml</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text"># environments/dev.yaml
variables:
  host: https://api.dev.example.com
  api_token: secret</pre>
  </div>
  <p class="mt-6 max-w-2xl leading-relaxed text-muted">
    Select an environment with <span class="font-mono text-xs text-accent">ctrl+e</span>;
    <span class="font-mono text-xs text-text">&#123;{host}&#125;</span>-style placeholders are substituted at send time.
    Unknown placeholders are left as-is — except in the URL, where an unresolved placeholder fails
    loudly and points you at <span class="font-mono text-xs text-accent">ctrl+e</span>.
  </p>
  <h2 class="mt-10 font-mono text-lg font-bold text-text">the environment manager</h2>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    <span class="font-mono text-xs text-accent">ctrl+/</span> opens the command palette: <span class="font-mono text-xs text-text">Environments</span>
    opens the environment manager — a tab bar of environments (<span class="font-mono text-xs text-accent">ctrl+e</span> cycles tabs,
    <span class="font-mono text-xs text-accent">a</span>/<span class="font-mono text-xs text-accent">r</span>/<span class="font-mono text-xs text-accent">d</span>
    add/edit/delete <span class="font-mono text-xs text-text">key=value</span> variables, <span class="font-mono text-xs text-text">enter</span> activates the tab's env).
    A leading <span class="font-mono text-xs text-accent">/</span> in the add-variable prompt creates a new empty environment instead.
  </p>
</DocsLayout>
```

- [ ] **Step 4: Verify** — build+test green; `grep -o 'api.dev.example.com' dist/docs/environments/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/environments.astro && git commit -m "feat(site): docs environments page"`

---

### Task 9: Scripting page

**Files:** Create `site/src/pages/docs/scripting.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs scripting page', () => {
  const d = page('docs/scripting/index.html');
  assert.match(d, /req\.headers\[/);
  assert.match(d, /os\.time/);
  assert.match(d, /expected 200/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
---
<DocsLayout title="Scripting" current="docs/scripting">
  <h1 class="font-mono text-3xl font-extrabold text-text">scripting</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Each request can carry <span class="font-mono text-xs text-accent">pre</span> and
    <span class="font-mono text-xs text-accent">post</span> hooks written in sandboxed Lua —
    no filesystem, no network; only <span class="font-mono text-xs text-text">os.time</span> is exposed.
    Edit them in the <span class="font-mono text-xs text-text">Scripts</span> tab of the request editor.
  </p>

  <h2 class="mt-10 font-mono text-lg font-bold text-text">pre</h2>
  <p class="mt-3 max-w-2xl leading-relaxed text-muted">
    Mutate the request before it is sent; a returned table merges into <span class="font-mono text-xs text-text">&#123;{...}&#125;</span> interpolation:
  </p>
  <div class="mt-4 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">scripts.lua — pre</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text"><span class="text-faint">-- pre</span>
req.headers[<span class="text-gold">"X-Session"</span>] = store.get(<span class="text-gold">"token"</span>)
req.query[<span class="text-gold">"page"</span>] = <span class="text-gold">"2"</span>
<span class="text-accent">return</span> { host = <span class="text-gold">"https://api.example.com"</span> }</pre>
  </div>

  <h2 class="mt-10 font-mono text-lg font-bold text-text">post</h2>
  <p class="mt-3 max-w-2xl leading-relaxed text-muted">
    Inspect the response; returning falsy or a string fails the send with that message:
  </p>
  <div class="mt-4 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">scripts.lua — post</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text"><span class="text-faint">-- post</span>
<span class="text-accent">if</span> response.status_code ~= 200 <span class="text-accent">then</span>
  <span class="text-accent">return</span> <span class="text-gold">"expected 200, got "</span> .. tostring(response.status_code)
<span class="text-accent">end</span>
<span class="text-accent">local</span> id = string.match(response.body, <span class="text-gold">'"id": (%d+)'</span>)
store.set(<span class="text-gold">"last_id"</span>, id)</pre>
  </div>

  <h2 class="mt-10 font-mono text-lg font-bold text-text">globals</h2>
  <div class="mt-4 overflow-x-auto">
    <table class="docs-table">
      <thead><tr><th>global</th><th>meaning</th></tr></thead>
      <tbody>
        <tr><td class="text-accent">req</td><td>method, url, body, headers, query — mutable in pre</td></tr>
        <tr><td class="text-accent">env</td><td>active variables</td></tr>
        <tr><td class="text-accent">store.get(key) / store.set(key, value)</td><td>session store — one response feeds the next request</td></tr>
        <tr><td class="text-accent">response</td><td>status, status_code, headers, body — post only</td></tr>
        <tr><td class="text-accent">os.time</td><td>the only exposed standard function</td></tr>
      </tbody>
    </table>
  </div>
</DocsLayout>
```

### Task 10: Importers page

**Files:** Create `site/src/pages/docs/importers.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs importers page', () => {
  const d = page('docs/importers/index.html');
  assert.match(d, /--dry-run/);
  assert.match(d, /workspace--environment/);
  assert.match(d, /--strict/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
const base = import.meta.env.BASE_URL;
---
<DocsLayout title="Importers" current="docs/importers">
  <h1 class="font-mono text-3xl font-extrabold text-text">$ lazypost import</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Converts Postman and Insomnia exports into a lazypost collection without any TUI interaction.
  </p>
  <div class="mt-6 overflow-hidden rounded-xl border border-line2 bg-panel font-mono text-xs">
    <div class="border-b border-line bg-navbar px-4 py-2.5 text-faint">examples</div>
    <div class="border-b border-line p-4 leading-6 text-text">
      <div><span class="text-accent">$</span> lazypost import postman-collection.json <span class="text-muted">-env postman-dev.json -env postman-prod.json -dir ./collections/my-api</span></div>
      <div class="mt-2"><span class="text-accent">$</span> lazypost import insomnia-export.yaml <span class="text-muted">-dir ./collections/my-api --dry-run</span></div>
      <div class="mt-2"><span class="text-accent">$</span> lazypost import insomnia-export-folder/ <span class="text-muted">-dir ./collections/my-api</span></div>
    </div>
  </div>
  <div class="mt-4 overflow-x-auto">
    <table class="docs-table">
      <thead><tr><th>flag</th><th>meaning</th></tr></thead>
      <tbody>
        <tr><td class="text-accent">-dir &lt;target&gt;</td><td>Required. Target collection directory; refuses to touch an existing directory unless --force.</td></tr>
        <tr><td class="text-accent">-env &lt;file&gt;</td><td>Import an environment file (Postman export JSON or Insomnia environment YAML). Repeatable.</td></tr>
        <tr><td class="text-accent">--format postman|insomnia</td><td>Override automatic format detection.</td></tr>
        <tr><td class="text-accent">--dry-run</td><td>Parse, validate, and print what would be imported — never writes. Also never creates the target directory.</td></tr>
        <tr><td class="text-accent">--force</td><td>Replace an existing target: moved aside and removed only after the new tree is in place.</td></tr>
        <tr><td class="text-accent">--strict</td><td>Fail the import when any warning is produced.</td></tr>
      </tbody>
    </table>
  </div>
  <h2 class="mt-10 font-mono text-lg font-bold text-text">behavior</h2>
  <ul class="mt-4 max-w-2xl space-y-3 leading-relaxed text-muted">
    <li><span class="text-accent">-</span> Sources: Postman Collection v2.1 JSON, Insomnia v4 JSON (<span class="font-mono text-xs text-text">__export_format: 4</span>), and Insomnia v5 YAML — a single file or a full export directory.</li>
    <li><span class="text-accent">-</span> Format detection is automatic from file contents; <span class="font-mono text-xs text-text">--format</span> only for ambiguous inputs.</li>
    <li><span class="text-accent">-</span> Workspaces become top-level folders; Insomnia export directories combine all workspaces, skipping unrelated resources (mock servers, OpenAPI documents) with warnings.</li>
    <li><span class="text-accent">-</span> Collection/base variables become a <span class="font-mono text-xs text-text">base</span> environment plus one per named environment; multi-workspace imports namespace them as <span class="font-mono text-xs text-text">workspace--environment</span>; Insomnia's <span class="font-mono text-xs text-text">&#123;{ _.var }}`</span> placeholders normalize to <span class="font-mono text-xs text-text">&#123;{var}}`</span>.</li>
    <li><span class="text-accent">-</span> Everything validates first, stages in a temporary sibling directory, then renames into place; filename collisions get deterministic <span class="font-mono text-xs text-text">-2</span>, <span class="font-mono text-xs text-text">-3</span> suffixes with warnings; a <span class="font-mono text-xs text-text">config/config.yaml</span> marker is created automatically.</li>
    <li><span class="text-accent">-</span> Unsupported features (JS pre/test scripts, multipart/binary/GraphQL bodies, unsupported auth) are reported per request and omitted — never guessed. <span class="font-mono text-xs text-text">--strict</span> promotes any warning to a failure.</li>
  </ul>
</DocsLayout>
```

- [ ] **Step 4: Verify** — build+test green; `grep -o 'workspace--environment' dist/docs/importers/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/importers.astro && git commit -m "feat(site): docs importers page"`

---

### Task 11: Themes page

**Files:** Create `site/src/pages/docs/themes.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs themes page', () => {
  const d = page('docs/themes/index.html');
  assert.match(d, /custom YAML themes/);
  assert.match(d, /$XDG_CONFIG_HOME|XDG_CONFIG_HOME/);
  assert.match(d, /example\.yaml/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with EXACTLY:

```astro
---
import DocsLayout from '../../layouts/DocsLayout.astro';
const base = import.meta.env.BASE_URL;
const presets = ['dracula', 'catppuccin', 'solarized', 'gruvbox', 'nord', 'tokyonight', 'one-dark', 'monokai'];
---
<DocsLayout title="Themes" current="docs/themes">
  <h1 class="font-mono text-3xl font-extrabold text-text">themes</h1>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Eight presets ship built in:
  </p>
  <div class="mt-4 flex flex-wrap gap-2">
    {presets.map((p) => (
      <span class="rounded-full border border-line bg-raised px-3 py-1.5 font-mono text-xs text-muted">{p}</span>
    ))}
  </div>
  <p class="mt-6 max-w-2xl leading-relaxed text-muted">
    Switch at runtime from the palette: <span class="font-mono text-xs text-accent">ctrl+/</span> then
    <span class="font-mono text-xs text-text">Switch theme</span>. The chosen theme is remembered between runs.
  </p>
  <h2 class="mt-10 font-mono text-lg font-bold text-text">custom YAML themes</h2>
  <p class="mt-4 max-w-2xl leading-relaxed text-muted">
    Custom themes live in <span class="font-mono text-xs text-text">~/.config/lazypost/themes/&lt;name&gt;.yaml</span>
    (or <span class="font-mono text-xs text-text">$XDG_CONFIG_HOME/lazypost/themes/</span>). Each file mirrors the theme
    colors as <span class="font-mono text-xs text-text">light</span>/<span class="font-mono text-xs text-text">dark</span> hex pairs;
    every key is optional and falls back to the default theme.
  </p>
  <p class="mt-6 font-mono text-sm text-faint">see the annotated template: <a href={`${base}docs/themes-example/`} class="text-accent hover:underline">example.yaml</a> — or in the repo: docs/themes/example.yaml</p>
  <div class="mt-6 overflow-hidden rounded-xl border border-line2 bg-panel">
    <div class="border-b border-line bg-navbar px-4 py-2.5 font-mono text-xs text-faint">example.yaml (abridged)</div>
    <pre class="overflow-x-auto p-4 font-mono text-xs leading-6 text-text"># docs/themes/example.yaml
accent: {{light: "#7aa2f7", dark: "#7aa2f7"}}
background: {light: "#c0caf5", dark: "#24283b"}
text: {light: "#0f111a", dark: "#c0caf5"}</pre>
  </div>
</DocsLayout>
```

- [ ] **Step 4: Verify** — build+test green; `grep -o 'XDG_CONFIG_HOME' dist/docs/themes/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/themes.astro && git commit -m "feat(site): docs themes page"`

---

### Task 12: FAQ page

**Files:** Create `site/src/pages/docs/faq.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs faq page', () => {
  const d = page('docs/faq/index.html');
  assert.match(d, /unsupported protocol scheme/);
  assert.match(d, /alt\+←|alt.*arrow/);
  assert.match(d, /--dry-run/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create with the Q&A list below (EXACT structure; each item is `<div class="rounded-lg border border-line bg-raised p-5"><div class="font-mono text-sm font-bold text-text">Q?</div><p class="mt-2 text-sm leading-relaxed text-muted">A</p></div>`):

1. **My first send fails with "unsupported protocol scheme"** — Most likely an unresolved URL placeholder: `{{host}}` with no active environment leaves escape-encoded braces in the URL. Set an environment (`ctrl+e`) — unresolved URL placeholders fail loudly and point at the environment switcher.
2. **Keyboard shortcuts with ctrl+arrows don't work** — macOS intercepts ctrl+←/→ for Mission Control/App Exposé. lazypost uses `alt+←/→` to switch editor tabs.
3. **Vim visual selection looks broken after the first token** — a known rendering quirk in early versions; the selection renderer strips ANSI codes first in current builds — update to the latest release.
4. **Yank vs paste** — yank copies to the system clipboard; paste reads the last internal yank/delete (not the system clipboard).
5. **My body JSON doesn't format** — bodies auto-format (2-space) on blur and on save; invalid JSON or placeholder-only bodies are never touched (placeholder-aware formatting keeps `{{var}}` intact).
6. **The response Headers tab is confusing** — the top row shows the exact executed URL (with substitutions) — check it if the response doesn't look like the request you expected.
7. **--dry-run said it would import but nothing appeared** — by design: `--dry-run` only reports and never creates the target directory.
8. **ctrl+e cycles my variables while I'm typing** — global bindings take precedence over editor defaults by design; type `{{` placeholders manually or bind around them — the editor powers live on the focused pane's bindings otherwise.
9. **A saved request shows a different name** — the sidebar name comes from the filename; the YAML `name:` field is display-only in the sidebar label context. Keep filenames descriptive.
10. **Docs, themes, or sample collections missing?** — `./sample-collections` and `./collections` are implicit collections but never auto-created; run lazypost in a directory and it initializes `config/config.yaml` for you.

Page shell: `<DocsLayout title="FAQ" current="docs/faq">`, h1 `// faq`, lead `<p>`: "Troubleshooting and behavior notes, drawn from real usage." then the 10 items stacked with `mt-4` (first with `mt-8`).

- [ ] **Step 4: Verify** — build+test green; `grep -o 'unsupported protocol scheme' dist/docs/faq/index.html | wc -l` → 1.
- [ ] **Step 5: Commit** — `git add site/src/pages/docs/faq.astro && git commit -m "feat(site): docs faq page"`

---

### Task 13: Docs themes example page + final verification

**Files:** Create `site/src/pages/docs/themes-example.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('docs themes example page', () => {
  const d = page('docs/themes-example/index.html');
  assert.match(d, /annotated template/);
  assert.match(d, /light/);
});
```

- [ ] **Step 2: Verify failure** — build+test; expected FAIL.

- [ ] **Step 3: Implement** — create `site/src/pages/docs/themes-example.astro`: `<DocsLayout title="example.yaml" current="none">`, h1 `example.yaml`, a `<pre>` with the FULL annotated template from `docs/themes/example.yaml` (read that file in the repo and port it verbatim into the pre; escape `{{`/`}}` as `&#123;`/`&#125;` if they appear; keep every comment line). If the file is short, the whole file; if long (>60 lines), include the first 40 lines plus a trailing comment `# ... (full file in the repo: docs/themes/example.yaml)`.

- [ ] **Step 4: End-to-end verification** (all on this branch):

```bash
cd site && npm run build && npm test   # expect ALL green ≥ 33 tests
```
then start preview and probe every docs route:
```bash
(cd site && (npm run preview -- --port 4321 >/dev/null 2>&1 &)) && sleep 4
for p in docs docs/quickstart docs/installation docs/keybindings docs/collections docs/environments docs/scripting docs/importers docs/themes docs/themes-example docs/faq; do
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:4321/lazypost/$p/")
  echo "$p -> $code"
done
pkill -f "astro preview" || true
```
Every route must print `200`. Also link-audit the internal hrefs: every `href="/lazypost/...` in dist/docs/*.html should have a matching file in dist (write a small python check: parse all hrefs containing '/lazypost/docs' across dist/docs, verify the target path exists as file or dir index.html).

- [ ] **Step 5: Commit** — `git add site/src/pages/docs/themes-example.astro && git commit -m "feat(site): docs themes example page"`

---

### Task 14: Landing nav polish + final commit

**Files:** Modify `site/src/components/Nav.astro`, `site/src/pages/index.astro`

- [ ] **Step 1: Add the failing test** — append:

```js

test('landing mounts theme script', () => {
  assert.match(html(), /lazypost-theme/);
});
```

(If this already passes because Task 1 added `<ThemeScript />`, still keep the test — it pins the landing behavior.)

- [ ] **Step 2: Implement** — nothing new unless the test fails; if green, skip implementation. Ensure `Nav.astro`'s docs link is present (Task 1) and `index.astro` mounts `<ThemeScript />` (Task 1). Verify the full suite one last time: `npm run build && npm test` ALL green.
- [ ] **Step 3: Final commit** (only if there are uncommitted changes — otherwise note "no changes"): `git add -A site && git commit -m "chore(site): docs section final polish"` (or skip if clean).
