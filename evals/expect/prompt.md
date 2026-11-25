You are writing paws `expect` checks for the rule files of one skill, following ONLY the paws skill's guide and the skill text in this folder.

Read first, in order:
1. guide/SKILL.md (the paws skill)
2. guide/expect.md (the guide for writing checks; follow it exactly)
3. guide/rules.md, section "## expect"

Inputs:
- skills/<skill>.md: the skill's text, as installed.
- bindings/<file>.md: the skill's rule files, with no expect block.

Output, in out/:
- the same file name for each rule file in bindings/, with an `expect:` block added at the end of the frontmatter and nothing else changed. If the skill has nothing a tool call can show for that rule file, copy the file unchanged and say why in the notes.
- out/notes.md: for each rule file, the list of firm rules you made (guide §1), then one line per check: its id, the skill sentence it measures, and one passing and one failing call.
- out/trace.jsonl, one JSON object per line:
  - per check: {"rule_file":"<file>","id":"<check id>","sentence":"<the skill sentence it measures, copied exactly from skills/<skill>.md>"}
  - per firm rule on your list that got no check: {"rule_file":"<file>","no_check":"<the skill sentence, copied exactly>","why":"<short reason>"}
  - per rule file left with no checks: {"rule_file":"<file>","none":"<why>"}

Test every check with the paws binary in a scratch folder:
  P=$PWD/try/$RANDOM; mkdir -p $P/home $P/proj/.paws/pre-tool; cp out/<file> $P/proj/.paws/pre-tool/
  cd $P/proj && PAWS_HOME=$P/home <this folder>/paws explain '<command>'   (for MCP, Write or Edit: explain --tool <name> '<json input>')
A check that doesn't compile shows as a "! problem in" line. Fix it.

Use only the files in this folder. Finish with a short summary: files written, the checks per file, and any rule file you left without checks and why.
