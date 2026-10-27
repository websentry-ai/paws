#!/usr/bin/env python3
"""A dependency-free stdio MCP server for verify-paws cases.

Tools: send_message(channel, text) appends {"channel","text"} to the JSONL file
named by argv[1] (the side effect cases assert on); list_messages() returns what
was sent. Speaks just enough MCP (initialize, tools/list, tools/call) for Claude Code.
"""
import json
import sys

LOG = sys.argv[1]
TOOLS = [
    {"name": "send_message", "description": "Send a chat message to a channel.",
     "inputSchema": {"type": "object", "properties": {"channel": {"type": "string"}, "text": {"type": "string"}},
                     "required": ["channel", "text"]}},
    {"name": "list_messages", "description": "List messages sent so far.",
     "inputSchema": {"type": "object", "properties": {}}},
]


def reply(id_, result):
    sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": id_, "result": result}) + "\n")
    sys.stdout.flush()


for line in sys.stdin:
    try:
        msg = json.loads(line)
    except json.JSONDecodeError:
        continue
    method, id_ = msg.get("method"), msg.get("id")
    if id_ is None:
        continue  # notifications
    if method == "initialize":
        reply(id_, {"protocolVersion": msg.get("params", {}).get("protocolVersion", "2025-06-18"),
                    "capabilities": {"tools": {}}, "serverInfo": {"name": "fakechat", "version": "1"}})
    elif method == "tools/list":
        reply(id_, {"tools": TOOLS})
    elif method == "tools/call":
        name, args = msg["params"]["name"], msg["params"].get("arguments", {})
        if name == "send_message":
            with open(LOG, "a") as f:
                f.write(json.dumps({"channel": args.get("channel"), "text": args.get("text")}) + "\n")
            text = f"sent to {args.get('channel')}"
        else:
            try:
                text = open(LOG).read() or "no messages"
            except FileNotFoundError:
                text = "no messages"
        reply(id_, {"content": [{"type": "text", "text": text}]})
    else:
        sys.stdout.write(json.dumps({"jsonrpc": "2.0", "id": id_, "error": {"code": -32601, "message": "unknown method"}}) + "\n")
        sys.stdout.flush()
