# Test Data Examples

This directory contains sample ResourceDescriptor JSON files for manual testing of the plugin via grpcurl.

## Usage

### 1. Start the Plugin

```bash
# Build and start the plugin for us-east-1
go run ./cmd/finfocus-plugin-aws-public

# Or use a pre-built binary
./finfocus-plugin-aws-public-us-east-1
```

**Output**: `PORT=<port>` (capture this port number)

### 2. Test with grpcurl

Install grpcurl if needed:

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### GetProjectedCost Examples

**EC2 Instance (t3.micro)**:

```bash
grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost < testdata/ec2-t3-micro-us-east-1.json
```

**Expected Response**:

```json
{
  "unitPrice": 0.0104,
  "currency": "USD",
  "costPerMonth": 7.592,
  "billingDetail": "On-demand Linux, shared tenancy, 730 hrs/month"
}
```

**EBS Volume (gp3, 100GB)**:

```bash
grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost < testdata/ebs-gp3-100gb-us-east-1.json
```

**Expected Response**:

```json
{
  "unitPrice": 0.08,
  "currency": "USD",
  "costPerMonth": 8.0,
  "billingDetail": "EBS gp3 storage, 100GB"
}
```

**EBS Volume (default size)**:

```bash
grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost < testdata/ebs-gp2-default-size-us-east-1.json
```

**Expected Response**:

```json
{
  "unitPrice": 0.1,
  "currency": "USD",
  "costPerMonth": 0.8,
  "billingDetail": "EBS gp2 storage, 8GB, defaulted to 8GB"
}
```

**S3 Bucket (stub service)**:

```bash
grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost < testdata/s3-bucket-us-east-1.json
```

**Expected Response**:

```json
{
  "unitPrice": 0,
  "currency": "USD",
  "costPerMonth": 0,
  "billingDetail": "S3 cost estimation not implemented - returning $0"
}
```

**Region Mismatch (us-west-2 resource with us-east-1 plugin)**:

```bash
grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost < testdata/ec2-m5-large-us-west-2.json
```

**Expected Error**:

```
ERROR:
  Code: FailedPrecondition
  Message: Resource region "us-west-2" does not match plugin region "us-east-1"
```

### Supports Examples

**Check EC2 Support**:

```bash
grpcurl -plaintext -d '{"resource": {"provider": "aws", "resource_type": "ec2", "region": "us-east-1"}}' \
  localhost:<port> finfocus.v1.CostSourceService/Supports
```

**Expected Response**:

```json
{
  "supported": true,
  "reason": ""
}
```

**Check S3 Support (stub)**:

```bash
grpcurl -plaintext -d '{"resource": {"provider": "aws", "resource_type": "s3", "region": "us-east-1"}}' \
  localhost:<port> finfocus.v1.CostSourceService/Supports
```

**Expected Response**:

```json
{
  "supported": true,
  "reason": "Limited support - returns $0 estimate"
}
```

**Check Wrong Region**:

```bash
grpcurl -plaintext -d '{"resource": {"provider": "aws", "resource_type": "ec2", "region": "us-west-2"}}' \
  localhost:<port> finfocus.v1.CostSourceService/Supports
```

**Expected Response**:

```json
{
  "supported": false,
  "reason": "Region not supported by this binary"
}
```

### Name RPC

```bash
grpcurl -plaintext localhost:<port> finfocus.v1.CostSourceService/Name
```

**Expected Response**:

```json
{
  "name": "aws-public"
}
```

## Test Data Files

| File | Description | Expected Cost |
|------|-------------|---------------|
| `ec2-t3-micro-us-east-1.json` | t3.micro EC2 instance | $7.592/month |
| `ebs-gp3-100gb-us-east-1.json` | 100GB gp3 EBS volume | $8.00/month |
| `ebs-gp2-default-size-us-east-1.json` | gp2 volume without size (defaults to 8GB) | $0.80/month |
| `s3-bucket-us-east-1.json` | S3 bucket (stub service) | $0/month |
| `ec2-m5-large-us-west-2.json` | m5.large in us-west-2 (region mismatch) | ERROR |

## Creating Custom Test Data

ResourceDescriptor JSON format:

```json
{
  "resource": {
    "provider": "aws",
    "resource_type": "<ec2|ebs|s3|lambda|rds|dynamodb>",
    "sku": "<instance-type or volume-type>",
    "region": "<aws-region>",
    "tags": {
      "size": "<volume-size-GB>"
    }
  }
}
```

**Required Fields**:

- `provider`: Must be "aws"
- `resource_type`: One of: ec2, ebs, s3, lambda, rds, dynamodb
- `sku`: Instance type (EC2) or volume type (EBS)
- `region`: AWS region code (e.g., "us-east-1")

**Optional Fields**:

- `tags.size`: EBS volume size in GB (defaults to 8 if not specified)
- `tags.volume_size`: Alternative field name for EBS size

## Troubleshooting

**Connection Refused**:

- Verify plugin is running: `ps aux | grep finfocus-plugin`
- Check PORT was announced: Look for `PORT=<port>` in plugin output
- Ensure using correct port number in grpcurl

**gRPC Method Not Found**:

- Verify proto path: `finfocus.v1.CostSourceService/<method>`
- Check plugin implements all required RPCs: Name, Supports, GetProjectedCost

**Invalid JSON**:

- Validate JSON with `jq`: `cat testdata/file.json | jq`
- Ensure proper field names match proto definition

## Performance Testing

Test concurrent RPC calls:

```bash
# Launch 10 concurrent requests
for i in {1..10}; do
  grpcurl -plaintext -d @ localhost:<port> finfocus.v1.CostSourceService/GetProjectedCost \
    < testdata/ec2-t3-micro-us-east-1.json &
done
wait

# Check stderr for performance warnings (>50ms lookups)
```

## Integration Testing

For integration with finfocus-core:

1. Ensure plugin binary is in PATH or specify full path in core config
2. Core will start plugin as subprocess and capture PORT
3. Core connects via gRPC to localhost:<port>
4. Core calls GetProjectedCost for each resource in Pulumi state

See [quickstart.md](../specs/001-finfocus-aws-plugin/quickstart.md) for full integration examples.
