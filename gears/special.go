package gears

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/flosch/pongo2"

	. "flywheel.io/fw/util"
)

// Register flywheel functionality as template filters.
// At a glance, pongo2 only supports a global registry. Should fix that.
func init() {
	Check(pongo2.RegisterFilter("flag", templateFlag))
}

// The flag filter toggles some text based on a boolean.
// Yes, this is a glorified IF statement. Results in a simpler syntax for non-programmers.
//
// It's called "flag" because its intended use is to toggle the visibility of command-line-flags.
// Don't @ me.
//
// Example with it: [fast|flag:'--fast']
// Example without: {% if fast %} --fast {% endif %}
func templateFlag(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error) {
	if in.Bool() {
		return pongo2.AsValue(param.String()), nil
	} else {
		return pongo2.AsValue(""), nil
	}
}

func RenderTemplate(template string, config map[string]interface{}, files map[string]string) (string, error) {

	// Combine files and config into one namespace for templating. Files win.
	context := map[string]interface{}{}
	for k, v := range config {
		context[k] = v
	}
	// At this stage, file keys point to host paths, not guest.
	// Map paths to the guest, just as in PrepareLocalRunFiles.
	for name, path := range files {
		context[name] = GuestPathForInput(name, path)
	}

	// pongo2 lexer does not support additional template symbols. Find-replace manually for now
	modified := strings.Replace(template, "[", "{{", -1)
	modified = strings.Replace(modified, "]", "}}", -1)

	// Comment block for registering functions.
	// This turns out to not work the desired way in the python impl.
	// If harmonizing on pongo2, this (far better) approach could be used.
	//
	// // Add template functions to context. Do not override the 'fw' key if present.
	// if _, ok := context["fw"]; !ok {
	// 	context["fw"] = map[string]interface{}{
	// 		"add": templateAdd,
	// 	}
	// } else {
	// 	// If the 'fw' key is present already, warn!
	// 	Println()
	// 	Println("Warning! The config or file name 'fw' is reserved.")
	// 	Println("This provides helper functions that may be useful when templating a command.")
	// 	Println("We strongly suggest chosing a different name for your config value or file.")
	// 	Println()
	// }

	tpl, err := pongo2.FromString(modified)
	if err != nil {
		return "", err
	}

	out, err := tpl.Execute(pongo2.Context(context))
	return out, err
}

func (docker *D) PythonInstalled(imageName string) bool {
	containerID, cleanup := docker.CreateContainerFromImage(imageName, &container.Config{
		Entrypoint: []string{},
		Cmd:        []string{"sh", "-c", "hash python"},
	}, nil)
	defer cleanup()

	code := docker.WaitOnContainer(containerID)
	return code == 0
}
