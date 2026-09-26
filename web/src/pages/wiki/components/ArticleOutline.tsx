import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * Right-hand outline for the article read view: markdown headings extracted from the
 * structured content, scroll-spy highlight, click-to-scroll. Headings are located in
 * the DOM by text, so the markdown renderer needs no anchor plumbing.
 */
export function ArticleOutline({ content }: { content: string }) {
  const { t } = useTranslation();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const [active, setActive] = useState('');

  const headings = content
    .split('\n')
    .map(line => line.trim())
    .filter(line => /^#{1,3}\s+/.test(line))
    .map(line => ({
      level: line.match(/^#+/)?.[0].length ?? 2,
      text: line.replace(/^#+\s+/, '').replace(/[#*`]/g, '').trim()
    }))
    .filter(h => h.text);

  useEffect(() => {
    if (headings.length === 0) return;

    const headingEls = () =>
      Array.from(rootRef.current?.closest('.wiki-article-page')?.querySelectorAll('h1, h2, h3') || []).map(
        el => ({ el, text: (el.textContent || '').replace(/[#*`]/g, '').trim() })
      );

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
  }, [content]);

  if (headings.length === 0) return null;

  const scrollTo = (text: string) => {
    const els = Array.from(
      rootRef.current?.closest('.wiki-article-page')?.querySelectorAll('h1, h2, h3') || []
    );
    const target = els.find(el => (el.textContent || '').replace(/[#*`]/g, '').trim() === text);
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <div className="hidden w-200px shrink-0 xl:!block" ref={rootRef}>
      <div className="sticky top-12px">
        <div className="mb-8px text-xs font-500 uppercase color-[var(--ant-color-text-tertiary)]">
          {t('page.wiki.outline.title')}
        </div>
        {headings.map((h, i) => (
          <div
            className={`cursor-pointer truncate rounded-4px py-3px text-13px ${
              active === h.text ? 'bg-[var(--ant-color-primary-bg)] font-500 text-primary' : 'color-[var(--ant-color-text-secondary)]'
            }`}
            key={`${h.text}-${i}`}
            style={{ paddingLeft: (h.level - 1) * 12 + 4 }}
            onClick={() => scrollTo(h.text)}
          >
            {h.text}
          </div>
        ))}
      </div>
    </div>
  );
}
