package image

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ListImageTasksOpts supplies ordinary headers and a local row cap. Zero
// MaxItems is unlimited; no limit or pagination query is sent to the server.
type ListImageTasksOpts struct {
	Headers  map[string]string
	MaxItems int
}

// ListImageTasksOption is one option of ListImageTasks:
// WithListImageTasksOpts, WithListImageTasksHeaders (merges), request.Header
// (merges one header), request.MaxItems or request.Custom over
// *ListImageTasksOpts. Other shared calls, including request.Headers, fail
// with resource.ErrInvalidOption before HTTP.
type ListImageTasksOption = request.ListCall

var listImageTasksCalls = calls.Spec[ListImageTasksOpts]{
	Header: calls.MergeHeader[ListImageTasksOpts](WithListImageTasksHeaders),
	Paging: calls.Paging[ListImageTasksOpts]{MaxItems: calls.Value(func(c *ListImageTasksOpts) *int { return &c.MaxItems })},
}

// WithListImageTasksOpts snapshots and replaces the entire configuration.
func WithListImageTasksOpts(value ListImageTasksOpts) ListImageTasksOption {
	snapshot := copyListImageTasksOpts(value)
	return request.Custom(func(config *ListImageTasksOpts) error { *config = copyListImageTasksOpts(snapshot); return nil })
}
func WithListImageTasksHeaders(value map[string]string) ListImageTasksOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *ListImageTasksOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}

func copyListImageTasksOpts(value ListImageTasksOpts) ListImageTasksOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	return value
}
func parseListImageTasksOptions(options []ListImageTasksOption) (ListImageTasksOpts, error) {
	value, err := applyTaskOptions(ListImageTasksOpts{}, calls.Translate(options, listImageTasksCalls), copyListImageTasksOpts)
	if err != nil {
		return value, err
	}
	if value.MaxItems < 0 {
		return value, uploadInvalid("image task max items must be non-negative")
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, err
}
