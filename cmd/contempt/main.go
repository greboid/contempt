package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/csmith/contempt"
	"github.com/csmith/contempt/pkg/materials"
	"github.com/csmith/envflag/v2"
	"golang.org/x/exp/slices"
)

var (
	templateExplicit bool
	outputExplicit   bool
	templateName     = flag.String("template", "Dockerfile.gotpl", "The name of the template files")
	outputName       = flag.String("output", "Dockerfile", "The name of the output files")
	filter           = flag.String("project", "", "A comma-separated list of projects to generate, instead of all detected ones")
	sourceLink       = flag.String("source-link", "https://github.com/example/repo/blob/master/", "Link to a browsable version of the source repo")
	commit           = flag.Bool("commit", false, "Whether to automatically git commit each changed file")
	build            = flag.Bool("build", false, "Whether to automatically build on successful commit")
	forceBuild       = flag.Bool("force-build", false, "Whether to build projects regardless of changes")
	push             = flag.Bool("push", false, "Whether to automatically push on successful commit")
	pushRetries      = flag.Int("push-retries", 2, "How many times to retry pushing an image if it fails")
	workflowCommands = flag.Bool("workflow-commands", true, "Whether to output GitHub Actions workflow commands to format logs")
	registry         = flag.String("registry", "reg.c5h.io", "Registry to use for pushes and pulls")
	alpineMirror     = flag.String("alpine-mirror", "https://dl-cdn.alpinelinux.org/alpine/", "Base URL of the Alpine mirror to use to query version and package info")
	includesDir      = flag.String("includes", "_includes", "Folder of template files to include")
)

func main() {
	envflag.Parse()

	flag.Visit(func(f *flag.Flag) {
		if f.Name == "template" {
			templateExplicit = true
		}
		if f.Name == "output" {
			outputExplicit = true
		}
	})

	if flag.NArg() != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Required arguments missing: <input dir> <output dir>\n")
		flag.Usage()
		os.Exit(2)
	}

	projectDir, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to resolve project directory: %v", err)
	}

	contempt.InitTemplates(*registry, *alpineMirror, os.DirFS(filepath.Join(projectDir, *includesDir)))

	projects, _, err := contempt.FindProjects(projectDir, *templateName)
	if err != nil {
		log.Fatalf("Failed to find projects: %v", err)
	}

	checkExternalDependencies()

	filtered := strings.Split(*filter, ",")

	for i := range projects {
		if *filter != "" && !slices.Contains(filtered, projects[i]) {
			continue
		}

		if *workflowCommands {
			fmt.Printf("::group::%s\n", projects[i])
		}

		log.Printf("Generating project %s", projects[i])
		outPath := filepath.Join(flag.Arg(1), projects[i], *outputName)

		changes, err := contempt.Generate(*sourceLink, flag.Arg(0), filepath.Join(projects[i], *templateName), outPath)
		if err != nil {
			log.Fatalf("Failed to generate project %s: %v", projects[i], err)
		}

		if *commit && changes != nil && len(changes) > 0 {
			if err := doCommit(projects[i], *outputName, changes); err != nil {
				log.Printf("Failed to commit %s: %v", projects[i], err)
				if *workflowCommands {
					fmt.Printf("::endgroup::\n")
				}
				continue
			}
		}

		if (*commit && *build) || *forceBuild {
			imageName := fmt.Sprintf("%s/%s", *registry, projects[i])
			if err := runBuildahCommand(
				"bud",
				"--source-date-epoch",
				"0",
				"--rewrite-timestamp",
				"--omit-history",
                                "--layers=true",
				"--squash",
				"--tag",
				imageName,
				filepath.Join(flag.Arg(1), projects[i]),
			); err != nil {
				log.Fatalf("Failed to build %s: %v", projects[i], err)
			}

			if *push {
				success := false
				for r := 0; r <= *pushRetries && !success; r++ {
					if err := runBuildahCommand("push", imageName); err == nil {
						success = true
					} else {
						log.Printf("Failed to push %s [attempt %d/%d]: %v", projects[i], r+1, *pushRetries+1, err)
					}
				}
				if !success {
					log.Fatalf("Failed to push %s after %d attempts", projects[i], *pushRetries+1)
				}
			}
		}

		if *workflowCommands {
			fmt.Printf("::endgroup::\n")
		}
	}
}

func doCommit(project, outputForProject string, changes []materials.Change) error {
	if err := runGitCommand(
		"-C",
		flag.Arg(1),
		"add",
		filepath.Join(project, outputForProject),
	); err != nil {
		return err
	}

	// Check if there are actually any staged changes to commit
	if err := runGitCommand(
		"-C",
		flag.Arg(1),
		"diff",
		"--cached",
		"--quiet",
	); err == nil {
		// No staged changes, nothing to commit
		return nil
	}

	if err := runGitCommand(
		"-C",
		flag.Arg(1),
		"commit",
		"--no-gpg-sign",
		"-m",
		fmt.Sprintf("[%s] %s", project, formatChanges(changes)),
		filepath.Join(project, outputForProject),
	); err != nil {
		return err
	}
	return nil
}

func formatChanges(changes []materials.Change) string {
	if len(changes) == 0 {
		return "no detected changes"
	}

	builder := strings.Builder{}

	if len(changes) > 1 {
		builder.WriteString(fmt.Sprintf("%d changes\n", len(changes)))

		sort.Slice(changes, func(i, j int) bool {
			return changes[i].Material < changes[j].Material
		})
	}

	for i := range changes {
		oldVersion := changes[i].Old
		newVersion := changes[i].New
		if oldVersion == "" && newVersion == "" {
			builder.WriteString(fmt.Sprintf("\n%s unknown changes", changes[i].Material))
		} else if oldVersion == "" {
			builder.WriteString(fmt.Sprintf("\n%s (unknown)->%.8s", changes[i].Material, newVersion))
		} else if newVersion == "" {
			builder.WriteString(fmt.Sprintf("\n%s %.8s->(unknown)", changes[i].Material, oldVersion))
		} else {
			builder.WriteString(fmt.Sprintf("\n%s %.12s->%.12s", changes[i].Material, oldVersion, newVersion))
		}
	}

	return strings.TrimPrefix(builder.String(), "\n")
}

func runGitCommand(args ...string) error {
	return runCommand(exec.Command(
		"git",
		args...,
	))
}

func runBuildahCommand(args ...string) error {
	return runCommand(exec.Command(
		"/usr/bin/buildah",
		args...,
	))
}

func runCommand(cmd *exec.Cmd) error {
	log.Printf("Running \"%s\"", strings.Join(cmd.Args, "\" \""))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkExternalDependencies() {
	if *build || *forceBuild {
		if err := runBuildahCommand("--version"); err != nil {
			log.Fatalf("Contempt is configured to build, but buildah doesn't seem to be working: %v", err)
		}
	}

	if *commit {
		if err := runGitCommand("--version"); err != nil {
			log.Fatalf("Contempt is configured to commit, but git doesn't seem to be working: %v", err)
		}
	}
}
