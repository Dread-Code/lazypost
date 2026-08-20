import React from 'react';
import { useCallback, useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { docs } from '../data/docs.js';

const raw = import.meta.glob('../docs/*.md', { query: '?raw', import: 'default', eager: true });

function Tint({ line }) {
  const out = [];
  let rest = line;
  if (rest.startsWith('$ ')) {
    out.push(
      <span className="text-accent" key="p">
        {'$ '}
      </span>
    );
    rest = rest.slice(2);
  }
  if (/^(#|--)\s/.test(rest)) {
    out.push(
      <span className="text-faint" key="c">
        {rest}
      </span>
    );
    return <>{out}</>;
  }
  const parts = rest.split(/("[^"]*")/).filter(Boolean);
  parts.forEach((p, i) => {
    out.push(
      p.startsWith('"') ? (
        <span className="text-gold" key={i}>
          {p}
        </span>
      ) : (
        <span key={i}>{p}</span>
      )
    );
  });
  return <>{out}</>;
}

function CodeBlock({ className, children }) {
  const lang = String(className ?? '').replace(/^language-/, '') || 'text';
  const text = String(children ?? '').replace(/\n$/, '');
  const lines = text.split('\n');
  return (
    <div className="not-prose overflow-hidden rounded-xl border border-line2 bg-panel">
      <div className="border-b border-line bg-navbar px-4 py-2 font-mono text-xs font-bold text-faint">
        <span>{lang}</span>
      </div>
      <pre className="overflow-x-auto p-4 font-mono text-xs leading-6 text-text">
        {lines.map((ln, i) => (
          <span className="block" key={i}>
            <Tint line={ln} />
          </span>
        ))}
      </pre>
    </div>
  );
}

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
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            pre({ children }) {
              return <>{children}</>;
            },
            table({ children }) {
              return (
                <div className="not-prose overflow-x-auto rounded-xl border border-line2 bg-panel">
                  <table className="docs-terminal-table">{children}</table>
                </div>
              );
            },
            code({ className, children }) {
              if (className) {
                return <CodeBlock className={className} children={children} />;
              }
              return (
                <code className="rounded-md border border-line bg-raised px-1 py-0.5 text-gold">
                  {children}
                </code>
              );
            },
          }}
        >
          {markdown}
        </ReactMarkdown>
      </article>
    </div>
  );
}
