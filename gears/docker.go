package gears

import (
	. "fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

var background = context.Background()

func CheckDocker() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return cli, err
	}

	_, err = cli.Ping(background)
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
		Println("https://store.docker.com/search?offering=community&type=edition")
		Println()
		Println("If you have, check that you can run 'docker ps' successfully.")
		os.Exit(1)
	}

	return cli
}

//
func CreateContainerWithCleanup(docker *client.Client, ctx context.Context, config *container.Config, hostConfig *container.HostConfig, containerName string) (string, func(), error) {

	// ctx context.Context, config *container.Config,
	// hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string

	resp, err := docker.ContainerCreate(ctx, config, hostConfig, nil, containerName)
	containerId := resp.ID

	cleanup := func() {
		docker.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{
			// RemoveVolumes: true,
			// RemoveLinks: true,
			Force: true,
		})
	}

	return containerId, cleanup, err
}

func TranslateEnvToEnvArray(env map[string]string) []string {
	var result []string

	for key, value := range env {
		result = append(result, key+"="+value)
	}

	return result
}
