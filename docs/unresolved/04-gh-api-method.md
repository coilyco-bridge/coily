# 4. gh api's --method flag clashes with coily's structure

Category: Known bugs and rough edges

`gh api --method POST /graphql` is a common shape. Our generated
coily wrapper passes it through correctly, but if the user invokes it
without `--method` (relying on GET default), IsSet returns false and we
don't forward. That's fine. But certain flags like `--field key=value` may
pass through incorrectly with a single c.String value because they're
repeatable and we only capture the last one. Same fix as #1 (flag types,
including StringSliceFlag).
