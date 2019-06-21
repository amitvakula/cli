package main

import (
	"os"

	"flywheel.io/fw/command"
	. "flywheel.io/fw/util"
)

var Version = "8.5.0"
var BuildHash = "dev"
var BuildDate = "dev"

func InvokeCommand(args []string) int {
	defer GracefulRecover()

	// Hack: Add -- for gear run
	if len(args) >= 4 && args[1] == "gear" && args[2] == "run" && args[3] != "--" {
		tmp := make([]string, 3)
		copy(tmp, args)
		tmp = append(tmp, "--")
		args = append(tmp, args[3:]...)
	}

	cmd := command.BuildCommand(Version, BuildHash, BuildDate)
	cmd.SetArgs(args[1:])

	err := cmd.Execute()
	if err != nil {
		return -1
	}
	return 0
}

func main() {
	InvokeCommand(os.Args)
}
