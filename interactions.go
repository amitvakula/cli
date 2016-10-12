package main

import (
	"encoding/json"
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	humanize "github.com/dustin/go-humanize"
	homedir "github.com/mitchellh/go-homedir"
)

func check(err error) {
	if err != nil {
		Println(err)
		os.Exit(1)
	}
}

type creds struct {
	Host     string `json:"host"`
	Key      string `json:"key"`
	Insecure bool   `json:"insecure"`
}

func loadCreds() *creds {
	dir, err := homedir.Expand("~/.config/flywheel")
	check(err)

	b, err := ioutil.ReadFile(filepath.Join(dir, "user.json"))
	check(err)

	var c creds
	err = json.Unmarshal(b, &c)
	check(err)

	return &c
}

func makeClient() *Client {
	c := loadCreds()
	return NewApiKeyClient(c.Host, c.Key, c.Insecure)
}

func (c *creds) save() {
	raw, err := json.MarshalIndent(c, "", "\t")
	check(err)

	dir, err := homedir.Expand("~/.config/flywheel")
	check(err)

	err = os.MkdirAll(dir, 0755)
	check(err)

	err = ioutil.WriteFile(filepath.Join(dir, "user.json"), raw, 0644)
	check(err)
}

func login(host, key string, insecure bool) {
	client := NewApiKeyClient(host, key, insecure)

	user, _, err := client.GetCurrentUser()
	check(err)

	c := &creds{
		Host:     host,
		Key:      key,
		Insecure: insecure,
	}
	c.save()

	Println("Logged in as", user.Firstname, user.Lastname, "<"+user.Email+">")
}

func ls(upath string) {
	client := makeClient()
	parts := strings.Split(upath, "/")

	user, _, err := client.GetCurrentUser()
	check(err)

	path := resolve(client, parts)
	base := path[len(path)-1]
	var parent interface{}
	if len(path) > 1 {
		parent = path[len(path)-2]
	}

	print(base, parent, user.Id)
}

func download(upath, savePath string) {
	client := makeClient()
	parts := strings.Split(upath, "/")

	path := resolve(client, parts)
	var err error

	file, ok := path[len(path)-1].(*File)
	if !ok {
		Println("Path does not refer to a file")
		os.Exit(1)
	}

	if savePath == "--" {
		_, err := client.Download(file.Name, path[len(path)-2], os.Stdout)
		check(err)
		return
	}

	if savePath == "" {
		savePath = file.Name
	}

	resp, err := client.DownloadToFile(file.Name, path[len(path)-2], savePath)
	check(err)

	Println("Wrote", humanize.Bytes(uint64(resp.ContentLength)), "to", savePath)
}
