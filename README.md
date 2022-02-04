# Contempt

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

Note: when using multiple registries, you will probably want to avoid specifying the `registery-user`
and `registry-pass` flags as there is no way to pass in multiple sets of credentials. If the flags
are not specified, then contempt will instead use the credentials saved by docker or podman. 

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
```

Returns the URL and checksum for the latest release of Postgres 13 or 14.

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

## Example

Check out [csmith/dockerfiles](https://github.com/csmith/dockerfiles) for a collection of
templates and outputs generated using contempt.
