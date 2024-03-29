package legacy

import (
	"flywheel.io/sdk/api"
)

type ContainerTicketRequestElem struct {
	Level string `json:"level"`
	Id    string `json:"_id"`
}

type ContainerTicketFilterElem struct {
	Include []string `json:"+,omitempty"`
	Exclude []string `json:"-,omitempty"`
}

type ContainerTicketFilter struct {
	Types *ContainerTicketFilterElem `json:"types"`
}

type ContainerTicketRequest struct {
	Nodes    []*ContainerTicketRequestElem `json:"nodes"`
	Filters  []*ContainerTicketFilter      `json:"filters,omitempty"`
	Optional bool                          `json:"optional"`
}

func NewContainerFilter(include []string, exclude []string) []*ContainerTicketFilter {
	return []*ContainerTicketFilter{
		{
			Types: &ContainerTicketFilterElem{
				Include: include,
				Exclude: exclude,
			},
		},
	}
}

type ContainerTicketResponse struct {
	Ticket    string `json:"ticket"`
	FileCount int    `json:"file_cnt"`
	Size      uint64 `json:"size"` // a 32-bit integer of bytes is only ~4GB. Let's be specific.
}

type Container interface {
	GetType() string
	GetId() string
	GetName() string
}

var _ Container = &User{}
var _ Container = &File{}
var _ Container = &Group{}
var _ Container = &Project{}
var _ Container = &Subject{}
var _ Container = &Session{}
var _ Container = &Acquisition{}

type User api.User

func (u *User) GetType() string {
	return "user"
}
func (u *User) GetId() string {
	return u.Id
}
func (u *User) GetName() string {
	return u.Id
}

type File api.File

func (u *File) GetType() string {
	return "file"
}
func (u *File) GetId() string {
	return u.Name
}
func (u *File) GetName() string {
	return u.Name
}

type Group api.Group

func (u *Group) GetType() string {
	return "group"
}
func (u *Group) GetId() string {
	return u.Id
}
func (u *Group) GetName() string {
	return u.Name
}

type Project api.Project

func (u *Project) GetType() string {
	return "project"
}
func (u *Project) GetId() string {
	return u.Id
}
func (u *Project) GetName() string {
	return u.Name
}

type Subject api.Subject

func (u *Subject) GetType() string {
	return "subject"
}
func (u *Subject) GetId() string {
	return u.Id
}
func (u *Subject) GetName() string {
	return u.Code
}

type Session api.Session

func (u *Session) GetType() string {
	return "session"
}
func (u *Session) GetId() string {
	return u.Id
}
func (u *Session) GetName() string {
	return u.Name
}
func (u *Session) GetSubjectCode() string {
	return u.Subject.Code
}

type Acquisition api.Acquisition

func (u *Acquisition) GetType() string {
	return "acquisition"
}
func (u *Acquisition) GetId() string {
	return u.Id
}
func (u *Acquisition) GetName() string {
	return u.Name
}
