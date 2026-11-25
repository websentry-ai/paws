// casegrade grades labelled cases against a folder of rule files with paws's own outcome.Grade. Each case is a
// short turn (the blocked call and what came next) labelled "follows" or "breaks". A breaks case is graded right when
// a check fails; a follows case when none does. Part of the expect eval (evals/expect), not shipped in paws.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/websentry-ai/paws/internal/hook"
	"github.com/websentry-ai/paws/internal/outcome"
	"github.com/websentry-ai/paws/internal/rule"
)

type callIn struct {
	Tool  string         `json:"tool"`
	Input map[string]any `json:"input"`
}

type caseIn struct {
	ID       string   `json:"id"`
	Skill    string   `json:"skill"`
	RuleFile string   `json:"rule_file"`
	Label    string   `json:"label"` // follows | breaks
	Sentence string   `json:"sentence"`
	Calls    []callIn `json:"calls"`
	Blocked  int      `json:"blocked"`
}

type caseOut struct {
	caseIn
	Status  string          `json:"status"` // ok | no-rule | no-match | bad
	Outcome string          `json:"outcome,omitempty"`
	Checks  map[string]bool `json:"checks,omitempty"`
	Counted map[string]bool `json:"counted,omitempty"` // checks the verdict uses
	Verdict string          `json:"verdict,omitempty"` // pass | fail | none
	Correct bool            `json:"correct"`
}

func main() {
	dir := flag.String("rules", "", "rule files")
	in := flag.String("cases", "", "cases.jsonl")
	out := flag.String("out", "", "result jsonl")
	lint := flag.Bool("lint", false, "print each rule file's checks and load errors as JSON lines, then exit")
	flag.Parse()
	if *lint {
		files, _ := filepath.Glob(filepath.Join(*dir, "*.md"))
		for _, f := range files {
			data, _ := os.ReadFile(f)
			r, err := rule.Parse(data, f)
			o := map[string]any{"file": filepath.Base(f)}
			if err != nil {
				o["parse_err"] = err.Error()
			} else {
				var ids []string
				labels := map[string]string{}
				for _, e := range r.Expect {
					ids = append(ids, e.ID+":"+e.When)
					labels[e.ID] = e.Label
				}
				o["checks"] = ids
				o["labels"] = labels
				if r.ExpectErr != nil {
					o["expect_err"] = r.ExpectErr.Error()
				}
			}
			b, _ := json.Marshal(o)
			fmt.Println(string(b))
		}
		return
	}
	f, err := os.Open(*in)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	rules := map[string]*rule.Rule{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for sc.Scan() {
		var c caseIn
		if json.Unmarshal(sc.Bytes(), &c) != nil {
			continue
		}
		o := caseOut{caseIn: c}
		r, ok := rules[c.RuleFile]
		if !ok {
			data, err := os.ReadFile(filepath.Join(*dir, c.RuleFile))
			if err == nil {
				r, _ = rule.Parse(data, c.RuleFile)
			}
			rules[c.RuleFile] = r
		}
		switch {
		case r == nil:
			o.Status = "no-rule"
		case c.Blocked < 0 || c.Blocked >= len(c.Calls):
			o.Status = "bad"
		default:
			var calls []outcome.Call
			for i, ci := range c.Calls {
				k := hook.CallOf(ci.Tool, ci.Input)
				k.At = base.Add(time.Duration(i) * time.Minute)
				if i == c.Blocked {
					k.Denied = true
				}
				calls = append(calls, k)
			}
			b := calls[c.Blocked]
			if !r.Matches(b.Tool, b.Target) {
				o.Status = "no-match"
				break
			}
			o.Status = "ok"
			g := outcome.Grade(r, calls, c.Blocked)
			o.Outcome, o.Checks = g.Outcome, g.Checks
			// With no retry, retry/before/after checks fail by construction (paws grades the block "other route"):
			// only never checks say anything then.
			when := map[string]string{}
			for _, e := range r.Expect {
				when[e.ID] = e.When
			}
			o.Counted = map[string]bool{}
			for id, pass := range g.Checks {
				if g.Retry == nil && when[id] != rule.ExpectNever {
					continue
				}
				o.Counted[id] = pass
			}
			o.Verdict = "none"
			if len(o.Counted) > 0 {
				o.Verdict = "pass"
				for _, p := range o.Counted {
					if !p {
						o.Verdict = "fail"
					}
				}
			}
			if c.Label == "breaks" {
				o.Correct = o.Verdict == "fail"
			} else {
				o.Correct = o.Verdict != "fail"
			}
		}
		b, _ := json.Marshal(o)
		fmt.Fprintln(w, string(b))
	}
}
