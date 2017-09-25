package legacy

import (
	. "fmt"
	"os"
	"strings"
	"strconv"

	"flywheel.io/sdk/api"

	. "flywheel.io/fw/util"
)

type GearConfig struct {
	Name        string
	CType       string
	Description string
	Default     interface{}
	Value       interface{}
}

func GenGearConfigs(gear *api.Gear) []*GearConfig {
	configs := []*GearConfig{}

	if gear == nil {
		return configs
	}

	for key, x := range gear.Config {
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

	for key, x := range gear.Inputs {

		if x["base"].(string) != "file" {
			continue
		}

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

func GenBatchGearConfigs(gear *api.Gear) []*GearConfig {
	configs := []*GearConfig{}

	if gear == nil {
		return configs
	}

	for key, x := range gear.Config {
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

	return configs
}

func GenConfigStruct(configs []*GearConfig) map[string]interface{} {
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
		case "array":
			c.Value = strings.Split(strVal, ",")
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

	return config
}

func GenInputs(configs []*GearConfig) map[string]string {

	result := map[string]string{}

	for _, c := range configs {
		switch c.CType {
		case "file":
			result[c.Name] = c.Value.(string)
		default:
		}
	}

	return result
}
