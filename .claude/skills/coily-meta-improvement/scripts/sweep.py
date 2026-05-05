#!/usr/bin/env python3
"""
Walk ~/.coily/audit/*.jsonl and ~/.claude/projects/*/*.jsonl over a window,
produce a structured report for finding-generation.

Outputs to stdout:
  - audit summary: per-verb counts, exit-code distribution, deny rate
  - audit top failures: rows with non-zero exit codes, grouped
  - claude denied-Bash: permission-denied bash invocations matching coily-relevant tools
  - cross-reference: raw tool invocations denied that have coily wrappers

Skips coilysiren-repo-recall.jsonl (reported as count only).
"""
import json
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path

AUDIT_DIR = Path.home() / ".coily" / "audit"
SESSIONS_DIR = Path.home() / ".claude" / "projects"
WINDOW_DAYS = 35

# Tools coily wraps. Raw use of these is a signal.
WRAPPED_TOOLS = {
    "aws", "gh", "kubectl", "docker", "tailscale",
    "systemctl", "journalctl",  # gaming/sirens-discord-ops
}


def parse_ts(s) -> datetime:
    """coily audit timestamps are either iso strings or unix epoch ints."""
    if isinstance(s, (int, float)):
        return datetime.fromtimestamp(s, tz=timezone.utc)
    if isinstance(s, str):
        # try int-as-string first (some rows store epoch as string)
        try:
            return datetime.fromtimestamp(int(s), tz=timezone.utc)
        except (ValueError, TypeError):
            pass
        return datetime.fromisoformat(s).astimezone(timezone.utc)
    raise TypeError(f"unhandled ts type: {type(s)}")


def walk_audit(window_start: datetime):
    """yield (repo, row) tuples for audit rows in window."""
    for jsonl in AUDIT_DIR.glob("*.jsonl"):
        repo = jsonl.stem
        if repo == "coilysiren-repo-recall":
            yield ("__SKIP__", {"repo": repo, "count": sum(1 for _ in jsonl.open())})
            continue
        try:
            with jsonl.open() as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    try:
                        row = json.loads(line)
                    except json.JSONDecodeError:
                        continue
                    ts = row.get("ts")
                    if not ts:
                        continue
                    try:
                        if parse_ts(ts) < window_start:
                            continue
                    except (ValueError, TypeError):
                        continue
                    yield (repo, row)
        except OSError:
            continue


def walk_sessions(window_start: datetime):
    """yield permission-denied bash entries from claude sessions in window."""
    if not SESSIONS_DIR.exists():
        return
    for jsonl in SESSIONS_DIR.rglob("*.jsonl"):
        try:
            mtime = datetime.fromtimestamp(jsonl.stat().st_mtime, tz=timezone.utc)
            if mtime < window_start:
                continue
            with jsonl.open() as f:
                for line in f:
                    if "Permission" not in line and "denied" not in line.lower():
                        continue
                    try:
                        rec = json.loads(line)
                    except json.JSONDecodeError:
                        continue
                    # extract any "Permission to use Bash with command X has been denied" shape
                    blob = json.dumps(rec)
                    m = re.search(
                        r"Permission to use Bash with command ([^\"]+?) has been denied",
                        blob,
                    )
                    if m:
                        yield (jsonl, m.group(1))
        except OSError:
            continue


def first_token(argv):
    """Return the bash binary called (first non-pipe non-coily token)."""
    if not argv:
        return None
    return argv[0]


def coily_verb_path(verb: str) -> str:
    """Map dotted verb name to dashed area for skill routing."""
    return verb.replace(".", "-")


def main():
    now = datetime.now(timezone.utc)
    window_start = now - timedelta(days=WINDOW_DAYS)

    # --- audit aggregation ---
    by_verb = Counter()
    by_verb_failed = Counter()
    by_repo = Counter()
    failures_by_verb = defaultdict(list)  # verb -> list of (ts, exit_code, argv)
    skipped = []

    for repo, row in walk_audit(window_start):
        if repo == "__SKIP__":
            skipped.append(row)
            continue
        verb = row.get("verb", "?")
        ec = row.get("exit_code", 0)
        by_verb[verb] += 1
        by_repo[repo] += 1
        if ec != 0:
            by_verb_failed[verb] += 1
            failures_by_verb[verb].append(
                (row.get("ts"), ec, row.get("argv", []))
            )

    # --- claude sessions: denied bash ---
    denied_cmds = []
    denied_first_tokens = Counter()
    for src, cmd in walk_sessions(window_start):
        denied_cmds.append((src, cmd))
        # extract first token of the denied command, ignoring leading subshells / cd
        # the cmd is the literal denied string from the perm message
        toks = cmd.split()
        if not toks:
            continue
        # strip leading "cd ... &&" if present
        if toks[0] == "cd" and "&&" in toks[:6]:
            try:
                idx = toks.index("&&")
                toks = toks[idx + 1 :]
            except ValueError:
                pass
        if toks:
            denied_first_tokens[toks[0]] += 1

    # --- report ---
    out = []
    out.append(f"=== sweep window: last {WINDOW_DAYS} days (since {window_start.isoformat()}) ===\n")

    out.append("## audit: row counts (skipped repo-recall, count noted)")
    for s in skipped:
        out.append(f"  - SKIP {s['repo']}: {s['count']} rows")
    out.append(f"  total in-window rows: {sum(by_verb.values())}")
    out.append(f"  unique verbs: {len(by_verb)}")
    out.append(f"  unique repos with activity: {len(by_repo)}")
    out.append("")

    out.append("## audit: top 30 verbs by count (with deny/fail rate)")
    for verb, count in by_verb.most_common(30):
        failed = by_verb_failed[verb]
        rate = (failed / count * 100) if count else 0
        out.append(f"  {count:6d}  {verb:40s}  failed={failed} ({rate:.1f}%)")
    out.append("")

    out.append("## audit: top 20 repos by activity")
    for repo, count in by_repo.most_common(20):
        out.append(f"  {count:6d}  {repo}")
    out.append("")

    out.append("## audit: verbs with non-trivial failure (count >= 3, fail >= 2)")
    notable = [
        (v, c, by_verb_failed[v])
        for v, c in by_verb.items()
        if c >= 3 and by_verb_failed[v] >= 2
    ]
    notable.sort(key=lambda x: -x[2])
    for verb, count, failed in notable[:30]:
        out.append(f"  {failed}/{count} failed: {verb}")
        # sample failure argv
        for ts, ec, argv in failures_by_verb[verb][:2]:
            argv_str = " ".join(str(a) for a in argv)[:140]
            out.append(f"      ts={ts} ec={ec} argv={argv_str}")
    out.append("")

    out.append("## claude sessions: denied bash invocations (top 20 first-tokens)")
    for tok, count in denied_first_tokens.most_common(20):
        wrapped = "  [WRAPPED-BY-COILY]" if tok in WRAPPED_TOOLS else ""
        out.append(f"  {count:5d}  {tok}{wrapped}")
    out.append(f"  total denied: {len(denied_cmds)}")
    out.append("")

    out.append("## claude sessions: sample denied commands matching wrapped tools")
    sample = []
    for src, cmd in denied_cmds:
        toks = cmd.split()
        if toks and toks[0] in WRAPPED_TOOLS:
            sample.append(cmd[:160])
        if len(sample) >= 20:
            break
    for s in sample:
        out.append(f"  - {s}")
    out.append("")

    out.append("## claude sessions: distinct verbs after `coily` (top 20)")
    coily_subverbs = Counter()
    for src, cmd in denied_cmds:
        toks = cmd.split()
        if len(toks) >= 2 and toks[0] == "coily":
            # take the first 3 tokens after coily for a verb path
            sub = ".".join(toks[1:4])
            coily_subverbs[sub] += 1
    for sub, count in coily_subverbs.most_common(20):
        out.append(f"  {count:5d}  coily {sub.replace('.', ' ')}")
    out.append("")

    sys.stdout.write("\n".join(out) + "\n")


if __name__ == "__main__":
    main()
