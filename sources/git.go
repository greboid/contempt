package sources

import (
	"fmt"
	"strings"

	"github.com/csmith/gitrefs"
	"github.com/hashicorp/go-version"
)

// LatestGitHubTag uses the GitHub API to find the tag for the latest stable release.
func LatestGitHubTag(repo string, prefix string) (string, error) {
	return LatestGitTag(fmt.Sprintf("https://github.com/%s", repo), prefix)
}

// LatestGitTag queries a remote git repository to find the latest semver tag, optionally stripping the given prefix
// from tags before processing.
func LatestGitTag(repo string, prefix string) (string, error) {
	refs, err := gitrefs.Fetch(repo)
	if err != nil {
		return "", err
	}

	best := version.Must(version.NewVersion("0.0.0"))
	bestTag := ""
	for ref := range refs {
		if strings.HasPrefix(ref, "refs/tags/") {
			tag := strings.TrimPrefix(ref, "refs/tags/")
			v, err := version.NewVersion(strings.TrimPrefix(tag, prefix))
			if err == nil && v.GreaterThanOrEqual(best) && v.Prerelease() == "" {
				best = v
				bestTag = tag
			}
		}
	}

	if bestTag == "" {
		return "", fmt.Errorf("no stable semver tags found")
	}
	return bestTag, nil
}
