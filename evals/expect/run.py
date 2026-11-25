#!/usr/bin/env python3
"""Expect eval: have a model write expect checks for 56 catalog skills from the guide alone, then grade the checks
on 1,609 labelled calls. Run it after changing skills/paws/expect.md or the model that writes checks.

    python3 evals/expect/run.py --out /tmp/expect-eval            # generate with Opus, then grade
    python3 evals/expect/run.py --out /tmp/expect-eval --grade-only

Each skill gets its own headless `claude -p` session in a fresh temp folder that holds only the guide, the skill's
text, its rule files and a paws binary. The session gets a scratch CLAUDE_CONFIG_DIR and an empty HOME, so no
skills, hooks, memory or history from the machine reach it. It logs in with the claude.ai login
(CLAUDE_CODE_OAUTH_TOKEN, or the macOS keychain entry Claude Code writes), never an API key.
A skill already generated is skipped, so a stopped run resumes.
"""
import argparse, concurrent.futures, hashlib, json, os, re, shutil, subprocess, sys, tempfile, threading, time

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(HERE))
DATA = os.path.join(HERE, 'data')
lock = threading.Lock()


def say(*a):
    with lock:
        print(time.strftime('%H:%M:%S'), *a, flush=True)


def build(out):
    os.makedirs(f'{out}/bin', exist_ok=True)
    for pkg, name in (('./cmd/paws', 'paws'), ('./evals/expect/casegrade', 'casegrade')):
        subprocess.run(['go', 'build', '-o', f'{out}/bin/{name}', pkg], cwd=REPO, check=True)
    os.makedirs(f'{out}/guide', exist_ok=True)
    for src in ('skills/paws/SKILL.md', 'skills/paws/expect.md', 'docs/rules.md'):
        shutil.copy(os.path.join(REPO, src), f'{out}/guide/')


def fetch(out, repo, ref):
    """A shallow checkout of one commit, the way paws fetches upstream skills. Only read, never run."""
    dest = f'{out}/src/{repo.replace("/", "_")}@{ref[:12]}'
    if os.path.isdir(f'{dest}/.git') and os.listdir(dest) != ['.git']:
        return dest
    os.makedirs(dest, exist_ok=True)
    env = dict(os.environ, GIT_TERMINAL_PROMPT='0', GIT_ASKPASS='')
    for args in (['init', '-q'], ['fetch', '-q', '--depth', '1', '--', 'https://' + repo, ref],
                 ['-c', 'advice.detachedHead=false', 'checkout', '-q', 'FETCH_HEAD']):
        subprocess.run(['git', *args], cwd=dest, env=env, check=True, capture_output=True)
    return dest


def skill_text(out, name, src):
    """SKILL.md as published, plus the names of the skill's other files. Placeholders stay unfilled: the labelled
    cases were written from this same text, so filling them would change the rules under the labels."""
    d = os.path.join(fetch(out, src['repo'], src['ref']), src['path'])
    raw = open(f'{d}/SKILL.md', 'rb').read()
    if hashlib.sha256(raw).hexdigest() != src['sha256']:
        sys.exit(f'{name}: SKILL.md at {src["repo"]}@{src["ref"]} does not match data/skills.json')
    text = raw.decode('utf-8')
    others = []
    for root, dirs, files in os.walk(d):
        dirs[:] = [x for x in dirs if not x.startswith('.')]
        others += [os.path.relpath(os.path.join(root, f), d) for f in files if not f.startswith('.')]
    others = sorted(set(others) - {'SKILL.md'})
    if others:
        text += '\n\n<!-- other files in this skill: ' + ', '.join(others) + ' -->\n'
    return text


def token():
    if os.environ.get('CLAUDE_CODE_OAUTH_TOKEN'):
        return os.environ['CLAUDE_CODE_OAUTH_TOKEN']
    raw = subprocess.run(['security', 'find-generic-password', '-s', 'Claude Code-credentials', '-w'],
                         capture_output=True, text=True, check=True).stdout
    return json.loads(raw)['claudeAiOauth']['accessToken']


def session(args, work, log):
    """One isolated headless session in work; its event stream goes to log."""
    cfg = tempfile.mkdtemp(prefix='expect-eval-cfg-')
    home = tempfile.mkdtemp(prefix='expect-eval-home-')
    try:
        env = {'HOME': home, 'PATH': os.environ['PATH'], 'TMPDIR': tempfile.gettempdir(), 'LANG': 'en_US.UTF-8',
               'CLAUDE_CONFIG_DIR': cfg, 'CLAUDE_CODE_OAUTH_TOKEN': token(), 'DISABLE_AUTOUPDATER': '1'}
        prompt = open(os.path.join(HERE, 'prompt.md')).read()
        with open(log, 'w') as f:
            subprocess.run([args.claude, '-p', prompt, '--model', args.model, '--no-session-persistence',
                            '--allowedTools', 'Read', 'Write', 'Edit', 'Glob', 'Grep', 'Bash',
                            '--output-format', 'stream-json', '--verbose'],
                           cwd=work, env=env, stdin=subprocess.DEVNULL, stdout=f, stderr=subprocess.STDOUT)
    finally:
        shutil.rmtree(cfg, ignore_errors=True)
        shutil.rmtree(home, ignore_errors=True)
    res = {}
    for line in open(log, errors='replace'):
        if '"result"' in line:
            try:
                d = json.loads(line)
            except ValueError:
                continue
            if d.get('type') == 'result':
                res = d
    return res


def generate(args, out, name, rules):
    d = f'{out}/runs/{name}'
    if os.path.exists(f'{d}/DONE'):
        return
    os.makedirs(d, exist_ok=True)
    tries = limits = 0
    while True:
        work = tempfile.mkdtemp(prefix='expect-eval-')
        try:
            for sub in ('skills', 'bindings', 'out'):
                os.makedirs(f'{work}/{sub}')
            shutil.copytree(f'{out}/guide', f'{work}/guide')
            shutil.copy(f'{out}/skills/{name}.md', f'{work}/skills/')
            for r in rules:
                shutil.copy(f'{DATA}/rules/{r}', f'{work}/bindings/')
            shutil.copy(f'{out}/bin/paws', f'{work}/paws')
            t0 = time.time()
            res = session(args, work, f'{d}/session.jsonl')
            if not res.get('is_error') and all(os.path.exists(f'{work}/out/{r}') for r in rules):
                shutil.copytree(f'{work}/out', f'{d}/out', dirs_exist_ok=True)
                json.dump({'secs': round(time.time() - t0), 'turns': res.get('num_turns')}, open(f'{d}/DONE', 'w'))
                say(f'{name}: done in {round(time.time() - t0)}s')
                return
            text = ((res.get('result') or '') + open(f'{d}/session.jsonl', errors='replace').read()[-3000:]).lower()
            if any(k in text for k in ('rate limit', 'usage limit', 'limit reached', 'hit your', 'overloaded', ' 429')):
                limits += 1
                wait = min(1800, 300 * limits)  # a 5-hour cap can take hours to reset: keep waiting
                say(f'{name}: rate limited, waiting {wait}s')
                time.sleep(wait)
                continue
            tries += 1
            say(f'{name}: failed (try {tries}): {(res.get("result") or "")[:160]!r}')
            if tries >= 3:
                open(f'{d}/FAILED', 'w').write(text[-3000:])
                return
        finally:
            shutil.rmtree(work, ignore_errors=True)


def pct(a, b):
    return f'{100 * a / b:.0f}%' if b else '-'


def grade(out, names):
    rules = f'{out}/rules'
    shutil.rmtree(rules, ignore_errors=True)
    os.makedirs(rules)
    for n in names:
        o = f'{out}/runs/{n}/out'
        for f in (os.listdir(o) if os.path.isdir(o) else []):
            if f.endswith('.md') and f != 'notes.md':
                shutil.copy(f'{o}/{f}', rules)
    cg = f'{out}/bin/casegrade'
    lint = [json.loads(l) for l in subprocess.run([cg, '-lint', '-rules', rules], capture_output=True, text=True,
                                                   check=True).stdout.splitlines()]
    split = json.load(open(f'{DATA}/split.json'))
    which = {s: k for k in split for s in split[k]}
    score = {'checks': sum(len(x.get('checks') or []) for x in lint),
             'rule_files_with_checks': sum(1 for x in lint if x.get('checks')),
             'rule_files': len(lint),
             'load_errors': sum(1 for x in lint if x.get('expect_err') or x.get('parse_err'))}
    for set_name in ('cases', 'probes'):
        res_path = f'{out}/{set_name}-graded.jsonl'
        subprocess.run([cg, '-rules', rules, '-cases', f'{DATA}/{set_name}.jsonl', '-out', res_path], check=True)
        agg = {}
        for l in open(res_path):
            r = json.loads(l)
            if r['status'] != 'ok' or r['skill'] not in names:
                continue
            for k in ((which.get(r['skill'], '?'), 'all') if set_name == 'cases' else ('all',)):
                a = agg.setdefault(k, {'n': 0, 'right': 0, 'breaks': 0, 'caught': 0, 'follows': 0, 'passed': 0})
                a['n'] += 1
                a['right'] += r['correct']
                if r['label'] == 'breaks':
                    a['breaks'] += 1
                    a['caught'] += r['correct']
                else:
                    a['follows'] += 1
                    a['passed'] += r['correct']
        score[set_name] = agg
    json.dump(score, open(f'{out}/score.json', 'w'), indent=1)
    base = json.load(open(f'{HERE}/baseline.json')) if os.path.exists(f'{HERE}/baseline.json') else {}
    print(f"\nchecks {score['checks']} in {score['rule_files_with_checks']}/{score['rule_files']} rule files, "
          f"load errors {score['load_errors']}"
          + (f"   (baseline {base['checks']}, {base['rule_files_with_checks']}, {base['load_errors']})" if base else ''))
    print(f"{'set':10} {'right':>14} {'breaks caught':>16} {'follows passed':>16}   baseline right / caught / passed")
    for k in ('dev', 'held', 'all'):
        a = score['cases'].get(k)
        b = base.get('cases', {}).get(k)
        if not a:
            continue
        print(f"{k:10} {pct(a['right'], a['n']):>5} ({a['right']}/{a['n']}) {pct(a['caught'], a['breaks']):>8} "
              f"({a['caught']}/{a['breaks']}) {pct(a['passed'], a['follows']):>7} ({a['passed']}/{a['follows']})   "
              + (f"{pct(b['right'], b['n'])} / {pct(b['caught'], b['breaks'])} / {pct(b['passed'], b['follows'])}"
                 if b else ''))
    p = score['probes'].get('all')
    if p:
        bp = base.get('probes', {}).get('all')
        print(f"probes     {p['right']}/{p['n']}" + (f"   (baseline {bp['right']}/{bp['n']})" if bp else ''))
    print(f'\nper case: {out}/cases-graded.jsonl   summary: {out}/score.json')


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--out', required=True, help='work folder; reuse it to resume or regrade')
    ap.add_argument('--model', default='opus')
    ap.add_argument('--parallel', type=int, default=5)
    ap.add_argument('--skills', help='comma-separated subset (default: all 56)')
    ap.add_argument('--claude', default='claude', help='Claude Code binary')
    ap.add_argument('--grade-only', action='store_true', help='grade the checks already in --out')
    args = ap.parse_args()
    out = os.path.abspath(args.out)
    manifest = json.load(open(f'{DATA}/skills.json'))
    names = args.skills.split(',') if args.skills else sorted(manifest)
    build(out)
    if not args.grade_only:
        os.makedirs(f'{out}/skills', exist_ok=True)
        for n in names:
            open(f'{out}/skills/{n}.md', 'w', encoding='utf-8').write(skill_text(out, n, manifest[n]))
        rules = {n: sorted(f for f in os.listdir(f'{DATA}/rules') if f.startswith(n + '__')) for n in names}
        with concurrent.futures.ThreadPoolExecutor(args.parallel) as ex:
            list(ex.map(lambda n: generate(args, out, n, rules[n]), names))
        failed = [n for n in names if os.path.exists(f'{out}/runs/{n}/FAILED')]
        if failed:
            say('no checks from:', ', '.join(failed))
    grade(out, names)


if __name__ == '__main__':
    main()
