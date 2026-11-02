// Shared helpers for the portal verification: a scratch laptop (HOME/PAWS_HOME/CLAUDE_CONFIG_DIR),
// the paws-backend API, a headless browser on the website. Configured only through the env that
// `portal.sh up` writes (run the scripts through `portal.sh run`, never directly).
import { execFileSync, execSync } from "node:child_process";
import { mkdirSync, writeFileSync, existsSync, readdirSync, readFileSync, appendFileSync } from "node:fs";
import { join } from "node:path";

const need = (k) => {
  const v = process.env[k];
  if (!v) {
    console.error(`missing ${k}: run the scripts through \`portal.sh run\` after \`portal.sh up\``);
    process.exit(2);
  }
  return v;
};

export const E = need("VERIFY_DIR"); // scratch root of this run
export const SITE = need("VERIFY_SITE"); // http://localhost:<site port>
export const API = need("VERIFY_API"); // http://127.0.0.1:<api port>
export const DB_URL = need("VERIFY_DB_URL");
export const PAWS_BIN = need("PAWS_BIN");
export const ARTIFACTS = need("VERIFY_ARTIFACTS");
export const BACKEND_DIR = need("VERIFY_BACKEND_DIR");
export const PYTHON = need("VERIFY_PYTHON");
export const USER = process.env.VERIFY_USER || "verify"; // the website's NEXT_PUBLIC_PAWS_DEV_TOKEN → user mock-verify
export const HOME = join(E, "home");
export const PAWS_HOME = join(HOME, ".paws");
export const CLAUDE_DIR = join(HOME, ".claude"); // CLAUDE_CONFIG_DIR of the scratch laptop
export const PROJ = join(HOME, "proj");
export const SHOTS = join(ARTIFACTS, "shots");
mkdirSync(SHOTS, { recursive: true });

const results = join(ARTIFACTS, "results.txt");
let failures = 0;
export function check(ok, what, extra = "") {
  const line = `  ${ok ? "PASS" : "FAIL"} ${what}`;
  console.log(line + (ok || !extra ? "" : `\n        ${String(extra).slice(0, 600)}`));
  appendFileSync(results, line + "\n");
  if (!ok) failures++;
}
export const step = (s) => {
  console.log(`\n• ${s}`);
  appendFileSync(results, `\n• ${s}\n`);
};
export const done = () => {
  console.log(failures ? `\n${failures} FAILED` : "\nALL PASS");
  process.exit(failures ? 1 : 0);
};

export const laptopEnv = (extra = {}) => ({
  ...process.env,
  HOME,
  PAWS_HOME,
  CLAUDE_CONFIG_DIR: CLAUDE_DIR,
  PAWS_PORTAL_API: API,
  PAWS_SYNC_INTERVAL: "0s",
  PAWS_CONFIG_INTERVAL: "0s",
  PATH: `${join(E, "fakebin")}:${process.env.PATH}`, // `open` is a recorder: no real browser
  ...extra,
});

/** Runs the paws CLI on the scratch laptop. Returns {code, out}. */
export function paws(args, { input, extra } = {}) {
  try {
    const out = execFileSync(PAWS_BIN, args, { cwd: PROJ, env: laptopEnv(extra), input, encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, out };
  } catch (e) {
    return { code: e.status, out: (e.stdout || "") + (e.stderr || "") };
  }
}

let toolSeq = 0;
let session = "verify-1";
export const setSession = (s) => (session = s);
/** One Claude Code hook event through `paws gate`, exactly as Claude Code sends it. Returns the hook output. */
export function gate(event, tool, toolInput, transcript = join(E, "transcript.jsonl"), extra = {}) {
  if (!existsSync(transcript)) writeFileSync(transcript, "");
  const ev = { session_id: session, transcript_path: transcript, cwd: PROJ, hook_event_name: event, tool_name: tool, tool_input: toolInput, tool_use_id: `toolu_${++toolSeq}` };
  const r = paws(["gate"], { input: JSON.stringify(ev), extra });
  try {
    return r.out.trim() ? JSON.parse(r.out) : {};
  } catch {
    return { raw: r.out, code: r.code };
  }
}
export const bash = (command, event = "PreToolUse", t) => gate(event, "Bash", { command }, t);
export const stop = () => gate("Stop", undefined, undefined);
export const denied = (out) => out.hookSpecificOutput?.permissionDecision === "deny";
export const reasonOf = (out) => out.hookSpecificOutput?.permissionDecisionReason || "";

/** A transcript in which Claude already loaded the given skills with the Skill tool. */
export function transcriptWith(skills, name = "t-loaded.jsonl") {
  const p = join(E, name);
  const lines = [];
  for (const s of skills) {
    lines.push({ type: "assistant", isSidechain: false, message: { id: `msg_${s}`, role: "assistant", content: [{ type: "tool_use", id: `toolu_${s}`, name: "Skill", input: { skill: s } }] } });
    lines.push({ type: "user", isSidechain: false, message: { role: "user", content: [{ type: "tool_result", tool_use_id: `toolu_${s}`, content: `Launching skill: ${s}` }] }, toolUseResult: { success: true, commandName: s } });
  }
  writeFileSync(p, lines.map((l) => JSON.stringify(l)).join("\n") + "\n");
  return p;
}

export async function api(path, { method = "GET", body, token = USER } = {}) {
  const res = await fetch(`${API}/paws/v1${path}`, {
    method,
    headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  return { status: res.status, json: await res.json().catch(() => null) };
}
export const skillRow = async (name, token = USER) => (await api("/skills", { token })).json.skills.find((s) => s.name === name);

export const psql = (sql) => execFileSync("psql", [DB_URL, "-tAc", sql], { encoding: "utf8" }).trim();
/** A paws-backend management command against this run's database (test mode). */
export const manage = (cmd) =>
  execSync(`${PYTHON} manage.py ${cmd} 2>&1`, {
    cwd: BACKEND_DIR,
    env: { ...process.env, ENVIRONMENT: "test", DEBUG: "true", DATABASE_URL: DB_URL },
  }).toString();

export const ls = (d) => (existsSync(d) ? readdirSync(d).sort() : []);
export const read = (p) => (existsSync(p) ? readFileSync(p, "utf8") : "");

export async function browser() {
  const pw = await import(need("PLAYWRIGHT_CORE"));
  const chromium = pw.chromium || pw.default.chromium;
  const b = await chromium.launch({ headless: true, executablePath: need("VERIFY_CHROME") });
  const ctx = await b.newContext({ viewport: { width: 1360, height: 900 } });
  const page = await ctx.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  page.on("console", (m) => m.type() === "error" && !/favicon|React DevTools|posthog/i.test(m.text()) && errors.push(m.text()));
  return { b, ctx, page, errors };
}
export const shot = (page, name) => page.screenshot({ path: join(SHOTS, `${name}.png`), fullPage: true });
export const text = (page) => page.locator("body").innerText();
