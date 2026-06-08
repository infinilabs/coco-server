export function TagPill({ text }: { text: string }) {
  return (
    <span className="inline-flex items-center rounded-md border border-slate-200 bg-white px-2 py-1 text-xs text-slate-700">
      {text}
    </span>
  );
}

