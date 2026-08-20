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
