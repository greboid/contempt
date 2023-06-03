# Contempt

Note: this repository currently contains the released version of contempt
under `cmd/contempt`, and new work-in-progress commands under
`cmd/contempt-generator`, `cmd/contempt-writer` and `cmd/contempt-builder`.
These new commands are not yet finished, and all the documentation below
refers to the old, single command.

---

Contempt is a tool to generate Dockerfiles (or Containerfiles) from templates.

It comes with support for various useful functions for getting the latest versions
of software, expanding out the dependency trees of package managers, and so on.

The most basic invocation of contempt takes a source directory and a destination
directory (these can be the same). Within these directories, it is expected that
each container image is defined in its own subdir, e.g.:

```
input
↳ image1
  ↳ Dockerfile.gotpl
  ↳ something-or-other.patch
↳ image2
  ↳ Dockerfile.gotpl
  
output
↳ image1
  ↳ Dockerfile
↳ image2
  ↳ Dockerfile
```

Contempt's job is to take those template files and generate the plain version
in the output directory:

```shell
go install github.com/csmith/contempt/cmd/contempt@latest
contempt input_dir output_dir
```

Contempt also has options to make a git commit every time an output file, build
the corresponding image using buildah, and push it to a registry:

```shell
contempt -commit -build -push . .
```

You can also limit contempt to a single project:

```shell
contempt -project=image1 . .
```

Other miscellaneous options are available:

```
Usage of contempt:
-build
    [BUILD] Whether to automatically build on successful commit
-commit
    [COMMIT] Whether to automatically git commit each changed file
-force-build
    [FORCE_BUILD] Whether to build projects regardless of changes
-output string
    [OUTPUT] The name of the output files (default "Dockerfile")
-project string
    [PROJECT] The name of a single project to generate, instead of all detected ones
-push
    [PUSH] Whether to automatically push on successful commit
-push-retries int
    [PUSH_RETRIES] How many times to retry pushing an image if it fails (default 2)
-registry string
    [REGISTRY] Registry to use for pushes and pulls (default "reg.c5h.io")
-registry-pass string
    [REGISTRY_PASS] Password to use when querying the container registry
-registry-user string
    [REGISTRY_USER] Username to use when querying the container registry
-source-link string
    [SOURCE_LINK] Link to a browsable version of the source repo (default "https://github.com/example/repo/blob/master/")
-template string
    [TEMPLATE] The name of the template files (default "Dockerfile.gotpl")
-workflow-commands
    [WORKFLOW_COMMANDS] Whether to output GitHub Actions workflow commands to format logs (default true)
```

In practice, you will probably want to set the `-registry` and `-source-link` parameters to point
at the correct place along with the `commit`/`build`/`push` options as required.

## Template functions

Contempt uses Go's built-in [text/template](https://golang.org/pkg/text/template/) package,
and provides the following functions:

### Image

```gotemplate
{{image "alpine"}}
```

Fetches the latest digest for the given image from the configured registry, and returns the fully-qualified
name with the digest (e.g. `reg.c5h.io/alpine@sha256:abcd...........`).

If the image name includes a registry, then it is used as-is regardless of the `registry` flag value:

```gotemplate
{{image "docker.io/library/hello-world"}}
```

Note: see below for information on passing credentials when using more than one registry.

### Registry

```gotemplate
{{registry}}
```

Returns the registry configured with the `-registry` flag.

### Alpine release

```gotemplate
{{alpine_url}}
{{alpine_checksum}}
```

Returns the URL and checksum for the latest release of Alpine.

### Golang release

```gotemplate
{{golang_url}}
{{golang_checksum}}
```

Returns the URL and checksum for the latest release of Golang.

### Postgres release

```gotemplate
{{postgres13_url}}
{{postgres13_checksum}}

{{postgres14_url}}
{{postgres14_checksum}}

{{postgres15_url}}
{{postgres15_checksum}}
```

Returns the URL and checksum for the latest release of Postgres 13, 14 or 15.

### Alpine packages

```gotemplate
RUN apk add --no-cache \
        {{range $key, $value := alpine_packages "ca-certificates" "musl" "tzdata" "rsync" -}}
        {{$key}}={{$value}} \
        {{end}};
```

Given one or more Alpine packages, resolves all of their dependencies and returns a flattened
list of all packages pinned to their current versions.

### GitHub tag

```gotemplate
{{github_tag "csmith/contempt"}}
{{prefixed_github_tag "csmith/contempt" "release-"}}
```

Returns the latest semver tag of the given repository. The "prefixed" variant will discard
the given prefix from tag names before comparing them using semver.

### Git tag

```gotemplate
{{git_tag "https://git.sr.ht/~csmith/example"}}
{{prefixed_git_tag "https://git.sr.ht/~csmith/example" "release-"}}
```

Returns the latest semver tag of the given repository. The "prefixed" variant will discard
the given prefix from tag names before comparing them using semver.

### Regex URL content

```gotemplate
{{regex_url_content "google_button" "https://www.google.com/" "I'm feeling (L[a-z]+)"}}

Requests the given URL over HTTP and attempts to match the regular expression.
Returns the text captured by the first capturing group in the regex.
The first argument is a friendly name used for logging and BOM tracking.
```

## Dealing with registry credentials

There are two cases in which contempt requires credentials: checking the latest digest for an image in a non-public
registry (when the `{{image}}` template function is used), and pushing built images (when the `-push` flag is used).

### Checking digests

You can supply a single set of credentials to use for checking digests using the `-registry-user` and `-registry-pass`
flags (or associated environment variables). If these options aren't passed and the registry is not public, then
credentials will be read from `~/.docker/config.json` if it exists, else `${XDG_RUNTIME_DIR}/containers/auth.json`.

### Pushing

For pushes, contempt expects `buildah` to handle authentication for it. To that end, you will probably want to call
`buildah login` before running contempt. Buildah will also read from `~/.docker/config.json` so a `docker login`
will also suffice.

### GitHub Actions

If you are running contempt using GitHub Actions (or possibly other CI tooling) and need to supply multiple sets
of credentials for the `{{image}}` function, you may encounter a number of inconvenient issues:

- The `XDG_RUNTIME_DIR` env var is not set, and the `/run/user` directory is not writable, meaning `buildah login`
  stores its credentials in `/var/tmp/containers-user-1001/containers/containers/auth.json`. Contempt will not
  read from this location when trying to find credentials for the `{{image}}` function.
- The default actions image comes pre-supplied with a `~/.docker/config.json` with credentials for Docker Hub.
  Because this file exists contempt won't even attempt to read `${XDG_RUNTIME_DIR}/containers/auth.json`, even
  if you've set the environment variable to a sensible value.

The simplest way to deal with this situation is to use `docker login` to write credentials to Docker's config file.

## Example

Check out [csmith/dockerfiles](https://github.com/csmith/dockerfiles) for a collection of
templates and outputs generated using contempt.
