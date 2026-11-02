package image

import (
	"encoding/json"
	"maps"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateTaskOpts supplies an input object and ordinary headers. Nil Input
// defaults to {}; raw values retain exact JSON numbers and explicit null.
// NullInput sends the top-level "input": null that Python's
// create_task(input=None) sends; it cannot be combined with a non-nil Input.
type CreateTaskOpts struct {
	Headers   map[string]string
	Input     map[string]json.RawMessage
	NullInput bool
}

// CreateTaskOption is one option of CreateTask: WithCreateTaskOpts,
// WithCreateTaskHeaders (merges), WithCreateTaskInput, request.Header (merges
// one header) or request.Custom over *CreateTaskOpts. Other shared calls,
// including request.Headers and request.Field, fail with
// resource.ErrInvalidOption before HTTP.
type CreateTaskOption = request.WriteCall

// GetTaskOpts supplies ordinary headers for one fixed task GET.
type GetTaskOpts struct{ Headers map[string]string }

// GetTaskOption is one option of GetTask: WithGetTaskOpts,
// WithGetTaskHeaders (merges), request.Header (merges one header) or
// request.Custom over *GetTaskOpts. Other shared calls, including
// request.Headers, fail with resource.ErrInvalidOption before HTTP.
type GetTaskOption = request.GetCall

// ListTasksOpts supplies six server query fields and local iteration controls.
// Empty strings omit queries; nil Limit omits, while an explicit zero is sent.
// MaxItems is local only and never changes the server's requested limit.
//
// BodyFilters, MarkerFallback and LinkHeader are explicit opt-ins for pinned
// Python Resource.list behaviors; see WithListTasksBodyFilter,
// WithListTasksMarkerFallback and WithListTasksLinkHeader. BodyFilters is keyed
// by Python Task attribute name and holds complete JSON values.
type ListTasksOpts struct {
	Headers                                map[string]string
	Limit                                  *int
	Marker, Type, Status, SortKey, SortDir string
	MaxItems                               int
	SinglePage                             bool
	BodyFilters                            map[string]json.RawMessage
	MarkerFallback                         bool
	LinkHeader                             bool
}

// ListTasksOption is one option of ListTasks and AllTasks: WithListTasksOpts,
// WithListTasksHeaders (merges), the typed filter, sort and Python paging
// options, request.Header (merges one header), request.Limit,
// request.Marker, request.MaxItems or request.Custom over *ListTasksOpts.
// Other shared calls, including request.Headers, request.Query and
// request.Paginated, fail with resource.ErrInvalidOption before HTTP.
type ListTasksOption = request.ListCall

var (
	createTaskCalls = calls.Spec[CreateTaskOpts]{Header: calls.MergeHeader[CreateTaskOpts](WithCreateTaskHeaders)}
	getTaskCalls    = calls.Spec[GetTaskOpts]{Header: calls.MergeHeader[GetTaskOpts](WithGetTaskHeaders)}
	listTasksCalls  = calls.Spec[ListTasksOpts]{
		Header: calls.MergeHeader[ListTasksOpts](WithListTasksHeaders),
		Paging: calls.Paging[ListTasksOpts]{
			Limit:    calls.Pointer(func(c *ListTasksOpts) **int { return &c.Limit }),
			Marker:   calls.Value(func(c *ListTasksOpts) *string { return &c.Marker }),
			MaxItems: calls.Value(func(c *ListTasksOpts) *int { return &c.MaxItems }),
		},
	}
)

func WithCreateTaskOpts(value CreateTaskOpts) CreateTaskOption {
	snapshot := copyCreateTaskOpts(value)
	return request.Custom(func(config *CreateTaskOpts) error { *config = copyCreateTaskOpts(snapshot); return nil })
}
func WithCreateTaskHeaders(value map[string]string) CreateTaskOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *CreateTaskOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}

// WithCreateTaskInput serializes and snapshots the supplied object immediately.
// Factory errors surface when the option is applied. Original top-level keys
// are checked before encoding/json can replace malformed UTF-8.
func WithCreateTaskInput(value map[string]any) CreateTaskOption {
	snapshot, err := marshalTaskInput(value)
	return request.Custom(func(config *CreateTaskOpts) error {
		if err != nil {
			return err
		}
		config.Input = copyTaskRawMap(snapshot)
		config.NullInput = false
		return nil
	})
}
func WithGetTaskOpts(value GetTaskOpts) GetTaskOption {
	snapshot := copyGetTaskOpts(value)
	return request.Custom(func(config *GetTaskOpts) error { *config = copyGetTaskOpts(snapshot); return nil })
}
func WithGetTaskHeaders(value map[string]string) GetTaskOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *GetTaskOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithListTasksOpts(value ListTasksOpts) ListTasksOption {
	snapshot := copyListTasksOpts(value)
	return request.Custom(func(config *ListTasksOpts) error { *config = copyListTasksOpts(snapshot); return nil })
}
func WithListTasksHeaders(value map[string]string) ListTasksOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *ListTasksOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithListTasksType(value string) ListTasksOption {
	return request.Custom(func(config *ListTasksOpts) error { config.Type = value; return nil })
}
func WithListTasksStatus(value string) ListTasksOption {
	return request.Custom(func(config *ListTasksOpts) error { config.Status = value; return nil })
}
func WithListTasksSortKey(value string) ListTasksOption {
	return request.Custom(func(config *ListTasksOpts) error { config.SortKey = value; return nil })
}
func WithListTasksSortDir(value string) ListTasksOption {
	return request.Custom(func(config *ListTasksOpts) error { config.SortDir = value; return nil })
}
func WithListTasksSinglePage(value bool) ListTasksOption {
	return request.Custom(func(config *ListTasksOpts) error { config.SinglePage = value; return nil })
}

func taskHeadersCopy(value map[string]string) map[string]string {
	owned := maps.Clone(value)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func copyTaskRawMap(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		owned[key] = append(json.RawMessage(nil), raw...)
	}
	return owned
}
func copyCreateTaskOpts(value CreateTaskOpts) CreateTaskOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.Input = copyTaskRawMap(value.Input)
	return value
}
func copyGetTaskOpts(value GetTaskOpts) GetTaskOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	return value
}
func copyListTasksOpts(value ListTasksOpts) ListTasksOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.BodyFilters = copyTaskRawMap(value.BodyFilters)
	if value.Limit != nil {
		snapshot := *value.Limit
		value.Limit = &snapshot
	}
	return value
}
func applyTaskOptions[T any, O ~func(*T) error](value T, options []O, copyValue func(T) T) (T, error) {
	value = copyValue(value)
	for _, apply := range options {
		if apply == nil {
			return value, uploadInvalid("nil task option")
		}
		candidate := copyValue(value)
		if err := apply(&candidate); err != nil {
			return value, err
		}
		value = copyValue(candidate)
	}
	return value, nil
}
func parseCreateTaskOptions(options []CreateTaskOption) (CreateTaskOpts, error) {
	value, err := applyTaskOptions(CreateTaskOpts{}, calls.Translate(options, createTaskCalls), copyCreateTaskOpts)
	if err != nil {
		return value, err
	}
	for key, raw := range value.Input {
		if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return value, uploadInvalid("task input requires valid UTF-8 keys and JSON values")
		}
	}
	if value.NullInput && value.Input != nil {
		return value, uploadInvalid("task input cannot be both null and an object")
	}
	if value.Input == nil && !value.NullInput {
		value.Input = make(map[string]json.RawMessage)
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, err
}
func parseGetTaskOptions(options []GetTaskOption) (GetTaskOpts, error) {
	value, err := applyTaskOptions(GetTaskOpts{}, calls.Translate(options, getTaskCalls), copyGetTaskOpts)
	if err == nil {
		value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	}
	return value, err
}
func parseListTasksOptions(options []ListTasksOption) (ListTasksOpts, url.Values, error) {
	value, err := applyTaskOptions(ListTasksOpts{}, calls.Translate(options, listTasksCalls), copyListTasksOpts)
	query := make(url.Values)
	if err != nil {
		return value, query, err
	}
	if value.MaxItems < 0 || value.Limit != nil && *value.Limit < 0 {
		return value, query, uploadInvalid("task limit and max items must be non-negative")
	}
	for key, text := range map[string]string{"marker": value.Marker, "type": value.Type, "status": value.Status, "sort_key": value.SortKey, "sort_dir": value.SortDir} {
		if err := taskQueryText(text); err != nil {
			return value, query, err
		}
		if text != "" {
			query.Set(key, text)
		}
	}
	if value.SortDir != "" && value.SortDir != "asc" && value.SortDir != "desc" {
		return value, query, uploadInvalid("task sort direction must be asc or desc")
	}
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	if err := validateTaskBodyFilters(value.BodyFilters); err != nil {
		return value, query, err
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, query, err
}
func marshalTaskInput(value map[string]any) (map[string]json.RawMessage, error) {
	for key := range value {
		if !utf8.ValidString(key) {
			return nil, uploadInvalid("task input key must be valid UTF-8")
		}
	}
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return schemaObject(encoded)
}
