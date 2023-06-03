package main

import (
	"bytes"
	"fmt"
	"strings"
)

type FromProcessor struct {
	digestLookup func(ref string) (string, string, error)
}

func NewFromProcessor(digestLookup func(ref string) (string, string, error)) *FromProcessor {
	return &FromProcessor{digestLookup: digestLookup}
}

func (f *FromProcessor) Take(buf *bytes.Buffer) (int, error) {
	// TODO: This can probably be abstracted
	line, _ := buf.ReadString('\n')
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "FROM") {
		return len(line), nil
	}
	return 0, nil
}

func (f *FromProcessor) Write(args []string, buf *bytes.Buffer) error {
	if len(args) == 0 {
		return fmt.Errorf("no image specified")
	}

	suffix := ""

	if len(args) > 1 {
		if strings.ToUpper(args[1]) != "AS" || len(args) != 3 {
			return fmt.Errorf("invalid arguments: expected '<image>' or '<image> AS <alias>'")
		}

		suffix = fmt.Sprintf(" AS %s", args[2])
	}

	image, digest, err := f.digestLookup(args[0])
	if err != nil {
		return fmt.Errorf("failed to get latest digest for %s: %v", args[0], err)
	}

	_, err = buf.WriteString(fmt.Sprintf("FROM %s@%s%s\n", image, digest, suffix))
	return err
}
