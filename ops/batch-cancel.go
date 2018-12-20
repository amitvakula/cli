package ops

import (
	"flywheel.io/sdk/api"

	. "flywheel.io/fw/util"
)

func BatchCancel(client *api.Client, id string) {
	count, _, err := client.CancelBatch(id)
	Check(err)

	Println("Cancelled", count, "jobs.")
}
