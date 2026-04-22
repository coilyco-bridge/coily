# 2. `coily whoami`

**What it is**: Aggregates `aws sts get-caller-identity`, `kubectl auth whoami` (with fallback to config view), and `gh api user` into one yaml block.

**How to invoke**: `coily whoami`

**Expected shape**: Top-level keys `aws`, `kubectl`, `gh`. Each is either a map of identity fields or a map with `error`.

**Test prompt**:
> Verify `./bin/coily whoami` in `/Users/kai/projects/coilysiren/coily/` produces yaml with top-level keys `aws`, `kubectl`, `gh`. For each tool that is authenticated on this host, assert the expected identity fields are present (aws: Account/Arn/UserId; gh: login/id; kubectl: current_context or username). For unauthenticated tools, assert an `error` field is present. Also run it 3 times and assert the audit log at ~/.local/state/coily/audit.jsonl gains 3 entries with verb="whoami".
