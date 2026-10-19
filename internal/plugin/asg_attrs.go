package plugin

import (
	"fmt"
	"strconv"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const asgLaunchTemplateKey = "launch_template"

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
//  3. "launch_template.instance_type" tag (dot-notation)
//  4. "launch_configuration.instance_type" tag (dot-notation)
//  5. "launch_template" tag parsed as Go map string
//  6. "launch_configuration" tag parsed as Go map string
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
	tags := pricingTags(resource)

	in := newResourceInputs(resource)
	instanceType := resolveInstanceType(resource.GetSku(), tags)
	if instanceType == "" {
		instanceType = instanceTypeFromAttributeStruct(in)
	}

	capacity, capacityDefaulted := resolveDesiredCapacity(tags)
	mixed := tags["mixed_instances_policy"] != "" ||
		in.Present("mixedInstancesPolicy", "mixed_instances_policy")
	if instanceType == "" {
		detail := "cannot determine instance type for ASG: set 'sku' field or 'instance_type' tag"
		if !capacityDefaulted {
			detail += fmt.Sprintf("; desired capacity %d", capacity)
		}
		if mixed {
			detail += "; mixed instance policies not yet supported"
		}
		return ASGAttributes{}, &dt, &PricingUnavailableError{
			Service:       "ASG",
			SKU:           "",
			BillingDetail: detail,
		}
	}
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
//
// Resolution priority (highest to lowest):
//  1. SKU field (resource.Sku)
//  2. "instance_type" tag (flat tag)
//  3. "launch_template.instance_type" tag (dot-notation from Pulumi flattening)
//  4. "launch_configuration.instance_type" tag (dot-notation)
//  5. "launch_template" tag parsed as Go map string (e.g., "map[instance_type:m5.large]")
//  6. "launch_configuration" tag parsed as Go map string
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

	// Pulumi may serialize nested objects as Go map strings (e.g., "map[instance_type:m5.large]").
	// This mirrors how EC2's rootBlockDevice is parsed via parseGoMapString.
	for _, key := range [...]string{asgLaunchTemplateKey, "launch_configuration"} {
		if raw := tags[key]; raw != "" {
			m := parseGoMapString(raw)
			if v := m["instance_type"]; v != "" {
				return v
			}
			if v := m["instanceType"]; v != "" {
				return v
			}
		}
	}

	return ""
}

// instanceTypeFromAttributeStruct reads a nested launch template,
// launch configuration, or mixed-instances policy from attributes.
// Terraform state keeps the instance type on a separate resource, so a
// missing type is normal. A struct on this resource is still honored.
func instanceTypeFromAttributeStruct(in resourceInputs) string {
	for _, key := range []string{
		"launchTemplate", asgLaunchTemplateKey,
		"launchConfiguration", "launch_configuration",
		"mixedInstancesPolicy", "mixed_instances_policy",
	} {
		st, ok := in.Struct(key)
		if !ok {
			continue
		}
		if v := structString(st, "instanceType", "instance_type"); v != "" {
			return v
		}
		for _, nested := range []string{"launchTemplate", asgLaunchTemplateKey} {
			inner, innerOK := structFieldStruct(st, nested)
			if !innerOK {
				continue
			}
			if v := structString(inner, "instanceType", "instance_type"); v != "" {
				return v
			}
		}
	}
	return ""
}

func structString(st *structpb.Struct, keys ...string) string {
	if st == nil {
		return ""
	}
	for _, key := range keys {
		val, ok := st.GetFields()[key]
		if !ok {
			continue
		}
		if s, parsed := scalarString(val); parsed {
			return s
		}
	}
	return ""
}

func structFieldStruct(st *structpb.Struct, key string) (*structpb.Struct, bool) {
	if st == nil {
		return nil, false
	}
	val, ok := st.GetFields()[key]
	if !ok || val == nil {
		return nil, false
	}
	if nested := val.GetStructValue(); nested != nil {
		return nested, true
	}
	list := val.GetListValue()
	if list == nil || len(list.GetValues()) == 0 {
		return nil, false
	}
	nested := list.GetValues()[0].GetStructValue()
	if nested == nil {
		return nil, false
	}
	return nested, true
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
					return 0, true
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
