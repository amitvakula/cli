package ops

import (
	. "fmt"
	"os"
	"text/tabwriter"
)

func Version(version, buildHash, buildDate string) {
	Println("flywheel-cli")

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	Fprintf(w, "%s\t%s\n", " Version:", version)
	Fprintf(w, "%s\t%s\n", " Git commit:", buildHash)
	Fprintf(w, "%s\t%s\n", " Built:", buildDate)
	w.Flush()
}
