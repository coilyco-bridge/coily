// Package verbclass classifies a verb path into one of three buckets:
// Read, Write, or Delete. Used by gen-passthrough at code-generation time
// and by hand-written verbs at registration time to compute the policy
// scope that gates a verb invocation.
//
// The three-bucket model replaces the older binary READONLY / MUTATING
// classifier. See docs/decisions on token-scoping for the rationale.
//
// Bias: when a verb is ambiguous, return Write rather than Read. A Read
// scope is a subset of a Write scope (a Write token grants Read), so over-
// classifying as Write costs an agent one extra token issuance, while
// under-classifying as Read silently drops the policy gate.
//
// Delete is its own bucket and is not subsumed by Write. To grant all
// three, issue a token with both `:write` and `:delete` scopes.
package verbclass

// Bucket is the three-bucket classification of a verb.
type Bucket int

const (
	// Read covers describe, list, get, head, view, status, and similar
	// read-only inspections.
	Read Bucket = iota
	// Write covers create, update, upsert, tag, label, attach, set,
	// patch, and similar mutations that change remote state but do not
	// destroy resources.
	Write
	// Delete covers destructive removals (delete, remove, destroy,
	// terminate, purge, revoke). Kept separate from Write because the
	// blast radius of a delete is categorically different from the
	// blast radius of an update.
	Delete
)

// String returns the lowercase scope-suffix form ("read", "write",
// "delete").
func (b Bucket) String() string {
	switch b {
	case Read:
		return "read"
	case Write:
		return "write"
	case Delete:
		return "delete"
	}
	return "unknown"
}

// Classify returns the bucket for a verb path. Group / parent nodes (any
// path with fewer than the leaf depth a coily verb requires) return Read,
// because group nodes never run an action and need a safe default.
//
// path is the slice of CLI tokens after the binary, e.g.
// ["route53", "delete-hosted-zone"] for `aws route53 delete-hosted-zone`,
// or ["rollout", "restart"] for `kubectl rollout restart`.
//
// When extending this function, prefer adding to the Delete and Read
// tables before expanding the Write fallback. Remember the bias: ambiguous
// verbs default to Write.
func Classify(path []string) Bucket {
	if len(path) == 0 {
		return Read
	}
	leaf := path[len(path)-1]

	if b, ok := classifyParented(path); ok {
		return b
	}
	if isDelete(leaf) {
		return Delete
	}
	if isRead(leaf) {
		return Read
	}
	if isSpecialRead(path, leaf) {
		return Read
	}
	if isWrite(leaf) {
		return Write
	}
	// Default: Write. Safer to require a token than to silently allow.
	return Write
}

// parentedEntry pairs a (parent, leaf) name with its bucket. Used by
// classifyParented to disambiguate kubectl/gh-style grouped verbs where the
// bare leaf name ("restart", "approve") would otherwise be ambiguous.
type parentedEntry struct {
	parent, leaf string
	bucket       Bucket
}

var parentedTable = []parentedEntry{
	{"rollout", "restart", Write},
	{"rollout", "undo", Write},
	{"rollout", "pause", Write},
	{"rollout", "resume", Write},
	{"rollout", "status", Read},
	{"rollout", "history", Read},
	{"certificate", "approve", Write},
	{"certificate", "deny", Write},
	{"auth", "reconcile", Write},
	{"auth", "can-i", Read},
	{"auth", "whoami", Read},
	{"config", "unset", Write},
	{"config", "use-context", Write},
	{"config", "set", Write},
	{"config", "view", Read},
	{"config", "current-context", Read},
	{"config", "get-contexts", Read},
	{"workflow", "run", Write},
	{"workflow", "enable", Write},
	{"workflow", "disable", Write},
	{"workflow", "list", Read},
	{"workflow", "view", Read},
	{"run", "rerun", Write},
	{"run", "cancel", Write},
	{"run", "delete", Delete},
	{"run", "list", Read},
	{"run", "view", Read},
	{"run", "watch", Read},
	{"run", "download", Read},
	{"pr", "checkout", Write},
	{"pr", "ready", Write},
	{"pr", "comment", Write},
	{"pr", "review", Write},
	{"pr", "merge", Write},
	{"pr", "close", Write},
	{"pr", "reopen", Write},
	{"pr", "lock", Write},
	{"pr", "unlock", Write},
	{"pr", "create", Write},
	{"pr", "edit", Write},
	{"pr", "list", Read},
	{"pr", "view", Read},
	{"pr", "diff", Read},
	{"pr", "status", Read},
	{"pr", "checks", Read},
	{"issue", "comment", Write},
	{"issue", "pin", Write},
	{"issue", "unpin", Write},
	{"issue", "transfer", Write},
}

func classifyParented(path []string) (Bucket, bool) {
	if len(path) < 2 {
		return Read, false
	}
	parent, leaf := path[len(path)-2], path[len(path)-1]
	for _, pm := range parentedTable {
		if parent == pm.parent && leaf == pm.leaf {
			return pm.bucket, true
		}
	}
	return Read, false
}

var deletePrefixes = []string{
	"delete-", "remove-", "destroy-", "purge-", "terminate-",
	"revoke-", "kill-", "deregister-",
}

var deleteBare = []string{
	"delete", "remove", "destroy", "purge", "terminate",
	"revoke", "kill", "rm", "rb",
}

func isDelete(leaf string) bool {
	for _, p := range deletePrefixes {
		if hasPrefix(leaf, p) {
			return true
		}
	}
	for _, v := range deleteBare {
		if leaf == v {
			return true
		}
	}
	return false
}

var readPrefixes = []string{
	"describe-", "list-", "get-", "head-", "show-", "view-",
	"find-", "search-", "test-", "lookup-", "select-",
}

var readBare = []string{
	// kubectl reads
	"describe", "logs", "events", "top", "diff", "wait",
	"api-resources", "api-versions", "cluster-info", "version",
	"explain", "config", "auth",
	// aws / general reads
	"ls", "head", "show", "status", "view", "find", "search",
	"cat", "tail", "history", "whoami",
}

func isRead(leaf string) bool {
	for _, p := range readPrefixes {
		if hasPrefix(leaf, p) {
			return true
		}
	}
	for _, v := range readBare {
		if leaf == v {
			return true
		}
	}
	return false
}

// isSpecialRead covers leaves that defy the prefix tables. sts verbs
// (assume-role, decode-authorization-message, etc.) return creds without
// mutating server state. presign generates a signed URL locally.
func isSpecialRead(path []string, leaf string) bool {
	if len(path) >= 1 && path[0] == "sts" {
		return true
	}
	if leaf == "presign" {
		return true
	}
	return false
}

var (
	// writePrefixes is the big list, taken from the previous classifier's
	// MUTATING prefixes minus the destructive ones now in deletePrefixes.
	writePrefixes = []string{
		// CRUD-ish (writes only)
		"create-", "update-", "put-", "post-",
		"modify-", "change-", "apply-", "set-",
		"add-", "patch-", "replace-", "upsert-",
		// Lifecycle / state transitions
		"register-",
		"associate-", "disassociate-", "attach-", "detach-",
		"enable-", "disable-", "activate-", "deactivate-",
		"start-", "stop-", "restart-", "reboot-",
		"resume-", "pause-", "suspend-",
		"rotate-", "refresh-", "reset-", "recycle-",
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
		"grant-",
		// sts (handled above as a special case but kept for safety)
		"assume-",
	}

	writeBare = []string{
		// kubectl
		"apply", "create", "patch", "replace", "edit",
		"label", "annotate", "scale", "autoscale", "set",
		"taint", "cordon", "uncordon", "drain", "expose", "run",
		// aws s3 short forms (mb makes a bucket, cp/mv/sync write objects)
		"cp", "mv", "mb", "sync",
		// gh
		"merge", "close", "reopen", "lock", "unlock", "pin", "unpin",
		"transfer", "archive", "unarchive", "fork", "clone", "upload",
		"rename", "checkout", "ready", "comment", "review",
	}
)

func isWrite(leaf string) bool {
	for _, p := range writePrefixes {
		if hasPrefix(leaf, p) {
			return true
		}
	}
	for _, v := range writeBare {
		if leaf == v {
			return true
		}
	}
	return false
}

// Service returns the immediate first sub-cli below the binary, used as
// the service half of a scope string. For empty paths the binary itself
// is the only meaningful service.
func Service(path []string) string {
	if len(path) == 0 {
		return ""
	}
	return path[0]
}

// hasPrefix is a small helper that doesn't pull in strings just for one
// call. Inlinable.
func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
