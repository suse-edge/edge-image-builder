package fileio

import (
	"fmt"
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
			expectedErr: "creating destination file: open : no such file or directory",
		},
		{
			name:        "Destination is a directory",
			source:      source,
			destination: tmpDir,
			expectedErr: fmt.Sprintf("creating destination file: open %s: is a directory", tmpDir),
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
