package ops

import (
	"math"
	"os"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
	prompt "github.com/segmentio/go-prompt"

	"github.com/kennygrant/sanitize"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func Download(client *api.Client, upath, savePath string, force bool, include []string, exclude []string) {
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err, aerr := legacy.ResolvePath(client, parts)
	Check(api.Coalesce(err, aerr))
	path := result.Path

	last := path[len(path)-1]

	// This logic is dangled between download-file and download-container.
	// It should get much nicer after some SDK fiddling.

	var parent interface{}
	var download, name string
	size := float64(0)

	file, ok := last.(*legacy.File)
	if ok {
		parent = path[len(path)-2]

		download = file.Name
		name = sanitize.Name(file.Name)
		prefix := ""
		suffix := ""

		splits := strings.SplitN(name, ".", 2)
		if len(splits) == 2 {
			prefix = splits[0]
			suffix = "." + splits[1]
		} else {
			prefix = name
		}

		i := 1

		for {
			if _, err := os.Stat(name); err == nil {
				name = prefix + "-" + strconv.Itoa(i) + suffix
				i++

				if i > 1000000 {
					Println("Could not find a viable filename for " + sanitize.Name(file.Name) + ".tar, check filesystem permissions?")
					Fatal(1)
				}
			} else {
				break
			}
		}

	} else {
		wat := last.(legacy.Container)

		if wat.GetType() == "group" {
			Println("Group downloads are currently not supported. Instead, you can download each project.")
			Fatal(1)
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

		if len(include) > 0 || len(exclude) > 0 {
			ticketRequest.Filters = legacy.NewContainerFilter(include, exclude)
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
		download = wat.GetName()
		name = sanitize.Name(wat.GetName()) + ".tar"
		i := 1

		for {
			if _, err := os.Stat(name); err == nil {
				name = sanitize.Name(wat.GetName()) + "-" + strconv.Itoa(i) + ".tar"
				i++

				if i > 1000000 {
					Println("Could not find a viable filename for " + sanitize.Name(wat.GetName()) + ".tar, check filesystem permissions?")
					Fatal(1)
				}
			} else {
				break
			}
		}

		size = float64(ticket.Size)
	}

	if savePath == "--" {
		_, err := legacy.Download(client, download, parent, os.Stdout)
		Check(err)
		return
	}

	if savePath == "" {
		savePath = name
	}

	resp, err := legacy.DownloadToFile(client, download, parent, savePath)
	Check(err)

	// Container downloads have content length of -1
	written := math.Max(float64(resp.ContentLength), size)

	Println("Wrote", humanize.Bytes(uint64(written)), "to", savePath)
}
