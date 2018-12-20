package gears

import (
	"github.com/docker/docker/client"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearCreate(client *api.Client, docker *client.Client) {
	d := &D{docker}

	// User info
	realName := FetchName(client)
	label, name, imageName, gType := gearCreatePrompts()

	// Image
	d.EnsureImageLocal(imageName)
	_, env := d.GetImageDetails(imageName)

	// Translate info to action
	defaultManifest, modifier := GenerateExampleManifestAndModifier(label, name, imageName, gType, env, realName)

	// Inflate
	cid, cleanup := d.CreateContainer(imageName)
	d.ExpandFlywheelFolder(imageName, cid, defaultManifest, modifier)
	cleanup()

	Println("\n")
	Println("Your gear is created and expanded to the current directory.")
	Println("Try `fw gear local` to run the gear!")
}
