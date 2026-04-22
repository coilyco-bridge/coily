# 2. Mutating-verb classifier is prefix-based and will misclassify

Category: Known bugs and rough edges

`cmd/gen-passthrough/main.go classifyVerb` uses a hand-maintained prefix
list. New AWS services will ship verbs with prefixes we haven't thought of
("promote-", "publish-", etc.). Kubernetes alpha verbs might slip through.
Consequence: a mutating verb classified as ReadOnly silently loses token
gating. Worth auditing once a week per `make update-fixtures` output diff.
Flag in the features.md test plan for feature #13.
