package ostools

import (
	"os"
	"path/filepath"
)

const defaultPermissions = 0744

// Open or create the file to append data only. It also creates intermediate directories as needed.
func OpenAppend(name string) (*os.File, error) {
	if err := MkdirAll(filepath.Dir(name)); err != nil {
		return nil, err
	}

	return os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, defaultPermissions)
}

// Tiny wrapper around the default os.WriteFile but creates any directory
// needed before attempting to write the file.
func WriteFile(name string, data []byte) error {
	if err := MkdirAll(filepath.Dir(name)); err != nil {
		return err
	}

	return os.WriteFile(name, data, defaultPermissions)
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
