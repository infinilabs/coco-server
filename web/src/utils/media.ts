// Shared image placeholder art for the main app, mirroring
// web/widgets/ui-search/src/utils/media.ts — keep the two copies visually
// identical so search results and knowledge-base pages show one style.

// Glyph-only (transparent background) so the surrounding tile's own
// background shows through in both light and dark themes, instead of the
// browser's default broken-image icon.
const IMAGE_FALLBACK_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 96 72"><g fill="none" stroke="#C9CDD4" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" opacity="0.9"><rect x="6" y="6" width="84" height="60" rx="8"/><circle cx="33" cy="25" r="6"/><path d="M14 57 L35 37 L47 49 L59 39 L82 57"/></g></svg>`;

export const IMAGE_FALLBACK_SRC = `data:image/svg+xml,${encodeURIComponent(IMAGE_FALLBACK_SVG)}`;
