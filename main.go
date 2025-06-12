package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/siyuan-infoblox/go-imports-group/pkg/cmd"
)

func main() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("unable to read build info")
		os.Exit(1)
	}
	if err := cmd.Execute(info.Main.Version); err != nil {
		os.Exit(1)
	}
}
