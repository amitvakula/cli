package gears

import (
	"encoding/json"
	. "fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	prompt "github.com/segmentio/go-prompt"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearCreate(client *api.Client, docker *client.Client) {

	user, _, err := client.GetCurrentUser()
	Check(err)

	Println("Welcome to gear creation! Let's get started.")
	Println()

	Println()
	gearLabel := PromptOrDefault("What will be the name of your gear?", "My Gear")
	gearName := strings.ToLower(gearLabel)
	gearName = strings.Replace(gearName, " ", "-", -1)

	gearDescription := "Gear created with gear builder"

	Println()

	choices := []string{"Python", "Just Linux (Ubuntu & Bash)", "Advanced configuration example", "Custom - use a docker image"}
	results := []string{"flywheel/gear-base-anaconda", "flywheel/base-gear-ubuntu", "flywheel/example-gear", ""}

	options := LoadOptions()

	for _, x := range options.CustomGearImages {
		choices = append(choices, "[Recent image] "+x)
		results = append(results, x)
	}

	if len(choices) != len(results) {
		panic("Choices and results are not of same length, consult developers")
	}

	Println("What sort of gear would you like to create?")
	for x, y := range choices {
		Println("\t", x+1, ")", y)
	}
	choice := -1

	for choice < 0 || choice > len(choices) {
		rawChoice := prompt.String("Choose")
		var err error
		choice, err = strconv.Atoi(rawChoice)
		choice--

		if err != nil {
			choice = -1
		}
	}
	image := results[choice]

	if image == "" {
		image = prompt.StringRequired("Enter the name of your docker image")

		stringInSlice := func(a string, list []string) bool {
			for _, b := range list {
				if b == a {
					return true
				}
			}
			return false
		}

		// Have we used this image recently?
		if !stringInSlice(image, options.CustomGearImages) {

			// Add custom image to the top of the recent list, trim list to most recent 5
			options.CustomGearImages = append([]string{image}, options.CustomGearImages...)
			if len(options.CustomGearImages) > 5 {
				options.CustomGearImages = options.CustomGearImages[:5]
			}

			options.Save()
		}
	}

	Println()
	author := PromptOrDefault("Who is the author of this gear?", user.Firstname+" "+user.Lastname)

	gearPath := "/flywheel/v0"

	Println()
	Println("Downloading gear...")
	pullProgress, err := docker.ImagePull(background, image, types.ImagePullOptions{})
	Check(err)
	io.Copy(ioutil.Discard, pullProgress)
	pullProgress.Close()

	containerId, cleanup, err := CreateContainerWithCleanup(docker, background, &container.Config{Image: image}, nil, "")
	Check(err)
	defer cleanup()

	var reader io.ReadCloser
	var stat types.ContainerPathStat
	// var copyError error

	Println("Reading gear contents...")
	reader, stat, err = docker.CopyFromContainer(background, containerId, gearPath)

	if err != nil && strings.HasSuffix(err.Error(), "no such file or directory") {
		Println()
		Println("This docker image does not appear to be a Flywheel Gear;")
		Println("  the /flywheel/v0 folder is missing.")
		Println("Providing an example gear script to get you started...")

		pullProgress, pullErr := docker.ImagePull(background, "flywheel/base-gear-ubuntu", types.ImagePullOptions{})
		Check(pullErr)
		io.Copy(ioutil.Discard, pullProgress)
		pullProgress.Close()

		containerId, cleanup, createErr := CreateContainerWithCleanup(docker, background, &container.Config{Image: "flywheel/base-gear-ubuntu"}, nil, "")
		Check(createErr)
		defer cleanup()

		reader, stat, err = docker.CopyFromContainer(background, containerId, gearPath)
	}
	Check(err)

	if !stat.Mode.IsDir() {
		Println("Error: container path", gearPath, "is not a folder!")
		os.Exit(1)
	}

	err = UntarGearFolder(reader)
	Check(err)

	gear := ManifestOrDefaultGear()

	gear.Name = gearName
	gear.Label = gearLabel
	gear.Description = gearDescription
	gear.Version = "0"
	gear.Author = author
	gear.Maintainer = author
	gear.License = "Other"
	gear.Source = ""
	gear.Url = ""
	gear.Custom = map[string]interface{}{}
	gear.Custom["gear-builder"] = map[string]interface{}{
		"image": image,
	}

	raw, err := json.MarshalIndent(gear, "", "\t")
	Check(err)
	err = ioutil.WriteFile("manifest.json", raw, 0644)
	Check(err)

	// Println(string(raw))

	Println()
	Println()
	Println("Your gear is created and expanded to the working directory.")
	Println("Try `fw gear run` to run the gear!")
}
