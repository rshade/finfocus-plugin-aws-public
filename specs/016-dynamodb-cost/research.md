# Research: DynamoDB Pricing Extraction

This document outlines the research and decisions for extracting DynamoDB pricing from the AWS Public Pricing API.

## Decision: Product Selection

**Decision**: Use `servicecode: AmazonDynamoDB` to filter for DynamoDB products.

**Rationale**: This is the standard AWS service code for DynamoDB in the Pricing API.

## Decision: Capacity Mode Filtering

**Decision**: Use `ProductFamily` and `group`/`usagetype` to distinguish between On-Demand and Provisioned capacity.

### On-Demand Mode
- **ProductFamily**: `Amazon DynamoDB PayPerRequest Throughput`
- **Read Requests**: `group: DDB-ReadUnits`
- **Write Requests**: `group: DDB-WriteUnits`
- **Unit**: Per Request (pricing is per 1M units, so we divide the API rate if it's per unit, or use as-is if per million. *Clarification: AWS usually prices per 1 million units in documentation, but the API may return per-unit price.*)

### Provisioned Mode
- **ProductFamily**: `Provisioned IOPS`
- **Read Units**: `usagetype` containing `ReadCapacityUnit`
- **Write Units**: `usagetype` containing `WriteCapacityUnit`
- **Unit**: Per RCU/WCU-Hour.

### Storage
- **ProductFamily**: `Database Storage`
- **Filter**: `usagetype` containing `TimedStorage-ByteHrs` (standard table storage).

## Decision: Pricing Logic Consolidation

**Decision**: All DynamoDB pricing for a region will be consolidated into a single `dynamoDBPrice` struct per region binary.

**Rationale**: DynamoDB doesn't have "instance types" like EC2, so a single struct per region is efficient and simple.

## Alternatives Considered

- **Multiple pricing structs per region**: Rejected as over-complicated; DynamoDB pricing is region-wide.
- **Dynamic fetching at runtime**: Rejected per Constitution (no network calls at runtime).
