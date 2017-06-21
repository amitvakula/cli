package legacy

import (
	"flywheel.io/sdk/api"
)

type ContainerTicketRequestElem struct {
	Level string `json:"level"`
	Id    string `json:"_id"`
}

type ContainerTicketRequest struct {
	Nodes    []*ContainerTicketRequestElem `json:"nodes"`
	Optional bool                          `json:"optional"`
}

type ContainerTicketResponse struct {
	Ticket    string `json:"ticket"`
	FileCount int    `json:"file_cnt"`
	Size      uint64 `json:"size"` // a 32-bit integer of bytes is only ~4GB. Let's be specific.
}

type Container interface {
	GetType() string
	GetId() string
}

var _ Container = &User{}
var _ Container = &File{}
var _ Container = &Group{}
var _ Container = &Project{}
var _ Container = &Session{}
var _ Container = &Acquisition{}

type User api.User

func (u *User) GetType() string {
	return "user"
}
func (u *User) GetId() string {
	return u.Id
}

type File api.File

func (u *File) GetType() string {
	return "file"
}
func (u *File) GetId() string {
	return u.Name
}

type Group api.Group

func (u *Group) GetType() string {
	return "group"
}
func (u *Group) GetId() string {
	return u.Id
}

type Project api.Project

func (u *Project) GetType() string {
	return "project"
}
func (u *Project) GetId() string {
	return u.Id
}

type Session api.Session

func (u *Session) GetType() string {
	return "session"
}
func (u *Session) GetId() string {
	return u.Id
}

type Acquisition api.Acquisition

func (u *Acquisition) GetType() string {
	return "acquisition"
}
func (u *Acquisition) GetId() string {
	return u.Id
}
