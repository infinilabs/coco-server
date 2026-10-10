// Re-baseline the type-check ratchet to the current error count.
// Run locally after paying down errors, then commit the updated file.
import { execSync } from "node:child_process";
import { writeFileSync } from "node:fs";
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
writeFileSync(baselineFile, `${count}\n`);
console.log(`baseline updated to ${count}`);
