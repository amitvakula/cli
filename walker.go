package main

import (
	. "fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
)

/*
func increment(step string, parent interface{}) interface{} {

	var a []interface{}

	switch parent := parent.(type) {
	case []*Group:
		a = filterGroupsById(step, parent)

	}

	if len(a) == 0 {
		Println("Could not find anything matching", step)
		os.Exit(1)
	}

	return
}*/

// resolve takes a human-meaningful slash-delimited path and attempts to walk the hierarchy, returning an array of discovered objects.
// Ambiguities
func resolve(client *Client, parts []string, path ...interface{}) []interface{} {

	// Handle initial and final conditions
	if len(path) == 0 {
		groups, _, err := client.GetGroups()
		check(err)
		return resolve(client, parts, groups)
	} else if len(parts) == 0 || parts[0] == "" {
		return path
	}

	base := path[len(path)-1]

	switch base := base.(type) {

	case []*Group:
		groupId := parts[0]
		group := findGroupById(groupId, base)
		projects, _, err := client.GetProjects()
		check(err)
		projects = filterProjectsByGroupId(groupId, projects)
		return resolve(client, parts[1:], group, projects)

	case []*Project:
		group := path[0]
		project := findProjectByName(parts[0], base)
		sessions, _, err := client.GetSessions()
		check(err)
		sessions = filterSessionsByProjectId(project.Id, sessions)
		return resolve(client, parts[1:], group, project, sessions)

	case []*Session:
		group := path[0]
		project := path[1]
		session := findSessionByName(parts[0], base)
		acqs, _, err := client.GetAcquisitions()
		check(err)
		acqs = filterAcquisitionsBySessionId(session.Id, acqs)
		return resolve(client, parts[1:], group, project, session, acqs)

	case []*Acquisition:
		group := path[0]
		project := path[1]
		session := path[2]
		acq := findAcquisitionByName(parts[0], base)
		return resolve(client, parts[1:], group, project, session, acq)

	case *Acquisition:
		group := path[0]
		project := path[1]
		session := path[2]
		acq := path[3]
		file := findFileByName(parts[0], base.Files)
		return resolve(client, parts[1:], group, project, session, acq, file)

	default:
		Printf("Error: resolving unexpected type %T\n", base)
		os.Exit(1)
	}
	return nil
}

var blueBold = color.New(color.FgBlue, color.Bold).SprintFunc()
var timeFormat = "Jan _2 15:04"

func print(results, parent interface{}, userId string, showDbIds bool) {

	// Format the table, printing to a platform- & pipe-friendly color writer
	w := tabwriter.NewWriter(color.Output, 0, 2, 1, ' ', 0)

	// Closure for printing object ID, if enabled
	printId := func(id string) {
		if showDbIds {
			Fprintf(w, "<id:%s>\t", id)
		}
	}

	// Closure for dedup
	printFiles := func(files []*File, perms []*Permission) {
		level := findPermission(userId, perms).Level
		for _, x := range files {
			printId(x.Name)
			Fprintf(w, "%s\t%s\t\t%s\t\n", level, x.Modified.Format(timeFormat), x.Name)
		}
	}

	switch results := results.(type) {
	case []*Group:
		for _, x := range results {
			level := findPermission(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\n", level, blueBold(x.Id), x.Name)
		}

	case []*Project:
		for _, x := range results {
			level := findPermission(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\n", level, blueBold(x.Name))
		}

	case []*Session:
		for _, x := range results {
			level := findPermission(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\n", level, x.Timestamp.Format(timeFormat), x.Subject.Name, blueBold(x.Name))
		}
		parent := parent.(*Project)
		printFiles(parent.Files, parent.Permissions)

	case []*Acquisition:
		for _, x := range results {
			level := findPermission(userId, x.Permissions).Level
			printId(x.Id)
			Fprintf(w, "%s\t%s\t%s\t%s\t\n", level, x.Timestamp.Format(timeFormat), x.Measurement, blueBold(x.Name))
		}
		parent := parent.(*Session)
		printFiles(parent.Files, parent.Permissions)

	case *Acquisition:
		printFiles(results.Files, results.Permissions)

	case *File:
		printFiles([]*File{results}, []*Permission{})

	default:
		Printf("Error: printing unexpected type %T\n", results)
		os.Exit(1)
	}

	w.Flush()
}
