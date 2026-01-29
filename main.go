package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chmouel/go-better-html-coverage/internal/generator"
	"github.com/chmouel/go-better-html-coverage/internal/parser"
)

func main() {
	var (
		profilePath string
		outputPath  string
		srcRoot     string
		noSyntax    bool
		noOpen      bool
	)

	flag.StringVar(&profilePath, "profile", "coverage.out", "coverage profile path")
	flag.StringVar(&outputPath, "o", "coverage.html", "output HTML file")
	flag.StringVar(&srcRoot, "src", ".", "source root directory")
	flag.BoolVar(&noSyntax, "no-syntax", false, "disable syntax highlighting by default")
	flag.BoolVar(&noOpen, "n", false, "do not open browser")
	flag.Parse()

	// Parse coverage data
	data, err := parser.Parse(profilePath, srcRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing coverage: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML report
	opts := generator.Options{NoSyntax: noSyntax}
	if err := generator.Generate(data, outputPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Coverage report written to %s\n", outputPath)
	fmt.Printf("Coverage: %.1f%% (%d/%d lines)\n",
		data.Summary.Percent,
		data.Summary.CoveredLines,
		data.Summary.TotalLines)

	// Open in browser unless -n flag is set
	if !noOpen {
		openBrowser(outputPath)
	}
}

func openBrowser(path string) {
	// Convert to absolute path for file:// URL
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	default:
		return
	}
	_ = cmd.Start()
}
