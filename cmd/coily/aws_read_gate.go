package main

import (
	"os"
	"strings"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/pkg/audit"
	"github.com/urfave/cli/v3"
)

// awsReadGate is the preflight gate wired onto the `coily ops aws`
// passthrough. It closes the read-side gap called out in
// coilyco-bridge/coily#54: read-only aws verbs used to land an audit row
// and pass straight through, no argv validation. The audit row documents
// the leak after the fact; it is not the layer that prevents the read.
//
// The gate is the pre-send deny. A read-only invocation (s3 ls, ec2
// describe-*, iam list-*, sts get-caller-identity, ...) is denied when one
// of its argv tokens matches a sensitive pattern (a secrets/state/backup
// bucket glob, an admin role ARN) the operator did not explicitly intend to
// expose. Read-only invocations exfiltrate (`s3 ls` on a secrets bucket),
// confirm threat-model state, and enumerate; CloudTrail and wide iam scoping
// are post-hoc and lazy respectively, so neither stops the read happening.
//
// Scope is read-only-against-sensitive-patterns only. Destructive verbs are
// a different layer (already gated) and out of scope for #54. Non-read verbs
// and reads that touch nothing sensitive pass through untouched.
//
// Audit rows land for both outcomes so the trail survives the boundary:
//   - denied read   -> a row with verb "ops.aws.read.denied" (reject),
//     written here before the hard error; the passthrough never runs, so
//     this is the only row for the invocation.
//   - explicitly-allowed sensitive read -> a row with verb
//     "ops.aws.read.allowed" (accept), written here; the passthrough then
//     runs and writes its own "ops.aws" row, so the allow decision and the
//     execution are both on the record.
//
// awsReadGate returns the closure cli.Command wiring expects. It captures
// the Runner so it can read the sensitive-pattern config and write the
// denied/allowed audit rows. Built via the (*Runner).awsReadGate method
// expression in the ptOps registry.
func (r *Runner) awsReadGate() func(argv []string) error {
	return func(argv []string) error {
		if !isAWSReadOnly(argv) {
			return nil
		}
		patterns := r.awsSensitiveReadPatterns()
		token, pattern, hit := awsSensitiveReadMatch(argv, patterns)
		if !hit {
			return nil
		}
		if r.awsSensitiveReadAllowed(token) {
			r.appendAWSReadRow(audit.DecisionAccept, "ops.aws.read.allowed", argv, 0, "")
			return nil
		}
		r.appendAWSReadRow(audit.DecisionReject, "ops.aws.read.denied", argv, 1,
			"sensitive read denied: token "+token+" matched pattern "+pattern)
		return cli.Exit(awsReadBlockMessage(token, pattern), 1)
	}
}

// isAWSReadOnly reports whether argv (the post-`coily ops aws` slice) is a
// read-only aws invocation: a service plus a read-only operation. Global
// flags (`--region`, `--profile`, `--output`, ...) are skipped so the
// service/operation are found wherever they sit. The read classifier keys
// off the operation verb prefix, the AWS CLI's own naming convention
// (describe-*, list-*, get-*, ...), plus the s3 high-level `ls`.
func isAWSReadOnly(argv []string) bool {
	_, op, ok := awsServiceOp(argv)
	if !ok {
		return false
	}
	return isAWSReadOp(op)
}

// awsServiceOp extracts the service and operation tokens from argv, skipping
// global flags and their values. Returns ok=false when either is absent.
func awsServiceOp(argv []string) (service, op string, ok bool) {
	positionals := awsPositionals(argv)
	if len(positionals) < 2 {
		return "", "", false
	}
	return positionals[0], positionals[1], true
}

// awsPositionals returns argv with flags (and the values of known
// value-taking global flags) removed, preserving order. The AWS CLI's
// value-taking global flags are enumerated so `--region us-east-1` does not
// leave `us-east-1` masquerading as the service token. Unknown `--flag value`
// pairs are conservatively treated as flag-only (value kept as positional);
// that is safe here because the only positionals the gate reads are the
// leading service/operation, which precede resource args.
func awsPositionals(argv []string) []string {
	valueFlags := map[string]bool{
		"--region": true, "--profile": true, "--output": true,
		"--endpoint-url": true, "--cli-read-timeout": true,
		"--cli-connect-timeout": true, "--color": true, "--ca-bundle": true,
		"--query": true,
	}
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if strings.HasPrefix(tok, "--") {
			if eq := strings.IndexByte(tok, '='); eq >= 0 {
				continue // --flag=value, self-contained
			}
			if valueFlags[tok] && i+1 < len(argv) {
				i++ // consume the value
			}
			continue
		}
		if strings.HasPrefix(tok, "-") && tok != "-" {
			continue // short flag
		}
		out = append(out, tok)
	}
	return out
}

// awsReadPrefixes are the AWS CLI operation-verb prefixes that denote a
// read. The CLI is consistent: data-returning operations are named
// describe-*, list-*, get-*, head-*, lookup-*, search-*, select-*, and
// batch-get-*. scan / query are the DynamoDB / CloudWatch Logs read verbs.
var awsReadPrefixes = []string{
	"describe-", "list-", "get-", "head-", "lookup-",
	"search-", "select-", "batch-get-",
}

// awsReadExact are read operations that are not prefix-shaped: the s3
// high-level `ls`, and the DynamoDB / Logs `scan` / `query`.
var awsReadExact = map[string]bool{
	"ls": true, "scan": true, "query": true,
}

// isAWSReadOp reports whether op is a read-only aws operation by the
// CLI's naming convention. Write-shaped verbs (cp, sync, mv, rb, rm,
// create-*, delete-*, put-*, update-*, ...) return false: they are the
// destructive layer, out of scope for #54's read-side gate.
func isAWSReadOp(op string) bool {
	if awsReadExact[op] {
		return true
	}
	for _, p := range awsReadPrefixes {
		if strings.HasPrefix(op, p) {
			return true
		}
	}
	return false
}

// awsSensitiveReadMatch scans argv's positional tokens (service/operation
// and any resource args) for the first that matches a sensitive pattern.
// Matching is case-insensitive. Returns the matched token, the pattern it
// hit, and ok.
func awsSensitiveReadMatch(argv, patterns []string) (token, pattern string, ok bool) {
	for _, tok := range awsPositionals(argv) {
		lower := strings.ToLower(tok)
		for _, pat := range patterns {
			if globMatch(strings.ToLower(pat), lower) {
				return tok, pat, true
			}
		}
	}
	return "", "", false
}

// globMatch reports whether s matches pattern, where `*` matches any run of
// characters (including `/` and `:`, unlike filepath.Match) and every other
// character is literal. A small purpose-built matcher rather than
// filepath.Match because the sensitive patterns include ARNs whose `/`
// separators filepath.Match's `*` would refuse to cross.
func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s // no wildcard: exact match
	}
	// Leading part must anchor at the start unless the pattern starts with *.
	if parts[0] != "" {
		if !strings.HasPrefix(s, parts[0]) {
			return false
		}
		s = s[len(parts[0]):]
	}
	// Middle parts must appear in order.
	for _, mid := range parts[1 : len(parts)-1] {
		if mid == "" {
			continue
		}
		idx := strings.Index(s, mid)
		if idx < 0 {
			return false
		}
		s = s[idx+len(mid):]
	}
	// Trailing part must anchor at the end unless the pattern ends with *.
	last := parts[len(parts)-1]
	if last != "" {
		return strings.HasSuffix(s, last)
	}
	return true
}

// awsSensitiveReadPatterns returns the sensitive-pattern globs the gate
// matches read-only argv against. A non-empty AWS.SensitiveReadPatterns in
// config replaces the defaults wholesale (so an operator can scope the gate
// to exactly their resources); empty falls back to
// defaultAWSSensitiveReadPatterns.
func (r *Runner) awsSensitiveReadPatterns() []string {
	if r != nil && r.Cfg != nil && len(r.Cfg.AWS.SensitiveReadPatterns) > 0 {
		return r.Cfg.AWS.SensitiveReadPatterns
	}
	return defaultAWSSensitiveReadPatterns
}

// defaultAWSSensitiveReadPatterns is the baked-in default-deny set. Every
// entry is a meaningful-name glob, never an opaque id (account ids and the
// like stay out of tracked files per the workspace safety rules; the `*`
// wildcard covers the account-id segment of an ARN). The set targets the
// resource classes a read leaks hardest: secrets / credentials, terraform
// state, backups, and admin/root role ARNs.
var defaultAWSSensitiveReadPatterns = []string{
	"*secret*",
	"*credential*",
	"*-private*",
	"*private-*",
	"*tfstate*",
	"*terraform-state*",
	"*-backup*",
	"*-backups*",
	"arn:aws:iam::*:role/*admin*",
	"arn:aws:iam::*:role/*root*",
}

// awsSensitiveReadAllowed reports whether a sensitive token is explicitly
// allowed, by either of the two documented escape hatches:
//   - config: a glob in AWS.AllowSensitiveReads matches the token (the
//     persistent allow, lives in committed .coily/config.yaml).
//   - per-invocation: COILY_AWS_ALLOW_SENSITIVE_READ is set non-empty (the
//     one-shot override for a deliberate read of a flagged resource).
func (r *Runner) awsSensitiveReadAllowed(token string) bool {
	if strings.TrimSpace(os.Getenv(awsAllowSensitiveReadEnv)) != "" {
		return true
	}
	if r == nil || r.Cfg == nil {
		return false
	}
	lower := strings.ToLower(token)
	for _, pat := range r.Cfg.AWS.AllowSensitiveReads {
		if globMatch(strings.ToLower(pat), lower) {
			return true
		}
	}
	return false
}

// awsAllowSensitiveReadEnv is the per-invocation allow override. Set it
// non-empty on a single `coily ops aws` call to permit one deliberate read
// of a flagged resource without editing config.
const awsAllowSensitiveReadEnv = "COILY_AWS_ALLOW_SENSITIVE_READ"

// appendAWSReadRow writes the gate's own audit row. Best-effort: a nil
// audit writer (some test runners) means the boundary still holds (the
// caller denies regardless) but no row is written. Session id is not
// threaded here because the gate signature is argv-only; Append fills the
// row id, and the verb name carries the security-relevant signal.
func (r *Runner) appendAWSReadRow(decision, verb string, argv []string, exit int, errMsg string) {
	if r == nil || r.Audit == nil {
		return
	}
	rec := audit.Record{
		Decision: decision,
		Verb:     verb,
		Argv:     append([]string{"aws"}, argv...),
		ExitCode: exit,
		Error:    errMsg,
	}
	_ = r.Audit.Append(rec)
}

// awsReadBlockMessage is the hard error printed when the gate denies a read.
// It names the issue, the matched token and pattern, and both allow paths so
// the operator can deliberately proceed without guessing the mechanism.
func awsReadBlockMessage(token, pattern string) string {
	return "coily: read-only aws denied - `" + token + "` matched the sensitive-read pattern `" + pattern + "` (coilyco-bridge/coily#54).\n\n" +
		"Read-only aws verbs can still exfiltrate (s3 ls on a secrets bucket),\n" +
		"confirm threat-model state, and enumerate. The audit row is the trail,\n" +
		"not the gate - so this read is denied pre-send by default.\n\n" +
		"To allow it deliberately:\n" +
		"  - one-shot:   set " + awsAllowSensitiveReadEnv + "=1 for this invocation\n" +
		"  - persistent: add a glob to aws.allow_sensitive_reads in .coily/config.yaml\n\n" +
		"Non-sensitive reads and write verbs are unaffected."
}
