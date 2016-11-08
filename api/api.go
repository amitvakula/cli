package api

import (
	"time"
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

type Gear struct {
	Name string `json:"name"`

	Category string                 `json:"category"`
	Input    map[string]interface{} `json:"input"`

	Manifest map[string]interface{} `json:"manifest"`
}
