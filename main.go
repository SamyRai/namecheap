package main

import (
	"os"

	"go.glpx.pro/zonekit/cmd"
)

func main() {
	// cmd.Execute prints the error itself (format depends on --output, which
	// only cobra has parsed by then), so main only needs to set the exit code.
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
