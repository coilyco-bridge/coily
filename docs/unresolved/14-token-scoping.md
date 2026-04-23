# 14. Token scoping granularity

Category: Open questions

Current: each verb has a scope like `aws.route53.change-resource-record-sets`.
Alternative: broader scopes like `aws.route53.write` that cover multiple
verbs. Narrower is more annoying for Kai (more tokens to issue). Broader
is weaker (a token for "any route53 write" includes "delete-hosted-zone"
which is much nastier than "upsert one record").

Current is narrower. May want a `--scope aws.route53.*` wildcard mode
in `coily auth issue`. Not built.

# Decision

Agree with the broader scope.

No wildcards, use an exhaustive list of verb buckets. Three distinct
buckets per service:

- `read` (includes describe, list, get, etc...)
- `write` (create, update, upsert, tag, etc...)
- `delete` (destructive removals, kept separate from write)

Scope string format: `aws.route53:read`, `aws.route53:write`,
`aws.route53:delete`.

No per-service overrides. Every service uses the same three buckets
even where individual writes have very different blast radius (e.g.
IAM create-user vs attach-policy). Parsing that difference is not
worth the complexity.

No migration path needed. Nothing is distributed yet, so existing
verb-level scopes can be replaced outright.
