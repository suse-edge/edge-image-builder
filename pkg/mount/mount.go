package mount

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/containers/common/pkg/subscriptions"
)

const (
	disableSuffix = "-disable"
)

// DisableDefaultMounts disables default mounts for all containers by creating an empty
// "mounts.conf" file at the override mount filepath provided by the user. Returns a function
// that can revert to the previous mount setup if needed, or an error if a problem has occured.
// If no filepath was provided, the default container override mount filepath will be used ("/etc/containers/mounts.conf").
// For more info - https://github.com/containers/common/blob/main/docs/containers-mounts.conf.5.md
func DisableDefaultMounts(overrideMountFilepath string) (revert func() error, err error) {
	mountFile := overrideMountFilepath
	if mountFile == "" {
		mountFile = subscriptions.OverrideMountsFile
	}

	disableMountFile := mountFile + disableSuffix

	_, err = os.Stat(mountFile)
	switch {
	case err == nil:
		if err = os.Rename(mountFile, disableMountFile); err != nil {
			return nil, fmt.Errorf("renaming existing %s mount override file: %w", mountFile, err)
		}

		if err = createEmptyFile(mountFile); err != nil {
			return nil, fmt.Errorf("creating empty %s mount override file: %w", mountFile, err)
		}

		return func() error {
			if err = os.Remove(mountFile); err != nil {
				return fmt.Errorf("removing empty %s file: %w", mountFile, err)
			}

			if err = os.Rename(disableMountFile, mountFile); err != nil {
				return fmt.Errorf("renaming original mounts.conf file from %s: %w", disableMountFile, err)
			}
			return nil
		}, nil
	case errors.Is(err, fs.ErrNotExist):
		if err = createEmptyFile(mountFile); err != nil {
			return nil, fmt.Errorf("creating empty %s file: %w", mountFile, err)
		}

		return func() error {
			if err = os.Remove(mountFile); err != nil {
				return fmt.Errorf("removing empty %s file: %w", mountFile, err)
			}
			return nil
		}, nil
	default:
		return nil, fmt.Errorf("describing file %s: %w", mountFile, err)
	}
}

func createEmptyFile(name string) error {
	emptyFile, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}

	if err = emptyFile.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", emptyFile.Name(), err)
	}

	return nil
}
