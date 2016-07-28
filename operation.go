package main

import (
	"encoding/json"
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	// "text/template"

	"github.com/inconshreveable/log15"

	"flywheel.io/deja/job"
	"flywheel.io/deja/provider"
)

func (p *Project) Use(args []string) {
	if len(args) != 1 {
		Println("use only takes 1 argument: the name of the gear base.")
		os.Exit(1)
	}

	base := args[0]
	p.Base = base

	switch base {
	case "python3":
		Println("Downloading python3....")
		p.Provision("http", "https://storage.googleapis.com/flywheel/etc/deja-flak/anaconda-4.0.0.tar.gz", "vu0:sha384:PtQAKtjrhhH2FBawPhzDxyAYAM6R7mW935Nd9O0reRnfVEuI8F_HkWFwYZbBVax3")
	default:
		Println("Unrecognized gear base.")
		os.Exit(1)
	}
}

func (p *Project) Provision(itype, uri, vu string) {
	i := &job.Input{
		Type:     itype,
		Location: "/",
		URI:      uri,
		VuID:     vu,
	}
	f := &job.Formula{
		Inputs: []*job.Input{
			i,
		},
		Target: job.Target{
			Command: []string{"echo", "Gear base downloaded & tested."},
			Env:     map[string]string{},
			Dir:     "/",
		},
		Outputs: []*job.Output{},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result := provider.Run(f, provider.Logger(log))

	p.Input = i
	p.Input.VuID = result.Formula.Inputs[0].VuID // Workaround until deja doesn't modify result from caching
	p.Save()

	os.Exit(result.Result.ExitCode)
}

func (p *Project) Import(args []string) {
	if len(args) != 1 {
		Println("use only takes 1 argument: the filepath or HTTP url of the custom gear base.")
		os.Exit(1)
	}

	base := args[0]
	itype := "file"

	if strings.HasPrefix(base, "http") {
		itype = "http"
	}

	p.Base = "custom"
	p.Provision(itype, base, "")
}

func (p *Project) Run(args []string) {
	dir := filepath.Dir(p.Path)

	f := &job.Formula{
		Inputs: []*job.Input{
			p.Input,
			&job.Input{
				Type:     "bind",
				URI:      dir,
				Location: dir,
			},
		},
		Target: job.Target{
			Command: args,
			Env:     p.Env,
			Dir:     dir,
		},
		Outputs: []*job.Output{},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result := provider.Run(f, provider.Logger(log))
	os.Exit(result.Result.ExitCode)
}

const RunTemplate = `#!/bin/sh -e

set -x
pwd
ls -laR

echo
echo "Wrapper script built by flywheel CLI gear exporter"
cat manifest.json
echo

`

func (p *Project) Export(args []string) {
	dir := filepath.Dir(p.Path)

	manifest := map[string]interface{}{
		"name":        "gear-builder-export",
		"label":       "An exported gear",
		"description": "Built with the flywheel CLI gear exporter",
		"author":      "",
		"url":         "https://unknown.example",
		"source":      "https://unknown.example",
		"license":     "Other",
		"version":     "0",

		"inputs": map[string]interface{}{},
	}

	command := p.Command
	preface := ""

	for name, input := range p.Inputs {
		// Type fiddling, fix with better structs
		x := manifest["inputs"].(map[string]interface{})
		y := input.(map[string]interface{})
		flag := y["flag"].(string)

		x[name] = map[string]interface{}{
			"base": "file",
		}

		inputName := "input_" + name

		preface += inputName + "=`find /flywheel/v0/input/" + name + " -type f | head -1`" + "\n"

		command = append(command, flag, "${"+inputName+"}")
	}

	outflag := p.Output["flag"].(string)
	command = append(command, outflag, "/flywheel/v0/output")

	manifestBytes, _ := json.MarshalIndent(manifest, "", "\t")
	end := "\n" // json encode has no trailing newline? fix?
	manifestBytes = append(manifestBytes, end...)

	manifestPath := filepath.Join(dir, "manifest.json")
	err := ioutil.WriteFile(manifestPath, manifestBytes, 0644)
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	bash := RunTemplate + preface + "\n" + strings.Join(command, " ") + "\n"
	bashBytes := []byte(bash)

	bashPath := filepath.Join(dir, "run")
	err = ioutil.WriteFile(bashPath, bashBytes, 0755)
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	// tmpl, err := template.New("runscript").Parse(RunTemplate)
	// if err != nil {
	// 	Println(err)
	// 	os.Exit(1)
	// }
	// err = tmpl.Execute(os.Stdout, p)
	// if err != nil {
	// 	Println(err)
	// 	os.Exit(1)
	// }

	f := &job.Formula{
		Inputs: []*job.Input{
			p.Input,
			&job.Input{
				Type:     "bind",
				URI:      dir,
				Location: "/flywheel/v0/",
			},
		},
		Target: job.Target{
			Command: []string{"echo", "Gear exporting..."},
			// Env:     p.Env,
			// Dir: "dir",
		},
		Outputs: []*job.Output{
			&job.Output{
				Type:     "file",
				Location: "/",
				URI:      "gear.tar",
			},
		},
	}

	log := log15.New()
	log.SetHandler(log15.LvlFilterHandler(log15.LvlError, log15.StderrHandler))
	result := provider.Run(f, provider.Logger(log))

	resultBytes, _ := json.MarshalIndent(result, "", "\t")
	Println(string(resultBytes))

	os.Exit(result.Result.ExitCode)
}

func (p *Project) Frun(args []string) {
	if len(args) != len(p.Inputs) {
		Println("There are", len(p.Inputs), "inputs specified in flywheel.json but you gave", len(args))
		os.Exit(1)
	}

	// Could use go's templating system instead
	input := 0
	for i, x := range p.Args {
		if strings.HasPrefix(x, "{{") {
			p.Args[i] = args[input]
			input++
		}
	}

	p.Run(p.Args)
}
