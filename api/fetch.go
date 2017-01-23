package api

import (
	"net/http"
)

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

func (c *Client) GetDownloadTicket(request *ContainerTicketRequest) (*ContainerTicketResponse, *http.Response, error) {

	var aerr *ApiError
	var ticket *ContainerTicketResponse

	resp, err := c.S.New().Post("download").BodyJSON(request).Receive(&ticket, &aerr)
	return ticket, resp, coalesce(err, aerr)
}
