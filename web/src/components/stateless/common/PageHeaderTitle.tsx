import type { ReactNode } from 'react';

/**
 * Stacked card-head title block: icon + title on the first line, description
 * directly beneath it — rendered as ONE unit in the ACard `title` slot so every
 * first-level page carries the same 标题加描述 header.
 */
const PageHeaderTitle = ({ description, icon, title }: { description?: string; icon?: ReactNode; title: string }) => (
  <div className='py-16px'>
    <div className='flex items-center gap-8px'>
      {icon}
      <span className='text-16px font-500 leading-24px'>{title}</span>
    </div>
    {description ? (
      <div className='mt-6px text-12px color-[var(--ant-color-text-tertiary)] whitespace-normal break-words'>{description}</div>
    ) : null}
  </div>
);

export default PageHeaderTitle;
