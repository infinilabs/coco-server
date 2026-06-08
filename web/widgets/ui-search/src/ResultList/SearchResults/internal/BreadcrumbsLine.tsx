export function BreadcrumbsLine({ breadcrumbs }: { breadcrumbs?: string[] }) {
  if (!breadcrumbs?.length) return null;
  return (
    <div className="min-w-0 truncate text-xs">
      {breadcrumbs.map((text, index) => (
        <span key={`${text}-${index}`}>
          {index > 0 ? <span className="mx-1">{">"}</span> : null}
          <span className="">{text}</span>
        </span>
      ))}
    </div>
  );
}

