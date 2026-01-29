package badge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBadge_FileOutput(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "badge.svg")

	err := GenerateBadge(85.5, tmpFile, DefaultThresholds())
	if err != nil {
		t.Fatalf("GenerateBadge failed: %v", err)
	}

	//nolint:gosec // G204: tmpFile is from t.TempDir
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read badge file: %v", err)
	}

	svg := string(content)
	if !strings.Contains(svg, "<svg") {
		t.Error("Badge doesn't contain SVG element")
	}
	if !strings.Contains(svg, "85.5%") {
		t.Error("Badge doesn't contain the coverage percentage")
	}
}

func TestGenerateBadge_Colors(t *testing.T) {
	thresholds := DefaultThresholds()
	tests := []struct {
		coverage float64
		color    string
		name     string
	}{
		{25.0, "#e05d44", "red for low coverage"},
		{55.0, "#dfb317", "yellow for medium coverage"},
		{85.0, "#4c1", "green for high coverage"},
		{0.0, "#e05d44", "red for zero coverage"},
		{100.0, "#4c1", "green for full coverage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg := generateSVG(tt.coverage, thresholds)
			if !strings.Contains(svg, tt.color) {
				t.Errorf("Expected color %s for coverage %.1f%%, got SVG: %s", tt.color, tt.coverage, svg)
			}
		})
	}
}

func TestGenerateBadge_ValidSVG(t *testing.T) {
	svg := generateSVG(72.5, DefaultThresholds())

	if !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("SVG doesn't have proper SVG namespace")
	}

	if !strings.Contains(svg, "<svg") {
		t.Error("SVG doesn't contain svg element")
	}

	if !strings.Contains(svg, "</svg>") {
		t.Error("SVG doesn't have closing svg tag")
	}

	if !strings.Contains(svg, "72.5%") {
		t.Error("SVG doesn't contain the coverage percentage")
	}
}

func TestGenerateBadge_EdgeCases(t *testing.T) {
	tests := []struct {
		coverage float64
		expected string
	}{
		{-10.0, "0.0%"},   // Negative clamped to 0
		{150.0, "100.0%"}, // Over 100 clamped to 100
		{50.0, "50.0%"},   // Normal value
		{33.333, "33.3%"}, // Decimal precision
	}

	for _, tt := range tests {
		t.Run("coverage "+tt.expected, func(t *testing.T) {
			svg := generateSVG(tt.coverage, DefaultThresholds())
			if !strings.Contains(svg, tt.expected) {
				t.Errorf("Expected %s in SVG for coverage %.3f", tt.expected, tt.coverage)
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	thresholds := DefaultThresholds()
	tests := []struct {
		coverage float64
		expected string
	}{
		{0, "#e05d44"},
		{40, "#e05d44"},
		{40.1, "#dfb317"},
		{69.9, "#dfb317"},
		{70, "#4c1"},
		{100, "#4c1"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			color := getColor(tt.coverage, thresholds)
			if color != tt.expected {
				t.Errorf("getColor(%.1f) = %s, want %s", tt.coverage, color, tt.expected)
			}
		})
	}
}

func TestGetColor_CustomThresholds(t *testing.T) {
	thresholds := Thresholds{
		Red:    50,
		Yellow: 80,
	}
	tests := []struct {
		coverage float64
		expected string
	}{
		{25, "#e05d44"},   // Below red threshold
		{50, "#e05d44"},   // At red threshold
		{65, "#dfb317"},   // Between red and yellow
		{79.9, "#dfb317"}, // Below yellow threshold
		{80, "#4c1"},      // At yellow threshold
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			color := getColor(tt.coverage, thresholds)
			if color != tt.expected {
				t.Errorf("getColor(%.1f) with custom thresholds = %s, want %s", tt.coverage, color, tt.expected)
			}
		})
	}
}
