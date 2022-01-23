# Unreleased

# v1.3.0

- Added `-push-retries` flag to specify how many times a failed push should be retried. Defaults to 2.

# v1.2.0

- Added option (enabled by default) to print workflow commands for GitHub Actions to group log output.
- Skip 'conflicts with' dependencies when resolving Alpine packages (thanks @Greboid)

# v1.1.0

- Make the project build order deterministic.
- LatestGitHubTag now uses [gitrefs](https://github.com/csmith/gitrefs) to get the latest tag instead of the GitHub API.
- Fix "Generated from" header when running with absolute paths

# v1.0.4

- Fix infinite loop if running with paths other than ".", or if projects are nested multiple directories deep.
- Explicitly error if dependencies can't be resolved.
- Improved error message if GitHub tags couldn't be resolved.

# v1.0.3

- Fixed image digests being truncated to "sha256:" and a single character in commit messages
- Increased size of versions shown in commit messages from 8 to 12

# v1.0.2

- Fixed commit messages showing old/new versions the wrong way around

# v1.0.1

- Flags can be specified as env vars

# v1.0.0

- Initial version
