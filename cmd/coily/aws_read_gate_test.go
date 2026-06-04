package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/audit"
	"forgejo.coilysiren.me/coilyco-flight-deck/cli-guard/shell"
)

// TestIsAWSReadOnly classifies the read-only surface the gate keys off.
// Reads (describe-/list-/get-/head-/lookup-/search-/select-/batch-get-,
// plus s3 ls / dynamodb scan / query) are read-only; write-shaped verbs
// and incomplete argv are not.
func TestIsAWSReadOnly(t *testing.T) {
	read := [][]string{
		{"s3", "ls", "s3://bucket"},
		{"ec2", "describe-instances"},
		{"iam", "list-roles"},
		{"sts", "get-caller-identity"},
		{"s3api", "head-object", "--bucket", "b", "--key", "k"},
		{"cloudtrail", "lookup-events"},
		{"dynamodb", "scan", "--table-name", "t"},
		{"dynamodb", "query", "--table-name", "t"},
		{"resource-groups", "search-resources"},
		{"s3api", "select-object-content"},
		{"dynamodb", "batch-get-item"},
		// global flags before the service must not derail detection.
		{"--region", "us-east-1", "ec2", "describe-vpcs"},
		{"--profile", "prod", "--output", "json", "iam", "list-users"},
		{"--region=us-west-2", "s3", "ls"},
	}
	for _, argv := range read {
		if !isAWSReadOnly(argv) {
			t.Errorf("isAWSReadOnly(%v) = false, want true", argv)
		}
	}

	notRead := [][]string{
		{"s3", "cp", "a", "b"},
		{"s3", "sync", "a", "b"},
		{"s3", "rb", "s3://bucket"},
		{"ec2", "create-vpc"},
		{"iam", "delete-role"},
		{"s3api", "put-object"},
		{"ec2", "update-security-group-rule-descriptions-ingress"},
		{"s3"},                    // no operation
		{},                        // empty
		{"--region", "us-east-1"}, // flags only
	}
	for _, argv := range notRead {
		if isAWSReadOnly(argv) {
			t.Errorf("isAWSReadOnly(%v) = true, want false", argv)
		}
	}
}

// TestAWSPositionals confirms flags are stripped while positionals keep
// their order. Known value-taking global flags consume their value; for
// other `--flag value` pairs the value is intentionally retained as a
// positional so resource args in `--bucket prod-secrets` form still reach
// the sensitive-token scan.
func TestAWSPositionals(t *testing.T) {
	cases := []struct {
		argv []string
		want []string
	}{
		{[]string{"--region", "us-east-1", "ec2", "describe-instances"}, []string{"ec2", "describe-instances"}},
		{[]string{"--region=us-east-1", "s3", "ls"}, []string{"s3", "ls"}},
		{[]string{"s3", "ls", "s3://secrets-bucket"}, []string{"s3", "ls", "s3://secrets-bucket"}},
		// --output consumes "json" (known value flag); --max-items is not a
		// known global value flag, so "5" is retained as a positional.
		{[]string{"--output", "json", "iam", "list-roles", "--max-items", "5"}, []string{"iam", "list-roles", "5"}},
		// A non-global value flag's value is retained so it can be scanned.
		{[]string{"s3api", "head-object", "--bucket", "prod-secrets"}, []string{"s3api", "head-object", "prod-secrets"}},
	}
	for _, c := range cases {
		got := awsPositionals(c.argv)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("awsPositionals(%v) = %v, want %v", c.argv, got, c.want)
		}
	}
}

// TestGlobMatch covers the ARN-aware matcher: `*` crosses `/` and `:`,
// matching is otherwise literal, and a wildcard-free pattern is exact.
func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pattern, s string
		want       bool
	}{
		{"*secret*", "s3://my-secrets-bucket", true},
		{"*secret*", "s3://public-logs", false},
		{"*-private*", "data-private-zone", true},
		{"arn:aws:iam::*:role/*admin*", "arn:aws:iam::123:role/orgadmin", true},
		{"arn:aws:iam::*:role/*admin*", "arn:aws:iam::123:role/readonly", false},
		{"*tfstate*", "company-tfstate-prod", true},
		{"exact", "exact", true},
		{"exact", "exactly", false},
		{"prefix*", "prefix-then-more", true},
		{"prefix*", "no-prefix", false},
		{"*suffix", "ends-with-suffix", true},
		{"*suffix", "suffix-then-more", false},
	}
	for _, c := range cases {
		if got := globMatch(c.pattern, c.s); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", c.pattern, c.s, got, c.want)
		}
	}
}

// newAWSGateRunner builds a Runner with the given AWS config and a temp
// audit log so the gate's rows can be read back.
func newAWSGateRunner(t *testing.T, awsCfg AWS) (*Runner, string) {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.jsonl")
	aw := audit.NewWriter(logPath)
	t.Cleanup(func() { _ = aw.Close() })
	return &Runner{
		Cfg:    &Config{AWS: awsCfg},
		Runner: &shell.Runner{Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin},
		Audit:  aw,
	}, logPath
}

func readAuditRows(t *testing.T, path string) []audit.Record {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("open audit log: %v", err)
	}
	defer f.Close()
	recs, err := audit.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return recs
}

// TestAWSReadGate_DeniesSensitiveAndLandsAuditRow is the core #54 claim:
// a read-only invocation touching a sensitive pattern is denied pre-send,
// and a denial audit row lands so the trail survives the boundary.
func TestAWSReadGate_DeniesSensitiveAndLandsAuditRow(t *testing.T) {
	r, logPath := newAWSGateRunner(t, AWS{})
	gate := r.awsReadGate()

	argv := []string{"s3", "ls", "s3://prod-secrets-bucket"}
	err := gate(argv)
	if err == nil {
		t.Fatal("gate allowed a sensitive read; want a hard deny")
	}
	if !strings.Contains(err.Error(), "coilyco-bridge/coily#54") {
		t.Errorf("deny message missing issue reference: %v", err)
	}

	rows := readAuditRows(t, logPath)
	if len(rows) != 1 {
		t.Fatalf("want exactly 1 audit row for a denied read, got %d", len(rows))
	}
	row := rows[0]
	if row.Verb != "ops.aws.read.denied" {
		t.Errorf("denied row verb = %q, want ops.aws.read.denied", row.Verb)
	}
	if row.Decision != audit.DecisionReject {
		t.Errorf("denied row decision = %q, want %q", row.Decision, audit.DecisionReject)
	}
	if len(row.Argv) == 0 || row.Argv[0] != "aws" {
		t.Errorf("denied row argv should lead with aws; got %v", row.Argv)
	}
}

// TestAWSReadGate_AllowsNonSensitiveSilently: a read that touches nothing
// sensitive passes with no gate-written row (the passthrough writes the
// normal ops.aws row downstream).
func TestAWSReadGate_AllowsNonSensitiveSilently(t *testing.T) {
	r, logPath := newAWSGateRunner(t, AWS{})
	gate := r.awsReadGate()

	if err := gate([]string{"ec2", "describe-instances"}); err != nil {
		t.Fatalf("gate denied a benign read: %v", err)
	}
	if rows := readAuditRows(t, logPath); len(rows) != 0 {
		t.Errorf("benign read wrote %d gate rows, want 0", len(rows))
	}
}

// TestAWSReadGate_IgnoresWriteVerbs: destructive verbs are a different
// layer (out of scope for #54). The gate passes them through even when they
// touch a sensitive resource.
func TestAWSReadGate_IgnoresWriteVerbs(t *testing.T) {
	r, logPath := newAWSGateRunner(t, AWS{})
	gate := r.awsReadGate()

	if err := gate([]string{"s3", "rb", "s3://prod-secrets-bucket"}); err != nil {
		t.Fatalf("gate denied a write verb (out of scope): %v", err)
	}
	if rows := readAuditRows(t, logPath); len(rows) != 0 {
		t.Errorf("write verb wrote %d gate rows, want 0", len(rows))
	}
}

// TestAWSReadGate_AllowViaConfig: a config allow-glob lets a sensitive read
// through and lands an ops.aws.read.allowed row marking the allow decision.
func TestAWSReadGate_AllowViaConfig(t *testing.T) {
	r, logPath := newAWSGateRunner(t, AWS{AllowSensitiveReads: []string{"*prod-secrets-bucket*"}})
	gate := r.awsReadGate()

	if err := gate([]string{"s3", "ls", "s3://prod-secrets-bucket"}); err != nil {
		t.Fatalf("config-allowed read was denied: %v", err)
	}
	rows := readAuditRows(t, logPath)
	if len(rows) != 1 {
		t.Fatalf("want 1 allowed-marker row, got %d", len(rows))
	}
	if rows[0].Verb != "ops.aws.read.allowed" {
		t.Errorf("allowed row verb = %q, want ops.aws.read.allowed", rows[0].Verb)
	}
	if rows[0].Decision != audit.DecisionAccept {
		t.Errorf("allowed row decision = %q, want %q", rows[0].Decision, audit.DecisionAccept)
	}
}

// TestAWSReadGate_AllowViaEnv: the per-invocation env override permits one
// sensitive read without touching config.
func TestAWSReadGate_AllowViaEnv(t *testing.T) {
	r, logPath := newAWSGateRunner(t, AWS{})
	t.Setenv(awsAllowSensitiveReadEnv, "1")
	gate := r.awsReadGate()

	if err := gate([]string{"s3api", "head-object", "--bucket", "prod-secrets"}); err != nil {
		t.Fatalf("env-allowed read was denied: %v", err)
	}
	rows := readAuditRows(t, logPath)
	if len(rows) != 1 || rows[0].Verb != "ops.aws.read.allowed" {
		t.Fatalf("env-allow did not write the allowed-marker row: %v", rows)
	}
}

// TestAWSReadGate_ConfigPatternsReplaceDefaults: a non-empty
// SensitiveReadPatterns replaces the baked-in defaults wholesale.
func TestAWSReadGate_ConfigPatternsReplaceDefaults(t *testing.T) {
	r, _ := newAWSGateRunner(t, AWS{SensitiveReadPatterns: []string{"*my-crown-jewels*"}})
	gate := r.awsReadGate()

	// A default-sensitive name is no longer flagged once defaults are replaced.
	if err := gate([]string{"s3", "ls", "s3://prod-secrets-bucket"}); err != nil {
		t.Errorf("default pattern still fired after config replaced it: %v", err)
	}
	// The configured pattern fires.
	if err := gate([]string{"s3", "ls", "s3://my-crown-jewels"}); err == nil {
		t.Error("configured sensitive pattern did not fire")
	}
}
