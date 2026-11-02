// Feature: billing + Pro gating + scheduled jobs + catalog + auth refusals (independent of the others).
// Was phase 5: billing page without Stripe, scheduled jobs (purge, weekly email), weekly email preview, catalog,
// Pro gating, DELETE /account.
import { SITE, API, USER, check, step, done, api, psql, manage, browser, shot, text } from "./lib.mjs";


const { b, page, errors } = await browser();
try {
  step("Billing page with Stripe not set up");
  await page.goto(`${SITE}/portal/settings/billing`);
  await page.getByText("Billing").first().waitFor({ timeout: 20000 });
  await page.waitForTimeout(1200);
  const bt = await text(page);
  console.log("    billing page says:", bt.slice(bt.indexOf("Billing"), bt.indexOf("Billing") + 300).replace(/\n+/g, " | "));
  await shot(page, "40-billing");
  const co = await api("/billing/checkout", { method: "POST" });
  check(co.status === 503, "checkout → 503 not configured", JSON.stringify(co));
  const wh = await fetch(`${API}/paws/v1/billing/webhook`, { method: "POST", body: "{}", headers: { "Stripe-Signature": "t=1,v1=bad" } });
  check(wh.status === 503 || wh.status === 400, `unsigned webhook rejected (${wh.status})`);

  step("Pro gating: 90 days locked on Free, open on Pro");
  await page.goto(`${SITE}/portal`);
  await page.getByText("times paws gave Claude a skill").waitFor({ timeout: 20000 }).catch(() => {});
  psql(`update paws_portal_pawsaccount set subscription_status='active', plan='pro' where email like '${USER}@%'`);
  const me = (await api("/me")).json;
  check(me.plan === "pro" && me.retention.chart_days === 90 && me.retention.log_days === 30, "active subscription → Pro 90/30", JSON.stringify(me));
  await page.goto(`${SITE}/portal?days=90`);
  await page.waitForTimeout(1500);
  await shot(page, "41-pro-home");
  await page.goto(`${SITE}/portal/settings/billing`);
  await page.waitForTimeout(1200);
  await shot(page, "42-billing-pro");
  console.log("    billing page (pro):", (await text(page)).match(/Billing[\s\S]{0,300}/)?.[0].replace(/\n+/g, " | "));
  psql(`update paws_portal_pawsaccount set subscription_status='', plan='free' where email like '${USER}@%'`);

  step("Purge job drops records past retention");
  const acct = psql(`select id from paws_portal_pawsaccount where email like '${USER}@%' limit 1`);
  psql(`insert into paws_portal_pawsevent (account_id, device_id, ts, event, skill, verdict, already_loaded, command, tool) select ${acct}, (select id from paws_portal_pawsdevice where account_id=${acct} limit 1), now() - interval '200 days', 'pre-tool', 'git', 'deny', false, 'old', 'Bash'`);
  const oldBefore = Number(psql(`select count(*) from paws_portal_pawsevent where account_id=${acct} and ts < now() - interval '100 days'`));
  const out = manage("purge_paws_events");
  const oldAfter = Number(psql(`select count(*) from paws_portal_pawsevent where account_id=${acct} and ts < now() - interval '100 days'`));
  check(oldBefore === 1 && oldAfter === 0, `old record purged (${oldBefore} → ${oldAfter})`, out.slice(-300));

  step("Weekly email job: no Resend key → logs and skips; preview page renders");
  const we = manage("send_weekly_email");
  console.log("    ", we.trim().split("\n").slice(-2).join(" | "));
  check(/skip|RESEND|not set|0 sent|sent/i.test(we), "weekly email job ran", we.slice(-300));
  await page.goto(`${SITE}/email/weekly`);
  await page.waitForTimeout(1500);
  await shot(page, "43-weekly-email");
  check((await text(page)).length > 100, "weekly email preview renders");

  step("Public catalog page");
  await page.goto(`${SITE}/catalog`);
  await page.waitForTimeout(1500);
  const ct = await text(page);
  check(ct.includes("humanizer") && ct.includes("secret-scan"), "catalog lists skills");
  await shot(page, "44-catalog");
  const cat = await api("/catalog", { token: null });
  check(cat.status === 200 && cat.json.skills.length >= 40, `API /catalog without auth: ${cat.json?.skills?.length} skills`);

  step("Auth: no token / bad token are refused");
  check((await api("/skills", { token: null })).status === 401, "no token → 401");
  check((await api("/config", { token: "pdt_nope" })).status === 401, "bad device token → 401");

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
