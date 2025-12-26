# Quickstart: Resource ID Passthrough

**Feature**: 001-resourceid-passthrough
**Time Estimate**: 30 minutes

## Prerequisites

- Go 1.25+ installed
- Repository cloned and on `001-resourceid-passthrough` branch
- `make develop` completed (carbon data generated)

## Step 1: Update Dependency (5 min)

```bash
# Update pulumicost-spec to v0.4.11
go get github.com/rshade/pulumicost-spec@v0.4.11
go mod tidy
```

Verify the update:

```bash
grep pulumicost-spec go.mod
# Should show: github.com/rshade/pulumicost-spec v0.4.11
```

## Step 2: Implement ID Passthrough (10 min)

Edit `internal/plugin/recommendations.go` around line 112-123.

**Before**:

```go
// Populate correlation info (T008): Use tags for correlation
for _, rec := range recs {
    if rec.Resource != nil {
        // Use resource_id tag if available for correlation
        if resourceID := resource.Tags["resource_id"]; resourceID != "" {
            rec.Resource.Id = resourceID
        }
        // ... rest of logic
    }
}
```

**After**:

```go
// Populate correlation info: Native Id takes priority over tag
for _, rec := range recs {
    if rec.Resource != nil {
        // Priority 1: Use native Id field (new in pulumicost-spec v0.4.11)
        if id := strings.TrimSpace(resource.Id); id != "" {
            rec.Resource.Id = id
        } else if resourceID := resource.Tags["resource_id"]; resourceID != "" {
            // Priority 2: Fall back to resource_id tag for backward compatibility
            rec.Resource.Id = resourceID
        }
        // ... rest of logic unchanged
    }
}
```

Add import if not present:

```go
import "strings"
```

## Step 3: Add Unit Tests (10 min)

Add test cases to `internal/plugin/recommendations_test.go`:

```go
func TestGetRecommendations_IDPassthrough(t *testing.T) {
    tests := []struct {
        name       string
        nativeID   string
        tagID      string
        expectedID string
    }{
        {
            name:       "native ID takes priority",
            nativeID:   "native-id",
            tagID:      "tag-id",
            expectedID: "native-id",
        },
        {
            name:       "falls back to tag when native empty",
            nativeID:   "",
            tagID:      "tag-id",
            expectedID: "tag-id",
        },
        {
            name:       "whitespace native ID treated as empty",
            nativeID:   "   ",
            tagID:      "tag-id",
            expectedID: "tag-id",
        },
        {
            name:       "empty when neither present",
            nativeID:   "",
            tagID:      "",
            expectedID: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation here
        })
    }
}
```

## Step 4: Verify (5 min)

```bash
# Run linting
make lint

# Run tests
make test

# Build with region tag to verify compilation
go build -tags region_use1 ./cmd/pulumicost-plugin-aws-public
```

## Success Criteria

- [ ] `go.mod` shows pulumicost-spec v0.4.11+
- [ ] `make lint` passes
- [ ] `make test` passes (including new ID passthrough tests)
- [ ] Build succeeds with region tag

## Next Steps

After implementation:

1. Run `/speckit.tasks` to generate task breakdown
2. Create PR with conventional commit message
3. Verify CI passes
