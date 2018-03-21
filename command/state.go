package command

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

// Creds represents the state that is stored in the user's homedir, at ConfigPath.
type Creds struct {
	Key      string `json:"key"`
	Insecure bool   `json:"insecure"`
}

// ConfigPath defines where the Creds struct is persisted on disk.
const ConfigPath = "~/.config/flywheel/user.json"

// LoadCreds attempts to load an existing config file from ConfigPath.
func LoadCreds() (*Creds, error) {
	path, err := homedir.Expand(ConfigPath)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Creds
	err = json.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// MakeClient attempts to create an SDK client using LoadCreds.
func MakeClient() (*api.Client, error) {
	creds, err := LoadCreds()
	if err != nil {
		return nil, err
	}

	return MakeClientWithCreds(creds.Key, creds.Insecure)
}

func MakeClientWithCreds(key string, insecure bool) (*api.Client, error) {
	_, _, _, err := api.ParseApiKey(key)
	if err != nil {
		return nil, errors.New("Invalid API key format. Please re-generate in the Flywheel user interface.")
	}

	opts := []api.ApiKeyClientOption{api.UserAgent(UserAgent)}
	if insecure {
		opts = append(opts, api.InsecureNoSSLVerification)
	}

	return api.NewApiKeyClient(key, opts...), nil
}

// Save persists the Creds to ConfigPath. Exits on error.
func (c *Creds) Save() {
	raw, err := json.MarshalIndent(c, "", "\t")
	Check(err)

	// Files should end in newlines
	raw = append(raw, []byte("\n")...)

	path, err := homedir.Expand(ConfigPath)
	Check(err)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	Check(err)

	err = ioutil.WriteFile(path, raw, 0644)
	Check(err)
}

func DeleteCreds() error {
	path, err := homedir.Expand(ConfigPath)
	if err != nil {
		return err
	}

	err = os.Remove(path)

	if os.IsNotExist(err) {
		return nil
	} else {
		return err
	}
}
