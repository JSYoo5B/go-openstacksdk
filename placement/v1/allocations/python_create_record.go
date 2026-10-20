package allocations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateRecordOpts carries only request headers; the allocations body is the
// CreateRecord argument.
type CreateRecordOpts struct{}
type CreateRecordOption = request.Option[CreateRecordOpts]

func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return request.WithHeader[CreateRecordOpts](key, value)
}

// CreateRecordResult reports the accepted response; Python returns None.
type CreateRecordResult struct {
	Header     http.Header
	StatusCode int
}

// CreateRecord implements create_allocations: Allocation.set posts the
// caller's per-consumer objects unchanged to POST allocations, so keys such as
// mappings, a null consumer_generation or an empty allocations map stay inside
// each consumer. A nil map sends {}. Every status below 400 is accepted and the
// request uses the client microversion.
func (a *API) CreateRecord(ctx context.Context, allocations map[string]any, options ...CreateRecordOption) (*CreateRecordResult, error) {
	const op = "CreateRecord"
	fail := func(err error) (*CreateRecordResult, error) {
		return nil, request.Wrap(op, "allocations", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(err)
	}
	if a == nil {
		return fail(fmt.Errorf("%w: allocation API is required", resource.ErrInvalidOption))
	}
	if allocations == nil {
		allocations = map[string]any{}
	}
	// Snapshot the caller's body before any option or HTTP runs.
	body, err := json.Marshal(allocations)
	if err != nil {
		return fail(fmt.Errorf("%w: allocations: %v", resource.ErrInvalidOption, err))
	}
	source, err := cloudread.Capture(ctx, a.client, "placement")
	if err != nil {
		return fail(err)
	}
	config, err := cloudread.ApplyReadOptions(ctx, CreateRecordOpts{}, options, cloudread.OwnReadConfig[CreateRecordOpts], source.Guard)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	if err != nil {
		return fail(err)
	}
	codes := make([]int, 0, 200)
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	response, err := rest.DoJSONGuarded(ctx, &source.Client, source.Guard, http.MethodPost, source.Client.ServiceURL("allocations"), json.RawMessage(body), nil, codes...)
	if err != nil {
		return fail(err)
	}
	if err := source.Guard(ctx); err != nil {
		return fail(response.Fail(err))
	}
	return &CreateRecordResult{Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}
