package main

import (
	"encoding/json"
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"flywheel.io/deja/job"
)

type Project struct {
	Path    string `json:"-"`
	Version int    `json:"version"`

	Base  string     `json:"base,omitempty"`
	Input *job.Input `json:"input,omitempty"`

	Env  map[string]string `json:"env,omitempty"`
	Args []string          `json:"args,omitempty"`

	Inputs map[string]interface{} `json:"inputs,omitempty"`
	Output map[string]interface{} `json:"output,omitempty"`

	Command []string `json:"command,omitempty"`

	/*
		"inputs": {
			"example": {
				"flag": "-i"
			}
		},
		"output": {
			"flag": "-o"
		},
		"command": [
			"python", "./rot13.py"
		]

	*/
}

func (p *Project) Encode() []byte {
	// TODO: could just implement io.Writer instead
	result, _ := json.MarshalIndent(p, "", "\t")
	end := "\n" // json encode has no trailing newline? fix?
	return append(result, end...)
}

func (p *Project) Save() {
	err := ioutil.WriteFile(p.Path, p.Encode(), 0644)
	if err != nil {
		Println(err)
		os.Exit(2)
	}
}

func NewProject() *Project {
	return &Project{
		Version: 0,
	}
}

func LoadProject(path string) *Project {
	var p Project
	data, err := ioutil.ReadFile(path)
	if err != nil {
		Println(err)
		os.Exit(1)
	}
	err = json.Unmarshal(data, &p)
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	p.Path = path
	return &p
}

const FlywheelFile = "flywheel.json"

func Setup() *Project {
	path, created := FindOrCreateFlywheelFile()

	if created {
		Println("Created new Flywheel project in", filepath.Dir(path))
	} else {
		// Println("Using existing Flywheel project in", filepath.Dir(path))
	}

	return LoadProject(path)
}

// Returns a flywheel file, and a bool indicating if the function created the file
func FindOrCreateFlywheelFile() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	dir := cwd
	for {
		file := filepath.Join(dir, FlywheelFile)
		info, err := os.Stat(file)

		if err == nil && !info.IsDir() {
			return file, false
		} else if err == nil && info.IsDir() {
			Println(file, "should be a file, but instead is a directory.")
			os.Exit(1)
		} else if os.IsNotExist(err) && dir != "/" {
			// pop up a directory
			dir = filepath.Dir(dir)
		} else if os.IsNotExist(err) {
			// We're at the root dir; create a new flywheel folder in the cwd
			file := filepath.Join(cwd, FlywheelFile)

			err := os.MkdirAll(filepath.Dir(file), 0755)
			if err != nil {
				Println(err)
				os.Exit(2)
			}

			project := NewProject()
			err = ioutil.WriteFile(file, project.Encode(), 0644)
			if err != nil {
				Println(err)
				os.Exit(2)
			}

			return file, true
		} else {
			Println(err)
			os.Exit(1)
		}
	}
}
