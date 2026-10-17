package anonsurf

import (
	"os"

	"gitlab.com/parrotsec/packages/anonsurf/internal/fileops"
)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func fileExists(path string) (bool, error) {
	return fileops.Exists(path)
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	return fileops.WriteAtomicWithDir(path, data, perm, 0700)
}

func removeFile(path string) error {
	return fileops.RemoveIfExists(path)
}
