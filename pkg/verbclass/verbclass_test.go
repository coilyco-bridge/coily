package verbclass_test

import (
	"testing"

	"github.com/coilysiren/coily/pkg/verbclass"
)

// TestClassify is a golden enumeration. Same shape as the old
// gen-passthrough TestClassifyVerb but with a third bucket. When this
// fails on a manifest refresh, an upstream CLI added a verb whose prefix
// the classifier does not yet recognize. Add to the prefix table, add a
// case here, regenerate.
func TestClassify(t *testing.T) {
	cases := []struct {
		name   string
		path   []string
		bucket verbclass.Bucket
	}{
		// --- Reads ---
		{"aws list-hosted-zones", []string{"route53", "list-hosted-zones"}, verbclass.Read},
		{"aws get-hosted-zone", []string{"route53", "get-hosted-zone"}, verbclass.Read},
		{"aws describe-instances", []string{"ec2", "describe-instances"}, verbclass.Read},
		{"aws head-bucket", []string{"s3api", "head-bucket"}, verbclass.Read},
		{"aws head-object", []string{"s3api", "head-object"}, verbclass.Read},
		{"aws test-dns-answer", []string{"route53", "test-dns-answer"}, verbclass.Read},
		{"aws sts assume-role", []string{"sts", "assume-role"}, verbclass.Read},
		{"aws sts get-caller-identity", []string{"sts", "get-caller-identity"}, verbclass.Read},
		{"aws s3 ls", []string{"s3", "ls"}, verbclass.Read},
		{"aws s3 presign", []string{"s3", "presign"}, verbclass.Read},
		{"aws s3api select-object-content", []string{"s3api", "select-object-content"}, verbclass.Read},

		// --- Writes ---
		{"aws create-hosted-zone", []string{"route53", "create-hosted-zone"}, verbclass.Write},
		{"aws update-health-check", []string{"route53", "update-health-check"}, verbclass.Write},
		{"aws put-object", []string{"s3api", "put-object"}, verbclass.Write},
		{"aws change-resource-record-sets", []string{"route53", "change-resource-record-sets"}, verbclass.Write},
		{"aws associate-vpc", []string{"route53", "associate-vpc-with-hosted-zone"}, verbclass.Write},
		{"aws disassociate-vpc", []string{"route53", "disassociate-vpc-from-hosted-zone"}, verbclass.Write},
		{"aws activate-ksk", []string{"route53", "activate-key-signing-key"}, verbclass.Write},
		{"aws enable-dnssec", []string{"route53", "enable-hosted-zone-dnssec"}, verbclass.Write},
		{"aws send-command", []string{"ssm", "send-command"}, verbclass.Write},
		{"aws restore-object", []string{"s3api", "restore-object"}, verbclass.Write},
		{"aws copy-object", []string{"s3api", "copy-object"}, verbclass.Write},
		{"aws upload-part", []string{"s3api", "upload-part"}, verbclass.Write},
		{"aws abort-multipart-upload", []string{"s3api", "abort-multipart-upload"}, verbclass.Write},
		{"aws tag-resource", []string{"ssm", "tag-resource"}, verbclass.Write},
		{"aws untag-resource", []string{"ssm", "untag-resource"}, verbclass.Write},
		{"aws attach-policy", []string{"iam", "attach-policy"}, verbclass.Write},
		{"aws detach-policy", []string{"iam", "detach-policy"}, verbclass.Write},
		{"aws s3 cp", []string{"s3", "cp"}, verbclass.Write},
		{"aws s3 mv", []string{"s3", "mv"}, verbclass.Write},
		{"aws s3 mb", []string{"s3", "mb"}, verbclass.Write},
		{"aws s3 sync", []string{"s3", "sync"}, verbclass.Write},

		// --- Deletes ---
		{"aws delete-hosted-zone", []string{"route53", "delete-hosted-zone"}, verbclass.Delete},
		{"aws delete-traffic-policy", []string{"route53", "delete-traffic-policy"}, verbclass.Delete},
		{"aws remove-tags-from-resource", []string{"ssm", "remove-tags-from-resource"}, verbclass.Delete},
		{"aws terminate-session", []string{"ssm", "terminate-session"}, verbclass.Delete},
		{"aws revoke-token", []string{"sso", "revoke-token"}, verbclass.Delete},
		{"aws deregister-task", []string{"ssm", "deregister-task-from-maintenance-window"}, verbclass.Delete},
		{"aws s3 rm", []string{"s3", "rm"}, verbclass.Delete},
		{"aws s3 rb", []string{"s3", "rb"}, verbclass.Delete},

		// --- kubectl bare verbs ---
		{"kubectl apply", []string{"apply"}, verbclass.Write},
		{"kubectl create", []string{"create"}, verbclass.Write},
		{"kubectl delete", []string{"delete"}, verbclass.Delete},
		{"kubectl patch", []string{"patch"}, verbclass.Write},
		{"kubectl replace", []string{"replace"}, verbclass.Write},
		{"kubectl edit", []string{"edit"}, verbclass.Write},
		{"kubectl label", []string{"label"}, verbclass.Write},
		{"kubectl annotate", []string{"annotate"}, verbclass.Write},
		{"kubectl scale", []string{"scale"}, verbclass.Write},
		{"kubectl autoscale", []string{"autoscale"}, verbclass.Write},
		{"kubectl drain", []string{"drain"}, verbclass.Write},

		// kubectl reads
		{"kubectl describe", []string{"describe"}, verbclass.Read},
		{"kubectl logs", []string{"logs"}, verbclass.Read},
		{"kubectl events", []string{"events"}, verbclass.Read},
		{"kubectl top", []string{"top"}, verbclass.Read},

		// kubectl parented
		{"kubectl rollout restart", []string{"rollout", "restart"}, verbclass.Write},
		{"kubectl rollout status", []string{"rollout", "status"}, verbclass.Read},
		{"kubectl rollout history", []string{"rollout", "history"}, verbclass.Read},
		{"kubectl certificate approve", []string{"certificate", "approve"}, verbclass.Write},
		{"kubectl auth reconcile", []string{"auth", "reconcile"}, verbclass.Write},
		{"kubectl auth can-i", []string{"auth", "can-i"}, verbclass.Read},
		{"kubectl config use-context", []string{"config", "use-context"}, verbclass.Write},
		{"kubectl config get-contexts", []string{"config", "get-contexts"}, verbclass.Read},
		{"kubectl config view", []string{"config", "view"}, verbclass.Read},

		// --- gh ---
		{"gh pr merge", []string{"pr", "merge"}, verbclass.Write},
		{"gh pr close", []string{"pr", "close"}, verbclass.Write},
		{"gh pr list", []string{"pr", "list"}, verbclass.Read},
		{"gh pr view", []string{"pr", "view"}, verbclass.Read},
		{"gh issue pin", []string{"issue", "pin"}, verbclass.Write},
		{"gh workflow run", []string{"workflow", "run"}, verbclass.Write},
		{"gh workflow list", []string{"workflow", "list"}, verbclass.Read},
		{"gh run rerun", []string{"run", "rerun"}, verbclass.Write},
		{"gh run delete", []string{"run", "delete"}, verbclass.Delete},
		{"gh run download", []string{"run", "download"}, verbclass.Read},
		{"gh release upload", []string{"release", "upload"}, verbclass.Write},
		{"gh release create", []string{"release", "create"}, verbclass.Write},
		{"gh release delete", []string{"release", "delete"}, verbclass.Delete},
		{"gh repo archive", []string{"repo", "archive"}, verbclass.Write},
		{"gh repo fork", []string{"repo", "fork"}, verbclass.Write},
		{"gh secret set", []string{"secret", "set"}, verbclass.Write},
		{"gh secret delete", []string{"secret", "delete"}, verbclass.Delete},

		// --- empty path is the safe default (Read). Group nodes never
		// reach Classify in practice (gen-passthrough only calls it for
		// leaves) so their classification is undefined. ---
		{"empty path", []string{}, verbclass.Read},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := verbclass.Classify(tc.path)
			if got != tc.bucket {
				t.Errorf("Classify(%v) = %s, want %s", tc.path, got, tc.bucket)
			}
		})
	}
}

func TestService(t *testing.T) {
	cases := []struct {
		path []string
		want string
	}{
		{[]string{"route53", "list-hosted-zones"}, "route53"},
		{[]string{"s3", "cp"}, "s3"},
		{[]string{"pr", "merge"}, "pr"},
		{[]string{"apply"}, "apply"},
		{[]string{}, ""},
	}
	for _, tc := range cases {
		if got := verbclass.Service(tc.path); got != tc.want {
			t.Errorf("Service(%v) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestBucketString(t *testing.T) {
	if verbclass.Read.String() != "read" {
		t.Errorf("Read.String() = %q", verbclass.Read.String())
	}
	if verbclass.Write.String() != "write" {
		t.Errorf("Write.String() = %q", verbclass.Write.String())
	}
	if verbclass.Delete.String() != "delete" {
		t.Errorf("Delete.String() = %q", verbclass.Delete.String())
	}
}
