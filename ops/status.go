package ops

import (
	"errors"
	"fmt"
	"strings"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func Status(client *api.Client) {
	if client == nil {
		Println("You are not currently logged in.")
		Println("Try `fw login` to login to Flywheel.")
		Fatal(1)
	}

	id, err := GetLoginId(client)
	if err != nil {
		Println(err)
		Println()
		Println("Could not authenticate - are you sure your API key is up to date?")
		Println("Try `fw login` to login to Flywheel.")
		Fatal(1)
	}

	// Shenanigans: grab the URL string and drop the API prefix for a convenient browser URL
	req, err := client.Sling.Request()
	Check(err)
	hostname := strings.TrimSuffix(req.URL.String(), "/api/")
	hostname = strings.TrimSuffix(hostname, ":443")

	Println("You are currently logged in as", id, "to", hostname)
}

func GetLoginId(client *api.Client) (string, error) {
	if client == nil {
		return "", errors.New("Not logged in")
	}

	// Get auth status to determine if we're device or user
	status, _, err := client.GetAuthStatus()
	if err != nil {
		return "", err
	}

	if status.Device != nil && *status.Device {
		// If we're a device, return the device id
		return fmt.Sprintf("Device(%s)", status.Origin.Id), nil
	} else {
		user, _, err := client.GetCurrentUser()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s", user.Firstname, user.Lastname), nil
	}
}
