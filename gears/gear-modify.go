package gears

import (
	"encoding/json"
	. "fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/client"

	. "flywheel.io/fw/util"
)

// Dupe from sdk for convenience
func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var hexRunes = []rune("0123456789abcdef")

func RandStringOfLength(n int, runes []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
func RandString() string {
	return RandStringOfLength(10, letterRunes)
}
func RandStringLower() string {
	return strings.ToLower(RandStringOfLength(10, letterRunes))
}

//

func GearModify(docker *client.Client, quiet bool) {
	cwd, err := os.Getwd()
	Check(err)

	// Dupe from gear-run; map-struct deserialize and state-map could be of use here
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
	//
	// And some additions...
	containerMaybe, containerDefined := gearBuilderConfig["container"]
	containerName := ""
	if containerDefined {
		containerName = containerMaybe.(string)
	}
	//

	if !quiet {
		Println()
		Println("This prompt will place you inside your gear! Any changes you make here will be persisted for future runs and when you upload the final product.")
		Println()
		Println("To exit this shell, use `exit` or press Control-D.")
		Println()
		Println()
	}

	// TTYs are complex: http://www.linusakesson.net/programming/tty
	// We can't trivially fake one out; this is more complex than merely bouncing a stream vis a vis gear-run.
	// (And even that's not escaped 100% correctly at present.)
	// An initial, naive attempt immediately failed due to resize events that should have been obvious in retrospect.
	//
	// Our options:
	//
	// 1) Reverse-engineer and re-implement the docker CLI's entire run/attach/exec apparatus
	//
	// 2) Fitness-eval, import & attempt to use a library from 2015 that claims to solve this very problem
	//        https://github.com/fgrehm/go-dockerpty
	//
	//        Notably, both resizes and direct links to Docker's source code are in this library
	//        Basically, the author took our option #1 and ran with it
	//        ~400 LOC, not too bad. Might be a good project to adapt / update?
	//
	//        For bonus points, check out "import syscall" and convince me this will work on all platforms.
	//
	// 3) Exec-wrap.
	//
	// #3 is the only option that won't take longer than allowed and won't immediately fail in some scenario that we haven't foreseen.
	// A healthy invocation of `docker ps` is a prerequisite for the gear builder, so this shouldn't pose a problem.

	cmd := exec.Command("docker", "run", "-it", "-v", cwd+":/flywheel/v0", "--entrypoint", "", "--cidfile", ".containerId", image, "bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	Check(err)
	err = cmd.Wait()
	Check(err)

	containerNameBytes, err := ioutil.ReadFile(".containerId")
	Check(err)
	containerName = string(containerNameBytes)
	os.Remove(".containerId")

	// Docker's aggressive, intentional inflexibility strikes again:
	// Reconfiguring a container is not well-supported, but our use case demands it.
	// Gear-run dynamically determines mounts at runtime, by design.
	//
	// One of these things is not like the other:
	//     https://godoc.org/github.com/moby/moby/api/types/container#Config
	//     https://godoc.org/github.com/moby/moby/api/types#ContainerStartOptions
	//
	// So, we need to commit an image on gear-modify exit.
	// This will lead to `docker image` spam, if the user is so-inclined to interact with the docker CLI.
	// GC is also not an option; we don't have a graph to mark or sweep.
	//
	// Cleanup:
	// docker rmi -f $(docker images --filter=reference='gear-builder*' -q)
	// This is no better or worse than standard Docker usage.
	//
	// The best workaround I could find involves jumping into docker's filesystem and editing the stored JSON.
	// This is insane, unsupported, and probably won't work on platforms that hide away behind a hypervisor.
	//
	// As an upside, it will at least be a bit clearer how to share images with another daemon.

	stamp := time.Now().Format("20060102150405")
	label := "gear-builder-" + RandStringLower() + "-" + stamp

	cmd = exec.Command("docker", "commit", "-m", "Gear produced from `fw gear modify`", containerName, label)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	Check(err)
	err = cmd.Wait()
	Check(err)

	// Save changes
	gear.Custom["gear-builder"].(map[string]interface{})["image"] = label
	gear.Custom["gear-builder"].(map[string]interface{})["container"] = containerName
	// Dupe, state-map
	raw, err := json.MarshalIndent(gear, "", "\t")
	Check(err)
	err = ioutil.WriteFile("manifest.json", raw, 0644)
	Check(err)
}
