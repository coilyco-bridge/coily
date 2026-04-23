// Package verbclass holds the read-only / mutating heuristic used by the
// pass-through codegen. Pulled out into its own package so both
// cmd/gen-passthrough (which writes the policy.Kind into generated code) and
// cmd/subcli-scope (which writes a per-tool classification snapshot for diff
// review) call the same code instead of carrying duplicated prefix lists.
//
// When adding new cases here, bias toward Mutating. A false positive just
// makes an agent ask for a token. A false negative silently drops the
// policy gate.
package verbclass

import "strings"

// Classify returns true for Mutating. Heuristic: the leaf verb name starts
// with one of a known set of mutation prefixes, or matches an exact bare
// verb, or is a leaf under a mutating group (e.g. kubectl rollout restart).
// Parent/group nodes return false. Only leaves matter.
func Classify(path []string) bool {
	if len(path) == 0 {
		return false
	}
	leaf := path[len(path)-1]

	// Mutating leaves, considered only in the context of a specific parent
	// group. Used for kubectl's grouped verbs where the bare leaf name
	// ("restart", "resume", "approve") would otherwise be ambiguous.
	type parented struct{ parent, leaf string }
	parentedMutating := []parented{
		{"rollout", "restart"},
		{"rollout", "undo"},
		{"rollout", "pause"},
		{"rollout", "resume"},
		{"certificate", "approve"},
		{"certificate", "deny"},
		{"auth", "reconcile"},
		{"config", "unset"},
		{"config", "use-context"},
		{"workflow", "run"},
		{"workflow", "enable"},
		{"workflow", "disable"},
		{"run", "rerun"},
		{"run", "cancel"},
		{"pr", "checkout"},
		{"pr", "ready"},
		{"pr", "comment"},
		{"pr", "review"},
		{"issue", "comment"},
	}
	if len(path) >= 2 {
		parent := path[len(path)-2]
		for _, pm := range parentedMutating {
			if parent == pm.parent && leaf == pm.leaf {
				return true
			}
		}
	}

	// Mutating prefixes. Matched against the leaf verb. The trailing dash
	// keeps "get-" from swallowing "getter" (hypothetical) but also lets
	// the bare form match via the exactBareVerbs list below.
	prefixes := []string{
		// CRUD-ish
		"create-", "delete-", "update-", "put-", "post-",
		"modify-", "change-", "apply-", "set-", "remove-",
		"add-", "patch-", "replace-",
		// Lifecycle / state transitions
		"register-", "deregister-",
		"associate-", "disassociate-", "attach-", "detach-",
		"enable-", "disable-", "activate-", "deactivate-",
		"start-", "stop-", "restart-", "reboot-", "terminate-",
		"resume-", "pause-", "suspend-",
		"rotate-", "refresh-", "reset-", "revoke-", "recycle-",
		"cancel-", "abort-", "complete-",
		"promote-", "demote-", "publish-", "unpublish-",
		"accept-", "reject-", "approve-", "deny-",
		// Data movement
		"move-", "copy-", "rename-", "restore-", "backup-",
		"upload-", "download-", "import-", "export-",
		"send-", "sync-",
		// Versioning / upgrade
		"upgrade-", "downgrade-", "rollback-", "rollforward-",
		// Tagging / labeling
		"tag-", "untag-", "label-", "unlabel-", "annotate-",
		// Auth-ish writes
		"grant-", "deny-",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(leaf, p) {
			return true
		}
	}

	// Bare-verb leaves. kubectl- and gh-style commands where the whole
	// leaf is the verb.
	exactBareVerbs := []string{
		// kubectl
		"apply", "create", "delete", "patch", "replace", "edit",
		"label", "annotate", "scale", "autoscale", "set",
		"taint", "cordon", "uncordon", "drain", "expose", "run",
		// aws s3 short forms
		"cp", "mv", "rm", "rb", "mb", "sync",
		// gh
		"merge", "close", "reopen", "lock", "unlock", "pin", "unpin",
		"transfer", "archive", "unarchive", "fork", "clone", "upload",
		"rename",
	}
	for _, v := range exactBareVerbs {
		if leaf == v {
			return true
		}
	}

	return false
}

// LabelMutating and LabelReadOnly are the labels used in snapshot files. Kept
// as exported constants so callers (and tests) can compare without typo risk.
const (
	LabelMutating = "MUTATING"
	LabelReadOnly = "READONLY"
)

// Label returns LabelMutating or LabelReadOnly for the given path.
// Convenience for snapshot files where a stable string is wanted.
func Label(path []string) string {
	if Classify(path) {
		return LabelMutating
	}
	return LabelReadOnly
}
