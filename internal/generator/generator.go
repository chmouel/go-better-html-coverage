package generator

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"

	"github.com/chmouel/go-better-html-coverage/internal/model"
)

//go:embed assets/*
var assets embed.FS

type templateData struct {
	CSS      template.CSS
	JS       template.JS
	DataJSON template.JS
	Config   template.JS
}

// Options configures the HTML report generation.
type Options struct {
	NoSyntax bool // Disable syntax highlighting by default
}

// Generate creates an HTML coverage report and writes it to the output path.
func Generate(data *model.CoverageData, outputPath string, opts Options) error {
	// Read assets
	cssBytes, err := assets.ReadFile("assets/style.css")
	if err != nil {
		return fmt.Errorf("reading CSS: %w", err)
	}

	jsBytes, err := assets.ReadFile("assets/app.js")
	if err != nil {
		return fmt.Errorf("reading JS: %w", err)
	}

	htmlBytes, err := assets.ReadFile("assets/template.html")
	if err != nil {
		return fmt.Errorf("reading HTML template: %w", err)
	}

	// Convert coverage data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling coverage data: %w", err)
	}

	// Build config JSON
	config := map[string]interface{}{
		"syntaxEnabled": !opts.NoSyntax,
	}
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("coverage").Parse(string(htmlBytes))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	//nolint:gosec // G203: CSS/JS are from embedded assets, JSON is marshaled from our data
	td := templateData{
		CSS:      template.CSS(cssBytes),
		JS:       template.JS(jsBytes),
		DataJSON: template.JS(dataJSON),
		Config:   template.JS(configJSON),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	if outputPath == "" || outputPath == "-" {
		// stdout
		if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
		return nil
	}

	// Write output file
	if err := os.WriteFile(outputPath, buf.Bytes(), 0o644); err != nil { //nolint:gosec // G306: HTML report should be readable
		return fmt.Errorf("writing output file: %w", err)
	}

	return nil
}
