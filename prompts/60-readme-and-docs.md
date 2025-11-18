# Prompt: Update README and basic docs for `pulumicost-plugin-aws-public`

You are implementing the `pulumicost-plugin-aws-public` Go plugin.
Repo: `pulumicost-plugin-aws-public`.

## Goal

Update or create a `README.md` (and optionally `RELEASING.md`) that explains:

- What this plugin does
- The gRPC protocol it implements
- How it's built and distributed (region-specific binaries, GoReleaser)
- How PulumiCost core calls it via gRPC

## 1. README structure

Ensure `README.md` contains at least:

### Title & summary

- Title: `pulumicost-plugin-aws-public`
- Short description:

  > A fallback PulumiCost plugin that estimates AWS resource costs from public on-demand pricing when no billing data is available. Implements the CostSourceService gRPC interface.

### Features

- Estimates monthly/hourly cost for:
  - EC2 instances (on-demand, Linux/Shared)
  - EBS volumes
- Uses **public AWS pricing** embedded into the binary (no AWS credentials required at runtime)
- Region-specific binaries:
  - `pulumicost-plugin-aws-public-us-east-1`
  - `pulumicost-plugin-aws-public-us-west-2`
  - `pulumicost-plugin-aws-public-eu-west-1`
- gRPC service implementing `pulumicost.v1.CostSourceService`
- Thread-safe for concurrent gRPC calls
- Safe to use in CI as a "best effort" cost signal

### How it works

Explain at a high level:

- At release time:
  - `tools/generate-pricing` fetches AWS public pricing
  - It trims the data down to a small JSON per region (`data/aws_pricing_<region>.json`)
  - GoReleaser builds one binary per region with that JSON embedded via `//go:embed`
- At runtime:
  - PulumiCost core starts the appropriate region binary as a subprocess
  - Plugin announces `PORT=<port>` to stdout
  - Core connects to the plugin via gRPC on `127.0.0.1:<port>`
  - Core calls `GetProjectedCost()`, `Supports()`, `Name()` RPCs for each resource
  - Plugin returns cost estimates using the embedded pricing data

### gRPC Protocol

Document the key RPCs:

#### Name()

Returns the plugin identifier:

```json
{
  "name": "aws-public"
}
```

#### Supports(ResourceDescriptor)

Checks if the plugin can estimate a resource:

```json
// Request
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "region": "us-east-1"
  }
}

// Response
{
  "supported": true,
  "reason": "Fully supported"
}
```

#### GetProjectedCost(ResourceDescriptor)

Returns cost estimate for a single resource:

```json
// Request
{
  "resource": {
    "provider": "aws",
    "resource_type": "ec2",
    "sku": "t3.micro",
    "region": "us-east-1"
  }
}

// Response
{
  "unit_price": 0.0104,
  "currency": "USD",
  "cost_per_month": 7.592,
  "billing_detail": "On-demand Linux/Shared instance, t3.micro, 730 hrs/month"
}
```

#### Error Handling

For region mismatches, returns gRPC error with `ERROR_CODE_UNSUPPORTED_REGION`:

```
code: FailedPrecondition
message: "Resource in region us-west-2 but plugin compiled for us-east-1"
```

Core can detect this and fetch/run the correct region binary.

### ResourceDescriptor Format

The proto input for each resource:

```protobuf
message ResourceDescriptor {
  string provider = 1;       // "aws"
  string resource_type = 2;  // "ec2", "ebs", "s3", etc.
  string sku = 3;            // instance type (e.g., "t3.micro") or volume type (e.g., "gp3")
  string region = 4;         // "us-east-1", "us-west-2", etc.
  map<string, string> tags = 5;  // For EBS, may contain "size" or "volume_size"
}
```

### Usage (standalone)

Provide a small example using grpcurl:

```bash
# Start the plugin
./pulumicost-plugin-aws-public-us-east-1
# Output: PORT=12345

# In another terminal, call GetProjectedCost via grpcurl
grpcurl -plaintext \
  -d '{
    "resource": {
      "provider": "aws",
      "resource_type": "ec2",
      "sku": "t3.micro",
      "region": "us-east-1"
    }
  }' \
  localhost:12345 \
  pulumicost.v1.CostSourceService/GetProjectedCost
```

### Building & releasing

Summarize GoReleaser usage:

```bash
# Test local build with dummy pricing data
goreleaser build --snapshot --clean

# Binaries are in dist/ directory, organized by region and OS/arch

# For release (requires tag)
git tag v0.1.0
goreleaser release --clean
```

Mention that releases produce tarballs per region, OS, and arch.

### Integration with PulumiCost Core

Explain the flow:

1. User runs PulumiCost CLI with their Pulumi stack
2. Core analyzes stack resources and groups by AWS region
3. For each region, core:
   - Downloads or locates the appropriate `pulumicost-plugin-aws-public-<region>` binary
   - Starts it as a subprocess
   - Reads `PORT=<port>` from stdout
   - Connects gRPC client to `127.0.0.1:<port>`
   - Calls `Supports()` for each resource to check compatibility
   - Calls `GetProjectedCost()` for supported resources
   - Aggregates costs across all regions
4. Core displays total estimated cost to user

### Protocol Dependencies

This plugin depends on proto definitions from:

- `github.com/rshade/pulumicost-spec/sdk/go/proto/pulumicost/v1`
  - `CostSourceService` gRPC service
  - `ResourceDescriptor`, `GetProjectedCostRequest/Response`
  - `SupportsRequest/Response`, `PricingSpec`
  - `ErrorCode` enum

Always refer to `pulumicost-spec/proto/` for the authoritative API contract.

## 2. RELEASING.md (optional but nice)

Create `RELEASING.md` with a concise checklist:

```markdown
# Releasing pulumicost-plugin-aws-public

## Pre-release checklist

- [ ] Ensure `tools/generate-pricing` is up to date and working
- [ ] Run tests: `go test ./...`
- [ ] Run local snapshot build: `goreleaser build --snapshot --clean`
- [ ] Test generated binaries manually:
  ```bash
  ./dist/pulumicost-plugin-aws-public-us-east-1_linux_amd64_v1/pulumicost-plugin-aws-public-us-east-1
  # Should output: PORT=<port>
  ```
- [ ] Test via grpcurl if possible

## Release process

1. Update version in relevant files if needed
2. Commit all changes
3. Tag the release:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
4. Run GoReleaser:
   ```bash
   goreleaser release --clean
   ```
5. Verify GitHub Release created with artifacts
6. Test download and execution of released binaries

## Post-release

- Update documentation if protocol changed
- Announce release in PulumiCost discussions/channels
```

## 3. Optional: CLAUDE.md updates

If `CLAUDE.md` exists, ensure it documents:

- gRPC protocol (not stdin/stdout)
- PORT announcement mechanism
- Thread safety requirements
- proto dependencies

## Acceptance criteria

- `README.md` clearly explains:
  - What the plugin does
  - gRPC protocol with example RPCs
  - Region-specific binary naming
  - High-level flow with PulumiCost core via gRPC
  - ResourceDescriptor format
  - Error handling (ERROR_CODE_UNSUPPORTED_REGION)
- Optional `RELEASING.md` exists with release checklist
- Documentation matches actual gRPC implementation (not stdin/stdout)

Update docs now.
