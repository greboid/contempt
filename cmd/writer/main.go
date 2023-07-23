package main

import (
	"flag"
	"fmt"
	"github.com/csmith/envflag"
	"os"
)

func main() {
	envflag.Parse()

	if flag.NArg() != 1 {
		_, _ = fmt.Fprintf(os.Stderr, "Expected one positional argument: <input dir>\n")
		flag.Usage()
		os.Exit(2)
	}
}
