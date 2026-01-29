package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chmouel/go-better-html-coverage/internal/generator"
	"github.com/chmouel/go-better-html-coverage/internal/parser"
)

func main() {
	var (
		profilePath string
		outputPath  string
		srcRoot     string
	)

	flag.StringVar(&profilePath, "profile", "coverage.out", "coverage profile path")
	flag.StringVar(&outputPath, "o", "coverage.html", "output HTML file")
	flag.StringVar(&srcRoot, "src", ".", "source root directory")
	flag.Parse()

	// Parse coverage data
	data, err := parser.Parse(profilePath, srcRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing coverage: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML report
	if err := generator.Generate(data, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Coverage report written to %s\n", outputPath)
	fmt.Printf("Coverage: %.1f%% (%d/%d lines)\n",
		data.Summary.Percent,
		data.Summary.CoveredLines,
		data.Summary.TotalLines)
}
