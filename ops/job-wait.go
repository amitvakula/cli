package ops

import (
	. "fmt"
	"time"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/util"
)

func JobWait(client *api.Client, id string) {
	first := true
	interval := 10 * time.Second
	state := api.JobState("")

	for state != api.Cancelled && state != api.Failed && state != api.Complete {

		if first {
			first = false
		} else {
			time.Sleep(interval)
		}

		job, _, err := client.GetJob(id)
		if err != nil {
			Println(err)
			Println("Will continue to retry. Press Control-C to exit.")
		}
		if job != nil {
			if job.State != state {
				state = job.State
				Println("Job is", state)
			}
			state = job.State
		}
	}

	if state == api.Complete {
		util.Fatal(0)
	} else {
		util.Fatal(1)
	}
}
