# Unreleased

# v1.5.1

- Fixed issue where all projects are ignored if the path is given as `.` (like in
  the examples in the README...)

# v1.5.0

- The `--project` flag can now contain multiple projects separated by commas.
- No longer recurses into directories which start with a `.` (e.g. `.git`) when finding projects.
- Now properly reports errors when finding projects, instead of panicing.
- Added `regex_url_content` template function (thanks @Greboid)

# v1.4.1

- Fix dependency resolution when using fully-qualified names in the `image` template function.

# v1.4.0

- When multiple materials change the commit message is now summarised as "N changes",
  the details are spread over multiple lines, and sorted alphabetically.
- The `image` template function now accepts fully-qualified image names, and will not
  pre-pend the default registry.
- Added `git_tag` and `prefixed_git_tag` template functions.
- Fixed `-push-retries` flag including the original push attempt in the count (i.e.,
  a value of `2` would retry once; a value of `0` would fail without trying.)

# v1.3.1

- `prefixed_github_tag` no longer includes the stripped prefix in the bill of materials.

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
