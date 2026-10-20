package networks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// FindSubnetSourceRecord supplies only create_subnet's fixed default network
// lookup. A nonnull tenant ID remains a raw member query, including false/0/"".
// It does not promote the legacy general FindRecord's raw-control parity.
func (a *API) FindSubnetSourceRecord(ctx context.Context, identity string, tenantID json.RawMessage) (*NetworkRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("FindSubnetSourceRecord", "networks", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("configured networks API required"))
	}
	selected := a.client
	profile := recordFindProfile
	profile.Body = slices.Clone(profile.Body)
	profile.Headers = slices.Clone(profile.Headers)
	profile.Query = slices.Clone(profile.Query)
	profile.SourceFindPolicy, profile.SourceStatusPolicy, profile.SourceLinkPolicy, profile.PythonLoadedEquality, profile.StopFaultRetryCallbacks = true, true, true, true, true
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected networks client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	policy := neutronfind.Options{}
	if tenantID != nil && !bytes.Equal(bytes.TrimSpace(tenantID), []byte("null")) {
		policy.Query = map[string]json.RawMessage{"tenant_id": bytes.Clone(tenantID)}
	}
	value, err := engine.Find(ctx, identity, policy)
	if value != nil {
		return &NetworkRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}, wrap(err)
	}
	var physical *resource.ResponseError
	if errors.As(err, &physical) && physical != nil {
		return &NetworkRecord{Envelope: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode}, wrap(err)
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) {
		return &NetworkRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}, wrap(err)
	}
	return nil, wrap(err)
}
