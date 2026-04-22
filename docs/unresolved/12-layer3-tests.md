# 12. Layer 3 integration tests (end-to-end)

Category: Incomplete features

Described in my earlier Layer 1/2/3 proposal. Layer 1 (fixture golden) and
Layer 2 (whoami smoke) are built. Layer 3 needs:

- A kind cluster for kubectl verbs.
- A dedicated AWS test prefix (`/coily/test/*` SSM, a test Route53 zone).
- A sandbox GitHub repo for gh verbs.

Punted. Not urgent because the pass-through layer is thin and unit tests
on pkg/* cover the interesting logic.
