// Feature: manage skills (run after onboard): manage skills in the browser and watch the laptop follow.
// Add a catalog skill as a short rule, edit actions, recent runs, turn off + undo, remove.
import { join } from "node:path";
import { HOME, PAWS_HOME, SITE, check, step, done, gate, bash, stop, transcriptWith, api, ls, read, browser, shot, text } from "./lib.mjs";

const portalDir = join(PAWS_HOME, "portal");
const ruleFiles = (name) => ls(portalDir).filter((f) => f.startsWith(name));
const skillId = async (name) => (await api("/skills")).json.skills.find((s) => s.name === name)?.id;
const both = transcriptWith(["git", "humanizer"]);

const { b, page, errors } = await browser();
try {
  step("Add secret-scan from the catalog, as a short rule");
  await page.goto(`${SITE}/portal/skills/new`);
  await page.getByPlaceholder("Search skills, rules, tools").fill("secret-scan");
  await page.getByRole("link", { name: "Add secret-scan" }).click();
  await page.getByText("Add secret-scan").first().waitFor({ timeout: 20000 });
  await page.getByText("Full skill").waitFor();
  await page.locator("label", { hasText: "about 40 tokens" }).click();
  const ruleBox = page.getByLabel("Rule text");
  await ruleBox.fill("Run gitleaks on staged files before committing. Never commit .env files.");
  await shot(page, "10-add-rule");
  await page.getByRole("button", { name: "Turn on secret-scan" }).click();
  await page.waitForURL(/saved=secret-scan/, { timeout: 20000 });
  await page.getByText("secret-scan is saved.").waitFor({ timeout: 20000 });
  check(true, "saved; home shows 'secret-scan is saved. Waiting for sync'");
  await shot(page, "11-home-added");
  const ss = (await api("/skills")).json.skills.find((s) => s.name === "secret-scan");
  check(ss?.load === "rule" && ss?.state === "waiting_for_sync", "API: load=rule, waiting for sync", JSON.stringify(ss));

  step("Laptop pulls it: a rule file, no SKILL.md install");
  stop();
  check(ruleFiles("secret-scan").length > 0, "secret-scan rule file written", ls(portalDir).join(", "));
  check(!ls(join(HOME, ".claude/skills")).includes("secret-scan"), "no SKILL.md installed for a rule");
  const rd = bash('git commit -m "x"', "PreToolUse", both);
  const reason = rd.hookSpecificOutput?.permissionDecisionReason || rd.hookSpecificOutput?.additionalContext || JSON.stringify(rd);
  check(/gitleaks/.test(reason), "git commit gets the rule text", reason);
  check((await api("/skills")).json.skills.find((s) => s.name === "secret-scan")?.state !== "waiting_for_sync", "no longer waiting after the laptop reports its hash");

  step("Edit git's actions: add npm publish");
  const gitId = await skillId("git");
  await page.goto(`${SITE}/portal/skills/${gitId}`);
  await page.getByText("Edit git").first().waitFor({ timeout: 20000 });
  await page.getByPlaceholder("Add a command, file pattern or MCP tool").fill("npm publish");
  await page.getByRole("button", { name: "Add", exact: true }).click();
  await page.getByText("npm publish").first().waitFor();
  await shot(page, "12-edit");
  const save = page.getByRole("button", { name: /^Save/ });
  await save.click();
  await page.waitForURL(/\/portal(\?|$)/, { timeout: 20000 });
  const gitRow = (await api("/skills")).json.skills.find((s) => s.name === "git");
  check(gitRow.actions.some((a) => /npm publish/.test(a.label || "") || /npm/.test(a.command_regex || "")), "API: git has an npm publish action", JSON.stringify(gitRow.actions.map((a) => a.label)));
  stop();
  const np = bash("npm publish");
  check(np.hookSpecificOutput?.permissionDecision === "deny" && /git/.test(np.hookSpecificOutput.permissionDecisionReason), "laptop now stops npm publish for git", JSON.stringify(np));
  const npOk = bash("npm publish", "PreToolUse", transcriptWith(["git"], "t-git.jsonl"));
  check(!npOk.hookSpecificOutput?.permissionDecision, "allowed once git is read");

  step("Selected skill: recent runs");
  gate("PostToolUse", "Bash", { command: "git push origin main" });
  stop();
  await page.goto(`${SITE}/portal?selected=${gitId}`);
  await page.getByText("Recent runs").waitFor({ timeout: 20000 });
  await page.getByText("paws gave it").first().waitFor({ timeout: 20000 });
  const panel = await text(page);
  check(/git commit -m/.test(panel) || /npm publish/.test(panel), "recent runs list the commands", panel.slice(panel.indexOf("Recent runs"), panel.indexOf("Recent runs") + 400));
  await shot(page, "13-selected");

  step("Turn off humanizer, then Undo");
  const humId = await skillId("humanizer");
  await page.goto(`${SITE}/portal?selected=${humId}`);
  await page.getByRole("button", { name: "Turn off" }).click();
  await page.getByText("humanizer is off.").waitFor({ timeout: 20000 });
  await shot(page, "14-off");
  stop();
  check(ruleFiles("humanizer").length === 0, "laptop removed humanizer's rules", ls(portalDir).join(", "));
  const pr = bash("gh pr create --title t --body b", "PreToolUse", transcriptWith(["git"], "t-git.jsonl"));
  check(!pr.hookSpecificOutput?.permissionDecision, "gh pr create no longer stopped", JSON.stringify(pr));
  await page.getByRole("button", { name: "Undo" }).click();
  await page.getByText("humanizer is off.").waitFor({ state: "detached", timeout: 20000 });
  stop();
  check(ruleFiles("humanizer").length > 0, "Undo: humanizer's rules are back on the laptop");
  const pr2 = bash("gh pr create --title t --body b", "PreToolUse", transcriptWith(["git"], "t-git.jsonl"));
  check(pr2.hookSpecificOutput?.permissionDecision === "deny", "gh pr create stopped again");

  step("Remove secret-scan from its edit page");
  await page.goto(`${SITE}/portal/skills/${ss.id}`);
  await page.getByRole("button", { name: "Remove", exact: true }).click();
  await page.getByRole("button", { name: "Remove secret-scan" }).click();
  await page.waitForURL(/\/portal(\?|$)/, { timeout: 20000 });
  check(!(await api("/skills")).json.skills.some((s) => s.name === "secret-scan"), "API: secret-scan gone");
  stop();
  check(ruleFiles("secret-scan").length === 0, "laptop removed secret-scan's rule");
  const after = bash('git commit -m "x"', "PreToolUse", both);
  check(!after.hookSpecificOutput?.permissionDecision, "git commit runs (git + humanizer read, no rule left)", JSON.stringify(after));

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
