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
	related_acq   = flag.Bool("related", false, "Group related Series into one Acquisition, not working")
	log_level     = flag.Int("log", 0, "Amount of output on stdout [0, 1, 2]")
)

type DicomZip struct {
	Files 			[]dicom.DicomFile
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
func dicomScan(client *api.Client, folder string, group_id string, project_label string, related_acq bool, log_level int) error {
	// check that user has permission to group
	err := check_group_perms(client, group_id)
	if err != nil {
		return err
	}

	sessions := make(map[string]Session)
	fmt.Println("Collecting Files...")
	all_files := make([]dicom.DicomFile, 0)

	err = fp.Walk(folder, fileWalker(&all_files, log_level))
	if err != nil {
		return err
	}
	err = sort_dicoms(sessions, &all_files, related_acq)
	if err != nil {
		return err
	}

	if log_level>0 {
		printTree(sessions, group_id, project_label)
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
		return nil
	}
	fmt.Println("Beginning upload.")
	fmt.Println()

	err = upload_dicoms(sessions, client, related_acq, log_level)
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

	err = fp.Walk(*folder, fileWalker(&all_files, *log_level))
	if err != nil {
		panic(err)
	}

	err = sort_dicoms(sessions, &all_files, *related_acq)
	if err != nil {
		panic(err)
	}


	if *log_level>0 {
		printTree(sessions, *group_id, *project_label)
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

	err = upload_dicoms(sessions, client, *related_acq, *log_level)
	if err != nil {
		panic(err)
	}

}

func printTree(sessions map[string]Session, group_id string, project_label string) {
	fmt.Printf("\nDerived hierarchy\n")
	fmt.Printf("%s\n\t%s\n", project_label, group_id)
	for _, session := range sessions {
		fmt.Printf("\t\t%s >>> %s\n", session.SdkSession.Name, session.SdkSession.Subject.Code)
		for _, acq := range session.Acquisitions {
			fmt.Printf("\t\t\t%s\n", acq.SdkAcquisition.Name)
		}
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
func extract_value(file dicom.DicomFile, lookup_string string) (string, error) {
	el, err := file.LookupElement(lookup_string)
	if err != nil {
		return "", err
	}
	return el.StringData(), err
}

// Found online at https://golangcode.com/create-zip-files-in-go/
func ZipFiles(filename string, dir_path string) error {
	dir, err := os.Open(dir_path)
	if err != nil {
		return err
	}
	filenames, err := dir.Readdirnames(0)
	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	// Add files to zip
	for _, file_name := range filenames {
		file := dir_path + "/" + file_name
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
func upload_dicoms(sessions map[string]Session, c *api.Client, related_acq bool, log_level int) error {
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
			err = ZipFiles(file_path, "tempDir/"+sdk_acquisition.Uid)
			if err != nil {
				fmt.Println("Failed to compress files for acqusition "+ sdk_acquisition.Name)
				return err
			}


			if related_acq {
				// fmt.Println("Related is true")
				RelatedInstanceUID, err := extract_value(acquisition.Files[0], "RelatedSeriesSequence")
				if err == nil {
					// for _, ele := range acquisition.Files[0].Elements {
					// 	fmt.Printf("%s\t%s\t%s\n", ele.TagStr, ele.Name, ele.Data)
					// }
					// fmt.Println("Has RelatedInstanceUID")
					if acq, ok := session.Acquisitions[RelatedInstanceUID]; ok {
						// fmt.Println("Here")
						sdk_acquisition.Name = acq.SdkAcquisition.Name
						sdk_acquisition.Uid = acq.SdkAcquisition.Uid
					}
				}
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
				if log_level > 1 {
					fmt.Println("  Uploaded", humanize.Bytes(uint64(update)))
				}
			}

			err = <-errc
			if err != nil {
				return err
			}
			if log_level > 0 {
				fmt.Println("Uploaded", file_name)
			}
		}
	}
	err = os.RemoveAll(tmp)
	if err != nil {
		fmt.Println("Failed to remove Directory")
		return err
	}
	err = os.RemoveAll("tempDir")
	fmt.Println("\nUpload Complete\n")
	return nil
}

// sorts dicoms by study instance uid and series instance uid (session, acquisition)
func sort_dicoms(sessions map[string]Session, files *[]dicom.DicomFile, related_acq bool) error {
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
		os.Mkdir("tempDir", 0777)

		if nerr == nil {
			if session, ok := sessions[StudyInstanceUID]; ok {
				// Session and Acqusition already in the map
				if acquisition, ok := session.Acquisitions[SeriesInstanceUID]; ok {
					session.Acquisitions[SeriesInstanceUID].Files = append(acquisition.Files, file)
					// Create file in its acqusition folder
					os.Symlink(file.Path, "tempDir/" + SeriesInstanceUID + "/" + SOPInstanceUID)
					dicoms_found++
					// Session in the map but no acquisition yet
				} else {
					sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
					new_acq := Acquisition{Files: make([]dicom.DicomFile, 0), SdkAcquisition: sdk_acquisition}
					session.Acquisitions[SeriesInstanceUID] = &new_acq
					session.Acquisitions[SeriesInstanceUID].Files = append(new_acq.Files, file)

					// Create Acquisition folder and file in it
					os.MkdirAll("tempDir/" + SeriesInstanceUID, 0777)
					os.Symlink(file.Path, "tempDir/" + SeriesInstanceUID + "/" + SOPInstanceUID)

					acquisitions_found++
					dicoms_found++
				}
				// Neither Session nor Acquisition is in the map
			} else {
				subject_code, err := extract_value(file, "PatientID")
				if err != nil {
					fmt.Println("No subject code for sesion", session_name)
				}
				// Create Session and Subject
				sdk_subject := api.Subject{Code: subject_code}
				sdk_session := api.Session{Subject: &sdk_subject, Name: session_name, Uid: StudyInstanceUID}
				sess := Session{SdkSession: sdk_session, Acquisitions: make(map[string]*Acquisition, 0)}

				// Create Acquisition
				sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
				new_acq := Acquisition{Files: make([]dicom.DicomFile, 0), SdkAcquisition: sdk_acquisition}
				new_acq.Files = append(new_acq.Files, file)
				sess.Acquisitions[SeriesInstanceUID] = &new_acq
				sessions[StudyInstanceUID] = sess

				// Create Appropriate folders and place file in it
				os.MkdirAll("tempDir/" + SeriesInstanceUID, 0777)
				os.Symlink(file.Path, "tempDir/" + SeriesInstanceUID + "/" + SOPInstanceUID)

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

func fileWalker(files *[]dicom.DicomFile, log_level int) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		// don't parse nested directories
		if info.IsDir() && log_level > 0 {
			fmt.Println("\tFrom", path)
		} else if ! info.IsDir() {
			file, err := processFile(path)
			if err != nil {
				// not a DICOM file
				if err == dicom.ErrNotDICM || err == io.EOF {
					return nil
				}
				panic(err)
			}
			*files = append(*files, file)
		}

		return err
	}
}

func processFile(path string) (dicom.DicomFile, error) {
	tags := []string{
		"0020000D", // Study Instance UID
		"0020000E", // Series Instance UID
		"00080018", // SOP Instance UID
		"00080020", // Study Date
		"00080021", // Series Date
		"00080030", // Study Time
		"00080031", // Series Time
		"00081030", // Study Description
		"0008103E", // Series Description
		// "00081250", // Related Series Sequence
		"00100020", // Patient ID
	}
	// tags = make([]string,0)
	di := dicom.DicomFile{}
	// bytes, err := ioutil.ReadFile(path)
	f, err := os.Open(path)
	if err != nil {
		return di, err
	}
	bytes := make([]byte, 132)
	_, err = f.Read(bytes)
	// fmt.Println(n1)
	if err != nil {
		// fmt.Fprintf(os.Stderr, "ERROR: failed to read file: '%s'\n", err)
		return di, err
	}
	f.Close()
	// fmt.Printf("Path: %s\tLength: %d\n", path, len(bytes))

	explicit := true
	di.Path = path
	if string(bytes[128:]) == "DICM" {
		err = di.ProcessFile(path, 132, explicit, tags)
		return di, err
	} else {
		files_skipped++
		return di, dicom.ErrNotDICM
	}
}
