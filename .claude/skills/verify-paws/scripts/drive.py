#!/usr/bin/env python3
"""Drive a real headless Claude Code session against a freshly built paws.

    drive.py doctor
    drive.py list
    drive.py run <case> [<case> ...] [--model sonnet] [--keep]
    drive.py run --all

Each case is a directory under ../cases/ holding case.toml, an optional
project/ tree (copied into a scratch git repo, usually holding .paws/ rules),
and an optional skills/ tree (copied into the scratch repo's .claude/skills/,
kept outside project/ so Claude Code never discovers them inside this repo).

Isolation: the scratch repo gets its own .claude/settings.json from
`paws init`, Claude runs with --setting-sources project so the user's own
hooks and settings stay out, and PAWS_HOME points paws's user dir (user rules,
audit log, installed packs) at the scratch area. Claude's auth is untouched.

Evidence goes to <repo>/.verify-artifacts/<run-id>/<case>/ and survives
cleanup. Exit codes: 0 PASS, 1 FAIL, 2 INDETERMINATE (agent did not run).
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import tomllib
from pathlib import Path

SKILL_DIR = Path(__file__).resolve().parent.parent
REPO = SKILL_DIR.parent.parent.parent
CASES = SKILL_DIR / "cases"
ARTIFACTS = REPO / ".verify-artifacts"

PASS, FAIL, INDETERMINATE = 0, 1, 2


# Markers that tie a process to the running Claude Code session. Auth and provider
# variables (CLAUDE_CODE_USE_BEDROCK, CLAUDE_CODE_OAUTH_TOKEN, ...) are kept.
SESSION_MARKERS = {
    "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_SSE_PORT", "CLAUDE_CODE_EXECPATH",
    "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_CHILD_SESSION", "CLAUDE_CODE_SESSION_ATTENDED",
    "CLAUDE_CODE_BRIDGE_SESSION_ID", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN",
}

EXPECT_KEYS = {
    "skills_invoked", "skills_not_invoked", "tool_result_contains", "tool_result_absent",
    "audit", "audit_absent", "audit_max", "final_contains", "shell",
}


def agent_env(extra: dict[str, str]) -> dict[str, str]:
    # The child must be a fresh, top-level session, not a child of this one.
    env = {k: v for k, v in os.environ.items() if k not in SESSION_MARKERS}
    env.update(extra)
    return env


def sh(cmd: list[str] | str, cwd: Path, env: dict[str, str] | None = None, timeout: int = 120) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, cwd=cwd, env=env, shell=isinstance(cmd, str), capture_output=True, text=True, timeout=timeout)


def build_paws(dest: Path) -> Path:
    dest.mkdir(parents=True, exist_ok=True)
    out = dest / "paws"
    r = sh(["go", "build", "-o", str(out), "./cmd/paws"], REPO)
    if r.returncode != 0:
        raise SystemExit(f"go build failed:\n{r.stderr}")
    return out


# ---------- doctor / list ----------

def cmd_doctor(_: argparse.Namespace) -> int:
    ok = True
    for tool in ("go", "git", "claude"):
        path = shutil.which(tool)
        print(f"{tool:<8} {path or 'MISSING'}")
        ok &= bool(path)
    if not ok:
        return FAIL
    print(f"claude   {sh(['claude', '--version'], REPO).stdout.strip()}")
    with tempfile.TemporaryDirectory() as d:
        build_paws(Path(d))
        print(f"paws     builds from {REPO}")
        r = sh(["claude", "-p", "Reply with exactly: PONG", "--model", "haiku", "--setting-sources", "project"],
               Path(d), env=agent_env({}), timeout=120)
    authed = "PONG" in r.stdout
    print(f"auth     {'ok' if authed else 'FAILED: ' + (r.stdout + r.stderr).strip()[:200]}")
    return PASS if authed else FAIL


def cmd_list(_: argparse.Namespace) -> int:
    for case in sorted(p for p in CASES.iterdir() if (p / "case.toml").exists()):
        spec = tomllib.loads((case / "case.toml").read_text())
        print(f"{case.name:<32} {spec.get('description', '')}")
    return PASS


# ---------- evidence parsing ----------

def parse_stream(path: Path) -> list[dict]:
    """Flatten stream-json into an ordered list of tool_use / tool_result / text events."""
    events = []
    for line in path.read_text().splitlines():
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue
        if msg.get("type") == "result":
            events.append({"kind": "result", "text": msg.get("result", ""), "is_error": msg.get("is_error")})
            continue
        if msg.get("type") not in ("assistant", "user"):
            continue
        content = (msg.get("message") or {}).get("content")
        if not isinstance(content, list):
            continue
        for block in content:
            t = block.get("type")
            if t == "tool_use":
                events.append({"kind": "tool_use", "name": block.get("name"), "input": block.get("input") or {}})
            elif t == "tool_result":
                c = block.get("content")
                text = c if isinstance(c, str) else json.dumps(c)
                events.append({"kind": "tool_result", "text": text, "is_error": bool(block.get("is_error"))})
            elif t == "text":
                events.append({"kind": "text", "text": block.get("text", "")})
    return events


def read_audit(path: Path) -> list[dict]:
    if not path.exists():
        return []
    out = []
    for line in path.read_text().splitlines():
        try:
            out.append(json.loads(line))
        except json.JSONDecodeError:
            pass
    return out


def subset(record: dict, want: dict) -> bool:
    return all(str(record.get(k)) == str(v) for k, v in want.items())


# ---------- grading ----------

def grade(expect: dict, events: list[dict], audit: list[dict], project: Path) -> list[dict]:
    checks = []

    def check(name: str, ok: bool, detail: str = "") -> None:
        checks.append({"check": name, "pass": bool(ok), "detail": detail})

    invoked = [e["input"].get("skill", "") for e in events if e["kind"] == "tool_use" and e["name"] == "Skill"]
    for skill in expect.get("skills_invoked", []):
        check(f"skill invoked: {skill}", any(s == skill or s.endswith(":" + skill) for s in invoked), f"invoked={invoked}")
    for skill in expect.get("skills_not_invoked", []):
        check(f"skill not invoked: {skill}", not any(s == skill or s.endswith(":" + skill) for s in invoked), f"invoked={invoked}")

    results = [e for e in events if e["kind"] == "tool_result"]
    for text in expect.get("tool_result_contains", []):
        check(f"tool result contains: {text!r}", any(text in r["text"] for r in results))
    for text in expect.get("tool_result_absent", []):
        check(f"tool result absent: {text!r}", not any(text in r["text"] for r in results))

    # Ordered subsequence: each wanted record must match a later audit line than the previous one.
    if "audit" in expect:
        i, missing = 0, None
        for want in expect["audit"]:
            while i < len(audit) and not subset(audit[i], want):
                i += 1
            if i == len(audit):
                missing = want
                break
            i += 1
        check("audit sequence", missing is None, f"first unmatched: {missing}" if missing else f"{len(expect['audit'])} records in order")
    for want in expect.get("audit_absent", []):
        hits = [a for a in audit if subset(a, want)]
        check(f"audit absent: {want}", not hits, f"{len(hits)} matching")
    for rule in expect.get("audit_max", []):
        want = {k: v for k, v in rule.items() if k != "max"}
        n = sum(subset(a, want) for a in audit)
        check(f"audit at most {rule['max']}: {want}", n <= rule["max"], f"{n} matching")

    final = next((e["text"] for e in reversed(events) if e["kind"] == "result"), "")
    for text in expect.get("final_contains", []):
        check(f"final reply contains: {text!r}", text in final)

    for cmd in expect.get("shell", []):
        r = sh(cmd, project)
        check(f"shell: {cmd}", r.returncode == 0, (r.stdout + r.stderr).strip()[:300])
    return checks


# ---------- run ----------

def run_case(case: Path, run_dir: Path, paws_bin: Path, model: str | None, keep: bool) -> int:
    spec = tomllib.loads((case / "case.toml").read_text())
    unknown = set(spec.get("expect", {})) - EXPECT_KEYS
    if unknown:
        print(f"  unknown [expect] keys: {sorted(unknown)}")
        return FAIL
    evidence = run_dir / case.name
    evidence.mkdir(parents=True)
    scratch = Path(tempfile.mkdtemp(prefix=f"paws-verify-{case.name}-"))
    try:
        return drive_case(case, spec, evidence, scratch, paws_bin, model)
    finally:
        if keep:
            print(f"  scratch kept at {scratch}")
        else:
            shutil.rmtree(scratch, ignore_errors=True)


def drive_case(case: Path, spec: dict, evidence: Path, scratch: Path, paws_bin: Path, model: str | None) -> int:
    project, paws_home = scratch / "proj", scratch / "paws-home"
    paws_home.mkdir()

    if (case / "project").exists():
        shutil.copytree(case / "project", project)
    else:
        project.mkdir()
    for rel in spec.get("from_catalog", []):
        # Ship-path check: the real catalog binding, copied as a repo rule.
        dest = project / ".paws" / "catalog" / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy(REPO / "catalog" / rel, dest)
    if (case / "skills").exists():
        shutil.copytree(case / "skills", project / ".claude" / "skills")
    if (case / "paws-home").exists():
        shutil.copytree(case / "paws-home", paws_home, dirs_exist_ok=True)

    env = agent_env({"PAWS_HOME": str(paws_home), "PATH": f"{paws_bin.parent}{os.pathsep}{os.environ['PATH']}"})
    for cmd in ["git init -q .", "git config user.email verify@paws.local", "git config user.name paws-verify",
                *spec.get("setup", [])]:
        r = sh(cmd, project, env)
        if r.returncode != 0:
            print(f"  setup failed: {cmd}\n{r.stderr}")
            return INDETERMINATE

    r = sh([str(paws_bin), "init", *spec.get("init_args", [])], project, env)
    if r.returncode != 0:
        print(f"  paws init failed: {r.stderr}")
        return INDETERMINATE
    shutil.copy(project / ".claude" / "settings.json", evidence / "settings.json")

    args = ["claude", "-p", spec["prompt"], "--model", model or spec.get("model", "sonnet"),
            "--setting-sources", "project", "--output-format", "stream-json", "--verbose"]
    if spec.get("allowed_tools"):
        args += ["--allowedTools", *spec["allowed_tools"]]
    started = time.time()
    try:
        r = sh(args, project, env, timeout=spec.get("timeout", 300))
        (evidence / "stream.jsonl").write_text(r.stdout)
        (evidence / "stderr.txt").write_text(r.stderr)
    except subprocess.TimeoutExpired as e:
        def text(v: bytes | str | None) -> str:
            return v.decode(errors="replace") if isinstance(v, bytes) else (v or "")
        (evidence / "stream.jsonl").write_text(text(e.stdout))
        (evidence / "stderr.txt").write_text(text(e.stderr))
        print(f"  agent timed out after {spec.get('timeout', 300)}s")
        return INDETERMINATE
    elapsed = round(time.time() - started, 1)

    events = parse_stream(evidence / "stream.jsonl")
    audit = read_audit(paws_home / "audit.log")
    (evidence / "audit.jsonl").write_text("".join(json.dumps(a) + "\n" for a in audit))
    (evidence / "events.json").write_text(json.dumps(events, indent=2))
    (evidence / "git-log.txt").write_text(sh("git log --format='%h %s' 2>/dev/null", project).stdout)

    if not any(e["kind"] == "result" for e in events):
        print(f"  agent produced no result; see {evidence / 'stderr.txt'}")
        return INDETERMINATE

    checks = grade(spec.get("expect", {}), events, audit, project)
    if not checks:
        checks = [{"check": "case declares at least one expectation", "pass": False, "detail": "empty [expect]"}]
    verdict = PASS if all(c["pass"] for c in checks) else FAIL
    (evidence / "result.json").write_text(json.dumps(
        {"case": case.name, "verdict": ["PASS", "FAIL", "INDETERMINATE"][verdict], "seconds": elapsed,
         "tools": [f"{e['name']}" for e in events if e["kind"] == "tool_use"], "checks": checks}, indent=2))
    for c in checks:
        print(f"  {'ok ' if c['pass'] else 'BAD'} {c['check']}" + ("" if c["pass"] else f"  [{c['detail']}]"))
    return verdict


def cmd_run(args: argparse.Namespace) -> int:
    names = sorted(p.name for p in CASES.iterdir() if (p / "case.toml").exists()) if args.all else args.cases
    if not names:
        print("name at least one case, or pass --all")
        return FAIL
    run_dir = ARTIFACTS / f"{time.strftime('%Y%m%d-%H%M%S')}-{os.getpid()}"
    paws_bin = build_paws(run_dir / "bin")
    worst = PASS
    for name in names:
        case = CASES / name
        if not (case / "case.toml").exists():
            print(f"{name}: no such case")
            worst = max(worst, FAIL)
            continue
        print(f"{name}")
        v = run_case(case, run_dir, paws_bin, args.model, args.keep)
        print(f"  => {['PASS', 'FAIL', 'INDETERMINATE'][v]}")
        worst = max(worst, v)
    print(f"evidence: {run_dir}")
    return worst


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    sub.add_parser("doctor").set_defaults(fn=cmd_doctor)
    sub.add_parser("list").set_defaults(fn=cmd_list)
    run = sub.add_parser("run")
    run.add_argument("cases", nargs="*")
    run.add_argument("--all", action="store_true")
    run.add_argument("--model")
    run.add_argument("--keep", action="store_true", help="keep the scratch repo for inspection")
    run.set_defaults(fn=cmd_run)
    args = p.parse_args()
    return args.fn(args)


if __name__ == "__main__":
    sys.exit(main())
