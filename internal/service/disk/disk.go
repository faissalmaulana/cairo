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
	ErrLinkTargetExists     = errors.New("link target already exists")
	ErrLinkNotFound         = errors.New("link not found")
	ErrNameRequired         = errors.New("name is required")
)

// publicRoot returns the public directory used as the target namespace for
// symlinks that expose objects without leaking the owner's account directory.
func (d *Disk) publicRoot() string {
	return filepath.Join(d.entrypoint, "public")
}

func NewDisk(entrypoint string) *Disk {
	return &Disk{
		entrypoint: entrypoint,
	}
}

func (d *Disk) Read(directory, path string) (io.ReadCloser, error) {
	if directory == "" {
		return nil, ErrDirectoryRequired
	}

	f, err := os.Open(filepath.Join(d.entrypoint, directory, path))
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

func (d *Disk) Delete(directory, path string) error {
	if directory == "" {
		return ErrDirectoryRequired
	}

	stored := filepath.Join(d.entrypoint, directory, path)
	if err := os.Remove(stored); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}
	return nil
}

// Link exposes the directory at <entrypoint>/<source> through the public
// namespace by creating a symlink at <entrypoint>/public/<name> pointing at
// it. Keeping the public name distinct from the source lets callers hide the
// owner-specific location (e.g. link an owner/bucket directory under the bare
// bucket hash). Linking is idempotent: an existing symlink at the target is
// left untouched; a regular file or directory occupying the target path is
// reported as ErrLinkTargetExists.
func (d *Disk) Link(source, name string) error {
	if source == "" {
		return ErrDirectoryRequired
	}
	if name == "" {
		return ErrNameRequired
	}

	src := filepath.Join(d.entrypoint, source)
	target := filepath.Join(d.publicRoot(), name)

	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		return ErrLinkTargetExists
	}
	if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	return os.Symlink(src, target)
}

// Unlink removes the symlink at <entrypoint>/public/<name>, taking the named
// asset out of the public namespace without touching the real directory it
// points to. A missing link is reported as ErrLinkNotFound; a regular file or
// directory occupying the target path is left untouched and reported as
// ErrLinkTargetExists.
func (d *Disk) Unlink(name string) error {
	if name == "" {
		return ErrNameRequired
	}

	target := filepath.Join(d.publicRoot(), name)

	info, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrLinkNotFound
		}
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return ErrLinkTargetExists
	}

	return os.Remove(target)
}
