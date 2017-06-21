package ops

import (
	. "fmt"
	"math"
	"os"
	"strings"

	humanize "github.com/dustin/go-humanize"
	prompt "github.com/segmentio/go-prompt"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func Download(client *api.Client, upath, savePath string, force bool) {
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err, aerr := legacy.ResolvePath(client, parts)
	Check(api.Coalesce(err, aerr))
	path := result.Path

	last := path[len(path)-1]

	// This logic is dangled between download-file and download-container.
	// It should get much nicer after some SDK fiddling.

	var parent interface{}
	var name string
	size := float64(0)

	file, ok := last.(*legacy.File)
	if ok {
		parent = path[len(path)-2]
		name = file.Name

	} else {
		wat := last.(legacy.Container)

		if wat.GetType() == "group" {
			Println("Group downloads are currently not supported. Instead, you can download each project.")
			os.Exit(1)
		}

		ticketRequest := &legacy.ContainerTicketRequest{
			Nodes: []*legacy.ContainerTicketRequestElem{
				{
					Level: wat.GetType(),
					Id:    wat.GetId(),
				},
			},
			Optional: true,
		}

		ticket, _, err := legacy.GetDownloadTicket(client, ticketRequest)
		Check(err)

		// Should make this second condition cleaner...
		if !force && savePath != "--" {
			Println()
			Println("This download will be about", humanize.Bytes(ticket.Size), "comprising", ticket.FileCount, "files.")

			proceed := prompt.Confirm("Continue? (yes/no)")
			Println()
			if !proceed {
				Println("Canceled.")
				return
			}
		}

		parent = ticket
		name = "download.tar"
		size = float64(ticket.Size)
	}

	if savePath == "--" {
		_, err := legacy.Download(client, name, parent, os.Stdout)
		Check(err)
		return
	}

	if savePath == "" {
		savePath = name
	}

	resp, err := legacy.DownloadToFile(client, name, parent, savePath)
	Check(err)

	// Container downloads have content length of -1
	written := math.Max(float64(resp.ContentLength), size)

	Println("Wrote", humanize.Bytes(uint64(written)), "to", savePath)
}
