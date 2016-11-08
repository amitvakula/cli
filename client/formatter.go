package client

import (
	. "fmt"
	"os"

	"text/tabwriter"

	"github.com/fatih/color"

	"flywheel.io/fw/api"
)

var blueBold = color.New(color.FgBlue, color.Bold).SprintFunc()
var timeFormat = "Jan _2 15:04"

func PrintResolve(r *api.ResolveResult) {

	// Format the table, printing to a platform- & pipe-friendly color writer
	w := tabwriter.NewWriter(color.Output, 0, 2, 1, ' ', 0)

	// Closure for printing object ID, if enabled
	printId := func(id string) {
		// if showDbIds {
		// 	Fprintf(w, "<id:%s>\t", id)
		// }
	}

	// Closure for dedup
	printFiles := func(files []*api.File, perms []*api.Permission) {
		// level := findPermission(userId, perms).Level
		for _, x := range files {
			printId(x.Name)
			Fprintf(w, "%s\t%s\t\t%s\t\n", "lvl", x.Modified.Format(timeFormat), x.Name)
		}
	}

	for _, node := range r.Path {
		switch x := node.(type) {
		case *api.Group:
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\n", "lvl", blueBold(x.Id), x.Name)
		case *api.Project:
			// level := findPermission(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\n", "lvl", blueBold(x.Name))

		case *api.Session:
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\n", "lvl", x.Timestamp.Format(timeFormat), x.Subject.Name, blueBold(x.Name))

		case *api.Acquisition:
			printFiles(x.Files, x.Permissions)

		case *api.File:
			printFiles([]*api.File{x}, []*api.Permission{})

		default:
			Printf("Error: printing unexpected type %T\n", node)
			os.Exit(1)
		}
	}

	w.Flush()
}
