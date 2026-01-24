package plugin

import (
	"testing"
)

// TestNewServiceResolver verifies that newServiceResolver creates a resolver
// with the correct initial state: original field set, other fields empty,
// and initialized flag false.
//
// This test ensures the lazy initialization contract is established at construction
// time - no computation should occur until accessor methods are called.
//
// Test workflow:
//  1. Creates resolvers with various resource type inputs (simple, Pulumi, empty)
//  2. Verifies the original field matches the input exactly
//  3. Confirms initialized flag is false (lazy initialization not triggered)
//  4. Validates normalizedType and serviceType are empty strings
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//
// Run with: go test -v ./internal/plugin/... -run TestNewServiceResolver
func TestNewServiceResolver(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{
			name:         "ec2 simple type",
			resourceType: "ec2",
		},
		{
			name:         "pulumi format eks",
			resourceType: "aws:eks/cluster:Cluster",
		},
		{
			name:         "empty string",
			resourceType: "",
		},
		{
			name:         "ebs volume type",
			resourceType: "ebs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := newServiceResolver(tt.resourceType)

			if resolver == nil {
				t.Fatal("newServiceResolver returned nil")
			}
			if resolver.original != tt.resourceType {
				t.Errorf("original = %q, want %q", resolver.original, tt.resourceType)
			}
			if resolver.initialized {
				t.Error("initialized should be false before first access")
			}
			if resolver.normalizedType != "" {
				t.Errorf("normalizedType should be empty before first access, got %q", resolver.normalizedType)
			}
			if resolver.serviceType != "" {
				t.Errorf("serviceType should be empty before first access, got %q", resolver.serviceType)
			}
		})
	}
}

// TestServiceResolver_LazyInitialization verifies that the resolver performs
// lazy initialization - computation only happens on first access to
// NormalizedType() or ServiceType().
//
// This test validates the core lazy initialization behavior that enables
// early request validation failures to skip expensive normalization/detection.
//
// Test workflow:
//  1. Creates resolver with various resource types (simple and Pulumi formats)
//  2. Verifies initialized flag is false before any accessor call
//  3. Calls either NormalizedType() or ServiceType() first (based on test case)
//  4. Confirms initialized flag becomes true after first access
//  5. Validates both cached values are correct regardless of access order
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//
// Run with: go test -v ./internal/plugin/... -run TestServiceResolver_LazyInitialization
func TestServiceResolver_LazyInitialization(t *testing.T) {
	tests := []struct {
		name                 string
		resourceType         string
		expectedNormalized   string
		expectedService      string
		accessNormalizedFirst bool
	}{
		{
			name:                 "access ServiceType first - ec2",
			resourceType:         "ec2",
			expectedNormalized:   "ec2",
			expectedService:      "ec2",
			accessNormalizedFirst: false,
		},
		{
			name:                 "access NormalizedType first - ec2",
			resourceType:         "ec2",
			expectedNormalized:   "ec2",
			expectedService:      "ec2",
			accessNormalizedFirst: true,
		},
		{
			name:                 "pulumi format eks - service first",
			resourceType:         "aws:eks/cluster:Cluster",
			expectedNormalized:   "eks",
			expectedService:      "eks",
			accessNormalizedFirst: false,
		},
		{
			name:                 "pulumi format ebs - normalized first",
			resourceType:         "aws:ec2/volume:Volume",
			expectedNormalized:   "ebs",
			expectedService:      "ebs",
			accessNormalizedFirst: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := newServiceResolver(tt.resourceType)

			// Verify not initialized before access
			if resolver.initialized {
				t.Fatal("resolver should not be initialized before first access")
			}

			// Access in specified order
			if tt.accessNormalizedFirst {
				normalized := resolver.NormalizedType()
				if normalized != tt.expectedNormalized {
					t.Errorf("NormalizedType() = %q, want %q", normalized, tt.expectedNormalized)
				}
			} else {
				service := resolver.ServiceType()
				if service != tt.expectedService {
					t.Errorf("ServiceType() = %q, want %q", service, tt.expectedService)
				}
			}

			// Verify initialized after first access
			if !resolver.initialized {
				t.Error("resolver should be initialized after first access")
			}

			// Verify both values are now available
			if resolver.NormalizedType() != tt.expectedNormalized {
				t.Errorf("NormalizedType() after init = %q, want %q", resolver.NormalizedType(), tt.expectedNormalized)
			}
			if resolver.ServiceType() != tt.expectedService {
				t.Errorf("ServiceType() after init = %q, want %q", resolver.ServiceType(), tt.expectedService)
			}
		})
	}
}

// TestServiceResolver_Memoization verifies that repeated calls to NormalizedType()
// and ServiceType() return the same cached values without recomputation.
//
// This test ensures the memoization pattern works correctly - values should be
// computed exactly once and all subsequent calls should return identical cached results.
//
// Test workflow:
//  1. Creates a resolver with a Pulumi-format RDS resource type
//  2. Calls ServiceType() and NormalizedType() to trigger initialization
//  3. Records the returned values as the expected baseline
//  4. Calls both methods 10 additional times in a loop
//  5. Verifies each call returns exactly the same values as the baseline
//  6. Confirms the final cached values match expected service type ("rds")
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//
// Run with: go test -v ./internal/plugin/... -run TestServiceResolver_Memoization
func TestServiceResolver_Memoization(t *testing.T) {
	resolver := newServiceResolver("aws:rds/instance:Instance")

	// First access
	service1 := resolver.ServiceType()
	normalized1 := resolver.NormalizedType()

	// Multiple subsequent accesses
	for i := 0; i < 10; i++ {
		service := resolver.ServiceType()
		normalized := resolver.NormalizedType()

		if service != service1 {
			t.Errorf("ServiceType() call %d = %q, want %q", i, service, service1)
		}
		if normalized != normalized1 {
			t.Errorf("NormalizedType() call %d = %q, want %q", i, normalized, normalized1)
		}
	}

	// Verify the cached values are what we expect
	if service1 != "rds" {
		t.Errorf("ServiceType() = %q, want %q", service1, "rds")
	}
}

// TestServiceResolver_EdgeCases verifies behavior with edge case inputs:
// empty strings, malformed types, and unknown resource types.
//
// This test ensures the resolver handles invalid/unusual inputs gracefully
// without panicking, which is critical for production stability.
//
// Test workflow:
//  1. Defines table of edge case inputs (empty, whitespace, malformed, special chars)
//  2. For each case, creates a resolver and defers panic recovery
//  3. Calls both NormalizedType() and ServiceType() accessors
//  4. Verifies no panic occurred (unless explicitly expected)
//  5. Confirms resolver is marked as initialized after access
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//
// Run with: go test -v ./internal/plugin/... -run TestServiceResolver_EdgeCases
func TestServiceResolver_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		wantPanic    bool
	}{
		{
			name:         "empty string",
			resourceType: "",
			wantPanic:    false,
		},
		{
			name:         "whitespace only",
			resourceType: "   ",
			wantPanic:    false,
		},
		{
			name:         "unknown service type",
			resourceType: "completely-unknown-type",
			wantPanic:    false,
		},
		{
			name:         "malformed pulumi format - missing parts",
			resourceType: "aws:",
			wantPanic:    false,
		},
		{
			name:         "malformed pulumi format - extra colons",
			resourceType: "aws:ec2:volume:extra:parts",
			wantPanic:    false,
		},
		{
			name:         "special characters",
			resourceType: "!@#$%^&*()",
			wantPanic:    false,
		},
		{
			name:         "very long string",
			resourceType: "aws:verylongservicenamethatexceedsnormallength/verylongresourcenamethatexceedsnormallength:VeryLongResourceType",
			wantPanic:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			resolver := newServiceResolver(tt.resourceType)

			// Should not panic on access
			_ = resolver.NormalizedType()
			_ = resolver.ServiceType()

			// Should be initialized after access
			if !resolver.initialized {
				t.Error("resolver should be initialized after access")
			}

			if tt.wantPanic {
				t.Error("expected panic but none occurred")
			}
		})
	}
}

// TestServiceResolver_EmptyStringBehavior verifies that empty input produces
// empty output, matching the existing behavior of normalizeResourceType and
// detectService with empty inputs.
//
// This test ensures backward compatibility with the underlying functions -
// the resolver should preserve their empty-in-empty-out behavior.
//
// Test workflow:
//  1. Creates a resolver with an empty string input
//  2. Calls NormalizedType() and captures the result
//  3. Calls ServiceType() and captures the result
//  4. Verifies both return empty strings (not nil, not error)
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//
// Run with: go test -v ./internal/plugin/... -run TestServiceResolver_EmptyStringBehavior
func TestServiceResolver_EmptyStringBehavior(t *testing.T) {
	resolver := newServiceResolver("")

	normalized := resolver.NormalizedType()
	service := resolver.ServiceType()

	// Empty input should produce empty output (matching existing behavior)
	if normalized != "" {
		t.Errorf("NormalizedType() for empty input = %q, want empty string", normalized)
	}
	if service != "" {
		t.Errorf("ServiceType() for empty input = %q, want empty string", service)
	}
}

// TestServiceResolver_AllSupportedServices verifies that the resolver correctly
// handles all service types that the plugin supports.
//
// This test provides comprehensive coverage of the resolver's integration with
// normalizeResourceType() and detectService() for all supported AWS services.
//
// Note: The expected values are based on the actual behavior of detectService()
// in projected.go, which includes specific mappings (e.g., alb/nlb -> elb).
//
// Test workflow:
//  1. Defines table of all supported resource types (simple and Pulumi formats)
//  2. Includes canonical forms, aliases (alb/nlb), and Pulumi-format types
//  3. For each type, creates a resolver and calls ServiceType()
//  4. Verifies the returned service type matches the expected mapping
//
// Prerequisites:
//   - No external dependencies or setup required
//   - Pure unit test with no side effects
//   - Test expectations must be updated if detectService() mappings change
//
// Run with: go test -v ./internal/plugin/... -run TestServiceResolver_AllSupportedServices
func TestServiceResolver_AllSupportedServices(t *testing.T) {
	tests := []struct {
		resourceType    string
		expectedService string
	}{
		// Simple types - canonical forms
		{"ec2", "ec2"},
		{"ebs", "ebs"},
		{"s3", "s3"},
		{"rds", "rds"},
		{"eks", "eks"},
		{"lambda", "lambda"},
		{"dynamodb", "dynamodb"},
		{"elb", "elb"},
		{"natgw", "natgw"},
		{"cloudwatch", "cloudwatch"},
		{"elasticache", "elasticache"},

		// ALB/NLB are normalized to ELB by detectService
		{"alb", "elb"},
		{"nlb", "elb"},

		// Pulumi format types - these go through normalizeResourceType first
		{"aws:ec2/instance:Instance", "ec2"},
		{"aws:ec2/volume:Volume", "ebs"},
		{"aws:s3/bucket:Bucket", "s3"},
		{"aws:rds/instance:Instance", "rds"},
		{"aws:eks/cluster:Cluster", "eks"},
		{"aws:lambda/function:Function", "lambda"},
		{"aws:dynamodb/table:Table", "dynamodb"},
		{"aws:cloudwatch/logGroup:LogGroup", "cloudwatch"},
		{"aws:elasticache/cluster:Cluster", "elasticache"},
		// Note: aws:ec2/natGateway:NatGateway currently resolves to "ec2" because
		// normalizeResourceType() extracts just the service prefix ("ec2"), not the
		// subresource. This is consistent with the two-step normalization pattern.
		// Use "natgw" or "nat_gateway" for explicit NAT Gateway resource types.
		{"aws:ec2/natGateway:NatGateway", "ec2"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			resolver := newServiceResolver(tt.resourceType)
			service := resolver.ServiceType()

			if service != tt.expectedService {
				t.Errorf("ServiceType() for %q = %q, want %q", tt.resourceType, service, tt.expectedService)
			}
		})
	}
}
