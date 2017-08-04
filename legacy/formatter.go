package legacy

import (
	. "fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"

	"flywheel.io/sdk/api"
)

func FindPermissionById(id string, perms []*api.Permission) *api.Permission {
	for _, x := range perms {
		if x.Id == id {
			return x
		}
	}
	return &api.Permission{
		Id:    id,
		Level: "none",
	}
}

var blueBold = color.New(color.FgBlue, color.Bold).SprintFunc()
var timeFormat = "Jan _2 15:04"

func resolvePermissions(x interface{}) []*api.Permission {
	switch x := x.(type) {
	case *Group:
		return x.Permissions
	case *Project:
		return x.Permissions
	case *Session:
		return x.Permissions
	case *Acquisition:
		return x.Permissions
	default:
		Printf("Error: resolvePermissions: unexpected type %T\n", x)
		os.Exit(1)
	}

	return nil
}

// Timestamps might be null (eye-roll), let's not NPE on silly data
func tryTimestampFormat(t *time.Time, layout string) string {
	if t != nil {
		return t.Format(layout)
	} else {
		return ""
	}
}

func PrintResolve(r *ResolveResult, userId string, showDbIds bool) {

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
		case *Group:
			level := FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\n", level, blueBold(x.Id))

		case *Project:
			level := FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\n", level, blueBold(x.Name))

		case *Session:
			level := FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\n", level, tryTimestampFormat(x.Timestamp, timeFormat), x.Subject.Code, blueBold(x.Name))

		case *Acquisition:
			level := FindPermissionById(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t\n", level, tryTimestampFormat(x.Timestamp, timeFormat), blueBold(x.Name))

		case *File:
			level := FindPermissionById(userId, resolvePermissions(parent)).Level
			printId(x.Name)
			Fprintf(w, "%s\t%s\t\t%s\t\n", level, x.Modified.Format(timeFormat), x.Name)

		default:
			Printf("Error: printing unexpected type %T\n", node)
			os.Exit(1)
		}
	}

	w.Flush()
}
