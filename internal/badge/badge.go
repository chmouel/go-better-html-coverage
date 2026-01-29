package badge

import (
	"fmt"
	"os"
)

// Thresholds defines the color thresholds for badge generation.
type Thresholds struct {
	Red    float64 // Upper threshold for red (0-Red is red)
	Yellow float64 // Upper threshold for yellow (Red-Yellow is yellow, Yellow+ is green)
}

// DefaultThresholds returns the default color thresholds.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Red:    40,
		Yellow: 70,
	}
}

// GenerateBadge creates an SVG badge showing coverage percentage and writes it to the specified path.
// If outputPath is "-", the badge is written to stdout.
func GenerateBadge(coverage float64, outputPath string, thresholds Thresholds) error {
	svg := generateSVG(coverage, thresholds)

	if outputPath == "-" {
		// Write to stdout
		if _, err := os.Stdout.WriteString(svg); err != nil {
			return fmt.Errorf("writing badge to stdout: %w", err)
		}
		return nil
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(svg), 0o644); err != nil { //nolint:gosec // G306: Badge should be readable
		return fmt.Errorf("writing badge file: %w", err)
	}

	return nil
}

// generateSVG creates the SVG content for the badge.
func generateSVG(coverage float64, thresholds Thresholds) string {
	// Clamp coverage to 0-100 range
	if coverage < 0 {
		coverage = 0
	}
	if coverage > 100 {
		coverage = 100
	}

	// Determine color based on coverage percentage
	color := getColor(coverage, thresholds)

	// Format coverage percentage
	label := fmt.Sprintf("%.1f%%", coverage)

	// SVG dimensions and positions
	leftWidth := 63
	rightWidth := 48
	height := 20
	totalWidth := leftWidth + rightWidth

	// SVG template - shields.io compatible design
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" role="img" aria-label="coverage: %s">
  <title>coverage: %s</title>
  <g shape-rendering="crispEdges">
    <rect width="%d" height="%d" fill="#555"/>
    <rect x="%d" width="%d" height="%d" fill="%s"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="11">
    <text aria-hidden="true" x="%d" y="15" fill="#010101" fill-opacity=".3">coverage</text>
    <text x="%d" y="14">coverage</text>
    <text aria-hidden="true" x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`,
		totalWidth,
		height,
		label,
		label,
		totalWidth,
		height,
		leftWidth,
		rightWidth,
		height,
		color,
		leftWidth/2,
		leftWidth/2,
		leftWidth+rightWidth/2,
		label,
		leftWidth+rightWidth/2,
		label,
	)

	return svg
}

// getColor returns the SVG color code based on coverage percentage and thresholds.
func getColor(coverage float64, thresholds Thresholds) string {
	switch {
	case coverage >= thresholds.Yellow:
		return "#4c1" // Green
	case coverage > thresholds.Red:
		return "#dfb317" // Yellow/Amber
	default:
		return "#e05d44" // Red
	}
}
