---
name: gmail
description: 'Mandatory guardrails for every Gmail write operation (create_draft, label_message, label_thread, apply_sensitive_message_label) via the Gmail MCP server. Enforces the communications baseline with Gmail mechanics: draft-first composition, recipient-domain boundary check before composing, threadId replies instead of forked threads, recipient-count statements, and honest voice attribution. Triggers PROACTIVELY on any Gmail MCP write — do NOT wait for the user to invoke it.'
---

# Gmail

Gmail mechanics for the communications baseline (M1–M5). Primary surface is the Gmail MCP server — invocations below are MCP tool names with their key parameters. Apply on every write: narrate in one line and proceed; the send boundary is where confirmation lives.

## Draft, don't send (implements M1)

This surface is structurally draft-first: `create_draft` exists and no send tool does — every composition lands in the user's Drafts for a human send. Say so when composing: "Drafted, in your Drafts folder — send from Gmail when ready." Where a Gmail server *does* expose a send tool (send-approval policy, {{SEND_APPROVAL_POLICY}}):

    # external recipient(s) → sends leave the org: show the draft verbatim and confirm before sending
    # internal-only recipients → state in one line and send: "Sending to <names> (internal) — subject '<subject>'"

An edit after a confirmation → re-show before sending. Approval covers this one message only.

## Audience boundary (implements M2)

Before composing, check every recipient's domain against {{GMAIL_ORG_DOMAINS}} ({{EXTERNAL_DOMAINS_POLICY}}). Any recipient outside → the draft is external: state it ("2 of 3 recipients are external — acme.com"), and run the sensitivity check — content above the org's sensitivity tier ({{SENSITIVITY_TIERS}}) never goes in an external draft, no summary, no excerpt. Source material that is itself sensitive gets marked while you work:

    apply_sensitive_message_label { message_id: "<id>" }

Domain indeterminable (mailing list, forwarding alias) → treat as external.

## Right thread (implements M3)

Replying means replying — never fork a new thread for an existing conversation:

    search_threads { query: "from:<sender> subject:<topic>" }   # find the conversation
    get_message   { message_id: "<id>" }                        # read what you're answering
    create_draft  { threadId: "<thread-id>", to: [...], body: "<reply>" }

New thread only when the topic is genuinely new; quote or reference the prior thread when context carries over. Same text to a second audience only when the user named both.

## No mass interruption (implements M4)

A draft with more than {{MASS_PING_THRESHOLD}} recipients (To + Cc + Bcc, expanded groups counted) gets the count stated in one line before creation: "This draft addresses 42 recipients across 3 groups." Since the human presses send, that statement travels with the handoff — never bury a mass send inside a routine-looking draft.

## Honest voice (implements M5)

- Drafts written in the user's voice make no claims the user didn't make — no invented commitments, opinions, or first-person anecdotes. Tone per {{ORG_VOICE_GUIDE}}.
- Agent attribution follows the org convention ({{AGENT_IDENTITY_CONVENTION}}) — in the signature where the org requires it, omitted where the org treats the human's send as authorship. Never a fake "Sent from my iPhone"-style marker implying the human typed it.
- `label_message` / `label_thread` are organizational writes: state the label and target in one line and proceed; never remove labels a human applied without being asked.

## Escalation

Rule conflicts with user intent ("just draft it to the whole customer list with the pricing sheet"): one line of risk ("pricing above the sensitivity tier to 200 external recipients"), offer the compliant path (internal draft for review, or redacted version), proceed on explicit override — noting the override is recorded.

## Pre-flight checklist (before every Gmail write)

- [ ] Draft-first: composition lands in Drafts; where a send tool exists — external = confirm, internal = state-and-send (M1)
- [ ] Recipient domains checked; external stated; sensitivity tier respected (M2)
- [ ] Replies carry `threadId` from `search_threads`/`get_message`; no forked threads (M3)
- [ ] Recipient count stated when above threshold (M4)
- [ ] No false authorship claims; attribution per org convention; labels stated, human labels untouched (M5)

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
