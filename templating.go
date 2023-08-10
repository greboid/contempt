package contempt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/greboid/contempt/sources"
)

var templateFuncs template.FuncMap

func init() {
	templateFuncs = template.FuncMap{
		"image":                image,
		"alpine_packages":      alpinePackages,
		"github_tag":           gitHubTag,
		"prefixed_github_tag":  prefixedGitHubTag,
		"git_tag":              gitTag,
		"prefixed_git_tag":     prefixedGitTag,
		"registry":             sources.Registry,
		"regex_url_content":    regexURLContent,
		"checkout_tag":         checkoutTag,
		"checkout_commit":      checkoutCommit,
		"copy_directories":     copyDirectories,
		"create_volumes":       createVolumes,
		"install_build_deps":   installBuildDeps,
		"install_run_deps":     installRunDeps,
		"go_build":             goBuild,
		"from_base_add_binary": fromBaseAddBinary,
		"tagify":               tagify,
		"list":                 list,
		"increment_int":        incrementByOne,
		"tagsafe":              tagsafe,
	}
	addRelease(templateFuncs, "alpine", sources.LatestAlpineRelease)
	addRelease(templateFuncs, "golang", sources.LatestGolangRelease)
	addRelease(templateFuncs, "postgres13", sources.LatestPostgresRelease("13"))
	addRelease(templateFuncs, "postgres14", sources.LatestPostgresRelease("14"))
	addRelease(templateFuncs, "postgres15", sources.LatestPostgresRelease("15"))
}

func image(ref string) string {
	im, digest, err := sources.LatestDigest(ref)
	if err != nil {
		log.Fatalf("Unable to get latest digest for ref %s: %v", ref, err)
	}
	materials[fmt.Sprintf("image:%s", ref)] = strings.TrimPrefix(digest, "sha256:")
	return fmt.Sprintf("%s@%s", im, digest)
}

func alpinePackages(packages ...string) map[string]string {
	res, err := sources.LatestAlpinePackages(packages...)
	if err != nil {
		log.Fatalf("Unable to get latest packages: %v", err)
	}
	for i := range res {
		materials[fmt.Sprintf("apk:%s", i)] = res[i]
	}
	return res
}

func gitHubTag(repo string) string {
	tag, err := sources.LatestGitHubTag(repo, "")
	if err != nil {
		log.Fatalf("Couldn't determine latest tag for repo %s: %v", repo, err)
	}
	materials[fmt.Sprintf("github:%s", repo)] = tag
	return tag
}

func prefixedGitHubTag(repo, prefix string) string {
	tag, err := sources.LatestGitHubTag(repo, prefix)
	if err != nil {
		log.Fatalf("Couldn't determine latest tag for repo %s with prefix '%s': %v", repo, prefix, err)
	}
	materials[fmt.Sprintf("github:%s", repo)] = strings.TrimPrefix(tag, prefix)
	return tag
}

func gitTag(repo string) string {
	tag, err := sources.LatestGitTag(repo, "")
	if err != nil {
		log.Fatalf("Couldn't determine latest tag for repo %s: %v", repo, err)
	}
	materials[fmt.Sprintf("git:%s", repo)] = tag
	return tag
}

func prefixedGitTag(repo, prefix string) string {
	tag, err := sources.LatestGitTag(repo, prefix)
	if err != nil {
		log.Fatalf("Couldn't determine latest tag for repo %s with prefix '%s': %v", repo, prefix, err)
	}
	materials[fmt.Sprintf("git:%s", repo)] = strings.TrimPrefix(tag, prefix)
	return tag
}

func regexURLContent(name, url, regex string) string {
	res, err := sources.RegexURLContent(url, regex)
	if err != nil {
		log.Fatalf("Couldn't find regex in url '%s'", name)
	}
	materials[fmt.Sprintf("regexurl:%s", name)] = res
	return res
}

func addRelease(funcs template.FuncMap, name string, provider func() (version, url, checksum string)) {
	var version, url, checksum string
	once := sync.Once{}
	check := func() {
		once.Do(func() {
			version, url, checksum = provider()
		})
	}

	funcs[fmt.Sprintf("%s_url", name)] = func() string {
		check()
		materials[name] = version
		return url
	}

	funcs[fmt.Sprintf("%s_checksum", name)] = func() string {
		check()
		return checksum
	}
}

func incrementByOne(x int) int {
	return x + 1
}

func Generate(sourceLink, inBase, inRelativePath, outFile string) ([]Change, error) {
	materials = make(map[string]string)
	oldMaterials := readBillOfMaterials(outFile)
	inFile := filepath.Join(inBase, inRelativePath)

	localTemplateFuncs := copyMap(templateFuncs)
	localTemplateFuncs["get"], localTemplateFuncs["set"], localTemplateFuncs["partial"] = getSet(localTemplateFuncs, make(map[string]interface{}))
	tpl := template.New(inFile)
	tpl.Funcs(localTemplateFuncs)

	if _, err := tpl.ParseFiles(inFile); err != nil {
		return nil, fmt.Errorf("unable to parse template file %s: %v", inFile, err)
	}

	writer := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(writer, filepath.Base(inFile), nil); err != nil {
		return nil, fmt.Errorf("unable to render template file %s: %v", outFile, err)
	}

	bom, _ := json.Marshal(materials)
	header := fmt.Sprintf("# Generated from %s%s\n# BOM: %s\n\n", sourceLink, inRelativePath, bom)

	content := append([]byte(header), writer.Bytes()...)
	if err := os.WriteFile(outFile, content, os.FileMode(0600)); err != nil {
		return nil, fmt.Errorf("unable to write container file to %s: %v", outFile, err)
	}

	return diffMaterials(oldMaterials, materials), nil
}
