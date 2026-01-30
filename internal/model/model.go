package model

// FileData represents a single source file with coverage information.
type FileData struct {
	ID        int      `json:"id"`
	Path      string   `json:"path"`                // module-relative path
	Lines     []string `json:"lines"`               // source lines
	Coverage  []int    `json:"coverage"`            // 0=no stmt, 1=uncovered, 2=covered
	DiffState []int    `json:"diffState,omitempty"` // diff mode only: 0=no change, 1=newly covered, 2=newly uncovered, 3=unchanged covered, 4=unchanged uncovered
}

// TreeNode represents a node in the file tree (directory or file).
type TreeNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // "dir" or "file"
	FileID   *int        `json:"fileId,omitempty"`
	Children []*TreeNode `json:"children,omitempty"`
}

// Summary contains overall coverage statistics.
type Summary struct {
	TotalLines   int     `json:"totalLines"`
	CoveredLines int     `json:"coveredLines"`
	Percent      float64 `json:"percent"`
}

// DiffSummary contains statistics about coverage changes between base and current.
type DiffSummary struct {
	NewlyCoveredLines   int     `json:"newlyCoveredLines"`
	NewlyUncoveredLines int     `json:"newlyUncoveredLines"`
	DeltaPercent        float64 `json:"deltaPercent"`
	BasePercent         float64 `json:"basePercent"`
}

// CoverageData is the complete data structure passed to the HTML template.
type CoverageData struct {
	Files       []FileData   `json:"files"`
	Tree        *TreeNode    `json:"tree"`
	Summary     Summary      `json:"summary"`
	DiffSummary *DiffSummary `json:"diffSummary,omitempty"`
	IsDiffMode  bool         `json:"isDiffMode"`
}
