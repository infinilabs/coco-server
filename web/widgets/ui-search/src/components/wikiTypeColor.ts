// Same palette and hash as the wiki console (WikiLinkTag / KbGraph) so an
// entity's type keeps its color across the article page, the graph canvas and
// the search result detail. Widget-local copy: the ui-search bundle cannot
// import from the host app's pages.
const TYPE_PALETTE = ['#722ed1', '#13c2c2', '#fa8c16', '#52c41a', '#eb2f96', '#2f54eb', '#faad14', '#a0d911'];

export function wikiTypeColor(type?: string): string {
  if (!type) return TYPE_PALETTE[0];
  let hash = 0;
  for (let i = 0; i < type.length; i += 1) hash = (hash * 31 + type.charCodeAt(i)) >>> 0;
  return TYPE_PALETTE[hash % TYPE_PALETTE.length];
}
