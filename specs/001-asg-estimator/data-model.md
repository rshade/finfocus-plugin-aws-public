# Data Model: ASG Cost Estimator

**Feature**: 001-asg-estimator | **Date**: 2026-03-26

## Entities

### ASGAttributes

Extracted from ResourceDescriptor tags for cost calculation.

| Field | Type | Source Priority | Default | Validation |
| ----- | ---- | --------------- | ------- | ---------- |
| InstanceType | string | sku → instance_type → launch_template.instance_type → launch_configuration.instance_type | (none — error) | Must be non-empty |
| DesiredCapacity | int | desired_capacity → desiredCapacity → min_size → minSize | 1 | Must be ≥ 0 |
| OS | string | operating_system → platform | "Linux" | "Linux", "Windows", "RHEL", "SUSE" |

### ASGConfig (Carbon)

Input to the ASG carbon estimator.

| Field | Type | Description |
| ----- | ---- | ----------- |
| InstanceType | string | EC2 instance type (e.g., "m5.large") |
| Region | string | AWS region for grid emission factor lookup |
| DesiredCapacity | int | Number of instances |
| Utilization | float64 | CPU utilization (0.0-1.0), default 0.5 |
| Hours | float64 | Hours per month (730 production, 160 dev) |

## Relationships

```text
AutoScalingGroup ──manages──▶ EC2 Instances (desiredCapacity count)
       │
       └──references──▶ LaunchTemplate OR LaunchConfiguration (instance type source)
```

- ASG → EC2: One-to-many. Cost = per-instance × count.
- ASG → LaunchTemplate: One-to-one. Configuration reference only (zero-cost
  resource). Instance type extracted from template properties in tags.

## Tag Key Mapping

Maps Pulumi state property names to canonical tag keys used in extraction.

| Pulumi Property | Tag Key (snake_case) | Tag Key (camelCase) |
| --------------- | -------------------- | ------------------- |
| desiredCapacity | desired_capacity | desiredCapacity |
| minSize | min_size | minSize |
| maxSize | max_size | maxSize |
| launchTemplate.instanceType | launch_template.instance_type | N/A |
| launchConfiguration.instanceType | launch_configuration.instance_type | N/A |

## Service Classification Entry

```text
Key: "aws:autoscaling:autoScalingGroup"
GrowthType: GROWTH_TYPE_STATIC (fixed cost)
AffectedByDevMode: true (160 hrs/month in dev)
ParentTagKeys: ["vpc_id"]
ParentType: "aws:ec2:vpc:Vpc"
Relationship: RelationshipWithin
```
