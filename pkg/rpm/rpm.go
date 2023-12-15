package rpm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

// CopyRPMs copies all ".rpm" files from src to dest and returns
// a list of the copeied ".rpm" base filenames
func CopyRPMs(src string, dest string) ([]string, error) {
	if dest == "" {
		return nil, fmt.Errorf("RPM destination directory cannot be empty")
	}

	list := []string{}

	rpms, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, rpm := range rpms {
		if filepath.Ext(rpm.Name()) == ".rpm" {
			sourcePath := filepath.Join(src, rpm.Name())
			destPath := filepath.Join(dest, rpm.Name())

			err := fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms)
			if err != nil {
				return nil, fmt.Errorf("copying file %s: %w", sourcePath, err)
			}
			list = append(list, rpm.Name())
		}
	}

	return list, nil
}
