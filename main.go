package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sters/go-test-file-splitter/go-public-func-splitter/splitter"
)

var (
	version = "dev"
	commit  = "none"    //nolint:gochecknoglobals
	date    = "unknown" //nolint:gochecknoglobals
)

func main() {
	var (
		showVersion bool
		publicFunc  bool
		testOnly    bool
	)

	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&publicFunc, "public-func", true, "Split public functions into individual files (default)")
	flag.BoolVar(&testOnly, "test-only", false, "Split only test functions")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <directory>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSplit Go files by public functions (default) or test functions.\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("go-public-func-splitter version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	directory := flag.Arg(0)

	// If test-only is specified, it overrides the default public-func mode
	if testOnly {
		publicFunc = false
	}

	var err error
	if publicFunc {
		err = splitter.SplitPublicFunctions(directory)
	} else {
		err = splitter.SplitTestFunctions(directory)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
