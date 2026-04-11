package main

import (
	"os"

	"github.com/xenos76/aws-probe/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
