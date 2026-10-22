package stackcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/templateloader"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ErrDeleteFailed is wrapped when delete_stack's post-wait lookup finds the
// stack in DELETE_FAILED.
var ErrDeleteFailed = errors.New("stack deletion failed")

type options struct {
	interval time.Duration
	loader   *templateloader.Loader
}

// Option configures the stack workflows.
type Option func(*options) error

// WithPollInterval replaces poll_for_events' five-second poll period. Zero
// polls without sleeping; negative values are rejected.
func WithPollInterval(interval time.Duration) Option {
	return func(o *options) error {
		if interval < 0 {
			return fmt.Errorf("%w: poll interval must not be negative", resource.ErrInvalidOption)
		}
		o.interval = interval
		return nil
	}
}

// WithTemplateLoader selects the loader for create and update. The default
// templateloader.New() has no object reader, like the Python proxy.
func WithTemplateLoader(loader *templateloader.Loader) Option {
	return func(o *options) error {
		if loader == nil {
			return fmt.Errorf("%w: template loader is required", resource.ErrInvalidOption)
		}
		o.loader = loader
		return nil
	}
}

func apply(ctx context.Context, client *gophercloud.ServiceClient, list []Option) (options, error) {
	config := options{interval: DefaultPollInterval}
	if ctx == nil {
		return config, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if client == nil {
		return config, fmt.Errorf("%w: orchestration client is required", resource.ErrInvalidOption)
	}
	for _, option := range list {
		if option == nil {
			return config, fmt.Errorf("%w: nil stack workflow option", resource.ErrInvalidOption)
		}
		if err := option(&config); err != nil {
			return config, err
		}
	}
	return config, ctx.Err()
}

// SearchRequest holds search_stacks' arguments. NameOrID is matched against
// the view's id and name by equality or fnmatch glob (empty skips the phase).
// Filters is nil, a JSON object of (nested) exact matches, or a JSON string
// holding a JMESPath expression.
type SearchRequest struct {
	NameOrID string
	Filters  *json.RawMessage
}

// SearchResult carries the matching rows and the JSON view the filters ran
// over. For a JMESPath expression, Value is the arbitrary expression output,
// Expression is true and Stacks is empty.
type SearchResult struct {
	Value      json.RawMessage
	Stacks     []*stacks.StackRow
	Inventory  []*stacks.StackRow
	Expression bool
}

// SearchStacks ports search_stacks: list every stack (GET /stacks with no
// query, following pages), then apply _utils._filter_list locally to the
// Python Stack.to_dict view of each row (see StackView).
func SearchStacks(ctx context.Context, client *gophercloud.ServiceClient, input SearchRequest, opts ...Option) (*SearchResult, error) {
	wrap := func(err error) error { return request.Wrap("search_stacks", "stacks", err) }
	if _, err := apply(ctx, client, opts); err != nil {
		return nil, wrap(err)
	}
	rows, err := stacks.New(client).Resources().AllRows(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	result := &SearchResult{Inventory: rows}
	views := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		if views[i], err = StackView(row.Raw); err != nil {
			return result, wrap(err)
		}
	}
	selected, err := cloudfilter.SelectSource(views, input.NameOrID, input.Filters, ctx.Err)
	if err != nil {
		return result, wrap(fmt.Errorf("%w: local stack filters: %w", resource.ErrInvalidOption, err))
	}
	result.Value, result.Expression = bytes.Clone(selected.Value), selected.Expression
	if !selected.Expression {
		for _, index := range selected.Indices {
			result.Stacks = append(result.Stacks, rows[index])
		}
	}
	return result, nil
}

// Acknowledgement is the accepted HTTP response of a stack mutation. It proves
// only that Heat accepted the request, not that the action finished.
type Acknowledgement struct {
	StatusCode int
	Header     http.Header
	Body       json.RawMessage
}

func acknowledge(response *http.Response, body json.RawMessage) *Acknowledgement {
	return &Acknowledgement{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: bytes.Clone(body)}
}

// DeleteRequest holds delete_stack's arguments.
type DeleteRequest struct {
	NameOrID string
	Wait     bool
}

// DeleteResult records every phase of delete_stack. Deleted is Python's
// return value: false only when the initial lookup found no stack.
type DeleteResult struct {
	Deleted bool
	// Stack is the initial get_stack(resolve_outputs=False) result.
	Stack *stacks.StackDetail
	// Marker is the id of the latest event captured before deleting (Wait
	// only); nil when the stack had no events.
	Marker json.RawMessage
	// Deletion is the DELETE /stacks/{id} acknowledgement. NotFound is true
	// when Heat answered 404, which delete_stack ignores.
	Deletion *Acknowledgement
	NotFound bool
	// WaitStatus is the terminal status reported by event polling, empty when
	// polling ended with an HTTP error, which delete_stack ignores; that
	// error is kept in WaitHTTPError.
	WaitStatus    string
	WaitHTTPError error
	// Final is the post-wait get_stack(resolve_outputs=False) result, nil
	// when the stack is gone.
	Final *stacks.StackDetail
}

func isHTTPError(err error) bool {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &unexpected)
}

// DeleteStack ports delete_stack. It looks the stack up once and returns
// Deleted=false without a mutation when it is absent. With Wait it captures
// the latest event marker, sends one DELETE /stacks/{id} (a 404 is ignored),
// polls DELETE events from the marker (HTTP errors end polling silently, as
// in Python), and fails with ErrDeleteFailed when a final lookup shows
// DELETE_FAILED. The DELETE is never replayed; on a later error the result
// keeps the acknowledgement.
func DeleteStack(ctx context.Context, client *gophercloud.ServiceClient, input DeleteRequest, opts ...Option) (*DeleteResult, error) {
	wrap := func(err error) error { return request.Wrap("delete_stack", "stacks", err) }
	config, err := apply(ctx, client, opts)
	if err != nil {
		return nil, wrap(err)
	}
	result := &DeleteResult{}
	stack, err := getStack(ctx, client, input.NameOrID, false)
	if err != nil || stack == nil {
		return result, wrap(err)
	}
	result.Stack = stack
	if input.Wait {
		if result.Marker, err = lastEventMarker(ctx, client, input.NameOrID); err != nil {
			return result, wrap(err)
		}
	}
	if err := ctx.Err(); err != nil {
		return result, wrap(err)
	}
	endpoint := client.ServiceURL("stacks", url.PathEscape(stack.ID))
	body, response, err := exchange(ctx, client, http.MethodDelete, endpoint, nil)
	switch {
	case err == nil:
		result.Deletion = acknowledge(response, body)
	case gophercloud.ResponseCodeIs(err, http.StatusNotFound):
		result.NotFound = true
		if response != nil {
			result.Deletion = &Acknowledgement{StatusCode: response.StatusCode, Header: response.Header.Clone()}
		}
	default:
		return result, wrap(err)
	}
	result.Deleted = true
	if !input.Wait {
		return result, nil
	}
	status, err := pollForEvents(ctx, client, input.NameOrID, "DELETE", result.Marker, config.interval)
	if err != nil {
		if !isHTTPError(err) {
			return result, wrap(err)
		}
		result.WaitHTTPError = err
	}
	result.WaitStatus = status
	final, err := getStack(ctx, client, input.NameOrID, false)
	if err != nil {
		return result, wrap(err)
	}
	result.Final = final
	if final != nil && final.Status == "DELETE_FAILED" {
		rawReason, ok := final.Raw.Body["stack_status_reason"]
		if !ok {
			rawReason = json.RawMessage("null")
		}
		reason, err := cloudfilter.PythonString(rawReason)
		if err != nil {
			reason = string(rawReason)
		}
		return result, wrap(fmt.Errorf("%w: Failed to delete stack %s: %s", ErrDeleteFailed, input.NameOrID, reason))
	}
	return result, nil
}

// StackRequest holds the arguments shared by create_stack and update_stack.
type StackRequest struct {
	// Tags is sent as given; nil sends "tags": null like Python's tags=None.
	Tags []string
	// Template selects the template, files and environment files that
	// templateloader.ReadEnvAndTemplates loads.
	Template templateloader.StackInput
	// Rollback defaults to true and is sent as disable_rollback = !Rollback.
	Rollback *bool
	// TimeoutSeconds defaults to 3600 and is sent as timeout_mins with
	// Python floor division by 60.
	TimeoutSeconds *int64
	// Parameters are Python's **parameters; nil sends {}.
	Parameters map[string]any
	// Wait polls stack events until the action completes or fails.
	Wait bool
}

func floorDiv(value, divisor int64) int64 {
	quotient := value / divisor
	if value%divisor != 0 && (value < 0) != (divisor < 0) {
		quotient--
	}
	return quotient
}

// body builds the create/update attributes: tags, disable_rollback,
// timeout_mins, parameters, then template, files and environment only when
// the loader produced them.
func (r StackRequest) body(ctx context.Context, loader *templateloader.Loader) (map[string]any, *templateloader.StackAttributes, error) {
	rollback := true
	if r.Rollback != nil {
		rollback = *r.Rollback
	}
	timeout := int64(3600)
	if r.TimeoutSeconds != nil {
		timeout = *r.TimeoutSeconds
	}
	parameters := r.Parameters
	if parameters == nil {
		parameters = map[string]any{}
	}
	var tags any
	if r.Tags != nil {
		tags = append([]string{}, r.Tags...)
	}
	body := map[string]any{
		"tags":             tags,
		"disable_rollback": !rollback,
		"timeout_mins":     floorDiv(timeout, 60),
		"parameters":       parameters,
	}
	if loader == nil {
		var err error
		if loader, err = templateloader.New(); err != nil {
			return nil, nil, err
		}
	}
	attrs, err := loader.ReadEnvAndTemplates(ctx, r.Template)
	if err != nil {
		return nil, nil, err
	}
	if attrs.Template != nil {
		body["template"] = attrs.Template
	}
	if attrs.Files != nil {
		body["files"] = attrs.Files
	}
	if attrs.Environment != nil {
		body["environment"] = attrs.Environment
	}
	return body, attrs, nil
}

// checkResponseBody follows Resource._translate_response for an accepted
// create or update: unparsable JSON is ignored, but valid JSON that is not an
// object, or whose stack member is not an object, fails.
func checkResponseBody(body json.RawMessage) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return nil
	}
	var document map[string]json.RawMessage
	if err := decodeObject(trimmed, &document); err != nil {
		return fmt.Errorf("stack response: %w", err)
	}
	if member, ok := document["stack"]; ok {
		var stack map[string]json.RawMessage
		if err := decodeObject(member, &stack); err != nil {
			return fmt.Errorf("stack response stack: %w", err)
		}
	}
	return nil
}

// CreateRequest holds create_stack's arguments.
type CreateRequest struct {
	Name string
	StackRequest
}

// MutationResult records every phase of create_stack and update_stack.
type MutationResult struct {
	// Attributes are the loaded template, files and environment.
	Attributes *templateloader.StackAttributes
	// Marker is update_stack's latest-event marker (Wait only).
	Marker json.RawMessage
	// Accepted is the POST or PUT acknowledgement; nil before it is sent.
	Accepted *Acknowledgement
	// WaitStatus is the terminal action_COMPLETE or action_FAILED status from
	// event polling. A failed status is not an error, as in Python; callers
	// inspect WaitStatus or Stack.
	WaitStatus string
	// Stack is the final get_stack result, nil when the stack is not found.
	Stack *stacks.StackDetail
}

// CreateStack ports create_stack: load the template and environments, POST
// /stacks once, poll CREATE events when Wait is set, and return the detailed
// stack from get_stack(name). If a later phase fails, the result keeps the
// creation acknowledgement and the error; the POST is never replayed and a
// canceled wait does not undo the remote creation.
func CreateStack(ctx context.Context, client *gophercloud.ServiceClient, input CreateRequest, opts ...Option) (*MutationResult, error) {
	wrap := func(err error) error { return request.Wrap("create_stack", "stacks", err) }
	config, err := apply(ctx, client, opts)
	if err != nil {
		return nil, wrap(err)
	}
	if err := (resource.ID(input.Name)).Validate(); err != nil {
		return nil, wrap(err)
	}
	body, attrs, err := input.body(ctx, config.loader)
	if err != nil {
		return nil, wrap(err)
	}
	body["stack_name"] = input.Name
	result := &MutationResult{Attributes: attrs}
	if err := ctx.Err(); err != nil {
		return result, wrap(err)
	}
	raw, response, err := exchange(ctx, client, http.MethodPost, client.ServiceURL("stacks"), body)
	if err != nil {
		return result, wrap(err)
	}
	result.Accepted = acknowledge(response, raw)
	if err := checkResponseBody(raw); err != nil {
		return result, wrap(err)
	}
	if input.Wait {
		if result.WaitStatus, err = pollForEvents(ctx, client, input.Name, "CREATE", nil, config.interval); err != nil {
			return result, wrap(err)
		}
	}
	result.Stack, err = getStack(ctx, client, input.Name, true)
	return result, wrap(err)
}

// UpdateRequest holds update_stack's arguments.
type UpdateRequest struct {
	NameOrID string
	StackRequest
}

// UpdateStack ports update_stack: load the template and environments, capture
// the latest event marker when Wait is set, PUT /stacks/{name_or_id} once,
// poll UPDATE events from the marker, and return get_stack(name_or_id). The
// PUT is never replayed; on a later error the result keeps the acknowledgement.
func UpdateStack(ctx context.Context, client *gophercloud.ServiceClient, input UpdateRequest, opts ...Option) (*MutationResult, error) {
	wrap := func(err error) error { return request.Wrap("update_stack", "stacks", err) }
	config, err := apply(ctx, client, opts)
	if err != nil {
		return nil, wrap(err)
	}
	if err := (resource.ID(input.NameOrID)).Validate(); err != nil {
		return nil, wrap(err)
	}
	body, attrs, err := input.body(ctx, config.loader)
	if err != nil {
		return nil, wrap(err)
	}
	result := &MutationResult{Attributes: attrs}
	if input.Wait {
		if result.Marker, err = lastEventMarker(ctx, client, input.NameOrID); err != nil {
			return result, wrap(err)
		}
	}
	if err := ctx.Err(); err != nil {
		return result, wrap(err)
	}
	endpoint := client.ServiceURL("stacks", url.PathEscape(input.NameOrID))
	raw, response, err := exchange(ctx, client, http.MethodPut, endpoint, body)
	if err != nil {
		return result, wrap(err)
	}
	result.Accepted = acknowledge(response, raw)
	if err := checkResponseBody(raw); err != nil {
		return result, wrap(err)
	}
	if input.Wait {
		if result.WaitStatus, err = pollForEvents(ctx, client, input.NameOrID, "UPDATE", result.Marker, config.interval); err != nil {
			return result, wrap(err)
		}
	}
	result.Stack, err = getStack(ctx, client, input.NameOrID, true)
	return result, wrap(err)
}
