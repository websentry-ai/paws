#!/bin/sh
# Exit 2 (deny) when a recursive rm targets root, a system dir, home, ., .., * or a $VAR-led path.
# Reads the paws event JSON on stdin; only the "target" field (the Bash command) is checked.
cmd=$(sed -nE 's/.*"target":"(([^"\\]|\\.)*)".*/\1/p' | sed -e 's/\\n/;/g' -e 's/\\t/ /g' -e 's/\\u0026/\&/g' -e 's/\\u003c/</g' -e 's/\\u003e/>/g' -e 's/\\"/"/g' -e 's/\\\\/\\/g')

set -f
printf '%s\n' "$cmd" | tr ';&|()' '\n\n\n\n\n' | while IFS= read -r seg; do
	set -- $seg
	while [ $# -gt 0 ] && { [ "$1" = sudo ] || case $1 in *=*) true ;; *) false ;; esac; }; do shift; done
	[ "$1" = rm ] || continue
	shift
	recursive=no
	targets=
	for a in "$@"; do
		case $a in
		--recursive) recursive=yes ;;
		--*) ;;
		-*[rR]*) recursive=yes ;;
		-*) ;;
		*) targets="$targets $a" ;;
		esac
	done
	[ $recursive = yes ] || continue
	for t in $targets; do
		t=$(printf '%s' "$t" | tr -d "\"'")
		case $t in
		'/' | '/*' | '~' | '~/' | '~/*' | '.' | './' | './*' | '..' | '../' | '../*' | '*' | '$'*)
			bad=$t ;;
		/bin | /boot | /dev | /etc | /home | /lib | /opt | /private | /sbin | /usr | /var | /Users | /System | /Library | /Applications | /bin/ | /etc/ | /usr/ | /var/ | /home/ | /Users/ | /System/ | /Library/)
			bad=$t ;;
		*) continue ;;
		esac
		echo "Blocked by paws rule \"rm-guard\": recursive rm on \"$bad\" (root, a system directory, home, ., .., * or a path that starts with a variable). Name the specific directory to delete instead." >&2
		exit 2
	done
done
