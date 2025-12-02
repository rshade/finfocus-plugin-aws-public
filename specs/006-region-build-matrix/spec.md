# Feature Specification: Automated Build Matrix for AWS Regions

**Feature Branch**: `006-region-build-matrix`  
**Created**: 2025-11-30  
**Status**: Draft  
**Input**: User description: "title:	Create automated build matrix for all AWS regions
state:	OPEN
author:	rshade
labels:	enhancement
comments:	0
assignees:	
projects:	
milestone:	
number:	7
--
### Description
Instead of manually adding each region, create an automated system that generates build configurations, embed files, and tests from a central region list.

### Implementation Tasks
- [ ] Create `regions.yaml` configuration file listing all supported regions
- [ ] Generate embed files from template
- [ ] Generate `.goreleaser.yaml` from template
- [ ] Auto-generate build tags
- [ ] Create verification script to ensure all regions are configured
- [ ] Update CI/CD to use region matrix
- [ ] Document the region addition process

### Benefits
- Adding new regions becomes a 1-line change in `regions.yaml` plus running generation scripts
- Reduces manual errors in maintaining region configurations
- Ensures consistency across all regions
- Makes maintenance easier by centralizing region definitions

### Acceptance Criteria
- Region configuration is driven by single YAML file
- All embed files are generated automatically
- GoReleaser config is generated
- Adding a new region requires only editing `regions.yaml`
- All existing regions continue to work

### Example `regions.yaml`
```yaml
regions:
  - id: use1
    name: us-east-1
    tag: region_use1
  - id: usw2
    name: us-west-2
    tag: region_usw2
  - id: euw1
    name: eu-west-1
    tag: region_euw1
  # ... more regions
```"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add New AWS Region Support (Priority: P1)

As a developer maintaining the plugin, I want to add support for a new AWS region by editing a single configuration file, so that the build system automatically generates all necessary files and configurations.

**Why this priority**: This is the core value proposition - simplifying region additions from manual processes to automated ones.

**Independent Test**: Can be fully tested by adding a new region to regions.yaml and verifying that all build artifacts are generated correctly.

**Acceptance Scenarios**:

1. **Given** a new region is added to regions.yaml, **When** the build system runs, **Then** embed files, goreleaser config, and build tags are generated automatically.
2. **Given** an existing region configuration, **When** the verification script runs, **Then** all regions are confirmed as properly configured.

---

### User Story 2 - Verify Region Configuration Consistency (Priority: P2)

As a CI/CD maintainer, I want to run a verification script that ensures all regions are properly configured, so that I can catch configuration errors before deployment.

**Why this priority**: Ensures reliability and prevents deployment issues from misconfigurations.

**Independent Test**: Can be fully tested by running the verification script and checking that it reports all regions as configured.

**Acceptance Scenarios**:

1. **Given** all regions are properly configured, **When** verification script runs, **Then** it reports success for all regions.
2. **Given** a region is missing configuration, **When** verification script runs, **Then** it reports the missing configuration.

---

### User Story 3 - Update CI/CD Pipeline (Priority: P3)

As a DevOps engineer, I want the CI/CD pipeline to use the region matrix automatically, so that builds are generated for all supported regions without manual intervention.

**Why this priority**: Completes the automation by integrating with deployment processes.

**Independent Test**: Can be fully tested by triggering the CI/CD pipeline and verifying builds are created for all regions in the matrix.

**Acceptance Scenarios**:

1. **Given** regions.yaml is updated, **When** CI/CD pipeline runs, **Then** it builds binaries for all regions listed.

---

### Edge Cases

- System MUST fail generation with error when regions.yaml contains invalid region names.
- How does system handle regions that require special pricing data?
- What if a region is removed from AWS services?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create a regions.yaml configuration file listing all supported AWS regions with their identifiers, names, and build tags (following existing region-tag.sh mapping).
- **FR-002**: System MUST generate embed_*.go files automatically from templates for each region, maintaining the current build tag structure.
- **FR-003**: System MUST generate .goreleaser.yaml configuration from templates, preserving the existing build structure with separate build blocks per region.
- **FR-004**: System MUST maintain the existing sequential region building approach (build-region.sh) to respect disk space constraints on build images.
- **FR-005**: System MUST provide a verification script that ensures all regions are properly configured and embed files exist.
- **FR-006**: System MUST update CI/CD pipeline to use the region matrix automatically while maintaining sequential builds.
- **FR-007**: System MUST document the process for adding new regions, emphasizing the single-line regions.yaml change.

### Key Entities *(include if feature involves data)*

- **Region**: Represents an AWS region with id (short code like 'use1'), name (full AWS name like 'us-east-1'), and tag (build tag like 'region_use1').

### Out of Scope

- Manual region additions
- Custom pricing overrides

### Assumptions

- Build images have limited disk space, requiring sequential region builds and cache cleanup between builds.
- Existing region-tag.sh mapping must be preserved for backward compatibility.
- Each region needs separate embed_*.go file with build tags for Go's conditional compilation.

## Clarifications

### Session 2025-11-30

- Q: How should invalid region names in regions.yaml be handled? → A: Fail generation with error
- Q: What are the explicit out-of-scope items for this feature? → A: Manual region additions and custom pricing overrides

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Adding a new region requires editing only regions.yaml and running generation scripts, taking less than 5 minutes.
- **SC-002**: All build configurations for existing regions remain functional after automation implementation.
- **SC-003**: Verification script runs in under 30 seconds and accurately identifies missing embed files or configuration issues.
- **SC-004**: CI/CD pipeline successfully builds for all regions in the matrix using sequential builds to respect disk constraints.