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
	t.Parallel()

	testCases := []struct {
		name      string
		content   string
		filename  string
		directory string
		seed      func(t *testing.T, entry string)
		wantPath  string
	}{
		{
			name:      "writes content into nested directory",
			content:   "HELLO,WORLD",
			filename:  "avatars/haaland.txt",
			directory: "profile",
			wantPath:  filepath.Join("profile", "avatars haaland.txt"),
		},
		{
			name:      "overwrites existing file",
			content:   "second",
			filename:  "a.txt",
			directory: "dir",
			seed: func(t *testing.T, entry string) {
				require.NoError(t, os.MkdirAll(filepath.Join(entry, "dir"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(entry, "dir", "a.txt"), []byte("first"), 0o644))
			},
			wantPath: filepath.Join("dir", "a.txt"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			entry := t.TempDir()
			d := disk.NewDisk(entry)

			if tc.seed != nil {
				tc.seed(t, entry)
			}

			code, err := d.Write(disk.DataInput{
				Src:       strings.NewReader(tc.content),
				Filename:  tc.filename,
				Directory: tc.directory,
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
	t.Parallel()

	t.Run("fails when directory is empty", func(t *testing.T) {
		t.Parallel()

		d := disk.NewDisk(t.TempDir())

		code, err := d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "",
		})
		assert.EqualError(t, err, "directory is required")
		assert.Equal(t, 1, code)
	})

	t.Run("fails when entrypoint is not a directory", func(t *testing.T) {
		t.Parallel()

		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		d := disk.NewDisk(entry)
		code, err := d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "dir",
		})
		assert.Error(t, err)
		assert.Equal(t, 1, code)
	})
}

func TestRead(t *testing.T) {
	t.Parallel()

	entry := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(entry, "profile"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(entry, "profile", "avatars haaland.txt"), []byte("HELLO,WORLD"), 0o644))

	d := disk.NewDisk(entry)

	rc, err := d.Read("avatars/haaland.txt", "profile")
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "HELLO,WORLD", string(got))
}

func TestReadError(t *testing.T) {
	t.Parallel()

	t.Run("fails when directory is empty", func(t *testing.T) {
		t.Parallel()

		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("a.txt", "")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when file does not exist", func(t *testing.T) {
		t.Parallel()

		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("missing.txt", "dir")
		assert.EqualError(t, err, "file not found")
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	entry := t.TempDir()
	file := filepath.Join(entry, "profile avatars haaland.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

	d := disk.NewDisk(entry)

	require.NoError(t, d.Delete("profile/avatars/haaland.txt"))

	_, err := os.Stat(file)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteError(t *testing.T) {
	t.Parallel()

	d := disk.NewDisk(t.TempDir())

	err := d.Delete("profile/avatars/haaland.txt")
	assert.EqualError(t, err, "file not found")
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("returns encoded names, skipping subdirectories", func(t *testing.T) {
		t.Parallel()

		entry := t.TempDir()
		dir := filepath.Join(entry, "profile")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))
		for _, name := range []string{"a.txt", "avatars haaland.txt"} {
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
		}

		d := disk.NewDisk(entry)

		got, err := d.List("profile")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"avatars/haaland.txt", "a.txt"}, got)
	})

	t.Run("returns empty slice for empty directory", func(t *testing.T) {
		t.Parallel()

		entry := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(entry, "profile"), 0o755))

		d := disk.NewDisk(entry)

		got, err := d.List("profile")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestListError(t *testing.T) {
	t.Parallel()

	t.Run("fails when directory is empty", func(t *testing.T) {
		t.Parallel()

		d := disk.NewDisk(t.TempDir())

		_, err := d.List("")
		assert.EqualError(t, err, "directory is required")
	})

	t.Run("fails when directory does not exist", func(t *testing.T) {
		t.Parallel()

		d := disk.NewDisk(t.TempDir())

		_, err := d.List("missing")
		assert.EqualError(t, err, "directory not found")
	})
}
