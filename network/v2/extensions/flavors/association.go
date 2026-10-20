package flavors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecords"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServiceProfileAssociation is the associate_flavor_with_service_profile
// result. Body is the whole response JSON that Python returns as a dict.
// ServiceProfileID is Body's service_profile.id when it is a string.
type ServiceProfileAssociation struct {
	ServiceProfileID *string
	Body             json.RawMessage
	Header           http.Header
	StatusCode       int
}

// associationProfile addresses flavors/{flavor_id}/service_profiles with the
// {"service_profile": {...}} envelope that Flavor's methods use.
func associationProfile(flavorID string) neutronfind.Profile {
	return neutronfind.Profile{Kind: "flavors", Path: "flavors/" + url.PathEscape(flavorID) + "/service_profiles",
		Singular: "service_profile", Plural: "service_profiles", IDKey: "id", NameKey: "name",
		PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}}}
}

func (a *API) association(ctx context.Context, op, method, flavorID, serviceProfileID string, options []AssociationOption) (*ownedrecords.Record, error) {
	t := a.target()
	if err := errors.Join(ownedrecords.ValidateID("flavor ID", flavorID), ownedrecords.ValidateID("service profile ID", serviceProfileID)); err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	engine, err := t.Engine(ctx, associationProfile(flavorID))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := ownedrecords.Apply(ctx, engine, AssociationOpts{}, ownedrecords.CopyAction, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	transport := neutronfind.Options{Microversion: policy.Microversion, Headers: policy.Headers}
	if method == http.MethodDelete {
		value, err := engine.Mutate(ctx, method, serviceProfileID, nil, nil, transport)
		return value, t.Wrap(ctx, op, err)
	}
	id, _ := json.Marshal(serviceProfileID)
	value, err := engine.Mutate(ctx, method, "", map[string]json.RawMessage{"id": id}, nil, transport)
	return value, t.Wrap(ctx, op, err)
}

// AssociateServiceProfile implements
// associate_flavor_with_service_profile(flavor, service_profile): POST
// flavors/{flavor_id}/service_profiles with
// {"service_profile": {"id": service_profile_id}}. Python sends it with
// raise_exc=False and returns resp.json() for any status; Go accepts only
// 200..399 and returns the HTTP error (with status and body) for 400 and
// above. A response that is not JSON fails as Python's resp.json() does.
func (a *API) AssociateServiceProfile(ctx context.Context, flavorID, serviceProfileID string, options ...AssociationOption) (*ServiceProfileAssociation, error) {
	const op = "AssociateServiceProfile"
	value, err := a.association(ctx, op, http.MethodPost, flavorID, serviceProfileID, options)
	if err != nil {
		return nil, err
	}
	if !json.Valid(value.Envelope) {
		cause := ownedrecords.Invalid("association response must contain JSON")
		return nil, a.target().Wrap(ctx, op, &resource.ResponseError{Body: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, Cause: cause})
	}
	result := &ServiceProfileAssociation{Body: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(value.Envelope, &envelope) == nil {
		var member map[string]json.RawMessage
		if json.Unmarshal(envelope["service_profile"], &member) == nil {
			var id string
			if json.Unmarshal(member["id"], &id) == nil && member["id"] != nil && string(bytes.TrimSpace(member["id"])) != "null" {
				result.ServiceProfileID = &id
			}
		}
	}
	return result, nil
}

// DisassociateServiceProfile implements
// disassociate_flavor_from_service_profile(flavor, service_profile): DELETE
// flavors/{flavor_id}/service_profiles/{service_profile_id}. Python ignores
// the response (raise_exc=False) and returns None; Go returns nil for
// 200..399 and the HTTP error for 400 and above, including 404.
func (a *API) DisassociateServiceProfile(ctx context.Context, flavorID, serviceProfileID string, options ...AssociationOption) error {
	_, err := a.association(ctx, "DisassociateServiceProfile", http.MethodDelete, flavorID, serviceProfileID, options)
	return err
}
