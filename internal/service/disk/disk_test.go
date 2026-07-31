package disk_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faissalmaulana/cairo/internal/service/disk"
)

func TestWrite(t *testing.T) {
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
				t.Helper()
				if err := os.MkdirAll(filepath.Join(entry, "dir"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(entry, "dir", "a.txt"), []byte("first"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantPath: filepath.Join("dir", "a.txt"),
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
				Src:       strings.NewReader(tc.content),
				Filename:  tc.filename,
				Directory: tc.directory,
			})
			if err != nil {
				t.Fatal(err)
			}
			if code != 0 {
				t.Fatalf("expected code 0, got %d", code)
			}

			got, err := os.ReadFile(filepath.Join(entry, tc.wantPath))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.content {
				t.Fatalf("expected %q, got %q", tc.content, string(got))
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	t.Run("fails when directory is empty", func(t *testing.T) {
		d := disk.NewDisk(t.TempDir())

		code, err := d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "",
		})
		if err == nil || err.Error() != "directory is required" {
			t.Fatalf("expected error %q, got %v", "directory is required", err)
		}
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
	})

	t.Run("fails when entrypoint is not a directory", func(t *testing.T) {
		entry := filepath.Join(t.TempDir(), "entry")
		if err := os.WriteFile(entry, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		d := disk.NewDisk(entry)
		code, err := d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "dir",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if code != 1 {
			t.Fatalf("expected code 1, got %d", code)
		}
	})
}

func TestRead(t *testing.T) {
	entry := t.TempDir()
	if err := os.MkdirAll(filepath.Join(entry, "profile"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "profile", "avatars haaland.txt"), []byte("HELLO,WORLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := disk.NewDisk(entry)

	rc, err := d.Read("avatars/haaland.txt", "profile")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "HELLO,WORLD" {
		t.Fatalf("expected %q, got %q", "HELLO,WORLD", string(got))
	}
}

func TestReadError(t *testing.T) {
	t.Run("fails when directory is empty", func(t *testing.T) {
		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("a.txt", "")
		if err == nil || err.Error() != "directory is required" {
			t.Fatalf("expected error %q, got %v", "directory is required", err)
		}
	})

	t.Run("fails when file does not exist", func(t *testing.T) {
		d := disk.NewDisk(t.TempDir())

		_, err := d.Read("missing.txt", "dir")
		if err == nil || err.Error() != "file not found" {
			t.Fatalf("expected error %q, got %v", "file not found", err)
		}
	})
}

func TestDelete(t *testing.T) {
	entry := t.TempDir()
	file := filepath.Join(entry, "profile avatars haaland.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := disk.NewDisk(entry)

	if err := d.Delete("profile/avatars/haaland.txt"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, stat error: %v", err)
	}
}

func TestDeleteError(t *testing.T) {
	d := disk.NewDisk(t.TempDir())

	err := d.Delete("profile/avatars/haaland.txt")
	if err == nil || err.Error() != "file not found" {
		t.Fatalf("expected error %q, got %v", "file not found", err)
	}
}
