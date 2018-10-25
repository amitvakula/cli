package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/ops"
)

func (o *opts) job() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "job",
		Short:            "Start or manage server jobs",
		PersistentPreRun: o.RequireClient,
	}

	cmd.AddCommand(o.jobRun())
	cmd.AddCommand(o.jobStatus())
	cmd.AddCommand(o.jobWait())
	cmd.AddCommand(o.jobListGears())

	AddDelegateCommand(cmd, "retry", "Retry failed or completed job(s)")

	return cmd
}

func (o *opts) jobRun() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [gear]",
		Short: "Start a job.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ops.JobRun(o.Client, args)
		},
	}

	// This is a silly hack to allow a passthrough -h to the dynamically generated command.
	// Replacements welcome. Dupe with batch run command.
	jobDefaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 3 || args[2] == "-h" || args[2] == "--help" {
			jobDefaultHelpFunc(cmd, args)
		} else {
			ops.JobRun(o.Client, []string{args[2], "-h"})
		}

	})
	cmd.Flags().SetInterspersed(false)
	//

	return cmd
}

func (o *opts) jobStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [job-id]",
		Short: "Check the status of a job.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ops.JobStatus(o.Client, args[0])
		},
	}

	return cmd
}

func (o *opts) jobWait() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait [job-id]",
		Short: "Wait for a job to finish.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ops.JobWait(o.Client, args[0])
		},
	}

	return cmd
}

func (o *opts) jobListGears() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-gears",
		Short: "List available gears.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ListGears(o.Client)
		},
	}

	return cmd
}
