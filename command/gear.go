package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/gears"
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

	return cmd
}

func (o *opts) gearCreate() *cobra.Command {
	var clearCustomList bool = false
	cmd := &cobra.Command{
		Use:    "create",
		Short:  "Create a new gear in the current folder",
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			gears.GearCreate(o.Client, gears.DockerOrBust(), clearCustomList)
		},
	}
	cmd.Flags().BoolVar(&clearCustomList, "clear-custom-containers", false, "Clear the custom container list")

	return cmd
}

func (o *opts) gearRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run your gear from the current folder",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			gears.GearRun(o.Client, gears.DockerOrBust(), args)
		},
	}

	// This is a silly hack to allow a passthrough -h to the dynamically generated command.
	// Replacements welcome. Dupe with batch run and etc commands.
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 2 || args[1] == "-h" || args[1] == "--help" {
			defaultHelpFunc(cmd, args)
		} else {
			gears.GearRun(o.Client, gears.DockerOrBust(), []string{args[1], "-h"})
		}

	})
	cmd.Flags().SetInterspersed(false)
	//

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
	var category string
	var file string
	var project string

	cmd := &cobra.Command{
		Use:    "upload",
		Short:  "Upload your local gear to Flywheel",
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			gears.GearUpload(o.Client, gears.DockerOrBust(), category, file, project)
		},
	}
	cmd.Flags().StringVarP(&category, "category", "c", "converter", "Gear category: converter or analysis")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Write the result to a gzipped tarball instead of flywheel (-- for stdout)")

	// This feature was canceled:
	// https://github.com/flywheel-io/core/pull/1213
	// cmd.Flags().StringVarP(&project, "project", "p", "", "Limit visibility of the gear to a specific project")

	return cmd
}
