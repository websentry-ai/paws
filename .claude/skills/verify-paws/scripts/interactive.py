#!/usr/bin/env python3
"""Drive the real interactive Claude Code TUI in tmux against a freshly built paws.

    interactive.py list
    interactive.py run <case> [<case> ...] [--model sonnet] [--keep]
    interactive.py run --all

For what headless can't show: the TUI's own rendering of paws denies and the
skill load that follows. Cases live under ../cases-interactive/ and use the
same layout and [expect] keys as drive.py, plus:

    answer = "yes" | "no"        how to answer each permission prompt (default: fail on one)
    [expect] screen_contains     text that must appear on screen at some point
    [expect] screen_absent       text that must never appear
    [expect] prompts             exact number of permission prompts answered

Same isolation as drive.py: scratch repo and PAWS_HOME, --setting-sources project,
fake binaries first on PATH, Write/Edit of ~/.paws and ~/.claude disallowed. Skills
and tool results are graded from the session transcript Claude Code writes.
Evidence (screens, transcript, audit) goes to .verify-artifacts/<run-id>/<case>/.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import tomllib
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import drive  # noqa: E402

CASES = drive.SKILL_DIR / "cases-interactive"
COLS, ROWS = 140, 50
EXPECT_KEYS = drive.EXPECT_KEYS - {"final_contains"} | {"screen_contains", "screen_absent", "prompts"}

READY_RE = re.compile(r"for shortcuts|shift\+tab")  # the footer under the input box
TRUST_RE = re.compile(r"trust this folder", re.I)
PERMISSION_RE = re.compile(r"^\s*❯?\s*1\.\s+Yes\b", re.M)
NO_OPTION_RE = re.compile(r"^\s*❯?\s*(\d)\.\s+No\b", re.M)


class Tmux:
    def __init__(self) -> None:
        self.sock = f"paws-verify-{os.getpid()}"

    def __call__(self, *args: str, check: bool = True) -> str:
        r = subprocess.run(["tmux", "-L", self.sock, *args], capture_output=True, text=True)
        if check and r.returncode:
            raise RuntimeError(f"tmux {args}: {r.stderr}")
        return r.stdout

    def screen(self) -> str:
        return self("capture-pane", "-t", "v", "-p", "-J", check=False)

    def keys(self, *keys: str) -> None:
        self("send-keys", "-t", "v", *keys)

    def type(self, text: str) -> None:
        self("send-keys", "-t", "v", "-l", "--", text)


def transcript_events(path: Path) -> list[dict]:
    """The main-thread tool_use / tool_result events from a session transcript."""
    events = []
    for line in path.read_text().splitlines():
        try:
            e = json.loads(line)
        except json.JSONDecodeError:
            continue
        if e.get("isSidechain") or e.get("type") not in ("assistant", "user"):
            continue
        content = (e.get("message") or {}).get("content")
        if not isinstance(content, list):
            continue
        for b in content:
            if b.get("type") == "tool_use":
                events.append({"kind": "tool_use", "name": b.get("name"), "input": b.get("input") or {}})
            elif b.get("type") == "tool_result":
                c = b.get("content")
                events.append({"kind": "tool_result", "text": c if isinstance(c, str) else json.dumps(c),
                               "is_error": bool(b.get("is_error"))})
    return events


def find_transcript(project: Path, since: float) -> Path | None:
    """The newest session transcript Claude Code wrote for this cwd after `since`. Read only."""
    root = Path.home() / ".claude" / "projects"
    want = str(project.resolve())
    best = None
    for f in root.glob("*/*.jsonl"):
        if f.stat().st_mtime < since:
            continue
        with f.open() as fh:
            for line in fh:
                try:
                    cwd = json.loads(line).get("cwd")
                except json.JSONDecodeError:
                    continue
                if cwd:
                    if Path(cwd).resolve() == Path(want) and (best is None or f.stat().st_mtime > best.stat().st_mtime):
                        best = f
                    break
    return best


def run_tui(spec: dict, project: Path, env: dict[str, str], evidence: Path, model: str | None) -> tuple[list[str], int, str]:
    """Start claude in tmux, send the prompt, answer permission prompts, wait until idle.
    Returns every distinct screen seen, the number of prompts answered, and an error."""
    tmux = Tmux()
    args = ["claude", "--model", model or spec.get("model", "sonnet"), "--setting-sources", "project"]
    if spec.get("allowed_tools"):
        args += ["--allowedTools", *spec["allowed_tools"]]
    args += ["--disallowedTools", *drive.NEVER_EDIT]
    keep = ("HOME", "USER", "LANG", "TMPDIR", "PAWS_HOME", "PATH")
    child = {k: env[k] for k in keep if k in env} | {k: v for k, v in env.items() if k in spec.get("env", {})}
    child |= {"TERM": "xterm-256color", "SHELL": "/bin/zsh", "LANG": env.get("LANG", "en_US.UTF-8")}
    tmux("new-session", "-d", "-s", "v", "-x", str(COLS), "-y", str(ROWS), "-c", str(project), "--",
         "env", "-i", *[f"{k}={v}" for k, v in child.items()], *args)
    tmux("set", "-g", "status", "off")
    screens: list[str] = []
    answered, error = 0, ""

    def seen(s: str) -> str:
        if not screens or screens[-1] != s:
            screens.append(s)
        return s

    def wait(pred, timeout: float) -> str | None:
        end = time.time() + timeout
        while time.time() < end:
            s = seen(tmux.screen())
            if pred(s):
                return s
            time.sleep(0.3)
        return None

    try:
        s = wait(lambda s: TRUST_RE.search(s) or READY_RE.search(s), 60)
        if s is None:
            return screens, answered, "claude did not start"
        # Workspace trust dialog for the new scratch dir: move the cursor to "Yes", then confirm.
        for _ in range(6):
            if not TRUST_RE.search(s):
                break
            tmux.keys("Enter" if re.search(r"❯\s*(\d\.\s*)?Yes", s) else "Down")
            time.sleep(0.6)
            s = seen(tmux.screen())
        if wait(lambda s: not TRUST_RE.search(s) and READY_RE.search(s), 60) is None:
            return screens, answered, "stuck on the trust dialog"
        time.sleep(1)
        tmux.type(spec["prompt"])
        time.sleep(0.5)
        tmux.keys("Enter")

        still, last = time.time(), ""
        end = time.time() + spec.get("timeout", 300)
        while time.time() < end:
            time.sleep(0.4)
            s = seen(tmux.screen())
            if PERMISSION_RE.search(s):
                answer = spec.get("answer")
                if answer == "yes":
                    tmux.keys("1")
                elif answer == "no":
                    m = NO_OPTION_RE.search(s)
                    tmux.keys(m.group(1) if m else "Escape")
                else:
                    return screens, answered, "unexpected permission prompt"
                answered += 1
                time.sleep(1.5)
                continue
            if s != last:
                still, last = time.time(), s
            elif "esc to interrupt" not in s and time.time() - still > 8 and READY_RE.search(s):
                break
        else:
            error = "timed out"
        tmux.type("/exit")
        tmux.keys("Enter")
        time.sleep(1.5)
    finally:
        (evidence / "screens.txt").write_text("\n\n=====\n\n".join(screens))
        tmux("kill-server", check=False)
    return screens, answered, error


def run_case(case: Path, run_dir: Path, installer: drive.Installer, model: str | None, keep: bool) -> int:
    spec = tomllib.loads((case / "case.toml").read_text())
    unknown = set(spec.get("expect", {})) - EXPECT_KEYS
    if unknown:
        print(f"  unknown [expect] keys: {sorted(unknown)}")
        return drive.FAIL
    evidence = run_dir / case.name
    evidence.mkdir(parents=True)
    scratch = Path(tempfile.mkdtemp(prefix=f"paws-verify-tui-{case.name}-"))
    try:
        prepared = drive.prepare(case, spec, evidence, scratch, installer)
        if isinstance(prepared, int):
            return prepared
        project, paws_home, env = prepared
        started = time.time()
        screens, answered, error = run_tui(spec, project, env, evidence, model)
        if error:
            print(f"  {error}; see {evidence / 'screens.txt'}")
            return drive.INDETERMINATE
        transcript = find_transcript(project, started)
        events = transcript_events(transcript) if transcript else []
        if transcript:
            shutil.copy(transcript, evidence / "transcript.jsonl")
        audit = drive.read_audit(paws_home / "audit.log")
        (evidence / "audit.jsonl").write_text("".join(json.dumps(a) + "\n" for a in audit))

        expect = spec.get("expect", {})
        checks = drive.grade({k: v for k, v in expect.items() if k in drive.EXPECT_KEYS}, events, audit, project, env)
        text = "\n".join(screens)
        for t in expect.get("screen_contains", []):
            checks.append({"check": f"screen shows: {t!r}", "pass": t in text, "detail": ""})
        for t in expect.get("screen_absent", []):
            checks.append({"check": f"screen never shows: {t!r}", "pass": t not in text, "detail": ""})
        if "prompts" in expect:
            checks.append({"check": f"permission prompts answered: {expect['prompts']}",
                           "pass": answered == expect["prompts"], "detail": f"answered={answered}"})
        if not transcript:
            checks.append({"check": "session transcript found", "pass": False, "detail": ""})
        verdict = drive.PASS if checks and all(c["pass"] for c in checks) else drive.FAIL
        (evidence / "result.json").write_text(json.dumps(
            {"case": case.name, "verdict": ["PASS", "FAIL", "INDETERMINATE"][verdict],
             "seconds": round(time.time() - started, 1), "prompts": answered, "checks": checks}, indent=2))
        for c in checks:
            print(f"  {'ok ' if c['pass'] else 'BAD'} {c['check']}" + ("" if c["pass"] else f"  [{c['detail']}]"))
        return verdict
    finally:
        if keep:
            print(f"  scratch kept at {scratch}")
        else:
            shutil.rmtree(scratch, ignore_errors=True)


def cmd_list(_: argparse.Namespace) -> int:
    for case in sorted(p for p in CASES.iterdir() if (p / "case.toml").exists()):
        print(f"{case.name:<32} {tomllib.loads((case / 'case.toml').read_text()).get('description', '')}")
    return drive.PASS


def cmd_run(args: argparse.Namespace) -> int:
    if not shutil.which("tmux"):
        print("tmux is required")
        return drive.FAIL
    names = sorted(p.name for p in CASES.iterdir() if (p / "case.toml").exists()) if args.all else args.cases
    if not names:
        print("name at least one case, or pass --all")
        return drive.FAIL
    run_dir = drive.ARTIFACTS / f"{time.strftime('%Y%m%d-%H%M%S')}-{os.getpid()}-tui"
    installer = drive.Installer.go(run_dir)
    worst = drive.PASS
    for name in names:
        case = CASES / name
        if not (case / "case.toml").exists():
            print(f"{name}: no such case")
            worst = max(worst, drive.FAIL)
            continue
        print(name)
        v = run_case(case, run_dir, installer, args.model, args.keep)
        print(f"  => {['PASS', 'FAIL', 'INDETERMINATE'][v]}")
        worst = max(worst, v)
    print(f"evidence: {run_dir}")
    return worst


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
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
