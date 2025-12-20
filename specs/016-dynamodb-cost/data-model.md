# Data Model: DynamoDB Pricing

## Internal Types (`internal/pricing/types.go`)

### `dynamoDBPrice`
Represents the regional pricing for DynamoDB.

```go
type dynamoDBPrice struct {
    OnDemandReadPrice   float64 // $/read request unit
    OnDemandWritePrice  float64 // $/write request unit
    ProvisionedRCUPrice float64 // $/RCU-hour
    ProvisionedWCUPrice float64 // $/WCU-hour
    StoragePrice        float64 // $/GB-month
    Currency            string
}
```

## Extended Interfaces (`internal/pricing/client.go`)

### `PricingClient`
New methods to expose DynamoDB pricing.

```go
type PricingClient interface {
    // ... existing ...
    DynamoDBOnDemandReadPrice() (float64, bool)
    DynamoDBOnDemandWritePrice() (float64, bool)
    DynamoDBStoragePricePerGBMonth() (float64, bool)
    DynamoDBProvisionedRCUPrice() (float64, bool)
    DynamoDBProvisionedWCUPrice() (float64, bool)
}
```

## Relationships

- The `Client` struct in `internal/pricing` will hold a `*dynamoDBPrice` field.
- The `Client` is initialized by parsing embedded JSON files, which will now include `AmazonDynamoDB` entries.
