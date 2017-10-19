package gears

import (
	. "fmt"
	"io"
	"io/ioutil"
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

	// Closure because we need this twice
	createContainer := func() (string, error) {
		resp, err := docker.ContainerCreate(ctx, config, hostConfig, nil, containerName)
		return resp.ID, err
	}

	// Attempt to just create the container.
	containerId, createErr := createContainer()

	// Attempt to pull if container create failed.
	// Should we check specifically for "no such image"?. Maybe, but maybe not. Let's go for broke.
	if createErr != nil {

		// Is there even an image to try?
		if config.Image == "" {
			Println("No image provided and ref does not exist locally")
			return "", func() {}, createErr
		}

		Println("Downloading " + config.Image + "...")

		pullProgress, pullErr := docker.ImagePull(ctx, config.Image, types.ImagePullOptions{})
		io.Copy(ioutil.Discard, pullProgress)
		pullProgress.Close()

		// Even the pull failed? Give up.
		if pullErr != nil {
			return "", func() {}, pullErr
		}

		// Okay, the pull succeeded, let's try to create again.
		containerId, createErr = createContainer()
	}

	cleanup := func() {
		docker.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{
			// RemoveVolumes: true,
			// RemoveLinks: true,
			Force: true,
		})
	}

	return containerId, cleanup, createErr
}

func TranslateEnvToEnvArray(env map[string]string) []string {
	var result []string

	for key, value := range env {
		result = append(result, key+"="+value)
	}

	return result
}
