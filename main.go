package main

import (
	"github.com/gillesdemey/go-dicom"
	prompt "github.com/segmentio/go-prompt"

	humanize "github.com/dustin/go-humanize"

	"io/ioutil"
	"io"
	"os"
	fp "path/filepath"
	"strings"
	"sync"
    "fmt"
	"archive/zip"
	"errors"
	"flag"
	"flywheel.io/sdk/api"
)


var (
	folder = flag.String("folder", "", "Folder with DICOM images to extract")
	group_id = flag.String("group", "", "Group Id")
	project_label = flag.String("project", "", "Flywheel project to upload files to")
	api_key = flag.String("api", "", "API key to login")
)

type DicomPath struct {
	dicom.DicomFile
	Path string
}

type Acquisition struct {
	SdkAcquisition api.Acquisition
	Files []DicomPath
}

type Session struct {
	SdkSession api.Session
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

// TODO: fill this out :)
func dicomScan(client *api.Client, folder string, group_id string, project_label string) error {

	return errors.New("Not implemented")
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


	// folder input, find .dcm files
	all_files := make([]DicomPath,0)
	sessions := make(map[string]Session)
	fmt.Println("Collecting Files...")
	err = fp.Walk(*folder, fileWalker_project(&all_files))
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
		whatever, dicoms_found, "images\n")
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

// make the dicom date and time fields somewhat readable when used as the container label
func parsable_time(dicom_time string) string {
	time_array := strings.Split(dicom_time, "")
	time := fmt.Sprintf("%s-%s-%s %s:%s:%s", strings.Join(time_array[:4],""), strings.Join(time_array[4:6],""), strings.Join(time_array[6:8],""), strings.Join(time_array[8:10],""), strings.Join(time_array[10:12],""), strings.Join(time_array[12:],""))
	return time
}

// determine the name of a session, acquisition, or file
// takes study or series as argument because then it's easier to find date and time of the dicom
func determine_name(file DicomPath, level string) (string, error) {
	POSSIBLE_NAMES := map[string]([]string){
		"Study" : []string{
			"StudyDescription",
			"StudyDate", // Will need to do extra black magic for datetime
			"StudyInstanceUID",
		},
		"Series" : []string{
			"SeriesDescription",
			"SeriesDate", // Same as for Sessions
			"SeriesInstanceUID",
		},
		"File" : []string{
			"SeriesDescription",
			"SeriesId",
		},
	}
	var err error
	for attempt, tag := range POSSIBLE_NAMES[level] {
		name, err := extract_value(file, tag)
		if err == nil {
			if attempt == 1 && level != "File" {
				name2, err := extract_value(file, level + "Time")
				if err == nil {
					return parsable_time(name+name2), nil
				}
			} else {
				return name, nil
			}
		}
	}
	return "", err
}

// simple function to deal with only needing values of dicom elements
func extract_value(file DicomPath, lookup_string string) (string, error) {
	el, err := file.LookupElement(lookup_string)
	if err != nil {
		return "", err
	}
	return el.Value[0].(string), err
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

// uploads dicoms as zips at the acquisition level, creating the needed containers as it goes
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
			meta_string := fmt.Sprintf("{\"group\": {\"_id\": \"%s\"},\"project\": {\"label\": \"%s\"},\"session\": {\"uid\": \"%s\",\"label\": \"%s\",\"subject\": {\"code\": \"%s\"}},\"acquisition\": {\"uid\": \"%s\",\"label\": \"%s\",\"files\": [{\"name\": \"%s\"}]}}",
			*group_id, *project_label, sdk_session.Uid, sdk_session.Name, sdk_session.Subject.Code, sdk_acquisition.Uid, sdk_acquisition.Name, file_name)
			metadata := []byte(meta_string)
			src := &api.UploadSource{Name: file_name, Path: file_path}
			prog, errc := c.UploadSimple("upload/uid", metadata, src)

			for update := range prog {
				fmt.Println("  Uploaded", humanize.Bytes(uint64(update)))
			}

			err = <-errc
			if err != nil {
				panic(err)
			}
			fmt.Println("Uploaded",file_name)
		}
	}
	err = os.RemoveAll(tmp)
	if err != nil {
		panic(err)
	}
	fmt.Println("\nUpload Complete\n")
	return nil
}

// sorts dicoms by study instance uid and series instance uid (session, acquisition)
func sort_dicoms(sessions map[string]Session, files *[]DicomPath) error {
	fmt.Println("\nSorting ...")
	for _,file := range *files{
		session_name, nerr := determine_name(file, "Study")
		if nerr != nil {
			files_skipped++
		}
		acquisition_name, nerr := determine_name(file, "Series")
		if nerr != nil {
			files_skipped++
		}
		StudyInstanceUID,_ := extract_value(file, "StudyInstanceUID")
		SeriesInstanceUID,_ := extract_value(file, "SeriesInstanceUID")
		StudyInstanceUID = strings.Replace(StudyInstanceUID, ".", "", -1)
		SeriesInstanceUID = strings.Replace(SeriesInstanceUID, ".", "", -1)
		if nerr == nil {
			if session, ok := sessions[StudyInstanceUID]; ok {
				if acquisition, ok := session.Acquisitions[SeriesInstanceUID]; ok {
					session.Acquisitions[SeriesInstanceUID].Files = append(acquisition.Files, file)
					dicoms_found++
				} else {
					sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid:SeriesInstanceUID}
					new_acq := Acquisition{Files: make([]DicomPath,0), SdkAcquisition: sdk_acquisition}
					session.Acquisitions[SeriesInstanceUID] = &new_acq
					session.Acquisitions[SeriesInstanceUID].Files = append(new_acq.Files,file)
					acquisitions_found++
					dicoms_found++
				}
			} else {
				subject_code, err := extract_value(file, "PatientID")
				if err != nil {
					fmt.Println("No subject code for sesion", session_name)
				}
				sdk_subject := api.Subject{Code: subject_code}
				sdk_session := api.Session{Subject: &sdk_subject, Name: session_name, Uid: StudyInstanceUID}
				sess := Session{SdkSession:sdk_session, Acquisitions: make(map[string]*Acquisition, 0)}
				sdk_acquisition := api.Acquisition{Name: acquisition_name, Uid: SeriesInstanceUID}
				new_acq := Acquisition{Files: make([]DicomPath,0), SdkAcquisition: sdk_acquisition}
				new_acq.Files = append(new_acq.Files,file)
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

func dir_step(files *[]DicomPath, path string, info os.FileInfo, err error) error {
	// Recurse into directories
	if info.IsDir() {
		fmt.Println("\tFrom", path)
		if strings.HasSuffix(path, info.Name()) {
			return nil
		}
		return fp.Walk(path+"/"+info.Name(), fileWalker_project(files))
	}
	// ignore if not a DICOM file
	// although later on figure out how to implement attachments
	if fp.Ext(info.Name()) != ".dcm" {
		return nil
	}
	*files = append(*files, processFile(path))
	return err
}

//wrapper for the actual fileWalker function so that it can collect files in an array
func fileWalker_project(files *[]DicomPath) func(string, os.FileInfo, error) error {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}
		return dir_step(files, path, info, err)
	}
}

// processes files, courtesy of the library with some modifications
func processFile(path string) DicomPath {

	buff, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	done := new(sync.WaitGroup)
	done.Add(1)
	dcm := &DicomPath{}
	// fmt.Println(project)

	go func() {
		gw := new(sync.WaitGroup)

		// parser
		ppln := dcm.Parse(buff)



		dcm.Discard(ppln, gw)
		gw.Wait()
		done.Done()
	}()
	done.Wait()
	// _,_ := dcm.LookupElement("StudyInstanceUID")
	// fmt.Println(el.Value)
	dcm.Path = path
	return *dcm
}
