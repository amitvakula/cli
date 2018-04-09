package command

import (
	. "fmt"
	"os"

	"github.com/spf13/cobra"

	"flywheel.io/fw/ops"
	. "flywheel.io/fw/util"
)

func (o *opts) login() *cobra.Command {
	var insecure bool
	cmd := &cobra.Command{
		Use:   "login [api-key]",
		Short: "Login to a Flywheel instance",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client, err := MakeClientWithCreds(args[0], insecure)
			Check(err)
			user, _, err := client.GetCurrentUser()
			Check(err)
			creds := &Creds{Key: args[0], Insecure: insecure}
			creds.Save()
			Println("You are now logged in as", user.Firstname, user.Lastname+"!")
		},
	}

	cmd.Flags().BoolVar(&insecure, "insecure", false, "Ignore SSL errors")
	return cmd
}
func (o *opts) logout() *cobra.Command {
	var insecure bool
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Delete your saved API key",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			Check(DeleteCreds())
			Println("You are now logged out.")
		},
	}

	cmd.Flags().BoolVar(&insecure, "insecure", false, "Ignore SSL errors")
	return cmd
}

func (o *opts) status() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "See your current login status",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ops.Status(o.Client)
		},
	}

	return cmd
}

func (o *opts) ls() *cobra.Command {
	var showIds bool
	cmd := &cobra.Command{
		Use:    "ls [path]",
		Short:  "Show remote files",
		Args:   cobra.MaximumNArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				args = append(args, "")
			}

			ops.Ls(o.Client, args[0], showIds)
		},
	}

	cmd.Flags().BoolVar(&showIds, "ids", false, "Display database identifiers")

	return cmd
}

func (o *opts) download() *cobra.Command {
	var output string
	var force bool
	var include []string
	var exclude []string
	cmd := &cobra.Command{
		Use:    "download [source-path]",
		Short:  "Download a remote file or container",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {

			if len(include) > 0 && len(exclude) > 0 {
				Println("The --include and --exclude filters are mutually exclusive; use one or the other.")
				os.Exit(1)
			}

			ops.Download(o.Client, args[0], output, force, include, exclude)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Destination filename (-- for stdout)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force download, without prompting")
	cmd.Flags().StringSliceVarP(&include, "include", "i", []string{}, "Download only these types")
	cmd.Flags().StringSliceVarP(&exclude, "exclude", "e", []string{}, "Download everything but these types")

	return cmd
}

func (o *opts) upload() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upload [destination-path] [local-file]",
		Short:  "Upload a remote file",
		Args:   cobra.ExactArgs(2),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.Upload(o.Client, args[0], args[1])
		},
	}
	return cmd
}
