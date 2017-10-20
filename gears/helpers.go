package gears

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	. "fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/flosch/pongo2"
	prompt "github.com/segmentio/go-prompt"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func RenderTemplate(template string, context map[string]interface{}) (string, error) {
	tpl, err := pongo2.FromString(template)
	if err != nil {
		return "", err
	}

	out, err := tpl.Execute(pongo2.Context(context))
	return out, err
}

// >:|
func RenderTemplateStringMap(template string, context map[string]string) (string, error) {
	x := map[string]interface{}{}

	for key, value := range context {
		x[key] = value
	}

	return RenderTemplate(template, x)
}

func stripCtlFromBytes(str string) string {

	// This has lots of work to do yet. Should try to reuse prior work.

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
		Check(json.Unmarshal(plan, &gear))
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
