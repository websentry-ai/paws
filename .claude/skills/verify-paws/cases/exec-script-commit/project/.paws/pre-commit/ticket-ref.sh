#!/bin/sh
# Deny a commit whose command does not mention a PAWS-<n> ticket.
if cat | grep -Eq 'PAWS-[0-9]+'; then
  exit 0
fi
echo "Commit messages must reference a ticket, e.g. 'PAWS-101: initial'. Retry with a ticket reference." >&2
exit 2
