package disk_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faissalmaulana/cairo/internal/service/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {

	testCases := []struct {
		name         string
		content      string
		filename     string
		directory    string
		subdirectory string
		seed         func(t *testing.T, entry string)
		wantPath     string
	}{
		{
			name:         "writes content into nested subdirectory",
			content:      "HELLO,WORLD",
			filename:     "haaland.txt",
			directory:    "profile",
			subdirectory: "avatars",
			wantPath:     filepath.Join("profile", "avatars", "haaland.txt"),
		},
		{
			name:         "overwrites existing file",
			content:      "second",
			filename:     "a.txt",
			directory:    "dir",
			subdirectory: "sub",
			seed: func(t *testing.T, entry string) {
				require.NoError(t, os.MkdirAll(filepath.Join(entry, "dir", "sub"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(entry, "dir", "sub", "a.txt"), []byte("first"), 0o644))
			},
			wantPath: filepath.Join("dir", "sub", "a.txt"),
		},
		{
			name:         "add new subdirectory",
			content:      "hello,world",
			filename:     "b.txt",
			directory:    "dir",
			subdirectory: "sub-b",
			seed: func(t *testing.T, entry string) {
				// existing subdir
				require.NoError(t, os.MkdirAll(filepath.Join(entry, "dir", "sub"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(entry, "dir", "sub", "a.txt"), []byte("first"), 0o644))
			},
			wantPath: filepath.Join("dir", "sub-b", "b.txt"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			entry := t.TempDir()
			d := disk.NewDisk(entry)

			if tc.seed != nil {
				tc.seed(t, entry)
			}

			code, err := d.Write(disk.DataInput{
				Src:          strings.NewReader(tc.content),
				Filename:     tc.filename,
				Directory:    tc.directory,
				Subdirectory: tc.subdirectory,
			})
			assert.NoError(t, err)
			assert.Equal(t, 0, code)

			got, err := os.ReadFile(filepath.Join(entry, tc.wantPath))
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(got))
		})
	}
}

func TestWriteError(t *testing.T) {

	t.Run("fails when directory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		code, err := d.Write(disk.DataInput{
			Src:          strings.NewReader("data"),
			Filename:     "a.txt",
			Directory:    "",
			Subdirectory: "sub",
		})
		assert.EqualError(t, err, "directory is required")
		assert.Equal(t, 1, code)
	})

	t.Run("fails when subdirectory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		code, err := d.Write(disk.DataInput{
			Src:          strings.NewReader("data"),
			Filename:     "a.txt",
			Directory:    "dir",
			Subdirectory: "",
		})
		assert.EqualError(t, err, "subdirectory is required")
		assert.Equal(t, 1, code)
	})

	t.Run("fails when entrypoint is not a directory", func(t *testing.T) {

		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		d := disk.NewDisk(entry)
		code, err := d.Write(disk.DataInput{
			Src:          strings.NewReader("data"),
			Filename:     "a.txt",
			Directory:    "dir",
			Subdirectory: "sub",
		})
		assert.Error(t, err)
		assert.Equal(t, 1, code)
	})
}

func TestRead(t *testing.T) {

	entry := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(entry, "profile", "avatars"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(entry, "profile", "avatars", "haaland.txt"), []byte("HELLO,WORLD"), 0o644))

	d := disk.NewDisk(entry)

	rc, err := d.Read("profile", "avatars/haaland.txt")
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "HELLO,WORLD", string(got))
}

func TestReadError(t *testing.T) {

	t.Run("fails when directory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("", "sub/a.txt")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when file does not exist", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("dir", "sub/missing.txt")
		assert.EqualError(t, err, "file not found")
	})
}

func TestDelete(t *testing.T) {

	entry := t.TempDir()
	file := filepath.Join(entry, "profile", "avatars", "haaland.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

	d := disk.NewDisk(entry)

	require.NoError(t, d.Delete("profile", "avatars/haaland.txt"))

	_, err := os.Stat(file)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteError(t *testing.T) {

	t.Run("fails when directory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Delete("", "sub/a.txt")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when file does not exist", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Delete("profile", "avatars/haaland.txt")
		assert.EqualError(t, err, "file not found")
	})
}

func TestList(t *testing.T) {

	t.Run("returns file names, skipping subdirectories", func(t *testing.T) {

		entry := t.TempDir()
		dir := filepath.Join(entry, "profile", "avatars")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		// another subdir
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "nested"), 0o755))
		for _, name := range []string{"a.txt", "haaland.txt"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
		}

		d := disk.NewDisk(entry)

		got, err := d.List("profile", "avatars")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"haaland.txt", "a.txt"}, got)
	})

	t.Run("returns empty slice for empty directory", func(t *testing.T) {

		entry := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "profile", "avatars"), 0o755))

		d := disk.NewDisk(entry)

		got, err := d.List("profile", "avatars")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestListError(t *testing.T) {

	t.Run("fails when directory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		_, err := d.List("", "sub")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when subdirectory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		_, err := d.List("dir", "")
		assert.EqualError(t, err, "subdirectory is required")
	})

	t.Run("fails when directory does not exist", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		_, err := d.List("missing", "sub")
		assert.EqualError(t, err, "directory not found")
	})
}

func TestLink(t *testing.T) {

	t.Run("creates a public symlink to the directory", func(t *testing.T) {

		entry := t.TempDir()
		dir := filepath.Join(entry, "user-123", "avatars")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("data"), 0o644))

		d := disk.NewDisk(entry)

		require.NoError(t, d.Link("user-123", "user-123"))

		target := filepath.Join(entry, "public", "user-123")
		info, err := os.Lstat(target)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "target should be a symlink")

		// Reading through the public link reaches the real content.
		got, err := os.ReadFile(filepath.Join(target, "avatars", "a.txt"))
		require.NoError(t, err)
		assert.Equal(t, "data", string(got))
	})

	t.Run("public name can hide the parent directory", func(t *testing.T) {

		entry := t.TempDir()
		dir := filepath.Join(entry, "user-123", "abc123")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("data"), 0o644))

		d := disk.NewDisk(entry)

		require.NoError(t, d.Link(filepath.Join("user-123", "abc123"), "abc123"))

		target := filepath.Join(entry, "public", "abc123")
		info, err := os.Lstat(target)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "target should be a symlink")
		_, err = os.Lstat(filepath.Join(entry, "public", "user-123"))
		assert.True(t, os.IsNotExist(err), "owner directory must not appear under public")

		// Reading through the public link reaches the real content.
		got, err := os.ReadFile(filepath.Join(target, "a.txt"))
		require.NoError(t, err)
		assert.Equal(t, "data", string(got))
	})

	t.Run("is idempotent when the symlink already exists", func(t *testing.T) {

		entry := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "user-123"), 0o755))

		d := disk.NewDisk(entry)

		require.NoError(t, d.Link("user-123", "user-123"))
		require.NoError(t, d.Link("user-123", "user-123"))
	})

	t.Run("fails when a real path already occupies the target", func(t *testing.T) {

		entry := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "user-123"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "public", "user-123"), 0o755))

		d := disk.NewDisk(entry)

		err := d.Link("user-123", "user-123")
		assert.EqualError(t, err, "link target already exists")
	})

	t.Run("fails when source directory is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Link("", "user-123")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when public name is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Link("user-123", "")
		assert.EqualError(t, err, "name is required")
	})
}

func TestUnlink(t *testing.T) {

	t.Run("removes the public symlink but keeps the real directory", func(t *testing.T) {

		entry := t.TempDir()
		dir := filepath.Join(entry, "user-123")
		require.NoError(t, os.MkdirAll(dir, 0o755))

		d := disk.NewDisk(entry)
		require.NoError(t, d.Link("user-123", "user-123"))

		require.NoError(t, d.Unlink("user-123"))

		_, err := os.Lstat(filepath.Join(entry, "public", "user-123"))
		assert.True(t, os.IsNotExist(err), "public link should be gone")

		// Real directory is untouched.
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("removes the link keyed by public name regardless of source", func(t *testing.T) {

		entry := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "user-123", "abc123"), 0o755))

		d := disk.NewDisk(entry)
		require.NoError(t, d.Link(filepath.Join("user-123", "abc123"), "abc123"))

		require.NoError(t, d.Unlink("abc123"))

		_, err := os.Lstat(filepath.Join(entry, "public", "abc123"))
		assert.True(t, os.IsNotExist(err), "public link should be gone")
	})

	t.Run("fails when the link does not exist", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Unlink("user-123")
		assert.EqualError(t, err, "link not found")
	})

	t.Run("fails when a real path occupies the target", func(t *testing.T) {

		entry := t.TempDir()
		// not a link
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "public", "user-123"), 0o755))

		d := disk.NewDisk(entry)

		err := d.Unlink("user-123")
		assert.EqualError(t, err, "link target already exists")

		// The real directory is not removed.
		info, err := os.Stat(filepath.Join(entry, "public", "user-123"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("fails when name is empty", func(t *testing.T) {

		d := disk.NewDisk(t.TempDir())

		err := d.Unlink("")
		assert.EqualError(t, err, "name is required")
	})
}
