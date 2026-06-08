export function AuthorDate({ author, date }: { author?: string; date?: string }) {
  if (!author && !date) return null;
  return (
    <div className="min-w-0 truncate text-xs">
      {author ? <span className="">{author}</span> : null}
      {author && date ? <span className="mx-1">·</span> : null}
      {date ? <span className="">{date}</span> : null}
    </div>
  );
}

