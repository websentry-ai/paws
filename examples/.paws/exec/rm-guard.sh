#!/bin/sh
# Read the event JSON, deny (exit 2) on any recursive force delete, any target.
payload=$(cat)
case "$payload" in
  *"rm -rf"*|*"rm -fr"*)
    echo "paws rm-guard: refusing recursive force delete (rm -rf/-fr), any target." >&2
    exit 2 ;;
esac
exit 0
