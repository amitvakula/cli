package main

import (
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/sling"
)

type ApiError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

type Permission struct {
	Id    string `json:"_id"`
	Level string `json:"access"`
	Site  string `json:"site"`
}

type Subject struct {
	Id   string `json:"_id"`
	Name string `json:"code"`
}

type Origin struct {
	Id     string `json:"id"`
	Method string `json:"method"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

type ApiKey struct {
	Key string `json:"key"`

	Created  time.Time `json:"created"`
	LastUsed string    `json:"last_used"`
}

type User struct {
	Id        string `json:"_id"`
	Email     string `json:"email"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	ApiKey    ApiKey `json:"api_key"`

	Avatar  string            `json:"avatar"`
	Avatars map[string]string `json:"avatar"` // lol, whatever

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Root     bool      `json:"root"`
}

type File struct {
	Name   string `json:"name"`
	Origin Origin `json:"origin"`
	Size   int    `json:"size"`

	Instrument   string   `json:"instrument"`
	Mimetype     string   `json:"mimetype"`
	Measurements []string `json:"measurements"`
	Type         string   `json:"type"`
	Tags         []string `json:"tags"`

	Metadata map[string]interface{} `json:"metadata"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

type Group struct {
	Id   string `json:"_id"`
	Name string `json:"name"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`

	Permissions []*Permission `json:"roles"`
}

type Project struct {
	Id      string `json:"_id"`
	Name    string `json:"label"`
	GroupId string `json:"group"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Files    []*File   `json:"files"`

	Public      bool          `json:"public"`
	Permissions []*Permission `json:"permissions"`
}

type Session struct {
	Id        string `json:"_id"`
	Name      string `json:"label"`
	GroupId   string `json:"group"`
	ProjectId string `json:"project"`

	Subject   *Subject  `json:"subject"`
	Timestamp time.Time `json:"timestamp"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Files    []*File   `json:"files"`

	Public      bool          `json:"public"`
	Permissions []*Permission `json:"permissions"`
}

type Acquisition struct {
	Id        string `json:"_id"`
	Name      string `json:"label"`
	SessionId string `json:"session"`

	Instrument  string    `json:"instrument"`
	Measurement string    `json:"measurement"`
	Timestamp   time.Time `json:"timestamp"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Files    []*File   `json:"files"`

	Public      bool          `json:"public"`
	Permissions []*Permission `json:"permissions"`
}

type Client struct {
	C *http.Client
	S *sling.Sling
}

func NewApiKeyClient(host, key string, insecureSkipVerify bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}

	kt := &ApiKeyTransport{
		Key:       key,
		Transport: tr,
	}

	hc := kt.Client()

	sc := sling.New().
		Base("https://" + host + "/").Path("api/").
		Client(hc)

	return &Client{
		C: hc,
		S: sc,
	}
}

// coalesce will extract an API error message into a golang error, if applicable.
func coalesce(err error, aerr *ApiError) error {
	if err != nil {
		return err
	} else if aerr != nil {
		return errors.New(aerr.Message)
	} else {
		return nil
	}
}

func (c *Client) GetCurrentUser() (*User, *http.Response, error) {
	var aerr *ApiError
	var user *User
	resp, err := c.S.New().Get("users/self").Receive(&user, &aerr)
	return user, resp, coalesce(err, aerr)
}

func (c *Client) GetUsers() ([]*User, *http.Response, error) {
	var aerr *ApiError
	var users []*User
	resp, err := c.S.New().Get("users").Receive(&users, &aerr)
	return users, resp, coalesce(err, aerr)
}

func (c *Client) GetGroups() ([]*Group, *http.Response, error) {
	var aerr *ApiError
	var groups []*Group
	resp, err := c.S.New().Get("groups").Receive(&groups, &aerr)
	return groups, resp, coalesce(err, aerr)
}

func (c *Client) GetProjects() ([]*Project, *http.Response, error) {
	var aerr *ApiError
	var projects []*Project
	resp, err := c.S.New().Get("projects").Receive(&projects, &aerr)
	return projects, resp, coalesce(err, aerr)
}

func (c *Client) GetSessions() ([]*Session, *http.Response, error) {
	var aerr *ApiError
	var sessions []*Session
	resp, err := c.S.New().Get("sessions").Receive(&sessions, &aerr)
	return sessions, resp, coalesce(err, aerr)
}

func (c *Client) GetAcquisitions() ([]*Acquisition, *http.Response, error) {
	var aerr *ApiError
	var acqs []*Acquisition
	resp, err := c.S.New().Get("acquisitions").Receive(&acqs, &aerr)
	return acqs, resp, coalesce(err, aerr)
}

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
