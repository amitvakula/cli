package command

import (
	. "fmt"

	"github.com/spf13/cobra"

	"flywheel.io/fw/dicom"
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
	cmd := &cobra.Command{
		Use:    "download [source-path]",
		Short:  "Download a remote file or container",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.Download(o.Client, args[0], output, force)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "Destination filename (-- for stdout)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force download, without prompting")

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

func (o *opts) importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "import [folder]",
		Short:  "Import a structured folder into Flywheel",
		Args:   cobra.ExactArgs(1),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			ops.ScanUpload(o.Client, args[0])
		},
	}

	return cmd
}

func (o *opts) scan() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "scan [folder] [group] [project]",
		Short:  "Scan a folder of dicom files and upload",
		Args:   cobra.ExactArgs(3),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {

			// related_acq is marked as not working, so force-set to false
			// force-set log_level to 2; change out for a verbose flag later.

			dicom.Scan(o.Client, args[0], args[1], args[2], false, 2)
		},
	}

	return cmd
}
