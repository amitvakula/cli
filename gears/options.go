package gears

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"

	. "flywheel.io/fw/util"
)

// OptionsPath defines where the Options struct is persisted on disk.
const OptionsPath = "~/.config/flywheel/options.json"

type Options struct {
	CustomGearImages []string `json:"custom-gear-images"`
}

func LoadOptions() *Options {
	path, err := homedir.Expand(OptionsPath)
	if err != nil {
		return &Options{}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return &Options{}
	}

	var o Options
	err = json.Unmarshal(b, &o)
	if err != nil {
		return &Options{}
	}

	return &o
}

func (o *Options) Save() {
	raw, err := json.MarshalIndent(o, "", "\t")
	Check(err)

	// Files should end in newlines
	raw = append(raw, []byte("\n")...)

	path, err := homedir.Expand(OptionsPath)
	Check(err)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	Check(err)

	err = ioutil.WriteFile(path, raw, 0644)
	Check(err)
}
