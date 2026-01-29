package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chmouel/go-better-html-coverage/internal/model"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "coverage.html")

	id := 0
	data := &model.CoverageData{
		Files: []model.FileData{
			{
				ID:       0,
				Path:     "main.go",
				Lines:    []string{"package main", "", "func main() {}", ""},
				Coverage: []int{0, 0, 2, 0},
			},
		},
		Tree: &model.TreeNode{
			Name: ".",
			Type: "dir",
			Children: []*model.TreeNode{
				{Name: "main.go", Type: "file", FileID: &id},
			},
		},
		Summary: model.Summary{
			TotalLines:   1,
			CoveredLines: 1,
			Percent:      100.0,
		},
	}

	err := Generate(data, outputPath, Options{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	htmlStr := string(content)

	// Check for HTML structure
	if !strings.Contains(htmlStr, "<!doctype html>") {
		t.Error("output should contain DOCTYPE")
	}
	if !strings.Contains(htmlStr, "<title>Coverage Report</title>") {
		t.Error("output should contain title")
	}

	// Check for embedded CSS
	if !strings.Contains(htmlStr, "<style>") {
		t.Error("output should contain embedded CSS")
	}

	// Check for embedded JS
	if !strings.Contains(htmlStr, "<script>") {
		t.Error("output should contain embedded JS")
	}

	// Check for coverage data JSON
	if !strings.Contains(htmlStr, "window.COVERAGE_DATA") {
		t.Error("output should contain COVERAGE_DATA")
	}
	if !strings.Contains(htmlStr, "main.go") {
		t.Error("output should contain file path in JSON data")
	}
}

func TestGenerateMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "coverage.html")

	id0 := 0
	id1 := 1
	data := &model.CoverageData{
		Files: []model.FileData{
			{
				ID:       0,
				Path:     "pkg/foo.go",
				Lines:    []string{"package pkg", "func Foo() {}"},
				Coverage: []int{0, 2},
			},
			{
				ID:       1,
				Path:     "pkg/bar.go",
				Lines:    []string{"package pkg", "func Bar() {}"},
				Coverage: []int{0, 1},
			},
		},
		Tree: &model.TreeNode{
			Name: ".",
			Type: "dir",
			Children: []*model.TreeNode{
				{
					Name: "pkg",
					Type: "dir",
					Children: []*model.TreeNode{
						{Name: "bar.go", Type: "file", FileID: &id1},
						{Name: "foo.go", Type: "file", FileID: &id0},
					},
				},
			},
		},
		Summary: model.Summary{
			TotalLines:   2,
			CoveredLines: 1,
			Percent:      50.0,
		},
	}

	err := Generate(data, outputPath, Options{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(outputPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	htmlStr := string(content)

	// Check both files are in output
	if !strings.Contains(htmlStr, "foo.go") {
		t.Error("output should contain foo.go")
	}
	if !strings.Contains(htmlStr, "bar.go") {
		t.Error("output should contain bar.go")
	}
	if !strings.Contains(htmlStr, "pkg") {
		t.Error("output should contain pkg directory")
	}
}

func TestGenerateEmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "coverage.html")

	data := &model.CoverageData{
		Files: []model.FileData{},
		Tree: &model.TreeNode{
			Name:     ".",
			Type:     "dir",
			Children: []*model.TreeNode{},
		},
		Summary: model.Summary{
			TotalLines:   0,
			CoveredLines: 0,
			Percent:      0,
		},
	}

	err := Generate(data, outputPath, Options{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}
}

func TestGenerateInvalidPath(t *testing.T) {
	data := &model.CoverageData{
		Files: []model.FileData{},
		Tree: &model.TreeNode{
			Name:     ".",
			Type:     "dir",
			Children: []*model.TreeNode{},
		},
		Summary: model.Summary{},
	}

	err := Generate(data, "/nonexistent/directory/coverage.html", Options{})
	if err == nil {
		t.Error("expected error for invalid output path")
	}
}

func TestGenerateOverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "coverage.html")

	// Write initial content
	if err := os.WriteFile(outputPath, []byte("old content"), 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to write initial file: %v", err)
	}

	data := &model.CoverageData{
		Files: []model.FileData{},
		Tree: &model.TreeNode{
			Name:     ".",
			Type:     "dir",
			Children: []*model.TreeNode{},
		},
		Summary: model.Summary{},
	}

	err := Generate(data, outputPath, Options{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(outputPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(content) == "old content" {
		t.Error("file should have been overwritten")
	}
	if !strings.Contains(string(content), "<!doctype html>") {
		t.Error("file should contain new HTML content")
	}
}

func TestGenerateSpecialCharactersInCode(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "coverage.html")

	id := 0
	data := &model.CoverageData{
		Files: []model.FileData{
			{
				ID:   0,
				Path: "main.go",
				Lines: []string{
					"package main",
					`fmt.Println("<script>alert('xss')</script>")`,
					`var x = "a & b < c > d"`,
				},
				Coverage: []int{0, 2, 2},
			},
		},
		Tree: &model.TreeNode{
			Name: ".",
			Type: "dir",
			Children: []*model.TreeNode{
				{Name: "main.go", Type: "file", FileID: &id},
			},
		},
		Summary: model.Summary{
			TotalLines:   2,
			CoveredLines: 2,
			Percent:      100.0,
		},
	}

	err := Generate(data, outputPath, Options{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(outputPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// The JSON should be properly escaped
	htmlStr := string(content)
	if !strings.Contains(htmlStr, "window.COVERAGE_DATA") {
		t.Error("output should contain COVERAGE_DATA")
	}
}
