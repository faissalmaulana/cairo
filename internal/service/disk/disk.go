package disk

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Disk struct {
	entrypoint string
}

type DataInput struct {
	Src       io.Reader
	Filename  string
	Directory string
}

func NewDisk(entrypoint string) *Disk {
	return &Disk{
		entrypoint: entrypoint,
	}
}

func (d *Disk) Write(data DataInput) error {
	if data.Directory == "" {
		return errors.New("directory is required")
	}

	if err := os.MkdirAll(filepath.Join(d.entrypoint, data.Directory), 0755); err != nil {
		return err
	}

	f, err := os.CreateTemp(d.entrypoint, "tempfile")
	if err != nil {
		return err
	}

	defer os.Remove(f.Name())
	defer f.Close()

	if _, err := io.Copy(f, data.Src); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	dst := filepath.Join(d.entrypoint, data.Directory, d.decodeFilename(data.Filename))
	return os.Rename(f.Name(), dst)
}

func (d *Disk) decodeFilename(filename string) string {

	return strings.Join(strings.Split(filename, "/"), " ")
}

func (d *Disk) encodeFilename(filename string) string {

	return strings.Join(strings.Split(filename, " "), "/")
}
