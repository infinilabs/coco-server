export function isBlobUrl(url: string | undefined | null): url is string {
  return typeof url === "string" && url.startsWith("blob:");
}
