export function joinRel(rel?: string, target?: string) {
  const defaultRel = target === "_blank" ? "noreferrer noopener" : "";
  if (!rel) return defaultRel || undefined;
  if (!defaultRel) return rel;
  const set = new Set([...rel.split(" "), ...defaultRel.split(" ")].filter(Boolean));
  return [...set].join(" ");
}

