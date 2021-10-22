package contempt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"text/template"

	"github.com/csmith/contempt/sources"
)

var templateFuncs template.FuncMap

func init() {
	templateFuncs = template.FuncMap{
		"image":               image,
		"alpine_packages":     alpinePackages,
		"github_tag":          gitHubTag,
		"prefixed_github_tag": prefixedGitHubTag,
		"registry":            sources.Registry,
	}
	addRelease("alpine", sources.LatestAlpineRelease)
	addRelease("golang", sources.LatestGolangRelease)
	addRelease("postgres13", sources.LatestPostgresRelease("13"))
	addRelease("postgres14", sources.LatestPostgresRelease("14"))
}

func image(ref string) string {
	im, digest, err := sources.LatestDigest(ref)
	if err != nil {
		log.Fatalf("Unable to get latest digest for ref %s: %v", ref, err)
	}
	materials[fmt.Sprintf("image:%s", ref)] = digest
	return fmt.Sprintf("%s/%s@%s", im, ref, digest)
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
		log.Fatalf("Couldn't determine latest tag: %v", err)
	}
	materials[fmt.Sprintf("github:%s", repo)] = tag
	return tag
}

func prefixedGitHubTag(repo, prefix string) string {
	tag, err := sources.LatestGitHubTag(repo, prefix)
	if err != nil {
		log.Fatalf("Couldn't determine latest tag: %v", err)
	}
	materials[fmt.Sprintf("github:%s", repo)] = tag
	return tag
}

func addRelease(name string, provider func() (version, url, checksum string)) {
	var version, url, checksum string
	once := sync.Once{}
	check := func() {
		once.Do(func() {
			version, url, checksum = provider()
		})
	}

	templateFuncs[fmt.Sprintf("%s_url", name)] = func() string {
		check()
		materials[name] = version
		return url
	}

	templateFuncs[fmt.Sprintf("%s_checksum", name)] = func() string {
		check()
		return checksum
	}
}

func Generate(sourceLink, inFile, outFile string) ([]Change, error) {
	materials = make(map[string]string)
	oldMaterials := readBillOfMaterials(outFile)

	tpl := template.New(inFile)
	tpl.Funcs(templateFuncs)

	if _, err := tpl.ParseFiles(inFile); err != nil {
		return nil, fmt.Errorf("unable to parse template file %s: %v", inFile, err)
	}

	writer := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(writer, outFile, nil); err != nil {
		return nil, fmt.Errorf("unable to render template file %s: %v", outFile, err)
	}

	bom, _ := json.Marshal(materials)
	header := fmt.Sprintf("# Generated from %s%s\n# BOM: %s\n\n", sourceLink, inFile, bom)

	content := append([]byte(header), writer.Bytes()...)
	if err := os.WriteFile(outFile, content, os.FileMode(0600)); err != nil {
		return nil, fmt.Errorf("unable to write container file to %s: %v", outFile, err)
	}

	return diffMaterials(oldMaterials, materials), nil
}
