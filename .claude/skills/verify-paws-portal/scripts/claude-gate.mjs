// Feature: the gate with REAL Claude Code, configured from the portal. Independent of the other features
// (its own portal user and scratch repo). Costs a few model calls.
//
// Isolated (preferred): ~/.config/paws-verify/claude-token holds a `claude setup-token` token, so Claude runs
// with CLAUDE_CONFIG_DIR=<scratch> and nothing touches the real ~/.claude. The portal turns on `git` (a full
// skill, installed into the scratch config) and `caveman-commit` as a short rule carrying a unique token.
// Fallback (no token file): Claude uses the developer's normal login with --setting-sources project, and only
// the rule skill is turned on, so paws installs nothing into the real ~/.claude.
import { execFileSync, spawnSync } from "node:child_process";
import { mkdirSync, writeFileSync, copyFileSync, existsSync, rmSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import { E, API, ARTIFACTS, PAWS_BIN, check, step, done, api, read, ls } from "./lib.mjs";

const TOKEN_FILE = process.env.VERIFY_CLAUDE_TOKEN_FILE || join(homedir(), ".config/paws-verify/claude-token");
const isolated = existsSync(TOKEN_FILE) && read(TOKEN_FILE).trim() !== "";
const USER = "verify-claude"; // its own portal account (test-mode bearer token)
const PROBE = `PAWS-${Math.random().toString(36).slice(2, 8).toUpperCase()}`;
const root = join(E, "claude-gate");
const proj = join(root, "proj");
const pawsHome = join(root, "paws");
const claudeDir = join(root, "claude");
const out = join(ARTIFACTS, "claude-gate");
const SESSION_MARKERS = ["CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SSE_PORT", "CLAUDE_CODE_EXECPATH", "CLAUDE_CODE_SESSION_ID",
  "CLAUDE_CODE_CHILD_SESSION", "CLAUDE_CODE_SESSION_ATTENDED", "CLAUDE_CODE_BRIDGE_SESSION_ID", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN"];

if (!isolated) {
  console.log(`\nNo Claude token at ${TOKEN_FILE}: running the fallback (normal login, rule skill only).
To run fully isolated (and cover a full skill loaded with the Skill tool), once, in your own terminal:
  claude setup-token            # sign in; it prints a long-lived token
  mkdir -p ~/.config/paws-verify && pbpaste > ${TOKEN_FILE} && chmod 600 ${TOKEN_FILE}
(copy the printed token first; never paste it into a chat)`);
}

function env(extra = {}) {
  const e = Object.fromEntries(Object.entries(process.env).filter(([k]) => !SESSION_MARKERS.includes(k)));
  Object.assign(e, { PAWS_HOME: pawsHome, PAWS_PORTAL_API: API, PAWS_SYNC_INTERVAL: "0s", PAWS_CONFIG_INTERVAL: "0s", ...extra });
  if (isolated) Object.assign(e, { CLAUDE_CONFIG_DIR: claudeDir, CLAUDE_CODE_OAUTH_TOKEN: read(TOKEN_FILE).trim() });
  else delete e.CLAUDE_CONFIG_DIR;
  return e;
}
const pawsRun = (args) => {
  const r = spawnSync(PAWS_BIN, args, { cwd: proj, env: env(), encoding: "utf8" });
  return { code: r.status, out: (r.stdout || "") + (r.stderr || "") };
};
const git = (cmd) => execFileSync("sh", ["-c", cmd], { cwd: proj, encoding: "utf8", env: { ...process.env, GIT_AUTHOR_NAME: "v", GIT_AUTHOR_EMAIL: "v@x", GIT_COMMITTER_NAME: "v", GIT_COMMITTER_EMAIL: "v@x" } });

step(`Real Claude Code gate (${isolated ? "isolated config + token" : "fallback: normal login, rule only"})`);
rmSync(root, { recursive: true, force: true });
for (const d of [proj, pawsHome, claudeDir, out]) mkdirSync(d, { recursive: true });
git("git init -q && git commit -q --allow-empty -m base && printf 'hello\\n' > notes.txt && git add notes.txt");
const init = pawsRun(["init"]);
check(init.code === 0 && existsSync(join(proj, ".claude/settings.json")), "paws init registered the hook in the scratch repo", init.out);

await api("/account", { method: "DELETE", token: USER });
const catalog = (await api("/catalog", { token: null })).json.skills;
const fromCatalog = (name, extra = {}) => {
  const c = catalog.find((s) => s.name === name);
  return { name: c.name, kind: c.kind, source: "catalog", load: "full", version: c.version, content_hash: c.content_hash, upstream: c.upstream, actions: c.default_actions, ...extra };
};
const wanted = [fromCatalog("caveman-commit", { load: "rule", rule_text: `Every commit subject must start with ${PROBE}: followed by a short summary.` })];
if (isolated) wanted.push(fromCatalog("git"));
for (const s of wanted) {
  const r = await api("/skills", { method: "POST", body: s, token: USER });
  check(r.status === 201 || r.status === 200, `portal: turned on ${s.name} (${s.load})`, JSON.stringify(r.json));
}

const key = (await api("/keys", { method: "POST", token: USER })).json.key;
check(pawsRun(["login", "--key", key]).code === 0, "laptop logged in with a one-time key");
const sync = pawsRun(["sync"]);
check(sync.code === 0, "paws sync pulled the portal skills", sync.out);
if (isolated) check(ls(join(claudeDir, "skills")).includes("git"), "git installed into the scratch CLAUDE_CONFIG_DIR");

if (process.env.VERIFY_SKIP_CLAUDE === "1") {
  console.log("    VERIFY_SKIP_CLAUDE=1: setup checked, claude not run");
  done();
}

step("Claude commits the staged change");
const args = ["-p", "Commit the staged change in this repo with a short message. Do it now, without asking.", "--model", process.env.VERIFY_MODEL || "sonnet",
  // --setting-sources project also hides user skills (CLAUDE_CONFIG_DIR/skills), where paws installs them,
  // so only the fallback uses it, to keep the developer's own hooks out. The isolated config has none.
  ...(isolated ? [] : ["--setting-sources", "project"]), "--output-format", "stream-json", "--verbose",
  "--allowedTools", "Bash(git:*)", "Skill", "--disallowedTools", "Edit(~/.paws/**)", "Edit(~/.claude/**)"];
const run = spawnSync("claude", args, { cwd: proj, env: env(), encoding: "utf8", timeout: 300000, maxBuffer: 64 << 20 });
writeFileSync(join(out, "stream.jsonl"), run.stdout || "");
const events = (run.stdout || "").split("\n").filter(Boolean).map((l) => { try { return JSON.parse(l); } catch { return {}; } });
const blocks = events.flatMap((e) => (Array.isArray(e.message?.content) ? e.message.content : []));
const toolUses = blocks.filter((c) => c.type === "tool_use");
const results = blocks.filter((c) => c.type === "tool_result").map((c) => (typeof c.content === "string" ? c.content : JSON.stringify(c.content)));
const final = events.find((e) => e.type === "result");
check(final && !final.is_error, "claude finished its turn", (run.stderr || "").slice(-400) || JSON.stringify(final).slice(0, 300));

const audit = read(join(pawsHome, "audit.log")).split("\n").filter(Boolean).map((l) => JSON.parse(l));
copyFileSync(join(proj, ".claude/settings.json"), join(out, "settings.json"));
writeFileSync(join(out, "audit.jsonl"), audit.map((a) => JSON.stringify(a)).join("\n") + "\n");
const log = git("git log --format=%s -3");
writeFileSync(join(out, "git-log.txt"), log);

check(audit.some((a) => a.verdict === "deny" && /caveman-commit/.test(a.rule || "")), "paws denied the first git commit (portal rule)", JSON.stringify(audit.filter((a) => a.verdict).map((a) => [a.rule, a.verdict, a.satisfied_by])));
check(results.some((r) => r.includes(PROBE)), `the deny carried the rule text (${PROBE}) to Claude`);
if (isolated) {
  check(toolUses.some((t) => t.name === "Skill" && /git/.test(JSON.stringify(t.input))), "Claude loaded the git skill with the Skill tool");
  check(audit.some((a) => /^git--/.test(a.rule || "") && a.verdict === "allow" && a.satisfied_by === "skill-load"), "retry allowed: git satisfied by skill-load");
}
check(log.split("\n")[0].startsWith(`${PROBE}:`), `the commit follows the rule: "${log.split("\n")[0]}"`);

step("The portal sees it");
let seen = null;
for (let i = 0; i < 20 && !seen; i++) {
  const h = (await api("/skills", { token: USER })).json.headline;
  if (h.about_to_act > 0) seen = h;
  else await new Promise((r) => setTimeout(r, 1000));
}
if (!seen) {
  console.log("    the async Stop hook did not upload before claude exited; uploading with paws sync");
  pawsRun(["sync"]);
  seen = (await api("/skills", { token: USER })).json.headline;
}
check(seen.about_to_act > 0, `portal counts the run: ${JSON.stringify(seen)}`);
console.log(`    evidence: ${out}`);
done();
