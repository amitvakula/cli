package client

import (
	. "fmt"
	// "os"
	"strings"

	// humanize "github.com/dustin/go-humanize"

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
	parts := strings.Split(upath, "/")

	user, _, err := client.GetCurrentUser()
	Check(err)

	_, _ = parts, user
	// path := resolve(client, parts)
	// base := path[len(path)-1]
	// var parent interface{}
	// if len(path) > 1 {
	// 	parent = path[len(path)-2]
	// }

	// print(base, parent, user.Id, showDbIds)
}

func Download(upath, savePath string) {
	client := MakeClient()
	parts := strings.Split(upath, "/")

	_, _ = client, parts
	// path := resolve(client, parts)
	// var err error

	// file, ok := path[len(path)-1].(*File)
	// if !ok {
	// 	Println("Path does not refer to a file")
	// 	os.Exit(1)
	// }

	// if savePath == "--" {
	// 	_, err := client.Download(file.Name, path[len(path)-2], os.Stdout)
	// 	Check(err)
	// 	return
	// }

	// if savePath == "" {
	// 	savePath = file.Name
	// }

	// resp, err := client.DownloadToFile(file.Name, path[len(path)-2], savePath)
	// Check(err)

	// Println("Wrote", humanize.Bytes(uint64(resp.ContentLength)), "to", savePath)
}

func Upload(upath, sendPath string) {
	client := MakeClient()
	parts := strings.Split(upath, "/")

	_, _ = client, parts
	// path := resolve(client, parts)

	// resp, err := client.UploadFromFile(sendPath, path[len(path)-1], nil, sendPath)
	// Check(err)

	// Println("Wrote", humanize.Bytes(uint64(resp.ContentLength)), "to", sendPath)
}
