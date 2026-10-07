package image

import (
	"context"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// CreateTask sends one asynchronous task request. Nil input becomes an owned
// empty object; task-specific requirements and execution remain server-owned.
func (s *Service) CreateTask(ctx context.Context, taskType string, options ...CreateTaskOption) (*TaskInfo, error) {
	p, err := s.captureTaskSource(ctx)
	if err == nil && (taskType == "" || !utf8.ValidString(taskType)) {
		err = uploadInvalid("task type must be nonempty valid UTF-8")
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CreateTask", err)
	}
	policy, err := parseCreateTaskOptions(append([]CreateTaskOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrapImageMutationError(ctx, "CreateTask", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodPost, p.base+"tasks", map[string]any{"type": taskType, "input": policy.Input}, nil, http.StatusCreated)
	return decodeTaskInfoResponse(ctx, p, "CreateTask", response, err)
}

// GetTask fetches one literal ID without Name lookup, missing suppression or a
// schema/status gate. Returned identifiers and links are passive response data.
func (s *Service) GetTask(ctx context.Context, taskID string, options ...GetTaskOption) (*TaskInfo, error) {
	p, err := s.captureTaskSource(ctx)
	if err == nil {
		err = validateTaskID(taskID)
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetTask", err)
	}
	policy, err := parseGetTaskOptions(append([]GetTaskOption(nil), options...))
	if err = p.finish(ctx, policy.Headers, err); err != nil {
		return nil, wrapImageMutationError(ctx, "GetTask", err)
	}
	response, err := rest.DoJSON(ctx, p.client, http.MethodGet, p.base+"tasks/"+url.PathEscape(taskID), nil, nil, http.StatusOK)
	return decodeTaskInfoResponse(ctx, p, "GetTask", response, err)
}
