package main

import (
	"bytes"

	"github.com/csmith/contempt/sources"
)

// Processor is responsible for processing a single type of Contempt directive within a Dockerfile/Containerfile.
type Processor interface {
	// Take returns the length of any previously generated instructions at the start of the buffer.
	// Take may freely read from the given buffer. If there are no relevant instructions, 0 should be returned.
	Take(buf *bytes.Buffer) (int, error)

	// Write generates new output in the buffer using the given arguments.
	// TODO: This should also supply a BOM
	Write(args []string, buf *bytes.Buffer) error
}

var processors = map[string]Processor{
	"from": NewFromProcessor(sources.LatestDigest),
}
