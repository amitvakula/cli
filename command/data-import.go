package command

import (
	"github.com/spf13/cobra"

	"flywheel.io/fw/dicom"
	"flywheel.io/fw/gears"
	"flywheel.io/fw/ops"
)

func (o *opts) importCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data into Flywheel",
	}

	cmd.AddCommand(o.importFolder())
	cmd.AddCommand(o.importDicom())
	cmd.AddCommand(o.importBids())

	return cmd
}

func (o *opts) importFolder() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "folder [folder]",
		Short:  "Import a structured folder",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ScanUpload(o.Client, args[0])
		},
	}

	return cmd
}

func (o *opts) importDicom() *cobra.Command {
	var quiet bool
	var noTree bool
	var local bool

	cmd := &cobra.Command{
		Use:    "dicom [folder] [group] [project]",
		Short:  "Import a folder of dicom files",
		Args:   cobra.ExactArgs(3),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {

			// related_acq is marked as not working, so force-set to false
			// force-set log_level to 2; change out for a verbose flag later.

			dicom.Scan(o.Client, args[0], args[1], args[2], false, quiet, noTree, local)
		},
	}

	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Show less scan and upload progress")
	cmd.Flags().BoolVar(&noTree, "no-tree", false, "Do not show upload summary tree")
	cmd.Flags().BoolVarP(&local, "local", "l", false, "Save derived hierarchy locally")

	return cmd
}

func (o *opts) importBids() *cobra.Command {

	cmd := &cobra.Command{
		Use:    "bids [folder] [group] [project]",
		Short:  "Import a BIDS project to the destination project (requires Docker)",
		Args:   cobra.ExactArgs(3),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ImportBids(gears.DockerOrBust(), o.Credentials.Key, args[0], args[1], args[2])
		},
	}

	return cmd
}
