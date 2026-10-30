// placeholder rows shown while the first page of results is still in flight —
// each row mirrors a result card's box (px-16px py-14px card + separator) so
// the reserved height lands inside the 131–153px range real cards measure at
// and swapping skeleton -> rows never moves the page
const ROW_COUNT = 6;

const SkeletonRow = ({ titleWidth, summaryLines }: { titleWidth: string; summaryLines: number[] }) => (
  <div className='animate-pulse px-16px py-14px'>
    <div className='flex items-center gap-8px'>
      <div className='h-18px w-18px flex-none rounded-4px bg-slate-200 dark:bg-slate-700' />
      <div className={`h-14px ${titleWidth} rounded-4px bg-slate-200 dark:bg-slate-700`} />
    </div>
    <div className='mt-10px flex flex-col gap-8px'>
      {summaryLines.map((width, index) => (
        <div
          className='h-12px rounded-4px bg-slate-200/90 dark:bg-slate-700/90'
          key={index}
          style={{ width: `${width}%` }}
        />
      ))}
    </div>
    <div className='mt-12px h-10px w-32% rounded-4px bg-slate-200/80 dark:bg-slate-700/80' />
  </div>
);

export function SkeletonList() {
  return (
    <div>
      {Array.from({ length: ROW_COUNT }, (_, index) => (
        <div key={index}>
          <SkeletonRow
            titleWidth={['w-55%', 'w-70%', 'w-45%', 'w-62%', 'w-50%', 'w-58%'][index % 6]}
            summaryLines={index % 2 === 0 ? [100, 92, 65] : [100, 58]}
          />
          {index < ROW_COUNT - 1 && (
            <div className='mx-16px border-b border-solid border-slate-200/70 dark:border-slate-700/50' />
          )}
        </div>
      ))}
    </div>
  );
}

export default SkeletonList;
