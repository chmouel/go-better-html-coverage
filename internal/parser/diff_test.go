package parser

import (
	"testing"

	"github.com/chmouel/go-better-html-coverage/internal/model"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name               string
		base               *model.CoverageData
		current            *model.CoverageData
		wantNewlyCovered   int
		wantNewlyUncovered int
		wantDiffStates     []int // expected diff states for first file
	}{
		{
			name: "newly covered line",
			base: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a", "b"}, Coverage: []int{1, 0}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 0, Percent: 0},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a", "b"}, Coverage: []int{2, 0}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 1, Percent: 100},
			},
			wantNewlyCovered:   1,
			wantNewlyUncovered: 0,
			wantDiffStates:     []int{DiffStateNewlyCovered, DiffStateNoChange},
		},
		{
			name: "regression - was covered now uncovered",
			base: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{2}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 1, Percent: 100},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{1}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 0, Percent: 0},
			},
			wantNewlyCovered:   0,
			wantNewlyUncovered: 1,
			wantDiffStates:     []int{DiffStateNewlyUncovered},
		},
		{
			name: "unchanged covered",
			base: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{2}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 1, Percent: 100},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{2}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 1, Percent: 100},
			},
			wantNewlyCovered:   0,
			wantNewlyUncovered: 0,
			wantDiffStates:     []int{DiffStateUnchangedCovered},
		},
		{
			name: "unchanged uncovered",
			base: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{1}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 0, Percent: 0},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a"}, Coverage: []int{1}},
				},
				Summary: model.Summary{TotalLines: 1, CoveredLines: 0, Percent: 0},
			},
			wantNewlyCovered:   0,
			wantNewlyUncovered: 0,
			wantDiffStates:     []int{DiffStateUnchangedUncovered},
		},
		{
			name: "new file not in base",
			base: &model.CoverageData{
				Files:   []model.FileData{},
				Summary: model.Summary{TotalLines: 0, CoveredLines: 0, Percent: 0},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "new.go", Lines: []string{"a", "b"}, Coverage: []int{2, 1}},
				},
				Summary: model.Summary{TotalLines: 2, CoveredLines: 1, Percent: 50},
			},
			wantNewlyCovered:   1,
			wantNewlyUncovered: 1,
			wantDiffStates:     []int{DiffStateNewlyCovered, DiffStateNewlyUncovered},
		},
		{
			name: "mixed changes",
			base: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a", "b", "c", "d"}, Coverage: []int{2, 1, 2, 0}},
				},
				Summary: model.Summary{TotalLines: 3, CoveredLines: 2, Percent: 66.67},
			},
			current: &model.CoverageData{
				Files: []model.FileData{
					{ID: 0, Path: "foo.go", Lines: []string{"a", "b", "c", "d"}, Coverage: []int{2, 2, 1, 0}},
				},
				Summary: model.Summary{TotalLines: 3, CoveredLines: 2, Percent: 66.67},
			},
			wantNewlyCovered:   1, // line b: 1->2
			wantNewlyUncovered: 1, // line c: 2->1
			wantDiffStates:     []int{DiffStateUnchangedCovered, DiffStateNewlyCovered, DiffStateNewlyUncovered, DiffStateNoChange},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeDiff(tt.base, tt.current)

			if !result.IsDiffMode {
				t.Error("expected IsDiffMode to be true")
			}

			if result.DiffSummary == nil {
				t.Fatal("expected DiffSummary to be non-nil")
			}

			if result.DiffSummary.NewlyCoveredLines != tt.wantNewlyCovered {
				t.Errorf("NewlyCoveredLines = %d, want %d",
					result.DiffSummary.NewlyCoveredLines, tt.wantNewlyCovered)
			}

			if result.DiffSummary.NewlyUncoveredLines != tt.wantNewlyUncovered {
				t.Errorf("NewlyUncoveredLines = %d, want %d",
					result.DiffSummary.NewlyUncoveredLines, tt.wantNewlyUncovered)
			}

			if len(result.Files) > 0 && tt.wantDiffStates != nil {
				file := result.Files[0]
				if len(file.DiffState) != len(tt.wantDiffStates) {
					t.Fatalf("DiffState length = %d, want %d",
						len(file.DiffState), len(tt.wantDiffStates))
				}
				for i, want := range tt.wantDiffStates {
					if file.DiffState[i] != want {
						t.Errorf("DiffState[%d] = %d, want %d", i, file.DiffState[i], want)
					}
				}
			}
		})
	}
}

func TestComputeDiff_DeltaPercent(t *testing.T) {
	base := &model.CoverageData{
		Files: []model.FileData{
			{ID: 0, Path: "foo.go", Lines: []string{"a", "b"}, Coverage: []int{2, 1}},
		},
		Summary: model.Summary{TotalLines: 2, CoveredLines: 1, Percent: 50.0},
	}
	current := &model.CoverageData{
		Files: []model.FileData{
			{ID: 0, Path: "foo.go", Lines: []string{"a", "b"}, Coverage: []int{2, 2}},
		},
		Summary: model.Summary{TotalLines: 2, CoveredLines: 2, Percent: 100.0},
	}

	result := ComputeDiff(base, current)

	wantDelta := 50.0
	if result.DiffSummary.DeltaPercent != wantDelta {
		t.Errorf("DeltaPercent = %f, want %f", result.DiffSummary.DeltaPercent, wantDelta)
	}

	if result.DiffSummary.BasePercent != 50.0 {
		t.Errorf("BasePercent = %f, want 50.0", result.DiffSummary.BasePercent)
	}
}
