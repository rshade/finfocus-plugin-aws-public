# DynamoDB ResourceDescriptor Mapping

The `pbc.ResourceDescriptor` for DynamoDB is mapped as follows:

| Field | Value | Notes |
|-------|-------|-------|
| `Provider` | `"aws"` | Required |
| `ResourceType` | `"dynamodb"` | Required |
| `Sku` | `"on-demand"` \| `"provisioned"` | Defaults to `"on-demand"` if empty |
| `Region` | e.g., `"us-east-1"` | Must match the plugin binary region |

## Tags (Usage Inputs)

### On-Demand Mode
- `read_requests_per_month`: Monthly read request units (numeric string)
- `write_requests_per_month`: Monthly write request units (numeric string)
- `storage_gb`: Table storage in GB (numeric string)

### Provisioned Mode
- `read_capacity_units`: Provisioned RCUs (numeric string)
- `write_capacity_units`: Provisioned WCUs (numeric string)
- `storage_gb`: Table storage in GB (numeric string)
