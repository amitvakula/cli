package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/ops"
)

func (o *opts) batch() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "batch",
		Short:            "Start or manage server batch jobs",
		PersistentPreRun: o.RequireClient,
	}

	cmd.AddCommand(o.batchRun())
	cmd.AddCommand(o.batchCancel())

	return cmd
}

func (o *opts) batchRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [gear] [folders...]",
		Short: "Start a batch job.",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			ops.BatchRun(o.Client, args)
		},
	}

	// This is a silly hack to allow a passthrough -h to the dynamically generated command.
	// Replacements welcome. Dupe with job run command.
	batchDefaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 3 || args[2] == "-h" || args[2] == "--help" {
			batchDefaultHelpFunc(cmd, args)
		} else {
			ops.BatchRun(o.Client, []string{args[2], "-h"})
		}

	})
	cmd.Flags().SetInterspersed(false)
	//

	return cmd
}

func (o *opts) batchCancel() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [batch-id]",
		Short: "Cancel a batch job.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ops.BatchCancel(o.Client, args[0])
		},
	}

	return cmd
}
