package main

import (
	"github.com/coilysiren/coily/pkg/ops/aws"
)

func init() {
	rt := getRuntime()
	registerCommand(aws.Command(rt.runner, rt.issuer, rt.audit))
}
