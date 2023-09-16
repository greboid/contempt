package contempt

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

func copyMap[K, V comparable](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}
	return result
}

func list(v ...string) []string {
	return v
}

func tagsafe(input string) string {
	input = strings.ReplaceAll(input, "-", "_")
	input = strings.ReplaceAll(input, "/", "_")
	input = strings.ReplaceAll(input, " ", "_")
	return fmt.Sprintf("%s", input)
}

func getSet(funcMap template.FuncMap, data map[string]interface{}) (func(string) string, func(string, any) string, func(string) string) {
	return func(name string) string {
			return data[name].(string)
		},
		func(name string, variable any) string {
			variableType := reflect.ValueOf(variable).Type()
			if variableType.Kind() == reflect.Map {
				if variableType.Elem().Kind() == reflect.String {
					data[name] = variable.(map[string]string)
				} else if variableType.Elem().Kind() == reflect.Int {
					data[name] = variable.(map[string]int)
				} else if variableType.Elem().Kind() == reflect.Interface {
					data[name] = variable.(map[string]interface{})
				}
			} else if variableType.Kind() == reflect.Slice {
				if variableType.Elem().Kind() == reflect.String {
					data[name] = variable.([]string)
				} else if variableType.Elem().Kind() == reflect.Int {
					data[name] = variable.(map[string]int)
				} else if variableType.Elem().Kind() == reflect.Interface {
					data[name] = variable.([]interface{})
				}
			} else {
				data[name] = variable
			}
			return ""
		},
		func(name string) string {
			inFile := filepath.Join(flag.Arg(0), "./_partials", name)
			tpl := template.New(inFile)
			tpl.Funcs(funcMap)

			if _, err := tpl.ParseFiles(inFile); err != nil {
				log.Fatalf("unable to parse partial %s: %v", inFile, err)
			}

			writer := &bytes.Buffer{}
			if err := tpl.ExecuteTemplate(writer, filepath.Base(inFile), data); err != nil {
				log.Fatalf("unable to parse partial %s: %v", inFile, err)
			}
			return writer.String()
		}
}

func checkoutTag(repo string) string {
	output := "#Ensure git is installed\n"
	gitDeps := alpinePackages("git")
	if len(gitDeps) > 0 {
		output += fmt.Sprintf("RUN set -eux; \\\n")
		output += fmt.Sprintf("\tapk add --no-cache \\\n")
		i := len(gitDeps)
		for key, value := range gitDeps {
			i--
			output += fmt.Sprintf("\t\t%s=%s", key, value)
			if i != 0 {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += fmt.Sprintf(" \n")
			}
		}
	}
	output += "#Get latest tag and clone the repo\n"
	output += fmt.Sprintf("ARG %s=%s\n", tagify(repo), gitHubTag(repo))
	output += fmt.Sprintf("RUN git clone --depth=1 -b $%s --single-branch https://github.com/%s /src/%s\n", tagify(repo), repo, repo)
	return output
}
func checkoutCommit(repo string, branch string, hash string) string {
	output := "#Ensure git is installed\n"
	gitDeps := alpinePackages("git")
	if len(gitDeps) > 0 {
		output += fmt.Sprintf("RUN set -eux; \\\n")
		output += fmt.Sprintf("\tapk add --no-cache \\\n")
		i := len(gitDeps)
		for key, value := range gitDeps {
			i--
			output += fmt.Sprintf("\t\t%s=%s", key, value)
			if i != 0 {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += fmt.Sprintf(" \n")
			}
		}
	}
	output += "#Get latest tag and clone the repo\n"
	output += fmt.Sprintf("RUN git clone -b %s --single-branch https://github.com/%s /src/%s\n", branch, repo, tagify(repo))
	output += fmt.Sprintf("WORKDIR /src/%s\n", tagify(repo))
	output += fmt.Sprintf("RUN git checkout %s", hash)
	return output
}
func copyDirectories(repo string, directories ...string) string {
	output := ""
	if len(directories) > 0 {
		output += "#Ensure rsync is installed\n"
		rsyncDeps := alpinePackages("rsync")
		if len(rsyncDeps) > 0 {
			output += fmt.Sprintf("RUN apk add --no-cache \\\n")
			i := len(rsyncDeps)
			for key, value := range rsyncDeps {
				i--
				output += fmt.Sprintf("\t%s=%s", key, value)
				if i != 0 {
					output += fmt.Sprintf(" \\\n")
				} else {
					output += fmt.Sprintf(" \n")
				}
			}
		}
		output += "#Copy directories into rootfs\n"
		length := len(directories) - 1
		for i := range directories {
			output += fmt.Sprintf("RUN rsync -ap /src/%s%s/ /rootfs/\n", tagify(repo), directories[i])
			if i < length {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += fmt.Sprintf(" \n")
			}
		}
	}
	return output
}
func createVolumes(volumes ...string) string {
	output := ""
	length := len(volumes) - 1
	if len(volumes) > 0 {
		output += fmt.Sprintf("#Create required volumes\n")
		output += fmt.Sprintf("RUN mkdir -p \\\n")
		for i := range volumes {
			output += fmt.Sprintf("\t/rootfs%s", volumes[i])
			if i < length {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += "\n"
			}
		}
	}
	return output
}
func installBuildDeps(packages ...string) string {
	output := "#Install build dependencies\n"
	if len(packages) > 0 {
		deps := alpinePackages(packages...)
		output += "RUN apk add --no-cache \\\n"
		i := len(deps)
		for key, value := range deps {
			i--
			output += fmt.Sprintf("\t%s=%s", key, value)
			if i != 0 {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += "\n"
			}
		}
	}
	return output
}
func installRunDeps(packages ...string) string {
	output := ""
	if len(packages) > 0 {
		output += "#Ensure rsync is installed\n"
		rsyncDeps := alpinePackages("rsync")
		if len(rsyncDeps) > 0 {
			output += fmt.Sprintf("RUN set -eux; \\\n")
			output += fmt.Sprintf("\tapk add --no-cache \\\n")
			i := len(rsyncDeps)
			for key, value := range rsyncDeps {
				i--
				output += fmt.Sprintf("\t%s=%s", key, value)
				if i != 0 {
					output += fmt.Sprintf(" \\\n")
				} else {
					output += fmt.Sprintf(" \n")
				}
			}
		}
		output += "#Add packages into the runtime rootfs\n"
		deps := alpinePackages(packages...)
		output += "RUN apk add --no-cache \\\n"
		i := len(deps)
		for key, value := range deps {
			i--
			output += fmt.Sprintf("\tapk add --no-cache %s=%s; \\\n", key, value)
			output += fmt.Sprintf("\tapk info -qL %s | rsync -aq --files-from=- / /rootfs/", key)
			if i != 0 {
				output += fmt.Sprintf(" \\\n")
			} else {
				output += "\n"
			}
		}
	}
	return output
}
func goBuild(repo string, binary string, builddir string, tags []string, buildVars []string, skipLicenses []string) string {
	tags = append(tags, "netgo", "osusergo")
	tagsString := fmt.Sprintf("-tags %s", strings.Join(tags, ","))
	buildVarsString := ""
	for i := range buildVars {
		buildVarsString += fmt.Sprintf(" -X %s", buildVars[i])
	}
	skipLicensesString := ""
	for i := range skipLicenses {
		skipLicensesString += fmt.Sprintf(" --ignore %s", skipLicenses[i])
	}
	output := fmt.Sprintf("WORKDIR /src/%s\n", tagify(repo))
	output += fmt.Sprintf("RUN go build %s -a -trimpath -ldflags='-w -buildid=%s' -o /rootfs/%s ./%s\n", tagsString, buildVarsString, binary, builddir)
	output += fmt.Sprintf("RUN go-licenses save ./... --save_path=/notices --force %s\n", skipLicensesString)
	output += fmt.Sprintf("RUN cp -r /notices /rootfs/\n")
	return output
}
func fromBaseAddBinary(binary string, entryArgs []string, runArgs []string) string {
	entryArgsString := fmt.Sprintf("\"/%s\"", binary)
	if len(entryArgs) > 0 {
		entryArgsString += " ,"
	}
	for i := range entryArgs {
		entryArgsString += fmt.Sprintf("\"%s\", ", entryArgs[i])
	}
	output := fmt.Sprintf("FROM %s\n", image("base"))
	output += fmt.Sprintf("COPY --from=build --chown=65532:65532 /rootfs/ /\n")
	output += fmt.Sprintf("ENTRYPOINT [%s]\n", entryArgsString)
	if len(runArgs) > 0 {
		length := len(runArgs) - 1
		output += "CMD ["
		for i := range runArgs {
			output += fmt.Sprintf("\"%s\"", runArgs[i])
			if i < length {
				output += ", "
			}
		}
		output += "]\n"
	}
	return output
}

func tagify(input string) string {
	input = strings.ReplaceAll(input, "-", "_")
	input = strings.ReplaceAll(input, "/", "_")
	input = strings.ReplaceAll(input, " ", "_")
	return fmt.Sprintf("TAG%s", input)
}
