package gear

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	. "fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/inconshreveable/log15"

	"flywheel.io/deja/job"
	"flywheel.io/deja/provider"

	. "flywheel.io/fw/util"
)

type ExchangeDoc struct {
	GitCommit  string `json:"git-commit"`
	RootfsHash string `json:"rootfs-hash"`
	RootfsUrl  string `json:"rootfs-url"`
}

type wat map[string]interface{}

type Manifest struct {
	Author      string                            `json:"author"`
	Config      map[string]map[string]interface{} `json:"config"`
	Custom      map[string]interface{}            `json:"custom,omitempty"`
	Description string                            `json:"description"`
	Flywheel    string                            `json:"flywheel"`
	Inputs      map[string]map[string]interface{} `json:"inputs"`
	Label       string                            `json:"label"`
	License     string                            `json:"license"`
	Maintainer  string                            `json:"maintainer"`
	Name        string                            `json:"name"`
	Source      string                            `json:"source"`
	Url         string                            `json:"url"`
	Version     string                            `json:"version"`
}

type GearDoc struct {
	Category string       `json:"category"`
	Exchange *ExchangeDoc `json:"exchange"`
	Gear     *Manifest    `json:"gear"`
}

func hashFile(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha512.New384()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	result := hex.EncodeToString(hash.Sum(nil))
	return result, nil
}

func importAndTestGear(itype, uri, vu string) (*job.FormulaResult, error) {
	i := &job.Input{
		Type:     itype,
		Location: "/",
		URI:      uri,
		VuID:     vu,
	}
	f := &job.Formula{
		Inputs: []*job.Input{
			i,
		},
		Target: job.Target{
			Command: []string{"echo", "Gear imported & tested."},
			Env:     map[string]string{},
			Dir:     "/",
		},
		Outputs: []*job.Output{},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	return provider.Run(f, provider.Logger(log))
}

var gearJson = "gear.json"

func Import(filepath string) {

	Println("Checking gear contents...")
	hash, err := hashFile(filepath)
	Check(err)

	Println("Importing gear...")
	vu := "vu0:x-sha384:" + hash
	result, err := importAndTestGear("file", filepath, vu)
	Check(err)
	if result.Result.ExitCode != 0 {
		Println("Non-zero import exit code", result.Result.ExitCode)
		os.Exit(result.Result.ExitCode)
	}

	Println("Exporting manifest...")
	manifestPath := "/opt/deja/cache/vu0-x-sha384/" + hash + "/flywheel/v0/manifest.json"
	_, err = os.Stat(manifestPath)
	Check(err)

	manifestRaw, err := ioutil.ReadFile(manifestPath)
	Check(err)

	var manifest Manifest
	err = json.Unmarshal(manifestRaw, &manifest)
	Check(err)

	gear := &GearDoc{
		Category: "converter",
		Exchange: &ExchangeDoc{
			RootfsHash: "sha384:" + hash,
			RootfsUrl:  filepath,
		},
		Gear: &manifest,
	}

	if gear.Gear.Custom == nil {
		gear.Gear.Custom = map[string]interface{}{}
	}

	gearEncode, _ := json.MarshalIndent(gear, "", "\t")
	newline := "\n" // json encode has no trailing newline
	gearEncode = append(gearEncode, newline...)

	err = ioutil.WriteFile(gearJson, gearEncode, 0644)
	Check(err)

	Println("Created gear.json in the current folder.")
}

func LoadGear() *GearDoc {
	_, err := os.Stat(gearJson)
	if os.IsNotExist(err) {
		return nil
	}

	gearRaw, err := ioutil.ReadFile(gearJson)
	var gear GearDoc
	err = json.Unmarshal(gearRaw, &gear)
	Check(err)

	if gear.Gear.Custom == nil {
		gear.Gear.Custom = map[string]interface{}{}
	}

	return &gear
}

func SetDownload(url string) {
	gear := LoadGear()
	if gear == nil {
		Println("No gear found! Is there a gear.json in your current folder?")
		os.Exit(1)
	}

	gear.Exchange.RootfsUrl = url

	gearEncode, _ := json.MarshalIndent(gear, "", "\t")
	newline := "\n" // json encode has no trailing newline
	gearEncode = append(gearEncode, newline...)

	err := ioutil.WriteFile(gearJson, gearEncode, 0644)
	Check(err)
}

type GearConfig struct {
	Name        string
	CType       string
	Description string
	Default     interface{}
	Value       interface{}
}

func GetGearConfigs() []*GearConfig {
	gear := LoadGear()
	configs := []*GearConfig{}

	if gear == nil {
		return configs
	}

	for key, x := range gear.Gear.Config {
		config := &GearConfig{
			Name:  key,
			CType: "string",
		}

		cType, ok := x["type"]
		if ok {
			config.CType = cType.(string)
		}

		cDefault, ok := x["default"]
		if ok {
			config.Default = cDefault
		}

		cDescription, ok := x["description"]
		if ok {
			config.Description = cDescription.(string)
		}

		configs = append(configs, config)
	}

	for key, x := range gear.Gear.Inputs {
		config := &GearConfig{
			Name:  key,
			CType: "file",
		}

		cDescription, ok := x["description"]
		if ok {
			config.Description = cDescription.(string)
		}

		configs = append(configs, config)
	}

	return configs
}

func RunGear(configs []*GearConfig) {

	gear := LoadGear()

	// Detect types and parse from string
	for _, c := range configs {

		strVal := c.Value.(string)

		if c.Default == nil && strVal == "" {
			Println(c.Name, "is a required field.")
			os.Exit(1)
		} else if strVal == "" {
			c.Value = c.Default
			continue
		}

		switch c.CType {
		case "file":
		case "string":
			// Nothing to do here
		case "number":
			f, err := strconv.ParseFloat(strVal, 64)
			Check(err)
			c.Value = f
		case "integer":
			f, err := strconv.Atoi(strVal)
			Check(err)
			c.Value = f
		case "boolean":
			f, err := strconv.ParseBool(strVal)
			Check(err)
			c.Value = f
		default:
			Println("Unknown config type", c.CType)
			os.Exit(1)
		}
	}

	// Construct an engine-formated config.json file
	config := map[string]interface{}{}
	for _, c := range configs {
		if c.CType != "file" {
			config[c.Name] = c.Value
		}
	}
	configOut := map[string]interface{}{
		"config": config,
	}
	configRaw, _ := json.MarshalIndent(configOut, "", "\t")

	// Base formula
	f := &job.Formula{
		Inputs: []*job.Input{
			{
				Type:     "file",
				Location: "/",
				URI:      gear.Exchange.RootfsUrl,
				VuID:     "vu0:x-" + gear.Exchange.RootfsHash,
			},
		},
		Target: job.Target{
			Command: []string{"bash", "-c", "rm -rf output 2>/dev/null; mkdir -p output; ./run; echo \"Exit was $?\""},
			Dir:     "/flywheel/v0",
		},
		Outputs: []*job.Output{},
	}

	// Add files
	for _, c := range configs {
		if c.CType == "file" {
			_, err := os.Stat(c.Value.(string))
			Check(err)

			f.Inputs = append(f.Inputs, &job.Input{
				Type:     "cp",
				URI:      c.Value.(string),
				Location: "/flywheel/v0/input/" + c.Name,
			})
		}
	}

	// Bind the output directory
	err := os.MkdirAll("output", 0755)
	if err != nil {
		Println(err)
		os.Exit(1)
	}
	f.Inputs = append(f.Inputs, &job.Input{
		Type:     "bind",
		URI:      "output",
		Location: "/flywheel/v0/output",
	})

	// Config.json
	dir, err := ioutil.TempDir("", "fw")
	Check(err)
	err = ioutil.WriteFile(dir+"/config.json", configRaw, 0644)
	Check(err)
	f.Inputs = append(f.Inputs, &job.Input{
		Type:     "cp",
		URI:      dir + "/config.json",
		Location: "/flywheel/v0",
	})

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result, err := provider.Run(f, provider.Logger(log))

	Check(err)
	os.Exit(result.Result.ExitCode)
}
