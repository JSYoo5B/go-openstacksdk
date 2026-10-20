package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// CloudRecords binds a literal container and its source before workflow options
// run. Each exchange reuses the same sticky source guard and owned headers.
// It performs no discovery, absence handling, ACL classification or refresh.
type CloudRecords struct {
	prepared *preparedMetadata
	headers  map[string]string
}

func (a *API) PrepareCloudRecords(ctx context.Context, name string) (*CloudRecords, error) {
	p, err := a.captureMetadata(ctx, name)
	if err != nil {
		return nil, request.Wrap("PrepareCloudRecords", "containers", err)
	}
	return &CloudRecords{prepared: p, headers: map[string]string{}}, nil
}
func (c *CloudRecords) Check(ctx context.Context) error {
	if c == nil || c.prepared == nil {
		return metadataContextError(ctx, metadataInvalid("prepared container workflow is required"))
	}
	return c.prepared.check(ctx)
}
func (c *CloudRecords) SetHeaders(ctx context.Context, values map[string]string) error {
	if err := c.Check(ctx); err != nil {
		return err
	}
	owned, err := validateMetadataHeaders(values)
	if err != nil {
		return err
	}
	for key := range owned {
		if key == "X-Container-Read" {
			return metadataInvalid("X-Container-Read is owned by the container access policy")
		}
	}
	c.headers = owned
	return c.Check(ctx)
}
func (c *CloudRecords) Get(ctx context.Context) (*ContainerRecord, error) {
	return c.exchange(ctx, http.MethodHead, nil, nil, false)
}
func (c *CloudRecords) Create(ctx context.Context, public bool) (*ContainerRecord, error) {
	values := map[string]string{}
	seeds := map[string]json.RawMessage{}
	if public {
		values["X-Container-Read"] = ".r:*,.rlistings"
		seeds["read_ACL"] = json.RawMessage(`".r:*,.rlistings"`)
	}
	return c.exchange(ctx, http.MethodPut, values, seeds, true)
}
func (c *CloudRecords) SetAccess(ctx context.Context, acl string) (*ContainerRecord, error) {
	if !metadataFieldValue(acl) {
		return nil, metadataInvalid("invalid container ACL")
	}
	// BaseResource.set_metadata does not seed read_ACL with the requested value.
	// Only an actual POST response header can populate it in the returned Resource.
	return c.exchange(ctx, http.MethodPost, map[string]string{"X-Container-Read": acl}, nil, false)
}
func (c *CloudRecords) exchange(ctx context.Context, method string, values map[string]string, seedHeaders map[string]json.RawMessage, hasBody bool) (*ContainerRecord, error) {
	wrap := func(err error) error { return request.Wrap(method, "containers", metadataContextError(ctx, err)) }
	if err := c.Check(ctx); err != nil {
		return nil, wrap(err)
	}
	p := c.prepared
	client := *p.client
	client.MoreHeaders = cloneMetadataHeaders(p.client.MoreHeaders)
	owned := cloneMetadataHeaders(c.headers)
	for key, value := range owned {
		client.MoreHeaders[key] = value
	}
	for key, value := range values {
		owned[key] = value
		client.MoreHeaders[key] = value
	}
	accepted, rejected := recordStatuses()
	response, err := rest.DoJSONGuardedRejectionsHeaders(ctx, &client, p.check, method, p.target, nil, owned,
		rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	err = p.observe(ctx, response, err)
	if response == nil {
		response = cloudRejectedEvidence(err)
	}
	if response == nil {
		return nil, wrap(err)
	}
	if err != nil {
		return &ContainerRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, wrap(err)
	}
	name, _ := json.Marshal(p.name)
	seedBody := map[string]json.RawMessage{"name": name}
	projected := *response
	if !hasBody {
		projected.Body = nil
	}
	value, projectionErr := recordProjection(&projected, seedBody, seedHeaders)
	if value != nil {
		value.Envelope = bytes.Clone(response.Body)
	}
	if projectionErr = errors.Join(projectionErr, p.check(ctx)); projectionErr != nil {
		if value != nil {
			value.Resource = nil
		}
		return value, wrap(response.Fail(projectionErr))
	}
	return value, nil
}
func cloudRejectedEvidence(err error) *rest.Response {
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) {
		var renewal *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &renewal) || !errors.As(renewal.ErrOriginal, &native) {
			return nil
		}
	}
	return &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
}
