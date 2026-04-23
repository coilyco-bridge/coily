# 3. aws s3api help field is groff garbage

Category: Known bugs and rough edges

The s3api group still shows `S3API()    S3API()` in its help field
because its help text does not have a DESCRIPTION section in the normal
form. Cosmetic. Fix: extend `summary()` in cmd/subcli-scope/main.go to
pull from SYNOPSIS or NAME sections when DESCRIPTION is absent.

# Decision

Yes
