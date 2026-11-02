// Feature: counts, uploads + redaction, insights history, suggestions. Run after onboard + manage.
import { join } from "node:path";
import { HOME, PAWS_HOME, SITE, API, check, step, done, gate, bash, stop, transcriptWith, setSession, api, psql, ls, read, paws, browser, shot, text } from "./lib.mjs";

const skills = async () => (await api("/skills")).json;
const row = async (name) => (await skills()).skills.find((s) => s.name === name);

const { b, page, errors } = await browser();
try {
  step("A new session where Claude reads git by itself counts as 'Claude read it'");
  const before = await row("git");
  setSession("s2-claude-read");
  const r = bash('git commit -m "by itself"', "PreToolUse", transcriptWith(["git", "humanizer"], "t-s2.jsonl"));
  check(!r.hookSpecificOutput?.permissionDecision, "allowed: already read");
  stop();
  const after = await row("git");
  check(after.claude_read === before.claude_read + 1, `claude_read ${before.claude_read} → ${after.claude_read}`, JSON.stringify(after));
  await page.goto(`${SITE}/portal?selected=${after.id}`);
  await page.getByText("Claude read it", { exact: true }).nth(1).waitFor({ timeout: 20000 });
  check(true, "recent runs show a 'Claude read it' run");

  step("Uploads are redacted; Settings shows the last upload");
  const SECRET = "sk-live-9f8e7d6c5b4a39281706f5e4d3c2b1a0";
  gate("PostToolUse", "Bash", { command: `curl -H "Authorization: Bearer ${SECRET}" "https://api.stripe.com/v1/charges?api_key=hunter2hunter2"` });
  gate("PostToolUse", "Edit", { file_path: join(HOME, "proj/src/app.ts"), old_string: "a", new_string: "b" });
  const dry = paws(["sync", "--dry-run"]);
  check(!dry.out.includes(SECRET) && !dry.out.includes("hunter2hunter2"), "sync --dry-run shows no secret", dry.out.slice(0, 800));
  check(!dry.out.includes(HOME), "no absolute paths in the upload", dry.out.match(/"file":"[^"]*"/g)?.join(" "));
  stop();
  const last = await api("/uploads/last");
  const lastStr = JSON.stringify(last.json);
  check(last.status === 200 && last.json.count >= 2, "API /uploads/last has the records", lastStr.slice(0, 300));
  check(!lastStr.includes(SECRET) && !lastStr.includes("hunter2hunter2"), "stored upload has no secret", lastStr.slice(0, 600));
  check(/src\/app\.ts/.test(lastStr), "file path stored relative to the repo");
  check(!/"(prompt|content|old_string|new_string)"/.test(lastStr), "no prompts or file contents uploaded");
  await page.goto(`${SITE}/portal/settings`);
  await page.getByText("Last upload").waitFor({ timeout: 20000 });
  await page.waitForTimeout(1000);
  const st = await text(page);
  check(/curl/.test(st) && !st.includes(SECRET), "Settings → Last upload shows the redacted command", st.slice(st.indexOf("Last upload"), st.indexOf("Last upload") + 600));
  await shot(page, "20-settings");

  step("History over 7 days → Insights chart, recent loads, per-skill table");
  const token = JSON.parse(read(join(PAWS_HOME, "credentials.json"))).token;
  const records = [];
  for (let d = 1; d <= 7; d++)
    for (let i = 0; i < d + 2; i++) {
      const ts = new Date(Date.now() - d * 86400e3 - i * 60e3).toISOString().replace(/\.\d+Z$/, "Z");
      const already = i % 3 === 0;
      records.push({ kind: "rule", ts, event: "pre-tool", skill: i % 2 ? "humanizer" : "git", command: i % 2 ? "gh pr create --title x" : "git commit -m x", tool: "Bash", verdict: already ? "allow" : "deny", already_loaded: false });
      if (already) records[records.length - 1] = { ...records[records.length - 1], verdict: "allow", already_loaded: true, satisfied_by: "skill-load" };
    }
  const res = await fetch(`${API}/paws/v1/sync`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" }, body: JSON.stringify({ records, applied_config_hash: null, inventory: { skills: [], rules: [] } }) });
  const sj = await res.json();
  check(res.status === 200 && sj.accepted === records.length, `device sync accepted ${sj.accepted}/${records.length}`, JSON.stringify(sj));
  const ins = (await api("/insights?days=30")).json;
  check(ins.daily.filter((d) => d.about_to_act > 0).length >= 7, "insights: daily series has 7+ days", JSON.stringify(ins.daily.slice(-9)));
  check(ins.recent_loads.length > 0 && ins.skills.length >= 2, "insights: recent loads + per-skill");
  check(ins.retention.chart_days === 30 && ins.retention.log_days === 7, "free retention 30/7", JSON.stringify(ins.retention));
  await page.goto(`${SITE}/portal/insights`);
  await page.getByText(/Recent|recent/).first().waitFor({ timeout: 20000 });
  await page.waitForTimeout(1200);
  const it = await text(page);
  check(!it.includes("The chart shows after two days"), "chart is drawn (not the empty message)");
  check(it.includes("humanizer") && it.includes("git"), "per-skill rows present");
  await shot(page, "21-insights");
  const ins90 = await api("/insights?days=90");
  check(ins90.status === 200 && ins90.json.daily.length <= 31, "free plan can't see 90 days (clamped)", `days=${ins90.json?.daily?.length} status=${ins90.status}`);

  step("Suggestions: Claude often runs docker build, no skill covers it");
  setSession("s3");
  for (let i = 0; i < 5; i++) gate("PostToolUse", "Bash", { command: `docker build -t app:${i} .` });
  stop();
  const sug = (await api("/suggestions")).json.suggestions;
  const dock = sug.find((s) => s.skill_name === "docker");
  check(!!dock, "a docker suggestion appears", JSON.stringify(sug));
  await page.goto(`${SITE}/portal`);
  await page.getByText("Suggestions").first().waitFor({ timeout: 20000 });
  await page.waitForTimeout(800);
  const ht = await text(page);
  check(/docker build/.test(ht), "home shows the docker suggestion", ht.slice(ht.indexOf("Suggestions"), ht.indexOf("Suggestions") + 400));
  await shot(page, "22-suggestion");
  const card = page.locator("li, div").filter({ hasText: /docker build/ }).filter({ has: page.getByRole("button", { name: "Dismiss" }) }).last();
  await card.getByRole("button", { name: "Dismiss" }).click();
  await page.waitForTimeout(1500);
  check(!((await api("/suggestions")).json.suggestions.some((s) => s.skill_name === "docker")), "dismissed suggestion stays gone");

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
