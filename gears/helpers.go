package gears

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/flosch/pongo2"
	prompt "github.com/segmentio/go-prompt"

	. "flywheel.io/fw/util"
	"flywheel.io/sdk/api"
)

var (
	// Default to OS preference
	BaseTempDir = ""

	// Easily identified state dirs
	BaseTempPrefix = "fw-gear-builder-"
)

func init() {
	switch runtime.GOOS {
	case "darwin":
		// On Mac a default tempdir goes to something on the order of /var/folders/fp/znt3073d3313h1r_sbj9292h0000gn/T/fw-gear-builder594038416
		// This then causes drama, presumably due to the containers-in-VM situation; pants with suspenders.
		BaseTempDir = "/tmp/"
	}
}

func FetchName(client *api.Client) string {
	Println("Checking Flywheel connectivity...")
	user, _, err := client.GetCurrentUser()
	Check(err)
	return user.Firstname + " " + user.Lastname
}

func RenderTemplate(template string, context map[string]interface{}) (string, error) {

	// This is disabled server-side pending a threat model review.
	// Further, this restricts the range of valid config values beyond what the gear spec allows.
	// Ref pongo2/context.go/checkForValidIdentifiers.
	//
	// Example:
	// [Error (where: checkForValidIdentifiers)] Context-key 'whatever-it-takes' (value: 'false') is not a valid identifier.
	//
	// Disabled with imports active to leave the code path hot.

	tpl, err := pongo2.FromString(template)
	if err != nil {
		return "", err
	}

	_, _ = tpl.Execute(pongo2.Context(context))
	return template, nil
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

				err = os.Remove(header.Name)
				if err != nil {
					return err
				}

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

func TarCWD(out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	tr := tar.NewWriter(out)
	defer tr.Close()

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(strings.Replace(path, cwd, GearPath, -1), string(filepath.Separator))

		err = tr.WriteHeader(header)
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(tr, file)
		if err != nil {
			return err
		}

		return file.Close()
	})

	return err
}

func MakeGearTicket(client *api.Client, doc *api.GearDoc) (string, error) {
	Println("Checking that gear is ready to upload...")
	var aerr *api.Error

	ticketMap := &struct {
		Ticket string `json:"ticket,omitempty"`
	}{}

	_, err := client.New().Post("gears/prepare-add").BodyJSON(doc).Receive(&ticketMap, &aerr)

	// Exception case: handle version conflicts
	if err == nil && aerr != nil && strings.Contains(aerr.Message, "already exists") {
		return "", api.Coalesce(err, aerr)
	}

	// Other errors are fatal
	Check(api.Coalesce(err, aerr))

	if ticketMap.Ticket == "" {
		FatalWithMessage("Server response was empty; contact support.")
	}

	return ticketMap.Ticket, nil
}

func MakeGearTicketReslient(client *api.Client, doc *api.GearDoc) string {
	for {
		ticket, sendErr := MakeGearTicket(client, doc)

		if sendErr == nil {
			return ticket
		}

		i, castErr := strconv.Atoi(doc.Gear.Version)

		if castErr != nil {
			Check(sendErr)
		}

		runConfirmFatal(createConfirm("Version " + strconv.Itoa(i) + " already exists, bump to " + strconv.Itoa(i+1) + "?"))

		doc.Gear.Version = strconv.Itoa(i + 1)

		err := ioutil.WriteFile(ManifestName, FormatBytes(doc.Gear), 0640)
		Check(err)
	}
}

func FinishGearTicket(client *api.Client, ticket, repo, digest string) {
	ticketMap := &struct {
		Ticket string `json:"ticket,omitempty"`
		Repo   string `json:"repo,omitempty"`
		Digest string `json:"pointer,omitempty"`
	}{
		Ticket: ticket,
		Repo:   repo,
		Digest: digest,
	}

	var aerr *api.Error
	var result map[string]interface{}
	_, err := client.New().Post("gears/save").BodyJSON(ticketMap).Receive(&result, &aerr)
	Check(api.Coalesce(err, aerr))
	// PrintFormat(result)
}
