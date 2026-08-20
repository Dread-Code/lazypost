import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import vm from 'node:vm';

export const ROOT = join(import.meta.dirname, '..', '..');
export const DIST = join(import.meta.dirname, '..', 'dist');

export const html = () => {
  const p = join(DIST, 'index.html');
  return existsSync(p) ? readFileSync(p, 'utf8') : '';
};

export const page = (p) => {
  const full = join(DIST, p);
  return existsSync(full) ? readFileSync(full, 'utf8') : '';
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

test('css emits design tokens', () => {
  assert.match(css(), /--color-accent:/, 'compiled css must contain --color-accent');
});

test('css sets smooth scrolling', () => {
  assert.match(css(), /scroll-behavior:smooth/);
});

test('hero copy, ctas and badges render', () => {
  const h = html();
  assert.match(h, /An API client that lives in your terminal\./);
  assert.match(h, /\$ get started/);
  assert.match(h, /view on github/);
  assert.match(h, /macOS/);
  assert.match(h, /Linux/);
  assert.match(h, /MIT License/);
  assert.match(h, /1\.25\+/);
  assert.doesNotMatch(h, /Go 1\.25\+/);
  assert.match(h, /v0\.4\.0/);
  assert.match(h, /img\.shields\.io/);
  assert.match(h, /github\.com\/Dread-Code\/lazypost/);
  assert.match(h, /href="#install"/);
});

test('terminal window renders the top bar and screenshot', () => {
  const h = html();
  assert.match(h, /lazypost — ~\/collections/);
  assert.match(h, /alt="lazypost terminal window"/);
});

test('what-is section renders', () => {
  const h = html();
  assert.match(h, /\/\/ what is lazypost/);
  assert.match(h, /see the docs/);
  assert.match(h, /Posting-inspired/);
  assert.match(h, /Bubble Tea/);
  assert.match(h, /sandboxed <span class="font-mono text-accent">Lua<\/span>/);
});

test('features grid renders six cards', () => {
  const h = html();
  for (const title of ['vim editing', 'yaml collections', 'lua scripting', 'environments', 'importers', 'themes']) {
    assert.match(h, new RegExp(`> ${title}`), `missing feature: ${title}`);
  }
  assert.match(h, /\{\{variable\}\} interpolation/);
});

test('install section renders tabs and commands', () => {
  const h = html();
  assert.match(h, /\$ install lazypost/);
  assert.match(h, /id="install"/);
  assert.match(h, /curl \| sh/);
  assert.match(h, /pin a version/);
  assert.match(h, /go build/);
  assert.match(h, /install\.sh \| sh/);
  assert.match(h, /sh -s -- v0\.4\.0/);
  assert.match(h, /PREFIX=\/usr\/local/);
  assert.match(h, /then run/);
  assert.match(h, /aria-selected="true"/);
});

test('showcase section renders panels', () => {
  const h = html();
  assert.match(h, /\$ lazypost/);
  assert.match(h, /a look around/);
  for (const label of ['vim editing', 'lua chaining', 'curl import', 'themes']) {
    assert.match(h, new RegExp(label), `missing showcase panel: ${label}`);
  }
  assert.match(h, /X-Session/);
  assert.match(h, /ctrl\+g/);
  assert.match(h, /store<\/span>\.<span class="text-text">set<\/span>/);
});

test('showcase screenshot asset emitted', () => {
  const dir = join(DIST, '_astro');
  assert.ok(existsSync(dir) && readdirSync(dir).some((n) => n.endsWith('.png')), 'dist/_astro must contain a hashed png');
});

test('community strip and footer render', () => {
  const h = html();
  assert.match(h, /Open source, MIT licensed\./);
  assert.match(h, /star on github/);
  assert.match(h, /M12 17\.27/);
  assert.match(h, /lazypost<\/span> · MIT License · by\n<a href="https:\/\/github\.com\/Dread-Code"/);
  assert.match(h, /target="_blank" rel="noopener"/);
});

test('pages workflow exists and deploys site/dist', () => {
  const wf = read('.github/workflows/pages.yml');
  assert.match(wf, /actions\/deploy-pages@v4/);
  assert.match(wf, /path: site\/dist/);
  assert.match(wf, /npm run build/);
  assert.match(wf, /paths:\s*\[\s*site\/\*\*/s);
});

test('css uses dracula palette', () => {
  const c = css();
  assert.match(c, /--color-accent:\s*#bd93f9/);
  assert.match(c, /--color-bg:\s*#282a36/);
  assert.match(c, /--color-text:\s*#f8f8f2/);
});

test('hero terminal has glow backdrop', () => {
  const h = html();
  assert.match(h, /blur-3xl/);
  assert.match(h, /bg-accent\/20/);
  assert.match(h, /aria-hidden="true"/);
});

test('theme swatch buttons carry theme data', () => {
  const h = html();
  assert.match(h, /data-theme="dracula"/);
  assert.match(h, /data-accent=/);
  assert.match(h, /data-bg=/);
  assert.match(h, /data-cyan=/);
});

test('selection colors follow theme variables', () => {
  assert.match(css(), /var\(--color-accent\)/);
  assert.match(css(), /var\(--color-bg\)/);
});

test('lua panel uses theme color classes', () => {
  const h = html();
  assert.match(h, /text-accent">local</);
  assert.match(h, /text-gold">"X-Session"/);
  assert.match(h, /text-faint">-- pre</);
});

test('lua globals colored consistently in pre and post', () => {
  const h = html();
  assert.match(h, /text-cyan">req<\/span>\.<span class="text-text">headers<\/span>/);
  assert.match(h, /text-cyan">store<\/span>\.<span class="text-text">get<\/span>/);
  assert.match(h, /text-cyan">req<\/span>\.<span class="text-text">query<\/span>/);
  assert.doesNotMatch(h, /text-accent">req<\/span>/);
});

test('importers panel renders', () => {
  const h = html();
  assert.match(h, /postman-collection\.json/);
  assert.match(h, /insomnia-export\.yaml/);
  assert.match(h, /--dry-run/);
});

test('page() helper reads built docs files', () => {
  assert.equal(page('index.html'), html());
  assert.equal(page('nope/index.html'), '');
});

test('theme script parses in a browser-like context', () => {
  const src = read('site/src/components/ThemeScript.astro');
  const m = src.match(/<script>([\s\S]*?)<\/script>/);
  assert.ok(m, 'script not found in ThemeScript.astro');
  const body = m[1].replace(/^import[^\n]*\n/, '');
  assert.doesNotThrow(() => new vm.Script(body));
});

test('landing links to docs', () => {
  assert.match(html(), /\/lazypost\/docs\//);
});

test('docs index page builds', () => {
  assert.ok(existsSync(join(DIST, 'docs/index.html')), 'dist/docs/index.html must exist');
});

test('docs routes pre-render with content', () => {
  for (const slug of DOC_SLUGS) {
    const p = slug === 'index' ? 'docs/index.html' : `docs/${slug}/index.html`;
    assert.match(page(p), new RegExp(DOC_SIGNATURES[slug]), `missing content on ${p}`);
  }
});

test('docs use real urls, no hash routing', () => {
  const d = page('docs/keybindings/index.html');
  assert.match(d, /\/lazypost\/docs\/faq\//);
  assert.doesNotMatch(d, /#\/|href="#/);
});

test('old docs routes are gone', () => {
  const dir = join(DIST, 'docs');
  assert.ok(existsSync(dir), 'docs dir missing');
  const names = readdirSync(dir);
  for (const n of names) {
    const base = n.endsWith('.html') ? n.slice(0, -5) : n;
    assert.ok(DOC_SLUGS.includes(base), `unexpected docs entry: ${n}`);
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

test('theme script bundle ships lazypost-theme', () => {
  const dir = join(DIST, '_astro');
  const js = existsSync(dir) ? readdirSync(dir).filter((n) => n.endsWith('.js')) : [];
  assert.ok(js.length > 0, 'no bundled js in dist/_astro');
  const all = js.map((f) => readFileSync(join(dir, f), 'utf8')).join('\n');
  assert.match(all, /lazypost-theme/);
});

test('docs pages link the compiled stylesheet', () => {
  for (const p of ['docs/index.html', 'docs/faq/index.html', 'docs/keybindings/index.html']) {
    assert.match(page(p), /<link rel="stylesheet" href="[^"]*\.css"/, `missing stylesheet link on ${p}`);
  }
});

test('nav logo links to the home page', () => {
  assert.match(html(), /href="\/lazypost\/"/);
  assert.match(page('docs/index.html'), /href="\/lazypost\/"/);
});

test('css ships typography prose styles', () => {
  assert.match(css(), /\.prose/);
});

const DOC_SLUGS = ['index', 'quickstart', 'installation', 'keybindings', 'collections', 'environments', 'scripting', 'importers', 'themes', 'faq'];

const DOC_SIGNATURES = {
  index: 'Quick start',
  quickstart: 'ctrl\\+l edits the URL',
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
  for (const slug of DOC_SLUGS.filter((s) => s !== 'index')) {
    assert.match(read(`site/src/docs/${slug}.md`), new RegExp(DOC_SIGNATURES[slug]), `missing signature in docs/${slug}.md`);
  }
});

test('docs store anchors to real urls', () => {
  const src = read('site/src/components/DocsApp.jsx');
  assert.match(src, /pushState/);
  assert.match(src, /react-markdown/);
  assert.match(src, /popstate/);
});

test('docs home is the classic card grid page', () => {
  const d = page('docs/index.html');
  assert.match(d, /\/\/ docs/);
  assert.match(d, /First request in under a minute\./);
  assert.match(d, /hidden w-52 shrink-0 lg:block/);
  assert.doesNotMatch(d, /<h1[^>]*>docs<\/h1>/);
});

test('footer renders only on the docs home', () => {
  assert.match(page('docs/index.html'), /MIT License/);
  assert.doesNotMatch(page('docs/keybindings/index.html'), /<footer/);
});
