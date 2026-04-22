# 14. Token scoping granularity

Category: Open questions

Current: each verb has a scope like `aws.route53.change-resource-record-sets`.
Alternative: broader scopes like `aws.route53.write` that cover multiple
verbs. Narrower is more annoying for Kai (more tokens to issue). Broader
is weaker (a token for "any route53 write" includes "delete-hosted-zone"
which is much nastier than "upsert one record").

Current is narrower. May want a `--scope aws.route53.*` wildcard mode
in `coily auth issue`. Not built.
