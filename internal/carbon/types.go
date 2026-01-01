// Package carbon provides carbon emission estimation for AWS resources
// using Cloud Carbon Footprint (CCF) methodology.
package carbon

// TotalCarbonEstimate contains composite carbon result with operational and embodied carbon.
type TotalCarbonEstimate struct {
	// OperationalCarbon is carbon from energy consumption (compute + storage) in gCO2e.
	OperationalCarbon float64

	// EmbodiedCarbon is carbon from hardware manufacturing (amortized monthly) in gCO2e.
	EmbodiedCarbon float64

	// TotalCarbon is the sum of operational + embodied carbon in gCO2e.
	TotalCarbon float64
}

// EmbodiedCarbonConfig contains configuration for embodied carbon calculation.
type EmbodiedCarbonConfig struct {
	// Enabled indicates whether to include embodied carbon in estimates.
	Enabled bool

	// ServerLifespanMonths is the server lifespan for amortization (default: 48 months / 4 years).
	ServerLifespanMonths int

	// EmbodiedCarbonPerServer is the total embodied carbon per server in kgCO2e (default: 1000).
	EmbodiedCarbonPerServer float64
}

// DefaultEmbodiedCarbonConfig returns the default embodied carbon configuration
// based on CCF methodology.
func DefaultEmbodiedCarbonConfig() EmbodiedCarbonConfig {
	return EmbodiedCarbonConfig{
		Enabled:                 false,
		ServerLifespanMonths:    48,   // 4 years
		EmbodiedCarbonPerServer: 1000, // kgCO2e
	}
}

// EC2InstanceConfig contains configuration for EC2 instance carbon estimation.
type EC2InstanceConfig struct {
	// InstanceType is the EC2 instance type (e.g., "t3.micro").
	InstanceType string

	// Region is the AWS region.
	Region string

	// Utilization is the CPU utilization (0.0 to 1.0, default: 0.50).
	Utilization float64

	// Hours is the operating hours.
	Hours float64

	// IncludeGPU indicates whether to include GPU power consumption (default: true).
	IncludeGPU bool

	// IncludeEmbodiedCarbon indicates whether to include embodied carbon (default: false).
	IncludeEmbodiedCarbon bool

	// EmbodiedConfig is the embodied carbon configuration (used if IncludeEmbodiedCarbon is true).
	EmbodiedConfig EmbodiedCarbonConfig
}

// EBSVolumeConfig contains configuration for EBS volume carbon estimation.
type EBSVolumeConfig struct {
	// VolumeType is the EBS volume type (gp2, gp3, io1, io2, st1, sc1).
	VolumeType string

	// SizeGB is the volume size in gigabytes.
	SizeGB float64

	// Region is the AWS region.
	Region string

	// Hours is the storage duration in hours.
	Hours float64
}

// S3StorageConfig contains configuration for S3 storage carbon estimation.
type S3StorageConfig struct {
	// StorageClass is the S3 storage class (STANDARD, STANDARD_IA, etc.).
	StorageClass string

	// SizeGB is the storage size in gigabytes.
	SizeGB float64

	// Region is the AWS region.
	Region string

	// Hours is the storage duration in hours.
	Hours float64
}

// LambdaFunctionConfig contains configuration for Lambda function carbon estimation.
type LambdaFunctionConfig struct {
	// MemoryMB is the allocated memory in megabytes.
	MemoryMB int

	// DurationMs is the average invocation duration in milliseconds.
	DurationMs int

	// Invocations is the total number of invocations.
	Invocations int64

	// Architecture is the CPU architecture (x86_64, arm64). Defaults to x86_64.
	Architecture string

	// Region is the AWS region.
	Region string
}

// RDSInstanceConfig contains configuration for RDS instance carbon estimation.
type RDSInstanceConfig struct {
	// InstanceType is the RDS instance class (EC2-equivalent, e.g., "db.m5.large").
	InstanceType string

	// Region is the AWS region.
	Region string

	// MultiAZ indicates Multi-AZ deployment (doubles compute and storage carbon).
	MultiAZ bool

	// StorageType is the storage type (gp2, gp3, io1, io2).
	StorageType string

	// StorageSizeGB is the storage size in gigabytes.
	StorageSizeGB float64

	// Utilization is the CPU utilization (0.0 to 1.0, default: 0.50).
	Utilization float64

	// Hours is the operating hours.
	Hours float64
}

// DynamoDBTableConfig contains configuration for DynamoDB table carbon estimation.
type DynamoDBTableConfig struct {
	// SizeGB is the table storage size in gigabytes.
	SizeGB float64

	// Region is the AWS region.
	Region string

	// Hours is the storage duration in hours.
	Hours float64
}

// EKSClusterConfig contains configuration for EKS cluster carbon estimation.
type EKSClusterConfig struct {
	// Region is the AWS region.
	Region string
}
