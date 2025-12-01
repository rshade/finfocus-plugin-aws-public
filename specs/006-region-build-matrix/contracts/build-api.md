# Region Configuration API Contract

**Date**: 2025-11-30
**Feature**: specs/006-region-build-matrix/spec.md

## regions.yaml Schema

### Purpose
Defines the central configuration for all supported AWS regions, driving automated generation of build files and configurations.

### Schema Definition

```yaml
type: object
properties:
  regions:
    type: array
    items:
      type: object
      properties:
        id:
          type: string
          description: Short region identifier (e.g., "use1")
          pattern: "^[a-z0-9]+$"
          minLength: 3
          maxLength: 5
        name:
          type: string
          description: Full AWS region name (e.g., "us-east-1")
          pattern: "^[a-z0-9-]+$"
        tag:
          type: string
          description: Build tag identifier (e.g., "region_use1")
          pattern: "^region_[a-z0-9]+$"
      required:
        - id
        - name
        - tag
required:
  - regions
```

### Validation Rules

1. **ID Uniqueness**: All region IDs must be unique within the file
2. **Tag Consistency**: Tag must follow "region_{id}" pattern
3. **AWS Region Validity**: Name must be a valid AWS region identifier
4. **Mapping Consistency**: ID must match output of `scripts/region-tag.sh`

### Generated Outputs

For each region, the following files are generated:

#### embed_{id}.go
```go
//go:build {tag}

package pricing

import _ "embed"

//go:embed data/aws_pricing_{name}.json
var rawPricingJSON []byte
```

#### .goreleaser.yaml (build block)
```yaml
- id: {name}
  main: ./cmd/pulumicost-plugin-aws-public
  binary: pulumicost-plugin-aws-public-{name}
  env:
    - CGO_ENABLED=0
  goos:
    - linux
    - darwin
    - windows
  goarch:
    - amd64
    - arm64
  tags:
    - {tag}
  ldflags:
    - -s -w -X main.version={{.Version}}
```

### Error Handling

- **Invalid YAML**: Generation fails with parse error
- **Missing required fields**: Generation fails with validation error
- **Duplicate IDs**: Generation fails with uniqueness error
- **Invalid region name**: Generation fails with AWS validation error
- **Tag mismatch**: Generation fails with consistency error

### Versioning

- Schema changes require updating generation tools
- Backward compatibility maintained for existing regions
- New fields can be added as optional</content>
<parameter name="filePath">specs/006-region-build-matrix/contracts/region-api.md