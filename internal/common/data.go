package common

import (
	"embed"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ReadData reads a file from an embedded FS if the path starts with "config/",
// otherwise falls back to reading from the real filesystem.
func ReadData(path string, fs embed.FS) ([]byte, error) {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if strings.HasPrefix(normalized, "config/") {
		data, err := fs.ReadFile(normalized)
		if err == nil {
			return data, nil
		}
	}
	return os.ReadFile(path)
}

// LoadYAMLList loads a YAML file at path using readFn and unmarshals it into a
// slice of T. readFn is expected to be a closure that wraps the package-level
// readData (which already knows about the embedded FS).
func LoadYAMLList[T any](path string, readFn func(string) ([]byte, error)) ([]T, error) {
	data, err := readFn(path)
	if err != nil {
		return nil, err
	}
	var items []T
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
