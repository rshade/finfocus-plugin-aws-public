# Research: Automated Build Matrix for AWS Regions

**Date**: 2025-11-30
**Feature**: specs/006-region-build-matrix/spec.md

## Template-Based Go Embed File Generation

**Decision**: Use Go's text/template package with a custom generator tool in tools/generate-embeds/

**Rationale**: Maintains existing embed file structure while enabling YAML-driven generation. The approach is simple, testable, and integrates well with the existing build system.

**Alternatives considered**:
- String replacement: Too error-prone for complex templates
- Code generation libraries: Overkill for this use case
- Manual maintenance: Defeats the automation goal

**Implementation approach**:
- Create regions.yaml with region definitions
- Build generator tool using text/template
- Generate embed_*.go files following existing build tag patterns
- Integrate with Makefile and CI/CD

## GoReleaser Configuration Generation

**Decision**: Create a Go-based generator tool using github.com/goreleaser/goreleaser/v2/pkg/config

**Rationale**: Provides type safety, validation, and maintainability while eliminating 200+ lines of repetitive YAML. Follows existing patterns in the codebase.

**Alternatives considered**:
- Shell script generation: Error-prone and hard to test
- YAML templating tools: Less type-safe than Go structs
- Manual maintenance: High duplication risk

**Implementation approach**:
- Use config.Project struct to build configuration programmatically
- Generate separate build blocks for each region
- Maintain existing performance and structure constraints
- Add to CI/CD pipeline for automatic config updates

## YAML Processing in Go

**Decision**: Use goccy/go-yaml for parsing regions.yaml configuration

**Rationale**: Modern, actively maintained library with good error messages and YAML 1.2 support. Provides clean struct unmarshaling for region data.

**Alternatives considered**:
- gopkg.in/yaml.v3: Already in use, but goccy is more actively maintained
- sigs.k8s.io/yaml: Good but adds JSON conversion overhead
- Custom parsing: Unnecessary complexity

**Implementation approach**:
- Define RegionConfig struct matching YAML structure
- Parse regions.yaml into slice of regions
- Use parsed data to drive template generation
- Validate region data during parsing

## Integration Strategy

**Decision**: Extend existing build system with new generation scripts

**Rationale**: Maintains backward compatibility while adding automation. Uses familiar patterns from existing scripts.

**Implementation approach**:
- Add generate-embeds and generate-goreleaser targets to Makefile
- Update CI/CD to run generation before builds
- Preserve sequential building for disk space constraints
- Add verification script to ensure generated files are consistent</content>
<parameter name="filePath">specs/006-region-build-matrix/research.md