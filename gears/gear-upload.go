package gears

import (
	"bufio"
	"encoding/json"
	"errors"
	. "fmt"
	"io"
	// "io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	prompt "github.com/segmentio/go-prompt"

	"github.com/cheggaaa/pb"
	"github.com/klauspost/pgzip"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearUpload(client *api.Client, docker *client.Client, category, filepath, project string) {
	cwd, err := os.Getwd()
	Check(err)

	gearCategory := api.GearCategory(category)

	if gearCategory != api.ConverterGear && gearCategory != api.AnalysisGear {
		Fprintln(os.Stderr, "Invalid gear category. Check `fw gear upload -h` for options.")
		os.Exit(1)
	}

	// WOT
	gear := TryToLoadManifest()

	if gear == nil {
		Fprintln(os.Stderr, "No gear found! Try `fw gear create` first.")
		os.Exit(1)
	}

	if gear.Custom == nil || gear.Custom["gear-builder"] == nil || gear.Custom["gear-builder"].(map[string]interface{})["image"] == nil {
		Fprintln(os.Stderr, "The gear manifest in this folder does not have the gear-builder information it needs.")
		Fprintln(os.Stderr, "Try `fw gear create` first.")
		os.Exit(1)
	}

	gearBuilderConfig := gear.Custom["gear-builder"].(map[string]interface{})
	image := gearBuilderConfig["image"].(string)
	// WOT

	_, err = os.Stat("output")
	if err == nil {
		Fprintln(os.Stderr, "Output folder exists and will be deleted as part of the upload process.")
		proceed := prompt.Confirm("Continue? (yes/no)")
		Fprintln(os.Stderr)
		if !proceed {
			Fprintln(os.Stderr, "Canceled.")
			os.Exit(1)
		}

		Check(os.RemoveAll("output"))
	}
	projectId := ""

	if project != "" {
		parts := strings.Split(project, "/")

		result, _, err, aerr := legacy.ResolvePath(client, parts)
		Check(api.Coalesce(err, aerr))

		path := result.Path
		target := path[len(path)-1]

		switch target := target.(type) {
		case *legacy.Project:
			projectId = target.Id
		default:
			Check(errors.New("The project path must be a project, not any other container type."))
		}
	}

	now := time.Now()
	doc := &api.GearDoc{
		// Id:
		Category: gearCategory,
		Gear:     gear,
		// Source:
		Created:  &now,
		Modified: &now,
	}

	if projectId != "" {
		doc.Projects = []string{projectId}
	}

	if filepath == "" {
		Fprintln(os.Stderr, "Checking that gear is ready to upload...")
		// gear-check added here rather than SDK, for now.
		var aerr *api.Error
		_, err = client.New().Post("gears/check").BodyJSON(doc).Receive(nil, &aerr)
		Check(api.Coalesce(err, aerr))

		Fprintln(os.Stderr, "Uploading gear to Flywheel...")
	}

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
		Fprintln(os.Stderr, err)
	default:
		// Fprintln(os.Stderr, "Err chan is useless")
	}

	if status.StatusCode != 0 {
		Fprintln(os.Stderr, "Exit code was", status.StatusCode)
	}
	Check(err)

	out, err := docker.ContainerLogs(background, containerId, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	Check(err)

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		Fprintln(os.Stderr, stripCtlFromBytes(scanner.Text()))
	}
	Check(scanner.Err())

	docRaw, err := json.MarshalIndent(doc, "", "\t")
	Check(err)

	// Closure to hide streams you should not read from
	pr := func() *io.PipeReader {
		stream, err := docker.ContainerExport(background, containerId)
		Check(err)

		// Stream output through concurrent gzip
		// Slightly unintuitive because of the mixed reader/writer conventions.
		// Flow: stream --> gzW --> pw --> matched to pr by io.Pipe --> consumer by io.Copy or UploadSource
		//   if UploadSource:
		//     member of UploadSource struct --> consumed by client.UploadSimple

		pr, pw := io.Pipe()
		gzW := pgzip.NewWriter(pw)
		go func() {
			io.Copy(gzW, stream)
			gzW.Close()
			pw.Close()
		}()

		return pr
	}()

	if filepath == "--" {
		_, err := io.Copy(os.Stdout, pr)
		Check(err)
	} else if filepath != "" {
		file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0755)
		Check(err)

		_, err = io.Copy(file, pr)
		Check(err)
	} else {
		gearUpload := &api.UploadSource{Reader: pr, Name: "gear.tar.gz"}
		progress, errChan := client.UploadSimple("/api/gears/temp", docRaw, gearUpload)

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
		Fprintln(os.Stderr, "Waiting for server...")

		err = <-errChan
		Check(err)

		Fprintln(os.Stderr)
		Fprintln(os.Stderr)
		Fprintln(os.Stderr, "Done! You should now see your gear in the Flywheel web interface,")
		Fprintln(os.Stderr, "or find it with `fw job list-gears`.")
	}
}
