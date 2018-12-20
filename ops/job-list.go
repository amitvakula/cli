package ops

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/fatih/color"

	"flywheel.io/sdk/api"

	. "flywheel.io/fw/util"
)

var greenBold = color.New(color.FgGreen, color.Bold).SprintFunc()

func ListGears(client *api.Client) {
	gears, _, err := client.GetAllGears()
	Check(err)

	// Change the type so we can sort
	gearsCast := Gears(gears)
	sort.Sort(gearsCast)

	// Format the table, printing to a platform- & pipe-friendly color writer
	w := tabwriter.NewWriter(color.Output, 0, 2, 1, ' ', 0)

	for _, x := range gearsCast {
		fmt.Fprintf(w, "%s\t%s\n", greenBold(x.Gear.Name), x.Gear.Label)
	}

	w.Flush()
}

// Gears satisfies sort.Interface for sorting by gear name.
type Gears []*api.GearDoc

func (g Gears) Len() int {
	return len(g)
}
func (g Gears) Less(i, j int) bool {
	return g[i].Gear.Name < g[j].Gear.Name
}
func (g Gears) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}
