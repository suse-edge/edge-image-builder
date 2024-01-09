package fileio

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	const (
		source        = "file_io.go" // use the source code file as a valid input
		destDirPrefix = "eib-copy-file-test-"
	)

	tmpDir, err := os.MkdirTemp("", destDirPrefix)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		source      string
		destination string
		perms       os.FileMode
		expectedErr string
	}{
		{
			name:        "Source file does not exist",
			source:      "<missing>",
			expectedErr: "opening source file: open <missing>: no such file or directory",
		},
		{
			name:        "Destination is an empty file",
			source:      source,
			destination: "",
			expectedErr: "creating file with permissions: creating file: open : no such file or directory",
		},
		{
			name:        "Destination is a directory",
			source:      source,
			destination: tmpDir,
			expectedErr: fmt.Sprintf("creating file with permissions: creating file: open %s: is a directory", tmpDir),
		},
		{
			name:        "File is successfully copied",
			source:      source,
			destination: fmt.Sprintf("%s/copy.go", tmpDir),
			perms:       NonExecutablePerms,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CopyFile(test.source, test.destination, test.perms)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)

				src, err := os.ReadFile(test.source)
				require.NoError(t, err)

				dest, err := os.ReadFile(test.destination)
				require.NoError(t, err)
				assert.Equal(t, src, dest)

				info, err := os.Stat(test.destination)
				require.NoError(t, err)
				assert.Equal(t, test.perms, info.Mode())
			}
		})
	}
}

func TestCopyFileN(t *testing.T) {
	const (
		destDirPrefix  = "eib-copy-file-n-test-"
		srcFileContent = "CopyFileN test"
	)

	tmpDir, err := os.MkdirTemp("", destDirPrefix)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	buffer := bytes.NewBufferString(srcFileContent)

	tests := []struct {
		name        string
		source      io.Reader
		destination string
		perms       os.FileMode
		expectedErr string
	}{
		{
			name:        "Destination is an empty file",
			source:      buffer,
			destination: "",
			expectedErr: "creating file with permissions: creating file: open : no such file or directory",
		},
		{
			name:        "Destination is a directory",
			source:      buffer,
			destination: tmpDir,
			expectedErr: fmt.Sprintf("creating file with permissions: creating file: open %s: is a directory", tmpDir),
		},
		{
			name:        "File is successfully copied",
			source:      buffer,
			destination: fmt.Sprintf("%s/copy", tmpDir),
			perms:       NonExecutablePerms,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CopyFileN(test.source, test.destination, test.perms, 4096)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)

				dest, err := os.ReadFile(test.destination)
				require.NoError(t, err)
				assert.Equal(t, []byte(srcFileContent), dest)

				info, err := os.Stat(test.destination)
				require.NoError(t, err)
				assert.Equal(t, test.perms, info.Mode())
			}
		})
	}
}
