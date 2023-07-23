package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/csmith/contempt/internal"
	"github.com/csmith/envflag"
	"golang.org/x/exp/slices"
	"gopkg.in/osteele/liquid.v1"
	"io/fs"
	"os"
	"path"
	"strings"
)

var (
	flags    = flag.NewFlagSet("orchestrator", flag.ExitOnError)
	registry = flags.String("registry", "", "The name of the registry that images are pushed to")
	template = flags.String("template", "", "Path of the template to read")
	output   = flags.String("output", "", "Path to output the generated file")
)

func main() {
	envflag.Parse(envflag.WithFlagSet(flags))

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintf(os.Stderr, "Expected one positional argument: <input dir>\n")
		flags.Usage()
		os.Exit(2)
	}

	flags.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			_, _ = fmt.Fprintf(os.Stderr, "Missing required flag: %s\n", f.Name)
			flags.Usage()
			os.Exit(2)
		}
	})

	s := os.DirFS(flags.Arg(0))

	files, err := internal.FindFiles(s, func(s string) bool {
		return strings.ToLower(s) == "dockerfile" || strings.ToLower(s) == "containerfile"
	})

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to scan directory '%s': %v\n", flag.Arg(0), err)
		os.Exit(3)
	}

	var deps []target
	for i := range files {
		name := path.Base(path.Dir(files[i]))
		needed, err := readDependencies(s, *registry, files[i])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to find dependencies of %s: %v\n", files[i], err)
			os.Exit(4)
		}

		deps = append(deps, target{name, needed})
	}

	deps, err = orderDependencies(deps)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to order dependencies: %v\n", err)
		os.Exit(5)
	}

	tpl, err := os.ReadFile(*template)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to load template from '%s': %v\n", *template, err)
		os.Exit(6)
	}

	engine := liquid.NewEngine()
	out, err := engine.ParseAndRender(tpl, liquid.Bindings{
		"targets": deps,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to generate output: %v\n", err)
		os.Exit(7)
	}

	err = os.WriteFile(*output, out, os.FileMode(0644))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to write output to '%s': %v\n", *output, err)
		os.Exit(8)
	}
}

type target struct {
	Name   string   `liquid:"name"`
	Needed []string `liquid:"needed"`
}

// TODO: At some point this should switch to read the template syntax used by the writer
func readDependencies(s fs.FS, registry string, p string) ([]string, error) {
	f, err := s.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Ignore dependencies on yourself
	ownName := path.Base(path.Dir(p))

	var dependencies []string
	repo := fmt.Sprintf("%s/", strings.ToLower(registry))
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == "from" && strings.HasPrefix(parts[1], repo) {
			name := strings.TrimPrefix(parts[1], repo)
			name, _, _ = strings.Cut(name, "@")
			name, _, _ = strings.Cut(name, ":")
			if name != ownName {
				dependencies = append(dependencies, name)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return dependencies, nil
}

// TODO: There's probably a better way to do this (e.g. by traversing the graph). Do that and test.
func orderDependencies(deps []target) ([]target, error) {
	var ordered []target

	satisfied := func(dep target) bool {
		for i := range dep.Needed {
			found := false
			for j := range ordered {
				if ordered[j].Name == dep.Needed[i] {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	for len(deps) > 0 {
		var batch []target
		var remaining []target
		for d := range deps {
			if satisfied(deps[d]) {
				batch = append(batch, deps[d])
			} else {
				remaining = append(remaining, deps[d])
			}
		}
		deps = remaining

		if len(batch) == 0 {
			return nil, fmt.Errorf("could not find any satisfied dependencies - is there a loop? Pending: %#v, selected: %#v", deps, ordered)
		}

		slices.SortFunc(batch, func(i, j target) bool {
			return i.Name < j.Name
		})
		ordered = append(ordered, batch...)
	}

	return ordered, nil
}
