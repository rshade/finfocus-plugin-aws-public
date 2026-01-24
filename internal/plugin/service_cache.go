package plugin

// serviceResolver caches the normalized resource type and detected service
// for a single ResourceDescriptor within one request lifecycle.
//
// This type implements a memoization pattern to avoid redundant calls to
// normalizeResourceType() and detectService() during request processing.
// Previously, these functions were called 2-3 times per request across
// validation, support checks, and cost routing. With serviceResolver,
// the computation happens exactly once per resource.
//
// Thread Safety: serviceResolver is designed for single-goroutine use within
// one gRPC request. Each request creates its own instance, so no synchronization
// is required. The struct is safe for concurrent reads after initialization.
//
// Memory Profile: ~64 bytes per instance (3 string headers + bool + padding).
// For batch operations (100 resources), this is ~6.4 KB total, well within
// the <100 bytes per resource constraint (SC-005).
//
// Usage:
//
//	resolver := newServiceResolver(resource.ResourceType)
//	normalized := resolver.NormalizedType()  // computed on first call, cached
//	service := resolver.ServiceType()         // computed on first call, cached
//
// Lifecycle:
//  1. Created via newServiceResolver() at request entry point
//  2. Initialized lazily on first access to NormalizedType() or ServiceType()
//  3. Subsequent accesses return cached values (no recomputation)
//  4. Garbage collected when request completes (no cleanup needed)
type serviceResolver struct {
	// original is the resource_type string from ResourceDescriptor, stored as-is.
	// This field is immutable after construction.
	original string

	// normalizedType is the result of normalizeResourceType(original).
	// Computed lazily on first access to NormalizedType() or ServiceType().
	normalizedType string

	// serviceType is the result of detectService(normalizedType).
	// Computed lazily on first access to ServiceType().
	serviceType string

	// initialized tracks whether computation has occurred.
	// Once true, normalizedType and serviceType are immutable.
	initialized bool
}

// newServiceResolver creates a new serviceResolver for the given resource type string.
//
// The resolver performs lazy initialization - no computation happens until
// NormalizedType() or ServiceType() is called. This allows early validation
// failures to skip the computation entirely.
//
// Parameters:
//   - resourceType: The original resource_type from ResourceDescriptor
//     (e.g., "ec2", "aws:eks/cluster:Cluster", "ebs")
//
// Returns:
//   - A pointer to a new serviceResolver instance
//
// Example:
//
//	resolver := newServiceResolver(resource.ResourceType)
//	if resolver.ServiceType() == "ec2" {
//	    return p.estimateEC2(traceID, resource, req)
//	}
func newServiceResolver(resourceType string) *serviceResolver {
	return &serviceResolver{
		original: resourceType,
	}
}

// NormalizedType returns the normalized resource type string.
//
// On first call, this method invokes ensureInitialized() which computes
// and caches both the normalized type (via normalizeResourceType(original))
// and the service type (via detectService(normalizedType)). Subsequent calls
// to either NormalizedType() or ServiceType() return the cached values
// without recomputation.
//
// The normalized type handles Pulumi-format resource types like
// "aws:eks/cluster:Cluster" by extracting the service identifier.
//
// Returns:
//   - The normalized resource type string (e.g., "ec2", "eks", "ebs")
//   - Empty string if the original was empty or could not be normalized
func (r *serviceResolver) NormalizedType() string {
	r.ensureInitialized()
	return r.normalizedType
}

// ServiceType returns the detected service type string.
//
// On first call, this method ensures the resolver is initialized by computing
// both normalizedType and serviceType. Subsequent calls return the cached value.
//
// The service type is used for routing cost estimation logic to the appropriate
// service handler (EC2, EBS, RDS, EKS, Lambda, etc.).
//
// Returns:
//   - The detected service type (e.g., "ec2", "ebs", "rds", "eks")
//   - Empty string if the service could not be detected
func (r *serviceResolver) ServiceType() string {
	r.ensureInitialized()
	return r.serviceType
}

// ensureInitialized performs lazy initialization of the resolver.
//
// This method is idempotent - calling it multiple times has no additional effect
// after the first call. It computes:
//  1. normalizedType = normalizeResourceType(original)
//  2. serviceType = detectService(normalizedType)
//
// The computation happens exactly once per resolver instance.
func (r *serviceResolver) ensureInitialized() {
	if r.initialized {
		return
	}
	r.normalizedType = normalizeResourceType(r.original)
	r.serviceType = detectService(r.normalizedType)
	r.initialized = true
}
