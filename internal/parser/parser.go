package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chmouel/go-better-html-coverage/internal/model"
	"golang.org/x/tools/cover"
)

// Parse reads a coverage profile and source files, returning CoverageData.
func Parse(profilePath, srcRoot string) (*model.CoverageData, error) {
	profiles, err := cover.ParseProfiles(profilePath)
	if err != nil {
		return nil, fmt.Errorf("parsing coverage profile: %w", err)
	}

	// Detect module path from go.mod
	modPath, err := detectModulePath(srcRoot)
	if err != nil {
		return nil, fmt.Errorf("detecting module path: %w", err)
	}

	var files []model.FileData
	totalLines := 0
	coveredLines := 0

	for i, p := range profiles {
		// Convert module path to file path
		relPath := strings.TrimPrefix(p.FileName, modPath+"/")
		if relPath == p.FileName {
			// File might be in the root of the module
			relPath = p.FileName
		}

		fullPath := filepath.Join(srcRoot, relPath)
		lines, err := readLines(fullPath)
		if err != nil {
			// Try stripping module prefix differently
			parts := strings.SplitN(p.FileName, "/", 4)
			if len(parts) >= 4 {
				altPath := filepath.Join(srcRoot, parts[3])
				lines, err = readLines(altPath)
				if err != nil {
					continue // Skip files we can't read
				}
				relPath = parts[3]
			} else {
				continue
			}
		}

		coverage := computeLineCoverage(lines, p.Blocks)
		fd := model.FileData{
			ID:       i,
			Path:     relPath,
			Lines:    lines,
			Coverage: coverage,
		}
		files = append(files, fd)

		// Compute stats
		for _, c := range coverage {
			if c > 0 {
				totalLines++
				if c == 2 {
					coveredLines++
				}
			}
		}
	}

	tree := buildTree(files)

	percent := 0.0
	if totalLines > 0 {
		percent = float64(coveredLines) / float64(totalLines) * 100
	}

	return &model.CoverageData{
		Files: files,
		Tree:  tree,
		Summary: model.Summary{
			TotalLines:   totalLines,
			CoveredLines: coveredLines,
			Percent:      percent,
		},
	}, nil
}

// FilterByPaths filters coverage data to only include files in the provided set.
func FilterByPaths(data *model.CoverageData, allowed map[string]struct{}) *model.CoverageData {
	if data == nil {
		return nil
	}

	filteredFiles := make([]model.FileData, 0, len(data.Files))
	totalLines := 0
	coveredLines := 0

	for _, file := range data.Files {
		if _, ok := allowed[file.Path]; !ok {
			continue
		}
		file.ID = len(filteredFiles)
		filteredFiles = append(filteredFiles, file)

		for _, c := range file.Coverage {
			if c > 0 {
				totalLines++
				if c == 2 {
					coveredLines++
				}
			}
		}
	}

	tree := buildTree(filteredFiles)
	percent := 0.0
	if totalLines > 0 {
		percent = float64(coveredLines) / float64(totalLines) * 100
	}

	return &model.CoverageData{
		Files: filteredFiles,
		Tree:  tree,
		Summary: model.Summary{
			TotalLines:   totalLines,
			CoveredLines: coveredLines,
			Percent:      percent,
		},
	}
}

func detectModulePath(srcRoot string) (string, error) {
	goModPath := filepath.Join(srcRoot, "go.mod")
	f, err := os.Open(goModPath) //nolint:gosec // path is from srcRoot argument
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if modPath, found := strings.CutPrefix(line, "module "); found {
			return modPath, nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path) //nolint:gosec // path is from coverage profile
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func computeLineCoverage(lines []string, blocks []cover.ProfileBlock) []int {
	coverage := make([]int, len(lines))

	for _, b := range blocks {
		for line := b.StartLine; line <= b.EndLine && line <= len(lines); line++ {
			idx := line - 1 // Convert to 0-indexed
			if idx < 0 || idx >= len(coverage) {
				continue
			}
			if b.NumStmt > 0 {
				if b.Count > 0 {
					coverage[idx] = 2 // covered
				} else if coverage[idx] == 0 {
					coverage[idx] = 1 // uncovered (but has statements)
				}
			}
		}
	}
	return coverage
}

func buildTree(files []model.FileData) *model.TreeNode {
	root := &model.TreeNode{
		Name:     ".",
		Type:     "dir",
		Children: []*model.TreeNode{},
	}

	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		insertPath(root, parts, f.ID)
	}

	sortTree(root)
	return root
}

func insertPath(node *model.TreeNode, parts []string, fileID int) {
	if len(parts) == 0 {
		return
	}

	name := parts[0]
	isFile := len(parts) == 1

	// Find existing child
	var child *model.TreeNode
	for _, c := range node.Children {
		if c.Name == name {
			child = c
			break
		}
	}

	if child == nil {
		child = &model.TreeNode{
			Name: name,
		}
		if isFile {
			child.Type = "file"
			id := fileID
			child.FileID = &id
		} else {
			child.Type = "dir"
			child.Children = []*model.TreeNode{}
		}
		node.Children = append(node.Children, child)
	}

	if !isFile {
		insertPath(child, parts[1:], fileID)
	}
}

func sortTree(node *model.TreeNode) {
	if node.Children == nil {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		// Directories first, then files
		if node.Children[i].Type != node.Children[j].Type {
			return node.Children[i].Type == "dir"
		}
		return node.Children[i].Name < node.Children[j].Name
	})

	for _, c := range node.Children {
		sortTree(c)
	}
}
