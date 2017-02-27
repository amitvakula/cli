package main

import (
	. "fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	// Register all implementations
	_ "flywheel.io/deja/flak/features"

	"flywheel.io/fw/builder"
	"flywheel.io/fw/gear"
)

func init() {

	gearCmd := &cobra.Command{
		Use:   "gear",
		Short: "Gear commands",
	}
	RootCmd.AddCommand(gearCmd)

	var importFilepath string
	gearImportCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a gear to run locally.",
		Run: func(cmd *cobra.Command, args []string) {
			if importFilepath == "" {
				Println("--filepath flag is required.")
				os.Exit(1)
			}
			gear.Import(importFilepath)
		},
	}
	gearImportCmd.Flags().StringVarP(&importFilepath, "filepath", "f", "", "Gear filepath (must be local)")
	gearCmd.AddCommand(gearImportCmd)

	var downloadUrl string
	gearSetDownloadCmd := &cobra.Command{
		Use:   "set-download",
		Short: "Set a download URL for this gear after uploading it somewhere.",
		Run: func(cmd *cobra.Command, args []string) {
			if downloadUrl == "" {
				Println("--url flag is required.")
				os.Exit(1)
			}
			gear.SetDownload(downloadUrl)
		},
	}
	gearSetDownloadCmd.Flags().StringVarP(&downloadUrl, "url", "u", "", "Gear download url")
	gearCmd.AddCommand(gearSetDownloadCmd)

	var configs interface{}
	var values interface{}

	gearRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a gear locally",
		Run: func(cmd *cobra.Command, args []string) {

			// Merge value slice with config slice for convenience
			cs := configs.([]*gear.GearConfig)
			for x := range cs {
				cs[x].Value = values.([]string)[x]
			}

			// RunGear can parse out the strings into their correct types.
			gear.RunGear(cs)
		},
	}

	// If there is a gear.json in the current folder, read it and generate flags at launch-time.
	cs := gear.GetGearConfigs()
	values = make([]string, len(cs))
	configs = cs
	for index, config := range cs {
		gearRunCmd.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	// Okay, so, okay, so...
	// Dynamically generating flags at runtime --> we need to use interface{} type for configs/values.
	// Otherwise, you'd need to cast when creating [Type]Var flags, and you can't take the address of a cast.
	// So, each flag, at the parser level, must be the lowest-common denominator: a string.
	// Then, we manually get typed scalars back from the strings the user passed. Fine.
	// EXCEPT - the default usage template happily includes the TYPE of each flag!
	// This results in a farce:
	//   Flags:
	//     --boolean string    Any boolean.
	//
	// Well that's no fun. Options from here are:
	// A) Solve the compile-time declaration of dynamic slices that can *also* be casted for cobra;
	// B) Ditch half of cobra here and start doing things ourselves;
	// C) Override the default usage template, and either hide the types or backfill them out.
	//
	// Option C-1 is the least-irritating for now. Such is life.
	gearRunCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace | trimStringLiterals}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)

	// This function will regex-replace every line in a template output. Ref above block.
	// Before:
	//      --boolean string    Any boolean.
	// After:
	//      --boolean    Any boolean.
	trimStringLiterals := func(s string) string {
		removeStringLiteral := regexp.MustCompile(`^( +\-\-.*?) string(.*)$`)

		parts := strings.Split(s, "\n")
		for x, part := range parts {
			parts[x] = removeStringLiteral.ReplaceAllString(part, "${1}${2}")
		}

		return strings.Join(parts, "\n")
	}
	cobra.AddTemplateFunc("trimStringLiterals", trimStringLiterals)

	gearCmd.AddCommand(gearRunCmd)

	//

	builderCmd := &cobra.Command{
		Use:   "builder",
		Short: "Builder commands",
	}
	RootCmd.AddCommand(builderCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new flywheel gear",
		Run: func(cmd *cobra.Command, args []string) {
			builder.Setup()
			Println("Created a new gear in the current folder.")
		},
	}
	builderCmd.AddCommand(createCmd)

	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Use a new flywheel gear",
		Run: func(cmd *cobra.Command, args []string) {
			project := builder.Setup()
			project.Use(args)
		},
	}
	builderCmd.AddCommand(useCmd)

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run your gear locally",
		Run: func(cmd *cobra.Command, args []string) {
			project := builder.Setup()
			project.Run(args)
		},
	}
	runCmd.Flags().SetInterspersed(false)
	builderCmd.AddCommand(runCmd)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export your gear to disk",
		Run: func(cmd *cobra.Command, args []string) {
			project := builder.Setup()
			project.Export(args)
		},
	}
	builderCmd.AddCommand(exportCmd)

	// uploadCmd := &cobra.Command{
	// 	Use:   "upload",
	// 	Short: "Upload your gear to the Flywheel system",
	// 	Run: func(cmd *cobra.Command, args []string) {
	// 		project := builder.Setup()
	// 		project.Upload(args)
	// 	},
	// }
	// builderCmd.AddCommand(uploadCmd)
}
