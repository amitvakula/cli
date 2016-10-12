package main

import (
	. "fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	var loginUrl string
	var loginKey string
	var loginInsecure bool
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to a Flywheel instance",
		Run: func(cmd *cobra.Command, args []string) {
			if loginUrl == "" {
				Println("--host flag is required.")
				os.Exit(1)
			} else if loginKey == "" {
				Println("--key flag is required.")
				os.Exit(1)
			}

			parsedUrl, err := url.Parse(loginUrl)
			check(err)
			login(parsedUrl.Host, loginKey, loginInsecure)
		},
	}
	loginCmd.Flags().StringVarP(&loginUrl, "host", "H", "", "Host URL (https://example.flywheel.io)")
	loginCmd.Flags().StringVarP(&loginKey, "key", "k", "", "Your API key")
	loginCmd.Flags().BoolVar(&loginInsecure, "insecure", false, "Ignore SSL errors")
	RootCmd.AddCommand(loginCmd)

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "Show remote files",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				ls("")
			} else if len(args) == 1 {
				ls(args[0])
			} else {
				Println("ls takes one argument: the path of the files to list.")
				os.Exit(1)
			}

		},
	}
	RootCmd.AddCommand(lsCmd)

	var downloadOutput string
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download a remote file",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				Println("ls takes one argument: the path of the files to list.")
				os.Exit(1)
			}
			download(args[0], downloadOutput)
		},
	}
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Destination filename (-- for stdout)")
	RootCmd.AddCommand(downloadCmd)
}
