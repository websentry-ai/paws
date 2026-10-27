# gmail digest

Gmail MCP writes: send, reply, forward, draft or label. A sent message is the org speaking and cannot be unsent. Apply to this call and its follow-ups.

## M1: draft first
- `create_draft` lands in the user's Drafts for a human send. Say: "Drafted, in your Drafts folder — send from Gmail when ready."
- If a send tool exists: external recipient(s) → show the draft verbatim (text, recipients, count) and send only on explicit approval; internal-only → one line ("Sending to <names> (internal) — subject '<subject>'") and send. Any edit after approval → re-show. Approval covers one message.

## M2: audience boundary, before composing
- Check every recipient's domain against the authenticated Gmail account's domain; any non-org domain is external. Mailing lists or aliases you can't resolve → external.
- External → state it ("2 of 3 recipients are external — acme.com").
- Anything not already public never goes in an external draft: no summary, no excerpt, no "just this once".
- Mark sensitive source mail as you work: `apply_sensitive_message_label { message_id: "<id>" }`.

## M3: right thread
- Replying means replying: `search_threads { query: "from:<sender> subject:<topic>" }` → `get_message { message_id }` → `create_draft { threadId: "<thread-id>", to: [...], body }`.
- New thread only for a genuinely new topic. Same text to a second audience only when the user named both.

## M4: no mass interruption
More than 5 recipients (To + Cc + Bcc, groups expanded) → state the count in one line before creating the draft ("This draft addresses 42 recipients across 3 groups"). Never bury a mass send in a routine-looking draft.

## M5: honest voice
- No claims the user didn't make: no invented commitments, opinions or anecdotes. Tone: plain, concise, no emojis.
- Attribution: `[drafted by AI agent for <user>]` in the signature where the org requires it; omitted where the human's send counts as authorship. Never a fake "Sent from my iPhone" marker.
- Labels: state label and target in one line; never remove labels a human applied unless asked.

## Escalation
Rule vs user intent: one line of risk, offer the compliant path (internal draft for review, redacted version), proceed on explicit override and say it is recorded.

## Checklist
- [ ] Draft-first; external = confirm, internal = state-and-send where sending exists
- [ ] Recipient domains checked; external stated; nothing non-public to external recipients
- [ ] Replies carry `threadId`; no forked threads
- [ ] Recipient count stated above 5
- [ ] No false authorship; attribution per convention; human labels untouched
