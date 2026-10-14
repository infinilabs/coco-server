// placeholder for the facet rail while the aggregation query is still in
// flight — mounted so the 240px sider column exists from the first paint and
// the center column never reflows when real facets pop it open. Rows mimic a
// FilterCheckboxGroup (title + checkbox rows); the group heights roughly match
// the default-expanded groups the real rail renders first
const SkeletonGroup = ({ rows, titleWidth }: { rows: number; titleWidth: string }) => (
  <div className='animate-pulse flex flex-col gap-10px'>
    <div className={`h-12px ${titleWidth} rounded-4px bg-slate-200 dark:bg-slate-700`} />
    {Array.from({ length: rows }, (_, index) => (
      <div className='flex items-center gap-8px' key={index}>
        <div className='h-14px w-14px flex-none rounded-2px border border-solid border-slate-300 dark:border-slate-600' />
        <div
          className='h-12px rounded-4px bg-slate-200/90 dark:bg-slate-700/90'
          style={{ width: `${72 - index * 8}%` }}
        />
      </div>
    ))}
  </div>
);

export function AggregationsSkeleton() {
  return (
    <div className='flex flex-col gap-24px pt-4px'>
      <SkeletonGroup rows={4} titleWidth='w-42%' />
      <SkeletonGroup rows={4} titleWidth='w-36%' />
      <SkeletonGroup rows={3} titleWidth='w-48%' />
    </div>
  );
}

export default AggregationsSkeleton;
