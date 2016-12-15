package main

import (
	"os"

	"github.com/spf13/cobra"
)

const Version = "3.5.0-dev"

var RootCmd = &cobra.Command{
	Use:   "fw",
	Short: "Flywheel command-line interface",
}

func main() {
	// Run
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
