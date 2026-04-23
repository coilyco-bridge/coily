package verbclass

import "testing"

// TestClassify is a golden-style enumeration. Each entry is a CLI path
// (leaf last) and the expected Mutating classification. When this test
// fails on a manifest refresh it is almost always because an upstream CLI
// added a verb with a prefix Classify does not yet recognize. Add the new
// prefix, add a case here, and regenerate.
//
// Bias: prefer Mutating when in doubt. A false positive just prompts for
// a token. A false negative silently drops the policy gate.
func TestClassify(t *testing.T) {
	cases := []struct {
		name     string
		path     []string
		mutating bool
	}{
		// --- Clear mutations: AWS-style hyphenated prefixes ---
		{"aws create-hosted-zone", []string{"route53", "create-hosted-zone"}, true},
		{"aws delete-hosted-zone", []string{"route53", "delete-hosted-zone"}, true},
		{"aws put-object", []string{"s3api", "put-object"}, true},
		{"aws change-resource-record-sets", []string{"route53", "change-resource-record-sets"}, true},
		{"aws associate-vpc", []string{"route53", "associate-vpc-with-hosted-zone"}, true},
		{"aws send-command", []string{"ssm", "send-command"}, true},
		{"aws promote-read-replica", []string{"rds", "promote-read-replica"}, true},
		{"aws publish-version", []string{"lambda", "publish-version"}, true},
		// Clear reads
		{"aws list-hosted-zones", []string{"route53", "list-hosted-zones"}, false},
		{"aws get-hosted-zone", []string{"route53", "get-hosted-zone"}, false},
		{"aws describe-instances", []string{"ec2", "describe-instances"}, false},
		{"aws head-bucket", []string{"s3api", "head-bucket"}, false},
		// kubectl bare verbs
		{"kubectl apply", []string{"apply"}, true},
		{"kubectl delete", []string{"delete"}, true},
		{"kubectl logs", []string{"logs"}, false},
		{"kubectl describe", []string{"describe"}, false},
		// kubectl parented
		{"kubectl rollout restart", []string{"rollout", "restart"}, true},
		{"kubectl rollout status", []string{"rollout", "status"}, false},
		{"kubectl certificate approve", []string{"certificate", "approve"}, true},
		{"kubectl auth reconcile", []string{"auth", "reconcile"}, true},
		{"kubectl auth can-i", []string{"auth", "can-i"}, false},
		{"kubectl config use-context", []string{"config", "use-context"}, true},
		{"kubectl config view", []string{"config", "view"}, false},
		// gh
		{"gh pr merge", []string{"pr", "merge"}, true},
		{"gh pr list", []string{"pr", "list"}, false},
		{"gh workflow run", []string{"workflow", "run"}, true},
		{"gh run rerun", []string{"run", "rerun"}, true},
		{"gh run list", []string{"run", "list"}, false},
		{"gh run download", []string{"run", "download"}, false},
		{"gh release upload", []string{"release", "upload"}, true},
		{"gh repo fork", []string{"repo", "fork"}, true},
		// aws s3 short forms
		{"aws s3 cp", []string{"s3", "cp"}, true},
		{"aws s3 ls", []string{"s3", "ls"}, false},
		// Group / parent nodes
		{"empty path", []string{}, false},
		{"aws route53 group", []string{"route53"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.path)
			if got != tc.mutating {
				t.Errorf("Classify(%v) = %v, want %v", tc.path, got, tc.mutating)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	if got := Label([]string{"route53", "create-hosted-zone"}); got != LabelMutating {
		t.Errorf("Label mutating = %q, want %s", got, LabelMutating)
	}
	if got := Label([]string{"route53", "list-hosted-zones"}); got != LabelReadOnly {
		t.Errorf("Label readonly = %q, want %s", got, LabelReadOnly)
	}
	if got := Label(nil); got != LabelReadOnly {
		t.Errorf("Label nil = %q, want %s", got, LabelReadOnly)
	}
}
