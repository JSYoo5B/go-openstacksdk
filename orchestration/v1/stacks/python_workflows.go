package stacks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ActionAcknowledgement records that Heat accepted a stack action request. It
// is an acknowledgement only: the stack status must be observed separately, for
// example with StackScope.Wait.
type ActionAcknowledgement struct {
	// Action is the Heat action key that was sent: check, suspend or resume.
	Action     string
	StatusCode int
	Header     http.Header
}

var actionOKCodes = []int{200, 201, 202, 204}

func (s *StackScope) action(ctx context.Context, name string, value any) (*ActionAcknowledgement, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap(name, "stacks", err)
	}
	endpoint := s.api.client.ServiceURL("stacks", url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), "actions")
	response, err := s.api.client.Post(ctx, endpoint, map[string]any{name: value}, nil, &gophercloud.RequestOpts{OkCodes: actionOKCodes})
	if err != nil {
		return nil, request.Wrap(name, "stacks", err)
	}
	return &ActionAcknowledgement{Action: name, StatusCode: response.StatusCode, Header: response.Header.Clone()}, nil
}

// Check asks Heat to verify the stack's resources by POSTing {"check": ""} to
// the fixed name/ID actions endpoint. The call is an acknowledgement; it is
// never replayed and does not wait for CHECK_COMPLETE.
func (s *StackScope) Check(ctx context.Context) (*ActionAcknowledgement, error) {
	return s.action(ctx, "check", "")
}

// Suspend asks Heat to suspend the stack with {"suspend": null}. The call is an
// acknowledgement only.
func (s *StackScope) Suspend(ctx context.Context) (*ActionAcknowledgement, error) {
	return s.action(ctx, "suspend", nil)
}

// Resume asks Heat to resume the stack with {"resume": null}. The call is an
// acknowledgement only.
func (s *StackScope) Resume(ctx context.Context) (*ActionAcknowledgement, error) {
	return s.action(ctx, "resume", nil)
}

func (s *StackScope) document(ctx context.Context, operation, route string) (*resource.RawResource, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap(operation, "stacks", err)
	}
	endpoint := s.api.client.ServiceURL("stacks", url.PathEscape(s.identity.Name), url.PathEscape(s.identity.ID), route)
	var raw json.RawMessage
	response, err := s.api.client.Get(ctx, endpoint, &raw, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap(operation, "stacks", err)
	}
	var document resource.RawResource
	if err := document.UnmarshalJSON(raw); err != nil {
		return nil, request.Wrap(operation, "stacks", err)
	}
	document.Header = response.Header.Clone()
	document.StatusCode = response.StatusCode
	return &document, nil
}

// Export returns the stack export document (GET .../export). The complete JSON
// object, including unknown fields and exact numbers, stays in Body.
func (s *StackScope) Export(ctx context.Context) (*resource.RawResource, error) {
	return s.document(ctx, "export", "export")
}

// Environment returns the environment the stack was created or updated with.
func (s *StackScope) Environment(ctx context.Context) (*resource.RawResource, error) {
	return s.document(ctx, "environment", "environment")
}

// Files returns the map of file names to file contents used by the stack. A
// non-object body is rejected rather than returned as an arbitrary value.
func (s *StackScope) Files(ctx context.Context) (*resource.RawResource, error) {
	return s.document(ctx, "files", "files")
}

// Template returns the stack template as the original JSON object. Unlike
// stacktemplates.API.Get it does not re-encode or indent the document.
func (s *StackScope) Template(ctx context.Context) (*resource.RawResource, error) {
	return s.document(ctx, "template", "template")
}

// StackDetail is a detailed stack GET result with both the typed native model
// and the complete raw stack object, so fields that the native model does not
// declare (deletion_time, stack_owner, parent, files, environment, template,
// stack_user_project_id, files_container and others) remain available.
type StackDetail struct {
	StackResource
	Raw resource.RawResource
}

type findOptions struct {
	resolveOutputs bool
	ignoreMissing  bool
}

// FindOption controls API.FindNameOrID.
type FindOption func(*findOptions) error

// WithFindResolveOutputs sets Heat's resolve_outputs query. The default is true
// and the query is omitted; false appends resolve_outputs=False to the URL.
func WithFindResolveOutputs(value bool) FindOption {
	return func(o *findOptions) error { o.resolveOutputs = value; return nil }
}

// WithFindIgnoreMissing makes absence, including a stack in DELETE_COMPLETE or
// ADOPT_COMPLETE, return nil, nil instead of resource.ErrNotFound.
func WithFindIgnoreMissing() FindOption {
	return func(o *findOptions) error { o.ignoreMissing = true; return nil }
}

// FindNameOrID performs one GET /stacks/{nameOrID}, the endpoint that Heat
// redirects to the canonical name/ID location. A stack in DELETE_COMPLETE or
// ADOPT_COMPLETE is reported as missing. No list request or duplicate scan is
// made; Heat enforces name uniqueness. The envelope must contain a stack object
// with a complete canonical identity.
func (a *API) FindNameOrID(ctx context.Context, nameOrID string, options ...FindOption) (*StackDetail, error) {
	if err := ctx.Err(); err != nil {
		return nil, request.Wrap("find", "stacks", err)
	}
	config := findOptions{resolveOutputs: true}
	for _, apply := range options {
		if apply == nil {
			return nil, fmt.Errorf("%w: nil find option", resource.ErrInvalidOption)
		}
		if err := apply(&config); err != nil {
			return nil, err
		}
	}
	if err := validateSegment(nameOrID); err != nil {
		return nil, err
	}
	endpoint := a.client.ServiceURL("stacks", url.PathEscape(nameOrID))
	if !config.resolveOutputs {
		endpoint += "?resolve_outputs=False"
	}
	var envelope map[string]json.RawMessage
	_, err := a.client.Get(ctx, endpoint, &envelope, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return missingStack(nameOrID, config, err)
		}
		return nil, request.Wrap("find", "stacks", err)
	}
	object, ok := envelope["stack"]
	if !ok || strings.TrimSpace(string(object)) == "null" {
		return nil, request.Wrap("find", "stacks", fmt.Errorf("stack response is missing its stack object"))
	}
	detail := &StackDetail{}
	if err := detail.Raw.UnmarshalJSON(object); err != nil {
		return nil, request.Wrap("find", "stacks", err)
	}
	if err := json.Unmarshal(object, &detail.RetrievedStack); err != nil {
		return nil, request.Wrap("find", "stacks", err)
	}
	detail.Detailed = true
	if _, err := detail.Identity(); err != nil {
		return nil, request.Wrap("find", "stacks", err)
	}
	if status := strings.ToUpper(detail.Status); status == "DELETE_COMPLETE" || status == "ADOPT_COMPLETE" {
		return missingStack(nameOrID, config, nil)
	}
	return detail, nil
}

func missingStack(reference string, config findOptions, cause error) (*StackDetail, error) {
	if config.ignoreMissing {
		return nil, nil
	}
	return nil, &resource.NotFoundError{Resource: "stacks", Reference: reference, Cause: cause}
}
