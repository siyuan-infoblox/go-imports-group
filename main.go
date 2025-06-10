package main

import (
	"os"

	"github.com/siyuan-infoblox/go-imports-group/pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
