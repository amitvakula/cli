package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/gears"
	"flywheel.io/fw/ops"
)

func (o *opts) exportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export data from Flywheel",
	}

	cmd.AddCommand(o.exportBids())

	return cmd
}

func (o *opts) exportBids() *cobra.Command {
	var projectLabel string
	var sourceData bool = false

	cmd := &cobra.Command{
		Use:    "bids [dest folder]",
		Short:  "Export a BIDS project to the destination folder",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ExportBids(gears.DockerOrBust(), o.Credentials.Key, args[0], projectLabel, sourceData)
		},
	}
	cmd.Flags().StringVarP(&projectLabel, "project", "p", "", "The label of the project to export")
	cmd.Flags().BoolVar(&sourceData, "source-data", false, "Include sourcedata in BIDS export")

	return cmd
}
