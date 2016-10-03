package main

import (
	. "fmt"
	"os"

	"github.com/inconshreveable/log15"
	"github.com/spacemonkeygo/errors"

	// Register all implementations
	_ "flywheel.io/deja/flak/features"
)

func main() {
	log := log15.New()
	_ = log

	// Hackaround: prevent log-on-creation errors
	errors.LogMethod = func(format string, v ...interface{}) {
		Println("Error created")
	}

	// Remove name of binary
	args := os.Args[1:]

	if len(args) <= 0 {
		Println("Usage: scientist {init|use|import|run|frun|export|login|ls|download}")
		os.Exit(0)
	}

	command := args[0]
	args = args[1:]

	switch command {
	case "init":
		break
	case "use":
		project := Setup()
		project.Use(args)
	case "import":
		project := Setup()
		project.Import(args)
	case "run":
		project := Setup()
		project.Run(args)
	case "frun":
		project := Setup()
		project.Frun(args)
	case "export":
		project := Setup()
		project.Export(args)
	case "login":
		Login(args)
	case "ls":
		Ls(args)
	case "download":
		Download(args)

	default:
		Println("Unknown command")
		os.Exit(1)
	}
}
