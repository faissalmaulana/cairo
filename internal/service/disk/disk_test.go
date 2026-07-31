package disk_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/faissalmaulana/cairo/internal/service/disk"
	"github.com/stretchr/testify/suite"
)

type WriteSuite struct {
	suite.Suite
	entry string
	d     *disk.Disk
}

func (s *WriteSuite) SetupTest() {
	s.entry = s.T().TempDir()
	s.d = disk.NewDisk(s.entry)
}

func TestWriteSuite(t *testing.T) {
	suite.Run(t, new(WriteSuite))
}

func (s *WriteSuite) TestWrite() {
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
				s.Require().NoError(os.MkdirAll(filepath.Join(entry, "dir"), 0o755))
				s.Require().NoError(os.WriteFile(filepath.Join(entry, "dir", "a.txt"), []byte("first"), 0o644))
			},
			wantPath: filepath.Join("dir", "a.txt"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.seed != nil {
				tc.seed(s.T(), s.entry)
			}

			code, err := s.d.Write(disk.DataInput{
				Src:       strings.NewReader(tc.content),
				Filename:  tc.filename,
				Directory: tc.directory,
			})
			s.Require().NoError(err)
			s.Equal(0, code)

			got, err := os.ReadFile(filepath.Join(s.entry, tc.wantPath))
			s.Require().NoError(err)
			s.Equal(tc.content, string(got))
		})
	}
}

func (s *WriteSuite) TestRead() {
	s.Run("reads streamed content from file", func() {
		dir := filepath.Join(s.entry, "profile")
		s.Require().NoError(os.MkdirAll(dir, 0o755))
		s.Require().NoError(os.WriteFile(filepath.Join(dir, "avatars haaland.txt"), []byte("HELLO,WORLD"), 0o644))

		rc, err := s.d.Read("avatars/haaland.txt", "profile")
		s.Require().NoError(err)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		s.Require().NoError(err)
		s.Equal("HELLO,WORLD", string(got))
	})
}

func (s *WriteSuite) TestReadError() {
	s.Run("fails when directory is empty", func() {
		_, err := s.d.Read("a.txt", "")
		s.EqualError(err, "directory is required")
	})

	s.Run("fails when file does not exist", func() {
		_, err := s.d.Read("missing.txt", "dir")
		s.EqualError(err, "file not found")
	})
}

func (s *WriteSuite) TestWriteError() {
	s.Run("fails when directory is empty", func() {
		code, err := s.d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "",
		})

		s.EqualError(err, "directory is required")
		s.Equal(1, code)
	})

	s.Run("fails when entrypoint is not a directory", func() {
		entry := filepath.Join(s.T().TempDir(), "entry")
		s.Require().NoError(os.WriteFile(entry, []byte("x"), 0o644))

		d := disk.NewDisk(entry)
		code, err := d.Write(disk.DataInput{
			Src:       strings.NewReader("data"),
			Filename:  "a.txt",
			Directory: "dir",
		})

		s.Error(err)
		s.Equal(1, code)
	})
}
