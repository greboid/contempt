package sources

import (
	"context"
	"flag"
	"fmt"
	"strings"
	tt "text/template"

	"github.com/csmith/contempt/pkg/template"
	"github.com/csmith/latest/v3"
)

var (
	registryUser = flag.String("registry-user", "", "Username to use when querying the container registry")
	registryPass = flag.String("registry-pass", "", "Password to use when querying the container registry")
)

func ImageSource(registry string) template.FunctionSource {
	return func(writer template.BomWriter) tt.FuncMap {
		return tt.FuncMap{
			"registry": func() string { return registry },
			"image": func(ref string) (string, error) {
				digest, err := latest.ImageDigest(context.Background(), ref, &latest.ImageOptions{
					Registry: registry,
					Username: *registryUser,
					Password: *registryPass,
				})

				if err != nil {
					return "", err
				}

				var image string
				if index := strings.IndexByte(ref, '.'); index != -1 && index < strings.IndexByte(ref, '/') {
					image = ref
				} else {
					image = fmt.Sprintf("%s/%s", registry, ref)
				}

				digest = writer.Write(fmt.Sprintf("image:%s", ref), strings.TrimPrefix(digest, "sha256:"))
				return fmt.Sprintf("%s@sha256:%s", image, digest), nil
			},
		}
	}
}
