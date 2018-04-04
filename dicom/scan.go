package dicom

import (
	dicom "github.com/grailbio/go-dicom"
	tag "github.com/grailbio/go-dicom/dicomtag"
	prompt "github.com/segmentio/go-prompt"

	humanize "github.com/dustin/go-humanize"
	fp "path/filepath"

	"archive/zip"
	"encoding/json"
	"flywheel.io/sdk/api"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	. "flywheel.io/fw/util"
)

type DicomFile struct {
	*dicom.DataSet
	Path string
}

type Acquisition struct {
	SdkAcquisition api.Acquisition
	Files          map[string]DicomFile // Key is SOP Instance UID
}

type Session struct {
	SdkSession   api.Session
	Acquisitions map[string]*Acquisition // Key is Series Instance UID
}

var sessions_found = 0
var acquisitions_found = 0
var dicoms_found = 0
var sessions_uploaded = 0
var acquisitions_uploaded = 0
var files_skipped = 0

// TODO: check for group permissions before scanning

// replace panics with {return err}

func Scan(client *api.Client, folder string, group_id string, project_label string, related_acq bool, quiet, noTree bool, local bool) {
	// check that user has permission to group
	group_label, err := check_group_perms(client, group_id)
	Check(err)

	// Check if user gave project id as input ex. <id:43f34f8439fh34f>
	r := regexp.MustCompile(`^\<id:([\da-z]+)\>$`)
	matches := r.FindStringSubmatch(project_label)
	if len(matches) == 2 {
		project, _, err := client.GetProject(matches[1])
		Check(err)
		project_label = project.Name
	}

	sessions := make(map[string]Session)
	fmt.Println("Collecting Files...")
	all_files := make([]DicomFile, 0)

	err = fp.Walk(folder, fileWalker(&all_files, quiet))
	Check(err)
	err = sort_dicoms(sessions, &all_files, related_acq)
	Check(err)

	if !noTree {
		printTree(sessions, group_label, project_label)
	}

	// Save hierarchy if location flag set to valid directory
	if local {
		Check(save_hierarchy(client, sessions, group_label, project_label))
	}

	// Summary of what is to be uploaded
	whatever := "                     "
	fmt.Println("This scan consists of:\n",
		whatever, sessions_found, "sessions,\n",
		whatever, acquisitions_found, "acquisitions,\n",
		whatever, files_skipped, "files skipped\n")
	proceed := prompt.Confirm("Confirm upload? (yes/no)")
	fmt.Println()
	if !proceed {
		fmt.Println("Canceled.")
		return
	}
	fmt.Println("Beginning upload.")
	fmt.Println()

	err = upload_dicoms(sessions, client, related_acq, group_id, project_label, quiet)
	Check(err)
}

func copy_file(file_path string, copy_path string) error {
	file, err := os.Open(file_path)
	if err != nil {
		return err
	}
	copy, err := os.Create(copy_path)
	if err != nil {
		return err
	}
	_, err = io.Copy(copy, file)
	if err != nil {
		return err
	}
	file.Close()
	copy.Close()
	return nil
}

func save_hierarchy(client *api.Client, sessions map[string]Session, group_label string, project_label string) error {
	fmt.Println("Creating tree locally...")
	var perm os.FileMode = 0777

	base_path := group_label + "/" + project_label
	err := os.MkdirAll(base_path, perm)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		path := base_path + "/" + session.SdkSession.Subject.Code + "/" + session.SdkSession.Name
		for _, acq := range session.Acquisitions {
			err = os.MkdirAll(path+"/"+acq.SdkAcquisition.Name, perm)
			if err != nil {
				return err
			}
			for sop, file := range acq.Files {
				err = copy_file(file.Path, path+"/"+acq.SdkAcquisition.Name+"/"+sop)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func printTree(sessions map[string]Session, group_label string, project_label string) {
	fmt.Printf("\nDerived hierarchy\n")
	fmt.Printf("%s\n\t%s\n", group_label, project_label)
	for _, session := range sessions {
		fmt.Printf("\t\t%s >>> %s\n", session.SdkSession.Name, session.SdkSession.Subject.Code)
		for _, acq := range session.Acquisitions {
			fmt.Printf("\t\t\t%s\n", acq.SdkAcquisition.Name)
		}
	}
}

func check_group_perms(client *api.Client, group_id string) (string, error) {
	group, _, err := client.GetGroup(group_id)
	return group.Name, err
}

// make the dicom date and time fields somewhat readable when used as the container label
func parsable_time(dicom_time string) string {
	time_array := strings.Split(dicom_time, "")
	time := fmt.Sprintf("%s-%s-%s %s:%s:%s", strings.Join(time_array[:4], ""), strings.Join(time_array[4:6], ""), strings.Join(time_array[6:8], ""), strings.Join(time_array[8:10], ""), strings.Join(time_array[10:12], ""), strings.Join(time_array[12:], ""))
	return time
}

// determine the name of a session, acquisition, or file
// takes study or series as argument because then it's easier to find date and time of the dicom
func determine_name(file DicomFile, level string) (string, error) {
	POSSIBLE_NAMES := map[string]([]string){
		"Study": []string{
			"StudyDescription",
			"StudyDate", // Will need to do extra black magic for datetime
			"StudyInstanceUID",
		},
		"Series": []string{
			"SeriesDescription",
			"SeriesDate", // Same as for Sessions
			"SeriesInstanceUID",
		},
		"File": []string{
			"SeriesDescription",
			"SeriesInstanceUID",
		},
	}
	var err error
	for attempt, tag := range POSSIBLE_NAMES[level] {
		name, err := extract_value(file, tag)
		if err == nil {
			if attempt == 1 && level != "File" {
				name2, err := extract_value(file, level+"Time")
				if err == nil {
					return parsable_time(name + name2), nil
				}
			} else {
				return name, nil
			}
		}
	}
	return "", err
}

// simple function to deal with only needing values of dicom elements
func extract_value(file DicomFile, lookup_string string) (string, error) {
	el, err := file.FindElementByName(lookup_string)
	if err != nil {
		return "", err
	}
	s, err := el.GetString()
	return s, err
}

// Found online at https://golangcode.com/create-zip-files-in-go/
func ZipFiles(newfile io.Writer, acq *Acquisition) error {
	filenames := make([]string, 0, len(acq.Files))
	for _, di := range acq.Files {
		filenames = append(filenames, di.Path)
	}

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range filenames {
		zipfile, err := os.Open(file)
		if err != nil {
			return err
		}

		// Get the file information
		info, err := zipfile.Stat()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Change to deflate to gain better compression
		// see http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, zipfile)
		if err != nil {
			return err
		}
		zipfile.Close()
	}
	return nil
}

// uploads dicoms as zips at the acquisition level, uses upload/uid endpoint
func upload_dicoms(sessions map[string]Session, c *api.Client, related_acq bool, group_id string, project_label string, quiet bool) error {

	for _, session := range sessions {
		sdk_session := session.SdkSession
		sessions_uploaded++

		for _, acquisition := range session.Acquisitions {
			sdk_acquisition := acquisition.SdkAcquisition
			acquisitions_uploaded++
			paths := make([]string, 0)
			for _, file := range acquisition.Files {
				paths = append(paths, file.Path)
			}
			file_name := strings.TrimRight(sdk_acquisition.Name, " ") + ".dcm.zip"

			metadata := map[string]interface{}{
				"group": map[string]interface{}{
					"_id": group_id,
				},
				"project": map[string]interface{}{
					"label": project_label,
				},
				"session": map[string]interface{}{
					"uid":   sdk_session.Uid,
					"label": sdk_session.Name,
					"subject": map[string]interface{}{
						"code": sdk_session.Subject.Code,
					},
				},
				"acquisition": map[string]interface{}{
					"uid":   sdk_acquisition.Uid,
					"label": sdk_acquisition.Name,
					"files": []interface{}{
						map[string]interface{}{
							"name": file_name,
						},
					},
				},
			}

			metadata_bytes, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			uploadfile, newfile := io.Pipe()
			go func() {
				defer newfile.Close()
				err = ZipFiles(newfile, acquisition)
				if err != nil {
					fmt.Println("Failed to compress files for acqusition " + sdk_acquisition.Name)
					Check(err)
				}
			}()
			src := &api.UploadSource{Name: file_name, Reader: uploadfile}
			prog, errc := c.UploadSimple("upload/uid", metadata_bytes, src)

			for update := range prog {
				if !quiet {
					fmt.Println("  Uploaded", humanize.Bytes(uint64(update)))
				}
			}

			err = <-errc
			if err != nil {
				return err
			}

			fmt.Println("Uploaded", file_name)
		}
	}
	fmt.Println("\nUpload Complete\n")
	return nil
}

// sorts dicoms by study instance uid and series instance uid (session, acquisition)
func sort_dicoms(sessions map[string]Session, files *[]DicomFile, related_acq bool) error {
	fmt.Println("\nSorting ...")

	for _, file := range *files {
		session_name, nerr := determine_name(file, "Study")
		acquisition_name, nerr := determine_name(file, "Series")
		StudyInstanceUID, _ := extract_value(file, "StudyInstanceUID")
		SeriesInstanceUID, _ := extract_value(file, "SeriesInstanceUID")
		SOPInstanceUID, _ := extract_value(file, "SOPInstanceUID")
		// Api expects uid without dots
		StudyInstanceUID = strings.Replace(StudyInstanceUID, ".", "", -1)
		SeriesInstanceUID = strings.Replace(SeriesInstanceUID, ".", "", -1)

		if nerr == nil {
			if session, ok := sessions[StudyInstanceUID]; ok {
				// Session and Acqusition already in the map
				if _, ok := session.Acquisitions[SeriesInstanceUID]; ok {
					session.Acquisitions[SeriesInstanceUID].Files[SOPInstanceUID] = file
					dicoms_found++
					// Session in the map but no acquisition yet
				} else {
					sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
					new_acq := Acquisition{Files: make(map[string]DicomFile, 0), SdkAcquisition: sdk_acquisition}
					session.Acquisitions[SeriesInstanceUID] = &new_acq
					session.Acquisitions[SeriesInstanceUID].Files[SOPInstanceUID] = file

					acquisitions_found++
					dicoms_found++
				}
				// Neither Session nor Acquisition is in the map
			} else {
				subject_code, err := extract_value(file, "PatientID")
				if err != nil {
					fmt.Println("No subject code for session", session_name)
				}
				// Create Session and Subject
				sdk_subject := api.Subject{Code: subject_code}
				sdk_session := api.Session{Subject: &sdk_subject, Name: session_name, Uid: StudyInstanceUID}
				sess := Session{SdkSession: sdk_session, Acquisitions: make(map[string]*Acquisition, 0)}

				// Create Acquisition
				sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
				new_acq := Acquisition{Files: make(map[string]DicomFile, 0), SdkAcquisition: sdk_acquisition}
				new_acq.Files[SOPInstanceUID] = file
				sess.Acquisitions[SeriesInstanceUID] = &new_acq
				sessions[StudyInstanceUID] = sess

				sessions_found++
				acquisitions_found++
				dicoms_found++
			}
		}
	}
	fmt.Println("Sorting Complete\n")
	return nil
}

func fileWalker(files *[]DicomFile, quiet bool) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// don't parse nested directories
		if info.IsDir() && !quiet {
			fmt.Println("\tFrom", path)
		} else if !info.IsDir() {
			file, err := processFile(path)
			if err != nil {
				if err.Error() == "Keyword 'DICM' not found in the header" || err.Error() == "Skip: requested 128, available 0" {
					// not a DICOM file
					files_skipped++
					return nil
				} else {
					Check(err)
					return err
				}
			}
			*files = append(*files, file)
		}
		return err
	}
}

func processFile(path string) (DicomFile, error) {
	var err error
	// tags = make([]string,0)
	di := &DicomFile{}
	tagList := []tag.Tag{
		tag.SeriesInstanceUID,
		tag.StudyInstanceUID,
		tag.StudyDate,
		tag.StudyTime,
		tag.SeriesDate,
		tag.SeriesTime,
		tag.StudyDescription,
		tag.SeriesDescription,
		tag.PatientID,
		tag.SOPInstanceUID,
	}
	di.DataSet, err = dicom.ReadDataSetFromFile(path, dicom.ReadOptions{DropPixelData: true, StopAtTag: &tag.StackID, ReturnTags: tagList})
	di.Path = path
	return *di, err
}
