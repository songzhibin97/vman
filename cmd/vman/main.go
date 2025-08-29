package main

import (
	"os"

	"github.com/songzhibin97/vman/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
