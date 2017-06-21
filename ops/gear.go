package ops

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	. "fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	prompt "github.com/segmentio/go-prompt"
	"github.com/spf13/cobra"

	"github.com/cheggaaa/pb"

	"flywheel.io/fw/legacy"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func UntarGearFolder(reader io.Reader) error {

	var buffer bytes.Buffer

	_, err := io.Copy(&buffer, reader)
	if err != nil {
		return err
	}

	tr := tar.NewReader(&buffer)

	for {
		header, err := tr.Next()

		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		} else if header == nil {
			continue
		}

		// Ignore v0 folder
		header.Name = strings.TrimLeft(header.Name, "v0/")
		header.Name = strings.TrimRight(header.Name, "/")

		if header.Name == "" || header.Name == "input" || header.Name == "output" {
			continue
		}

		switch header.Typeflag {

		case tar.TypeDir:
			_, err := os.Stat(header.Name)

			if err != nil {
				err := os.MkdirAll(header.Name, 0755)

				if err != nil {
					return err
				}
			}

		case tar.TypeReg:

			// Ask user before deleting any existing files
			_, err := os.Stat(header.Name)
			if err == nil {
				Println("\nFile \"" + header.Name + "\" already exists in this folder and in the gear.")
				proceed := prompt.Confirm("Replace local file? (yes/no)")
				if !proceed {
					continue
				}
			}

			f, err := os.OpenFile(header.Name, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, tr)
			if err != nil {
				return err
			}

		default:
			Println("Ignoring nonregular file from gear:", header.Name)
		}
	}
}

func TryToLoadManifest() *api.Gear {
	var gear api.Gear

	plan, err := ioutil.ReadFile("manifest.json")
	if err == nil {
		err = json.Unmarshal(plan, &gear)
	}

	if err != nil {
		return nil
	} else {
		return &gear
	}
}

func ManifestOrDefaultGear() *api.Gear {
	gear := TryToLoadManifest()

	if gear != nil {
		return gear
	} else {
		return &api.Gear{
			Name:        "empty-gear",
			Label:       "Empty Gear",
			Description: "An empty gear manifest. Fill out this file!",
			Version:     "0",
			Author:      "You!",
			Maintainer:  "You!",
			License:     "Other",
			Source:      "http://example.example",
			Url:         "http://example.example",
		}
	}
}

func PromptOrDefault(promptS, defaultP string) string {
	result := prompt.String(promptS + " [leave blank for " + defaultP + "]")

	if result == "" {
		return defaultP
	} else {
		return result
	}
}

func GearCreate(client *api.Client, docker *client.Client) {

	user, _, err := client.GetCurrentUser()
	Check(err)

	Println("Welcome to gear creation! Let's get started.")
	Println()

	Println()
	gearLabel := PromptOrDefault("What will be the name of your gear?", "My Gear")
	gearName := strings.ToLower(gearLabel)
	gearName = strings.Replace(gearName, " ", "-", -1)

	// Println()
	// gearDescription := prompt.String("If you like, describe your gear in a few words")
	gearDescription := "Gear created with gear builder"

	Println()

	choices := []string{"Just linux (bash)", "Custom - use a docker image"}
	results := []string{"flywheel/example-gear", ""}

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
	}

	Println()
	author := PromptOrDefault("Who is the author of this gear?", user.Firstname+" "+user.Lastname)

	Println()
	source := PromptOrDefault("Is there a website users can go to learn more?", "http://example.example")

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

	Println("Reading gear contents...")
	reader, stat, err := docker.CopyFromContainer(background, containerId, gearPath)

	if err != nil && strings.HasSuffix(err.Error(), "no such file or directory") {
		Println()
		Println("This docker image does not appear to be a Flywheel Gear; the /flywheel/v0 folder is missing.")
		Println("Support for creating gears from non-gear images is not currently available.")
		os.Exit(1)
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
	gear.Source = source
	gear.Url = source
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
			config := legacy.GenConfigStruct(configsCast)
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

	tmpfile, err := ioutil.TempFile("", "fw-gear-builder")
	Check(err)
	defer os.Remove(tmpfile.Name())

	raw, err := json.MarshalIndent(config, "", "\t")
	Check(err)
	err = ioutil.WriteFile(tmpfile.Name(), raw, 0644)
	Check(err)

	pullProgress, err := docker.ImagePull(background, image, types.ImagePullOptions{})
	Check(err)
	io.Copy(ioutil.Discard, pullProgress)
	pullProgress.Close()

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
	tempdir, err := ioutil.TempDir("", "fw-gear-builder")
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

		tempdir, err := ioutil.TempDir("", "fw-gear-builder")
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

	// Remove input folder, ignoring errors. Quirk of mounting
	os.Remove("input")
	os.Remove("config.json")
}

func stripCtlFromBytes(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= 32 && c != 127 {
			b[bl] = c
			bl++
		}
	}
	return string(b[:bl])
}

func GearUpload(client *api.Client, docker *client.Client) {
	cwd, err := os.Getwd()
	Check(err)

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

	Println("Checking that docker image is available...")

	pullProgress, err := docker.ImagePull(background, image, types.ImagePullOptions{})
	Check(err)
	io.Copy(ioutil.Discard, pullProgress)
	pullProgress.Close()

	Println("Uploading gear to Flywheel...")

	containerId, cleanup, err := CreateContainerWithCleanup(docker, background,
		&container.Config{
			Image:      image,
			WorkingDir: "/flywheel/v0",
			// oh boy
			Cmd: []string{"bash", "-c", "shopt -s dotglob && rm -rf /Flywheel && mkdir -p /flywheel/v0 && cp /tmp/flywheel-copy-target/* /flywheel/v0/"},
			Env: []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
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

	// dest, err := os.Create("result.tar")
	// Check(err)
	stream, err := docker.ContainerExport(background, containerId)
	Check(err)

	// n, err := io.Copy(dest, stream)
	// Println(n)
	// Check(err)
	// Check(dest.Close())

	now := time.Now()
	doc := &api.GearDoc{
		// Id:
		Category: "converter",
		Gear:     gear,
		// Source:
		Created:  &now,
		Modified: &now,
	}

	raw, err := json.MarshalIndent(doc, "", "\t")
	Check(err)
	_ = raw

	gearUpload := &api.UploadSource{Reader: stream, Name: "gear.tar"}
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

	for x := range progress {
		bar.Add64(x)
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
