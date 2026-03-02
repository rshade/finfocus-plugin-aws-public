# Quickstart: Port Binding & Contract Violation Fixes

## Running with the `--port` Flag

To start the plugin on a specific port, use the `--port` flag:

```bash
./finfocus-plugin-aws-public-us-east-1 --port 9090
```

Verify it is listening on the correct port:
```bash
# In the logs, you should see:
# PORT=9090
```

## Verifying Contract Compliance (Fallback)

To verify that the plugin correctly returns empty results when pricing is missing (allowing Core fallback):

1. Request actual cost for an unknown instance type (e.g., `z9.superlarge`).
2. Verify the gRPC response `Results` field is an empty array.

```bash
# Using grpcurl
grpcurl -plaintext -d '{"resource": {"type": "ec2", "metadata": {"InstanceType": "z9.superlarge"}}}' localhost:9090 finfocus.v1.CostSourceService/GetActualCost
```

## Verifying Zero-Cost Resources

Verify that IAM roles or VPCs return $0 as expected:

```bash
grpcurl -plaintext -d '{"resource": {"type": "iam"}}' localhost:9090 finfocus.v1.CostSourceService/GetActualCost
```
