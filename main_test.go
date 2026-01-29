package main

import (
	"regexp"
	"testing"

	"github.com/chmouel/go-better-html-coverage/internal/model"
)

func TestArrayFlags(t *testing.T) {
	var flags arrayFlags

	// Test empty state
	if flags.String() != "" {
		t.Errorf("expected empty string for empty flags, got %s", flags.String())
	}

	// Test adding values
	if err := flags.Set("pattern1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := flags.Set("pattern2"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
	if flags[0] != "pattern1" {
		t.Errorf("expected pattern1, got %s", flags[0])
	}
	if flags[1] != "pattern2" {
		t.Errorf("expected pattern2, got %s", flags[1])
	}

	// Test String method
	expected := "pattern1,pattern2"
	if flags.String() != expected {
		t.Errorf("expected %s, got %s", expected, flags.String())
	}
}

func TestFilterByRegex(t *testing.T) {
	data := &model.CoverageData{
		Files: []model.FileData{
			{ID: 0, Path: "internal/parser/parser.go", Coverage: []int{0, 2}},
			{ID: 1, Path: "internal/parser/mock_parser.go", Coverage: []int{0, 1}},
			{ID: 2, Path: "internal/model/model.go", Coverage: []int{0, 2}},
		},
		Tree: &model.TreeNode{
			Name: ".",
			Type: "dir",
		},
		Summary: model.Summary{
			TotalLines:   3,
			CoveredLines: 2,
			Percent:      66.67,
		},
	}

	tests := []struct {
		name          string
		patterns      []string
		expectedFiles int
		expectError   bool
	}{
		{
			name:          "valid patterns",
			patterns:      []string{`mock_.*\.go$`},
			expectedFiles: 2,
			expectError:   false,
		},
		{
			name:          "multiple patterns",
			patterns:      []string{`mock_`, `model\.go$`},
			expectedFiles: 1,
			expectError:   false,
		},
		{
			name:          "invalid regex",
			patterns:      []string{`[invalid`},
			expectedFiles: 0,
			expectError:   true,
		},
		{
			name:          "no patterns",
			patterns:      []string{},
			expectedFiles: 3,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterByRegex(data, tt.patterns)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Files) != tt.expectedFiles {
				t.Errorf("expected %d files, got %d", tt.expectedFiles, len(result.Files))
			}
		})
	}
}

func TestFilterByRegexInvalidPattern(t *testing.T) {
	data := &model.CoverageData{
		Files: []model.FileData{
			{ID: 0, Path: "test.go", Coverage: []int{0, 2}},
		},
	}

	_, err := filterByRegex(data, []string{"[invalid"})
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
	if err != nil && !regexp.MustCompile(`invalid regex pattern`).MatchString(err.Error()) {
		t.Errorf("error message should mention invalid regex pattern, got: %v", err)
	}
}
