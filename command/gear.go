package command

import (
	"strconv"

	"github.com/spf13/cobra"

	"flywheel.io/fw/gears"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func (o *opts) gear() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gear",
		Short: "Gear commands (requires Docker)",
	}

	cmd.AddCommand(o.gearCreate())
	cmd.AddCommand(o.gearRun())
	cmd.AddCommand(o.gearModify())
	cmd.AddCommand(o.gearUpload())
	cmd.AddCommand(o.gearDocs())

	return cmd
}

func (o *opts) gearCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "create",
		Short:  "Create a new gear in the current folder",
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			gears.GearCreate(o.Client, gears.DockerOrBust())
		},
	}
	return cmd
}

func (o *opts) gearRun() *cobra.Command {

	// Use parsed manifest invocation values
	innerCmd := func(cmd *cobra.Command, args []string, gear *api.Gear, config map[string]interface{}, files map[string]string, apiKeyInputs []string) {
		if gear == nil {
			Fatal(gears.ManifestRequired)
		}

		gears.GearRun(o.Client, gears.DockerOrBust(), o.Credentials.Key, gear, config, files, apiKeyInputs)
	}

	// Dynamically generated command to load the invocation
	cmd := gears.GenerateCommand(
		"local",
		"Run your gear from the current folder",
		gears.TryToLoadCWDManifest,
		innerCmd,
	)

	// Require API key in the event of an API key input type
	cmd.PreRun = o.RequireClient
	return cmd
}

func (o *opts) gearModify() *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "modify",
		Short: "Modify your gear from an interactive shell",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			gears.GearModify(gears.DockerOrBust(), quiet)
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress help message at start")

	return cmd
}

func (o *opts) gearUpload() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upload",
		Short:  "Upload your local gear to Flywheel",
		Args:   cobra.ExactArgs(0),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {

			host, port, _, _ := api.ParseApiKey(o.Credentials.Key)

			domain := host
			if port != 443 {
				domain += ":" + strconv.Itoa(port)
			}

			gear := gears.RequireCWDManifest()

			gears.GearUpload(o.Client, gears.DockerOrBust(), o.Credentials.Key, domain, gear)
		},
	}

	return cmd
}

func (o *opts) gearDocs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Show some links to gear building documentation",
		Args:  cobra.ExactArgs(0),
		Long: `
Feeling lost? Try starting with ` + "`fw gear create`" + ` first.

Or, try our story-based documentation online:
https://docs.flywheel.io/display/EM/Building+Gears

For a technical review of the manifest, check out:
https://github.com/flywheel-io/gears/tree/master/spec
`,
	}

	return cmd
}
