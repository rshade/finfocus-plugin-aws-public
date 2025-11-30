# Data Model: Pricing Data Schema

**Feature**: `003-ca-sa-region-support`

## Overview
The pricing data is stored as a static JSON file embedded into the binary. The structure is identical for all regions.

## Entities

### PricingRoot
The root object of the generated JSON file.

| Field | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `region` | string | AWS Region identifier | `"ca-central-1"` |
| `currency` | string | ISO 4217 Currency Code | `"USD"` |
| `ec2` | map[string]InstancePricing | Map of instance type to pricing details | `{"t3.micro": ...}` |
| `ebs` | map[string]VolumePricing | Map of volume type to pricing details | `{"gp3": ...}` |

### InstancePricing
Pricing details for an EC2 instance.

| Field | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `instance_type` | string | The API name of the instance | `"t3.micro"` |
| `operating_system` | string | OS platform | `"Linux"` |
| `tenancy` | string | Hosting model | `"Shared"` |
| `hourly_rate` | number | Cost per hour in `currency` | `0.0104` |

### VolumePricing
Pricing details for an EBS volume.

| Field | Type | Description | Example |
| :--- | :--- | :--- | :--- |
| `volume_type` | string | The API name of the volume | `"gp3"` |
| `rate_per_gb_month` | number | Cost per GB-month in `currency` | `0.08` |

## Storage
- **File Path**: `internal/pricing/data/aws_pricing_<region>.json`
- **Embedding**: The file is read into a `[]byte` variable named `rawPricingJSON` using `//go:embed`.