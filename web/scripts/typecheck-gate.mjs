// Type-check ratchet gate.
//
// The codebase carries a large pre-existing type-error debt (see
// .typecheck-baseline). Making `tsc --noEmit` hard-fail today would block
// every PR; doing nothing lets runtime-breaking type errors ship green
// (vite/esbuild strips types without checking them). The ratchet keeps both
// risks in check: CI fails only when the error count GROWS past the
// committed baseline, and the baseline is lowered as debt is paid down.
//
// Lower the baseline after fixing errors:  pnpm typecheck:update-baseline
import { execSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const webRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const baselineFile = path.join(webRoot, ".typecheck-baseline");

let output = "";
try {
  output = execSync("pnpm exec tsc --noEmit --skipLibCheck", {
    cwd: webRoot,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
    maxBuffer: 128 * 1024 * 1024,
  });
} catch (error) {
  output = `${error.stdout || ""}${error.stderr || ""}`;
}

const count = (output.match(/error TS/g) || []).length;
const baseline = existsSync(baselineFile)
  ? Number.parseInt(readFileSync(baselineFile, "utf8").trim(), 10)
  : 0;

if (!Number.isFinite(baseline) || baseline < 0) {
  console.error(`invalid baseline in ${baselineFile}: expected a non-negative integer`);
  process.exit(1);
}

if (count > baseline) {
  console.error(`typecheck errors grew: ${count} > baseline ${baseline}`);
  console.error("fix the new errors before merging;");
  console.error("if an error count change is intentional, run: pnpm typecheck:update-baseline");
  process.exit(1);
}

console.log(`typecheck: ${count} errors (baseline ${baseline}) — OK`);
