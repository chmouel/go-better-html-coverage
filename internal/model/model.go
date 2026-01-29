package model

// FileData represents a single source file with coverage information.
type FileData struct {
	ID       int      `json:"id"`
	Path     string   `json:"path"`     // module-relative path
	Lines    []string `json:"lines"`    // source lines
	Coverage []int    `json:"coverage"` // 0=no stmt, 1=uncovered, 2=covered
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

// CoverageData is the complete data structure passed to the HTML template.
type CoverageData struct {
	Files   []FileData `json:"files"`
	Tree    *TreeNode  `json:"tree"`
	Summary Summary    `json:"summary"`
}
