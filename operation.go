package main

import (
	"crypto/tls"
	"encoding/json"
	. "fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dghubble/sling"
	"github.com/inconshreveable/log15"

	"flywheel.io/deja/job"
	"flywheel.io/deja/provider"
)

func client(host, key string) (*http.Client, *sling.Sling) {

	// Create a custom transport that accepts the passed insecure setting
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Load transport into a client
	client := &http.Client{Transport: tr}

	// Load http client into library
	return client, sling.New().
		Base("https://"+host+"/").Path("api/").
		Set("Authorization", "scitran-user "+key).
		Client(client)
}

type self struct {
	Id        string                 `json:"_id"`
	Key       map[string]interface{} `json:"api_key"`
	Email     string                 `json:"email"`
	Firstname string                 `json:"firstname"`
	Lastname  string                 `json:"lastname"`
}

type apierr struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type group struct {
	Id   string `json:"_id"`
	Name string `json:"name"`
}
type project struct {
	Id    string `json:"_id"`
	Group string `json:"group"`
	Label string `json:"label"`
}
type session struct {
	Id        string `json:"_id"`
	Group     string `json:"group"`
	ProjectId string `json:"project"`
	Label     string `json:"label"`
}
type file struct {
	Name string `json:"name"`
}
type acquisition struct {
	Id        string `json:"_id"`
	SessionId string `json:"session"`
	Label     string `json:"label"`
	Files     []file `json:"files"`
}

func check(resp *http.Response, err error, aerr apierr) {
	if err != nil {
		Println(err)
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		Println(resp)
		Println()
		Println(aerr.Message)
		os.Exit(1)
	}
}

func Login(args []string) {
	if len(args) != 2 {
		Println("login takes 2 arguments: a URL pointing at the FLywheel web app, and your API key.")
		os.Exit(1)
	}

	flyS := args[0]
	key := args[1]

	url, err := url.Parse(flyS)
	if err != nil {
		Println(err)
		os.Exit(1)
	}
	_, client := client(url.Host, key)

	var user self
	var aerr apierr
	resp, err := client.New().Get("users/self").Receive(&user, &aerr)
	check(resp, err, aerr)

	Println("Logged in as", user.Firstname, user.Lastname, "<"+user.Email+">")

	creds := map[string]string{
		"host": url.Host,
		"key":  key,
	}

	whelp, _ := json.MarshalIndent(creds, "", "\t")

	err = ioutil.WriteFile("user.json", whelp, 0644)
	if err != nil {
		Println(err)
		os.Exit(1)
	}
}

func LsGroups(client *sling.Sling) []group {
	var aerr apierr
	var groups []group

	resp, err := client.New().Get("groups").Receive(&groups, &aerr)
	check(resp, err, aerr)
	return groups
}

func LsProjects(client *sling.Sling, group string) []project {
	var aerr apierr
	var projects []project

	resp, err := client.New().Get("projects").Receive(&projects, &aerr)
	check(resp, err, aerr)

	var filteredProjects []project
	for _, x := range projects {
		if x.Group == group {
			filteredProjects = append(filteredProjects, x)
		}
	}
	return filteredProjects
}

func GetProjectId(client *sling.Sling, group, project string) string {
	projects := LsProjects(client, group)

	for _, x := range projects {
		if x.Label == project {
			return x.Id
		}
	}

	Println(group, project)

	Println("Project", project, "not found")
	os.Exit(1)
	return ""
}

func LsSessions(client *sling.Sling, group, project string) []session {
	var aerr apierr
	var sessions []session

	resp, err := client.New().Get("sessions").Receive(&sessions, &aerr)
	check(resp, err, aerr)

	projectId := GetProjectId(client, group, project)

	var filteredSessions []session
	for _, x := range sessions {
		if x.Group == group && x.ProjectId == projectId {
			filteredSessions = append(filteredSessions, x)
		}
	}
	return filteredSessions
}

func GetSessionId(client *sling.Sling, group, project, session string) string {
	sessions := LsSessions(client, group, project)
	projectId := GetProjectId(client, group, project)

	for _, x := range sessions {
		if x.ProjectId == projectId && x.Label == session {
			return x.Id
		}
	}

	Println("Session", session, "not found")
	os.Exit(1)
	return ""
}

func LsAcquisitions(client *sling.Sling, group, project, session string) []acquisition {
	var aerr apierr
	var acquisitions []acquisition

	resp, err := client.New().Get("acquisitions").Receive(&acquisitions, &aerr)
	check(resp, err, aerr)

	sessionId := GetSessionId(client, group, project, session)

	var filteredAcquisitions []acquisition
	for _, x := range acquisitions {
		if x.SessionId == sessionId {
			filteredAcquisitions = append(filteredAcquisitions, x)
		}
	}
	return filteredAcquisitions
}

func DownloadFile(client *sling.Sling, hclient *http.Client, group, project, session, acquisition, filename string) {

	acquisitions := LsAcquisitions(client, group, project, session)
	for _, y := range acquisitions {
		if y.Label == acquisition {
			for _, x := range y.Files {
				if x.Name == filename {

					file, err := os.Create(filename)
					if err != nil {
						Println(err)
						os.Exit(1)
					}
					defer file.Close()

					req, err := client.New().Get("acquisitions/" + y.Id + "/files/" + filename).Request()
					if err != nil {
						Println(err)
						os.Exit(1)
					}

					resp, err := hclient.Do(req)
					if err != nil {
						Println(err)
						os.Exit(1)
					}

					_, err = io.Copy(file, resp.Body)
					if err != nil {
						Println(err)
						os.Exit(1)
					}

				}
			}
		}
	}

}

func Download(args []string) {
	b, err := ioutil.ReadFile("user.json")
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	var creds map[string]string
	err = json.Unmarshal(b, &creds)
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	hclient, client := client(creds["host"], creds["key"])

	if len(args) != 1 {
		Println("Download requires one argument: the path of the file to download.")
		os.Exit(1)
	}

	path := args[0]
	frags := strings.Split(path, "/")

	if len(frags) != 5 {
		Println("Download path must be of form 'group/project/session/acquisition/filename'")
		os.Exit(1)
	}

	DownloadFile(client, hclient, frags[0], frags[1], frags[2], frags[3], frags[4])
}

func Ls(args []string) {
	b, err := ioutil.ReadFile("user.json")
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	var creds map[string]string
	err = json.Unmarshal(b, &creds)
	if err != nil {
		Println(err)
		os.Exit(1)
	}

	_, client := client(creds["host"], creds["key"])

	level := "groups"
	group := ""
	project := ""
	session := ""
	acquisition := ""
	if len(args) == 1 {
		path := args[0]

		frags := strings.Split(path, "/")

		switch len(frags) {
		case 1:
			level = "projects"
			group = frags[0]
		case 2:
			level = "sessions"
			group = frags[0]
			project = frags[1]
		case 3:
			level = "acquisitions"
			group = frags[0]
			project = frags[1]
			session = frags[2]
		case 4:
			level = "files"
			group = frags[0]
			project = frags[1]
			session = frags[2]
			acquisition = frags[3]
		default:
			Println("I'm not sure what you mean by " + path)
			os.Exit(1)
		}
	}

	switch level {
	case "groups":
		groups := LsGroups(client)

		for _, y := range groups {
			Println(y.Id, y.Name)
		}
	case "projects":
		projects := LsProjects(client, group)

		for _, y := range projects {
			Println(y.Label)
		}
	case "sessions":
		sessions := LsSessions(client, group, project)

		for _, y := range sessions {
			Println(y.Label)
		}
	case "acquisitions":
		acquisitions := LsAcquisitions(client, group, project, session)

		for _, y := range acquisitions {
			Println(y.Label)
		}
	case "files":
		acquisitions := LsAcquisitions(client, group, project, session)

		for _, y := range acquisitions {
			if y.Label == acquisition {
				for _, x := range y.Files {
					Println(x.Name)
				}
			}
		}
	default:
		Println("Error, unknown hierarchy level")
		os.Exit(1)
	}

}

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

	inputs := []*job.Input{
		p.Input,
		&job.Input{
			Type:     "bind",
			URI:      dir,
			Location: dir,
		},
	}

	// Throw input flags in a map for easy lookup.
	flags := map[string]interface{}{}
	for _, y := range p.Inputs {
		y := y.(map[string]interface{})
		flag := y["flag"].(string)
		flags[flag] = nil
	}

	// Throw output flag in map.
	flags[p.Output["flag"].(string)] = nil

	for index, flag := range args {
		_, isFlag := flags[flag]

		// Check that it's an input flags that was not passed as the very last argument.
		if isFlag && index != len(args)-1 {
			param := args[index+1]
			baseDir := filepath.Dir(p.Path)

			abs, err := filepath.Abs(param)
			if err != nil {
				Println(err)
				os.Exit(1)
			}

			// If the parameter inside the project dir, we don't need extra mounts
			if strings.HasPrefix(abs, baseDir) {
				continue
			}

			// Check the potential mount path
			mountPath := abs
			info, err := os.Stat(abs)
			if err != nil && !os.IsNotExist(err) {
				Println(err)
				os.Exit(1)
			}

			// If the path does not exist or is a file, assume we should mount the folder instead.
			if info == nil || !info.IsDir() {
				mountPath = filepath.Dir(abs)
			}

			// Println("Flag", flag, "was passed", param, "which is outside project directory. Mounting", mountPath)

			// Add bind mount to the formula.
			inputs = append(inputs, &job.Input{
				Type:     "bind",
				URI:      mountPath,
				Location: mountPath,
			})
		}
	}

	f := &job.Formula{
		Inputs: inputs,
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
