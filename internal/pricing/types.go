package pricing

// pricingData is the top-level structure for unmarshaling embedded JSON
type pricingData struct {
	Region   string                      `json:"region"`
	Currency string                      `json:"currency"`
	EC2      map[string]ec2OnDemandPrice `json:"ec2"`
	EBS      map[string]ebsVolumePrice   `json:"ebs"`
}

// ec2OnDemandPrice represents a single EC2 instance pricing entry
type ec2OnDemandPrice struct {
	InstanceType    string  `json:"instance_type"`
	OperatingSystem string  `json:"operating_system"`
	Tenancy         string  `json:"tenancy"`
	HourlyRate      float64 `json:"hourly_rate"`
}

// ebsVolumePrice represents a single EBS volume type pricing entry
type ebsVolumePrice struct {
	VolumeType     string  `json:"volume_type"`
	RatePerGBMonth float64 `json:"rate_per_gb_month"`
}
