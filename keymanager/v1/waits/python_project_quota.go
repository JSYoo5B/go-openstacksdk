package waits

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
)

const projectQuotaKind = "project-quotas"

// ProjectQuota polls the fixed admin project override route. It declares no
// status: select a string/null quota, id or name attribute for status waiting.
func ProjectQuota(projectID string) Target {
	encoded, _ := json.Marshal(projectID)
	view, err := projectACLQuota(projectQuotaKind, map[string]json.RawMessage{"id": encoded}, nil)
	return Target{kind: projectQuotaKind, id: projectID, seed: &Record{Kind: projectQuotaKind, RequestID: projectID, Resource: view}, err: err}
}

// ForProjectQuota snapshots actual owned record state; RequestID, not any
// response ID/HREF, determines subsequent GETs. No receipt is fabricated.
func ForProjectQuota(value *quotas.ProjectQuotaRecord) Target {
	if value == nil || value.Resource == nil {
		return Target{err: invalid("a ProjectQuotaRecord with declared Resource is required")}
	}
	view, err := projectACLQuota(projectQuotaKind, aclQuotaFields(projectQuotaKind, value.Resource.Body), value.Wire)
	if err != nil {
		return Target{err: err}
	}
	view.Header, view.StatusCode = value.Header.Clone(), value.StatusCode
	return fromRecord(projectQuotaKind, value.RequestID, &Record{Resource: view, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode})
}
func projectQuotaAttribute(attribute string) bool {
	return attribute == "location" || aclQuotaDeclared(projectQuotaKind, attribute)
}
func fetchProjectQuota(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, id string, previous *Record) (*Record, error) {
	return fetchACLQuota(ctx, source, guard, projectQuotaKind, id, source.Client.ServiceURL("project-quotas", url.PathEscape(id)), "project_quotas", previous)
}
