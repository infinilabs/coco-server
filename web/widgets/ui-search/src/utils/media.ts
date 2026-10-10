// Shared helpers for rendering downloaded/binary content as images: fallback
// art, extension-aware filenames, and blob URLs that survive being opened or
// copied while the user is still looking at them.

// Glyph-only (transparent background) so the surrounding container's own tile
// background shows through in both light and dark themes, instead of the
// browser's default broken-image icon.
const IMAGE_FALLBACK_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 96 72"><g fill="none" stroke="#C9CDD4" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" opacity="0.9"><rect x="6" y="6" width="84" height="60" rx="8"/><circle cx="33" cy="25" r="6"/><path d="M14 57 L35 37 L47 49 L59 39 L82 57"/></g></svg>`;

export const IMAGE_FALLBACK_SRC = `data:image/svg+xml,${encodeURIComponent(IMAGE_FALLBACK_SVG)}`;

// A valid 1x1 transparent image for the window between render and the
// authorized blob landing: a src-less <img> paints the browser's
// broken-image glyph plus alt text, a decodable one paints nothing.
export const IMAGE_LOADING_SRC =
  "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7";

const MIME_EXTENSION_MAP: Record<string, string> = {
  jpeg: "jpg",
  "svg+xml": "svg",
  "quicktime": "mov",
  "msword": "doc",
  "vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
  "vnd.ms-powerpoint": "ppt",
  "vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
  "vnd.ms-excel": "xls",
  "vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
  "plain": "txt",
  "markdown": "md"
};

export function extensionFromMime(mime: string | undefined | null): string | undefined {
  if (typeof mime !== "string") return undefined;
  const subtype = mime.split(";")[0]?.split("/")[1]?.toLowerCase();
  if (!subtype) return undefined;
  return MIME_EXTENSION_MAP[subtype] ?? subtype;
}

// An attachment/title often has no extension (e.g. "深度研究报告"); a file
// saved that way is unopenable in most OSes, so append one whenever we can
// derive it from the document's metadata or the blob's own content type.
export function ensureFilenameExtension(filename: string | undefined, extension: string | undefined): string {
  const trimmed = (filename ?? "").trim();
  const ext = (extension ?? "").replace(/^\./, "").trim();
  if (!trimmed) return ext ? `file.${ext}` : "file";
  if (!ext || /\.[A-Za-z0-9]{1,8}$/.test(trimmed)) return trimmed;
  return `${trimmed}.${ext}`;
}

// Object URLs are opaque UUIDs; appending a `#name.ext` fragment keeps the
// URL resolvable (fragments are ignored when looking up the blob store) while
// making a copied/opened address — and the browser's save-file hint — carry a
// real filename with an extension instead of the bare UUID.
export function createNamedObjectUrl(blob: Blob, filename?: string): string {
  const url = URL.createObjectURL(blob);
  const name = filename?.trim().replace(/["#%<>\\^`{|}\s]+/g, "_");
  return name ? `${url}#${encodeURIComponent(name)}` : url;
}

// Revoking an object URL the instant its component unmounts breaks the tab the
// user just opened via "open in new tab" (and any copied address). Keep the URL
// alive for a grace period — bounded, since preview panes remount per document.
export function revokeObjectUrlLater(url: string, delayMs = 10 * 60 * 1000): void {
  window.setTimeout(() => URL.revokeObjectURL(url), delayMs);
}
