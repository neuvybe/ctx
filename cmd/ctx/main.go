package main

import (
	"fmt"
	"os"

	"github.com/neuvybe/ctx/pkg/ctx"
)

func main() {
	if err := ctx.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}