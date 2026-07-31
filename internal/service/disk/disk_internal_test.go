package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeFilename(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces path separators with spaces",
			input:    "avatars/haaland.txt",
			expected: "avatars haaland.txt",
		},
		{
			name:     "handles nested path separators",
			input:    "a/b/c.txt",
			expected: "a b c.txt",
		},
		{
			name:     "leaves plain filename unchanged",
			input:    "notes.txt",
			expected: "notes.txt",
		},
		{
			name:     "handles empty filename",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := NewDisk(t.TempDir())
			assert.Equal(t, tc.expected, d.decodeFilename(tc.input))
		})
	}
}

func TestEncodeFilename(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces spaces with path separators",
			input:    "avatars haaland.txt",
			expected: "avatars/haaland.txt",
		},
		{
			name:     "leaves plain filename unchanged",
			input:    "notes.txt",
			expected: "notes.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := NewDisk(t.TempDir())
			assert.Equal(t, tc.expected, d.encodeFilename(tc.input))
		})
	}
}

func TestFilenameRoundTrip(t *testing.T) {
	t.Parallel()

	filenames := []string{"avatars/haaland.txt", "a/b/c.txt", "notes.txt"}

	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			t.Parallel()

			d := NewDisk(t.TempDir())
			assert.Equal(t, filename, d.encodeFilename(d.decodeFilename(filename)))
		})
	}
}
