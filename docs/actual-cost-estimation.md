# Actual Cost Estimation Limitations

The `aws-public` plugin estimates actual costs using public pricing data and runtime calculations. This document explains how it works, its accuracy limitations, and when to use it versus a dedicated FinOps plugin.

## How It Works

The plugin estimates actual costs using the following formula:

```
actual_cost = projected_hourly_rate × hours_running
```

Where:
- `projected_hourly_rate` = Monthly projected cost / 730 hours
- `hours_running` = Time between resource creation timestamp and query end time

## Accuracy Levels

| Resource Origin | Accuracy | Notes |
|-----------------|----------|-------|
| `pulumi up` | **Good** | Pulumi "Created" timestamp matches actual cloud creation time within seconds. |
| `pulumi import` | **Poor** | "Created" timestamp is set to the import time, not the actual cloud creation time. |

## Known Limitations

### 1. Imported Resources Are Inaccurate

When you run `pulumi import` to bring existing cloud resources under Pulumi management, the `Created` timestamp in the state file is set to the time of import. The plugin cannot see when the resource was originally created in AWS.

**Example:**
- EC2 instance launched: **January 1, 2024**
- Imported to Pulumi: **December 24, 2024**
- aws-public sees Created: **December 24, 2024**
- **Estimated runtime: 0 days** (actual: 358 days)
- **Estimated cost: Near $0** (actual: significant)

### 2. No Stop/Start Tracking

Pulumi state tracks resource existence, not runtime state. The plugin assumes the resource has been running 100% of the time since creation. It cannot account for stopped instances (which have lower or zero compute costs).

### 3. No Billing Features

Because the plugin uses public on-demand pricing data, it cannot account for account-specific billing features:
- **Reserved Instances (RIs)**: Upfront payments and discounted hourly rates are ignored.
- **Spot Instances**: Variable spot pricing is not supported; on-demand rates are used.
- **Savings Plans**: Compute/EC2 Savings Plans discounts are not applied.
- **Free Tier**: Free tier usage limits are not tracked.
- **Volume Discounts**: Tiered pricing (e.g., S3/data transfer) is estimated based on the single resource's usage, not aggregated account usage.
- **Credits**: AWS credits are not visible.

## When to Use

Use `aws-public` for actual cost estimation when:
- You need a quick, rough estimate without configuring AWS credentials.
- You are in a development/testing environment.
- You cannot use a plugin that requires read access to your AWS billing data.

## For Accurate Costs

For accurate actual cost reporting, we strongly recommend using a dedicated FinOps plugin that integrates with your cloud provider's billing data:

- **[Vantage Plugin](https://github.com/rshade/pulumicost-plugin-vantage)**: Integrates with Vantage to provide precise historical cost data.
- **AWS Cost Explorer Plugin** (Planned): Will integrate directly with AWS Cost Explorer API.
- **Kubecost Plugin** (Planned): For Kubernetes-specific cost visibility.
