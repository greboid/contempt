package sources

import (
	"log"
	"net/url"
)

func LatestAlpineRelease() (latest string, downloadUrl string, checksum string) {
	alpineBaseUrl, err := url.JoinPath(*alpineMirror, "latest-stable/releases/x86_64/")
	if err != nil {
		log.Fatalf("Unable to build path to alpine repo: %v", err)
	}

	var (
		alpineReleaseIndex = alpineBaseUrl + "latest-releases.yaml"
		alpineReleaseTitle = "Mini root filesystem"
	)

	var releases []struct {
		Title    string `yaml:"title"`
		File     string `yaml:"file"`
		Checksum string `yaml:"sha256"`
		Version  string `yaml:"version"`
	}

	if err := DownloadYaml(alpineReleaseIndex, &releases); err != nil {
		log.Fatalf("Unable to download Alpine release information: %v", err)
	}

	for i := range releases {
		if releases[i].Title == alpineReleaseTitle {
			return releases[i].Version, alpineBaseUrl + releases[i].File, releases[i].Checksum
		}
	}

	log.Fatalf("No Alpine release found matching '%s'", alpineReleaseTitle)
	return
}
