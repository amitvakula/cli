package gears

import (
	"os"
	"path"
	"path/filepath"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

// Example assets used when the base image does not have gear components

// Encapsulate the changes needed to modify an upstream manifest, or use a completely default one
func GenerateExampleManifestAndModifier(label, name, imageName, category string, env map[string]string, realName string) (*api.Gear, func(*api.Gear) *api.Gear) {

	basic := &api.Gear{
		Name:        name,
		Label:       label,
		Description: "Created by the gear builder. Edit the manifest file to give this gear a description!",
		Version:     "0",
		Inputs: map[string]map[string]interface{}{
			"dicom": {
				"base": "file",
				"type": map[string]interface{}{
					"enum": []string{"dicom"},
				},
				"optional":    true,
				"description": "Any dicom file.",
			},
			"api-key": {
				"base": "api-key",
			},
		},
		Config: map[string]map[string]interface{}{
			"address": {
				"default":     "Example",
				"type":        "string",
				"description": "String example: any text.",
			},
			"cost": {
				"default":     3.5,
				"type":        "number",
				"description": "Float example: any real number.",
			},
			"age": {
				"default":     7,
				"type":        "integer",
				"description": "Integer example: any whole number.",
			},
			"fast": {
				"default":     false,
				"type":        "boolean",
				"description": "Boolean example: a toggle.",
			},
			"nickname": {
				"default":     "Jimmy",
				"minLength":   2,
				"maxLength":   15,
				"type":        "string",
				"description": "String length example: 2 to 15 characters long.",
			},
			"phone": {
				"default":     "555-5555",
				"pattern":     "^[0-9]{3}-[0-9]{4}$",
				"type":        "string",
				"description": "String regex example: any phone number, no area code.",
			},
			"show-example": {
				"default":     false,
				"type":        "boolean",
				"description": "Show example features in the gear script!",
			},
		},
		Environment: map[string]string{
			"Example_Environment_Variable": "Set gear environment variables here.",
		},
		Command:    "./example.sh --age {{age}} --cost {{cost}}",
		Author:     realName,
		Maintainer: realName,
		Cite:       "List citations here.",
		License:    "Other",
		Source:     "",
		Url:        "",

		Custom: map[string]interface{}{
			"gear-builder": &GearBuilderInfo{
				Image:    imageName,
				Category: category,
			},
		},
	}

	// Use upstream environment decls if present
	if len(env) > 0 {
		basic.Environment = env
	}

	// Modifer function that incorporates GB changes into an upstream manifest.
	modifier := func(g *api.Gear) *api.Gear {
		g.Name = name
		g.Label = label
		g.Version = "0"
		g.Author = realName
		g.Maintainer = realName
		g.Custom = map[string]interface{}{}
		g.Custom["gear-builder"] = map[string]interface{}{
			"image":    imageName,
			"category": category,
		}

		// Merge environment map with upstream; upstream wins.
		if len(env) > 0 {

			finalEnv := map[string]string{}

			for k, v := range g.Environment {
				finalEnv[k] = v
			}

			for k, v := range env {
				finalEnv[k] = v
			}

			// If a manifest key was overridden, verbosely report it.
			if len(finalEnv) < len(g.Environment)+len(env) {
				Println()
				Println("Both the local manifest and the docker image have environment variables.")
				Println()
				Println("Original on disk:")
				PrintFormat(g.Environment)
				Println()
				Println("Original from docker image:")
				PrintFormat(env)
				Println()
				Println("The merged result:")
				PrintFormat(finalEnv)
				Println()
				Println("If desired, you may edit the manifest file to edit this merge.")
				Println()

				g.Environment = finalEnv
			}

		}

		return g
	}

	return basic, modifier
}

const ExampleRunScript = `#!/usr/bin/env bash
set -euo pipefail

echo "This is an example run script."
echo "Modify as desired, or set the manifest 'command' key to something else entirely."
echo

echo "This command was called with arguments:"
echo "$0" "$@"
echo

# Some simple diagnostics
set -x
env
cat /flywheel/v0/config.json
find /flywheel/v0/input -type f | xargs -r file || true
`

const ExamplePythonScript = `#!/usr/bin/env python

import codecs, json

print('This is an example python script.')
print('Modify as desired, or set the manifest "command" key to something else entirely.')

invocation  = json.loads(open('config.json').read())
config      = invocation['config']
inputs      = invocation['inputs']
destination = invocation['destination']


# Display everything provided to the job

def display(section):
	print(json.dumps(section, indent=4, sort_keys=True))

print('\nConfig:')
display(config)
print('\nDestination:')
display(destination)
print('\nInputs:')
display(inputs)

# Check a config value to see if example features were requested
if not config['show-example']:
	exit(1)

# Check if the flywheel SDK is installed
try:
	import flywheel

	# Make some simple calls to the API
	fw = flywheel.Flywheel(inputs['api-key']['key'])
	user = fw.get_current_user()
	config = fw.get_config()

	print('You are logged in as ' + user.firstname + ' ' + user.lastname + ' at ' + config.site.api_url[:-4])

except ImportError:
	print('\nFlywheel SDK is not installed, try "fw gear modify" and then "pip install flywheel-sdk".\n')


# Check if requests is installed
try:
	import requests

	# Find out how many people are in space \o/
	r = requests.get(
		'https://www.howmanypeopleareinspacerightnow.com/peopleinspace.json',

		headers={
			'Host': 'www.howmanypeopleareinspacerightnow.com',
			'User-Agent': 'curl/7.47.0',
			'Accept': '*/*',
		},
	)

	# Save astronaut data as metadata
	if r.ok:
		data = r.json()
		astronauts = data['people']

		print("There are " + str(len(astronauts)) + " in space today:")
		for astronaut in astronauts:
			print("\t" + astronaut['name'])

		metadata = {
			'session' : {
				'info': {
					'astronauts': astronauts
				}
			}
		}

		with open('output/.metadata.json', 'wb') as f:
		    json.dump(metadata, codecs.getwriter('utf-8')(f), ensure_ascii=False)

	else:
		# Might be that the API is down today. Check to see if we still have a space program?
		print('Not sure how many people are in space :(')
		print()
		print(r.text)


except ImportError:
	print('\nRequests is not installed, try "fw gear modify" and then "pip install requests".\n')

`

func BasicInvocation(config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"config": config,
		"destination": map[string]interface{}{
			"id":   "aex",
			"type": "acquisition",
		},
		"inputs": map[string]interface{}{},
	}
}

func BasicInvocationInput(inputName, inputPath string) map[string]interface{} {
	size := int64(0)
	info, err := os.Stat(inputPath)
	if err == nil {
		size = info.Size()
	}

	return map[string]interface{}{
		"base": "file",
		"hierarchy": map[string]interface{}{
			"id":   "aex",
			"type": "acquisition",
		},
		"location": map[string]interface{}{
			"name": filepath.Base(inputPath),
			"path": path.Join(GearPath, "input", inputName, filepath.Base(inputPath)),
		},
		"object": map[string]interface{}{
			"classification": map[string]interface{}{
				"Intent":      []string{},
				"Measurement": []string{},
			},
			"info":         map[string]interface{}{},
			"measurements": []string{},
			"mimetype":     "",
			"modality":     "",
			"size":         size,
			"tags":         []string{},
			"type":         "",
		},
	}
}

func BasicApiKeyInput(apikey string) map[string]interface{} {
	return map[string]interface{}{
		"base": "api-key",
		"key":  apikey,
	}
}
