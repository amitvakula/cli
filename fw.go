package main

import (
	"os"

	"flywheel.io/fw/command"
	. "flywheel.io/fw/util"
)

var Version = "4.1.0"
var BuildHash = "dev"
var BuildDate = "dev"

func main() {
	defer GracefulRecover()

	err := command.BuildCommand(Version, BuildHash, BuildDate).Execute()
	if err != nil {
		os.Exit(-1)
	}
}
