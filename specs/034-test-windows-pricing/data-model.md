# Data Model: EC2 Pricing Differentiation

## Entities

### ResourceDescriptor (Input)
Represents the AWS resource to be estimated.
- `Provider`: Must be "aws"
- `ResourceType`: Must be "ec2"
- `Sku`: Instance type (e.g., "t3.medium")
- `Region`: Must be "us-east-1"
- `Tags`:
    - `platform`: "linux", "windows", "rhel", "suse"
    - `tenancy`: "shared", "dedicated"
    - `arch` / `architecture`: "x86_64", "arm64"

### GetProjectedCostResponse (Output)
- `CostPerMonth`: The calculated monthly cost (hourly * 730)
- `UnitPrice`: The hourly on-demand rate
- `BillingDetail`: Human-readable description including platform and tenancy

## Validations & Mappings

| Tag | Input Value | Normalized Value (AWS Pricing) |
|-----|-------------|------------------------------|
| platform | windows | Windows |
| platform | (other) | Linux |
| tenancy | dedicated | Dedicated |
| tenancy | (other) | Shared |
