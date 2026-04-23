# 17. The generated pass-through surface is opinionated by scope

Category: Open questions

`configs/scopes/aws.yaml` allows only route53/s3/s3api/ssm/sts. If Kai
suddenly needs `aws lambda list-functions`, they have to edit the scope
yaml, `make update-fixtures`, `make install`. That's a multi-minute
flow for a one-off query. Maybe a `coily aws raw ...` escape hatch? But
that defeats the whole design. Maybe a "bring your own aws" allowlist in
config where the user can add services without rebuilding.

# decision

no escape hatches is literally the entire design
