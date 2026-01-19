# Research: IAM Zero-Cost Resource Handling

**Status**: Complete
**Date**: 2026-01-19

## Decisions

### 1. Resource Normalization Strategy
- **Decision**: Use case-insensitive prefix matching for `aws:iam/*`.
- **Rationale**:
    - **Future-proof**: Automatically captures all IAM resource types (User, Role, Policy, etc.) without maintaining an exhaustive list.
    - **Robustness**: Case-insensitive matching (`aws:iam/user:User` vs `AWS:IAM/USER:USER`) handles potential variations in Pulumi SDK outputs.
    - **Safety**: AWS confirms the entire IAM feature set is free, so there is no risk of accidentally zero-costing a paid resource (unlike services where only *some* components are free).
- **Alternatives Considered**:
    - *Explicit Allowlist*: Rejected due to maintenance burden and risk of missing valid IAM resources causing "unsupported" warnings.
