package main

import (
	. "fmt"
	"os"
)

func findPermission(id string, perms []*Permission) *Permission {
	for _, x := range perms {
		if x.Id == id {
			return x
		}
	}
	return &Permission{
		Id:    id,
		Level: "none",
	}
}

func findGroupById(groupId string, groups []*Group) *Group {
	for _, x := range groups {
		if x.Id == groupId {
			return x
		}
	}

	Println("No", groupId, "group found.")
	os.Exit(1)
	return nil
}

func findProjectByName(projectName string, projects []*Project) *Project {
	for _, x := range projects {
		if x.Name == projectName {
			return x
		}
	}

	Println("No", projectName, "project found.")
	os.Exit(1)
	return nil
}

func findSessionByName(sessionName string, sessions []*Session) *Session {
	for _, x := range sessions {
		if x.Name == sessionName {
			return x
		}
	}

	Println("No", sessionName, "session found.")
	os.Exit(1)
	return nil
}

func findAcquisitionByName(acqName string, acqs []*Acquisition) *Acquisition {
	for _, x := range acqs {
		if x.Name == acqName {
			return x
		}
	}

	Println("No", acqName, "session found.")
	os.Exit(1)
	return nil
}

func findFileByName(fileName string, files []*File) *File {
	for _, x := range files {
		if x.Name == fileName {
			return x
		}
	}

	Println("No", fileName, "file found.")
	os.Exit(1)
	return nil
}

func filterProjectsByGroupId(groupId string, projects []*Project) []*Project {
	var filtered []*Project

	for _, x := range projects {
		if x.GroupId == groupId {
			filtered = append(filtered, x)
		}
	}

	return filtered
}

func filterSessionsByProjectId(projectId string, groups []*Session) []*Session {
	var filtered []*Session

	for _, x := range groups {
		if x.ProjectId == projectId {
			filtered = append(filtered, x)
		}
	}

	return filtered
}

func filterAcquisitionsBySessionId(sessionId string, acqs []*Acquisition) []*Acquisition {
	var filtered []*Acquisition

	for _, x := range acqs {
		if x.SessionId == sessionId {
			filtered = append(filtered, x)
		}
	}

	return filtered
}
