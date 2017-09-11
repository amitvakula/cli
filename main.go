package main

import (
	dicom "github.com/DavidGamba/go-dicom/dcmdump"
	prompt "github.com/segmentio/go-prompt"

	humanize "github.com/dustin/go-humanize"
	fp "path/filepath"

	"archive/zip"
	"encoding/json"
	"errors"
	"flag"
	"flywheel.io/sdk/api"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

var (
	folder        = flag.String("folder", "", "Folder with DICOM images to extract")
	group_id      = flag.String("group", "", "Group Id")
	project_label = flag.String("project", "", "Flywheel project to upload files to")
	api_key       = flag.String("api", "", "API key to login")
)

type DicomPath struct {
	dicom.DicomFile
	Path string
}

type Acquisition struct {
	SdkAcquisition api.Acquisition
	Files          []dicom.DicomFile
}

type Session struct {
	SdkSession   api.Session
	Acquisitions map[string]*Acquisition
}

var sessions_found = 0
var acquisitions_found = 0
var dicoms_found = 0
var sessions_uploaded = 0
var acquisitions_uploaded = 0
var files_skipped = 0

func init() {
	flag.Parse()
}

// TODO: check for group permissions before scanning

// replace panics with {return err}

// To be used by cli
func dicomScan(client *api.Client, folder string, group_id string, project_label string) error {
	// check that user has permission to group
	err := check_group_perms(client, group_id)
	if err != nil {
		return err
	}

	sessions := make(map[string]Session)
	fmt.Println("Collecting Files...")
	all_files := make([]dicom.DicomFile, 0)

	err = fp.Walk(folder, fileWalker(&all_files))
	if err != nil {
		return err
	}
	err = sort_dicoms(sessions, &all_files)
	if err != nil {
		return err
	}

	// Summary of what is to be uploaded
	whatever := "                     "
	fmt.Println("This scan consists of:\n",
		whatever, sessions_found, "sessions,\n",
		whatever, acquisitions_found, "acquisitions,\n",
		whatever, dicoms_found, "images\n")
	proceed := prompt.Confirm("Confirm upload? (yes/no)")
	fmt.Println()
	if !proceed {
		fmt.Println("Canceled.")
		return nil
	}
	fmt.Println("Beginning upload.")
	fmt.Println()

	err = upload_dicoms(sessions, client)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	client := api.NewApiKeyClient(*api_key)

	if project_label == nil {
		panic(errors.New("No project label given, use -project flag."))
	}

	if group_id == nil {
		panic(errors.New("No group_id given, use -group flag."))
	}

	if folder == nil {
		panic(errors.New("No folder given, use -folder flag."))
	}

	if api_key == nil {
		panic(errors.New("No api_key given, use -api flag."))
	}

	// check if api key is valid
	_, _, err := client.GetCurrentUser()
	if err != nil {
		panic(err)
	}

	sessions := make(map[string]Session)
	fmt.Println("Collecting Files...")

	all_files := make([]dicom.DicomFile, 0)

	err = fp.Walk(*folder, fileWalker(&all_files))
	if err != nil {
		panic(err)
	}
	err = sort_dicoms(sessions, &all_files)
	if err != nil {
		panic(err)
	}

	// Summary of what is to be uploaded
	whatever := "                     "
	fmt.Println("This scan consists of:\n",
		whatever, sessions_found, "sessions,\n",
		whatever, acquisitions_found, "acquisitions,\n",
		whatever, dicoms_found, "images\n",
		whatever, files_skipped, "files skipped\n")
	proceed := prompt.Confirm("Confirm upload? (yes/no)")
	fmt.Println()
	if !proceed {
		fmt.Println("Canceled.")
		return
	}
	fmt.Println("Beginning upload.")
	fmt.Println()

	err = upload_dicoms(sessions, client)
	if err != nil {
		panic(err)
	}

}

func check_group_perms(client *api.Client, group_id string) error {
	_, _, err := client.GetGroup(group_id)
	return err
}

// make the dicom date and time fields somewhat readable when used as the container label
func parsable_time(dicom_time string) string {
	time_array := strings.Split(dicom_time, "")
	time := fmt.Sprintf("%s-%s-%s %s:%s:%s", strings.Join(time_array[:4], ""), strings.Join(time_array[4:6], ""), strings.Join(time_array[6:8], ""), strings.Join(time_array[8:10], ""), strings.Join(time_array[10:12], ""), strings.Join(time_array[12:], ""))
	return time
}

// determine the name of a session, acquisition, or file
// takes study or series as argument because then it's easier to find date and time of the dicom
func determine_name(file dicom.DicomFile, level string) (string, error) {
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
			"SeriesId",
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
func extract_value(file dicom.DicomFile, lookup_string string) (string, error) {
	el, err := file.LookupElement(lookup_string)
	if err != nil {
		return "", err
	}
	return el.StringData(), err
}

// Found online at https://golangcode.com/create-zip-files-in-go/
func ZipFiles(filename string, files []string) error {

	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {

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
func upload_dicoms(sessions map[string]Session, c *api.Client) error {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		return err
	}
	for _, session := range sessions {
		sdk_session := session.SdkSession
		sessions_uploaded++

		if err != nil {
			return err
		}
		for _, acquisition := range session.Acquisitions {
			sdk_acquisition := acquisition.SdkAcquisition
			acquisitions_uploaded++
			if err != nil {
				return err
			}
			paths := make([]string, 0)
			for _, file := range acquisition.Files {
				paths = append(paths, file.Path)
			}
			file_name := sdk_acquisition.Name + ".dcm.zip"
			file_path := tmp + "/" + file_name
			err = ZipFiles(file_path, paths)
			if err != nil {
				return err
			}

			metadata := map[string]interface{}{
				"group": map[string]interface{}{
					"_id": *group_id,
				},
				"project": map[string]interface{}{
					"label": *project_label,
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

			src := &api.UploadSource{Name: file_name, Path: file_path}
			prog, errc := c.UploadSimple("upload/uid", metadata_bytes, src)

			for update := range prog {
				fmt.Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			err = <-errc
			if err != nil {
				return err
			}
			fmt.Println("Uploaded", file_name)
		}
	}
	err = os.RemoveAll(tmp)
	if err != nil {
		return err
	}
	fmt.Println("\nUpload Complete\n")
	return nil
}

// sorts dicoms by study instance uid and series instance uid (session, acquisition)
func sort_dicoms(sessions map[string]Session, files *[]dicom.DicomFile) error {
	fmt.Println("\nSorting ...")
	for _, file := range *files {
		session_name, nerr := determine_name(file, "Study")
		acquisition_name, nerr := determine_name(file, "Series")
		StudyInstanceUID, _ := extract_value(file, "StudyInstanceUID")
		SeriesInstanceUID, _ := extract_value(file, "SeriesInstanceUID")
		// Api expects uid without dots
		StudyInstanceUID = strings.Replace(StudyInstanceUID, ".", "", -1)
		SeriesInstanceUID = strings.Replace(SeriesInstanceUID, ".", "", -1)

		if nerr == nil {
			if session, ok := sessions[StudyInstanceUID]; ok {
				// Session and Acqusition already in the map
				if acquisition, ok := session.Acquisitions[SeriesInstanceUID]; ok {
					session.Acquisitions[SeriesInstanceUID].Files = append(acquisition.Files, file)
					dicoms_found++
					// Session in the map but no acquisition yet
				} else {
					sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
					new_acq := Acquisition{Files: make([]dicom.DicomFile, 0), SdkAcquisition: sdk_acquisition}
					session.Acquisitions[SeriesInstanceUID] = &new_acq
					session.Acquisitions[SeriesInstanceUID].Files = append(new_acq.Files, file)
					acquisitions_found++
					dicoms_found++
				}
				// Neither Session nor Acquisition is in the map
			} else {
				subject_code, err := extract_value(file, "PatientID")
				if err != nil {
					fmt.Println("No subject code for sesion", session_name)
				}
				sdk_subject := api.Subject{Code: subject_code}
				sdk_session := api.Session{Subject: &sdk_subject, Name: session_name, Uid: StudyInstanceUID}
				sess := Session{SdkSession: sdk_session, Acquisitions: make(map[string]*Acquisition, 0)}
				sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
				new_acq := Acquisition{Files: make([]dicom.DicomFile, 0), SdkAcquisition: sdk_acquisition}
				new_acq.Files = append(new_acq.Files, file)
				sess.Acquisitions[SeriesInstanceUID] = &new_acq
				sessions[StudyInstanceUID] = sess
				sessions_found++
				acquisitions_found++
				dicoms_found++
			}
		}
	}
	fmt.Println("Sorting Complete\n")
	// Can't error if errors are always nil <- not ideal though
	return nil
}

func fileWalker(files *[]dicom.DicomFile) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		// don't parse nested directories
		if info.IsDir() {
			fmt.Println("\tFrom", path)
		} else {

			file, err := processFile(path)
			if err != nil {
				// not a DICOM file
				if err == dicom.ErrNotDICM {
					return nil
				}
				return err
			}
			*files = append(*files, file)
		}

		return err
	}
}
func processFile(path string) (dicom.DicomFile, error) {
	di := dicom.DicomFile{}
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "ERROR: failed to read file: '%s'\n", err)
		return di, err
	}

	// Intro
	n := 128
	// DICM
	m := n + 4

	explicit := true
	di.Path = path
	if string(bytes[n:m]) == "DICM" {
		di.ProcessFile(bytes, m, explicit)
		return di, nil
	} else {
		files_skipped++
		return di, dicom.ErrNotDICM
	}
}
