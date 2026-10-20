package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaResource retains the request target separately from the response fields.
// Native integer models represent null as zero; Body distinguishes those values.
type QuotaResource struct {
	Quota
	ProjectID  string
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// DefaultQuotaResource is deployment-wide and has no invented project identity.
type DefaultQuotaResource struct {
	Quota
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// QuotaResponseError retains successful HTTP evidence when decoding fails.
// It unwraps to the original JSON, body-read or context error.
type QuotaResponseError struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Cause      error
}

func (e *QuotaResponseError) Error() string {
	return fmt.Sprintf("quota response HTTP %d: %v", e.StatusCode, e.Cause)
}
func (e *QuotaResponseError) Unwrap() error { return e.Cause }

type quotaWire struct {
	body   []byte
	header http.Header
	status int
}

func (w quotaWire) fail(err error) error {
	return &QuotaResponseError{Body: append([]byte(nil), w.body...), Header: w.header.Clone(), StatusCode: w.status, Cause: err}
}

func (a *API) requestQuota(ctx context.Context, method, endpoint string, body json.RawMessage, accepted []int) (quotaWire, error) {
	options := &gophercloud.RequestOpts{OkCodes: accepted, KeepResponseBody: true}
	if body != nil {
		options.JSONBody = body
	}
	response, err := a.client.Request(ctx, method, endpoint, options)
	if err != nil {
		return quotaWire{}, err
	}
	wire := quotaWire{header: response.Header.Clone(), status: response.StatusCode}
	defer response.Body.Close()
	wire.body, err = io.ReadAll(response.Body)
	if err != nil {
		return wire, wire.fail(err)
	}
	if err := ctx.Err(); err != nil {
		return wire, wire.fail(err)
	}
	return wire, nil
}

func decodeQuotaObject(body []byte) (Quota, map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return Quota{}, nil, err
	}
	if fields == nil {
		return Quota{}, nil, gophercloud.ErrUnexpectedType{Expected: "quota JSON object", Actual: "null"}
	}
	var quota Quota
	if err := json.Unmarshal(body, &quota); err != nil {
		return Quota{}, nil, err
	}
	return quota, fields, nil
}

func decodeQuota(wire quotaWire, projectID string) (*QuotaResource, error) {
	var envelope struct {
		Quota json.RawMessage `json:"quota"`
	}
	if err := json.Unmarshal(wire.body, &envelope); err != nil {
		return nil, wire.fail(err)
	}
	quota, fields, err := decodeQuotaObject(envelope.Quota)
	if err != nil {
		return nil, wire.fail(err)
	}
	return &QuotaResource{Quota: quota, ProjectID: projectID, Body: fields, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}

type getOptions struct{ fields []string }
type GetOption func(*getOptions) error

// WithGetFields requests repeated fields query parameters, copied at creation.
func WithGetFields(fields ...string) GetOption {
	fields = append([]string(nil), fields...)
	return func(options *getOptions) error {
		if err := validateFields(fields); err != nil {
			return err
		}
		options.fields = fields
		return nil
	}
}

func withFields(endpoint string, fields []string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	query := u.Query()
	for _, field := range fields {
		query.Add("fields", field)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (s *ProjectQuotaScope) Get(ctx context.Context, options ...GetOption) (*QuotaResource, error) {
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	var config getOptions
	for _, apply := range options {
		if apply == nil {
			return nil, quotaError("Get", s.projectID, fmt.Errorf("%w: nil get option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaError("Get", s.projectID, err)
		}
	}
	endpoint, err := withFields(s.api.quotaEndpoint(s.projectID), config.fields)
	if err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	wire, err := s.api.requestQuota(ctx, http.MethodGet, endpoint, nil, []int{200})
	if err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID)
	return value, quotaError("Get", s.projectID, err)
}

func (a *API) Defaults(ctx context.Context) (*DefaultQuotaResource, error) {
	if err := a.validateQuotaClient(ctx); err != nil {
		return nil, quotaError("Defaults", "", err)
	}
	wire, err := a.requestQuota(ctx, http.MethodGet, a.quotaEndpoint("defaults"), nil, []int{200})
	if err != nil {
		return nil, quotaError("Defaults", "", err)
	}
	value, err := decodeQuota(wire, "")
	if err != nil {
		return nil, quotaError("Defaults", "", err)
	}
	return &DefaultQuotaResource{Quota: value.Quota, Body: value.Body, Header: value.Header, StatusCode: value.StatusCode}, nil
}

// Defaults reads global deployment settings, not a project-specific endpoint.
func (s *ProjectQuotaScope) Defaults(ctx context.Context) (*DefaultQuotaResource, error) {
	return s.api.Defaults(ctx)
}
