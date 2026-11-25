package rule

import "testing"

func regexRule(t *testing.T, rx string) *Rule {
	t.Helper()
	return mustRule(t, "---\nname: r\nevent: pre-tool\nmatcher:\n  tool: Bash\n  command_regex: '"+rx+"'\naction:\n  type: require-skill\n  skill: s\n---\n")
}

// A heredoc body that no shell runs is text: what it says can't trigger a rule. The command reading it still does.
func TestHeredocBodyNoShellRuns(t *testing.T) {
	rm := regexRule(t, `(^|[;&|\n])\s*rm\s+-rf\b`)
	commit, push := gitRule(t, "pre-commit"), gitRule(t, "pre-push")
	for _, line := range []string{
		"cat <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat <<'EOF' > notes.md\ngit push\nrm -rf build\nEOF",
		"cat > notes.md <<EOF\ngit push\nrm -rf build\nEOF",
		"cat >> notes.md <<-'EOF'\n\tgit push\n\trm -rf build\n\tEOF",
		"cd docs && cat > notes.md <<'EOF'\ngit push\nrm -rf build\nEOF\nwc -l notes.md",
		"tee notes.md <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat > notes.md 2>&1 <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat > /dev/null <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat > a.md <<'A'\ngit push\nA\ncat >> a.md <<'B'\nrm -rf build\nB\nwc -l a.md",
		"cat > notes.md <<'EOF'\ngit push\nrm -rf build\nEOF\ngh pr comment 7 --body-file notes.md",
		"cat > notes.md <<'EOF'\ngit push\nrm -rf build\nEOF\ncat notes.md && git add notes.md",
		"cat 3> notes.md 1>&3 <<'EOF'\ngit push\nrm -rf build\nEOF",
		"sudo tee /etc/notes >/dev/null <<'EOF'\ngit push\nrm -rf build\nEOF",
		"> notes.md <<'EOF'\ngit push\nrm -rf build\nEOF",
		"gh pr comment 7 --body-file - <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat <<'EOF' | gh pr comment 7 --body-file -\ngit push\nrm -rf build\nEOF",
		"cat <<'EOF' | tee notes.md\ngit push\nrm -rf build\nEOF",
		"gh pr comment 7 --body \"$(cat <<'EOF'\ngit push\nrm -rf build\nEOF\n)\"",
		"echo \"$(cat <<'EOF'\ngit push\nrm -rf build\nEOF\n)\" > notes.md",
		"if true; then cat <<'EOF' > notes.md\ngit push\nrm -rf build\nEOF\nfi",
		"cat <<'A' > a.md && cat <<'B' > b.md\ngit push\nA\nrm -rf build\nB",
		"git tag -a v1 -F - <<'EOF'\ngit push\nrm -rf build\nEOF",
		"gh api repos/o/r/issues -F body=@- <<'EOF'\ngit push\nrm -rf build\nEOF",
		"cat > notes.md <<'EOF'\nrm -rf build\nEOF\ngit add notes.md && wc -l notes.md",
	} {
		if rm.Matches("Bash", line) || commit.Matches("Bash", line) || push.Matches("Bash", line) {
			t.Errorf("%q: a body no shell runs fired a rule", line)
		}
	}

	// The command reading the body still fires.
	for _, c := range []struct {
		rule *Rule
		line string
	}{
		{commit, "git commit -F - <<'EOF'\nfix: x\n\ngit push and rm -rf build later\nEOF"},
		{commit, "git commit -m \"$(cat <<'EOF'\nfix: x\n\ngit push\nEOF\n)\""},
		{commit, "cat <<'EOF' | git commit -F -\nfix: x\nEOF"},
		{regexRule(t, `(^|[;&|\n])\s*gh\s+pr\s+create\b`), "gh pr create --title t --body-file - <<'EOF'\nrm -rf build\nEOF"},
		{regexRule(t, `(^|[;&|\n])\s*(sudo\s+)?tee\s+/etc/`), "sudo tee /etc/notes <<'EOF'\nx\nEOF"},
		{regexRule(t, `(^|[;&|\n])\s*cat\s+<<`), "cat <<'EOF' > notes.md\nx\nEOF"},
	} {
		if !c.rule.Matches("Bash", c.line) {
			t.Errorf("%q: the command itself should still fire", c.line)
		}
	}
	for _, line := range []string{
		"git commit -F - <<'EOF'\nfix: x\n\ngit push\nEOF",
		"gh pr create --body-file - <<'EOF'\ngit push\nEOF",
	} {
		if push.Matches("Bash", line) {
			t.Errorf("%q: git push is only in the body", line)
		}
	}
}

// A body that a shell, an interpreter or an unknown command reads keeps matching as before.
func TestHeredocBodyAShellRuns(t *testing.T) {
	rm := regexRule(t, `(^|[;&|\n])\s*rm\s+-rf\b`)
	push := gitRule(t, "pre-push")
	for _, line := range []string{
		"bash <<'EOF'\nrm -rf build\nEOF",
		"sh -s <<'EOF'\nrm -rf build\nEOF",
		"zsh <<EOF\nrm -rf build\nEOF",
		"sudo bash -euo pipefail <<'EOF'\nrm -rf build\nEOF",
		"ssh host bash <<'EOF'\nrm -rf build\nEOF",
		"ssh host <<'EOF'\nrm -rf build\nEOF",
		"ssh -T host 'bash -s' <<'EOF'\nrm -rf build\nEOF",
		"xargs sh <<'EOF'\nrm -rf build\nEOF",
		"cat <<'EOF' | bash\nrm -rf build\nEOF",
		"cat <<'EOF' | tee run.log | sh\nrm -rf build\nEOF",
		"{ cat <<'EOF'\nrm -rf build\nEOF\n} | bash",
		"bash -c \"$(cat <<'EOF'\nrm -rf build\nEOF\n)\"",
		"eval \"$(cat <<'EOF'\nrm -rf build\nEOF\n)\"",
		"bash <(cat <<'EOF'\nrm -rf build\nEOF\n)",
		"cat <<'EOF' > >(bash)\nrm -rf build\nEOF",
		"cat > /tmp/log > >(sh) <<'EOF'\nrm -rf build\nEOF",
		"cat > notes.md 3> >(sh) 1>&3 <<'EOF'\nrm -rf build\nEOF",
		"tee >(sh) <<'EOF'\nrm -rf build\nEOF",
		"cat() { sh; }; cat <<'EOF'\nrm -rf build\nEOF",
		"git() { sh; }; git commit -F - <<'EOF'\nrm -rf build\nEOF",
		"eval \"$DEFS\"; cat <<'EOF'\nrm -rf build\nEOF",
		"source ./defs.sh; cat > notes.md <<'EOF'\nrm -rf build\nEOF",
		"cat > /dev/stdout <<'EOF' | sh\nrm -rf build\nEOF",
		"cat > /dev/fd/1 <<'EOF' | sh\nrm -rf build\nEOF",
		"git stripspace <<'EOF' | sh\nrm -rf build\nEOF",
		"cat > Makefile <<'EOF'\nall:\n\trm -rf build\nEOF\nmake",
		"cat > build.mk <<'EOF'\nall:\n\trm -rf build\nEOF\nmake -f build.mk",
		"cat > package.json <<'EOF'\nrm -rf build\nEOF\nnpm run x",
		"cat > ~/.zshrc <<'EOF'\nrm -rf build\nEOF",
		"tee /dev/fd/3 3> >(sh) <<'EOF'\nrm -rf build\nEOF",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\ncat t.sh | sh",
		"git apply <<'EOF'\nrm -rf build\nEOF",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\nwc \"$(sh t.sh)\"",
		"cat 1<>t.sh <<'EOF'\nrm -rf build\nEOF\nsh t.sh",
		"cat > task.command <<'EOF'\nrm -rf build\nEOF\nopen task.command",
		"F=t.sh; cat > \"$F\" <<'EOF'\nrm -rf build\nEOF\nsh /tmp/t.sh",
		"cat > main.go <<'EOF'\nrm -rf build\nEOF\ngo run .",
		"f() { :; }; cat > notes.md <<'EOF'\nrm -rf build\nEOF",
		"cat > notes.md <<'EOF'\nrm -rf build\nEOF\nls | xargs", // xargs alone runs echo
		"./cat <<'EOF'\nrm -rf build\nEOF",
		"./sudo cat <<'EOF'\nrm -rf build\nEOF",
		"/usr/bin/env cat <<'EOF'\nrm -rf build\nEOF",
		"cat > filter.jq <<'EOF'\nrm -rf build\nEOF\njq -f filter.jq input.json",
		"/tmp/git commit -F - <<'EOF'\nrm -rf build\nEOF",
		"GIT_EDITOR=sh git commit -e -F - <<'EOF'\nrm -rf build\nEOF",
		"git commit --edit -F - <<'EOF'\nrm -rf build\nEOF",
		"git commit -eF - <<'EOF'\nrm -rf build\nEOF",
		"git -ccore.editor=sh commit -e -F - <<'EOF'\nrm -rf build\nEOF",
		"env GIT_EDITOR=sh git commit -F - <<'EOF'\nrm -rf build\nEOF",
		"git -c core.editor=sh commit -e -m \"$(cat <<'EOF'\nrm -rf build\nEOF\n)\"",
		"printf -v script '%s' \"$(cat <<'EOF'\nrm -rf build\nEOF\n)\"; $script",
		"cat > notes.md <<'EOF'\nrm -rf build\nEOF\nGIT_SSH_COMMAND='sh notes.md' git push",
		"cat > notes.md <<'EOF'\nrm -rf build\nEOF\ngit -c core.sshCommand='sh notes.md' push",
		"cat > notes.md <<'EOF'\nrm -rf build\nEOF\ngit -ccore.sshCommand='sh notes.md' push",
		"T=$S/t.sh; cat > $T <<'EOF'\nrm -rf build\nEOF\n$T",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\ncp t.sh bin/run && bin/run",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\nbash -c 'sh t.sh'",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\neval './t.sh'",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\nbash -c \"bash -c './t.sh'\"",
		"tee notes.md >(sh) <<'EOF' >/dev/null\nrm -rf build\nEOF",
		"cat <<'EOF' >&2\nrm -rf build\nEOF", // stderr: paws doesn't follow it
		"git -c 'alias.run=!sh' run <<'EOF'\nrm -rf build\nEOF",
		"gh deploy <<'EOF'\nrm -rf build\nEOF", // a gh alias may run a shell
		"script=$(cat <<'EOF'\nrm -rf build\nEOF\n)",
		"f() { cat <<'EOF'\nrm -rf build\nEOF\n}; f | sh",
		"docker exec -i box sh <<'EOF'\nrm -rf build\nEOF",
		"kubectl exec -i pod -- bash <<'EOF'\nrm -rf build\nEOF",
		"make -f - <<'EOF'\nall:\n\trm -rf build\nEOF",
		"while read d; do echo \"$d\"; done <<'EOF'\nrm -rf build\nEOF",
		// A script the line writes and then runs, sources or makes executable.
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\nchmod +x t.sh && ./t.sh",
		"cat > \"$S/t.sh\" <<'EOF'\nrm -rf build\nEOF\nbash \"$S/t.sh\"",
		"cat <<'EOF' > t.sh; . t.sh\nrm -rf build\nEOF",
		"tee t.sh <<'EOF' >/dev/null\nrm -rf build\nEOF\nsh t.sh",
		"cat > $S/bin/run.sh <<'EOF'\nrm -rf build\nEOF\n$S/bin/run.sh",
		"cat > t.sh <<'EOF'\nrm -rf build\nEOF\n./t.sh",
		"cat > ./t.sh <<'EOF'\nrm -rf build\nEOF\nbash t.sh",
		// $( ) in an unquoted body runs, even when the body itself is only written to a file.
		"cat > notes.md <<EOF\nbuilt at $(rm -rf build)\nEOF",
	} {
		if !rm.Matches("Bash", line) {
			t.Errorf("%q: a body that may run should still fire", line)
		}
	}
	if !push.Matches("Bash", "bash <<'EOF'\ngit push\nEOF") || !push.Matches("Bash", "cat <<'EOF' | sh\ngit push\nEOF") {
		t.Error("git push in a body fed to a shell should still fire pre-push")
	}
	// xargs makes the body the arguments of the command it runs.
	for _, line := range []string{"xargs git <<'EOF'\npush origin main\nEOF", "cat <<'EOF' | xargs -n3 git\npush origin main\nEOF"} {
		if !regexRule(t, `(^|\n)push origin`).Matches("Bash", line) {
			t.Errorf("%q: the body is the arguments xargs passes to git", line)
		}
	}

	// Code written to a file that an interpreter then runs stays visible too.
	for _, line := range []string{
		"cat > task.py <<'EOF'\nimport shutil; shutil.rmtree(\"build\")\nEOF\npython3 task.py",
		"cat > ./task.py <<'EOF'\nimport shutil; shutil.rmtree(\"build\")\nEOF\npython3 < task.py",
		"tee main.go <<'EOF' >/dev/null\nshutil.rmtree\nEOF\ngo run main.go",
		"cat > $S/main.go <<'EOF'\nshutil.rmtree\nEOF\ncd $S && go run main.go",
	} {
		if !regexRule(t, `shutil\.rmtree`).Matches("Bash", line) {
			t.Errorf("%q: an interpreter runs the file", line)
		}
	}
	if regexRule(t, `shutil\.rmtree`).Matches("Bash", "cat > task.py <<'EOF'\nimport shutil; shutil.rmtree(\"build\")\nEOF\ngit add task.py") {
		t.Error("a file only added to git runs nothing")
	}

	// Interpreters keep their body visible: a rule may name the SQL or code inside.
	for line, rx := range map[string]string{
		"psql \"$DATABASE_URL\" <<'SQL'\nDROP TABLE users;\nSQL":                `DROP\s+TABLE`,
		"python3 - <<'PY'\nimport shutil; shutil.rmtree(\"build\")\nPY":         `shutil\.rmtree`,
		"node <<'JS'\nrequire(\"fs\").rmSync(\"build\", {recursive: true})\nJS": `rmSync`,
		"mysql -h prod <<'SQL'\ndelete from users;\nSQL":                        `(?i)delete\s+from`,
	} {
		if !regexRule(t, rx).Matches("Bash", line) {
			t.Errorf("%q should match %s", line, rx)
		}
	}
}

func TestDataTextCutsOnlyTheBody(t *testing.T) {
	for line, want := range map[string]string{
		"cat <<'EOF' > notes.md\ngit push\nEOF\ngit status": "cat <<'EOF' > notes.md\n\nEOF\ngit status",
		"cat <<EOF > notes.md\nat $(date) push\nEOF":        "cat <<EOF > notes.md\n$(date)\nEOF",
		"cat <<-EOF\n\tat $(date) push\n\tEOF":              "cat <<-EOF\n$(date)\n\tEOF",
		"bash <<'EOF'\ngit push\nEOF":                       "bash <<'EOF'\ngit push\nEOF",
		"git commit -m 'unclosed":                           "git commit -m 'unclosed", // doesn't parse: read whole
	} {
		if _, got := parseLine(line); got != want {
			t.Errorf("parseLine(%q) text = %q, want %q", line, got, want)
		}
	}
}
