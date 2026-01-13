# Data Model: Batch Recommendation Processing

## Entities

### ResourceDescriptor (Proto)
*From `finfocus.v1`*
- `name`: string
- `resource_type`: string
- `provider`: string
- `region`: string
- `sku`: string
- `tags`: map<string, string>
- `usage`: map<string, double>
- `resource_id`: string (New usage for correlation)

### ProcessingContext
*Internal struct to hold state during request processing*
- `Scope`: `[]*ResourceDescriptor` (The list of resources to analyze)
- `Filter`: `*RecommendationFilter` (Criteria to apply)
- `BatchStats`: `Stats` (Track processed, matched, savings for summary log)

### Stats
*Internal struct for logging aggregation*
- `TotalResources`: int
- `MatchedResources`: int
- `TotalSavings`: float64

## Data Flow

1. **Request Ingestion**: `GetRecommendationsRequest` received.
2. **Normalization**:
   - Input: `TargetResources` (list), `Filter` (struct).
   - Output: `ProcessingContext`.
   - Transformation:
     - If `len(TargetResources) > 0`: `Scope = TargetResources`.
     - Else: `Scope = [ { Sku: Filter.Sku, ... } ]` (Legacy construction).
3. **Filtering**:
   - Input: `ProcessingContext`.
   - Output: `filtered_resources` (`[]*ResourceDescriptor`).
   - Logic: `Select r From Scope Where matches(r, Filter)`.
4. **Estimation & Correlation**:
   - Input: `filtered_resources`.
   - Output: `[]*Recommendation`.
   - Logic: 
     - Map each resource to `generateRecommendation(r)`.
     - **Critical**: Populate `Recommendation.Resource.ResourceId` (or generic correlation field) with `r.ResourceId` > `r.Arn` > `r.Name`.
5. **Logging**:
   - Aggregated summary logged once per batch.
   - Individual warnings logged if correlation ID is missing or resource unsupported.

## Validation Rules

- **Batch Size**: `len(TargetResources) <= 100`.
- **Provider**: `resource.provider == "aws"`.
- **ResourceType**: Must be one of `aws:ec2:Instance`, `aws:ebs:Volume` (current support).