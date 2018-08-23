package ops

import (
	"encoding/json"
	. "fmt"
	"os"
	"regexp"
	"strings"

	prompt "github.com/segmentio/go-prompt"
	"github.com/spf13/cobra"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func BatchRun(client *api.Client, args []string, optionalInputPolicy string) {

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
				result, _, err, aerr := legacy.ResolvePathString(client, arg)
				Check(api.Coalesce(err, aerr))
				path := result.Path
				last := path[len(path)-1]

				wat, ok := last.(legacy.Container)
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
			configsCast := configs.([]*legacy.GearConfig)
			for x := range configsCast {
				configsCast[x].Value = values.([]string)[x]
			}

			// construct map from flags
			config := legacy.GenConfigStruct(configsCast)
			configOut, _ := json.MarshalIndent(config, "", "\t")
			Println("Batch gear configuration:")
			Println(string(configOut))

			// Println("Running on sessions:")
			// for _, x := range targets {
			// 	Println(x.Id)
			// }

			proposal, _, err := client.ProposeBatch(gearDoc.Id, config, []string{"batch", "cli"}, targets, optionalInputPolicy)
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

			jobs, _, err := client.StartBatch(proposal.Id)
			Check(err)
			Println("Batch", proposal.Id, "has been queued with", len(jobs), "jobs.")
		},
	}

	// cmd flag add
	cs := legacy.GenBatchGearConfigs(gear)
	configs = cs
	values = make([]string, len(cs))

	for index, config := range cs {
		batchRunActual.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	batchRunActual.SetArgs(args)
	batchRunActual.Execute()
}
