package command

import (
	"flywheel.io/sdk/api"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"log"
	"os"
)

type queryRequest struct {
	Where   string `json:"where"`
	Table   string `json:"table"`
	Dataset string `json:"dataset"`
}

type queryResult struct {
	JobId   string `json:"jobId"`
	Rows [][]string `json:"rows"`
	Colums []string `json:"columns"`
}

func (o *opts) ghcCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ghc",
		Short: "Google Healthcare API commands",
	}

	cmd.AddCommand(o.bigQueryCommand())

	return cmd
}

func (o *opts) bigQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query [DATASET] [TABLE] [SQL:WHERE]",
		Short:  "Run a query",
		Args:   cobra.ExactArgs(3),
		PreRun: o.RequireClient,
		Run: func(cmd *cobra.Command, args []string) {
			res1D := &queryRequest{
				Where:   args[2],
				Table:   args[1],
				Dataset:   args[0]}
			log.Println("Running big query")

			var aerr *api.Error
			var response *queryResult

			o.Client.New().Post("ghc/query").BodyJSON(res1D).Receive(&response, &aerr)

			fmt.Printf("Job Id: %s\n", response.JobId)

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader(response.Colums)

			for _, row := range response.Rows {
				table.Append(row)
			}

			table.Render()
		},
	}

	return cmd
}
