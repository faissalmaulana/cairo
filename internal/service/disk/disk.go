package disk

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Disk struct {
	entrypoint string
}

type DataInput struct {
	Src          io.Reader
	Filename     string
	Directory    string
	Subdirectory string
}

var (
	ErrDirectoryRequired    = errors.New("directory is required")
	ErrSubdirectoryRequired = errors.New("subdirectory is required")
	ErrFileNotFound         = errors.New("file not found")
	ErrDirectoryNotFound    = errors.New("directory not found")
)

func NewDisk(entrypoint string) *Disk {
	return &Disk{
		entrypoint: entrypoint,
	}
}

func (d *Disk) Read(filename, directory, subdirectory string) (io.ReadCloser, error) {
	if directory == "" {
		return nil, ErrDirectoryRequired
	}
	if subdirectory == "" {
		return nil, ErrSubdirectoryRequired
	}

	path := filepath.Join(d.entrypoint, directory, subdirectory, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	return f, nil
}

func (d *Disk) List(directory, subdirectory string) ([]string, error) {
	if directory == "" {
		return nil, ErrDirectoryRequired
	}
	if subdirectory == "" {
		return nil, ErrSubdirectoryRequired
	}

	entries, err := os.ReadDir(filepath.Join(d.entrypoint, directory, subdirectory))
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
		names = append(names, entry.Name())
	}

	return names, nil
}

func (d *Disk) Write(data DataInput) (int, error) {
	if data.Directory == "" {
		return 1, ErrDirectoryRequired
	}
	if data.Subdirectory == "" {
		return 1, ErrSubdirectoryRequired
	}

	if err := os.MkdirAll(filepath.Join(d.entrypoint, data.Directory, data.Subdirectory), 0755); err != nil {
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

	dst := filepath.Join(d.entrypoint, data.Directory, data.Subdirectory, data.Filename)
	if err := os.Rename(f.Name(), dst); err != nil {
		return 1, err
	}

	return 0, nil
}

func (d *Disk) Delete(filename, directory, subdirectory string) error {
	if directory == "" {
		return ErrDirectoryRequired
	}
	if subdirectory == "" {
		return ErrSubdirectoryRequired
	}

	stored := filepath.Join(d.entrypoint, directory, subdirectory, filename)
	if err := os.Remove(stored); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	return nil
}
