package client

import (
	"encoding/json"
	. "fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	prompt "github.com/segmentio/go-prompt"
	"github.com/spf13/cobra"

	oapi "flywheel.io/fw/api"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

type GearConfig struct {
	Name        string
	CType       string
	Description string
	Default     interface{}
	Value       interface{}
}

func genGearConfigs(gear *api.Gear) []*GearConfig {
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

func genBatchGearConfigs(gear *api.Gear) []*GearConfig {
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

func genConfigStruct(configs []*GearConfig) map[string]interface{} {
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

	return config
}

func genInputs(configs []*GearConfig) map[string]string {

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

func BatchCancel(id string) {
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)

	count, _, err := c.CancelBatch(id)
	Check(err)

	Println("Cancelled", count, "jobs.")
}

func BatchRun(args []string) {
	// Client
	var c *api.Client
	var oc *oapi.Client
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)
	oc = MakeClient()

	// Slice
	gearName := args[0]
	args = args[1:]

	// Find gear name from remote
	gears, _, err := c.GetAllGears()
	Check(err)
	var gearDoc *api.GearDoc
	for _, x := range gears {
		if x.Gear.Name == gearName {
			gearDoc = x
		}
	}
	if gearDoc == nil {
		Println("No gear found with name", gearName)
		os.Exit(1)
	}
	gear := gearDoc.Gear

	// New command template in the style of commands_linux.go
	dummyCmd := &cobra.Command{}
	defaultTemplate := dummyCmd.UsageTemplate()
	gearCmdTemplate := strings.Replace(defaultTemplate, ".LocalFlags.FlagUsages | trimRightSpace", ".LocalFlags.FlagUsages | trimRightSpace | trimStringLiterals", 1)
	trimStringLiterals := func(s string) string {
		removeStringLiteral := regexp.MustCompile(`^( +\-\-.*?) string(.*)$`)

		parts := strings.Split(s, "\n")
		for x, part := range parts {
			parts[x] = removeStringLiteral.ReplaceAllString(part, "${1}${2}")
		}

		return strings.Join(parts, "\n")
	}
	cobra.AddTemplateFunc("trimStringLiterals", trimStringLiterals)
	_ = gearCmdTemplate

	// cmd storage
	var configs interface{}
	var values interface{}

	// cmd base
	batchRunActual := &cobra.Command{
		Use:   "run",
		Short: "Start a batch job.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				Println("Specify at least one folder to run a batch gear on.")
				os.Exit(1)
			}

			targets := []*api.ContainerReference{}

			for _, arg := range args {
				result, _, err, aerr := oc.ResolvePathString(arg)
				Check(coalesce(err, aerr))
				path := result.Path
				last := path[len(path)-1]

				wat, ok := last.(oapi.Container)
				if !ok {
					Println("Each path must resolve to a container, not a file.")
					os.Exit(1)
				}

				aType := wat.GetType()

				if aType != "session" {
					Println("Batch run is not currently supported at the", aType, "level. Run at the session level instead.")
					os.Exit(1)
				}

				targets = append(targets, &api.ContainerReference{
					Id:   wat.GetId(),
					Type: wat.GetType(),
				})
			}

			// Merge value slice with config slice for convenience
			configsCast := configs.([]*GearConfig)
			for x := range configsCast {
				configsCast[x].Value = values.([]string)[x]
			}

			// construct map from flags
			config := genConfigStruct(configsCast)
			configOut, _ := json.MarshalIndent(config, "", "\t")
			Println("Batch gear configuration:")
			Println(string(configOut))

			// Println("Running on sessions:")
			// for _, x := range targets {
			// 	Println(x.Id)
			// }

			proposal, _, err := c.ProposeBatch(gearDoc.Id, config, []string{"batch", "cli"}, targets)
			Check(err)

			if proposal.Id == "" {
				Println("Batch proposal did not match any valid targets - nothing to do.")
				return
			}

			Println("Batch would run against", len(proposal.Matched), "targets.")
			if len(proposal.Ambiguous) > 0 {
				Println(len(proposal.Ambiguous), "targets could be run multiple ways and are therefore skipped.")
			}
			if len(proposal.MissingPermissions) > 0 {
				Println(len(proposal.MissingPermissions), "targets are not writable by you and are skipped.")
			}
			if len(proposal.NotMatched) > 0 {
				Println(len(proposal.NotMatched), "targets did not match the gear and are skipped.")
			}

			proceed := prompt.Confirm("Continue? (yes/no)")
			Println()
			if !proceed {
				Println("Canceled.")
				return
			}

			jobs, _, err := c.StartBatch(proposal.Id)
			Check(err)
			Println("Batch", proposal.Id, "has been queued with", len(jobs), "jobs.")
		},
	}

	// cmd flag add
	cs := genBatchGearConfigs(gear)
	configs = cs
	values = make([]string, len(cs))

	for index, config := range cs {
		batchRunActual.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	batchRunActual.SetArgs(args)
	batchRunActual.Execute()
}
