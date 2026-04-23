# 2. Mutating-verb classifier is prefix-based and will misclassify

Category: Known bugs and rough edges

`cmd/gen-passthrough/main.go classifyVerb` uses a hand-maintained prefix
list. New AWS services will ship verbs with prefixes we haven't thought of
("promote-", "publish-", etc.). Kubernetes alpha verbs might slip through.
Consequence: a mutating verb classified as ReadOnly silently loses token
gating. Worth auditing once a week per `make update-fixtures` output diff.
Flag in the features.md test plan for feature #13.

# Decision

 It sounds like what we can / should do is maintain a list of -every- command, even the ones we filter out, so we know when a new one is added?

Yes, do this via

Snapshot with the classification attached. Same file, but each line is aws ec2 promote-read-replica READONLY. Now the diff tells you not just "new verb appeared" but "new verb appeared and here's how classifyVerb labeled it" - much faster to eyeball for wrong-direction misses.
