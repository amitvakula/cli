package gears

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/net/context"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

var ctx = context.Background()

// Clean up function params with embedded struct
type D struct {
	*client.Client
}

func (docker *D) IsImageLocal(imageName string) bool {

	if !strings.Contains(imageName, ":") && !strings.Contains(imageName, "@") {
		Println("Assuming 'latest' as the image tag.")
		imageName += ":latest"
	}

	Println("Checking if", imageName, "is available locally...")

	images, err := docker.ImageList(ctx, types.ImageListOptions{All: true})
	Check(err)

	for _, image := range images {
		for _, digest := range image.RepoDigests {
			if digest == imageName {
				Println("\tFound digest locally.")
				return true
			}
		}
		for _, tag := range image.RepoTags {
			if tag == imageName {
				Println("\tFound tag locally.")
				return true
			}
		}
	}

	Println("Image is not installed locally.")
	return false
}

func (docker *D) PullImage(imageName string) {
	Println("Pulling", imageName, "from registry...")

	stream, err := docker.ImagePull(ctx, imageName, types.ImagePullOptions{})
	Check(err)

	auxCallback := func(j jsonmessage.JSONMessage) {
		// PrintFormat(j)
	}

	err = jsonmessage.DisplayJSONMessagesStream(stream, os.Stderr, 0, true, auxCallback)
	Check(err)
	stream.Close()

	Println("\tImage downloaded.")
}

func (docker *D) EnsureImageLocal(imageName string) {
	if !docker.IsImageLocal(imageName) {
		docker.PullImage(imageName)
	}
}

// Ensure image is local first
func (docker *D) CreateContainer(imageName string) (string, func()) {
	return docker.CreateContainerFromImage(imageName, nil, nil)
}

// Ensure image is local first
func (docker *D) CreateContainerFromImage(imageName string, config *container.Config, hostConfig *container.HostConfig) (string, func()) {
	Println("Creating container from", imageName, "...")

	if config == nil {
		config = &container.Config{}
	}
	config.Image = imageName

	body, err := docker.ContainerCreate(ctx, config, hostConfig, nil, "")
	Check(err)

	for _, warning := range body.Warnings {
		Println(warning)
	}

	containerId := body.ID
	Println("\tCreated", containerId[:12])

	cleanup := func() {
		Println("Removing container", containerId[:12], "...")
		err := docker.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{
			// RemoveVolumes: true,
			// RemoveLinks: true,
			Force: true,
		})
		Check(err)
		Println("\tRemoved container")
	}

	return containerId, cleanup
}

// Consistent way to launch a local gear.
// Ensure image is local first
func (docker *D) CreateContainerForGear(imageName string, env map[string]string, command []string, mounts []mount.Mount) (string, func()) {

	return docker.CreateContainerFromImage(imageName, &container.Config{
		WorkingDir: GearPath,
		Entrypoint: []string{}, // prevent upstream image from interfering
		Cmd:        command,
		Env:        TranslateEnvToEnvArray(env),
	},
		&container.HostConfig{
			Mounts: mounts,
		})
}

func (docker *D) InspectImage(imageName string) *types.ImageInspect {
	Println("Inspecting", imageName, "...")
	details, _, err := docker.ImageInspectWithRaw(ctx, imageName)
	Check(err)

	if details.Os != "linux" {
		FatalWithMessage("Unrecognized OS", details.Os, "must be linux")
	}

	if details.Architecture != "amd64" {
		FatalWithMessage("Unrecognized architecture", details.Architecture, "must be amd64")
	}

	Println("\tImage is the correct architecture.")

	return &details
}

// Return details with info parsed into gear format
func (docker *D) GetImageDetails(imageName string) (*types.ImageInspect, map[string]string) {
	details := docker.InspectImage(imageName)

	env := map[string]string{}

	if details.Config != nil {
		env = TranslateEnvArrayToEnv(details.Config.Env)
	} else {
		Println("Warning: image", imageName, "does not have a configuration. Try `docker inspect", imageName+"` for more info")
	}

	return details, env
}

// Start and watch logs, get retcode
func (docker *D) ObserveContainer(containerID string) int64 {
	err := docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	Check(err)

	Println("Attaching to logs...")
	Println()

	stream, err := docker.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	defer stream.Close()
	Check(err)

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, stream)
	Check(err)

	return docker.WaitForContainerResults(containerID)
}

// Wait for a retcode without logging
func (docker *D) WaitOnContainer(containerID string) int64 {
	err := docker.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	Check(err)
	return docker.WaitForContainerResults(containerID)
}

// Wait for container return code
func (docker *D) WaitForContainerResults(containerID string) int64 {
	statusChan, errorChan := docker.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	// Println("Waiting on status...")
	status := <-statusChan

	if status.Error != nil && status.Error.Message != "" {
		Println("Error while waiting for container to exit: " + status.Error.Message)
	}

	// Non-blocking select; errorChan only sends on error
	select {
	case err := <-errorChan:
		Check(err)
	default:
	}

	return status.StatusCode
}

// Generate auth token
func (docker *D) LoginToRegistry(client *api.Client, domain, apiKey string) string {
	Println("Logging into Flywheel gear registry...")
	user, _, err := client.GetCurrentUser()

	authConfig := types.AuthConfig{
		ServerAddress: domain,
		Username:      user.Email,
		Password:      apiKey,
	}
	Check(err)

	_, err = docker.RegistryLogin(ctx, authConfig)
	Check(err)

	authJson, _ := json.Marshal(authConfig)
	authEncode := base64.URLEncoding.EncodeToString(authJson)

	Println("\tLogin successful.")
	return authEncode
}

func (docker *D) TagImage(src, dst string) {
	Println("Tagging the image as", dst, "locally...")
	err := docker.ImageTag(ctx, src, dst)
	Check(err)
}

func (docker *D) TagContainer(src, dst string) {
	Println("Tagging the container as", dst, "locally...")

	_, err := docker.ContainerCommit(ctx, src, types.ContainerCommitOptions{
		Reference: dst,
	})
	Check(err)
}

func (docker *D) PushImage(imageDst, token string) string {
	Println("Pushing", imageDst, "to registry...")
	readcloser, err := docker.ImagePush(ctx, imageDst, types.ImagePushOptions{RegistryAuth: token})
	Check(err)

	digest := ""
	auxCallback := func(j jsonmessage.JSONMessage) {
		var raw interface{}
		err := json.Unmarshal(*j.Aux, &raw)

		if err == nil {
			decode := raw.(map[string]interface{})
			// PrintFormat(decode)

			if d, ok := decode["Digest"].(string); ok {
				digest = d
			}
		}
	}

	err = jsonmessage.DisplayJSONMessagesStream(readcloser, os.Stderr, 0, true, auxCallback)
	Check(err)
	readcloser.Close()

	if digest == "" {
		FatalWithMessage("Pushed image but found no digest; contact support.")
	}

	Println("\tImage uploaded.")
	return digest
}

func (docker *D) ExpandFlywheelFolder(imageName, containerId string, defaultManifest *api.Gear, modifier func(*api.Gear) *api.Gear) {
	Println("Checking if image has gear contents...")
	reader, stat, err := docker.CopyFromContainer(ctx, containerId, GearPath)

	// If there is no gear path, or the path is not a directory, use example content
	if (err != nil && strings.Contains(err.Error(), "No such")) || (err == nil && !stat.Mode.IsDir()) {
		Println("\tNo gear contents. Providing a starter kit...")

		python := docker.PythonInstalled(imageName)

		// Write a nice script based on installed language
		if python {
			defaultManifest.Command = "./example.py"

			// Write example run script
			_, err = os.Stat("example.py")
			if err == nil {
				runConfirmFatal(confirmReplaceScriptPmtP)
			}
			err = ioutil.WriteFile("example.py", []byte(ExamplePythonScript), 0750)
			Check(err)
		} else {
			// Write example run script
			_, err = os.Stat("example.sh")
			if err == nil {
				runConfirmFatal(confirmReplaceScriptPmt)
			}
			err = ioutil.WriteFile("example.sh", []byte(ExampleRunScript), 0750)
			Check(err)
		}

		// Write manifest
		_, err := os.Stat(ManifestName)
		if err == nil {
			runConfirmFatal(confirmReplaceManifestPmt)
		}
		err = ioutil.WriteFile(ManifestName, FormatBytes(defaultManifest), 0640)
		Check(err)

	} else {
		Check(err)
		Println("\tExpanding gear contents...")
		err = UntarGearFolder(reader)

		// Can be nil
		localManifest := TryToLoadCWDManifest()

		if localManifest != nil {
			Println("Modifying local manifest...")
			localManifest = modifier(localManifest)

		} else {
			// Should not happen to sane images since we checked for a gear folder already
			Println("Warning: no", ManifestName, "was present from the image despite having a ", GearPath, "folder. Generating a default one.")
			localManifest = defaultManifest
		}

		err = ioutil.WriteFile(ManifestName, FormatBytes(localManifest), 0640)
		Check(err)
	}
}

func (docker *D) SaveCwdIntoContainer(containerID string) {
	Println("Packing up current folder...")
	errChan := make(chan error)
	reader, writer := io.Pipe()

	go func() {

		err := TarCWD(writer)
		writer.Close()
		errChan <- err
	}()

	err := docker.CopyToContainer(ctx, containerID, "/", reader, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	Check(err)
	Check(<-errChan)
	Println("\t Finished packing")
}

// Lay out temporary files/directorys and set up a list of mounts needed for a local gear run.
func PrepareLocalRunFiles(gear *api.Gear, invocation map[string]interface{}, files map[string]string) ([]mount.Mount, func()) {
	cwd, err := os.Getwd()
	Check(err)

	tempdirs := []string{}

	// Write a config.json file
	configFile, err := ioutil.TempFile(BaseTempDir, BaseTempPrefix)
	Check(err)
	err = ioutil.WriteFile(configFile.Name(), FormatBytes(invocation), 0644)
	Check(err)

	// path.Join used here for targets, not filepath.Join; containers always have forward slashes
	mounts := []mount.Mount{}

	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   cwd,
		Target:   GearPath,
		ReadOnly: false,
	})
	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   configFile.Name(),
		Target:   path.Join(GearPath, ConfigName),
		ReadOnly: false,
	})

	// Prevent input folder from getting stuff in it
	baseInputFolder, err := ioutil.TempDir(BaseTempDir, BaseTempPrefix)
	Check(err)
	tempdirs = append(tempdirs, baseInputFolder)

	mounts = append(mounts, mount.Mount{
		Type:     "bind",
		Source:   baseInputFolder,
		Target:   "/flywheel/v0/input",
		ReadOnly: false,
	})

	// Mount input files
	for name, path := range files {
		// Println("Processing", name, path)
		_, err := os.Stat(path)
		Check(err)

		path, err := filepath.Abs(path)
		Check(err)

		tempdir, err := ioutil.TempDir(BaseTempDir, BaseTempPrefix)
		Check(err)
		tempdirs = append(tempdirs, tempdir)

		mounts = append(mounts, mount.Mount{
			Type:     "bind",
			Source:   tempdir,
			Target:   "/flywheel/v0/input/" + name,
			ReadOnly: false,
		})

		mounts = append(mounts, mount.Mount{
			Type:     "bind",
			Source:   path,
			Target:   "/flywheel/v0/input/" + name + "/" + filepath.Base(path),
			ReadOnly: true,
		})
	}

	// Clean output dir, similar to engine
	outputDir := filepath.Join(cwd, "output")
	Check(os.RemoveAll(outputDir))
	Check(os.Mkdir(outputDir, 0700))

	cleanup := func() {
		Check(os.Remove(configFile.Name()))

		for _, folder := range tempdirs {
			Check(os.RemoveAll(folder))
		}

		// Silent delete attempts due to mounting:

		os.Remove(path.Join(cwd, "input"))
		os.Remove(path.Join(cwd, "output"))

		cfg := path.Join(cwd, ConfigName)
		stat, err := os.Stat(cfg)
		if err == nil && stat.Size() == 0 {
			os.Remove(cfg)
		}

	}

	return mounts, cleanup
}
