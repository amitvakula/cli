package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/gears"
	"flywheel.io/fw/ops"
)

func (o *opts) bidsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bids",
		Short: "Run the BIDS validator on a directory",
	}

	cmd.AddCommand(o.validateBids())

	return cmd
}

func (o *opts) validateBids() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "validate [dest folder]",
		Short:  "Validate a BIDS folder using the official BIDS validator",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ValidateBids(gears.DockerOrBust(), args[0])
		},
	}

	return cmd
}
