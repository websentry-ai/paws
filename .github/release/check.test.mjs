import assert from "node:assert/strict";
import test from "node:test";
import { publishedVersions, releasePlan } from "./check.mjs";

for (const [version, published, releases, tag] of [
  ["0.2.0", [], ["0.2.0-rc.3"], "latest"],
  ["0.10.0", ["0.9.0"], [], "latest"],
  ["0.2.0-rc.10", ["0.2.0-rc.9"], [], "next"],
  ["0.2.0", ["0.2.0-rc.3"], [], "latest"],
  ["0.2.0", [], ["0.2.0"], "latest"],
]) {
  test(`${version} can publish after npm ${published} and tags ${releases}`, () => {
    assert.deepEqual(releasePlan(version, published, releases), { version, "npm-tag": tag });
  });
}

for (const [version, published, releases] of [
  ["0.2.0", ["0.2.0"], []],
  ["0.1.0", ["0.2.0"], []],
  ["0.2.0-rc.3", ["0.2.0"], []],
  ["0.2.0-rc.9", ["0.2.0-rc.10"], []],
  ["0.2.0", ["0.3.0", "0.1.0"], []],
  ["0.2.0+rebuild", ["0.2.0"], []],
  ["0.1.0", [], ["0.2.0-rc.3"]],
]) {
  test(`${version} cannot publish after npm ${published} and tags ${releases}`, () => {
    assert.throws(() => releasePlan(version, published, releases), /Bump npm\/package.json/);
  });
}

for (const version of ["v0.2.0", "0.2", "garbage", null]) {
  test(`rejects invalid package version ${version}`, () => {
    assert.throws(() => releasePlan(version, []), /Invalid package version/);
  });
}

test("a missing npm package permits the first release", async () => {
  assert.deepEqual(await publishedVersions("unbound-paws", async () => new Response(null, { status: 404 })), []);
});

test("registry outages fail the check instead of permitting a release", async () => {
  await assert.rejects(publishedVersions("unbound-paws", async () => new Response(null, { status: 503 })), /HTTP 503/);
});

test("malformed registry metadata fails the check", async () => {
  await assert.rejects(publishedVersions("unbound-paws", async () => Response.json({})), /no versions map/);
});

test("reads every published version, including prereleases", async () => {
  const request = async () => Response.json({ versions: { "0.1.0": {}, "0.2.0-rc.3": {} } });
  assert.deepEqual(await publishedVersions("unbound-paws", request), ["0.1.0", "0.2.0-rc.3"]);
});
