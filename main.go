// Package main is the entry point for the aws-probe CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/xenos76/aws-probe/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
