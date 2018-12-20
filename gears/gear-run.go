package gears

import (
	"github.com/docker/docker/client"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearRun(client *api.Client, docker *client.Client, apiKey string, gear *api.Gear, config map[string]interface{}, files map[string]string, apiKeyInputs []string) {
	d := &D{docker}

	// Job info
	info := ParseGearBuilerInformation(gear)
	env, command, invocation := CreateInvocationComponents(gear, config, files, apiKey, apiKeyInputs)

	// Prep filesystem
	mounts, cleanupFiles := PrepareLocalRunFiles(gear, invocation, files)

	// Run job
	d.EnsureImageLocal(info.Image)
	containerID, cleanupContainer := d.CreateContainerForGear(info.Image, env, command, mounts)

	// Track
	retcode := d.ObserveContainer(containerID)
	Println()
	cleanupContainer()
	cleanupFiles()
	if retcode != 0 {
		Println("Exit code was", retcode)
	}
}
