package ostools

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const defaultPermissions fs.FileMode = 0744

var ErrTooManyPermissionsGiven = errors.New("too_many_permissions_given")

// Open or create the file to append data only. It also creates intermediate directories as needed.
func OpenAppend(name string) (*os.File, error) {
	if err := MkdirAll(filepath.Dir(name)); err != nil {
		return nil, err
	}

	return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, defaultPermissions)
}

// Tiny wrapper around the default os.WriteFile but creates any directory
// needed before attempting to write the file.
// If the file mode is not given, it will default to 0744.
func WriteFile(name string, data []byte, perm ...fs.FileMode) error {
	if err := MkdirAll(filepath.Dir(name)); err != nil {
		return err
	}

	var filePermissions fs.FileMode

	switch len(perm) {
	case 0:
		filePermissions = defaultPermissions
	case 1:
		filePermissions = perm[0]
	default:
		return ErrTooManyPermissionsGiven
	}

	return os.WriteFile(name, data, filePermissions)
}

// Remove all files matching the given pattern.
func RemovePattern(pattern string) error {
	files, err := filepath.Glob(pattern)

	if err != nil {
		return err
	}

	for _, f := range files {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	return nil
}
