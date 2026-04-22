package main

import (
	"github.com/coilysiren/coily/pkg/ops/kubectl"
)

func init() {
	rt := getRuntime()
	registerCommand(kubectl.Command(rt.runner, rt.issuer, rt.audit))
}
