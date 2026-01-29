package model

import "testing"

func TestFileData(t *testing.T) {
	fd := FileData{
		ID:       1,
		Path:     "test.go",
		Lines:    []string{"package main", "", "func main() {}"},
		Coverage: []int{0, 0, 2},
	}
	if fd.ID != 1 {
		t.Errorf("expected ID=1, got %d", fd.ID)
	}
	if fd.Path != "test.go" {
		t.Errorf("expected Path=test.go, got %s", fd.Path)
	}
}

func TestSummary(t *testing.T) {
	s := Summary{
		TotalLines:   100,
		CoveredLines: 80,
		Percent:      80.0,
	}
	if s.Percent != 80.0 {
		t.Errorf("expected Percent=80.0, got %f", s.Percent)
	}
}
