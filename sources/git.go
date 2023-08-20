package sources

import (
	"fmt"
	"github.com/csmith/gitrefs"
)

// LatestGitHubTag uses the GitHub API to find the tag for the latest stable release.
func LatestGitHubTag(repo string, prefix string) (string, error) {
	return LatestGitTag(fmt.Sprintf("https://github.com/%s", repo), prefix)
}

// LatestGitTag queries a remote git repository to find the latest semver tag, optionally stripping the given prefix
// from tags before processing.
func LatestGitTag(repo string, prefix string) (string, error) {
	tag, _, err := gitrefs.LatestTagIgnoringPrefix(repo, prefix)
	if err != nil {
		return "", err
	}

	return tag, nil
}
