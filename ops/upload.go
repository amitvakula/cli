package ops

import (
	. "fmt"
	"strings"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func Upload(client *api.Client, upath, sendPath string) {
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	result, _, err, aerr := legacy.ResolvePath(client, parts)
	Check(api.Coalesce(err, aerr))
	path := result.Path

	_, err = legacy.UploadFromFile(client, sendPath, path[len(path)-1], nil, sendPath)
	Check(err)

	Println("Uploaded file to", upath+".")
}
