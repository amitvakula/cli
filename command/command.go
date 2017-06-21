package command

import (
	. "fmt"
	"os"

	"github.com/spf13/cobra"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/ops"
)

// Each command is separated into its own function.
// This is to prevent any flag variable pointers from cross-contaminating.

func BuildCommand(version, buildHash, buildDate string) *cobra.Command {
	o := opts{}

	cmd := o.fw()
	cmd.AddCommand(o.login())
	cmd.AddCommand(o.logout())
	cmd.AddCommand(o.status())
	cmd.AddCommand(o.ls())
	cmd.AddCommand(o.download())
	cmd.AddCommand(o.upload())
	cmd.AddCommand(o.batch())
	cmd.AddCommand(o.gear())
	cmd.AddCommand(o.job())
	cmd.AddCommand(o.scan())
	cmd.AddCommand(o.version(version, buildHash, buildDate))

	return cmd
}

type opts struct {
	Client *api.Client
}

func (o *opts) fw() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fw",
		Short: "Flywheel command-line interface",

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			//
			// Check RequireClient below when making any changes to this function.
			//
			client, _ := MakeClient()
			o.Client = client
		},
	}

	return cmd
}

func (o *opts) version(version, buildHash, buildDate string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ops.Version(version, buildHash, buildDate)
		},
	}

	return cmd
}

// Helper func that requires a valid API key on disk
func (o *opts) RequireClient(cmd *cobra.Command, args []string) {

	// If you use RequireClient as a PersistentPreRun on a subcommand, it
	// will obliterate the root command's closure. For this reason, duplicate
	// what it does here!
	if o.Client == nil {
		client, _ := MakeClient()
		o.Client = client
	}

	if o.Client == nil {
		Println("You are not currently logged in.")
		Println("Try `fw login` to login to Flywheel.")
		os.Exit(1)
	}
}