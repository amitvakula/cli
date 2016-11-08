package api

import (
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func (c *Client) Download(filename string, parent interface{}, dest io.Writer) (*http.Response, error) {
	url := ""
	switch parent := parent.(type) {
	case *Project:
		url = "projects/" + parent.Id + "/files/" + filename
	case *Session:
		url = "sessions/" + parent.Id + "/files/" + filename
	case *Acquisition:
		url = "acquisitions/" + parent.Id + "/files/" + filename
	default:
		return nil, errors.New("Cannot download from unknown container type")
	}

	req, err := c.S.New().Get(url).Request()
	if err != nil {
		return nil, err
	}

	resp, err := c.C.Do(req)
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

func (c *Client) DownloadToFile(filename string, parent interface{}, destPath string) (*http.Response, error) {
	file, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	resp, err := c.Download(filename, parent, file)
	if err != nil {
		os.Remove(destPath) // silently attempt to remove broken file
	}
	return resp, err
}

func (c *Client) Upload(filename string, parent interface{}, metadata []byte, src io.Reader) (*http.Response, error) {
	url := ""
	switch parent := parent.(type) {
	case *Project:
		url = "projects/" + parent.Id + "/files/" + filename
	case *Session:
		url = "sessions/" + parent.Id + "/files/" + filename
	case *Acquisition:
		url = "acquisitions/" + parent.Id + "/files/" + filename
	case *Gear:
		url = "gears/" + parent.Name + "?upload=true"
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

	req, err := c.S.New().Post(url).
		Body(reader).
		Set("Content-Type", contentType).
		Request()
	if err != nil {
		return nil, err
	}

	resp, err := c.C.Do(req)
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

func (c *Client) UploadFromFile(filename string, parent interface{}, metadata []byte, filepath string) (*http.Response, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return c.Upload(filename, parent, metadata, fd)
}
