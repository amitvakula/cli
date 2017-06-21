package ops

import (
	"strings"
	"sync"

	"flywheel.io/sdk/api"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
)

func Ls(client *api.Client, upath string, showDbIds bool) {
	upath = strings.TrimRight(upath, "/")
	parts := strings.Split(upath, "/")

	var wg sync.WaitGroup
	var user *api.User
	var result *legacy.ResolveResult

	go func() {
		var err error
		user, _, err = client.GetCurrentUser()
		Check(err)
		wg.Done()
	}()

	go func() {
		var err error
		var aerr *api.Error
		result, _, err, aerr = legacy.ResolvePath(client, parts)
		Check(api.Coalesce(err, aerr))
		wg.Done()
	}()

	wg.Add(2)
	wg.Wait()
	legacy.PrintResolve(result, user.Id, showDbIds)
}
