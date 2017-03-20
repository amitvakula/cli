package main

import (
	. "fmt"
	"net/url"
	"os"
	"regexp"

	"github.com/spf13/cobra"

	"flywheel.io/fw/client"
	. "flywheel.io/fw/util"
)

func init() {

	// Generate a template without the "[flags]" string.
	// Copies the template from a default command and modifies it, making it potentially more fragile
	// but less forked from upstream. This approach could be adopted in commands_linux.go
	//
	// https://github.com/flywheel-io/cli/issues/21
	// https://github.com/spf13/cobra/issues/395
	// https://godoc.org/github.com/spf13/cobra#Command.UsageTemplate
	dummyCmd := &cobra.Command{}
	defaultTemplate := dummyCmd.UsageTemplate()
	templateWithoutFlags := regexp.MustCompile("\\[flags\\]").ReplaceAllString(defaultTemplate, "")
	_ = templateWithoutFlags

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
			Check(err)
			client.Login(parsedUrl.Host, loginKey, loginInsecure)
		},
	}
	loginCmd.Flags().StringVarP(&loginUrl, "host", "H", "", "Host URL (https://example.flywheel.io)")
	loginCmd.Flags().StringVarP(&loginKey, "key", "k", "", "Your API key")
	loginCmd.Flags().BoolVar(&loginInsecure, "insecure", false, "Ignore SSL errors")
	RootCmd.AddCommand(loginCmd)

	var lsDbIds bool
	lsCmd := &cobra.Command{
		Use:   "ls [path]",
		Short: "Show remote files",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				client.Ls("", lsDbIds)
			} else if len(args) == 1 {
				client.Ls(args[0], lsDbIds)
			} else {
				Println("ls takes one argument: the path of the files to list.")
				os.Exit(1)
			}

		},
	}
	lsCmd.Flags().BoolVar(&lsDbIds, "ids", false, "Display database identifiers")
	RootCmd.AddCommand(lsCmd)

	var downloadOutput string
	var downloadForce bool
	downloadCmd := &cobra.Command{
		Use:   "download [source_path]",
		Short: "Download a remote file or container",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				Println("ls takes one argument: the path of the files to list.")
				os.Exit(1)
			}
			client.Download(args[0], downloadOutput, downloadForce)
		},
	}
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Destination filename (-- for stdout)")
	downloadCmd.Flags().BoolVarP(&downloadForce, "force", "f", false, "Force download, without prompting")
	RootCmd.AddCommand(downloadCmd)

	uploadCmd := &cobra.Command{
		Use:   "upload [destination_path] [local_file]",
		Short: "Upload a remote file",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				Println("upload takes two arguments: the remote upload path, and the file to upload.")
				os.Exit(1)
			}
			client.Upload(args[0], args[1])
		},
	}
	uploadCmd.SetUsageTemplate(templateWithoutFlags)
	RootCmd.AddCommand(uploadCmd)

	scanCmd := &cobra.Command{
		Use:   "scan [folder]",
		Short: "Scan and upload a folder",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				Println("scan takes one arguments: the folder to upload")
				os.Exit(1)
			}
			client.ScanUpload(args[0])
		},
	}
	scanCmd.SetUsageTemplate(templateWithoutFlags)
	RootCmd.AddCommand(scanCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			Println("flywheel-cli version", Version)
		},
	}
	versionCmd.SetUsageTemplate(templateWithoutFlags)
	RootCmd.AddCommand(versionCmd)

	batchCmd := &cobra.Command{
		Use:   "batch",
		Short: "Start or manage server batch jobs",
	}
	RootCmd.AddCommand(batchCmd)

	batchRunCmd := &cobra.Command{
		Use:   "run [gear] [folders...]",
		Short: "Start a batch job.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				Println("run takes two or more arguments: the gear name, and the folder(s) to run on.")
				os.Exit(1)
			}

			client.BatchRun(args)
		},
	}
	batchDefaultHelpFunc := batchRunCmd.HelpFunc()
	batchRunCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 3 || args[2] == "-h" || args[2] == "--help" {
			batchDefaultHelpFunc(cmd, args)
		} else {
			client.BatchRun([]string{args[2], "-h"})
		}

	})
	batchRunCmd.Flags().SetInterspersed(false) // passthrough
	batchCmd.AddCommand(batchRunCmd)

	batchCancelCmd := &cobra.Command{
		Use:   "cancel [batch-id]",
		Short: "Cancel a batch job.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				Println("run takes one argument: the batch id.")
				os.Exit(1)
			}

			client.BatchCancel(args[0])
		},
	}
	batchCmd.AddCommand(batchCancelCmd)

	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Start or manage server jobs",
	}
	RootCmd.AddCommand(jobCmd)

	jobRunCmd := &cobra.Command{
		Use:   "run [gear]",
		Short: "Start a job.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				Println("run takes one argument: the gear name.")
				os.Exit(1)
			}

			client.JobRun(args)
		},
	}
	jobDefaultHelpFunc := jobRunCmd.HelpFunc()
	jobRunCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if len(args) < 3 || args[2] == "-h" || args[2] == "--help" {
			jobDefaultHelpFunc(cmd, args)
		} else {
			client.JobRun([]string{args[2], "-h"})
		}

	})
	jobRunCmd.Flags().SetInterspersed(false) // passthrough
	jobCmd.AddCommand(jobRunCmd)

	jobStatusCmd := &cobra.Command{
		Use:   "status [job-id]",
		Short: "Check the status of a job.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				Println("status takes one argument: the job id.")
				os.Exit(1)
			}

			client.JobStatus(args[0])
		},
	}
	jobCmd.AddCommand(jobStatusCmd)

	jobWaitCmd := &cobra.Command{
		Use:   "wait [job-id]",
		Short: "Wait for a job to finish.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				Println("wait takes one argument: the job id.")
				os.Exit(1)
			}

			client.JobWait(args[0])
		},
	}
	jobCmd.AddCommand(jobWaitCmd)

	jobListGearsCmd := &cobra.Command{
		Use:   "list-gears",
		Short: "List available gears.",
		Run: func(cmd *cobra.Command, args []string) {
			client.ListGears()
		},
	}
	jobCmd.AddCommand(jobListGearsCmd)
}
