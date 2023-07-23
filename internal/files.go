package internal

import (
	"io/fs"
	"strings"
)

func FindFiles(f fs.FS, matcher func(string) bool) ([]string, error) {
	var files []string
	err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && len(d.Name()) > 1 && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if !d.IsDir() && matcher(d.Name()) {
			files = append(files, path)
		}

		return err
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
