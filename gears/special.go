package gears

import (
	"github.com/docker/docker/api/types/container"
)

func (docker *D) PythonInstalled(imageName string) bool {
	containerID, cleanup := docker.CreateContainerFromImage(imageName, &container.Config{
		Entrypoint: []string{},
		Cmd:        []string{"sh", "-c", "hash python"},
	}, nil)
	defer cleanup()

	code := docker.WaitOnContainer(containerID)
	return code == 0
}
