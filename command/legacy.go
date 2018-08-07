package command

import (
	"github.com/spf13/cobra"
)

// Keep legacy import & export in case the new functions
// aren't working out for someone
func (o *opts) legacyCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "legacy",
		Short:  "Old versions of commands",
		Hidden: true,
	}

	cmd.AddCommand(o.importCommand())
	cmd.AddCommand(o.exportCommand())

	return cmd
}
