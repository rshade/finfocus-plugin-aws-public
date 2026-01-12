package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/rshade/finfocus-plugin-aws-public/internal/regionsconfig"
)

// main is the program entrypoint. It parses command-line flags for the regions
// config path (`-config`), template path (`-template`) and output directory
// (`-output`); loads the YAML configuration and template file; and generates a
// per-region embed_<region.ID>.go file in the output directory by executing the
// template with each region's data. Errors are written to stderr and cause the
// process to exit with a non-zero status; successful generation prints a
// confirmation per region.
func main() {
	configPath := flag.String("config", "regions.yaml", "Path to regions config")
	templatePath := flag.String("template", "embed_template.go.tmpl", "Path to template")
	outputDir := flag.String("output", "./internal/pricing", "Output directory")
	flag.Parse()

	// Load and validate config using shared regionsconfig package (FR-004, FR-005)
	config, err := regionsconfig.LoadAndValidate(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Load template
	tmpl, err := template.ParseFiles(*templatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading template: %v\n", err)
		os.Exit(1)
	}

	// Generate files
	for _, region := range config.Regions {
		if err := generateEmbedFile(region, tmpl, *outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating file for %s: %v\n", region.Name, err)
			os.Exit(1)
		}
		fmt.Printf("Generated embed file for %s\n", region.Name)
	}
}

// generateEmbedFile creates an embed_<region.ID>.go file in outputDir by executing
// tmpl with the region's ID, Name, and Tag as template data.
//
// The generated file is named "embed_<region.ID>.go" and written to outputDir.
// Parameters:
//   - region: source region values; its ID, Name and Tag are provided to the template.
//   - tmpl: parsed template used to render the file.
//   - outputDir: directory where the generated file will be created.
//
// It returns any error encountered while creating, writing, or closing the file. If closing
// the file fails but a prior error occurred, the prior error is preserved.
func generateEmbedFile(region regionsconfig.RegionConfig, tmpl *template.Template, outputDir string) (err error) {
	filename := fmt.Sprintf("embed_%s.go", region.ID)
	destPath := filepath.Join(outputDir, filename)

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing file: %w", cerr)
		}
	}()

	data := struct {
		ID   string
		Name string
		Tag  string
	}{
		ID:   region.ID,
		Name: region.Name,
		Tag:  region.Tag,
	}

	if err = tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}