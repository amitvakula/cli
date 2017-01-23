package client

import (
	. "fmt"
	"math"
	"os"
	"strings"
	"sync"

	humanize "github.com/dustin/go-humanize"
	prompt "github.com/segmentio/go-prompt"

	"flywheel.io/fw/api"
	. "flywheel.io/fw/util"
)

func Login(host, key string, insecure bool) {
	client := api.NewApiKeyClient(host, key, insecure)

	user, _, err := client.GetCurrentUser()
	Check(err)

	c := &Creds{
		Host:     host,
		Key:      key,
		Insecure: insecure,
	}
	c.Save()

	Println("Logged in as", user.Firstname, user.Lastname, "<"+user.Email+">")
}

func Ls(upath string, showDbIds bool) {
	client := MakeClient()
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	var wg sync.WaitGroup
	var user *api.User
	var result *api.ResolveResult

	go func() {
		var err error
		user, _, err = client.GetCurrentUser()
		Check(err)
		wg.Done()
	}()

	go func() {
		var err error
		result, _, err = client.ResolvePath(parts)
		Check(err)
		wg.Done()
	}()

	wg.Add(2)
	wg.Wait()
	PrintResolve(result, user.Id, showDbIds)
}

func Download(upath, savePath string, force bool) {
	client := MakeClient()
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err := client.ResolvePath(parts)
	Check(err)
	path := result.Path

	last := path[len(path)-1]

	// This logic is dangled between download-file and download-container.
	// It should get much nicer after some SDK fiddling.

	var parent interface{}
	var name string
	size := float64(0)

	file, ok := last.(*api.File)
	if ok {
		parent = path[len(path)-2]
		name = file.Name

	} else {
		wat := last.(api.Container)

		if wat.GetType() == "group" {
			Println("Group downloads are currently not supported. Instead, you can download each project.")
			os.Exit(1)
		}

		ticketRequest := &api.ContainerTicketRequest{
			Nodes: []*api.ContainerTicketRequestElem{
				{
					Level: wat.GetType(),
					Id:    wat.GetId(),
				},
			},
			Optional: true,
		}

		ticket, _, err := client.GetDownloadTicket(ticketRequest)
		Check(err)

		// Should make this second condition cleaner...
		if !force && savePath != "--" {
			Println()
			Println("This download will be about", humanize.Bytes(ticket.Size), "comprising", ticket.FileCount, "files.")

			proceed := prompt.Confirm("Continue? (yes/no)")
			Println()
			if !proceed {
				Println("Cancelled.")
				return
			}
		}

		parent = ticket
		name = "download.tar"
		size = float64(ticket.Size)
	}

	if savePath == "--" {
		_, err := client.Download(name, parent, os.Stdout)
		Check(err)
		return
	}

	if savePath == "" {
		savePath = name
	}

	resp, err := client.DownloadToFile(name, parent, savePath)
	Check(err)

	// Container downloads have content length of -1
	written := math.Max(float64(resp.ContentLength), size)

	Println("Wrote", humanize.Bytes(uint64(written)), "to", savePath)
}

func Upload(upath, sendPath string) {
	client := MakeClient()
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err := client.ResolvePath(parts)
	Check(err)
	path := result.Path

	_, err = client.UploadFromFile(sendPath, path[len(path)-1], nil, sendPath)
	Check(err)

	Println("Uploaded file to", upath+".")
}
