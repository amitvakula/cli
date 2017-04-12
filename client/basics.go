package client

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	"flywheel.io/fw/api"
	. "flywheel.io/fw/util"
)

type Creds struct {
	Host     string `json:"host"`
	Key      string `json:"key"`
	Insecure bool   `json:"insecure"`
}

func LoadCreds() *Creds {
	dir, err := homedir.Expand("~/.config/flywheel")
	Check(err)

	b, err := ioutil.ReadFile(filepath.Join(dir, "user.json"))
	Check(err)

	var c Creds
	err = json.Unmarshal(b, &c)
	Check(err)

	return &c
}

func MakeClient() *api.Client {
	c := LoadCreds()
	return api.NewApiKeyClient(c.Host, c.Key, c.Insecure)
}

func MakeClientSafe() (*api.Client, error) {
	dir, err := homedir.Expand("~/.config/flywheel")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(dir, "user.json"))
	if err != nil {
		return nil, err
	}

	var c Creds
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return api.NewApiKeyClient(c.Host, c.Key, c.Insecure), nil
}

func (c *Creds) Save() {
	raw, err := json.MarshalIndent(c, "", "\t")
	Check(err)

	dir, err := homedir.Expand("~/.config/flywheel")
	Check(err)

	err = os.MkdirAll(dir, 0755)
	Check(err)

	err = ioutil.WriteFile(filepath.Join(dir, "user.json"), raw, 0644)
	Check(err)
}
