#!/bin/sh
# Read the event JSON, deny (exit 2) on any recursive force delete, any target.
#
# This is text matching against the literal command string, not real shell
# parsing — it catches common forms (extra whitespace, split/reordered flags,
# case, long flags) but not deliberate obfuscation (command substitution,
# encoding, aliases). Treat this as a coarse example, not a hardened boundary.
payload=$(cat)
norm=$(printf '%s' "$payload" | tr '[:upper:]' '[:lower:]' | tr -s '[:blank:]' ' ')

if printf '%s' "$norm" | grep -Eq '(^|[^a-z])rm ' \
  && printf '%s' "$norm" | grep -Eq -- '(^| )-[a-z]*r[a-z]*( |$)|--recursive' \
  && printf '%s' "$norm" | grep -Eq -- '(^| )-[a-z]*f[a-z]*( |$)|--force'; then
  echo "paws rm-guard: refusing recursive force delete (rm -rf and variants), any target." >&2
  exit 2
fi
exit 0
