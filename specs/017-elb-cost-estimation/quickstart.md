# Quickstart: ELB Cost Estimation

## Usage

### 1. Estimate Application Load Balancer (ALB)
Provide the SKU as `alb` and optionally the LCU usage.

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elb",
    "sku": "alb",
    "region": "us-east-1",
    "tags": {
      "lcu_per_hour": "5"
    }
  }
}
```

### 2. Estimate Network Load Balancer (NLB)
Provide the SKU as `nlb` and optionally the NLCU usage.

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elb",
    "sku": "nlb",
    "region": "us-east-1",
    "tags": {
      "nlcu_per_hour": "2"
    }
  }
}
```

### 3. Generic Capacity Tag
You can also use `capacity_units` which will apply to the active LB type.

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "elb",
    "sku": "alb",
    "region": "us-east-1",
    "tags": {
      "capacity_units": "10"
    }
  }
}
```

## Testing
Run integration tests for ELB:

```bash
go test ./internal/plugin -v -run TestAWSPublicPlugin_GetProjectedCost/elb
```