/* locale key parity checker — zh-cn is the source of truth; every other
 * language must mirror its key tree exactly (values may differ).
 *
 * Run: node scripts/check-locales.mjs   (or: pnpm locale:check)
 * Exits non-zero on drift so it can gate CI; prints each missing key with
 * its file and dotted path for a one-line fix. */
import ts from 'typescript';
import { existsSync, readdirSync, readFileSync } from 'node:fs';
import { basename, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'locales', 'langs');
const sourceLang = 'zh-cn';
const files = readdirSync(join(root, sourceLang)).filter(f => f.endsWith('.ts'));
// files compared directly must not be re-compared through index.ts imports —
// expanding them would double-report every drift under both paths
const directlyChecked = new Set(files);

/** collect the dotted key paths of the top-level object literal in a locale module */
function collectKeys(file) {
  const sf = ts.createSourceFile(file, readFileSync(file, 'utf8'), ts.ScriptTarget.ESNext, true);
  // default imports from local modules — a property whose value is one of
  // these identifiers contributes the imported file's keys (e.g. en-us
  // pulls `request` from ./request while zh-cn inlines it)
  const imports = new Map();
  sf.forEachChild(node => {
    if (
      ts.isImportDeclaration(node) &&
      node.importClause?.name &&
      ts.isStringLiteral(node.moduleSpecifier) &&
      node.moduleSpecifier.text.startsWith('.')
    ) {
      imports.set(node.importClause.name.text, join(dirname(file), `${node.moduleSpecifier.text}.ts`));
    }
  });
  let target;
  const scan = node => {
    if (target) return;
    if (ts.isVariableDeclaration(node) && node.initializer && ts.isObjectLiteralExpression(node.initializer)) {
      target = node.initializer;
      return;
    }
    ts.forEachChild(node, scan);
  };
  ts.forEachChild(sf, scan);
  if (!target) throw new Error(`no top-level object literal found in ${file}`);
  const keys = new Set();
  const addImported = (importName, path) => {
    const importedFile = imports.get(importName);
    if (!importedFile || !existsSync(importedFile)) return;
    if (directlyChecked.has(basename(importedFile))) return;
    for (const k of collectKeys(importedFile)) keys.add(path ? `${path}.${k}` : k);
  };
  const walk = (obj, prefix) => {
    for (const prop of obj.properties) {
      if (ts.isShorthandPropertyAssignment(prop)) {
        const name = prop.name.getText(sf);
        addImported(name, prefix ? `${prefix}.${name}` : name);
        continue;
      }
      if (!ts.isPropertyAssignment(prop)) continue;
      const name = prop.name.getText(sf).replace(/^['"]|['"]$/g, '');
      const path = prefix ? `${prefix}.${name}` : name;
      if (ts.isObjectLiteralExpression(prop.initializer)) walk(prop.initializer, path);
      else if (ts.isIdentifier(prop.initializer)) addImported(prop.initializer.getText(sf), path);
      else keys.add(path);
    }
  };
  walk(target, '');
  return keys;
}

let drift = 0;
for (const lang of readdirSync(root).filter(d => d !== sourceLang)) {
  for (const f of files) {
    const base = join(root, sourceLang, f);
    const mirror = join(root, lang, f);
    if (!existsSync(mirror)) continue;
    const baseKeys = collectKeys(base);
    const mirrorKeys = collectKeys(mirror);
    const missing = [...baseKeys].filter(k => !mirrorKeys.has(k)).sort();
    const extra = [...mirrorKeys].filter(k => !baseKeys.has(k)).sort();
    for (const k of missing) {
      console.error(`[${lang}/${f}] missing key: ${k}`);
      drift += 1;
    }
    for (const k of extra) {
      console.error(`[${lang}/${f}] extra key (not in ${sourceLang}): ${k}`);
      drift += 1;
    }
  }
}

if (drift > 0) {
  console.error(`\nlocale drift: ${drift} key(s) misaligned with ${sourceLang}`);
  process.exit(1);
}
console.log(`locale keys aligned (${files.length} file(s) x ${readdirSync(root).length - 1} language(s))`);
