package introspection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// unprocessedDataVersion is the inspector API version the pinned Python
// openstacksdk pins for the unprocessed-data route.
const unprocessedDataVersion = "1.17"

// StartedError reports that the inspector acknowledged the start request but
// the follow-up status read failed. The introspection may be running.
type StartedError struct {
	NodeID string
	Cause  error
}

func (e *StartedError) Error() string {
	return fmt.Sprintf("introspection of node %q was accepted but reading its status failed: %v", e.NodeID, e.Cause)
}

func (e *StartedError) Unwrap() error { return e.Cause }

// StartAndGet starts introspection and then reads its status once, matching
// the two requests behind start_introspection (POST then GET). The start is
// never replayed. If only the status read fails the error is a *StartedError.
// The node argument is used as the URL identity without a lookup; a name works
// only if the inspector accepts it.
func (a *API) StartAndGet(ctx context.Context, nodeID string, opts StartOpts, options ...StartIntrospectionOption) (*Introspection, error) {
	if nodeID == "" {
		return nil, request.Wrap("StartAndGet", "introspection", fmt.Errorf("%w: node identity must not be empty", resource.ErrInvalidOption))
	}
	if err := a.StartIntrospection(ctx, nodeID, opts, options...); err != nil {
		return nil, err
	}
	value, err := a.GetIntrospectionStatus(ctx, nodeID)
	if err != nil {
		return nil, &StartedError{NodeID: nodeID, Cause: err}
	}
	return value, nil
}

type abortConfig struct{ ignoreMissing bool }

// AbortOption customizes AbortIntrospectionWithOptions.
type AbortOption func(*abortConfig)

// WithAbortIgnoreMissing sets whether a 404 answer is treated as success. The
// default is true, like abort_introspection(ignore_missing=True).
func WithAbortIgnoreMissing(ignore bool) AbortOption {
	return func(c *abortConfig) { c.ignoreMissing = ignore }
}

// AbortIntrospectionWithOptions aborts like AbortIntrospection but ignores a
// missing introspection unless WithAbortIgnoreMissing(false) is given. The
// acknowledgement is not proof that the introspection stopped; wait with
// WaitUntilFinishedIgnoringError.
func (a *API) AbortIntrospectionWithOptions(ctx context.Context, nodeID string, options ...AbortOption) error {
	cfg := abortConfig{ignoreMissing: true}
	for _, option := range options {
		if option == nil {
			return request.Wrap("AbortIntrospection", "introspection", fmt.Errorf("%w: nil abort option", resource.ErrInvalidOption))
		}
		option(&cfg)
	}
	err := a.AbortIntrospection(ctx, nodeID)
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if cfg.ignoreMissing && errors.As(err, &unexpected) && unexpected.Actual == http.StatusNotFound {
		return nil
	}
	return err
}

// GetIntrospectionDataRaw returns the exact JSON document of the processed
// (default Python behavior) or unprocessed introspection data. The format is
// not stable and unknown fields and large numbers are preserved. The
// unprocessed route is requested with inspector API version 1.17 whatever the
// client's version is, as in the pinned source.
func (a *API) GetIntrospectionDataRaw(ctx context.Context, nodeID string, processed bool) (json.RawMessage, error) {
	if nodeID == "" {
		return nil, request.Wrap("GetIntrospectionData", "introspection", fmt.Errorf("%w: node identity must not be empty", resource.ErrInvalidOption))
	}
	url := a.client.ServiceURL("introspection", nodeID, "data")
	opts := &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}, KeepResponseBody: true}
	client := a.client
	if !processed {
		url += "/unprocessed"
		// The Gophercloud client owns the version header, so pin the version
		// on a shallow copy and leave the shared client untouched.
		pinned := *a.client
		pinned.Microversion = unprocessedDataVersion
		client = &pinned
	}
	resp, err := client.Get(ctx, url, nil, opts)
	if err != nil {
		return nil, request.Wrap("GetIntrospectionData", "introspection", err)
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, request.Wrap("GetIntrospectionData", "introspection", err)
	}
	return raw, nil
}

// WaitUntilFinishedIgnoringError polls until Finished is true and, unlike
// WaitUntilFinished, treats the "error" state as an ordinary final result,
// like wait_for_introspection(ignore_error=True). Use it after an abort.
func (a *API) WaitUntilFinishedIgnoringError(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) (*Introspection, error) {
	if err := resource.ValidateWaitOptions(options...); err != nil {
		return nil, request.Wrap("WaitUntilFinished", "introspection", err)
	}
	id, err := a.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("WaitUntilFinished", "introspection", err)
	}
	waiter := resource.NewCollection(resource.Adapter[Introspection]{
		Kind:            "introspection",
		FixedWaitStatus: true,
		Get:             func(ctx context.Context, _ string) (*Introspection, error) { return a.GetIntrospectionStatus(ctx, id) },
		ID:              func(*Introspection) string { return id },
		Status: func(value *Introspection) string {
			if value.Finished {
				return "finished"
			}
			return "pending"
		},
	})
	value, err := waiter.Wait(ctx, resource.ID(id), "finished", options...)
	return value, request.Wrap("WaitUntilFinished", "introspection", err)
}
