package ops

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func JobRun(client *api.Client, args []string) {
	// Slice
	gearName := args[0]
	args = args[1:]

	// Find gear name from remote
	gears, _, err := client.GetAllGears()
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
	jobRunActual := &cobra.Command{
		Use:   "run",
		Short: "Start a job.",
		Run: func(cmd *cobra.Command, args []string) {
			// targets := []*api.ContainerReference{}

			// Merge value slice with config slice for convenience
			configsCast := configs.([]*legacy.GearConfig)
			for x := range configsCast {
				configsCast[x].Value = values.([]string)[x]
			}

			// construct map from flags
			config := legacy.GenConfigStruct(configsCast)
			configOut, _ := json.MarshalIndent(config, "", "\t")
			Println("Job configuration:")
			Println(string(configOut))

			// resolve inouts
			inputs := legacy.GenInputs(configsCast)
			sendInputs := map[string]interface{}{}

			for inputName, path := range inputs {
				result, _, err, aerr := legacy.ResolvePathString(client, path)
				Check(api.Coalesce(err, aerr))
				last := result.Path[len(result.Path)-1]

				file, ok := last.(*legacy.File)
				if !ok {
					Println("Input", inputName, "must resolve to a file.")
					os.Exit(1)
				}
				parent := result.Path[len(result.Path)-2].(legacy.Container)

				sendInputs[inputName] = &api.FileReference{
					Id:   parent.GetId(),
					Type: parent.GetType(),
					Name: file.Name,
				}
			}

			job := &api.Job{
				GearId: gearDoc.Id,
				Config: config,
				Inputs: sendInputs,
				Tags:   []string{"cli"},
			}

			jobId, _, err := client.AddJob(job)
			Check(err)

			Println("Job", jobId, "has been queued.")
		},
	}

	// cmd flag add
	cs := legacy.GenGearConfigs(gear)
	configs = cs
	values = make([]string, len(cs))

	for index, config := range cs {
		jobRunActual.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	jobRunActual.SetArgs(args)
	jobRunActual.Execute()
}
