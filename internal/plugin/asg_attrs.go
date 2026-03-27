package plugin

import (
	"strconv"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ASGAttributes contains extracted ASG configuration for cost estimation.
type ASGAttributes struct {
	InstanceType    string
	DesiredCapacity int
	OS              string
}

// ExtractASGAttributes extracts ASG configuration from a ResourceDescriptor.
//
// Instance type resolution priority (highest to lowest):
//  1. resource.Sku field
//  2. "instance_type" tag
//  3. "launch_template.instance_type" tag
//  4. "launch_configuration.instance_type" tag
//
// Capacity resolution priority:
//  1. "desired_capacity" or "desiredCapacity" tag
//  2. "min_size" or "minSize" tag
//  3. Default: 1
//
// OS resolution: "operating_system" or "platform" tag, default "Linux".
//
// Returns the extracted attributes and a DefaultsTracker recording any defaults applied.
// Returns an error if no instance type can be resolved.
func ExtractASGAttributes(resource *pbc.ResourceDescriptor) (ASGAttributes, *DefaultsTracker, error) {
	var dt DefaultsTracker
	tags := resource.GetTags()
	if tags == nil {
		tags = map[string]string{}
	}

	instanceType := resolveInstanceType(resource.GetSku(), tags)
	if instanceType == "" {
		return ASGAttributes{}, &dt, &PricingUnavailableError{
			Service:       "ASG",
			SKU:           "",
			BillingDetail: "cannot determine instance type for ASG: set 'sku' field or 'instance_type' tag",
		}
	}

	capacity, capacityDefaulted := resolveDesiredCapacity(tags)
	if capacityDefaulted {
		dt.Add("desired_capacity", "1", KindConfig)
	}

	os := resolveASGOS(tags)
	if os == defaultOS && tags["operating_system"] == "" && tags["platform"] == "" {
		dt.Add("operating_system", defaultOS, KindConfig)
	}

	return ASGAttributes{
		InstanceType:    instanceType,
		DesiredCapacity: capacity,
		OS:              os,
	}, &dt, nil
}

// resolveInstanceType resolves the EC2 instance type using priority-based lookup.
func resolveInstanceType(sku string, tags map[string]string) string {
	if sku != "" {
		return sku
	}
	if v := tags["instance_type"]; v != "" {
		return v
	}
	if v := tags["launch_template.instance_type"]; v != "" {
		return v
	}
	if v := tags["launch_configuration.instance_type"]; v != "" {
		return v
	}
	return ""
}

// resolveDesiredCapacity resolves the desired capacity from tags.
// Returns the capacity value and whether the default was used.
// Priority: desired_capacity → desiredCapacity → min_size → minSize → default 1.
func resolveDesiredCapacity(tags map[string]string) (int, bool) {
	for _, key := range [...]string{
		"desired_capacity", "desiredCapacity",
		"min_size", "minSize",
	} {
		if v := tags[key]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				if n < 0 {
					return 0, false
				}
				return n, false
			}
		}
	}
	return 1, true
}

// resolveASGOS resolves the operating system from tags.
func resolveASGOS(tags map[string]string) string {
	if v := tags["operating_system"]; v != "" {
		return normalizePlatform(v)
	}
	if v := tags["platform"]; v != "" {
		return normalizePlatform(v)
	}
	return defaultOS
}
