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
  assert.match(h, /★ star on github/);
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
  assert.match(h, /lazypost-theme/);
});

test('selection colors follow theme variables', () => {
  assert.match(css(), /var\(--color-accent\)/);
  assert.match(css(), /var\(--color-bg\)/);
});

test('theme switcher script parses in a browser-like context', () => {
  const src = read('site/src/components/Showcase.astro');
  const m = src.match(/<script is:inline>([\s\S]*?)<\/script>/);
  assert.ok(m, 'inline script not found in Showcase.astro');
  assert.doesNotThrow(() => new vm.Script(m[1]));
});

test('lua panel uses theme color classes', () => {
  const h = html();
  assert.match(h, /text-accent">local</);
  assert.match(h, /text-gold">"X-Session"/);
  assert.match(h, /text-faint">-- pre</);
});
