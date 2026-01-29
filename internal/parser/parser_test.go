package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chmouel/go-better-html-coverage/internal/model"
	"golang.org/x/tools/cover"
)

func TestComputeLineCoverage(t *testing.T) {
	lines := []string{
		"package main",
		"",
		"func main() {",
		"    fmt.Println(\"hello\")",
		"}",
	}

	blocks := []cover.ProfileBlock{
		{StartLine: 3, EndLine: 5, NumStmt: 2, Count: 1},
	}

	coverage := computeLineCoverage(lines, blocks)

	if len(coverage) != len(lines) {
		t.Errorf("expected coverage length %d, got %d", len(lines), len(coverage))
	}

	// Lines 3-5 should be covered (index 2-4)
	if coverage[2] != 2 {
		t.Errorf("line 3 should be covered (2), got %d", coverage[2])
	}
	if coverage[3] != 2 {
		t.Errorf("line 4 should be covered (2), got %d", coverage[3])
	}
	if coverage[4] != 2 {
		t.Errorf("line 5 should be covered (2), got %d", coverage[4])
	}

	// Lines 1-2 should have no statements
	if coverage[0] != 0 {
		t.Errorf("line 1 should have no stmt (0), got %d", coverage[0])
	}
}

func TestComputeLineCoverageUncovered(t *testing.T) {
	lines := []string{"a", "b", "c"}
	blocks := []cover.ProfileBlock{
		{StartLine: 2, EndLine: 2, NumStmt: 1, Count: 0},
	}

	coverage := computeLineCoverage(lines, blocks)
	if coverage[1] != 1 {
		t.Errorf("line 2 should be uncovered (1), got %d", coverage[1])
	}
}

func TestComputeLineCoverageOutOfBounds(t *testing.T) {
	lines := []string{"a", "b"}
	blocks := []cover.ProfileBlock{
		{StartLine: 5, EndLine: 10, NumStmt: 1, Count: 1},
	}

	coverage := computeLineCoverage(lines, blocks)
	// Should not panic, coverage should be all zeros
	for i, c := range coverage {
		if c != 0 {
			t.Errorf("line %d should be 0, got %d", i+1, c)
		}
	}
}

func TestBuildTree(t *testing.T) {
	files := []model.FileData{
		{ID: 0, Path: "internal/parser/parser.go"},
		{ID: 1, Path: "internal/parser/parser_test.go"},
		{ID: 2, Path: "main.go"},
		{ID: 3, Path: "internal/model/model.go"},
	}

	root := buildTree(files)

	if root.Name != "." {
		t.Errorf("root name should be '.', got %s", root.Name)
	}
	if root.Type != "dir" {
		t.Errorf("root type should be 'dir', got %s", root.Type)
	}

	// Root should have 2 children: internal (dir) and main.go (file)
	if len(root.Children) != 2 {
		t.Errorf("root should have 2 children, got %d", len(root.Children))
	}

	// After sorting, directories come first
	if root.Children[0].Name != "internal" {
		t.Errorf("first child should be 'internal', got %s", root.Children[0].Name)
	}
	if root.Children[0].Type != "dir" {
		t.Errorf("internal should be dir, got %s", root.Children[0].Type)
	}

	if root.Children[1].Name != "main.go" {
		t.Errorf("second child should be 'main.go', got %s", root.Children[1].Name)
	}
	if root.Children[1].Type != "file" {
		t.Errorf("main.go should be file, got %s", root.Children[1].Type)
	}
	if root.Children[1].FileID == nil || *root.Children[1].FileID != 2 {
		t.Errorf("main.go should have FileID=2")
	}
}

func TestBuildTreeSorting(t *testing.T) {
	files := []model.FileData{
		{ID: 0, Path: "z.go"},
		{ID: 1, Path: "a.go"},
		{ID: 2, Path: "m/file.go"},
		{ID: 3, Path: "b/file.go"},
	}

	root := buildTree(files)

	// Directories first (b, m), then files (a.go, z.go)
	expected := []struct {
		name    string
		nodeTyp string
	}{
		{"b", "dir"},
		{"m", "dir"},
		{"a.go", "file"},
		{"z.go", "file"},
	}

	if len(root.Children) != len(expected) {
		t.Fatalf("expected %d children, got %d", len(expected), len(root.Children))
	}

	for i, exp := range expected {
		if root.Children[i].Name != exp.name {
			t.Errorf("child %d: expected name %s, got %s", i, exp.name, root.Children[i].Name)
		}
		if root.Children[i].Type != exp.nodeTyp {
			t.Errorf("child %d: expected type %s, got %s", i, exp.nodeTyp, root.Children[i].Type)
		}
	}
}

func TestBuildTreeDeepNesting(t *testing.T) {
	files := []model.FileData{
		{ID: 0, Path: "a/b/c/d/file.go"},
	}

	root := buildTree(files)

	// Traverse the tree
	node := root
	expectedPath := []string{"a", "b", "c", "d", "file.go"}
	for i, name := range expectedPath {
		if len(node.Children) != 1 {
			t.Fatalf("at level %d, expected 1 child, got %d", i, len(node.Children))
		}
		node = node.Children[0]
		if node.Name != name {
			t.Errorf("at level %d, expected name %s, got %s", i, name, node.Name)
		}
	}

	if node.Type != "file" {
		t.Errorf("leaf should be file, got %s", node.Type)
	}
	if node.FileID == nil || *node.FileID != 0 {
		t.Errorf("leaf should have FileID=0")
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	files := []model.FileData{}
	root := buildTree(files)

	if root.Name != "." {
		t.Errorf("root name should be '.', got %s", root.Name)
	}
	if len(root.Children) != 0 {
		t.Errorf("root should have no children, got %d", len(root.Children))
	}
}

func TestDetectModulePath(t *testing.T) {
	tmpDir := t.TempDir()
	goMod := `module example.com/test

go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	modPath, err := detectModulePath(tmpDir)
	if err != nil {
		t.Fatalf("detectModulePath failed: %v", err)
	}
	if modPath != "example.com/test" {
		t.Errorf("expected module path 'example.com/test', got '%s'", modPath)
	}
}

func TestDetectModulePathNoModule(t *testing.T) {
	tmpDir := t.TempDir()
	goMod := `go 1.21
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	_, err = detectModulePath(tmpDir)
	if err == nil {
		t.Error("expected error for missing module directive")
	}
}

func TestDetectModulePathNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := detectModulePath(tmpDir)
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestReadLines(t *testing.T) {
	tmpDir := t.TempDir()
	content := "line1\nline2\nline3"
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte(content), 0o644) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines failed: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" {
		t.Errorf("expected 'line1', got '%s'", lines[0])
	}
}

func TestReadLinesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")
	err := os.WriteFile(path, []byte{}, 0o644) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readLines failed: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestReadLinesNotFound(t *testing.T) {
	_, err := readLines("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testmod

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create source file
	srcContent := `package main

func main() {
	println("hello")
}

func unused() {
	println("unused")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(srcContent), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create main.go: %v", err)
	}

	// Create coverage profile
	// Format: mode: set
	// file:startLine.startCol,endLine.endCol numStmt count
	coverageProfile := `mode: set
testmod/main.go:3.13,5.2 1 1
testmod/main.go:7.14,9.2 1 0
`
	coveragePath := filepath.Join(tmpDir, "coverage.out")
	if err := os.WriteFile(coveragePath, []byte(coverageProfile), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create coverage.out: %v", err)
	}

	// Parse
	data, err := Parse(coveragePath, tmpDir)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify files
	if len(data.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(data.Files))
	}

	file := data.Files[0]
	if file.Path != "main.go" {
		t.Errorf("expected path 'main.go', got '%s'", file.Path)
	}

	// Verify coverage: lines 3-4 covered, lines 7-8 uncovered
	// Line indices are 0-based
	// Source file has 9 lines (including trailing newline parsed as empty)
	if len(file.Coverage) != 9 {
		t.Errorf("expected 9 coverage entries, got %d", len(file.Coverage))
	}

	// Lines 3-4 (index 2-3) should be covered (2)
	if file.Coverage[2] != 2 {
		t.Errorf("line 3 should be covered (2), got %d", file.Coverage[2])
	}
	if file.Coverage[3] != 2 {
		t.Errorf("line 4 should be covered (2), got %d", file.Coverage[3])
	}

	// Lines 7-8 (index 6-7) should be uncovered (1)
	if file.Coverage[6] != 1 {
		t.Errorf("line 7 should be uncovered (1), got %d", file.Coverage[6])
	}
	if file.Coverage[7] != 1 {
		t.Errorf("line 8 should be uncovered (1), got %d", file.Coverage[7])
	}

	// Verify tree
	if data.Tree == nil {
		t.Fatal("expected tree, got nil")
	}
	if data.Tree.Name != "." {
		t.Errorf("expected root name '.', got '%s'", data.Tree.Name)
	}

	// Verify summary
	if data.Summary.TotalLines == 0 {
		t.Error("expected non-zero total lines")
	}
	if data.Summary.CoveredLines == 0 {
		t.Error("expected non-zero covered lines")
	}
	if data.Summary.Percent == 0 {
		t.Error("expected non-zero coverage percent")
	}
}

func TestParseNoGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	coveragePath := filepath.Join(tmpDir, "coverage.out")
	if err := os.WriteFile(coveragePath, []byte("mode: set\n"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create coverage.out: %v", err)
	}

	_, err := Parse(coveragePath, tmpDir)
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
}

func TestParseInvalidProfile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create go.mod: %v", err)
	}

	coveragePath := filepath.Join(tmpDir, "coverage.out")
	if err := os.WriteFile(coveragePath, []byte("invalid content"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to create coverage.out: %v", err)
	}

	_, err := Parse(coveragePath, tmpDir)
	if err == nil {
		t.Error("expected error for invalid coverage profile")
	}
}

func TestFilterByPaths(t *testing.T) {
	data := &model.CoverageData{
		Files: []model.FileData{
			{ID: 0, Path: "a.go", Coverage: []int{0, 2}},
			{ID: 1, Path: "b.go", Coverage: []int{0, 1}},
		},
		Tree: &model.TreeNode{
			Name: ".",
			Type: "dir",
		},
		Summary: model.Summary{
			TotalLines:   2,
			CoveredLines: 1,
			Percent:      50,
		},
	}

	filtered := FilterByPaths(data, map[string]struct{}{"b.go": {}})
	if len(filtered.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(filtered.Files))
	}
	if filtered.Files[0].Path != "b.go" {
		t.Fatalf("expected b.go, got %s", filtered.Files[0].Path)
	}
	if filtered.Files[0].ID != 0 {
		t.Fatalf("expected ID 0, got %d", filtered.Files[0].ID)
	}
	if filtered.Summary.TotalLines != 1 {
		t.Fatalf("expected total lines 1, got %d", filtered.Summary.TotalLines)
	}
	if filtered.Summary.CoveredLines != 0 {
		t.Fatalf("expected covered lines 0, got %d", filtered.Summary.CoveredLines)
	}
}
