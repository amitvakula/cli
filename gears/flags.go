package gears

import (
	"os"

	"github.com/spf13/cobra"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

// Func sig that adds parsed gear information & invocation to standard Cobra sig
type CommandWithGear func(cmd *cobra.Command, args []string, gear *api.Gear, config map[string]interface{}, files map[string]string, apiKeyInputs []string)

// Dynamically generate a command with flags that correspond to a gear
func GenerateCommand(use, short string, gearLookup func() *api.Gear, innerCmd CommandWithGear) *cobra.Command {

	// Initial variant of the command
	cmd := &cobra.Command{
		Use:   use,
		Short: short,

		// If no manifest is found, this version of the command runs without calling innerCmd
		Run: func(cmd *cobra.Command, args []string) {
			Println("No manifest found. This command needs a manifest to function.")
		},
	}

	// Look up manifest
	gear := gearLookup()

	// No flags to add if manifest is missing
	if gear == nil {
		return cmd
	}

	// Allocate fixed-length arrays of types, which gets around the lang's inabilty to dynamically allocate referenced variables.
	// Config:
	configKeys := make([]string, len(gear.Config))
	configTypes := make([]string, len(gear.Config))
	configOptionals := make([]bool, len(gear.Config))
	configStrings := make([]string, len(gear.Config))
	configInts := make([]int, len(gear.Config))
	configFloats := make([]float64, len(gear.Config))
	configBools := make([]bool, len(gear.Config))
	// Files:
	fileKeys := make([]string, len(gear.Inputs))
	filePaths := make([]string, len(gear.Inputs))
	fileOptionals := make([]bool, len(gear.Inputs))
	// API keys:
	apiKeyInputs := []string{}

	// Parse config values out of manifest and reference them into arrays
	i := 0
	for configName, configMap := range gear.Config {
		// Println("Processing config", configName, configMap)

		// Look up description, if any
		desc, _ := configMap["description"]
		descStr, _ := desc.(string)

		// Look up type, if any (fails in the switch otherwise)
		ty, _ := configMap["type"]
		tyStr, _ := ty.(string)

		// Look up default, if any. Gets casted in switch
		defaultV, hasDefault := configMap["default"]

		// Store name, type, and optional status for lookup after CLI parsing
		configKeys[i] = configName
		configTypes[i] = tyStr
		configOptionals[i] = hasDefault

		switch configTypes[i] {
		case "string":
			defaultC, _ := defaultV.(string)
			cmd.Flags().StringVar(&configStrings[i], configName, defaultC, descStr)
		case "integer":
			defaultC, _ := defaultV.(int)
			cmd.Flags().IntVar(&configInts[i], configName, defaultC, descStr)
		case "number":
			defaultC, _ := defaultV.(float64)
			cmd.Flags().Float64Var(&configFloats[i], configName, defaultC, descStr)
		case "boolean":
			defaultC, _ := defaultV.(bool)
			cmd.Flags().BoolVar(&configBools[i], configName, defaultC, descStr)
		default:
			Fatal("Parsed config value " + configName + " is of unknown type " + tyStr)
		}

		i++
	}

	// Function that reads the fixed-length arrays to produce a single shared map for invocation
	createConfigMap := func() map[string]interface{} {
		config := map[string]interface{}{}

		for i, configKey := range configKeys {
			// Println("Reading config", configKey, "i")

			// Non-optional, non-boolean types must be specified as a flag
			if !configOptionals[i] && configTypes[i] != "boolean" && !cmd.Flags().Changed(configKey) {
				Println()
				Println("--"+configKey, "is a required flag. Try --help for more information.")
				Println()
				Println("If you'd like to make this config value optional, set a `default` value!")
				os.Exit(1)
			}

			switch configTypes[i] {
			case "string":
				config[configKey] = configStrings[i]
			case "integer":
				config[configKey] = configInts[i]
			case "number":
				config[configKey] = configFloats[i]
			case "boolean":
				config[configKey] = configBools[i]
			default:
				Fatal("Read config value " + configKey + " is of unknown type " + configTypes[i])
			}
		}

		return config
	}

	// Parse inputs out of manifest and reference them into arrays
	i = 0
	for inputName, inputMap := range gear.Inputs {
		// Println("Processing input", inputName, inputMap)

		// Look up description, if any
		desc, _ := inputMap["description"]
		descStr, _ := desc.(string)

		// Check base type
		baseRaw, _ := inputMap["base"]
		baseStr, _ := baseRaw.(string)

		// Check optional setting, if any
		optionalRaw, _ := inputMap["optional"]
		optional, _ := optionalRaw.(bool)

		switch baseStr {
		case "file":
			fileKeys[i] = inputName
			fileOptionals[i] = optional
			cmd.Flags().StringVar(&filePaths[i], inputName, "", descStr)
		case "api-key":
			apiKeyInputs = append(apiKeyInputs, inputName)
		default:
			Fatal("Parsed input value " + inputName + " is of unknown type " + baseStr)
		}

		i++
	}

	// Function that reads the fixed-length arrays to produce a single file input map for invocation
	createFilepathMap := func() map[string]string {
		files := map[string]string{}

		for i, inputKey := range fileKeys {
			// Println("Reading input", inputKey, "i", fileOptionals[i])

			if inputKey == "" {
				// Println("Warning: input", i, "is empty")
				continue
			}

			// Non-optional, file inputs must be specified as a flag (and not empty string either)
			if !fileOptionals[i] && (!cmd.Flags().Changed(inputKey) || filePaths[i] == "") {
				Println()
				Println("--"+inputKey, "is a required flag. Try --help for more information.")
				Println()
				Println("If you'd like to make this file optional, add `\"optional\": \"true\"` to the input!")
				Println()
				os.Exit(1)
			}

			// Add file if specified
			if cmd.Flags().Changed(inputKey) {
				if filePaths[i] == "" {
					Fatal("Path to", inputKey, "file cannot be blank.")
				}
				files[inputKey] = filePaths[i]
			}
		}

		return files
	}

	// Command that ties it all together
	cmd.Run = func(cmd *cobra.Command, args []string) {
		config := createConfigMap()
		files := createFilepathMap()

		// Pass parsed information to command logic
		innerCmd(cmd, args, gear, config, files, apiKeyInputs)
	}

	return cmd
}
