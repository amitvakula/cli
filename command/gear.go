package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/ops"
)

func (o *opts) gear() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gear",
		Short: "Gear commands (requires Docker)",
	}

	cmd.AddCommand(o.gearCreate())
	cmd.AddCommand(o.gearRun())
	cmd.AddCommand(o.gearUpload())

	return cmd
}

func (o *opts) gearCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "create",
		Short:  "Create a new gear in the current folder",
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.GearCreate(o.Client, ops.DockerOrBust())
		},
	}

	return cmd
}

func (o *opts) gearRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run your gear from the current folder",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ops.GearRun(o.Client, ops.DockerOrBust(), args)
		},
	}

	// This is a silly hack to allow a passthrough -h to the dynamically generated command.
	// Replacements welcome. Dupe with batch run and etc commands.
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 2 || args[1] == "-h" || args[1] == "--help" {
			defaultHelpFunc(cmd, args)
		} else {
			ops.GearRun(o.Client, ops.DockerOrBust(), []string{args[1], "-h"})
		}

	})
	cmd.Flags().SetInterspersed(false)
	//

	return cmd
}

func (o *opts) gearUpload() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upload",
		Short:  "Upload your local gear to Flywheel",
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.GearUpload(o.Client, ops.DockerOrBust())
		},
	}

	return cmd
}
