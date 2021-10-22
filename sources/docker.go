package sources

import (
	"flag"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
)

var (
	registry     = flag.String("registry", "reg.c5h.io", "Registry to use for pushes and pulls")
	registryUser = flag.String("registry-user", "", "Username to use when talking to container registries")
	registryPass = flag.String("registry-pass", "", "Password to use when talking to container registries")
)

// LatestDigest finds the latest digest for the given image reference.
// If either the username or password is blank, falls back to using the default docker keychain.
func LatestDigest(ref string) (string, string, error) {
	var authOpt crane.Option

	if *registryUser == "" || *registryPass == "" {
		authOpt = crane.WithAuthFromKeychain(authn.DefaultKeychain)
	} else {
		authOpt = crane.WithAuth(&authn.Basic{
			Username: *registryUser,
			Password: *registryPass,
		})
	}

	image := fmt.Sprintf("%s/%s", *registry, ref)
	digest, err := crane.Digest(image, authOpt)
	return image, digest, err
}

func Registry() string {
	return *registry
}
