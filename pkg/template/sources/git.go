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
	gitTagUser = flag.String("git-tag-user", "", "Username to use when querying git tags")
	gitTagPass = flag.String("git-tag-pass", "", "Password to use when querying git tags")
)

func GitSource() template.FunctionSource {
	return func(writer template.BomWriter) tt.FuncMap {
		return tt.FuncMap{
			"git_tag": func(repo string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					repo,
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: true,
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("git:%s", repo), tag)
				return tag, nil
			},

			"prefixed_git_tag": func(repo, prefix string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					repo,
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: true,
							TrimPrefixes:     []string{prefix},
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("git:%s", repo), strings.TrimPrefix(tag, prefix))
				return tag, nil
			},

			"github_tag": func(repo string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					fmt.Sprintf("https://github.com/%s", repo),
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: true,
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("github:%s", repo), tag)
				return tag, nil
			},

			"unreleased_git_tag": func(repo string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					repo,
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: false,
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("git:%s", repo), tag)
				return tag, nil
			},

			"prefixed_unreleased_git_tag": func(repo, prefix string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					repo,
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: false,
							TrimPrefixes:     []string{prefix},
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("git:%s", repo), strings.TrimPrefix(tag, prefix))
				return tag, nil
			},

			"prefixed_github_tag": func(repo, prefix string) (string, error) {
				tag, _, err := latest.GitTag(
					context.Background(),
					fmt.Sprintf("https://github.com/%s", repo),
					&latest.GitTagOptions{
						Username: *gitTagUser,
						Password: *gitTagPass,
						TagOptions: latest.TagOptions{
							IgnoreDates:      true,
							IgnoreErrors:     true,
							IgnorePreRelease: true,
							TrimPrefixes:     []string{prefix},
						},
					},
				)
				if err != nil {
					return "", err
				}
				tag = writer.Write(fmt.Sprintf("github:%s", repo), strings.TrimPrefix(tag, prefix))
				return tag, nil
			},
		}
	}
}
