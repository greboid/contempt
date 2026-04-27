package materials

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

func ReadApprovedFromDir(dir string, outputName string) (BOM, error) {
	result := make(BOM)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.Name() != outputName {
			return nil
		}

		bom := Read(path)
		for material, version := range bom {
			result[material] = version
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to scan for approved versions: %w", err)
	}

	return result, nil
}

func IsAlwaysApproved(material string) bool {
	prefixes := []string{"image:", "apk:"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(material, prefix) {
			return true
		}
	}
	return false
}
