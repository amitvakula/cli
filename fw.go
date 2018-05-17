package main

import (
	"os"

	"flywheel.io/fw/command"
	. "flywheel.io/fw/util"
)

var Version = "5.0.3"
var BuildHash = "dev"
var BuildDate = "dev"

func main() {
	defer GracefulRecover()

	// Hack: Add -- for gear run
	if len(os.Args) >= 4 && os.Args[1] == "gear" && os.Args[2] == "run" && os.Args[3] != "--" {
		tmp := make([]string, 3)
		copy(tmp, os.Args)
		tmp = append(tmp, "--")
		os.Args = append(tmp, os.Args[3:]...)

	}

	err := command.BuildCommand(Version, BuildHash, BuildDate).Execute()
	if err != nil {
		os.Exit(-1)
	}
}
