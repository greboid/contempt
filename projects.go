package contempt

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// FindProjects returns a slice of all images that can be built from this repo, sorted such that images are positioned
// after all of their dependencies. It also returns a map of project names to the template file they use.
func FindProjects(dir string, templateNames ...string) ([]string, map[string]string, error) {
	depList := make(map[string][]string)
	projectTemplates := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		for _, tn := range templateNames {
			if d.Name() == tn {
				project := filepath.Dir(path)
				if _, err := os.Stat(filepath.Join(project, "IGNORE")); errors.Is(err, os.ErrNotExist) {
					name := filepath.Base(project)
					projectDeps, err := dependencies(project, tn)
					if err != nil {
						return err
					}
					projectTemplates[name] = tn
					depList[name] = projectDeps
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	var res []string
	satisfied := func(reqs []string) bool {
		found := 0
		for i := range reqs {
			if slices.Contains(res, reqs[i]) {
				found++
			}
		}
		return found == len(reqs)
	}

	for len(depList) > 0 {
		var batch []string
		for d := range depList {
			if satisfied(depList[d]) {
				batch = append(batch, d)
				delete(depList, d)
			}
		}
		if len(batch) == 0 {
			return nil, nil, fmt.Errorf("could not fully resolve dependencies: %#v", depList)
		}

		sort.Strings(batch)
		res = append(res, batch...)
	}

	return res, projectTemplates, nil
}

func dependencies(dir, templateName string) ([]string, error) {
	templatePath := filepath.Join(dir, templateName)

	calls, err := engine.DryRun(templatePath)
	if err != nil {
		return nil, err
	}

	var res []string
	for i := range calls["image"] {
		dep := calls["image"][i][0].(string)
		if index := strings.IndexByte(dep, '.'); index == -1 || index > strings.IndexByte(dep, '/') {
			res = append(res, dep)
		}
	}

	return res, nil
}
