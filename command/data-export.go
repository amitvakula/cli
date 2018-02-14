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
	var sessions arrayFlags
	var subjects arrayFlags
	var dataTypes arrayFlags

	cmd := &cobra.Command{
		Use:    "bids [dest folder]",
		Short:  "Export a BIDS project to the destination folder (requires Docker)",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ExportBids(gears.DockerOrBust(), o.Credentials.Key, args[0], projectLabel, sourceData, []string(sessions), []string(subjects), []string(dataTypes))
		},
	}
	cmd.Flags().StringVarP(&projectLabel, "project", "p", "", "The label of the project to export")
	cmd.Flags().Var(&sessions, "session", "Limit export to the given session names")
	cmd.Flags().BoolVar(&sourceData, "source-data", false, "Include sourcedata in BIDS export")
	cmd.Flags().Var(&subjects, "subject", "Limit export to the given subjects")
	cmd.Flags().Var(&dataTypes, "data-type", "Limit export to the given data-types (e.g. func)")

	return cmd
}
