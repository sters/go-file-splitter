package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sters/go-test-file-splitter/splitter"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSplit Go test files by individual test functions.\n")
		fmt.Fprintf(os.Stderr, "Each TestXxxx function will be extracted into its own file.\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("go-test-file-splitter version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	directory := flag.Arg(0)

	if err := splitter.SplitTestFiles(directory); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
