package ops

import (
	. "fmt"

	"flywheel.io/sdk/api"

	. "flywheel.io/fw/util"
)

func JobStatus(client *api.Client, id string) {
	job, _, err := client.GetJob(id)
	Check(err)

	Println("Job is", job.State+".")
}
