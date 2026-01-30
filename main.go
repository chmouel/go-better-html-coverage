package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/chmouel/go-better-html-coverage/internal/badge"
	"github.com/chmouel/go-better-html-coverage/internal/generator"
	"github.com/chmouel/go-better-html-coverage/internal/model"
	"github.com/chmouel/go-better-html-coverage/internal/parser"
)

type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func main() {
	var (
		profilePath     string
		basePath        string
		outputPath      string
		badgePath       string
		badgeThresholds string
		srcRoot         string
		ref             string
		noSyntax        bool
		noOpen          bool
		quiet           bool
		excludePatterns arrayFlags
	)

	flag.StringVar(&profilePath, "profile", "coverage.out", "coverage profile path")
	flag.StringVar(&basePath, "base", "", "base coverage profile for diff comparison")
	flag.StringVar(&outputPath, "o", "-", "output HTML file")
	flag.StringVar(&badgePath, "badge", "", "output SVG badge file")
	flag.StringVar(&badgeThresholds, "badge-threshold", "40,70", "badge color thresholds (red,yellow) e.g., 40,70")
	flag.StringVar(&srcRoot, "src", ".", "source root directory")
	flag.StringVar(&ref, "ref", "", "git ref or range to filter coverage")
	flag.BoolVar(&noSyntax, "no-syntax", false, "disable syntax highlighting by default")
	flag.BoolVar(&noOpen, "n", false, "do not open browser")
	flag.BoolVar(&quiet, "q", false, "quiet mode: suppress non-error output")
	flag.Var(&excludePatterns, "exclude", "regex pattern to exclude files (can be repeated)")
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

	// Compute diff if base profile is provided
	if basePath != "" {
		baseData, err := parser.Parse(basePath, srcRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing base coverage: %v\n", err)
			os.Exit(1)
		}
		data = parser.ComputeDiff(baseData, data)
	}

	if ref != "" {
		changedFiles, err := gitChangedFiles(srcRoot, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving git changes: %v\n", err)
			os.Exit(1)
		}
		data = parser.FilterByPaths(data, changedFiles)
	}

	if len(excludePatterns) > 0 {
		data, err = filterByRegex(data, excludePatterns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying exclusion patterns: %v\n", err)
			os.Exit(1)
		}
		if len(data.Files) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: all files excluded by patterns\n")
			os.Exit(1)
		}
	}

	// Generate HTML report
	opts := generator.Options{NoSyntax: noSyntax}
	if err := generator.Generate(data, outputPath, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Coverage report written to %s\n", outputPath)
		if data.IsDiffMode && data.DiffSummary != nil {
			fmt.Fprintf(os.Stderr, "Coverage: %.1f%% (Δ%+.1f%% from base)\n",
				data.Summary.Percent,
				data.DiffSummary.DeltaPercent)
			fmt.Fprintf(os.Stderr, "Changes: +%d newly covered, -%d regressions\n",
				data.DiffSummary.NewlyCoveredLines,
				data.DiffSummary.NewlyUncoveredLines)
		} else {
			fmt.Fprintf(os.Stderr, "Coverage: %.1f%% (%d/%d lines)\n",
				data.Summary.Percent,
				data.Summary.CoveredLines,
				data.Summary.TotalLines)
		}
	}

	// Generate badge if requested
	if badgePath != "" {
		thresholds, err := parseThresholds(badgeThresholds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing badge thresholds: %v\n", err)
			os.Exit(1)
		}
		if err := badge.GenerateBadge(data.Summary.Percent, badgePath, thresholds); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating badge: %v\n", err)
			os.Exit(1)
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, "Coverage badge written to %s\n", badgePath)
		}
	}

	// Open in browser unless -n flag is set
	if !noOpen {
		openBrowser(outputPath)
	}
}

func filterByRegex(data *model.CoverageData, patterns []string) (*model.CoverageData, error) {
	var regexps []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		regexps = append(regexps, re)
	}
	return parser.FilterByRegex(data, regexps), nil
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

func parseThresholds(input string) (badge.Thresholds, error) {
	parts := strings.Split(input, ",")
	if len(parts) != 2 {
		return badge.Thresholds{}, fmt.Errorf("expected format: red,yellow (e.g., 40,70)")
	}

	red, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return badge.Thresholds{}, fmt.Errorf("invalid red threshold: %w", err)
	}

	yellow, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return badge.Thresholds{}, fmt.Errorf("invalid yellow threshold: %w", err)
	}

	if red < 0 || red > 100 || yellow < 0 || yellow > 100 {
		return badge.Thresholds{}, fmt.Errorf("thresholds must be between 0 and 100")
	}

	if red >= yellow {
		return badge.Thresholds{}, fmt.Errorf("red threshold must be less than yellow threshold")
	}

	return badge.Thresholds{
		Red:    red,
		Yellow: yellow,
	}, nil
}
