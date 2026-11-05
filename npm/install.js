#!/usr/bin/env node
// Downloads the paws release binary for this platform into vendor/, verified against
// the release's checksums.txt. Runs as postinstall; bin/paws.js calls it again only if
// the binary is missing (e.g. npm was run with --ignore-scripts). No network after that.
//
// Asset names must match .goreleaser.yaml: paws_<version>_<os>_<arch>.tar.gz (.zip on Windows).
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");

const pkg = require("./package.json");
const VENDOR = path.join(__dirname, "vendor");
const EXE = process.platform === "win32" ? "paws.exe" : "paws";
const BINARY = path.join(VENDOR, EXE);
// PAWS_DOWNLOAD_BASE points at a mirror or a local test server holding the same assets.
const BASE = process.env.PAWS_DOWNLOAD_BASE || `https://github.com/websentry-ai/paws/releases/download/v${pkg.version}`;

function target() {
  const goos = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
  const goarch = { x64: "amd64", arm64: "arm64" }[process.arch];
  if (!goos || !goarch) {
    throw new Error(`paws has no build for ${process.platform}/${process.arch}`);
  }
  const ext = goos === "windows" ? "zip" : "tar.gz";
  return `paws_${pkg.version}_${goos}_${goarch}.${ext}`;
}

// Redirects may not downgrade to http: the archive and checksums.txt come from the same place,
// so a downgraded pair would verify against itself. Plain http is only for a local test BASE.
const ALLOW_HTTP = BASE.startsWith("http://");

function fetch(url, redirects = 5) {
  return new Promise((resolve, reject) => {
    if (url.startsWith("file://")) {
      fs.readFile(new URL(url), (err, data) => (err ? reject(err) : resolve(data)));
      return;
    }
    const client = url.startsWith("http://") ? require("http") : https;
    client
      .get(url, { headers: { "User-Agent": "unbound-paws-installer" } }, (res) => {
        if ([301, 302, 303, 307, 308].includes(res.statusCode) && res.headers.location && redirects > 0) {
          res.resume();
          const next = new URL(res.headers.location, url).toString();
          if (next.startsWith("http://") && !ALLOW_HTTP) {
            reject(new Error(`refusing redirect from ${url} to insecure ${next}`));
            return;
          }
          resolve(fetch(next, redirects - 1));
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`GET ${url} -> HTTP ${res.statusCode}`));
          return;
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function ensureBinary() {
  if (fs.existsSync(BINARY)) {
    return BINARY;
  }
  const asset = target();
  const [archive, sums] = await Promise.all([fetch(`${BASE}/${asset}`), fetch(`${BASE}/checksums.txt`)]);

  const line = sums.toString("utf8").split("\n").find((l) => l.trim().endsWith(`  ${asset}`));
  if (!line) {
    throw new Error(`checksums.txt has no entry for ${asset}`);
  }
  const want = line.trim().split(/\s+/)[0];
  const got = crypto.createHash("sha256").update(archive).digest("hex");
  if (got !== want) {
    throw new Error(`checksum mismatch for ${asset}: want ${want}, got ${got}`);
  }

  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "paws-"));
  try {
    const file = path.join(tmp, asset);
    fs.writeFileSync(file, archive);
    // bsdtar (macOS, Windows 10+) and GNU tar both read .tar.gz; only bsdtar reads .zip. On Windows,
    // Git Bash puts GNU tar first on PATH, so call the system bsdtar by its full path.
    const tar = process.platform === "win32" ? path.join(process.env.SystemRoot || "C:\\Windows", "System32", "tar.exe") : "tar";
    execFileSync(tar, ["-xf", file, "-C", tmp, EXE], { stdio: "ignore" });
    fs.mkdirSync(VENDOR, { recursive: true });
    const staged = path.join(VENDOR, `.${EXE}.${process.pid}`);
    fs.copyFileSync(path.join(tmp, EXE), staged);
    fs.chmodSync(staged, 0o755);
    fs.renameSync(staged, BINARY);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
  return BINARY;
}

// newer reports whether version a (x.y.z[-pre]) is newer than b, semver style: a release is newer
// than its own prereleases (0.4.0 > 0.4.0-rc.1).
function newer(a, b) {
  const [ca, prea = ""] = a.split(/-(.*)/);
  const [cb, preb = ""] = b.split(/-(.*)/);
  const pa = ca.split(".").map(Number);
  const pb = cb.split(".").map(Number);
  for (let i = 0; i < 3; i++) {
    if (pa[i] !== pb[i]) return pa[i] > pb[i];
  }
  if (prea === preb) return false;
  if (!prea || !preb) return !prea;
  // Semver: compare dot-separated identifiers; numbers numerically, and a number ranks below letters.
  const ia = prea.split(".");
  const ib = preb.split(".");
  for (let i = 0; i < Math.min(ia.length, ib.length); i++) {
    if (ia[i] === ib[i]) continue;
    const na = /^\d+$/.test(ia[i]);
    const nb = /^\d+$/.test(ib[i]);
    if (na && nb) return Number(ia[i]) > Number(ib[i]);
    if (na !== nb) return nb;
    return ia[i] > ib[i];
  }
  return ia.length > ib.length;
}

// refreshHookCopy upgrades the copy `paws init` made for the hook (<paws home>/bin), if there is one,
// so `npm i -g unbound-paws@latest` is the whole upgrade. Every repo's hook runs that one copy, so it
// is only replaced by a newer release (or when it no longer runs), never by an older pinned or npx'd
// one, and a local build (`paws dev`) is left alone. Same rename-into-place as init.
function refreshHookCopy(bin) {
  const home = process.env.PAWS_HOME && path.isAbsolute(process.env.PAWS_HOME) ? process.env.PAWS_HOME : path.join(os.homedir(), ".paws");
  const copy = path.join(home, "bin", EXE);
  // Only an unprivileged install by the copy's owner refreshes it: never root (`sudo npm i -g` would
  // run or write through a binary and folder the user controls), never Windows (no owner to check;
  // `npx unbound-paws@latest init` upgrades there), never through a symlink.
  if (process.platform === "win32" || process.getuid() === 0) return;
  const dir = path.dirname(copy);
  let st, dst;
  try {
    st = fs.lstatSync(copy);
    dst = fs.lstatSync(dir);
  } catch {
    return; // no copy yet
  }
  const uid = process.getuid();
  if (!st.isFile() || st.uid !== uid || !dst.isDirectory() || dst.uid !== uid || (dst.mode & 0o022) !== 0) return;
  let current = null;
  try {
    current = execFileSync(copy, ["version"], { encoding: "utf8", timeout: 5000 });
  } catch {
    // Doesn't run: replace it.
  }
  if (current !== null) {
    const m = /^paws (\d+\.\d+\.\d+(-\S+)?)$/.exec(current.trim());
    // A source build (0.2.0-dev, dev) is someone's own; leave it.
    if (!m || /dev/.test(m[1]) || !newer(pkg.version, m[1])) return;
  }
  const staged = path.join(dir, `.${EXE}.${process.pid}`);
  fs.copyFileSync(bin, staged, fs.constants.COPYFILE_EXCL); // never through a planted file or link
  fs.chmodSync(staged, 0o755);
  fs.renameSync(staged, copy);
  console.log(`updated the paws hook binary at ${copy}`);
}

// useNativeLauncher replaces bin/paws.js, the launcher npm links as `unbound-paws`, with the binary
// itself, so the command (and the Claude Code hook that runs it) doesn't start Node on every call.
// Windows keeps the Node launcher: npm's .cmd shim there always runs node.
function useNativeLauncher(bin) {
  if (process.platform === "win32") return;
  const launcher = path.join(__dirname, "bin", "paws.js");
  const staged = `${launcher}.${process.pid}`;
  fs.copyFileSync(bin, staged);
  fs.chmodSync(staged, 0o755);
  fs.renameSync(staged, launcher);
}

module.exports = { ensureBinary, BINARY };

if (require.main === module) {
  ensureBinary().then(
    (bin) => {
      console.log(`paws ${pkg.version} installed at ${bin}`);
      try {
        useNativeLauncher(bin);
      } catch (err) {
        // The Node launcher still works, just slower.
        console.warn(`unbound-paws: kept the Node launcher: ${err.message}`);
      }
      try {
        refreshHookCopy(bin);
      } catch (err) {
        console.warn(`unbound-paws: could not update the paws hook binary: ${err.message} (run \`unbound-paws init\` again)`);
      }
    },
    (err) => {
      // Never fail the npm install: bin/paws.js retries on first run and reports the error there.
      console.warn(`unbound-paws: could not download the paws binary yet: ${err.message}`);
    }
  );
}
