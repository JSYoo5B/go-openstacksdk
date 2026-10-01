package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type listOptions struct {
	projectID string
	maxItems  int
}

// ListOption configures SDK-side consumption of the single Neutron quota
// collection. It does not add server query or pagination parameters.
type ListOption func(*listOptions) error

// WithListProjectID restricts yielded rows to this exact project ID locally.
// Neutron still returns its whole quota override collection in one response.
func WithListProjectID(id string) ListOption {
	return func(options *listOptions) error {
		if err := resource.ID(id).Validate(); err != nil {
			return err
		}
		options.projectID = id
		return nil
	}
}

// WithListMaxItems stops consumption after n matching rows. It does not limit
// the HTTP response or decode rows after the limit has been reached.
func WithListMaxItems(n int) ListOption {
	return func(options *listOptions) error {
		if n <= 0 {
			return fmt.Errorf("%w: local maximum items must be positive", resource.ErrInvalidOption)
		}
		options.maxItems = n
		return nil
	}
}

// ListProjects lazily fetches quota rows for projects with quota overrides.
// The server merges those overrides with default limits. Projects without an
// override do not appear; this is not a Keystone project enumeration.
//
// Neutron's quota controller returns one array and supports no list query or
// continuation. Each visited row is decoded before applying the local filter.
func (a *API) ListProjects(ctx context.Context, options ...ListOption) iter.Seq2[*QuotaResource, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*QuotaResource, error) bool) {
		fail := func(err error) { yield(nil, request.Wrap("ListProjects", "network quota", err)) }
		if err := a.validateQuotaClient(ctx); err != nil {
			fail(err)
			return
		}
		var config listOptions
		for _, apply := range options {
			if apply == nil {
				fail(fmt.Errorf("%w: nil list option", resource.ErrInvalidOption))
				return
			}
			if err := apply(&config); err != nil {
				fail(err)
				return
			}
		}
		if err := ctx.Err(); err != nil {
			fail(err)
			return
		}
		result, statusCode := a.getQuota(ctx, a.client.ServiceURL("quotas"))
		var envelope map[string]json.RawMessage
		if err := result.ExtractInto(&envelope); err != nil {
			fail(err)
			return
		}
		if err := ctx.Err(); err != nil {
			fail(err)
			return
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(envelope["quotas"], &rows); err != nil {
			fail(err)
			return
		}
		if rows == nil {
			fail(gophercloud.ErrUnexpectedType{Expected: "quotas JSON array", Actual: "null"})
			return
		}
		if quotaContinuation(envelope) {
			fail(fmt.Errorf("%w: Neutron quota collection returned an unsupported continuation", resource.ErrUnsupported))
			return
		}
		count := 0
		for _, row := range rows {
			if err := ctx.Err(); err != nil {
				fail(err)
				return
			}
			value, err := decodeProjectQuota(row, result.Header, statusCode)
			if err != nil {
				fail(err)
				return
			}
			if config.projectID != "" && value.ProjectID != config.projectID {
				continue
			}
			if !yield(value, nil) {
				return
			}
			count++
			if config.maxItems != 0 && count >= config.maxItems {
				return
			}
		}
		if err := ctx.Err(); err != nil {
			fail(err)
		}
	}
}

// AllProjects consumes ListProjects and returns an independently owned slice.
// A response, row or cancellation error returns no partial result slice.
func (a *API) AllProjects(ctx context.Context, options ...ListOption) ([]*QuotaResource, error) {
	values := make([]*QuotaResource, 0)
	for value, err := range a.ListProjects(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Only explicit next values in recognized continuation shapes constrain the
// single-page contract. Other response metadata and unknown links stay allowed.
func quotaContinuation(envelope map[string]json.RawMessage) bool {
	var links []json.RawMessage
	if json.Unmarshal(envelope["quotas_links"], &links) == nil {
		for _, raw := range links {
			var link struct {
				Rel  string `json:"rel"`
				Href string `json:"href"`
			}
			if json.Unmarshal(raw, &link) == nil && link.Rel == "next" && link.Href != "" {
				return true
			}
		}
	}
	var rootLinks struct {
		Next string `json:"next"`
	}
	return json.Unmarshal(envelope["links"], &rootLinks) == nil && rootLinks.Next != ""
}
