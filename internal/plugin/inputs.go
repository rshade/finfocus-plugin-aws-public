package plugin

import (
	"math"
	"strconv"
	"strings"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// resourceInputs is the only reader of resource pricing inputs.
//
// Precedence follows ResourceDescriptor.attributes: an unset or empty
// attributes struct means the host sent none, and the lookup falls back to
// tags. When the same property is in both, attributes win. Within one source,
// keys are checked in the order the caller lists them, so an explicit
// canonical key wins over an alias when the caller names it first.
//
// Plugins must not log the attributes struct. This type returns extracted
// scalars only.
type resourceInputs struct {
	attrs *structpb.Struct
	tags  map[string]string
}

// newResourceInputs builds the lookup for one descriptor. A nil descriptor
// yields an empty lookup.
func newResourceInputs(resource *pbc.ResourceDescriptor) resourceInputs {
	if resource == nil {
		return resourceInputs{attrs: nil, tags: nil}
	}
	return resourceInputs{attrs: resource.GetAttributes(), tags: resource.GetTags()}
}

// newAttributeInputs builds the lookup for EstimateCostRequest.attributes,
// which has no tag fallback.
func newAttributeInputs(attrs *structpb.Struct) resourceInputs {
	return resourceInputs{attrs: attrs, tags: nil}
}

// String returns the first value among keys.
//
// Attributes are searched before tags. An empty attribute string is skipped
// so a tag can still supply the value. A tag that is present and empty is
// returned, so callers can reject an explicit empty tag. Numbers, booleans,
// and one-element lists are stringified. A one-key struct collapses to that
// value, matching finfocus core's ConvertValueToString.
func (in resourceInputs) String(keys ...string) (string, bool) {
	for _, key := range keys {
		if s, ok := attributeString(in.attrs, key); ok && s != "" {
			return s, true
		}
	}
	for _, key := range keys {
		v, ok := in.tags[key]
		if !ok {
			continue
		}
		return v, true
	}
	return "", false
}

// Float returns the first numeric value among keys.
// Number attributes and numeric strings are accepted. A non-numeric present
// value is skipped so a later key can match.
func (in resourceInputs) Float(keys ...string) (float64, bool) {
	return in.number(keys...)
}

// Int returns the first whole number among keys.
// Number attributes and integer strings are accepted. Values above 2^53 must
// arrive as strings, and those strings are parsed as integers rather than
// floats. A fractional number is not an integer.
func (in resourceInputs) Int(keys ...string) (int, bool) {
	for _, key := range keys {
		v, ok := pluginsdk.AttributeValue(in.attrs, key)
		if !ok || v == nil {
			continue
		}
		if n, parsed := valueInt(v); parsed {
			return n, true
		}
	}
	for _, key := range keys {
		raw, ok := in.tags[key]
		if !ok || raw == "" {
			continue
		}
		if n, parsed := parseIntString(raw); parsed {
			return n, true
		}
	}
	return 0, false
}

// Bool returns the first boolean among keys.
// Bool attributes and the strings "true" and "false" (any case) are accepted.
func (in resourceInputs) Bool(keys ...string) (bool, bool) {
	for _, key := range keys {
		v, ok := pluginsdk.AttributeValue(in.attrs, key)
		if !ok || v == nil {
			continue
		}
		switch kind := v.GetKind().(type) {
		case *structpb.Value_BoolValue:
			return kind.BoolValue, true
		case *structpb.Value_StringValue:
			if b, parsed := parseBoolString(kind.StringValue); parsed {
				return b, true
			}
		}
	}
	for _, key := range keys {
		raw, ok := in.tags[key]
		if !ok || raw == "" {
			continue
		}
		if b, parsed := parseBoolString(raw); parsed {
			return b, true
		}
	}
	return false, false
}

// Struct returns the struct at path. A one-element list of structs uses the
// first element. The path uses pluginsdk.AttributeValue rules.
func (in resourceInputs) Struct(path string) (*structpb.Struct, bool) {
	v, ok := pluginsdk.AttributeValue(in.attrs, path)
	if !ok || v == nil {
		return nil, false
	}
	if st := v.GetStructValue(); st != nil {
		return st, true
	}
	list := v.GetListValue()
	if list == nil || len(list.GetValues()) == 0 {
		return nil, false
	}
	st := list.GetValues()[0].GetStructValue()
	if st == nil {
		return nil, false
	}
	return st, true
}

// Present reports whether any key has a non-empty string or a nested struct.
func (in resourceInputs) Present(keys ...string) bool {
	if s, ok := in.String(keys...); ok && s != "" {
		return true
	}
	for _, key := range keys {
		if _, ok := in.Struct(key); ok {
			return true
		}
	}
	return false
}

func (in resourceInputs) number(keys ...string) (float64, bool) {
	for _, key := range keys {
		v, ok := pluginsdk.AttributeValue(in.attrs, key)
		if !ok || v == nil {
			continue
		}
		if f, parsed := valueNumber(v); parsed {
			return f, true
		}
	}
	for _, key := range keys {
		raw, ok := in.tags[key]
		if !ok || raw == "" {
			continue
		}
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		return f, true
	}
	return 0, false
}

func attributeString(attrs *structpb.Struct, key string) (string, bool) {
	v, ok := pluginsdk.AttributeValue(attrs, key)
	if !ok || v == nil {
		return "", false
	}
	if _, isNull := v.GetKind().(*structpb.Value_NullValue); isNull {
		return "", false
	}
	return scalarString(v)
}

// scalarString stringifies a scalar, a one-element list, or a one-key struct.
// An empty string and a multi-key struct return ("", false).
func scalarString(v *structpb.Value) (string, bool) {
	if v == nil {
		return "", false
	}
	switch kind := v.GetKind().(type) {
	case *structpb.Value_StringValue:
		if kind.StringValue == "" {
			return "", false
		}
		return kind.StringValue, true
	case *structpb.Value_NumberValue:
		return strconv.FormatFloat(kind.NumberValue, 'f', -1, 64), true
	case *structpb.Value_BoolValue:
		return strconv.FormatBool(kind.BoolValue), true
	case *structpb.Value_ListValue:
		if kind.ListValue == nil || len(kind.ListValue.GetValues()) == 0 {
			return "", false
		}
		return scalarString(kind.ListValue.GetValues()[0])
	case *structpb.Value_StructValue:
		fields := kind.StructValue.GetFields()
		if len(fields) != 1 {
			return "", false
		}
		for _, inner := range fields {
			return scalarString(inner)
		}
	}
	return "", false
}

func valueInt(v *structpb.Value) (int, bool) {
	switch kind := v.GetKind().(type) {
	case *structpb.Value_NumberValue:
		f := kind.NumberValue
		if f != math.Trunc(f) || f > math.MaxInt || f < math.MinInt {
			return 0, false
		}
		return int(f), true
	case *structpb.Value_StringValue:
		return parseIntString(kind.StringValue)
	default:
		return 0, false
	}
}

func parseIntString(raw string) (int, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n > math.MaxInt || n < math.MinInt {
		return 0, false
	}
	return int(n), true
}

func valueNumber(v *structpb.Value) (float64, bool) {
	switch kind := v.GetKind().(type) {
	case *structpb.Value_NumberValue:
		return kind.NumberValue, true
	case *structpb.Value_StringValue:
		if kind.StringValue == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(kind.StringValue, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func parseBoolString(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

// actualCostTags returns GetActualCostRequest tags. Request tags are cloud
// labels on the descriptor path. The call lives here so pricing code does
// not read tags itself.
func actualCostTags(req *pbc.GetActualCostRequest) map[string]string {
	if req == nil {
		return nil
	}
	return req.GetTags()
}

// recommendationFilterTags returns recommendation filter tags.
func recommendationFilterTags(filter *pbc.RecommendationFilter) map[string]string {
	if filter == nil {
		return nil
	}
	return filter.GetTags()
}

// descriptorTags returns a descriptor's tag map. Canonicalization and
// non-pricing readers use this so GetTags stays in this file.
func descriptorTags(resource *pbc.ResourceDescriptor) map[string]string {
	if resource == nil {
		return nil
	}
	return resource.GetTags()
}

// ensureDescriptorTags returns the descriptor tag map, creating it when absent.
// Callers that copy an alias onto a canonical key use this so the write lands
// on the descriptor the estimator receives.
func ensureDescriptorTags(resource *pbc.ResourceDescriptor) map[string]string {
	if resource == nil {
		return nil
	}
	if resource.GetTags() == nil {
		resource.Tags = map[string]string{}
	}
	return resource.GetTags()
}
