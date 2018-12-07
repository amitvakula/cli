package gears

import (
	. "fmt"
	"os"

	"github.com/docker/docker/client"
)

func CheckDocker() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return cli, err
	}

	_, err = cli.Ping(ctx)
	return cli, err
}

func DockerOrBust() *client.Client {
	cli, err := CheckDocker()

	// Gee, what a great user experience
	if err != nil {
		Println(err)
		Println()
		Println("Could not connect to Docker.")
		Println()
		Println("If you haven't installed yet, visit:")
		Println("https://docs.docker.com/install/#supported-platforms")
		Println()
		Println("If you have, check that you can run 'docker ps' successfully.")
		os.Exit(1)
	}

	return cli
}
