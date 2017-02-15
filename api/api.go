package api

import (
	"time"
)

type ApiError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

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

func (u *User) GetType() string {
	return "user"
}
func (u *User) GetId() string {
	return u.Id
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

func (u *File) GetType() string {
	return "file"
}
func (u *File) GetId() string {
	return u.Name
}

type Group struct {
	Id   string `json:"_id"`
	Name string `json:"name"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`

	Permissions []*Permission `json:"roles"`
}

func (u *Group) GetType() string {
	return "group"
}
func (u *Group) GetId() string {
	return u.Id
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

func (u *Project) GetType() string {
	return "project"
}
func (u *Project) GetId() string {
	return u.Id
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

func (u *Session) GetType() string {
	return "session"
}
func (u *Session) GetId() string {
	return u.Id
}

type Acquisition struct {
	Id        string `json:"_id"`
	Name      string `json:"label"`
	SessionId string `json:"session"`

	Timestamp time.Time `json:"timestamp"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Files    []*File   `json:"files"`

	Public      bool          `json:"public"`
	Permissions []*Permission `json:"permissions"`
}

func (u *Acquisition) GetType() string {
	return "acquisition"
}
func (u *Acquisition) GetId() string {
	return u.Id
}

type Gear struct {
	Name string `json:"name"`

	Category string                 `json:"category"`
	Input    map[string]interface{} `json:"input"`

	Manifest map[string]interface{} `json:"manifest"`
}
