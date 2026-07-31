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

var (
	ErrDirectoryRequired = errors.New("directory is required")
	ErrFileNotFound      = errors.New("file not found")
	ErrDirectoryNotFound = errors.New("directory not found")
)

func NewDisk(entrypoint string) *Disk {
	return &Disk{
		entrypoint: entrypoint,
	}
}

func (d *Disk) Read(filename, directory string) (io.ReadCloser, error) {
	if directory == "" {
		return nil, ErrDirectoryRequired
	}

	path := filepath.Join(d.entrypoint, directory, d.decodeFilename(filename))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return f, nil
}

// List returns the logical names of all files inside directory, encoded back
// to "/"-separated form. Subdirectories are skipped. The contents of the files
// are never read, so memory cost is proportional to the number of entries.
func (d *Disk) List(directory string) ([]string, error) {
	if directory == "" {
		return nil, ErrDirectoryRequired
	}

	entries, err := os.ReadDir(filepath.Join(d.entrypoint, directory))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDirectoryNotFound
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, d.encodeFilename(entry.Name()))
	}

	return names, nil
}

func (d *Disk) Write(data DataInput) (int, error) {
	if data.Directory == "" {
		return 1, ErrDirectoryRequired
	}

	if err := os.MkdirAll(filepath.Join(d.entrypoint, data.Directory), 0755); err != nil {
		return 1, err
	}

	f, err := os.CreateTemp(d.entrypoint, "tempfile")
	if err != nil {
		return 1, err
	}

	defer os.Remove(f.Name())
	defer f.Close()

	if _, err := io.Copy(f, data.Src); err != nil {
		return 1, err
	}

	if err := f.Sync(); err != nil {
		return 1, err
	}

	dst := filepath.Join(d.entrypoint, data.Directory, d.decodeFilename(data.Filename))
	if err := os.Rename(f.Name(), dst); err != nil {
		return 1, err
	}

	return 0, nil
}

func (d *Disk) Delete(path string) error {
	stored := filepath.Join(d.entrypoint, d.decodeFilename(path))
	if err := os.Remove(stored); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	return nil
}

func (d *Disk) decodeFilename(filename string) string {

	return strings.Join(strings.Split(filename, "/"), " ")
}

func (d *Disk) encodeFilename(filename string) string {

	return strings.Join(strings.Split(filename, " "), "/")
}
