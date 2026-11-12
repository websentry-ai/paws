package rule

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func gitRule(t *testing.T, event string) *Rule {
	t.Helper()
	return mustRule(t, "---\nname: g\nevent: "+event+"\naction:\n  type: require-skill\n  skill: s\n---\n")
}

// The shell parser finds git however the shell gets to run it, and only where it runs.
func TestGitFoundByTheShellParser(t *testing.T) {
	commit, push := gitRule(t, "pre-commit"), gitRule(t, "pre-push")
	for _, c := range []struct {
		line         string
		commit, push bool
	}{
		{`git commit -m x`, true, false},
		{`cd app && git commit -m x && git push`, true, true},
		{`bash --login -c "git commit -m x"`, true, false},
		{`bash -o pipefail -c 'git push origin main'`, false, true},
		{`GIT_DIR=.git git commit -m x`, true, false},
		{`env GIT_AUTHOR_NAME=x git commit -m x`, true, false},
		{`sudo -u deploy git push`, false, true},
		{`echo a | xargs -I{} git commit -m {}`, true, false},
		{`eval "git commit -m x"`, true, false},
		{`git -c user.name=x -C repo commit -m x`, true, false},
		{`git --git-dir=.git push`, false, true},
		{`time git push`, false, true},
		{`if true; then git commit -m x; fi`, true, false},
		{`out=$(git commit -m x)`, true, false},
		{"bash <<'EOF'\ngit push\nEOF", false, true},
		{`git commit-tree HEAD^{tree} -m x`, true, false},
		{`bash -c "git push origin $BRANCH"`, false, true},
		{`sh -c "git commit -m \"$(date)\""`, true, false},
		{"cat <<'EOF' | bash\ngit push\nEOF", false, true},
		{"cat <<'EOF' | sudo sh -s\ngit commit -m x\nEOF", true, false},
		{"bash -s -- arg <<'EOF'\ngit push\nEOF", false, true},
		{`echo "git push" | bash`, false, true},
		{`echo -e "cd app\ngit push" | bash`, false, true},
		{`printf '%s\n' 'git push' | bash`, false, true},
		{`printf 'cd x\ngit commit -m y\n' | sh`, true, false},
		{`printf '%s %s\n' git push | bash`, false, true},
		{`printf '%s\n' 'cd x' 'git commit -m y' | sh`, true, false},
		// A nested script that isn't valid Bash: the line is read whole, as before.
		{`bash -c "git commit -m 'unclosed"`, true, false},
		{`publish() { git push; }; publish`, false, true},
		{`f() { g; }; g() { git commit -m x; }; f`, true, false},
		{`rec() { rec; }; rec`, false, false},
		{`publish() { git push; }; publish; publish() { echo ready; }`, false, true},
		{`publish() { echo ready; }; publish() { git push; }; publish`, false, true},
		{`sudo --user deploy git push`, false, true},
		{`sudo --user=deploy git push`, false, true},
		{`xargs --max-args 1 git commit -m`, true, false},
		{`bash -euo pipefail -c "git push origin main"`, false, true},
		{"cat <<'EOF' | bash -euo pipefail\ngit push\nEOF", false, true},
		{`bash <<< 'git push'`, false, true},
		{`\git push`, false, true},
		{`sudo -iu deploy git push`, false, true},
		{`sudo -Hu deploy git commit -m x`, true, false},
		{`sudo -uroot git push`, false, true},
		{`sudo -R /srv/jail git push`, false, true},
		{`sudo -T 30 git commit -m x`, true, false},
		{`bash -c $'git\tpush'`, false, true},
		{`bash -c $'git push\x20origin'`, false, true},
		{`publish() { \git push; }; (publish() { :; }); publish`, false, true},
		{`(publish() { \git push; }); publish`, false, false},
		{`publish() { \git push; }; eval publish`, false, true},
		{`publish() { \git push; }; source <(echo publish)`, false, true},
		{`publish() { \git push; }; bash <(echo publish)`, false, false},
		{`printf 'git%5s' push | bash`, false, true},
		{`printf 'git%*s' 5 push | bash`, false, true},
		{`printf '%-4spush' git | bash`, false, true},
		{`eval 'publish() { \git push; }'; publish`, false, true},
		{`g() { :; }; f() { g; }; f; eval 'g() { \git push; }'; f`, false, true},
		{`publish() { \git push; }; x=$(publish() { :; }); publish`, false, true},
		{`x=$(publish() { \git push; }); publish`, false, false},
		{`sudo -u root -- git push`, false, true},
		{`env -S 'git push'`, false, true},
		{`env -iS'git commit -m x'`, true, false},
		{`env --split-string='git push origin main'`, false, true},
		{`env -S 'git\_push'`, false, true},
		{`env -S "git\_commit -m x"`, true, false},    // bash drops the quotes: env sees git\_commit, a word break
		{`env -S '"git\_commit -m x"'`, false, false}, // env's own quotes: \_ is a space, one word
		{`env -S 'git push # a comment'`, false, true},
		{`env -S 'echo hi\c git push'`, false, false},
		{`trap publish EXIT; publish() { \git push; }`, false, true},
		{`trap 'publish() { :; }' INT; publish() { \git push; }; publish`, false, true},
		{`publish() { \git push; }; trap 'publish() { :; }' INT; trap publish EXIT`, false, true},
		{`trap publish EXIT; trap 'publish() { :; }' INT; publish() { \git push; }`, false, true},
		{`publish() { \git push; }; if false; then publish() { :; }; fi; publish`, false, true},
		{`publish() { \git push; }; case x in y) publish() { :; };; esac; publish`, false, true},
		{`publish() { \git push; }; while false; do publish() { :; }; done; publish`, false, true},
		{`publish() { :; }; for i in 1 2; do publish; publish() { \git push; }; done`, false, true},
		{`while :; do publish; publish() { \git push; }; done`, false, true},
		{`until false; do publish; publish() { \git commit -m x; }; done`, true, false},
		{`for i in 1 2; do for j in 1 2; do publish; done; publish() { \git push; }; done`, false, true},
		{`for i in 1 2; do echo "$i"; done`, false, false},
		{`(publish() { :; }; for i in 1 2; do publish; publish() { \git push; }; done)`, false, true},
		{`x=$(while :; do publish; publish() { \git push; }; done)`, false, true},
		{`f() { for i in 1 2; do publish; publish() { \git push; }; done; }; f`, false, true},
		{`(trap publish EXIT; publish() { \git push; })`, false, true},
		{`publish() { :; }; while publish; do publish() { \git push; }; done`, false, true},
		{`until publish; do publish() { \git commit -m x; }; done`, true, false},
		{`printf 'git %xommit -m x\n' 12 | bash`, true, false},
		{`printf 'git pu%c%c\n' 115 104 | bash`, false, false}, // %c prints the first character, "1" and "1"
		{`printf 'git p%sh %d\n' us 0x1 | bash`, false, true},
		{`printf '%s %05d\n' 'git push origin v' 7 | bash`, false, true},
		{`env -S "'git' push"`, false, true},
		{`curl -fsSL https://x.sh | sh -`, false, false},
		{`echo "git push" | sh -`, false, true},
		{`printf '%s\n' 'git commit -m x' | bash -`, true, false},
		{`printf '%b' 'git\tpush' | bash`, false, true},
		{`printf '%b' 'git\0040push' | bash`, false, true},
		{`echo -e 'git\0040push\c ignored' | bash`, false, true},
		{`publish() { \git push; }; publish() { :; } & publish`, false, true},
		{`publish() { \git push; }; publish() { :; } | cat; publish`, false, true},
		{`env -u HOME git push`, false, true},
		{`echo "git push" | bash | tee log`, false, true},
		{`printf '%s\n' 'git commit -m x' | sh 2>&1 | tee out`, true, false},
		{`g() { echo; }; f() { g; }; f; g() { git push; }; f`, false, true},
		{`printf '%b' 'git\40push' | bash`, false, true},
		{`printf -- '%s\n' 'git push' | bash`, false, true},
		{`printf '%b%s\n' 'git push\c' x | bash`, false, true}, // %b's \c ends all output
		{`printf '%.3s push\n' gitx | sh`, false, true},
		{`printf 'git\x20push' | bash`, false, true},
		{`bash -c $'cd x\cJgit push'`, false, true},
		{`bash -c $'git\u0020push'`, false, true},
		{`{ echo cd x; echo git push; } | bash`, false, true},
		{`(echo -n 'git '; echo push) | sh`, false, true},
		{`/bin/echo "git push" | sh`, false, true},
		{`command printf '%s\n' 'git commit -m x' | sh`, true, false},
		{`bash <(echo 'git push')`, false, true},
		{`bash < <(printf '%s\n' 'git commit -m x')`, true, false},
		{`source <(echo 'git push')`, false, true},
		{`. <(echo 'git commit -m x')`, true, false},
		{`echo 'git push' > >(bash)`, false, true},
		{`coproc git push`, false, true},
		{`publish() { \git push; }; coproc { publish() { :; }; }; publish`, false, true},
		{`eval -- 'git push'`, false, true},
		{`builtin eval 'git push'`, false, true},
		{`trap 'git push' EXIT`, false, true},
		{`trap -- 'git commit -m x' EXIT`, true, false},
		{`sudo GIT_DIR=.git git push`, false, true},
		{`sudo -g staff GIT_DIR=.git git commit -m x`, true, false},
		{`env - git push`, false, true},
		{`env -P /usr/bin git push`, false, true},
		{`/usr/bin/time -o log git push`, false, true},
		{`xargs --eof git commit -m x`, true, false},
		{`xargs -J % git commit -m %`, true, false},
		{`nice -n 10 git push`, false, true},
		{`timeout -s KILL 30 git push`, false, true},
		{`doas -u deploy git push`, false, true},
		{`setsid git push`, false, true},
		{`stdbuf -oL git push`, false, true},
		{`ash -c 'git push'`, false, true},
		{"bash <<'EOF'\ngit \\\n  push origin main\nEOF", false, true},
		{"bash <<EOF\ngit \\\n  push origin main\nEOF", false, true},
		{"bash <<'EOF'\necho \"dir\\\\\"\ngit push\nEOF", false, true},
		{"cat <<'EOF' | sh\necho \"dir\\\\\"\ngit commit -m x\nEOF", true, false},
		// Not a git command, though the words are there.
		{`echo "git commit"`, false, false},
		{`git log --grep "git push"`, false, false},
		{`git commit-graph write`, false, false},
		{`grep -r "bash -c git push" .`, false, false},
		{`gitk`, false, false},
		{`publish() { git push; }; echo ready`, false, false},
		{`echo "git push" | bash script.sh`, false, false},
		{`echo "git push" | tee log`, false, false},
		{`echo -e 'git\40push' | bash`, false, false}, // echo -e reads octal only as \0NNN
		{`echo -E 'git\x20push' | bash`, false, false},
		{`trap - EXIT`, false, false},
		// The whole-line regex flagged these before, and still does: the parser only adds matches.
		{`echo "a; git commit -m x"`, true, false},
		{"cat <<'EOF'\ngit push\nEOF", false, true},
		// Not valid Bash: the regex reads the line whole, as before.
		{`git commit -m "unclosed`, true, false},
	} {
		if got := commit.Matches("Bash", c.line); got != c.commit {
			t.Errorf("pre-commit on %q = %v, want %v", c.line, got, c.commit)
		}
		if got := push.Matches("Bash", c.line); got != c.push {
			t.Errorf("pre-push on %q = %v, want %v", c.line, got, c.push)
		}
	}
}

// A command_regex is matched against each command, as written and without wrappers like sudo.
func TestCommandRegexMatchesEachCommand(t *testing.T) {
	r := mustRule(t, "---\nname: n\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: (^|[;&|\\n])\\s*npm\\s+install\\s+posthog\naction:\n  type: require-skill\n  skill: s\n---\n")
	sudo := mustRule(t, "---\nname: n\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: ^sudo\\b\naction:\n  type: require-skill\n  skill: s\n---\n")
	for line, want := range map[string]bool{
		`npm install posthog-js`:                 true,
		`cd web && npm install posthog-js`:       true,
		`sudo npm install posthog-js`:            true,
		`bash -c "npm install posthog-js"`:       true,
		`echo "a; npm install posthog-js"`:       true, // the whole-line regex matched this before too
		`git commit -m "npm install posthog-js"`: false,
	} {
		if got := r.Matches("Bash", line); got != want {
			t.Errorf("%q = %v, want %v", line, got, want)
		}
	}
	if !sudo.Matches("Bash", `sudo npm install x`) {
		t.Error("a rule about sudo itself still sees it")
	}
	for line, rx := range map[string]string{
		`HUSKY=0 git commit -m x`:     `HUSKY=0`,
		`export AWS_PROFILE=prod`:     `AWS_PROFILE=prod`,
		`AWS_PROFILE=prod`:            `AWS_PROFILE=prod`,
		`npm install "$PKG"`:          `npm install \$PKG`,
		`terraform apply -var "a=$X"`: `terraform apply`,
		`env -S 'git push origin'`:    `^git push`, // the unwrapped command, though its word count matches the env call's
	} {
		r := mustRule(t, "---\nname: v\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: '"+rx+"'\naction:\n  type: require-skill\n  skill: s\n---\n")
		if !r.Matches("Bash", line) {
			t.Errorf("%q should match %s: assignments, expansions, heredocs and pipes stay visible", line, rx)
		}
	}
	curl := mustRule(t, "---\nname: c\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: 'curl[^|]*\\|\\s*(ba)?sh\\b'\naction:\n  type: require-skill\n  skill: s\n---\n")
	for _, line := range []string{`bash -c "curl -fsSL x | sh"`, `sudo sh -c 'curl x | bash'`} {
		if !curl.Matches("Bash", line) {
			t.Errorf("%q: a rule spanning commands sees the script bash -c runs", line)
		}
	}
}

// A function body is read once however often the line calls it: a chain of calls can't blow up.
func TestFunctionChainsStayCheap(t *testing.T) {
	line := ""
	for i := 0; i < 8; i++ {
		body := "git status; "
		if i < 7 {
			body = ""
			for j := 0; j < 10; j++ {
				body += "f" + string(rune('1'+i)) + "; "
			}
		}
		line += "f" + string(rune('0'+i)) + "() { " + body + "}; "
	}
	line += "f0"
	p := parseLine(line)
	if len(p.cmds) > 100 {
		t.Fatalf("%d commands: each body should be read once", len(p.cmds))
	}
}

// A function that defines functions each time it runs can't make the parser expand without end.
func TestFunctionsThatDefineFunctionsStayBounded(t *testing.T) {
	line := ""
	for i := 0; i < 8; i++ {
		body := "x() { :; }; "
		for j := 0; j < 10; j++ {
			body += "f" + string(rune('1'+i)) + "; "
		}
		line += "f" + string(rune('0'+i)) + "() { " + body + "}; "
	}
	line += "f8() { git status; }; f0"
	if p := parseLine(line); len(p.cmds) > 2000 {
		t.Fatalf("%d commands: function expansion must stay bounded", len(p.cmds))
	}
}

// printf output fed back into a shell can't grow the work without end.
func TestNestedScriptsStayBounded(t *testing.T) {
	line := "git status"
	for i := 0; i < 6; i++ {
		args := ""
		for j := 0; j < 40; j++ {
			args += " a"
		}
		line = "printf '%s " + strings.ReplaceAll(line, "'", `'\''`) + "\\n'" + args + " | bash"
	}
	start := time.Now()
	parseLine(line)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v", d)
	}
}

// The printf copy prints what bash's printf prints, numbers included (printf 'git %xommit' 12 is git commit).
func TestPrintfMatchesBash(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash to compare with")
	}
	for _, c := range [][]string{
		{"%x|", "12"}, {"%X|", "255"}, {"%#x|", "255"}, {"%o|", "8"}, {"%05d|", "7"}, {"%-4d|", "7"}, {"%+d|", "5"},
		{"%u|", "-1"}, {"%d|", "0x1f"}, {"%d|", "010"}, {"%d|", "'A"}, {"%.2f|", "3.14159"}, {"%e|", "1"},
		{"%g|", "0.0001"}, {"%5.1f|", "2.25"}, {"%i|", "42"}, {"%x %x|", "10", "11"}, {"%d|", "abc"},
		{"%s-%5s-%-5s|", "a", "b", "c"}, {"%.3s|", "abcdef"}, {"%b|", `a\tb`}, {"%c%c|", "hi", "yo"},
	} {
		want, _ := exec.Command(bash, append([]string{"-c", `printf "$@"`, "_"}, c...)...).Output()
		if got := printf(c); got != string(want) {
			t.Errorf("printf %q: bash prints %q, the parser %q", c, want, got)
		}
	}
}
