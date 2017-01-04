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

func resolvePermissions(x interface{}) []*api.Permission {
	switch x := x.(type) {
	case *api.Group:
		return x.Permissions
	case *api.Project:
		return x.Permissions
	case *api.Session:
		return x.Permissions
	case *api.Acquisition:
		return x.Permissions
	default:
		Printf("Error: resolvePermissions: unexpected type %T\n", x)
		os.Exit(1)
	}

	return nil
}

func PrintResolve(r *api.ResolveResult, userId string, showDbIds bool) {

	// Format the table, printing to a platform- & pipe-friendly color writer
	w := tabwriter.NewWriter(color.Output, 0, 2, 1, ' ', 0)

	// Closure for printing object ID, if enabled
	printId := func(id string) {
		if showDbIds {
			Fprintf(w, "<id:%s>\t", id)
		}
	}

	var parent interface{}
	if len(r.Path) > 0 {
		parent = r.Path[len(r.Path)-1]
	}

	target := r.Children

	// Special case: leaf node. Back up the tree one level.
	if len(target) == 0 && len(r.Path) > 1 {
		target = []interface{}{parent}
		parent = r.Path[len(r.Path)-2]
	}

	for _, node := range target {
		switch x := node.(type) {
		case *api.Group:
			level := api.FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\n", level, blueBold(x.Id), x.Name)

		case *api.Project:
			level := api.FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\n", level, blueBold(x.Name))

		case *api.Session:
			level := api.FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\n", level, x.Timestamp.Format(timeFormat), x.Subject.Name, blueBold(x.Name))

		case *api.Acquisition:
			level := api.FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\t\n", level, x.Timestamp.Format(timeFormat), x.Measurement, blueBold(x.Name))

		case *api.File:
			level := api.FindPermissionById(userId, resolvePermissions(parent)).Level
			printId(x.Name)
			Fprintf(w, "%s\t%s\t\t%s\t\n", level, x.Modified.Format(timeFormat), x.Name)

		default:
			Printf("Error: printing unexpected type %T\n", node)
			os.Exit(1)
		}
	}

	w.Flush()
}
