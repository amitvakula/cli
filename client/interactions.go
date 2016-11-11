package client

import (
	. "fmt"
	"os"
	"strings"
	"sync"

	humanize "github.com/dustin/go-humanize"

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

func Download(upath, savePath string) {
	client := MakeClient()
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err := client.ResolvePath(parts)
	Check(err)
	path := result.Path

	file, ok := path[len(path)-1].(*api.File)
	if !ok {
		Println("Path does not refer to a file")
		os.Exit(1)
	}

	if savePath == "--" {
		_, err := client.Download(file.Name, path[len(path)-2], os.Stdout)
		Check(err)
		return
	}

	if savePath == "" {
		savePath = file.Name
	}

	resp, err := client.DownloadToFile(file.Name, path[len(path)-2], savePath)
	Check(err)

	Println("Wrote", humanize.Bytes(uint64(resp.ContentLength)), "to", savePath)
}

func Upload(upath, sendPath string) {
	client := MakeClient()
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err := client.ResolvePath(parts)
	Check(err)
	path := result.Path

	resp, err := client.UploadFromFile(sendPath, path[len(path)-1], nil, sendPath)
	Check(err)

	Println("Wrote", humanize.Bytes(uint64(resp.ContentLength)), "to", sendPath)
}
