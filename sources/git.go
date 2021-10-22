package sources

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
)

// LatestGitHubTag uses the GitHub API to find the tag for the latest stable release.
func LatestGitHubTag(repo string, prefix string) (string, error) {
	var releases []struct {
		Name string `json:"name"`
	}

	if err := DownloadJson(fmt.Sprintf("https://api.github.com/repos/%s/tags", repo), &releases); err != nil {
		return "", err
	}

	best := version.Must(version.NewVersion("0.0.0"))
	bestTag := ""
	for i := range releases {
		v, err := version.NewVersion(strings.TrimPrefix(releases[i].Name, prefix))
		if err == nil && v.GreaterThanOrEqual(best) && v.Prerelease() == "" {
			best = v
			bestTag = releases[i].Name
		}
	}

	if bestTag == "" {
		return "", fmt.Errorf("no stable semver tags found")
	}
	return bestTag, nil
}
