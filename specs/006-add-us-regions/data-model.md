# Data Model: Add support for additional US regions

## Entities

### Region
Represents an AWS region with its associated pricing data and configuration.

**Attributes:**
- `Name`: string - Region identifier (e.g., "us-west-1", "us-gov-west-1")
- `DisplayName`: string - Human-readable name (e.g., "N. California", "AWS GovCloud US-West")
- `PricingData`: map[string]PricingData - Service-specific pricing information
- `BuildTag`: string - Go build tag for region-specific compilation (e.g., "region_usw1")

**Relationships:**
- Contains multiple PricingData entries
- Referenced by build configuration

### PricingData
Contains cost information for AWS services within a specific region.

**Attributes:**
- `Service`: string - AWS service name (e.g., "EC2", "S3", "EBS")
- `ResourceType`: string - Specific resource type (e.g., "t3.micro", "gp2")
- `UnitPrice`: float64 - Cost per unit (e.g., per hour, per GB)
- `Currency`: string - Currency code (e.g., "USD")
- `BillingUnit`: string - Unit of measurement (e.g., "Hrs", "GB-Mo")

**Relationships:**
- Belongs to a Region
- Used for cost calculations

### BuildConfiguration
Defines how region-specific binaries are built and deployed.

**Attributes:**
- `Region`: string - Target region name
- `BuildTag`: string - Go build tag
- `BinaryName`: string - Output binary name (e.g., "pulumicost-plugin-aws-public-us-west-1")
- `EmbedFile`: string - Path to embedded pricing data file

**Relationships:**
- References a Region
- Used by GoReleaser configuration

## Validation Rules

- Region names must follow AWS naming conventions
- Pricing data must be non-negative
- Build tags must be unique across regions
- Currency must be supported (currently USD only)

## State Transitions

- Region: Static (no state changes)
- PricingData: Updated via data generation tools (no runtime changes)
- BuildConfiguration: Modified during build process