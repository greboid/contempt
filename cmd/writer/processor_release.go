package main

import (
	"bytes"
	"fmt"
	"strings"
)

type ReleaseProcessor struct {
	releases map[string]func() (latest string, url string, checksum string)
}

func NewReleaseProcessor(releases map[string]func() (latest string, url string, checksum string)) *ReleaseProcessor {
	return &ReleaseProcessor{
		releases: releases,
	}
}

func (r *ReleaseProcessor) Take(buf *bytes.Buffer) (int, error) {
	line, _ := buf.ReadString('\n')
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "ARG RELEASE_") {
		return len(line), nil
	}
	return 0, nil
}

func (r *ReleaseProcessor) Write(args []string, buf *bytes.Buffer) error {
	if len(args) != 2 {
		return fmt.Errorf("release requires two args: <project> <url|checksum|version>")
	}

	what := strings.ToUpper(args[1])
	if what != "URL" && what != "CHECKSUM" && what != "VERSION" {
		return fmt.Errorf("invalid option: must be one of url, checksum or version")
	}

	f, ok := r.releases[strings.ToLower(args[0])]
	if !ok {
		return fmt.Errorf("project not found: %s", args[0])
	}

	version, url, checksum := f()
	if what == "URL" {
		buf.WriteString(fmt.Sprintf("ARG RELEASE_%s_URL=\"%s\"", strings.ToUpper(args[0]), url))
	} else if what == "CHECKSUM" {
		buf.WriteString(fmt.Sprintf("ARG RELEASE_%s_CHECKSUM=\"%s\"", strings.ToUpper(args[0]), checksum))
	} else if what == "VERSION" {
		buf.WriteString(fmt.Sprintf("ARG RELEASE_%s_VERSION=\"%s\"", strings.ToUpper(args[0]), version))
	}
	return nil
}
