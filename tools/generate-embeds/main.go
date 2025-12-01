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

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func generateEmbedFile(region RegionConfig, tmpl *template.Template, outputDir string) error {
	filename := fmt.Sprintf("embed_%s.go", region.ID)
	destPath := filepath.Join(outputDir, filename)

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
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
		return err
	}

	return err
}
