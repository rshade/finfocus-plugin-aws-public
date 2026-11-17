package plugin

import (
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// recommendationIDNamespace is the UUIDv5 namespace for recommendation ids.
// The value is fixed so the same recommendation keeps the same id across processes.
var recommendationIDNamespace = uuid.MustParse("8f1c0c2e-4a6b-5d7e-9c1a-2b3d4e5f6071")

// recommendationIdentity is the resource key for a stable recommendation id.
// Priority matches correlation: descriptor id, then the resource_id tag, then the ARN.
// An empty result means the resource is anonymous and the id stays random.
func recommendationIdentity(resource *pbc.ResourceDescriptor) string {
	if resource == nil {
		return ""
	}
	if id := strings.TrimSpace(resource.GetId()); id != "" {
		return "id:" + id
	}
	if id := strings.TrimSpace(descriptorTags(resource)["resource_id"]); id != "" {
		return "tag:" + id
	}
	if arn := strings.TrimSpace(resource.GetArn()); arn != "" {
		return "arn:" + arn
	}
	return ""
}

// assignStableRecommendationID sets a name-based UUID when the resource has an
// identity. Prices and savings are not part of the key. A resource with no
// identity keeps the id the constructor minted, which is random, so two
// anonymous resources do not collide.
func assignStableRecommendationID(resource *pbc.ResourceDescriptor, rec *pbc.Recommendation) {
	if rec == nil {
		return
	}
	identity := recommendationIdentity(resource)
	if identity == "" {
		if rec.GetId() == "" {
			rec.Id = uuid.New().String()
		}
		return
	}
	rec.Id = uuid.NewSHA1(recommendationIDNamespace, []byte(recommendationIDKey(identity, rec))).String()
}

func recommendationIDKey(identity string, rec *pbc.Recommendation) string {
	var b strings.Builder
	b.WriteString(identity)
	b.WriteByte('\n')
	if info := rec.GetResource(); info != nil {
		b.WriteString(info.GetResourceType())
		b.WriteByte('\n')
		b.WriteString(info.GetRegion())
		b.WriteByte('\n')
		b.WriteString(info.GetSku())
	}
	b.WriteByte('\n')
	if modify := rec.GetModify(); modify != nil {
		b.WriteString(modify.GetModificationType())
		b.WriteByte('\n')
		writeSortedConfig(&b, modify.GetRecommendedConfig())
		return b.String()
	}
	b.WriteString(rec.GetActionType().String())
	return b.String()
}

func writeSortedConfig(b *strings.Builder, config map[string]string) {
	for _, key := range slices.Sorted(maps.Keys(config)) {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(config[key])
		b.WriteByte('\n')
	}
}
