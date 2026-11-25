---
name: slack
description: 'Mandatory guardrails for every Slack send operation (chat.postMessage, chat.scheduleMessage, chat.update, chat.delete, files.upload) via the Slack Web API (curl) or Slack MCP tools. Enforces the communications baseline with Slack mechanics: is_ext_shared boundary check, draft-first for external and mass sends, narrated internal sends, thread_ts replies, and @channel confirmation. Triggers PROACTIVELY on any Slack write — do NOT wait for the user to invoke it.'
---

# Slack

Slack mechanics for the communications baseline (M1–M5). Apply automatically on every Slack send — never skip. Primary surface is the Slack Web API via curl with `$SLACK_TOKEN` ({{SLACK_WORKSPACE}}); Slack MCP tools follow the same sequences (see MCP note at the end).

## Send convention (implements M1)

Internal one-off message: compose, state the destination in one line — "Sending to #eng-infra (42 members)" — and post:

    curl -X POST https://slack.com/api/chat.postMessage \
      -H "Authorization: Bearer $SLACK_TOKEN" -H "Content-Type: application/json" \
      -d '{"channel": "<channel-id>", "text": "<message text>"}'

External destination (per the boundary check below) or mass send (per M4): show the draft first and send only on approval, exactly as approved:

    Draft → #<channel-name> (<NN> members):
    ---
    <exact message text, verbatim, including mentions and links>
    ---

The org's send-approval policy ({{SEND_APPROVAL_POLICY}}) governs session pre-authorization. Any edit after an approval → re-preview; approval covers that one message only.

## Boundary check (implements M2)

Before the first post/upload to any conversation this session:

    curl -s https://slack.com/api/conversations.info \
      -H "Authorization: Bearer $SLACK_TOKEN" -d "channel=<channel-id>"

`is_ext_shared: true` or `is_shared: true` in the response → the channel is external (Slack Connect); it gets the M1 draft-approve flow, with that status stated in the preview. For DMs/group DMs, resolve members' email domains (`conversations.members` + `users.info`) — any domain outside the org's domains makes the destination external ({{EXTERNAL_DOMAINS_POLICY}}). Content above the org's sensitivity tier ({{SENSITIVITY_TIERS}}) never goes to an external destination. Check fails or fields missing → treat as external.

## Right channel, right thread (implements M3)

- Destination comes from {{CHANNEL_TOPIC_REGISTRY}}; incidents go to {{INCIDENT_CHANNEL}}. No registry entry → pick the closest match and state the destination in the send line. Resolve names to IDs, never guess an ID:

      curl -s https://slack.com/api/conversations.list \
        -H "Authorization: Bearer $SLACK_TOKEN" \
        -d "types=public_channel,private_channel&limit=200"

- Replying to an existing message: pass the parent's `ts` as `thread_ts` — never a new top-level post:

      curl -X POST https://slack.com/api/chat.postMessage \
        -H "Authorization: Bearer $SLACK_TOKEN" -H "Content-Type: application/json" \
        -d '{"channel": "<channel-id>", "thread_ts": "<parent-ts>", "text": "<message text>"}'

- Never post the same text to a second channel unless the user named both destinations.

## Mass interruption (implements M4)

`<!channel>`, `<!here>`, `<!everyone>` in the text, group DMs above {{MASS_PING_THRESHOLD}} recipients, and posts to {{ANNOUNCE_CHANNELS}} require explicit confirmation with the count stated, taken from `num_members` in the conversations.info response: "This notifies all <NN> members of #<channel> — confirm?" A scheduled send whose `post_at` lands outside recipients' working hours gets a one-line note of the local delivery time ("lands 06:10 local for most recipients"), then proceeds:

    curl -X POST https://slack.com/api/chat.scheduleMessage \
      -H "Authorization: Bearer $SLACK_TOKEN" -H "Content-Type: application/json" \
      -d '{"channel": "<channel-id>", "post_at": <epoch-seconds>, "text": "<message text>"}'

## Identity and tone (implements M5)

- Every message carries the org's agent marker ({{AGENT_IDENTITY_CONVENTION}}) — never strip it, never phrase the message as if the human typed it beyond what that convention allows.
- Tone follows {{ORG_VOICE_GUIDE}}.
- `chat.update` and `chat.delete` apply only to messages this agent posted (match on the send's returned `ts`). State old text → new text (or "deleting") in one line as you go; external destinations get the full M1 preview.

## MCP note

When the Slack MCP server is the surface, the same sequences apply operation-for-operation: conversation info check before `post_message` / `schedule_message` / `upload_file`; same send convention on every write. Map by operation, not tool name — server tool names vary.

## Escalation

Rule vs user intent: state the risk and the compliant path in one line, then proceed as asked — the override is audit-logged. Only external sends and mass pings still need a confirmed yes.

## Pre-flight checklist (before every Slack send)

- [ ] External or mass send previewed and approved; internal send narrated with channel + member count
- [ ] conversations.info run; `is_ext_shared`/`is_shared` checked; sensitivity tier respected
- [ ] Destination stated; replies carry `thread_ts`; no unasked cross-posts
- [ ] `<!channel>`/`<!here>`/large DM confirmed with `num_members`; off-hours schedule noted in one line
- [ ] Agent identity marker present; update/delete only own `ts`

## Category baseline: Communications

These rules apply to **every** message-sending tool. An instance skill (Slack, Gmail, …) adds tool-specific mechanics on top; it may tighten these rules but never loosen them. A sent message is the org speaking and cannot be unsent — but most messages are routine. Ceremony is reserved for the sends that can actually hurt: external recipients and mass interruption. Everything else is narrated, not negotiated.

### Rule M1: Draft externally, narrate internally

1. External recipients (per M2) and mass sends (per M4): compose the full message, show it verbatim — exact text, exact destination, recipient count — and send only on explicit approval. Any edit after approval → re-preview.
2. Internal one-off messages to org channels or members: compose, state the destination in one line ("Sending to #eng-infra, 42 members"), and send. No approval loop.
3. The org's send-approval policy ({{SEND_APPROVAL_POLICY}}) may tighten this or pre-authorize a session per audience; it never loosens the external/mass cases.

### Rule M2: Respect the audience boundary

Before sending, determine whether the destination is internal or external using the instance skill's exact check mechanics:

1. External destinations: shared/Connect channels, DMs including external members, email addresses outside the org's domains ({{EXTERNAL_DOMAINS_POLICY}}).
2. Content above the org's sensitivity tier ({{SENSITIVITY_TIERS}}) never crosses the boundary — no summary, no excerpt, no "just this once".
3. Boundary indeterminable → treat the destination as external.

### Rule M3: Right channel, right thread

- Post to the channel the org designates for the topic ({{CHANNEL_TOPIC_REGISTRY}}); no registry entry → pick the closest match and state the destination as you send.
- Replying to an existing message → reply in its thread, never a new top-level post.
- Never cross-post the same message to multiple channels unless the user explicitly asked for each destination.

### Rule M4: No mass interruption

@channel / @here / @everyone (and tool-specific equivalents) and group DMs above {{MASS_PING_THRESHOLD}} recipients require explicit same-session confirmation with the recipient count stated: "this reaches NN people — confirm?" A scheduled send that lands outside the recipients' working hours gets a one-line note of the local delivery time, then proceeds.

### Rule M5: Agent identity is honest

- The agent identifies itself per the org convention ({{AGENT_IDENTITY_CONVENTION}}) and never impersonates the human beyond it — no stripping the marker, no first-person claims the human didn't make.
- Tone and style follow {{ORG_VOICE_GUIDE}}.
- Edits and deletions apply only to messages the agent itself sent. State old → new (or "deleting") in one line as you go; external destinations get the full M1 preview.

### Pre-flight checklist (run before every outbound message)

- [ ] External or mass send previewed and approved; internal send narrated with destination + count (M1/M4)
- [ ] Internal/external determined via instance mechanics; sensitive content never crosses the boundary (M2)
- [ ] Channel matches the topic; replies threaded; no unasked cross-posts (M3)
- [ ] Identity marker present; edits/deletes only own messages (M5)

## Defaults

Values in angle brackets are org settings this skill ships without. Until your org fills them in (the `placeholders:` map in this skill's `config.yaml`, or an edit to this file), apply the default below.

- `<channel-topic-registry>`: No registry of topic → channel. Pick the closest-match channel and state the destination when sending.
- `<incident-channel>`: The most recently active channel matching `inc-*` or `incident-*`. Use it and state the choice when posting.
