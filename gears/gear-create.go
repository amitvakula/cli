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

func GearCreate(client *api.Client, docker *client.Client, clearCustomList bool, name, author, image string) {

	user, _, err := client.GetCurrentUser()
	CheckM(err, ApiFailedMsg)

	Println("Welcome to gear creation! Let's get started.")
	Println()

	// Resolve choices via flags or prompts
	name, label := determineNameAndLabel(name)
	image = determineImage(image, clearCustomList)
	author = determineAuthor(user, author)

	// Trigger acquisition of image, if needed
	containerId, cleanup, err := CreateContainerWithCleanup(docker, background, &container.Config{Image: image}, nil, "")
	CheckM(err, CreateContainerFailedMsg)
	defer cleanup()

	var reader io.ReadCloser
	var stat types.ContainerPathStat
	// var copyError error

	Println()
	Println("Reading gear contents...")
	gearPath := "/flywheel/v0"
	reader, stat, err = docker.CopyFromContainer(background, containerId, gearPath)

	if err != nil && strings.HasSuffix(err.Error(), "no such file or directory") {
		Println()
		Println("This docker image does not appear to be a Flywheel Gear;")
		Println("  the /flywheel/v0 folder is missing.")
		Println("Providing an example gear script to get you started...")

		containerId, cleanup, createErr := CreateContainerWithCleanup(docker, background, &container.Config{Image: "flywheel/base-gear-ubuntu"}, nil, "")
		CheckM(createErr, CopyFromContainerFailedMsg)
		defer cleanup()

		reader, stat, err = docker.CopyFromContainer(background, containerId, gearPath)
	}
	CheckM(err, CopyFromContainerFailedMsg)

	if !stat.Mode.IsDir() {
		Println("Error: container path", gearPath, "is not a folder!")
		os.Exit(1)
	}

	err = UntarGearFolder(reader)
	CheckM(err, UntarFailedMsg)

	gear := ManifestOrDefaultGear()

	gearDescription := "Gear created with gear builder."

	if gear.Description != "" {
		gearDescription = gear.Description + " [" + gearDescription + "]"
	}
	gear.Description = gearDescription

	gear.Name = name
	gear.Label = label
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
	CheckM(err, JsonFailedMsg)
	err = ioutil.WriteFile("manifest.json", raw, 0644)
	CheckM(err, WriteFailedMsg)

	Println()
	Println()
	Println("Your gear is created and expanded to the working directory.")
	Println("Try `fw gear run` to run the gear!")
}

func determineNameAndLabel(label string) (string, string) {
	if label == "" {
		label = PromptOrDefault("What will be the name of your gear?", "My Gear")
	}

	name := strings.Replace(label, " ", "-", -1)
	name = strings.ToLower(name)

	return name, label
}

func determineImage(image string, clearCustomList bool) string {
	options := LoadOptions()
	if clearCustomList {
		options.CustomGearImages = []string{}
		options.Save()
	}

	if image != "" {
		return image
	}

	Println()
	choices := []string{"Python", "Just Linux (Ubuntu & Bash)", "Advanced configuration example", "Custom - use a docker image"}
	results := []string{"flywheel/gear-base-anaconda", "flywheel/base-gear-ubuntu", "flywheel/example-gear", ""}

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
	image = results[choice]

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

	return image
}

func determineAuthor(user *api.User, author string) string {
	if author != "" {
		return author
	} else {
		return user.Firstname + " " + user.Lastname
	}
}
