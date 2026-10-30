import { RightOutlined } from '@ant-design/icons';
import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * Right-hand outline for the article read view: markdown headings extracted from the structured content, scroll-spy
 * highlight, click-to-scroll. Headings are located in the DOM by text, so the markdown renderer needs no anchor
 * plumbing.
 *
 * Rendered as a separate rail next to (not inside) the reading surface: a thin vertical track with an accent marker on
 * the active item, like Yuque/GitHub docs. The optional footer renders under the outline in the same sticky column —
 * the article's metadata card (aliases/tags/entities/relations/time) lives there, out of the reading flow.
 */
export function ArticleOutline({
  collapsed = false,
  content,
  footer,
  onToggle
}: {
  readonly collapsed?: boolean;
  readonly content: string;
  readonly footer?: ReactNode;
  readonly onToggle?: () => void;
}) {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const [active, setActive] = useState('');

  const headings = content
    .split('\n')
    .map(line => line.trim())
    .filter(line => /^#{1,3}\s+/.test(line))
    .map(line => ({
      level: line.match(/^#+/)?.[0].length ?? 2,
      text: line
        .replace(/^#+\s+/, '')
        .replace(/[#*`]/g, '')
        .trim()
    }))
    .filter(h => h.text);

  useEffect(() => {
    if (headings.length === 0) return undefined;

    const headingEls = () =>
      Array.from(rootRef.current?.closest('.wiki-article-page')?.querySelectorAll('h1, h2, h3') || []).map(el => ({
        el,
        text: (el.textContent || '').replace(/[#*`]/g, '').trim()
      }));

    const onScroll = () => {
      const els = headingEls();
      if (!els.length) return;
      let current = '';
      for (const { el, text } of els) {
        if (el.getBoundingClientRect().top <= 100) current = text;
      }
      setActive(current);
    };

    window.addEventListener('scroll', onScroll, true);
    onScroll();
    return () => window.removeEventListener('scroll', onScroll, true);
  }, [content, headings.length]);

  if (headings.length === 0) return null;
  // the rail also carries the footer metadata card — collapsing hides both;
  // the article header keeps a restore button
  if (collapsed) return null;

  const scrollTo = (text: string) => {
    const els = Array.from(rootRef.current?.closest('.wiki-article-page')?.querySelectorAll('h1, h2, h3') || []);
    const target = els.find(el => (el.textContent || '').replace(/[#*`]/g, '').trim() === text);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <nav
      aria-label={t('page.wiki.outline.title')}
      className='wiki-article-outline hidden w-220px shrink-0 xl:!block'
      ref={rootRef}
    >
      <div className='sticky top-12px max-h-[calc(100vh-120px)] overflow-y-auto pb-4 pr-2 pt-1'>
        <div className='mb-6px flex items-center justify-between gap-4px pl-12px pr-4px'>
          <span className='text-11px color-[var(--wiki-text-3)] font-600 tracking-0.08em uppercase'>
            {t('page.wiki.outline.title')}
          </span>
          {onToggle && (
            <button
              aria-label={t('page.wiki.outline.collapse')}
              className='wiki-outline-toggle'
              title={t('page.wiki.outline.collapse')}
              type='button'
              onClick={onToggle}
            >
              <RightOutlined />
            </button>
          )}
        </div>
        {/* quiet outline: the active item is darker, accented and carries a tiny
            glowing marker, hierarchy carried by indent and weight */}
        {headings.map((h, i) => {
          const isActive = active === h.text;
          return (
            <div
              key={`${h.text}-${i}`}
              style={{ paddingLeft: (h.level - 1) * 14 + 12 }}
              className={`relative cursor-pointer rounded-6px py-4px pr-6px text-13px leading-5 transition-colors ${
                isActive
                  ? 'font-550 color-[var(--wiki-accent)]'
                  : 'color-[var(--wiki-text-3)] hover:color-[var(--wiki-text)]'
              }`}
              onClick={() => scrollTo(h.text)}
            >
              {isActive && (
                <span className='absolute left-0 top-1/2 h-14px w-2px rounded-full bg-[var(--wiki-accent)] shadow-[0_0_6px_var(--wiki-accent-glow)] -translate-y-1/2' />
              )}
              <span className='block truncate'>{h.text}</span>
            </div>
          );
        })}
        {footer}
      </div>
    </nav>
  );
}
