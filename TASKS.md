# Task Queue

The ordered work queue for [`/pick-issue`](.claude/commands/pick-issue.md).
Each invocation takes the first `TODO` row whose dependencies are merged,
delivers it as one pull request, and stops.

Statuses:

- `TODO`: ready once its dependencies are merged
- `DECIDE`: needs an owner decision before anyone implements it

GitHub is the source of truth for progress. A closed issue counts as done,
and an open PR that says `Closes #N` means the issue is in review. This file
is updated by the owner, not by feature PRs.

Snapshot: 2026-10-04, `main` at finfocus-spec v0.7.3.

## Queue

| # | Issue | Title | Effort | Depends on | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | #404 | Drop hand-built legacy capability metadata | S | - | TODO | Unblocked: finfocus-spec#510 is in v0.7.3. Remove the explicit capability list workaround and its ROADMAP line |
| 2 | #351 | Re-enable strict checksum verification for region downloads | S | - | TODO | v0.1.9 `checksums.txt` is clean |
| 3 | #396 | Make region-binary downloads visible; plan offline-by-default | M | #351 | TODO | Keep within the CONTEXT.md router exception |
| 4 | #287 | Consolidate region mappings into `regions.yaml` | M | - | TODO | Shell scripts read `regions.yaml` via `tools/parse-regions` |
| 5 | #271 | Add us-gov-west-1 | M | #287 | TODO | First confirm the public Price List publishes GovCloud offers. If it does not, report that and stop |
| 6 | #272 | Add us-gov-east-1 | M | #271 | TODO | Same files as #271; do it after #271 merges |
| 7 | #388 | Accept Terraform-state attribute names and optional SKUs | L | - | TODO | Defines the shared input key map that #415 builds on |
| 8 | #415 | Read `ResourceDescriptor.attributes` with tag fallback | L | #388 | TODO | Split into several PRs: EC2/ASG, then validation, then one service each |
| 9 | #409 | Price EKS Fargate pods by vCPU and GiB | M | - | TODO | Static on-demand Fargate rates only |
| 10 | #393 | ECS Fargate cost estimator | L | #409 | TODO | Reuse the Fargate pricing from #409 |
| 11 | #394 | Amazon EFS cost estimator | M | - | TODO | New service: follow "Adding New AWS Services" in `CLAUDE.md` |
| 12 | #390 | Expose Price List `publicationDate` as a freshness stamp | M | - | TODO | |
| 13 | #391 | Pricing drift report between releases | L | #390 | TODO | Build-time tool only; no runtime network calls |
| 14 | #395 | Static ElastiCache, Lambda arm64, and NAT endpoint recommendations | M | - | TODO | |
| 15 | #392 | Opt-in cross-region cost and carbon comparison | L | - | TODO | Router feature; must stay opt-in |
| 16 | #212 | Validation warnings for missing CloudWatch pricing | S | - | TODO | |
| 17 | #144 | Tighten carbon estimation test thresholds | S | - | TODO | |
| 18 | #184 | Race detector test for parallel parsing | S | - | TODO | |
| 19 | #185 | Integration test for the binary size limit | S | - | TODO | |
| 20 | #227 | Unit tests for CSV parsing graceful degradation | S | - | TODO | |
| 21 | #259 | Remove hardcoded port duplication in Docker scripts | S | - | TODO | |
| 22 | #260 | Configurable metrics aggregation failure threshold | S | - | TODO | |
| 23 | #264 | Replace `//nolint:unused` with build-tag placeholders | S | - | TODO | |
| 24 | #265 | Return `traceLogger` by value to reduce allocations | S | - | TODO | Benchmark before and after |
| 25 | #266 | Replace sleep with a timeout in `web_server_test.go` | S | - | TODO | |
| 26 | #267 | Log ignored cleanup errors in `web_server_test.go` | S | #266 | TODO | Same file as #266 |
| 27 | #181 | Pricing metadata consistency validation | M | - | TODO | |
| 28 | #182 | CI memory profiling for parsing | M | - | TODO | |
| 29 | #174 | `grpcurl` integration tests for pricing | M | - | TODO | |
| 30 | #216 | E2E integration tests with Pulumi YAML fixtures | L | - | TODO | |

## Needs an Owner Decision

| Issue | Title | Effort | Decision needed |
| --- | --- | --- | --- |
| #129 | Integration test binary naming convention | S | Labeled `question`: pick a convention first |
| #83 | Refactor pricing client initialization | M | Scope is open-ended; define the target shape |
| #84 | Reduce memory used by pricing data parsing | L | Choose an approach (lazy loading or memory-mapped) and a memory target |
| #258 | Dual-layer capability discovery | L | Confirm whether it needs a finfocus-spec change first |

## Excluded

- #13: the Renovate Dependency Dashboard. It stays open by design (see
  `CONTEXT.md`).
- Spot pricing (was #406): moved to rshade/finfocus-plugin-aws-ce#56. This
  plugin quotes static On-Demand prices only (`CONTEXT.md` rule 5).
