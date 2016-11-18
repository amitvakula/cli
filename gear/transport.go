package gear

import (
	"io"
	"os"
	"path/filepath"

	"flywheel.io/deja/filesystem"
	"flywheel.io/deja/job"
)

func init() {
	t := &Cp{}
	job.RegisterInput(t)
}

type Cp struct {
}

func (t *Cp) Name() string {
	return "cp"
}

func (t *Cp) Cleanup(input *job.Input, fs filesystem.Fs) {

}

func (t *Cp) PrepareInput(input *job.Input, fs filesystem.Fs, workdir string) (*job.Input, error) {
	src, err := os.Open(input.URI)
	if err != nil {
		return input, err
	}
	defer src.Close()

	err = fs.MkdirAll("", 0755)
	if err != nil {
		return input, err
	}

	dest, err := fs.OpenFile(filepath.Base(input.URI), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return input, err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return input, err
	}

	return input, nil
}
