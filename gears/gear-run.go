package gears

import (
	"bufio"
	"encoding/json"
	. "fmt"
	// "io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"github.com/spf13/cobra"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func GearRun(client *api.Client, docker *client.Client, args []string) {
	// New command template in the style of commands_linux.go
	dummyCmd := &cobra.Command{}
	defaultTemplate := dummyCmd.UsageTemplate()
	gearCmdTemplate := strings.Replace(defaultTemplate, ".LocalFlags.FlagUsages | trimRightSpace", ".LocalFlags.FlagUsages | trimRightSpace | trimStringLiterals", 1)
	trimStringLiterals := func(s string) string {
		removeStringLiteral := regexp.MustCompile(`^( +\-\-.*?) string(.*)$`)

		parts := strings.Split(s, "\n")
		for x, part := range parts {
			parts[x] = removeStringLiteral.ReplaceAllString(part, "${1}${2}")
		}

		return strings.Join(parts, "\n")
	}
	cobra.AddTemplateFunc("trimStringLiterals", trimStringLiterals)
	_ = gearCmdTemplate

	// cmd storage
	var configs interface{}
	var values interface{}
	var image string

	// cmd base
	jobRunActual := &cobra.Command{
		Use:   "run",
		Short: "Start a job.",
		Run: func(cmd *cobra.Command, args []string) {
			// targets := []*api.ContainerReference{}

			// Merge value slice with config slice for convenience
			configsCast := configs.([]*legacy.GearConfig)
			for x := range configsCast {
				configsCast[x].Value = values.([]string)[x]
			}

			// construct map from flags
			config := map[string]interface{}{
				"config": legacy.GenConfigStruct(configsCast),
			}
			inputs := legacy.GenInputs(configsCast)

			GearRunActual(client, docker, image, config, inputs)
		},
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
	image = gearBuilderConfig["image"].(string)
	// WOT

	// cmd flag add
	cs := legacy.GenGearConfigs(gear)
	configs = cs
	values = make([]string, len(cs))

	for index, config := range cs {
		jobRunActual.Flags().StringVar(&values.([]string)[index], config.Name, "", config.Description)
	}

	jobRunActual.SetArgs(args)
	jobRunActual.Execute()
}

func GearRunActual(client *api.Client, docker *client.Client, image string, config map[string]interface{}, inputs map[string]string) {

	cwd, err := os.Getwd()
	Check(err)

	baseTempDir := ""
	baseTempPrefix := "fw-gear-builder-"

	switch runtime.GOOS {
	case "darwin":
		// Default tempdir goes to, like, /var/folders/fp/znt3073d3313h1r_sbj9292h0000gn/T/fw-gear-builder594038416
		// At which point docker throws up its hands and cries for no discernable reason

		baseTempDir = "/tmp/"
	case "linux":
		break
	default:
		Println("Warning: Gear builder is not yet tested on platform", runtime.GOOS)
		Println()
	}

	tmpfile, err := ioutil.TempFile(baseTempDir, baseTempPrefix)
	Check(err)
	defer os.Remove(tmpfile.Name())

	raw, err := json.MarshalIndent(config, "", "\t")
	Check(err)
	err = ioutil.WriteFile(tmpfile.Name(), raw, 0644)
	Check(err)

	// pullProgress, err := docker.ImagePull(background, image, types.ImagePullOptions{})
	// Check(err)
	// io.Copy(ioutil.Discard, pullProgress)
	// pullProgress.Close()

	mounts := []mount.Mount{}

	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   cwd,
		Target:   "/flywheel/v0",
		ReadOnly: false,
	})
	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   tmpfile.Name(),
		Target:   "/flywheel/v0/config.json",
		ReadOnly: false,
	})

	// Prevent input folder from getting stuff in it
	tempdir, err := ioutil.TempDir(baseTempDir, baseTempPrefix)
	Check(err)
	defer func(dir string) { os.RemoveAll(dir) }(tempdir)

	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   tempdir,
		Target:   "/flywheel/v0/input",
		ReadOnly: false,
	})

	// Mount input files
	for name, input := range inputs {
		_, err := os.Stat(input)
		Check(err)

		path, err := filepath.Abs(input)
		Check(err)

		tempdir, err := ioutil.TempDir(baseTempDir, baseTempPrefix)
		Check(err)
		defer func(dir string) { os.RemoveAll(dir) }(tempdir)

		mounts = append(mounts, mount.Mount{
			Type:     "bind",
			Source:   tempdir,
			Target:   "/flywheel/v0/input/" + name,
			ReadOnly: false,
		})

		mounts = append(mounts, mount.Mount{
			Type:     "bind",
			Source:   path,
			Target:   "/flywheel/v0/input/" + name + "/" + filepath.Base(input),
			ReadOnly: true,
		})
	}

	containerId, cleanup, err := CreateContainerWithCleanup(docker, background,
		&container.Config{
			Image:      image,
			WorkingDir: "/flywheel/v0",
			Entrypoint: []string{},
			Cmd:        []string{"bash", "-c", "rm -rf output; mkdir -p output; ./run"},
			Env:        []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
		},
		&container.HostConfig{
			Mounts: mounts,
		}, "")
	Check(err)
	defer cleanup()

	docker.ContainerStart(background, containerId, types.ContainerStartOptions{})
	Check(err)

	out, err := docker.ContainerLogs(background, containerId, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	defer out.Close()
	Check(err)

	// Docker logs streams contain an 8-byte header. For now, ignore it.
	// Technically, this read could return less than 8 bytes. Story for another day.
	//
	// Ref:
	// https://docs.docker.com/engine/api/v1.30/#operation/ContainerAttach
	header := make([]byte, 8)
	_, err = out.Read(header)
	Check(err)

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		Println(stripCtlFromBytes(scanner.Text()))
	}
	Check(scanner.Err())

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

	// Remove input folder, ignoring errors. Quirk of mounting
	os.Remove("input")
	os.Remove("config.json")
}
