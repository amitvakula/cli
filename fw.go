package main

import (
	"os"

	"github.com/spf13/cobra"
)

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
