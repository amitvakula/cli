package legacy

import (
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"flywheel.io/sdk/api"
)

func GetDownloadTicket(client *api.Client, request *ContainerTicketRequest) (*ContainerTicketResponse, *http.Response, error) {

	var aerr *api.Error
	var ticket *ContainerTicketResponse

	resp, err := client.Sling.New().Post("download").BodyJSON(request).Receive(&ticket, &aerr)
	return ticket, resp, api.Coalesce(err, aerr)
}

func Download(client *api.Client, filename string, parent interface{}, dest io.Writer) (*http.Response, error) {
	url := ""
	switch parent := parent.(type) {
	case *Project:
		url = "projects/" + parent.Id + "/files/" + filename
	case *Subject:
		url = "subjects/" + parent.Id + "/files/" + filename
	case *Session:
		url = "sessions/" + parent.Id + "/files/" + filename
	case *Acquisition:
		url = "acquisitions/" + parent.Id + "/files/" + filename
	case *ContainerTicketResponse:
		url = "download?ticket=" + parent.Ticket
	default:
		return nil, errors.New("Cannot download from unknown container type")
	}

	req, err := client.Sling.New().Get(url).Request()
	if err != nil {
		return nil, err
	}

	resp, err := client.Doer.Do(req)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode != 200 {
		// Needs robust handling for body & raw nils
		raw, _ := ioutil.ReadAll(resp.Body)
		return resp, errors.New(string(raw))
	}

	_, err = io.Copy(dest, resp.Body)
	return resp, err
}

func DownloadToFile(client *api.Client, filename string, parent interface{}, destPath string) (*http.Response, error) {
	file, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	resp, err := Download(client, filename, parent, file)
	if err != nil {
		os.Remove(destPath) // silently attempt to remove broken file
	}
	return resp, err
}

func Upload(client *api.Client, filename string, parent interface{}, metadata []byte, src io.Reader) (*http.Response, error) {
	url := ""
	switch parent := parent.(type) {
	case *Project:
		url = "projects/" + parent.Id + "/files"
	case *Subject:
		url = "subjects/" + parent.Id + "/files"
	case *Session:
		url = "sessions/" + parent.Id + "/files"
	case *Acquisition:
		url = "acquisitions/" + parent.Id + "/files"
	case *api.Gear:
		url = "gears/" + parent.Name + "?upload=true"
	case *Group:
		return nil, errors.New("Uploading files to a group is not supported" + filename)
	default:
		return nil, errors.New("Cannot upload to unknown container type")
	}

	filenames := []string{filename}

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	contentType := multipartWriter.FormDataContentType()

	// Stream the multipart encoding as it's read from the pipe -> http
	go func() {
		defer func() {
			multipartWriter.Close()
			writer.Close()
		}()

		// Add metadata, if any
		if len(metadata) > 0 {
			mWriter, err := multipartWriter.CreateFormField("metadata")
			if err != nil {
				return
			}
			_, err = mWriter.Write(metadata)
			if err != nil {
				return
			}
		}

		for i, filename := range filenames {

			// Create a form name for this file
			formTitle := "file"
			if len(filenames) > 1 {
				formTitle = strings.Join([]string{"file", strconv.Itoa(i + 1)}, "")
			}

			// Create a form entry for this file
			writer, err := multipartWriter.CreateFormFile(formTitle, filename)
			if err != nil {
				return
			}

			// Copy the file
			_, err = io.Copy(writer, src)
			if err != nil && err != io.EOF {
				return
			}
		}
	}()
	// END RAW

	req, err := client.Sling.New().Post(url).
		Body(reader).
		Set("Content-Type", contentType).
		Request()
	if err != nil {
		return nil, err
	}

	resp, err := client.Doer.Do(req)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode != 200 {
		// Needs robust handling for body & raw nils
		raw, _ := ioutil.ReadAll(resp.Body)
		return resp, errors.New(string(raw))
	}

	return resp, nil
}

func UploadFromFile(client *api.Client, filename string, parent interface{}, metadata []byte, filepath string) (*http.Response, error) {
	fd, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return Upload(client, filename, parent, metadata, fd)
}
