package contempt

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
)

// FindProjects returns a slice of all images that can be built from this repo, sorted such that images are positioned
// after all of their dependencies.
func FindProjects(dir, templateName string) []string {
	deps := make(map[string][]string)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == templateName {
			project := filepath.Dir(path)
			if _, err := os.Stat(filepath.Join(project, "IGNORE")); errors.Is(err, os.ErrNotExist) {
				deps[filepath.Base(project)] = dependencies(project, templateName)
			}
		}
		return nil
	})

	var res []string
	satisfied := func(reqs []string) bool {
		found := 0
		for i := range reqs {
			for j := range res {
				if res[j] == reqs[i] {
					found++
					break
				}
			}
		}
		return found == len(reqs)
	}

	for len(deps) > 0 {
		for d := range deps {
			if satisfied(deps[d]) {
				res = append(res, d)
				delete(deps, d)
			}
		}
	}

	return res
}

func dependencies(dir, templateName string) []string {
	var res []string
	fakeFunks := template.FuncMap{}
	for f := range templateFuncs {
		out := reflect.ValueOf(templateFuncs[f]).Type().Out(0).Kind()
		if f == "image" {
			fakeFunks[f] = func(dep string) string {
				res = append(res, dep)
				return ""
			}
		} else if out == reflect.Map {
			fakeFunks[f] = func(args ...string) map[string]string {
				return nil
			}
		} else if out == reflect.Slice {
			fakeFunks[f] = func(args ...string) []string {
				return nil
			}
		} else {
			fakeFunks[f] = func(args ...string) string {
				return ""
			}
		}
	}

	templatePath := filepath.Join(dir, templateName)
	tpl := template.New(templatePath)
	tpl.Funcs(fakeFunks)
	_, _ = tpl.ParseFiles(templatePath)
	_ = tpl.ExecuteTemplate(io.Discard, templateName, nil)
	return res
}
