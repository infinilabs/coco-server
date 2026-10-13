// The vite build empties ../.public (outDir + emptyOutDir), which also drops
// the widget assets the Go embed manifest serves from /widgets/*. Restore
// them from the local widget dist folders so a standalone web build keeps
// the embeddable preview working. Fresh clones without built dists just warn;
// `make build-widget` fills them in.
import { cp, mkdir } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const publicDir = path.join(webRoot, '..', '.public');
const widgets = ['searchbox', 'fullscreen'];

for (const name of widgets) {
  const dist = path.join(webRoot, 'widgets', name, 'dist');
  const target = path.join(publicDir, 'widgets', name);
  if (!existsSync(path.join(dist, 'index.js'))) {
    console.warn(`[copy-widgets] web/widgets/${name}/dist not built, skipping (run make build-widget)`);
    continue;
  }
  await mkdir(target, { recursive: true });
  await cp(path.join(dist, 'index.css'), path.join(target, 'index.css'));
  await cp(path.join(dist, 'index.js'), path.join(target, 'index.js'));
  console.log(`[copy-widgets] restored widgets/${name}`);
}
