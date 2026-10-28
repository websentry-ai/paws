import { execFileSync } from "node:child_process";
import { appendFileSync, readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import semver from "semver";

export function releasePlan(version, publishedVersions, releaseVersions = []) {
  const parsed = semver.parse(version);
  const canonical = parsed && parsed.version + (parsed.build.length ? `+${parsed.build.join(".")}` : "");
  if (!parsed || version !== canonical) {
    throw new Error(`Invalid package version: ${version}`);
  }
  const published = semver.rsort([...publishedVersions])[0];
  if (published && !semver.gt(version, published)) {
    throw new Error(`Version ${version} must be newer than npm version ${published}. Bump npm/package.json.`);
  }
  const released = semver.rsort([...releaseVersions])[0];
  if (released && semver.lt(version, released)) {
    throw new Error(`Version ${version} is older than GitHub release ${released}. Bump npm/package.json.`);
  }
  return { version, "npm-tag": parsed.prerelease.length ? "next" : "latest" };
}

export async function publishedVersions(name, request = fetch) {
  const response = await request(`https://registry.npmjs.org/${encodeURIComponent(name)}`, {
    headers: { Accept: "application/vnd.npm.install-v1+json" },
    signal: AbortSignal.timeout(30_000),
  });
  if (response.status === 404) return [];
  if (!response.ok) throw new Error(`npm registry returned HTTP ${response.status}.`);
  const metadata = await response.json();
  if (!metadata.versions || typeof metadata.versions !== "object" || Array.isArray(metadata.versions)) {
    throw new Error("npm registry response has no versions map.");
  }
  return Object.keys(metadata.versions);
}

async function main() {
  const pkg = JSON.parse(readFileSync(new URL("../../npm/package.json", import.meta.url), "utf8"));
  const tags = execFileSync("git", ["tag", "--list", "v*"], { encoding: "utf8" })
    .trim().split(/\r?\n/).map((tag) => tag.slice(1)).filter((tag) => semver.valid(tag));
  const plan = releasePlan(pkg.version, await publishedVersions(pkg.name), tags);
  const output = Object.entries(plan).map(([key, value]) => `${key}=${value}`).join("\n") + "\n";
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, output);
  console.log(`Release check passed: ${pkg.name}@${plan.version}, npm tag ${plan["npm-tag"]}.`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
