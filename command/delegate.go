package command

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"flywheel.io/fw/python/pkgdata"
)

type CommandDesc struct {
	Command     string
	Description string
}

type PkgVersionInfo struct {
	PythonVersion string `json:"python_version",omitempty`
	PyVer         string `json:"py_ver",omitempty`
	BuildTime     string `json:"build_time",omitempty`
}

type PythonPathInfo struct {
	PythonBinDir    string
	SitePackagePath string
}

var PythonCliCommand = []string{"-m", "flywheel_cli.main"}

const CachePath = "~/.cache/flywheel/"
const LibcExec = "libc-exe"
const SitePackagesName = "site-packages.zip"

// Add a command to parent that will be delegated to python
func AddDelegateCommand(parent *cobra.Command, command, description string) {
	cmd := &cobra.Command{
		Use:   command,
		Short: description,
		Args:  cobra.ArbitraryArgs,
		Run:   DelegateCommandToPython,
	}
	cmd.SetHelpFunc(DelegateCommandToPython)

	parent.AddCommand(cmd)
}

// Exits if the command is delegated
func DelegateCommandToPython(command *cobra.Command, args []string) {
	// Expand cacheDir
	cacheDir, err := homedir.Expand(CachePath)
	if err != nil {
		fmt.Print("Could not locate cache dir", err)
		os.Exit(1)
	}

	// Extract python and site-packages
	pythonPathInfo, err := PopulateCache(cacheDir)
	if err != nil {
		fmt.Print("Could not delegate command", err)
		os.Exit(1)
	}

	// Build the command string for the current platform
	var prog []string
	if runtime.GOOS == "windows" {
		prog = []string{filepath.Join(pythonPathInfo.PythonBinDir, "python.exe")}
	} else if runtime.GOOS == "linux" {
		prog = []string{
			filepath.Join(pythonPathInfo.PythonBinDir, LibcExec),
			filepath.Join(pythonPathInfo.PythonBinDir, "python"),
		}
	} else {
		prog = []string{filepath.Join(pythonPathInfo.PythonBinDir, "python")}
	}

	// Passthrough all args beyond program
	// NOTE: The args passed into this function are not always complete,
	// especially in the case of help invocation. Hence the direct use of
	// os.Args instead.
	prog = append(prog, PythonCliCommand...)
	prog = append(prog, os.Args[1:]...)

	// Launch the python command with piped stdin/stdout
	cmd := exec.Command(prog[0], prog[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PYTHONHOME=%s", pythonPathInfo.PythonBinDir),
	)

	err = cmd.Run()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

// Extract the python interpreter and site-packages.zip to the cache directory
func PopulateCache(cacheDir string) (*PythonPathInfo, error) {
	// Get version info
	data, err := pkgdata.Asset("version.json")
	if err != nil {
		return nil, err
	}

	versionInfo := PkgVersionInfo{}
	err = json.Unmarshal(data, &versionInfo)
	if err != nil {
		return nil, err
	}

	// Resolve paths relative to cache dir
	pythonDistName := fmt.Sprintf("python-%s", versionInfo.PythonVersion)
	pythonDir := filepath.Join(cacheDir, pythonDistName)
	pythonLibDir := filepath.Join(pythonDir, "lib", fmt.Sprintf("python%s", versionInfo.PyVer))
	sitePkgPath := filepath.Join(pythonLibDir, "site-packages")

	// Extract python, if python not present
	_, err = os.Stat(pythonDir)
	if os.IsNotExist(err) {
		err = ExtractAssetZip(pythonDistName+".zip", pythonDir)
	}
	if err != nil {
		return nil, err
	}

	// Copy site package, if out of date
	buildTimePath := filepath.Join(cacheDir, ".site-packages.buildtime")
	buildTime, err := ioutil.ReadFile(buildTimePath)
	if err != nil || string(buildTime) != versionInfo.BuildTime {
		// Extract site-packages.zip
		os.RemoveAll(sitePkgPath)
		err = ExtractAssetZip(SitePackagesName, sitePkgPath)

		// Write the build time to .site-packages.buildtime
		err = ioutil.WriteFile(buildTimePath, []byte(versionInfo.BuildTime), os.FileMode(0644))
		if err != nil {
			return nil, err
		}
	}

	return &PythonPathInfo{pythonDir, sitePkgPath}, nil
}

// Extract a go-bindata encoded asset zipfile to destDir
func ExtractAssetZip(assetName, destDir string) error {
	// See: https://golangcode.com/unzip-files-in-go/
	// Load asset
	dataInfo, err := pkgdata.AssetInfo(assetName)
	if err != nil {
		return err
	}

	data, err := pkgdata.Asset(assetName)
	if err != nil {
		return err
	}

	// Extract asset
	byteReader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(byteReader, dataInfo.Size())
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		inFile, err := f.Open()
		if err != nil {
			return err
		}
		defer inFile.Close()

		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
			if err != nil {
				return err
			}
			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, inFile)
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
