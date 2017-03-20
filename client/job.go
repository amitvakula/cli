package client

import (
	"encoding/json"
	. "fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	oapi "flywheel.io/fw/api"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

var greenBold = color.New(color.FgGreen, color.Bold).SprintFunc()

// Gears satisfies sort.Interface for sorting by gear name.
type Gears []*api.GearDoc

func (g Gears) Len() int {
	return len(g)
}
func (g Gears) Less(i, j int) bool {
	return g[i].Gear.Name < g[j].Gear.Name
}
func (g Gears) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func ListGears() {
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)

	gears, _, err := c.GetAllGears()
	Check(err)

	// Change the type so we can sort
	gearsCast := Gears(gears)
	sort.Sort(gearsCast)

	// Format the table, printing to a platform- & pipe-friendly color writer
	w := tabwriter.NewWriter(color.Output, 0, 2, 1, ' ', 0)

	for _, x := range gearsCast {
		Fprintf(w, "%s\t%s\n", greenBold(x.Gear.Name), x.Gear.Label)
	}

	w.Flush()
}

func JobStatus(id string) {
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)

	job, _, err := c.GetJob(id)
	Check(err)

	Println("Job", id, "is", job.State+".")
}

func JobWait(id string) {
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)

	first := true
	interval := 3 * time.Second
	state := api.JobState("")

	for state != api.Cancelled && state != api.Failed && state != api.Complete {

		if first {
			first = false
		} else {
			time.Sleep(interval)
		}

		job, _, err := c.GetJob(id)
		if err != nil {
			Println(err)
			Println("Will continue to retry. Press Control-C to exit.")
		}
		if job.State != state {
			state = job.State
			Println("Job is", state)
		}
		state = job.State
	}

	if state == api.Complete {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func JobRun(args []string) {
	// Client
	var c *api.Client
	var oc *oapi.Client
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)
	oc = MakeClient()
	_ = oc

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
	jobRunActual := &cobra.Command{
		Use:   "run",
		Short: "Start a job.",
		Run: func(cmd *cobra.Command, args []string) {
			// targets := []*api.ContainerReference{}

			// Merge value slice with config slice for convenience
			configsCast := configs.([]*GearConfig)
			for x := range configsCast {
				configsCast[x].Value = values.([]string)[x]
			}

			// construct map from flags
			config := genConfigStruct(configsCast)
			configOut, _ := json.MarshalIndent(config, "", "\t")
			Println("Job configuration:")
			Println(string(configOut))

			// resolve inouts
			inputs := genInputs(configsCast)
			sendInputs := map[string]interface{}{}

			for inputName, path := range inputs {
				result, _, err, aerr := oc.ResolvePathString(path)
				Check(coalesce(err, aerr))
				last := result.Path[len(result.Path)-1]

				file, ok := last.(*oapi.File)
				if !ok {
					Println("Input", inputName, "must resolve to a file.")
					os.Exit(1)
				}
				parent := result.Path[len(result.Path)-2].(oapi.Container)

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

			jobId, _, err := c.AddJob(job)
			Check(err)

			Println("Job", jobId, "has been queued.")
		},
	}

	// cmd flag add
	cs := genGearConfigs(gear)
	configs = cs
	values = make([]string, len(cs))

	for index, config := range cs {
		jobRunActual.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	jobRunActual.SetArgs(args)
	jobRunActual.Execute()
}
