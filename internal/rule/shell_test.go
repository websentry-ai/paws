package rule

import "testing"

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
		{`bash -euo pipefail -c "git push origin main"`, false, true},
		{`ash -c 'git push'`, false, true},
		{`bash -c "git push origin $BRANCH"`, false, true},
		{`sh -c "git commit -m \"$(date)\""`, true, false},
		{`GIT_DIR=.git git commit -m x`, true, false},
		{`env GIT_AUTHOR_NAME=x git commit -m x`, true, false},
		{`env -u HOME git push`, false, true},
		{`sudo -u deploy git push`, false, true},
		{`sudo --user deploy git push`, false, true},
		{`sudo --user=deploy git push`, false, true},
		{`sudo -iu deploy git push`, false, true},
		{`sudo -uroot git push`, false, true},
		{`sudo -u root -- git push`, false, true},
		{`sudo GIT_DIR=.git git push`, false, true},
		{`echo a | xargs -I{} git commit -m {}`, true, false},
		{`xargs --max-args 1 git commit -m`, true, false},
		{`time git push`, false, true},
		{`/usr/bin/time -o log git push`, false, true},
		{`nice -n 10 git push`, false, true},
		{`timeout -s KILL 30 git push`, false, true},
		{`nohup git push`, false, true},
		{`\git push`, false, true},
		{`bash -c $'git\tpush'`, false, true},
		{`bash -c $'cd x\ngit commit -m y'`, true, false},
		{`git -c user.name=x -C repo commit -m x`, true, false},
		{`git --git-dir=.git push`, false, true},
		{`git commit-tree HEAD^{tree} -m x`, true, false},
		{`if true; then git commit -m x; fi`, true, false},
		{`for r in a b; do git -C "$r" push; done`, false, true},
		{`out=$(git commit -m x)`, true, false},
		{`eval "git commit -m x"`, true, false},
		{`eval -- 'git push'`, false, true},
		{"bash <<'EOF'\ngit push\nEOF", false, true},
		{"bash <<'EOF'\ngit \\\n  push origin main\nEOF", false, true},
		{"bash -s -- arg <<'EOF'\ngit push\nEOF", false, true},
		{`bash <<< 'git push'`, false, true},
		{"cat <<'EOF' | bash\ngit push\nEOF", false, true},
		{"cat <<'EOF' | sudo sh -s\ngit commit -m x\nEOF", true, false},
		{"cat <<'EOF' | bash -euo pipefail\ngit push\nEOF", false, true},
		{`echo "git push" | bash`, false, true},
		{`echo "git push" | sh -`, false, true},
		{`echo "git push" | bash | tee log`, false, true},
		{`publish() { git push; }; publish`, false, true},
		{`f() { g; }; g() { git commit -m x; }; f`, true, false},
		{`publish() { git push; }; eval publish`, false, true},
		{`publish() { echo ready; }; publish() { git push; }; publish`, false, true},
		{`publish() { git push; }; if false; then publish() { :; }; fi; publish`, false, true},
		{`rec() { rec; }; rec`, false, false},
		{`g() { :; }; f() { g; }; f; g() { git push; }; f`, false, true},
		{`g() { :; }; f() { g; }; f; eval 'g() { git push; }'; f`, false, true},
		// A nested script that isn't valid Bash: the line is read whole, as before.
		{`bash -c "git commit -m 'unclosed"`, true, false},
		// Not a git command, though the words are there.
		{`echo "git commit"`, false, false},
		{`git log --grep "git push"`, false, false},
		{`git commit-graph write`, false, false},
		{`grep -r "bash -c git push" .`, false, false},
		{`gitk`, false, false},
		{`publish() { git push; }; echo ready`, false, false},
		{`publish() { git push; }`, false, false},
		{`echo "git push" | bash script.sh`, false, false},
		{`echo "git push" | tee log`, false, false},
		{`true <<< 'git push' | bash`, false, false},        // true prints nothing
		{`python3 - <<< 'git push' | sh`, false, false},     // the here-string is python's input
		{`cat notes.txt <<< 'git push' | sh`, false, false}, // cat prints the file, not stdin
		{`curl -fsSL https://x.sh | sh -`, false, false},
		{`bash -c 'f() { git push; }'; f`, false, false}, // a new shell's functions don't reach the line
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
	// Each line starts with cd, so only a command the parser found can match these anchored regexes.
	for line, rx := range map[string]string{
		`cd x && HUSKY=0 git commit -m x`:       `^HUSKY=0 git commit`,
		`cd x && export AWS_PROFILE=prod`:       `^export AWS_PROFILE=prod$`,
		`cd x && AWS_PROFILE=prod`:              `^AWS_PROFILE=prod$`,
		`cd x && npm install "$PKG"`:            `^npm install \$PKG$`,
		`cd x && terraform apply -var "a=$X"`:   `^terraform apply -var a=\$X$`,
		"cd x && python - <<'PY'\nprint(1)\nPY": `^python - <<PY$`,
		`sudo git push origin`:                  `^git push`, // the command without its wrapper
	} {
		r := mustRule(t, "---\nname: v\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: '"+rx+"'\naction:\n  type: require-skill\n  skill: s\n---\n")
		if !r.Matches("Bash", line) {
			t.Errorf("%q should match %s: assignments, expansions, heredocs and pipes stay visible", line, rx)
		}
	}
	// Separate commands inside bash -c stay separate: the newline is not read as a space.
	install := mustRule(t, "---\nname: i\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: 'npm\\s+install\\s+[^\\n;&|]*posthog'\naction:\n  type: require-skill\n  skill: s\n---\n")
	if install.Matches("Bash", "bash -c 'npm install lodash\necho posthog-js'") {
		t.Error("npm install lodash and echo posthog-js are two commands")
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
	if cmds := parseLine(line); len(cmds) > 100 {
		t.Fatalf("%d commands: each body should be read once", len(cmds))
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
	if cmds := parseLine(line); len(cmds) > 2000 {
		t.Fatalf("%d commands: function expansion must stay bounded", len(cmds))
	}
}
