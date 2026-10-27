package rule

import "testing"

func TestMCPActionClassifiesRealToolNames(t *testing.T) {
	cases := map[string]string{
		// slack (plugin and bare server names)
		"mcp__plugin_slack_slack__slack_send_message":       "send",
		"mcp__plugin_slack_slack__slack_schedule_message":   "send",
		"mcp__plugin_slack_slack__slack_send_message_draft": "send",
		"mcp__slack__slack_read_channel":                    "",
		"mcp__slack__slack_add_reaction":                    "create",
		// gmail (claude.ai connector)
		"mcp__claude_ai_Gmail__send_message":   "send",
		"mcp__claude_ai_Gmail__reply":          "send",
		"mcp__claude_ai_Gmail__forward":        "send",
		"mcp__claude_ai_Gmail__create_draft":   "send",
		"mcp__claude_ai_Gmail__search_threads": "",
		"mcp__claude_ai_Gmail__trash_thread":   "delete",
		"mcp__claude_ai_Gmail__label_message":  "update",
		// notion (dashes)
		"mcp__claude_ai_Notion__notion-create-pages":   "create",
		"mcp__claude_ai_Notion__notion-update-page":    "update",
		"mcp__claude_ai_Notion__notion-create-comment": "send",
		"mcp__claude_ai_Notion__notion-get-comments":   "",
		"mcp__claude_ai_Notion__notion-delete-pages":   "delete",
		// linear
		"mcp__plugin_linear_linear__save_issue":    "create",
		"mcp__plugin_linear_linear__save_comment":  "send",
		"mcp__plugin_linear_linear__list_comments": "",
		// github, drive, calendar, vercel
		"mcp__github__create_pull_request":             "create",
		"mcp__github__add_issue_comment":               "send",
		"mcp__github__merge_pull_request":              "update",
		"mcp__google_drive__upload_file":               "create",
		"mcp__claude_ai_Google_Calendar__create_event": "create",
		"mcp__vercel__deploy_to_vercel":                "",
		// browsers: navigation is not an action class
		"mcp__playwright__browser_click":  "",
		"mcp__claude-in-chrome__navigate": "",
		// review findings: leftmost verb wins, objects don't promote
		"mcp__plugin_slack_slack__slack_create_list":         "create",
		"mcp__plugin_slack_slack__slack_update_list":         "update",
		"mcp__plugin_slack_slack__slack_add_list_record":     "create",
		"mcp__claude_ai_Notion__notion-create-view":          "create",
		"mcp__claude_ai_Gmail__create_label":                 "create",
		"mcp__gitlab__create_merge_request":                  "create",
		"mcp__context7__resolve-library-id":                  "",
		"mcp__filesystem__write_file":                        "create",
		"mcp__github__push_files":                            "update",
		"mcp__claude_ai_Notion__notion-set-comment-reaction": "update",
		"mcp__claude_ai_Google_Calendar__respond_to_event":   "",
		"mcp__x__create_schedule":                            "create",
		"mcp__fetch__http_post":                              "create",
		"mcp__claude_ai_Gmail__update_draft":                 "send",
		"mcp__x__post_message":                               "send",
		// not MCP
		"Bash":          "",
		"mcp__noserver": "",
	}
	for tool, want := range cases {
		if got := MCPAction(tool); got != want {
			t.Errorf("MCPAction(%q) = %q, want %q", tool, got, want)
		}
	}
}

func TestToolPrefixWildcard(t *testing.T) {
	r := mustRule(t, "---\nname: x\nevent: pre-tool\nmatcher:\n  tool: mcp__slack__*|mcp__plugin_slack_slack__*\naction:\n  type: audit\n---\n")
	for _, tool := range []string{"mcp__slack__slack_send_message", "mcp__plugin_slack_slack__slack_read_channel"} {
		if !r.Matches(tool, "{}") {
			t.Errorf("%s should match the slack wildcard", tool)
		}
	}
	for _, tool := range []string{"mcp__slacker__x", "mcp__gmail__send", "Bash"} {
		if r.Matches(tool, "{}") {
			t.Errorf("%s must not match the slack wildcard", tool)
		}
	}
}

func TestMCPActionMatcher(t *testing.T) {
	r := mustRule(t, "---\nname: x\nevent: pre-tool\nmatcher:\n  mcp_action: send\naction:\n  type: audit\n---\n")
	if !r.Matches("mcp__claude_ai_Gmail__send_message", "{}") {
		t.Error("send should match a gmail send")
	}
	if r.Matches("mcp__claude_ai_Gmail__search_threads", "{}") || r.Matches("Bash", "git push") {
		t.Error("send must not match reads or non-MCP tools")
	}
	multi := mustRule(t, "---\nname: x\nevent: pre-tool\nmatcher:\n  mcp_action: create|delete\naction:\n  type: audit\n---\n")
	if !multi.Matches("mcp__github__create_pull_request", "{}") || !multi.Matches("mcp__x__delete_page", "{}") || multi.Matches("mcp__x__send_message", "{}") {
		t.Error("pipe list of actions misbehaves")
	}
}

func TestFileGlob(t *testing.T) {
	cases := []struct {
		glob, path string
		want       bool
	}{
		{"**/*.{ts,tsx}", "/home/me/repo/src/a/b.tsx", true},
		{"**/*.{ts,tsx}", "/home/me/repo/b.ts", true},
		{"**/*.{ts,tsx}", "/home/me/repo/b.js", false},
		{"src/**/*.ts", "/home/me/repo/src/a.ts", true},
		{"src/**/*.ts", "/home/me/repo/src/x/y/a.ts", true},
		{"src/**/*.ts", "/home/me/repo/lib/src2/a.ts", false},
		{"*.css", "/home/me/repo/styles/a.css", true},
		{".env*", "/home/me/repo/.env.local", true},
		{".env*", "/home/me/repo/x.env", false},
		{"/etc/*", "/etc/hosts", true},
		{"/etc/*", "/home/etc/hosts", false},
		{"secrets/**", "/repo/secrets/prod/key.pem", true},
		{"a?c.md", "/r/abc.md", true},
		{"a?c.md", "/r/a/c.md", false},
		{"docs/café/*.md", "/r/docs/café/a.md", true},
	}
	for _, c := range cases {
		re, err := globRegexp(c.glob)
		if err != nil {
			t.Fatalf("globRegexp(%q): %v", c.glob, err)
		}
		if got := re.MatchString(c.path); got != c.want {
			t.Errorf("glob %q on %q = %v, want %v (re %s)", c.glob, c.path, got, c.want, re)
		}
	}
	for _, bad := range []string{"{a,b", "{a,{b}}"} {
		if _, err := globRegexp(bad); err == nil {
			t.Errorf("globRegexp(%q) should fail", bad)
		}
	}
}

func TestFileGlobOnlyMatchesFileTools(t *testing.T) {
	r := mustRule(t, "---\nname: x\nevent: pre-tool\nmatcher:\n  file_glob: \"**/*.tsx\"\naction:\n  type: audit\n---\n")
	if !r.Matches("Edit", "/repo/src/App.tsx") || !r.Matches("Write", "/repo/App.tsx") {
		t.Error("file_glob should match Edit and Write on a .tsx path")
	}
	nb := mustRule(t, "---\nname: nb\nevent: pre-tool\nmatcher:\n  file_glob: \"**/*.ipynb\"\naction:\n  type: audit\n---\n")
	if !nb.Matches("NotebookEdit", "/repo/analysis.ipynb") {
		t.Error("file_glob should match NotebookEdit on its notebook_path")
	}
	if r.Matches("Bash", "cat /repo/src/App.tsx") || r.Matches("Grep", "/repo/App.tsx") {
		t.Error("file_glob must not match tools without a file_path target")
	}
}

func TestSessionStartAndAdviseValidation(t *testing.T) {
	ok := []string{
		"---\nname: x\nevent: session-start\naction:\n  type: require-skill\n  skill: caveman\n---\n",
		"---\nname: x\nevent: session-start\nmatcher:\n  command_regex: ^(startup|compact)$\naction:\n  type: advise\n  message: hi\n---\n",
		"---\nname: x\nevent: pre-tool\nmatcher:\n  tool: mcp__playwright__*\naction:\n  type: advise\n  message: use agent-browser\n---\n",
	}
	for _, md := range ok {
		if _, err := Parse([]byte(md), "t.md"); err != nil {
			t.Errorf("should parse: %v\n%s", err, md)
		}
	}
	bad := map[string]string{
		"block on session-start":   "---\nname: x\nevent: session-start\naction:\n  type: block\n---\n",
		"tool matcher on session":  "---\nname: x\nevent: session-start\nmatcher:\n  tool: Bash\naction:\n  type: audit\n---\n",
		"advise without message":   "---\nname: x\nevent: pre-tool\naction:\n  type: advise\n---\n",
		"advise on user-prompt":    "---\nname: x\nevent: user-prompt\naction:\n  type: advise\n  message: m\n---\n",
		"unknown mcp_action":       "---\nname: x\nevent: pre-tool\nmatcher:\n  mcp_action: shout\naction:\n  type: audit\n---\n",
		"unclosed brace file_glob": "---\nname: x\nevent: pre-tool\nmatcher:\n  file_glob: \"*.{ts\"\naction:\n  type: audit\n---\n",
	}
	for name, md := range bad {
		if _, err := Parse([]byte(md), "t.md"); err == nil {
			t.Errorf("%s: should fail to parse", name)
		}
	}
}
