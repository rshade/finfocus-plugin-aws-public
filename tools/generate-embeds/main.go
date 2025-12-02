package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/goccy/go-yaml"
)

// RegionConfig describes a single AWS region entry in regions.yaml.
type RegionConfig struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
	Tag  string `yaml:"tag"`
}

// Config contains all configured regions.
type Config struct {
	Regions []RegionConfig `yaml:"regions"`
}

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

	// Parse config
	config, err := loadConfig(*configPath)
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

// loadConfig reads the YAML file at filename and unmarshals its contents into a Config.
// It returns the populated Config on success, or an error if the file cannot be read,
// the YAML cannot be parsed, or no regions are defined.
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config.Regions) == 0 {
		return nil, fmt.Errorf("no regions defined in config file")
	}

	return &config, nil
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
func generateEmbedFile(region RegionConfig, tmpl *template.Template, outputDir string) (err error) {
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