package client

import (
	"encoding/json"
	"errors"
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	prompt "github.com/segmentio/go-prompt"

	oapi "flywheel.io/fw/api"
	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

func resolveLast(path []string) oapi.Container {
	result, _, err, _ := oc.ResolvePath(path)
	Check(err)

	if result == nil || result.Path == nil || len(result.Path) < 1 {
		return nil
	}

	return result.Path[len(result.Path)-1].(oapi.Container)
}

func scan(folder string, fn func(name string, mode os.FileMode)) {
	files, err := ioutil.ReadDir(folder)
	Check(err)

	for _, file := range files {
		name := file.Name()
		mode := file.Mode()
		fn(name, mode)
	}
}

type scanRoot struct {
	Children []*scanGroup
}

func (r *scanRoot) report() {
	for _, x := range r.Children {
		x.report("")
	}
}
func (r *scanRoot) inflate() {
	for _, x := range r.Children {
		x.inflate()
	}
}

const increment = "│   "
const supplicant = "├"
const spacer = "──"

func rE(exists bool) string {
	if exists {
		return " (using)"
	} else {
		return " (creating)"
	}
}

func retry(fn func() error) {
	for {
		err := fn()

		if err != nil {
			Println("An error occurred:", err.Error())
			proceed := prompt.Confirm("Retry? (yes/no)")
			Println()
			if !proceed {
				Println("Canceled.")
				os.Exit(1)
			}
		} else {
			break
		}
	}
}

func (r *scanRoot) discover(folder string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {
			path := []string{name}

			group := &scanGroup{
				Group: &api.Group{
					Id: name,
				},
			}

			if resolveLast(path) != nil {
				group.Exists = true
			}

			group.discover(filepath.Join(folder, name), path)

			r.Children = append(r.Children, group)
		} else {
			Println("File", name, "ignored as attachments to root are not allowed")
		}
	})
}

type scanGroup struct {
	*api.Group
	Exists   bool
	Children []*scanProject
}

func (r *scanGroup) report(i string) {
	Println(i + supplicant + spacer + r.Id + rE(r.Exists))

	for _, x := range r.Children {
		x.report(i + increment)
	}
}
func (r *scanGroup) inflate() {
	if !r.Exists {
		Println("Creating group", r.Id)

		retry(func() error {
			id, _, err := c.AddGroup(r.Group)
			r.Id = id
			return err
		})
	}

	for _, x := range r.Children {
		x.inflate(r.Id)
	}
}

func (r *scanGroup) discover(folder string, path []string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {
			newPath := append(path, name)

			project := &scanProject{
				Project: &api.Project{
					Name: name,
				},
			}

			c := resolveLast(newPath)
			if c != nil {
				project.Exists = true
				project.Id = c.GetId()
			}

			project.discover(filepath.Join(folder, name), newPath)

			r.Children = append(r.Children, project)
		} else {
			Println("File", name, "ignored as attachments to groups are not allowed")
		}
	})
}

type scanProject struct {
	*api.Project
	Exists      bool
	Children    []*scanSubject
	Attachments []*api.UploadSource
}

func (r *scanProject) report(i string) {
	Println(i + supplicant + spacer + r.Name + rE(r.Exists))

	for _, x := range r.Attachments {
		Println(i + increment + supplicant + spacer + x.Name)
	}

	for _, x := range r.Children {
		x.report(i + increment)
	}
}

func (r *scanProject) inflate(groupId string) {
	r.GroupId = groupId

	if !r.Exists {
		Println("Creating project", r.Name)

		retry(func() error {
			id, _, err := c.AddProject(r.Project)
			r.Id = id
			return err
		})
	}

	for _, x := range r.Attachments {
		Println("Upload file", x.Name)
		retry(func() error {
			progress, result := c.UploadSimple("projects/"+r.Id+"/files", nil, x)

			for update := range progress {
				Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			return <-result
		})
	}

	for _, x := range r.Children {
		x.inflate(groupId, r.Id)
	}
}

func (r *scanProject) discover(folder string, path []string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {

			subject := &scanSubject{
				Subject: &api.Subject{
					// Id:   randStringOfLength(24),
					Name: name,
				},
			}

			// does NOT append to path because subjects are not in resolver
			subject.discover(filepath.Join(folder, name), path)

			r.Children = append(r.Children, subject)
		} else {
			attachment := api.CreateUploadSourceFromFilenames(filepath.Join(folder, name))[0]
			r.Attachments = append(r.Attachments, attachment)
		}
	})
}

type scanSubject struct {
	*api.Subject
	Children []*scanSession
}

func (r *scanSubject) report(i string) {
	Println(i + supplicant + spacer + r.Name)

	for _, x := range r.Children {
		x.report(i + increment)
	}
}

func (r *scanSubject) inflate(groupId, projectId string) {
	for _, x := range r.Children {
		x.inflate(groupId, projectId)
	}
}

func (r *scanSubject) discover(folder string, path []string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {
			newPath := append(path, name)

			session := &scanSession{
				Session: &api.Session{
					Name:    name,
					Subject: r.Subject,
				},
			}

			c := resolveLast(newPath)
			if c != nil {
				session.Exists = true
				session.Id = c.GetId()
			}

			session.discover(filepath.Join(folder, name), newPath)

			r.Children = append(r.Children, session)
		} else {
			Println("File", name, "ignored as attachments to subjects are not allowed")
		}
	})
}

type scanSession struct {
	*api.Session
	Exists      bool
	Children    []*scanAcquisition
	Attachments []*api.UploadSource
}

func (r *scanSession) report(i string) {
	Println(i + supplicant + spacer + r.Name + rE(r.Exists))

	for _, x := range r.Attachments {
		Println(i + increment + supplicant + spacer + x.Name)
	}

	for _, x := range r.Children {
		x.report(i + increment)
	}
}

func (r *scanSession) inflate(groupId, projectId string) {
	// r.GroupId = groupId
	r.ProjectId = projectId

	if !r.Exists {
		Println("Creating session", r.Name)

		retry(func() error {
			id, _, err := c.AddSession(r.Session)
			r.Id = id
			return err
		})
	}

	for _, x := range r.Attachments {
		Println("Upload file", x.Name)
		retry(func() error {
			progress, result := c.UploadSimple("sessions/"+r.Id+"/files", nil, x)

			for update := range progress {
				Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			return <-result
		})
	}

	for _, x := range r.Children {

		metadata := map[string]interface{}{
			"project": map[string]interface{}{
				"_id": projectId,
			},
			"session": map[string]interface{}{
				"label": r.Name,
				// "subject": map[string]interface{}{
				// 	"code": r.Subject.Name,
				// },
			},
			"acquisition": map[string]interface{}{
				"label": x.Name,
				// "timestamp": "1970-01-01T06:00:00.000Z",
			},
		}

		x.inflate(r.Id, projectId, metadata)
	}
}

func (r *scanSession) discover(folder string, path []string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {
			newPath := append(path, name)

			acquisition := &scanAcquisition{
				Acquisition: &api.Acquisition{
					Name: name,
				},
			}

			c := resolveLast(newPath)
			if c != nil {
				acquisition.Exists = true
				acquisition.Id = c.GetId()
			}

			acquisition.discover(filepath.Join(folder, name), newPath)

			r.Children = append(r.Children, acquisition)
		} else {
			attachment := api.CreateUploadSourceFromFilenames(filepath.Join(folder, name))[0]
			r.Attachments = append(r.Attachments, attachment)
		}
	})
}

type scanAcquisition struct {
	*api.Acquisition
	Exists      bool
	Attachments []*api.UploadSource
	Packfiles   []*api.UploadSource
}

func (r *scanAcquisition) report(i string) {
	Println(i + supplicant + spacer + r.Name + rE(r.Exists))

	for _, x := range r.Attachments {
		Println(i + increment + supplicant + spacer + x.Name)
	}

	for _, x := range r.Packfiles {
		Println(i + increment + supplicant + spacer + " (*) " + x.Name)
	}
}

func (r *scanAcquisition) inflate(sessionId, projectId string, metadata map[string]interface{}) {
	r.SessionId = sessionId

	if !r.Exists {
		Println("Creating acquisition", r.Name)

		retry(func() error {
			id, _, err := c.AddAcquisition(r.Acquisition)
			r.Id = id
			return err
		})
	}

	for _, x := range r.Attachments {
		Println("Upload file", x.Name)
		retry(func() error {
			progress, result := c.UploadSimple("acquisitions/"+r.Id+"/files", nil, x)

			for update := range progress {
				Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			return <-result
		})
	}

	for _, x := range r.Packfiles {
		name := filepath.Base(x.Path)
		Println("Upload packfile", name)

		retry(func() error {

			metadata["packfile"] = map[string]interface{}{
				"type": name,
			}

			mdRaw, err := json.Marshal(&metadata)
			if err != nil {
				return err
			}
			mdString := string(mdRaw)

			var aerr *api.Error

			type tokenResponse struct {
				Token string `json:"token"`
			}

			var response *tokenResponse

			_, err = c.New().Post("projects/"+projectId+"/packfile-start").Receive(&response, &aerr)

			if err != nil {
				return err
			} else if aerr != nil {
				return errors.New(aerr.Message)
			} else if response == nil || response.Token == "" {
				return errors.New("Packfile token was empty or missing")
			}

			token := response.Token

			Println("Scanning", x.Path)
			var paths []*api.UploadSource
			scan(x.Path, func(name string, mode os.FileMode) {
				if mode.IsRegular() {
					src := api.CreateUploadSourceFromFilenames(filepath.Join(x.Path, name))[0]
					paths = append(paths, src)
				}
			})

			progress, result := c.UploadSimple("projects/"+projectId+"/packfile?token="+token, nil, paths...)

			for update := range progress {
				Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			err = <-result
			if err != nil {
				return err
			}

			/*
				metadata:{"project":{"_id":"58a47373d2b6ed0013a4a9fb"},"session":{"label":"01/01/70 00:00 AM","subject":{"code":"XXX"}},"acquisition":{"label":"Localizer","timestamp":"1970-01-01T06:00:00.000Z"},"packfile":{"type":"dicom"}}
			*/

			req, err := c.New().Get("projects/" + projectId + "/packfile-end?token=" + token + "&metadata=" + mdString).Request()

			if err != nil {
				return err
			}

			// Wait for SSE
			resp, err := c.Client.Do(req)

			if resp.StatusCode != 200 {
				// Needs robust handling for body & raw nils
				raw, _ := ioutil.ReadAll(resp.Body)
				return errors.New(string(raw))
			}
			return err
		})
	}
}

func (r *scanAcquisition) discover(folder string, path []string) {
	scan(folder, func(name string, mode os.FileMode) {
		if mode.IsDir() {
			packfile := api.CreateUploadSourceFromFilenames(filepath.Join(folder, name))[0]
			r.Packfiles = append(r.Packfiles, packfile)

		} else {
			attachment := api.CreateUploadSourceFromFilenames(filepath.Join(folder, name))[0]
			r.Attachments = append(r.Attachments, attachment)
		}
	})
}

var c *api.Client
var oc *oapi.Client

func ScanUpload(folder string) {
	x := LoadCreds()
	c = api.NewApiKeyClient(x.Host, x.Key, x.Insecure)
	oc = MakeClient()
	Println()

	root := &scanRoot{}

	root.discover(folder)

	Println()
	Println("The following data hierarchy was found:")
	Println()
	root.report()
	Println()
	proceed := prompt.Confirm("Confirm upload? (yes/no)")
	Println()
	if !proceed {
		Println("Canceled.")
		return
	}
	Println("Beginning upload.")
	Println()

	root.inflate()
}
