package main

import (
	"fmt"
	"os"

	"github.com/donmclean/ctx/pkg/ctx"
)

func main() {
	if err := ctx.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}