package ostools

import "os"

// Tiny wrapper around the default os.MkdirAll but apply standard permissions.
func MkdirAll(path string) error {
	return os.MkdirAll(path, defaultPermissions)
}

// Totally removes and recreates the given path.
func EmptyDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	return MkdirAll(path)
}
