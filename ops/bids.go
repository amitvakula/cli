package ops

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"golang.org/x/net/context"
)

//  TODO: Keep this up to date with major release version
const BidsContainerVersion = "0.3"
const BidsContainerName = "flywheel/bids-client"

func ImportBids(docker *client.Client, apiKey string, folder string, group_id string, projectLabel string) {
	// Make sure that bidsDir is an absolute path

	bidsDir, err := filepath.Abs(folder)
	if err != nil {
		Fprintln(os.Stderr, "Could not resolve source directory:", folder)
		os.Exit(1)
	}
	// If optional project label flag not given, use base directory of folder path
	if projectLabel == "" {
		projectLabel = filepath.Base(folder)
	}
	// Map The destination dir to /local/bids
	binding := Sprintf("%s:/local/bids", bidsDir)

	cmd := []string{
		"python", "code/upload_bids.py",
		"--bids-dir", "/local/bids",
		"--api-key", apiKey,
		"-g", group_id,
		"-p", projectLabel,
	}

	status, err := runBidsCmdInContainer(docker, []string{binding}, cmd)
	if err != nil {
		// Intentionally obtuse error message, ideally we would hide that we're
		// calling into a container
		Fprintln(os.Stderr, "Error importing BIDS data --", err.Error())
		os.Exit(1)
	} else {
		if status != 0 {
			os.Exit(int(status))
		}
	}
}

func ExportBids(docker *client.Client, apiKey string, folder string, projectLabel string, sourceData bool) {
	// Make sure that bidsDir is an absolute path
	bidsDir, err := filepath.Abs(folder)
	if err != nil {
		Fprintln(os.Stderr, "Could not resolve target directory:", folder)
		os.Exit(1)
	}

	// Map The destination dir to /local/bids
	binding := Sprintf("%s:/local/bids", bidsDir)

	cmd := []string{
		"python", "code/export_bids.py",
		"--bids-dir", "/local/bids",
		"--api-key", apiKey,
		"-p", projectLabel,
	}

	if sourceData {
		cmd = append(cmd, "--source-data")
	}

	status, err := runBidsCmdInContainer(docker, []string{binding}, cmd)
	if err != nil {
		// Intentionally obtuse error message, ideally we would hide that we're
		// calling into a container
		Fprintln(os.Stderr, "Error exporting BIDS data --", err.Error())
		os.Exit(1)
	} else {
		if status != 0 {
			os.Exit(int(status))
		}
	}
}

func runBidsCmdInContainer(docker *client.Client, bindings []string, cmd []string) (int64, error) {
	imageName := Sprintf("%s:%s", BidsContainerName, BidsContainerVersion)

	ctx := context.Background()

	// Pull the image every time
	out, err := docker.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return -1, err
	}

	// Make sure that we've read the full image
	defer out.Close()
	Fprintln(os.Stdout, "Preparing...")
	if _, err := ioutil.ReadAll(out); err != nil {
		return -1, err
	}

	// Capture Stdout and Stderr since we're going to pipe it
	containerCfg := container.Config{
		Image:        imageName,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostCfg := container.HostConfig{
		Binds:       bindings,
		NetworkMode: "host",
	}

	// Create and start the container instance
	resp, err := docker.ContainerCreate(ctx, &containerCfg, &hostCfg, nil, "")
	if err != nil {
		return -1, err
	}

	err = docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return -1, err
	}

	// Stream stdout / stderr from container
	go func() {
		reader, err := docker.ContainerLogs(context.Background(), resp.ID, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: false,
		})
		if err != nil {
			panic(err)
		}
		defer reader.Close()

		stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	}()

	// Wait for a final result, and return the status code
	statusCh, errCh := docker.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, err
		}
	case waitOk := <-statusCh:
		return waitOk.StatusCode, nil
	}

	return 0, nil
}
