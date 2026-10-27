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
          resolve(fetch(new URL(res.headers.location, url).toString(), redirects - 1));
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
    // bsdtar (macOS, Windows 10+) and GNU tar both read .tar.gz; bsdtar also reads .zip.
    execFileSync("tar", ["-xf", file, "-C", tmp, EXE], { stdio: "ignore" });
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

module.exports = { ensureBinary, BINARY };

if (require.main === module) {
  ensureBinary().then(
    (bin) => console.log(`paws ${pkg.version} installed at ${bin}`),
    (err) => {
      // Never fail the npm install: bin/paws.js retries on first run and reports the error there.
      console.warn(`unbound-paws: could not download the paws binary yet: ${err.message}`);
    }
  );
}
