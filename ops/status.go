package ops

import (
	"os"
	"strings"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func Status(client *api.Client) {
	if client == nil {
		Println("You are not currently logged in.")
		Println("Try `fw login` to login to Flywheel.")
		os.Exit(1)
	}

	user, _, err := client.GetCurrentUser()
	if err != nil {
		Println(err)
		Println()
		Println("Could not authenticate - are you sure your API key is up to date?")
		Println("Try `fw login` to login to Flywheel.")
		os.Exit(1)
	}

	// Shenanigans: grab the URL string and drop the API prefix for a convenient browser URL
	req, err := client.Sling.Request()
	Check(err)
	hostname := strings.TrimSuffix(req.URL.String(), "/api/")
	hostname = strings.TrimSuffix(hostname, ":443")

	Println("You are currently logged in as", user.Firstname, user.Lastname, "to", hostname)
}
