package gears

import (
	// "os"
	"path"

	"github.com/docker/docker/client"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearUpload(client *api.Client, docker *client.Client, apiKey, domain string, gear *api.Gear) {
	d := &D{docker}
	info := ParseGearBuilerInformation(gear)

	// Add registry domain and create full image target
	imageDst := domain + "/" + path.Base(gear.Name) + ":" + gear.Version

	doc := &api.GearDoc{
		Category: api.GearCategory(info.Category),
		Gear:     gear,
	}

	ticketID := MakeGearTicketReslient(client, doc)

	// Create container and tag dat
	Println("Creating container to save local changes...")
	containerID, cleanup := d.CreateContainer(info.Image)
	d.SaveCwdIntoContainer(containerID)
	d.TagContainer(containerID, imageDst)

	Println("Publishing to", imageDst, "...")
	token := d.LoginToRegistry(client, domain, apiKey)
	digest := d.PushImage(imageDst, token)

	Println("Adding gear to server...")
	FinishGearTicket(client, ticketID, path.Base(gear.Name), digest)
	cleanup()

	Println()
	Println()
	Println("Done! You should now see your gear in the Flywheel web interface,")
	Println("or find it with `fw job list-gears`.")
}
