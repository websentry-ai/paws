// Runs features in order and stops at the first failure (later features build on earlier state).
// `portal.sh run` → the browser/laptop suite; `portal.sh run claude-gate` → real Claude Code; any list works.
import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SUITE = ["onboard", "manage", "data", "updates", "account", "billing-jobs"];
const ALL = [...SUITE, "claude-gate"];
const asked = process.argv.slice(2);
const list = asked.length === 0 ? SUITE : asked[0] === "all" ? ALL : asked;
const unknown = list.filter((f) => !ALL.includes(f));
if (unknown.length) {
  console.error(`unknown feature(s): ${unknown.join(", ")}. Known: ${ALL.join(", ")}, or "all".`);
  process.exit(2);
}
const here = dirname(fileURLToPath(import.meta.url));
for (const f of list) {
  console.log(`\n==================== ${f} ====================`);
  const r = spawnSync(process.execPath, [join(here, `${f}.mjs`)], { stdio: "inherit" });
  if (r.status !== 0) {
    console.log(`\n${f}: FAILED (exit ${r.status}). Evidence: ${process.env.VERIFY_ARTIFACTS}`);
    process.exit(r.status || 1);
  }
}
console.log(`\nPASS: ${list.join(", ")}. Evidence: ${process.env.VERIFY_ARTIFACTS}`);
