#!/usr/bin/env node
// Runs the paws binary with this process's arguments and exit code.
"use strict";

const { spawnSync } = require("child_process");
const { ensureBinary } = require("../install.js");

ensureBinary().then(
  (bin) => {
    const res = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
    if (res.error) {
      console.error(`unbound-paws: ${res.error.message}`);
      process.exit(1);
    }
    process.exit(res.status === null ? 1 : res.status);
  },
  (err) => {
    console.error(`unbound-paws: could not install the paws binary: ${err.message}`);
    process.exit(1);
  }
);
