package history

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/websentry-ai/paws/internal/gate"
	"github.com/websentry-ai/paws/internal/rule"
)

// Signature is a call's coarse shape, safe to send. Every word in it comes from a fixed list paws ships or from the
// catalog, never from the user: an alias, a plugin, a script, a server or file type they named is counted as other.
//
//	Bash               a known program, plus the CLI's own subcommands ("kubectl delete", "aws s3api create-bucket",
//	                   "rm"); a SQL verb (DELETE, DROP...); "bash other" for anything else
//	Read/Write/Edit    "read" or "edit" plus a known file extension ("edit .tf")
//	mcp__server__tool  "mcp <server> <tool>" for a server the catalog names; else "mcp other <create|update|...>"
//	other tools        a Claude Code tool's name in lower case, else "other"
func (c *Catalog) Signature(tool, target string) string {
	switch {
	case tool == "Bash":
		return "bash " + bashSignature(target)
	case fileTools[tool]:
		verb := "edit"
		if tool == "Read" {
			verb = "read"
		}
		if ext := strings.ToLower(filepath.Ext(target)); extensions[ext] {
			return verb + " " + ext
		}
		return verb
	case strings.HasPrefix(tool, "mcp__"):
		return "mcp " + c.mcpSignature(tool)
	case builtinTools[tool]:
		return strings.ToLower(tool)
	}
	return "other"
}

func (c *Catalog) mcpSignature(tool string) string {
	raw, _, _ := strings.Cut(strings.TrimPrefix(tool, "mcp__"), "__")
	server, name, ok := gate.MCPParts(tool)
	if ok && c.knownServer(raw) && wordRe.MatchString(server) && wordRe.MatchString(name) {
		return server + " " + name // a server the catalog names: its tool names are the vendor's
	}
	class := rule.MCPAction(tool)
	if class == "" {
		class = "read"
	}
	return "other " + class
}

// knownServer reports whether one of the catalog's MCP server patterns covers raw, ignoring case. A pattern that
// is only *, which covers any server, doesn't count.
func (c *Catalog) knownServer(raw string) bool {
	raw = strings.ToLower(raw)
	for _, p := range c.MCPServers {
		if strings.Trim(p, "*") != "" && rule.MatchToolSpec(strings.ToLower(p), raw) {
			return true
		}
	}
	return false
}

// MCPServers returns the server parts of the MCP patterns in a matcher tool spec ("mcp__attio__create-note|Bash" is
// ["attio"]).
func MCPServers(spec string) []string {
	var out []string
	for _, p := range strings.Split(spec, "|") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(p), "mcp__"); ok {
			if server, _, _ := strings.Cut(rest, "__"); server != "" {
				out = append(out, server)
			}
		}
	}
	return out
}

var (
	fileTools = map[string]bool{"Read": true, "Write": true, "Edit": true, "NotebookEdit": true}
	wordRe    = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	// sqlVerb finds a destructive SQL statement anywhere in a command, as the gate's action keys do.
	sqlVerb   = regexp.MustCompile(`(?i:\b(delete)\s+from\b|\b(drop)\s+(?:table|database|schema|index|view)\b|\b(truncate)\s+table\b|\b(update)\s+\S+\s+set\b)`)
	chainSep  = regexp.MustCompile(`&&|\|\||[;|&\n]`)
	envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	// cloudOp is an operation the cloud vendor names, after one of its services: create-bucket, instances.
	cloudOp = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

func bashSignature(cmd string) string {
	if m := sqlVerb.FindStringSubmatch(cmd); m != nil {
		for _, g := range m[1:] {
			if g != "" {
				return strings.ToUpper(g)
			}
		}
	}
	for _, seg := range chainSep.Split(cmd, -1) {
		words := strings.Fields(seg)
		for len(words) > 0 && (envAssign.MatchString(words[0]) || words[0] == "sudo") {
			words = words[1:]
		}
		if len(words) == 0 || words[0] == "cd" || words[0] == "pushd" {
			continue
		}
		// A bare name only: /usr/bin/rm is rm, but ./deploy.sh and ~/bin/x are scripts of the user's own.
		name := words[0]
		if strings.HasPrefix(name, "/usr/") || strings.HasPrefix(name, "/bin/") || strings.HasPrefix(name, "/opt/homebrew/") {
			name = filepath.Base(name)
		}
		if !programs[name] {
			return "other"
		}
		key := []string{name}
		if cli, ok := subcommands[name]; ok && len(words) > 1 && cli.first[words[1]] {
			key = append(key, words[1])
			if cli.second != nil && len(words) > 2 && cli.second(words[2]) {
				key = append(key, words[2])
			}
		}
		return strings.Join(key, " ")
	}
	return "other"
}

// cliWords are the subcommands a signature may keep for one CLI: its own first-level subcommands, and what may
// follow one. An alias, plugin or extension the user added isn't in them, so it stops the signature at the program.
type cliWords struct {
	first  map[string]bool
	second func(string) bool
}

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

func in(words string) func(string) bool {
	m := set(words)
	return func(w string) bool { return m[w] }
}

var (
	ghVerbs = in(`create merge view list edit close reopen comment checkout diff review status delete clone fork rerun
		cancel watch download upload sync archive rename set ready lock unlock pin transfer develop checks run enable
		disable login logout refresh`)
	dockerWords = cliWords{first: set(`build run exec ps images pull push rm rmi stop start restart logs compose volume
		network system prune tag login logout inspect cp kill buildx container image save load`)}
	terraformWords = cliWords{first: set(`init plan apply destroy validate fmt import state output show refresh workspace
		providers taint untaint login force-unlock graph console`)}
	cloudOps = func(w string) bool { return cloudOp.MatchString(w) && len(w) <= 64 }
)

var subcommands = map[string]cliWords{
	"git": {first: set(`add am apply bisect blame branch checkout cherry-pick clean clone commit commit-tree config diff
		fetch gc grep init log merge mv pull push rebase reflog remote reset restore revert rm show stash status
		submodule switch tag worktree describe ls-files rev-parse`)},
	"gh": {first: set(`pr issue repo release run workflow api auth gist search secret variable label cache codespace
		project browse`), second: ghVerbs},
	"glab":      {first: set(`mr issue repo release ci pipeline api auth`), second: ghVerbs},
	"docker":    dockerWords,
	"podman":    dockerWords,
	"kubectl":   {first: set(`get describe apply create delete edit patch replace scale rollout logs exec port-forward cp label annotate expose run set config top drain cordon uncordon taint auth diff wait explain`)},
	"helm":      {first: set(`install upgrade uninstall rollback list status template lint package repo dependency pull push show get history search create test`)},
	"terraform": terraformWords,
	"tofu":      terraformWords,
	// Past one of its services, the cloud vendor names the operation; aliases only add top-level commands.
	"aws": {first: set(`s3 s3api ec2 iam rds dynamodb lambda cloudformation sts sqs sns ecs eks ecr logs cloudwatch
		route53 kms secretsmanager ssm cloudfront elasticache elbv2 elb autoscaling apigateway apigatewayv2 events
		stepfunctions kinesis firehose glue athena redshift organizations sso configure backup efs acm cognito-idp
		ses sesv2 bedrock bedrock-runtime sagemaker codebuild codepipeline codedeploy wafv2 guardduty`), second: cloudOps},
	"gcloud": {first: set(`compute container sql storage iam projects auth config run functions app pubsub secrets dns
		builds artifacts logging services kms redis spanner bigtable dataflow dataproc scheduler tasks`), second: cloudOps},
	"az": {first: set(`login account group vm aks acr storage keyvault network webapp functionapp sql cosmosdb monitor
		role ad deployment resource appservice containerapp`), second: cloudOps},
	"go":        {first: set(`build test run mod get install vet fmt generate env clean work tool version list`)},
	"cargo":     {first: set(`build test run check add install update publish clippy fmt new init bench doc clean remove fetch`)},
	"pip":       {first: set(`install uninstall freeze list show download wheel`)},
	"pip3":      {first: set(`install uninstall freeze list show download wheel`)},
	"uv":        {first: set(`pip add remove sync run lock venv tool init python build publish`)},
	"poetry":    {first: set(`install add remove update lock run build publish show shell init new`)},
	"npm":       {first: set(`install ci uninstall update publish unpublish run test audit init pack link version deprecate exec`)},
	"pnpm":      {first: set(`install add remove update publish run test exec`)},
	"brew":      {first: set(`install uninstall upgrade update list info search tap untap services cleanup doctor link unlink pin unpin outdated reinstall`)},
	"apt":       {first: set(`install remove purge update upgrade autoremove`)},
	"apt-get":   {first: set(`install remove purge update upgrade autoremove dist-upgrade`)},
	"systemctl": {first: set(`start stop restart reload enable disable status daemon-reload mask unmask`)},
	"vault":     {first: set(`read write kv login secrets auth policy token list delete operator`)},
	"supabase":  {first: set(`db functions migration start stop link login init gen secrets projects`)},
	"vercel":    {first: set(`deploy env link login logs domains pull promote rollback remove dev`)},
	"fly":       {first: set(`deploy launch secrets scale apps machines ssh logs status volumes postgres`)},
	"flyctl":    {first: set(`deploy launch secrets scale apps machines ssh logs status volumes postgres`)},
	"heroku":    {first: set(`run logs config ps apps releases pg addons`)},
	"wrangler":  {first: set(`deploy dev publish secret kv d1 r2 login tail pages`)},
	"stripe":    {first: set(`listen trigger login customers charges payments products prices`)},
	"firebase":  {first: set(`deploy init login use serve emulators functions hosting`)},
	"pulumi":    {first: set(`up preview destroy stack config login new refresh import`)},
	"kind":      {first: set(`create delete get load`)},
	"minikube":  {first: set(`start stop delete status service`)},
	"eksctl":    {first: set(`create delete get update upgrade scale utils`)},
}

// programs are the commands a signature may name.
var programs = set(`git gh glab docker podman kubectl helm terraform tofu aws gcloud az go cargo pip pip3 uv poetry
	brew apt apt-get systemctl vault supabase vercel fly flyctl heroku wrangler stripe firebase pulumi kind minikube
	eksctl npm pnpm yarn bun npx make rclone gsutil sf launchctl node deno python python3 pytest ruby java rm rmdir mv
	cp find ls cat head tail grep rg sed awk wc sort jq xargs echo mkdir touch chmod chown ln tar zip unzip curl wget
	ssh scp rsync dd mkfs diskutil kill pkill killall lsof ps open psql pg_dump pg_restore mysql mysqldump sqlite3
	mongosh mongodump redis-cli openssl gpg security defaults crontab shred truncate ansible ansible-playbook cdk sam
	serverless`)

// extensions are the file types a signature may name.
var extensions = set(`.tf .tfvars .hcl .py .js .ts .tsx .jsx .mjs .cjs .go .rs .java .kt .rb .php .cs .c .h .cpp
	.hpp .swift .m .sh .bash .zsh .ps1 .sql .md .mdx .txt .json .jsonl .yaml .yml .toml .ini .cfg .conf .env .xml .html
	.css .scss .vue .svelte .lock .csv .log .png .jpg .jpeg .gif .svg .webp .pdf .ipynb .proto .graphql .gradle .r
	.scala .lua .dart .ex .exs .zig .nix .tex .rst .gitignore .dockerignore`)

// builtinTools are Claude Code's own tools; any other tool name is the user's or a plugin's.
var builtinTools = set(`Bash Read Write Edit MultiEdit NotebookEdit Grep Glob LS WebFetch WebSearch Task Agent
	TodoWrite TodoRead ToolSearch SlashCommand BashOutput KillShell KillBash ExitPlanMode EnterPlanMode
	AskUserQuestion LSP Monitor TaskStop TaskOutput SendMessage`)
