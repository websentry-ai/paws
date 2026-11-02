// Feature: skill updates (spec §10 drift). Run after onboard (humanizer on, laptop logged in).
// The daily check finds a real upstream change on GitHub, the user approves it in the browser, the
// laptop installs the new commit from the same repo and path, and a download that doesn't match the
// approved hash keeps the installed version and doesn't report the config as applied.
import { join } from "node:path";
import { HOME, PAWS_HOME, SITE, check, step, done, api, paws, psql, manage, read, skillRow, browser, shot } from "./lib.mjs";

// An older humanizer commit whose SKILL.md differs from the upstream head, so the check has a diff to show.
const OLD_REF = "1f89fd1813cf5560938f28f9ed154b8c69b2f9ea";
const marker = () => read(join(HOME, ".claude/skills/humanizer/.paws-skill.json"));
const state = () => JSON.parse(read(join(PAWS_HOME, "portal/state.json")) || "{}");

const { b, page, errors } = await browser();
try {
  step("Daily check: humanizer's upstream moved on (live GitHub)");
  check(!!(await skillRow("humanizer")), "humanizer is on (run onboard first)");
  psql(`update paws_portal_pawsskill set upstream = jsonb_set(upstream::jsonb, '{ref}', '"${OLD_REF}"')::json, version='${OLD_REF.slice(0, 7)}' where name='humanizer'`);
  const out = manage("check_skill_updates");
  check(/'created': 1/.test(out), "check_skill_updates created 1 update", out.slice(-300));
  const ups = (await api("/updates")).json.updates;
  check(ups.length === 1 && ups[0].diff.length > 50, "API lists the update with a diff", JSON.stringify(ups).slice(0, 300));
  const toRef = psql("select to_ref from paws_portal_pawsskillupdate order by id desc limit 1");

  step("Home shows the banner; Updates page shows the diff and the decision");
  await page.goto(`${SITE}/portal`);
  await page.getByRole("link", { name: "Review updates" }).first().waitFor({ timeout: 30000 });
  check(true, "home: 'Review updates' banner");
  await page.goto(`${SITE}/portal/updates`);
  const update = page.getByRole("button", { name: "Update humanizer" });
  await update.waitFor({ timeout: 30000 });
  check(await update.isVisible(), "Update humanizer button visible (long diffs scroll inside their box)");
  await shot(page, "updates-diff");
  await update.click();
  await page.getByText(/humanizer updates to/).waitFor({ timeout: 30000 });
  const hum = await skillRow("humanizer");
  check(hum.version === toRef.slice(0, 7) && !hum.update_available, `portal pins humanizer ${hum.version}`, JSON.stringify(hum));

  step("Laptop installs the approved commit from the same repo and path");
  const s = paws(["sync"]);
  console.log("    ", s.out.trim().split("\n").join(" | "));
  const approvedHash = psql("select content_hash from paws_portal_pawsskill where name='humanizer'");
  check(s.code === 0 && /humanizer/.test(s.out) && !/skipped humanizer/.test(s.out), "sync applied humanizer", s.out);
  check(marker().includes(`github.com/blader/humanizer@${toRef}`) && marker().includes(approvedHash), "installed from blader/humanizer at the approved commit and hash", marker());
  check((await skillRow("humanizer")).state !== "waiting_for_sync", "portal: not waiting (laptop reported the new config)");

  step("A config whose hash doesn't match the download keeps the installed version");
  const applied = state().applied_config_hash;
  psql(`update paws_portal_pawsskill set content_hash='sha256:${"d".repeat(64)}', updated_at=now() where name='humanizer'`);
  const before = marker();
  const bad = paws(["sync"]);
  check(bad.code !== 0 && /keeping the installed version/.test(bad.out), "sync reports the mismatch and keeps the installed version", bad.out);
  check(marker() === before, "installed copy unchanged");
  check(state().applied_config_hash === applied, "config not reported as applied");
  check((await skillRow("humanizer")).state === "waiting_for_sync", "portal: humanizer waiting for sync");
  psql(`update paws_portal_pawsskill set content_hash='${approvedHash}', updated_at=now() where name='humanizer'`);
  check(paws(["sync"]).code === 0, "approved hash restored: sync clean");

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
