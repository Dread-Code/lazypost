import React from 'react';
import { useCallback, useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { docs } from '../data/docs.js';

const raw = import.meta.glob('../docs/*.md', { query: '?raw', import: 'default', eager: true });

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

  const markdown = raw[`../docs/${current}.md`] ?? '';

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
