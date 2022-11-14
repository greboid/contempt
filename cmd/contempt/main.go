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
	"github.com/csmith/contempt/sources"
	"github.com/csmith/envflag"
	"golang.org/x/exp/slices"
)

var (
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
)

func main() {
	envflag.Parse()

	if flag.NArg() != 2 {
		_, _ = fmt.Fprintf(os.Stderr, "Required arguments missing: <input dir> <output dir>\n")
		flag.Usage()
		os.Exit(2)
	}

	projects, err := contempt.FindProjects(flag.Arg(0), *templateName)
	if err != nil {
		log.Fatalf("Failed to find projects: %v", err)
	}

	filtered := strings.Split(*filter, ",")

	for i := range projects {
		if *filter == "" || slices.Contains(filtered, projects[i]) {
			if *workflowCommands {
				fmt.Printf("::group::%s\n", projects[i])
			}
			log.Printf("Checking project %s", projects[i])
			outPath := filepath.Join(flag.Arg(1), projects[i], *outputName)
			changes, err := contempt.Generate(*sourceLink, flag.Arg(0), filepath.Join(projects[i], *templateName), outPath)
			if err != nil {
				log.Fatalf("Failed to generate project %s: %v", projects[i], err)
			}

			if *commit {
				if err := doCommit(projects[i], changes); err != nil {
					log.Printf("Failed to commit %s: %v", projects[i], err)
					continue
				}
			}

			if (*commit && *build) || *forceBuild {
				imageName := fmt.Sprintf("%s/%s", sources.Registry(), projects[i])
				if err := runBuildahCommand(
					"bud",
					"--timestamp",
					"0",
					"--layers",
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
}

func doCommit(project string, changes []contempt.Change) error {
	if err := runGitCommand(
		"-C",
		flag.Arg(1),
		"add",
		filepath.Join(project, *outputName),
	); err != nil {
		return err
	}

	if err := runGitCommand(
		"-C",
		flag.Arg(1),
		"commit",
		"--no-gpg-sign",
		"-m",
		fmt.Sprintf("[%s] %s", project, formatChanges(changes)),
		filepath.Join(project, *outputName),
	); err != nil {
		return err
	}
	return nil
}

func runGitCommand(args ...string) error {
	gitCommand := exec.Command(
		"git",
		args...,
	)
	gitCommand.Stdout = os.Stdout
	gitCommand.Stderr = os.Stderr
	return gitCommand.Run()
}

func runBuildahCommand(args ...string) error {
	buildahCommand := exec.Command(
		"/usr/bin/buildah",
		args...,
	)
	buildahCommand.Stdout = os.Stdout
	buildahCommand.Stderr = os.Stderr
	return buildahCommand.Run()
}

func formatChanges(changes []contempt.Change) string {
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
