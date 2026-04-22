package main

import "testing"

// TestClassifyVerb is a golden-style enumeration. Each entry is a CLI path
// (leaf last) and the expected Mutating classification. When this test
// fails on a manifest refresh it is almost always because an upstream CLI
// added a verb with a prefix classifyVerb does not yet recognize. Add the
// new prefix, add a case here, and regenerate.
//
// Bias: prefer Mutating when in doubt. A false positive just prompts for
// a token. A false negative silently drops the policy gate.
func TestClassifyVerb(t *testing.T) {
	cases := []struct {
		name     string
		path     []string
		mutating bool
	}{
		// --- Clear mutations: AWS-style hyphenated prefixes ---
		{"aws create-hosted-zone", []string{"route53", "create-hosted-zone"}, true},
		{"aws delete-hosted-zone", []string{"route53", "delete-hosted-zone"}, true},
		{"aws update-health-check", []string{"route53", "update-health-check"}, true},
		{"aws put-object", []string{"s3api", "put-object"}, true},
		{"aws put-parameter", []string{"ssm", "put-parameter"}, true},
		{"aws change-resource-record-sets", []string{"route53", "change-resource-record-sets"}, true},
		{"aws associate-vpc", []string{"route53", "associate-vpc-with-hosted-zone"}, true},
		{"aws disassociate-vpc", []string{"route53", "disassociate-vpc-from-hosted-zone"}, true},
		{"aws activate-ksk", []string{"route53", "activate-key-signing-key"}, true},
		{"aws deactivate-ksk", []string{"route53", "deactivate-key-signing-key"}, true},
		{"aws enable-dnssec", []string{"route53", "enable-hosted-zone-dnssec"}, true},
		{"aws disable-dnssec", []string{"route53", "disable-hosted-zone-dnssec"}, true},
		{"aws register-task", []string{"ssm", "register-task-with-maintenance-window"}, true},
		{"aws deregister-task", []string{"ssm", "deregister-task-from-maintenance-window"}, true},
		{"aws send-command", []string{"ssm", "send-command"}, true},
		{"aws send-automation-signal", []string{"ssm", "send-automation-signal"}, true},
		{"aws resume-session", []string{"ssm", "resume-session"}, true},
		{"aws terminate-session", []string{"ssm", "terminate-session"}, true},
		{"aws stop-automation-execution", []string{"ssm", "stop-automation-execution"}, true},
		{"aws cancel-command", []string{"ssm", "cancel-command"}, true},
		{"aws reset-service-setting", []string{"ssm", "reset-service-setting"}, true},
		{"aws restore-object", []string{"s3api", "restore-object"}, true},
		{"aws copy-object", []string{"s3api", "copy-object"}, true},
		{"aws upload-part", []string{"s3api", "upload-part"}, true},
		{"aws upload-part-copy", []string{"s3api", "upload-part-copy"}, true},
		{"aws abort-multipart-upload", []string{"s3api", "abort-multipart-upload"}, true},
		{"aws complete-multipart-upload", []string{"s3api", "complete-multipart-upload"}, true},
		{"aws remove-tags-from-resource", []string{"ssm", "remove-tags-from-resource"}, true},
		{"aws unlabel-parameter-version", []string{"ssm", "unlabel-parameter-version"}, true},
		// Hypothetical but-plausible AWS verbs the old classifier missed.
		{"aws promote-read-replica", []string{"rds", "promote-read-replica"}, true},
		{"aws publish-version", []string{"lambda", "publish-version"}, true},
		{"aws attach-policy", []string{"iam", "attach-policy"}, true},
		{"aws detach-policy", []string{"iam", "detach-policy"}, true},
		{"aws import-key-pair", []string{"ec2", "import-key-pair"}, true},
		{"aws export-snapshot", []string{"ec2", "export-snapshot"}, true},
		{"aws reboot-instance", []string{"ec2", "reboot-instances"}, true},
		{"aws accept-peering", []string{"ec2", "accept-vpc-peering-connection"}, true},
		{"aws reject-peering", []string{"ec2", "reject-vpc-peering-connection"}, true},
		{"aws tag-resource", []string{"ssm", "tag-resource"}, true},
		{"aws untag-resource", []string{"ssm", "untag-resource"}, true},

		// --- Clear reads ---
		{"aws list-hosted-zones", []string{"route53", "list-hosted-zones"}, false},
		{"aws get-hosted-zone", []string{"route53", "get-hosted-zone"}, false},
		{"aws describe-instances", []string{"ec2", "describe-instances"}, false},
		{"aws head-bucket", []string{"s3api", "head-bucket"}, false},
		{"aws head-object", []string{"s3api", "head-object"}, false},
		{"aws test-dns-answer", []string{"route53", "test-dns-answer"}, false},

		// --- kubectl bare verbs ---
		{"kubectl apply", []string{"apply"}, true},
		{"kubectl create", []string{"create"}, true},
		{"kubectl delete", []string{"delete"}, true},
		{"kubectl patch", []string{"patch"}, true},
		{"kubectl replace", []string{"replace"}, true},
		{"kubectl edit", []string{"edit"}, true},
		{"kubectl label", []string{"label"}, true},
		{"kubectl annotate", []string{"annotate"}, true},
		{"kubectl scale", []string{"scale"}, true},
		{"kubectl autoscale", []string{"autoscale"}, true},
		{"kubectl taint", []string{"taint"}, true},
		{"kubectl cordon", []string{"cordon"}, true},
		{"kubectl uncordon", []string{"uncordon"}, true},
		{"kubectl drain", []string{"drain"}, true},
		{"kubectl expose", []string{"expose"}, true},
		{"kubectl run", []string{"run"}, true},
		{"kubectl set", []string{"set"}, true},

		// kubectl reads
		{"kubectl describe", []string{"describe"}, false},
		{"kubectl logs", []string{"logs"}, false},
		{"kubectl events", []string{"events"}, false},
		{"kubectl top", []string{"top"}, false},
		{"kubectl api-resources", []string{"api-resources"}, false},
		{"kubectl api-versions", []string{"api-versions"}, false},
		{"kubectl cluster-info", []string{"cluster-info"}, false},

		// kubectl parented leaves
		{"kubectl rollout restart", []string{"rollout", "restart"}, true},
		{"kubectl rollout undo", []string{"rollout", "undo"}, true},
		{"kubectl rollout pause", []string{"rollout", "pause"}, true},
		{"kubectl rollout resume", []string{"rollout", "resume"}, true},
		{"kubectl rollout status", []string{"rollout", "status"}, false},
		{"kubectl rollout history", []string{"rollout", "history"}, false},
		{"kubectl certificate approve", []string{"certificate", "approve"}, true},
		{"kubectl certificate deny", []string{"certificate", "deny"}, true},
		{"kubectl auth reconcile", []string{"auth", "reconcile"}, true},
		{"kubectl auth can-i", []string{"auth", "can-i"}, false},
		{"kubectl auth whoami", []string{"auth", "whoami"}, false},
		{"kubectl config use-context", []string{"config", "use-context"}, true},
		{"kubectl config unset", []string{"config", "unset"}, true},
		{"kubectl config set", []string{"config", "set"}, true},
		{"kubectl config set-cluster", []string{"config", "set-cluster"}, true},
		{"kubectl config set-context", []string{"config", "set-context"}, true},
		{"kubectl config delete-context", []string{"config", "delete-context"}, true},
		{"kubectl config get-contexts", []string{"config", "get-contexts"}, false},
		{"kubectl config current-context", []string{"config", "current-context"}, false},
		{"kubectl config view", []string{"config", "view"}, false},

		// --- gh ---
		{"gh pr merge", []string{"pr", "merge"}, true},
		{"gh pr close", []string{"pr", "close"}, true},
		{"gh pr reopen", []string{"pr", "reopen"}, true},
		{"gh pr lock", []string{"pr", "lock"}, true},
		{"gh pr unlock", []string{"pr", "unlock"}, true},
		{"gh pr ready", []string{"pr", "ready"}, true},
		{"gh pr checkout", []string{"pr", "checkout"}, true},
		{"gh pr create", []string{"pr", "create"}, true},
		{"gh pr edit", []string{"pr", "edit"}, true},
		{"gh pr comment", []string{"pr", "comment"}, true},
		{"gh pr review", []string{"pr", "review"}, true},
		{"gh pr list", []string{"pr", "list"}, false},
		{"gh pr view", []string{"pr", "view"}, false},
		{"gh pr diff", []string{"pr", "diff"}, false},
		{"gh pr status", []string{"pr", "status"}, false},
		{"gh pr checks", []string{"pr", "checks"}, false},
		{"gh issue pin", []string{"issue", "pin"}, true},
		{"gh issue unpin", []string{"issue", "unpin"}, true},
		{"gh issue transfer", []string{"issue", "transfer"}, true},
		{"gh issue develop", []string{"issue", "develop"}, false}, // branch-create side effect, but "develop" is not a known verb; flag
		{"gh workflow run", []string{"workflow", "run"}, true},
		{"gh workflow enable", []string{"workflow", "enable"}, true},
		{"gh workflow disable", []string{"workflow", "disable"}, true},
		{"gh workflow list", []string{"workflow", "list"}, false},
		{"gh workflow view", []string{"workflow", "view"}, false},
		{"gh run rerun", []string{"run", "rerun"}, true},
		{"gh run cancel", []string{"run", "cancel"}, true},
		{"gh run delete", []string{"run", "delete"}, true},
		{"gh run list", []string{"run", "list"}, false},
		{"gh run view", []string{"run", "view"}, false},
		{"gh run watch", []string{"run", "watch"}, false},
		// download fetches artifacts to disk. ReadOnly from the remote's
		// POV. Listed explicitly to document the choice.
		{"gh run download", []string{"run", "download"}, false},
		{"gh release upload", []string{"release", "upload"}, true},
		{"gh release download", []string{"release", "download"}, false},
		{"gh release create", []string{"release", "create"}, true},
		{"gh release delete", []string{"release", "delete"}, true},
		{"gh release list", []string{"release", "list"}, false},
		{"gh release view", []string{"release", "view"}, false},
		{"gh repo archive", []string{"repo", "archive"}, true},
		{"gh repo unarchive", []string{"repo", "unarchive"}, true},
		{"gh repo fork", []string{"repo", "fork"}, true},
		{"gh repo clone", []string{"repo", "clone"}, true},
		{"gh repo rename", []string{"repo", "rename"}, true},
		{"gh repo sync", []string{"repo", "sync"}, true},
		{"gh repo edit", []string{"repo", "edit"}, true},
		{"gh repo view", []string{"repo", "view"}, false},
		{"gh repo list", []string{"repo", "list"}, false},
		{"gh secret set", []string{"secret", "set"}, true},
		{"gh secret delete", []string{"secret", "delete"}, true},
		{"gh secret list", []string{"secret", "list"}, false},
		{"gh search repos", []string{"search", "repos"}, false},

		// --- aws s3 short forms ---
		{"aws s3 cp", []string{"s3", "cp"}, true},
		{"aws s3 mv", []string{"s3", "mv"}, true},
		{"aws s3 rm", []string{"s3", "rm"}, true},
		{"aws s3 rb", []string{"s3", "rb"}, true},
		{"aws s3 mb", []string{"s3", "mb"}, true},
		{"aws s3 sync", []string{"s3", "sync"}, true},
		{"aws s3 ls", []string{"s3", "ls"}, false},
		// Ambiguous, flagged: presign generates a signed URL. Server state
		// unchanged, so ReadOnly. The URL itself can be used for writes
		// later; that is its own policy question.
		{"aws s3 presign", []string{"s3", "presign"}, false},

		// --- Ambiguous middle ground (explicit expected value) ---
		// sts assume-role returns creds; no server-side state change.
		// Classified ReadOnly; noted in comments.
		{"aws sts assume-role", []string{"sts", "assume-role"}, false},
		{"aws sts assume-role-with-saml", []string{"sts", "assume-role-with-saml"}, false},
		{"aws sts decode-authorization-message", []string{"sts", "decode-authorization-message"}, false},
		{"aws sts get-caller-identity", []string{"sts", "get-caller-identity"}, false},
		// s3api select-object-content runs a query; no mutation.
		{"aws s3api select-object-content", []string{"s3api", "select-object-content"}, false},
		// kubectl diff/wait are passive inspections.
		{"kubectl diff", []string{"diff"}, false},
		{"kubectl wait", []string{"wait"}, false},
		{"aws route53 wait", []string{"route53", "wait"}, false},

		// --- Group / parent nodes (always ReadOnly) ---
		{"empty path", []string{}, false},
		{"aws route53 group", []string{"route53"}, false},
		{"gh pr group", []string{"pr"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyVerb(tc.path)
			if got != tc.mutating {
				t.Errorf("classifyVerb(%v) = %v, want %v", tc.path, got, tc.mutating)
			}
		})
	}
}
