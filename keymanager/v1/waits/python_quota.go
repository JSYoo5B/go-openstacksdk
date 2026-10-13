package waits

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const quotaKind = "quotas"
const effectiveQuotaSelector = "current-project"

// Quota owns GET /quotas for the provider token's current project. The internal
// current-project selector is not a wire member/project ID. Resource.id remains
// null unless present in actual metadata; it is never appended to the route.
// The model has no status descriptor; select a declared string/null attribute.
func Quota() Target {
	view, err := projectACLQuota(quotaKind, nil, nil)
	return Target{kind: quotaKind, id: effectiveQuotaSelector, seed: &Record{Kind: quotaKind, RequestID: effectiveQuotaSelector, Resource: view}, err: err}
}

// ForQuota snapshots actual effective quota fields and their response proof.
// Native Body is required, nullable raw descriptors retain their original
// numbers/types, and no unavailable byte Envelope is synthesized.
func ForQuota(value *quotas.Quota) Target {
	if value == nil || value.Body == nil {
		return Target{err: invalid("an effective Quota with original Body is required")}
	}
	wire := (&resource.RawResource{Metadata: value.Metadata}).Clone()
	view, err := projectACLQuota(quotaKind, aclQuotaFields(quotaKind, wire.Body), wire)
	if err != nil {
		return Target{err: err}
	}
	return fromRecord(quotaKind, effectiveQuotaSelector, &Record{Resource: view, Wire: wire, Header: wire.Header.Clone(), StatusCode: wire.StatusCode})
}
func quotaAttribute(attribute string) bool {
	return attribute == "location" || aclQuotaDeclared(quotaKind, attribute)
}
func fetchQuota(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, id string, previous *Record) (*Record, error) {
	return fetchACLQuota(ctx, source, guard, quotaKind, id, source.Client.ServiceURL("quotas"), "quotas", previous)
}
