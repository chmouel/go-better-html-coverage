package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chmouel/go-better-html-coverage/internal/generator"
	"github.com/chmouel/go-better-html-coverage/internal/parser"
)

func main() {
	var (
		profilePath string
		outputPath  string
		srcRoot     string
		ref         string
		noSyntax    bool
		noOpen      bool
	)

	flag.StringVar(&profilePath, "profile", "coverage.out", "coverage profile path")
	flag.StringVar(&outputPath, "o", "-", "output HTML file")
	flag.StringVar(&srcRoot, "src", ".", "source root directory")
	flag.StringVar(&ref, "ref", "", "git ref or range to filter coverage")
	flag.BoolVar(&noSyntax, "no-syntax", false, "disable syntax highlighting by default")
	flag.BoolVar(&noOpen, "n", false, "do not open browser")
	flag.Parse()

	// if outputPath is "-", it means stdout then don't try to open browser
	if outputPath == "-" {
		noOpen = true
	}

	// Parse coverage data
	data, err := parser.Parse(profilePath, srcRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing coverage: %v\n", err)
		os.Exit(1)
	}

	if ref != "" {
		changedFiles, err := gitChangedFiles(srcRoot, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving git changes: %v\n", err)
			os.Exit(1)
		}
		data = parser.FilterByPaths(data, changedFiles)
	}

	// Generate HTML report
	opts := generator.Options{NoSyntax: noSyntax}
	if err := generator.Generate(data, outputPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Coverage report written to %s\n", outputPath)
	fmt.Fprintf(os.Stderr, "Coverage: %.1f%% (%d/%d lines)\n",
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
	//nolint:gosec // G204: absPath is from filepath.Abs of our output file
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

func gitChangedFiles(repoRoot, ref string) (map[string]struct{}, error) {
	rangeSpec := ref
	if !strings.Contains(ref, "..") {
		rangeSpec = ref + "^.." + ref
	}

	//nolint:gosec // G204: rangeSpec is from user input but used safely
	cmd := exec.Command("git", "-C", repoRoot, "diff", "--name-only", "--diff-filter=ACMR", rangeSpec)
	output, err := cmd.Output()
	if err != nil {
		// Fallback for root commits (no parent)
		if !strings.Contains(ref, "..") {
			fallback := exec.Command("git", "-C", repoRoot, "show", "--pretty=", "--name-only", "--diff-filter=ACMR", ref)
			output, err = fallback.Output()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	files := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		files[line] = struct{}{}
	}
	return files, nil
}
