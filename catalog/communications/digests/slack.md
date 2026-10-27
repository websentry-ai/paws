# slack digest

Slack sends: `chat.postMessage`, `chat.scheduleMessage`, `chat.update`, `chat.delete`, `files.upload` (curl with `$SLACK_TOKEN`) or a Slack MCP send, draft or scheduled message. Apply before retrying.

## M1: narrate internal, draft external
- Internal one-off: compose, state destination in one line ("Sending to #eng-infra (42 members)"), post.
- External destination or mass send: show the draft first and send only on approval, exactly as approved:
  `Draft → #<channel-name> (<NN> members):` then the exact text, mentions and links included.
- Any edit after approval → re-preview. Approval covers one message.

## M2: boundary check before the first post to a conversation
- `curl -s https://slack.com/api/conversations.info -H "Authorization: Bearer $SLACK_TOKEN" -d "channel=<channel-id>"`
- `is_ext_shared: true` or `is_shared: true` → external (Slack Connect): draft-approve flow, status stated in the preview.
- DMs / group DMs: resolve member email domains (`conversations.members` + `users.info`); any non-org domain → external.
- Check fails or fields missing → treat as external.
- Anything not already public never goes to an external destination.

## M3: right channel, right thread
- Pick the channel for the topic; no registry is configured (`<channel-topic-registry>`), so choose the closest match and state the destination. Incidents → `<incident-channel>` (default: most recently active `inc-*` / `incident-*` channel, choice stated).
- Resolve names to IDs with `conversations.list`; never guess an ID.
- Replies pass the parent `ts` as `thread_ts`; never a new top-level post.
- No cross-posting unless the user named each destination.

## M4: mass interruption needs confirmation
- `<!channel>`, `<!here>`, `<!everyone>`, group DMs above 5 recipients, and posts to #general, #announcements or any all-* channel: confirm with the count from `num_members`: "This notifies all <NN> members of #<channel> — confirm?"
- Scheduled send (`post_at`) outside recipients' working hours → one-line note of local delivery time, then proceed.

## M5: identity and tone
- Every message carries `[sent by AI agent on behalf of <user>]`; never strip it or write as if the human typed it.
- Tone: plain, concise, no emojis.
- `chat.update` / `chat.delete` only on messages this agent posted (match the returned `ts`). State old → new (or "deleting") in one line; external destinations get the full preview.

## MCP
Same sequence per operation: conversation info check before `post_message` / `schedule_message` / `upload_file`.

## Escalation
State the risk and compliant path in one line, then proceed as asked (audit-logged). External sends and mass pings still need a confirmed yes.

## Checklist
- [ ] External/mass previewed and approved; internal narrated with channel + member count
- [ ] conversations.info run; shared flags checked; nothing non-public to external
- [ ] Destination stated; replies carry `thread_ts`; no unasked cross-posts
- [ ] Mass pings confirmed with `num_members`; off-hours schedule noted
- [ ] Agent marker present; update/delete only own `ts`
