package main

import (
	"github.com/coilysiren/coily/pkg/ops/gh"
)

func init() {
	rt := getRuntime()
	registerCommand(gh.Command(rt.runner, rt.issuer, rt.audit))
}
