#!/usr/bin/env python3
"""Run `paws explain -` with a made-up session transcript, for rules whose when: reads it.

    explain_tx.py TOOL TARGET [--edited PATH]... [--agents N] [--agents-before N]

The transcript holds --agents-before Agent calls, then a user prompt, then one successful
Edit per --edited path, then --agents Agent calls. TARGET is the file path for file tools,
else the command or prompt. Prints paws explain's output.
"""
import argparse
import json
import os
import subprocess
import tempfile

p = argparse.ArgumentParser()
p.add_argument("tool")
p.add_argument("target")
p.add_argument("--edited", action="append", default=[])
p.add_argument("--agents", type=int, default=0)
p.add_argument("--agents-before", type=int, default=0)
a = p.parse_args()

n = 0


def call(name, inp):
    global n
    n += 1
    return {"type": "assistant", "message": {"role": "assistant", "content": [
        {"type": "tool_use", "id": f"t{n}", "name": name, "input": inp}]}}, f"t{n}"


lines = []
for _ in range(a.agents_before):
    lines.append(call("Agent", {"prompt": "earlier task"})[0])
lines.append({"type": "user", "message": {"role": "user", "content": "go"}})
for path in a.edited:
    entry, tid = call("Edit", {"file_path": os.path.abspath(path)})
    lines += [entry, {"type": "user", "message": {"role": "user", "content": [
        {"type": "tool_result", "tool_use_id": tid, "content": "ok"}]}}]
for _ in range(a.agents):
    lines.append(call("Agent", {"prompt": "subtask"})[0])

with tempfile.NamedTemporaryFile("w", suffix=".jsonl", delete=False) as f:
    f.write("".join(json.dumps(x) + "\n" for x in lines))

key = "file_path" if a.tool in ("Read", "Write", "Edit") else ("prompt" if a.tool in ("Agent", "Task") else "command")
event = {"hook_event_name": "PreToolUse", "tool_name": a.tool, "tool_input": {key: a.target},
         "transcript_path": f.name, "cwd": os.getcwd()}
out = subprocess.run(["paws", "explain", "-"], input=json.dumps(event), capture_output=True, text=True)
print(out.stdout + out.stderr, end="")
os.unlink(f.name)
