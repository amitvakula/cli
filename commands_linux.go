package main

import (
	. "fmt"

	"github.com/spf13/cobra"

	// Register all implementations
	_ "flywheel.io/deja/flak/features"
)

func init() {

	gearCmd := &cobra.Command{
		Use:   "gear",
		Short: "Gear commands",
	}
	RootCmd.AddCommand(gearCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new flywheel gear",
		Run: func(cmd *cobra.Command, args []string) {
			Setup()
			Println("Created a new gear in the current folder.")
		},
	}
	gearCmd.AddCommand(createCmd)

	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Use a new flywheel gear",
		Run: func(cmd *cobra.Command, args []string) {
			project := Setup()
			project.Use(args)
		},
	}
	gearCmd.AddCommand(useCmd)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run your gear locally",
		Run: func(cmd *cobra.Command, args []string) {
			project := Setup()
			project.Run(args)
		},
	}
	runCmd.Flags().SetInterspersed(false)
	gearCmd.AddCommand(runCmd)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export your gear to disk",
		Run: func(cmd *cobra.Command, args []string) {
			project := Setup()
			project.Export(args)
		},
	}
	gearCmd.AddCommand(exportCmd)

	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload your gear to the Flywheel system",
		Run: func(cmd *cobra.Command, args []string) {
			project := Setup()
			project.Upload(args)
		},
	}
	gearCmd.AddCommand(uploadCmd)
}
