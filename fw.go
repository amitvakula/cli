package main

import (
	. "fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var Version = "3.7.2"
var BuildHash = "dev"
var BuildDate = "dev"

var RootCmd = &cobra.Command{
	Use:   "fw",
	Short: "Flywheel command-line interface",
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			Println(string(debug.Stack()))
			Println("Crash report:", r)
			Println("flywheel-cli version", Version)
		}
	}()

	// Run
	if err := RootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
