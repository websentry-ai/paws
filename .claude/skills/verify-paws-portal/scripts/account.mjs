// Feature: account. Run after onboard (ends with every laptop disconnected). Weekly email toggle, more laptops (key from Settings, browser device flow with
// Cancel and Connect), delete synced data, disconnect all → laptops log out and drop portal rules.
import { spawn } from "node:child_process";
import { join } from "node:path";
import { E, HOME, PAWS_HOME, PAWS_BIN, PROJ, API, SITE, USER, laptopEnv, check, step, done, paws, stop, api, ls, read, browser, shot, text } from "./lib.mjs";

const lap2 = { PAWS_HOME: join(HOME, ".paws-lap2") };
const lap3 = { PAWS_HOME: join(HOME, ".paws-lap3") };

/** Starts `paws login` (device flow) and resolves with {code, wait()} once it prints the code. */
function deviceLogin(extra) {
  const p = spawn(PAWS_BIN, ["login"], { cwd: PROJ, env: laptopEnv(extra) });
  let out = "";
  const exited = new Promise((res) => p.on("exit", (code) => res(code)));
  return new Promise((resolve) => {
    const on = (d) => {
      out += d;
      const m = out.match(/confirm code ([A-Z0-9-]+)/);
      if (m) resolve({ code: m[1], url: out.match(/Open (\S+)/)[1], wait: async () => ({ code: await exited, out }) });
    };
    p.stdout.on("data", on);
    p.stderr.on("data", on);
  });
}

const { b, page, errors } = await browser();
try {
  step("Weekly email: off, persists, back on");
  await page.goto(`${SITE}/portal/settings`);
  const box = page.locator("label", { hasText: "Weekly email" }).locator("input[type=checkbox]");
  await box.waitFor({ timeout: 20000 });
  const was = await box.isChecked();
  await box.click();
  await page.waitForTimeout(1000);
  check((await api("/me")).json.weekly_email === !was, `weekly_email → ${!was}`);
  await page.reload();
  await box.waitFor();
  check((await box.isChecked()) === !was, "toggle persists after reload");
  await box.click();
  await page.waitForTimeout(1000);
  check((await api("/me")).json.weekly_email === was, "toggled back");

  step("Second laptop with a key from Settings");
  await page.getByRole("button", { name: "New one-time key" }).click();
  await page.getByText(/paws login --key pk_/).waitFor({ timeout: 20000 });
  const key = (await text(page)).match(/paws login --key (pk_[A-Za-z0-9_-]+)/)[1];
  const l2 = paws(["login", "--key", key], { extra: lap2 });
  check(l2.code === 0 && /Logged in as/.test(l2.out), "laptop 2 logged in", l2.out);
  const s2 = paws(["sync"], { extra: lap2 });
  check(/git/.test(s2.out), "laptop 2 pulled the account's skills", s2.out);

  step("Third laptop: browser device flow, Cancel first");
  const d1 = await deviceLogin(lap3);
  check(d1.url.startsWith(SITE + "/connect"), "CLI points at the website's /connect", d1.url);
  for (let i = 0; i < 20 && !read(join(E, "opened.txt")).includes(d1.code); i++) await new Promise((r) => setTimeout(r, 100));
  check(read(join(E, "opened.txt")).includes(d1.code), "CLI tried to open the browser with the code (fake `open` recorded it)");
  await page.goto(d1.url);
  await page.getByText("Connect this laptop?").waitFor({ timeout: 20000 });
  check((await text(page)).includes(d1.code), "confirm card shows the same code");
  await shot(page, "30-connect-confirm");
  await page.getByRole("button", { name: "Cancel" }).first().click();
  const r1 = await d1.wait();
  check(r1.code !== 0, "Cancel: CLI stops without logging in", r1.out);
  check(!ls(lap3.PAWS_HOME).includes("credentials.json"), "no credentials written");

  step("Third laptop: Connect");
  const d2 = await deviceLogin(lap3);
  await page.goto(d2.url);
  await page.getByRole("button", { name: "Connect", exact: true }).click();
  const r2 = await d2.wait();
  check(r2.code === 0 && new RegExp(`Logged in as ${USER}@`).test(r2.out), "Connect: CLI logged in", r2.out);
  await page.waitForTimeout(500);
  await shot(page, "31-connected");
  const replay = await fetch(`${API}/paws/v1/device/code/${d2.code}`, { headers: { Authorization: `Bearer ${USER}` } });
  console.log("    reusing an approved code →", replay.status);

  step("Delete synced data");
  await page.goto(`${SITE}/portal/settings`);
  await page.getByRole("button", { name: "Delete synced data" }).click();
  await page.getByRole("button", { name: "Delete synced data" }).last().click();
  await page.getByText("Synced data deleted.").waitFor({ timeout: 20000 });
  const sk = (await api("/skills")).json;
  check(sk.headline.about_to_act === 0 && sk.skills.length >= 2, "counts zeroed, skills kept", JSON.stringify(sk.headline));
  check((await api("/insights")).json.recent_loads.length === 0, "insights emptied");

  step("Disconnect all: every laptop logs out on its next sync and drops portal rules");
  await page.getByRole("button", { name: "Disconnect all" }).click();
  await page.getByRole("button", { name: "Disconnect all" }).last().click();
  await page.waitForTimeout(1500);
  check((await api("/skills")).json.connected === false, "API: not connected");
  stop(); // laptop 1: next Stop gets 401
  check(!ls(PAWS_HOME).includes("credentials.json"), "laptop 1 forgot its login", ls(PAWS_HOME).join(","));
  check(ls(join(PAWS_HOME, "portal")).filter((f) => f.endsWith(".md")).length === 0, "laptop 1 removed portal rules", ls(join(PAWS_HOME, "portal")).join(","));
  const s3 = paws(["sync"], { extra: lap2 });
  check(s3.code !== 0 || /log ?in|revoked|Not logged/i.test(s3.out), "laptop 2: sync says log in again", s3.out);
  await page.reload();
  await page.waitForTimeout(1000);
  await shot(page, "32-disconnected");

  check(errors.length === 0, "no browser errors", errors.join("\n"));
} finally {
  await b.close();
}
done();
