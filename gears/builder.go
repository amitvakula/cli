package gears

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

const (
	GearPath      = "/flywheel/v0"
	GearInputPath = "/flywheel/v0/input"
	ManifestName  = "manifest.json"
	ConfigName    = "config.json"

	GearBuilderSectionKey = "custom.gear-builder"

	DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// GearBuilderInfo holds persistent state, stored in the manifest for convenience and transparency.
// Could possibly end up in the SDK, but needs flexibility to add more fields later.
type GearBuilderInfo struct {
	Category string `json:"category,omitempty"`
	Image    string `json:"image,omitempty"`
}

const GearBuilderSectionRectification = `This is usually due to using your own file, or an older version of the CLI.
Try adding something like this to your manifest:

"custom": {
	"gear-builder": {
		"category": "converter",
		"image": "python:3"
	}
}

Alternatively, create a new folder and try ` + "`fw gear create`" + ` there to create a new manifest.
`

// Extract GB info, failing verbosely if the structure is incorrect.
func ParseGearBuilerInformation(gear *api.Gear) *GearBuilderInfo {
	gbRaw, ok := gear.Custom["gear-builder"]
	if !ok {
		Println()
		Println("There does not appear to be a \"" + GearBuilderSectionKey + "\" section in your manifest.")
		Println(GearBuilderSectionRectification)
		os.Exit(1)
	}

	var gb *GearBuilderInfo

	err := mapstructure.Decode(gbRaw, &gb)
	if err != nil {
		Println()
		Println("Could not decode the \"" + GearBuilderSectionKey + "\" section in your manifest.")
		Println(GearBuilderSectionRectification)
		Println("The raw " + GearBuilderSectionKey + " section was:")
		PrintFormat(gbRaw)
		Println()
		Println("The specific decoding error was:")
		Check(err)
	}

	if gb.Image == "" {
		Println("The \"" + GearBuilderSectionKey + ".image\" key in your manifest is empty, or missing.")
		Println(GearBuilderSectionRectification)
		os.Exit(1)
	}

	if gb.Category == "" {
		Println("The \"" + GearBuilderSectionKey + ".category\" key in your manifest is empty, or missing.")
		Println(GearBuilderSectionRectification)
		os.Exit(1)
	}

	return gb
}

const GearBuilderInvalidJson = `The manifest in your current folder was not valid JSON.
This is usually due to forgetting a comma, or having an extra comma.

If you're lost, try dragging the ` + ManifestName + ` file to this online tool:
https://jsonlint.com

Or, on the command line, ` + "`cat " + ManifestName + " | jq`" + ` can be very helpful.
https://stedolan.github.io/jq

The specific parsing error was:
`

const (
	// On Mac, non-technical users frequently end up with curly quotation marks.

	curlySingleOpen  = "‘"
	curlySingleClose = "’"
	curlyDoubleOpen  = "“"
	curlyDoubleClose = "”"

	allCurlyCodePoints = curlySingleOpen + " " + curlySingleClose + " " + curlyDoubleOpen + " " + curlyDoubleClose
)

const GearBuilderCurlyMarks = `Additionally, some curly quotation marks (` + allCurlyCodePoints + `) were found in the manifest.
This is a common cause for errors on Mac computers.

Try opening the ` + ManifestName + ` file in a code editor (not a word processor).
You should always use straight quotation marks with JSON: "example".`

// Tries to load a manifest from the current folder
func TryToLoadCWDManifest() *api.Gear {
	var gear api.Gear

	// Fail silently if no file is present
	_, statErr := os.Stat(ManifestName)
	if statErr != nil {
		return nil
	}

	// Fail silently if file cannot be read
	rawFile, readErr := ioutil.ReadFile(ManifestName)
	if readErr != nil {
		return nil
	}

	// Parse gear
	parseErr := json.Unmarshal(rawFile, &gear)

	// If invalid JSON, fail loudly
	if parseErr != nil {
		Println()
		Println(GearBuilderInvalidJson)
		Println(parseErr)

		// Check for curly quotation marks

		if strings.ContainsAny(string(rawFile), allCurlyCodePoints) {
			Println()
			Println(GearBuilderCurlyMarks)
		}
		os.Exit(1)
	}

	return &gear
}

const ManifestRequired = "This command requires a manifest. Try `fw gear create` first."

func RequireCWDManifest() *api.Gear {
	gear := TryToLoadCWDManifest()
	if gear == nil {
		Fatal(ManifestRequired)
	}

	return gear
}

func TranslateEnvArrayToEnv(envA []string) map[string]string {
	env := map[string]string{}

	for _, eStr := range envA {
		split := strings.SplitN(eStr, "=", 2)

		if len(split) != 2 {
			Fatal("Docker environment `" + eStr + "` was not split cleanly")
		}

		env[split[0]] = split[1]
	}

	return env
}

func TranslateEnvToEnvArray(env map[string]string) []string {
	var result []string

	for key, value := range env {
		result = append(result, key+"="+value)
	}

	return result
}

var AllowedConfigBashCharacters = regexp.MustCompile("[^' _0123456789ACBEDGFIHKJMLONQPSRUTWVYXZacbedgfihkjmlonqpsrutwvyxz]+")

const FwEnvVarPrefix = "FW_CONFIG_"

// Given a gear and invocation config, create a valid environment set, command, and invocation for launching.
// Mirrored in core Job.generate_request
func CreateInvocationComponents(gear *api.Gear, config map[string]interface{}, files map[string]string, apiKey string, apiKeyInputs []string) (map[string]string, []string, map[string]interface{}) {
	runEnv := map[string]string{}

	// Copy over manifest vars
	for k, v := range gear.Environment {
		runEnv[k] = v
	}

	// Set a default PATH if not provided
	_, pathDefined := gear.Environment["PATH"]
	if !pathDefined {
		runEnv["PATH"] = DefaultPath
	}

	// Massage config into env vars
	for rawKey, rawValue := range config {

		// Whitelist bash env names
		key := AllowedConfigBashCharacters.ReplaceAllString(rawKey, "")
		key = strings.Replace(key, " ", "_", -1)
		key = strings.ToUpper(key)
		key = FwEnvVarPrefix + key

		// Stringify value
		switch value := rawValue.(type) {
		case bool:
			runEnv[key] = strconv.FormatBool(value)
		case int:
			runEnv[key] = strconv.Itoa(value)
		case float64:
			runEnv[key] = strconv.FormatFloat(value, 'f', -1, 64)
		case string:
			runEnv[key] = value
		default:
			Println()
			PrintFormat(config)
			Println()
			Fatal("Config", rawKey, "with value", rawValue, "is of unknown type")
		}
	}

	// Gears always run as a bash command
	command := []string{"bash", "-c"}

	// Use either the provided command, or default to ./run
	if gear.Command != "" {
		templated, err := RenderTemplate(gear.Command, config, files)
		if err != nil {
			Println()
			Println("Could not template out the command.")
			Println("This is probably due to a bad 'command' key in your manifest.")
			Println()
			Println("The command key was:")
			Println(gear.Command)
			Println()
			Println("The templating error was:")
			Println(err)
			Println()
			os.Exit(1)
		}

		command = append(command, templated)
	} else {
		command = append(command, "./run")
	}

	// Create a fake invocation
	inv := BasicInvocation(config)
	for inputName, path := range files {
		inv["inputs"].(map[string]interface{})[inputName] = BasicInvocationInput(inputName, path)
	}
	for _, inputName := range apiKeyInputs {
		inv["inputs"].(map[string]interface{})[inputName] = BasicApiKeyInput(apiKey)
	}

	return runEnv, command, inv
}
