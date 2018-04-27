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
		Use:   "folder [folder]",
		Short: "Import a structured folder",
		Long: `Import a folder with the following structure:

root-folder
└── group-id
    └── project-label
        └── subject-label
            └── session-label
                └── acquisition-label
                    ├── dicom
                    │   ├── 1.dcm
                    │   └── 2.dcm
                    ├── data.foo
                    └── scan.nii.gz

Files can be placed at the project level and below. Files to be uploaded via a packfile upload must be placed in a folder under the acquisition folder, the folder name will be used as the file type.`,
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
		Use:    "dicom [folder] [group-id] [project-label]",
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

	var projectLabel string

	cmd := &cobra.Command{
		Use:    "bids [folder] [group]",
		Short:  "Import a BIDS project to the destination project (requires Docker)",
		Args:   cobra.ExactArgs(2),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ImportBids(gears.DockerOrBust(), o.Credentials.Key, args[0], args[1], projectLabel)
		},
	}

	cmd.Flags().StringVarP(&projectLabel, "project", "p", "", "Label of project to import into")

	return cmd
}
