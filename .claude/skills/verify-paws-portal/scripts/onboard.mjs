// Feature: onboarding, the first feature to run (it resets the account and the scratch laptop). A new user. Setup page → one-time key → paws login --key → first sync → picks from what Claude ran
// → turn on → the laptop pulls them on the next Stop → the gate stops git commit until Claude reads the skill.
import { rmSync, mkdirSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";
import { join } from "node:path";
import { HOME, PAWS_HOME, PROJ, SITE, check, step, done, paws, gate, bash, stop, transcriptWith, api, ls, read, browser, shot, text } from "./lib.mjs";

await api("/account", { method: "DELETE" }); // start from a brand-new account
rmSync(HOME, { recursive: true, force: true });
mkdirSync(PROJ, { recursive: true });
execSync("git init -q && git commit -q --allow-empty -m init", { cwd: PROJ, env: { ...process.env, HOME, GIT_AUTHOR_NAME: "e", GIT_AUTHOR_EMAIL: "e@x", GIT_COMMITTER_NAME: "e", GIT_COMMITTER_EMAIL: "e@x" } });
// A skill the user already has on this laptop: onboarding should show it as "on from your laptop".
mkdirSync(join(HOME, ".claude/skills/team-notes"), { recursive: true });
writeFileSync(join(HOME, ".claude/skills/team-notes/SKILL.md"), "---\nname: team-notes\ndescription: Our team notes.\n---\nWrite notes.\n");

step("Laptop history before login: what Claude ran (PostToolUse action records)");
for (let i = 0; i < 12; i++) gate("PostToolUse", "Bash", { command: `git commit -m "change ${i}"` });
for (let i = 0; i < 4; i++) gate("PostToolUse", "Bash", { command: `gh pr create --title t${i} --body b` });
gate("PostToolUse", "Bash", { command: "curl -H 'Authorization: Bearer sk-live-abcdef0123456789abcdef' https://api.example.com/x?token=supersecret123" });
gate("PostToolUse", "Edit", { file_path: join(PROJ, "db/migrations/0002_add.sql"), old_string: "a", new_string: "b" });
check(ls(PAWS_HOME).length > 0, "paws wrote local records", ls(PAWS_HOME).join(","));

const { b, page, errors } = await browser();
try {
  step("New user opens the portal → setup");
  await page.goto(`${SITE}/portal`);
  await page.waitForURL(/\/portal\/setup/, { timeout: 30000 }).catch(() => {});
  check(page.url().includes("/portal/setup"), "first visit goes to setup", page.url());
  await page.getByText(/paws login --key pk_/).first().waitFor({ timeout: 30000 });
  const body = await text(page);
  check(body.includes("Waiting for paws to connect"), "waits for paws to connect");
  const key = body.match(/paws login --key (pk_[A-Za-z0-9_-]+)/)?.[1];
  check(!!key, "shows a one-time key command", body.slice(0, 400));
  await shot(page, "01-setup-connect");

  step("Laptop: paws login --key <one-time key>");
  const login = paws(["login", "--key", key]);
  check(login.code === 0 && /Logged in as/.test(login.out), "login with key", login.out);
  const reuse = paws(["login", "--key", key], { extra: { PAWS_HOME: join(HOME, ".paws-other") } });
  check(reuse.code !== 0, "the same key cannot be used twice", reuse.out);

  step("Setup page sees the connection; paws sync sends the first upload");
  await page.getByText("Connected.").first().waitFor({ timeout: 20000 });
  const sync = paws(["sync"]);
  check(sync.code === 0, "paws sync", sync.out);
  await page.getByText(/Found 1 skill|No skills on this laptop/).waitFor({ timeout: 20000 });
  check((await text(page)).includes("Found 1 skill"), "found the laptop's skill (team-notes)");
  await shot(page, "02-setup-connected");

  step("Pick page: picks from what Claude ran");
  await page.getByRole("link", { name: "Next" }).click();
  await page.getByText("Pick your house rules").waitFor({ timeout: 20000 });
  await page.getByText(/Picked from what Claude ran|Good first picks/).waitFor();
  const picks = await text(page);
  check(picks.includes("Picked from what Claude ran"), "picks come from real counts", picks.slice(0, 1500));
  check(/git commit/i.test(picks), "a pick mentions git commit");
  check(picks.includes("team-notes"), "laptop skill shown as already on");
  await shot(page, "03-pick");
  const btn = page.getByRole("button", { name: /Turn on \d+ skills?/ });
  const label = await btn.innerText();
  const n = Number(label.match(/\d+/)[0]);
  await btn.click();
  await page.getByText(/skills? saved/).waitFor({ timeout: 20000 });
  check((await text(page)).includes("Waiting for sync"), `saved ${n} skills, waiting for sync`);
  await shot(page, "04-saved");

  step("Skills home before the laptop syncs");
  await page.getByRole("link", { name: "See my skills" }).click();
  await page.waitForURL(/\/portal(\?|$)/);
  await page.getByText(/Waiting for sync|waiting/i).first().waitFor({ timeout: 20000 }).catch(() => {});
  const home1 = await text(page);
  check(/waiting for sync/i.test(home1), "home shows skills waiting for sync", home1.slice(0, 1200));
  await shot(page, "05-home-waiting");

  step("Laptop: next Stop hook pulls the config");
  const skillsApi = (await api("/skills")).json;
  const names = skillsApi.skills.map((s) => s.name);
  console.log("    on in portal:", names.join(", "));
  stop();
  const portalFiles = ls(join(PAWS_HOME, "portal"));
  check(portalFiles.length > 0, "managed rule files written", portalFiles.join(", "));
  for (const s of skillsApi.skills.filter((s) => s.kind === "skill" && s.load === "full"))
    check(ls(join(HOME, ".claude/skills")).includes(s.name), `skill ${s.name} installed in ~/.claude/skills`);
  const st = JSON.parse(read(join(PAWS_HOME, "portal", "state.json")) || read(join(PAWS_HOME, "portal-state.json")) || "{}");
  console.log("    state:", JSON.stringify(st).slice(0, 200));

  step("Gate: git commit is stopped until Claude reads the skill");
  const gitSkill = skillsApi.skills.find((s) => s.name === "git");
  check(!!gitSkill, "a picked skill runs before git commit", JSON.stringify(skillsApi.skills.map((s) => [s.name, s.actions.map((a) => a.label)])));
  if (gitSkill) {
    const deny = bash('git commit -m "real change"');
    const reason = deny.hookSpecificOutput?.permissionDecisionReason || "";
    check(deny.hookSpecificOutput?.permissionDecision === "deny", "first git commit is stopped", JSON.stringify(deny));
    check(reason.includes(gitSkill.name), `reason names ${gitSkill.name}`, reason);
    // humanizer also runs before a commit message (pre-commit), so Claude has read both here.
    const allow = bash('git commit -m "real change"', "PreToolUse", transcriptWith(["git", "humanizer"]));
    check(!allow.hookSpecificOutput?.permissionDecision, "after Claude reads it, git commit runs", JSON.stringify(allow));
    gate("PostToolUse", "Bash", { command: 'git commit -m "real change"' });
  }

  step("Stop uploads; home shows counts and no longer waits");
  stop();
  const home = (await api("/skills")).json;
  const gs = home.skills.find((s) => s.name === gitSkill?.name);
  console.log("    headline:", JSON.stringify(home.headline), "| git skill:", JSON.stringify({ state: gs?.state, times: gs?.times, paws_gave: gs?.paws_gave, claude_read: gs?.claude_read }));
  check(gs && gs.state === "ok", "git skill state ok after sync", JSON.stringify(gs));
  check(home.headline.paws_gave >= 1, "headline counts paws giving the skill");
  await page.reload();
  await page.getByText(gitSkill?.name || "zzz").first().waitFor({ timeout: 20000 });
  await page.waitForTimeout(800);
  await shot(page, "06-home-synced");
  check(!/waiting for sync/i.test(await text(page)), "home no longer says waiting for sync", (await text(page)).slice(0, 800));

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
