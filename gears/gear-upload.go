package gears

import (
	"bufio"
	"encoding/json"
	. "fmt"
	"io"
	// "io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	prompt "github.com/segmentio/go-prompt"

	"github.com/cheggaaa/pb"
	"github.com/klauspost/pgzip"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearUpload(client *api.Client, docker *client.Client, category string) {
	cwd, err := os.Getwd()
	Check(err)

	gearCategory := api.GearCategory(category)

	if gearCategory != api.Converter && gearCategory != api.Analysis {
		Println("Invalid gear category. Check `fw gear upload -h` for options.")
		os.Exit(1)
	}

	// WOT
	gear := TryToLoadManifest()

	if gear == nil {
		Println("No gear found! Try `fw gear create` first.")
		os.Exit(1)
	}

	if gear.Custom == nil || gear.Custom["gear-builder"] == nil || gear.Custom["gear-builder"].(map[string]interface{})["image"] == nil {
		Println("The gear manifest in this folder does not have the gear-builder information it needs.")
		Println("Try `fw gear create` first.")
		os.Exit(1)
	}

	gearBuilderConfig := gear.Custom["gear-builder"].(map[string]interface{})
	image := gearBuilderConfig["image"].(string)
	// WOT

	_, err = os.Stat("output")
	if err == nil {
		Println("Output folder exists and will be deleted as part of the upload process.")
		proceed := prompt.Confirm("Continue? (yes/no)")
		Println()
		if !proceed {
			Println("Canceled.")
			os.Exit(1)
		}

		Check(os.RemoveAll("output"))
	}

	// Println("Checking that docker image is available...")

	// pullProgress, err := docker.ImagePull(background, image, types.ImagePullOptions{})
	// Check(err)
	// io.Copy(ioutil.Discard, pullProgress)
	// pullProgress.Close()

	Println("Uploading gear to Flywheel...")

	containerId, cleanup, err := CreateContainerWithCleanup(docker, background,
		&container.Config{
			Image:      image,
			WorkingDir: "/flywheel/v0",
			// oh boy
			Entrypoint: []string{},
			Cmd:        []string{"bash", "-c", "shopt -s dotglob && rm -rf /Flywheel && mkdir -p /flywheel/v0 && cp /tmp/flywheel-copy-target/* /flywheel/v0/"},
			Env:        []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:     "bind",
					Source:   cwd,
					Target:   "/tmp/flywheel-copy-target",
					ReadOnly: true,
				},
			},
		}, "")
	Check(err)
	defer cleanup()

	docker.ContainerStart(background, containerId, types.ContainerStartOptions{})
	Check(err)

	statusChan, errorChan := docker.ContainerWait(background, containerId, container.WaitConditionNotRunning)

	status := <-statusChan
	// I guess the error channel of ContainerWait is completely untrustworthy
	select {
	case err = <-errorChan:
		Println(err)
	default:
		// Println("Err chan is useless")
	}

	if status.StatusCode != 0 {
		Println("Exit code was", status.StatusCode)
	}
	Check(err)

	out, err := docker.ContainerLogs(background, containerId, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	Check(err)

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		Println(stripCtlFromBytes(scanner.Text()))
	}
	Check(scanner.Err())

	stream, err := docker.ContainerExport(background, containerId)
	Check(err)

	// Stream output through concurrent gzip
	// Slightly unintuitive because of the mixed reader/writer conventions.
	// Flow: stream --> gzW --> pw --> matched to pr by io.Pipe --> member of UploadSource struct --> consumed by client.UploadSimple

	pr, pw := io.Pipe()
	gzW := pgzip.NewWriter(pw)
	go func() {
		io.Copy(gzW, stream)
		gzW.Close()
		pw.Close()
	}()

	// n, err := io.Copy(dest, stream)
	// Println(n)
	// Check(err)
	// Check(dest.Close())

	now := time.Now()
	doc := &api.GearDoc{
		// Id:
		Category: gearCategory,
		Gear:     gear,
		// Source:
		Created:  &now,
		Modified: &now,
	}

	raw, err := json.MarshalIndent(doc, "", "\t")
	Check(err)
	_ = raw

	gearUpload := &api.UploadSource{Reader: pr, Name: "gear.tar.gz"}
	progress, errChan := client.UploadSimple("/api/gears/temp", raw, gearUpload)

	bar := pb.New(1000000000000)
	bar.ShowPercent = false
	bar.ShowCounters = false
	bar.ShowTimeLeft = false
	bar.ShowBar = false
	bar.ShowSpeed = true
	bar.ShowFinalTime = true
	bar.SetUnits(pb.U_BYTES)
	bar.Start()

	total := int64(0)
	for x := range progress {
		bar.Add64(x - total)
		total = x
	}

	bar.Finish()
	Println("Waiting for server...")

	err = <-errChan
	Check(err)

	Println()
	Println()
	Println("Done! You should now see your gear in the Flywheel web interface,")
	Println("or find it with `fw job list-gears`.")
}
