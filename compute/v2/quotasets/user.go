package quotasets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/identity/v3/users"
	"gophercloudsdk/internal/project"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// UserQuotaScope fixes both the project and user. It intentionally does not
// embed ProjectQuotaScope: user operations cannot promote a project-wide Reset.
type UserQuotaScope struct {
	api       *API
	projectID string
	userID    string
}

// InUser binds an explicit user ID without a lookup, or resolves an exact user
// name using the inherited Keystone client or a WithIdentityClient override.
func (s *ProjectQuotaScope) InUser(ctx context.Context, ref resource.Ref, options ...ProjectOption) (*UserQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, quotaUserError("InUser", "", ref.String(), err)
	}
	if s == nil {
		return nil, quotaUserError("InUser", "", ref.String(), fmt.Errorf("%w: project quota scope is required", resource.ErrInvalidOption))
	}
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return nil, quotaUserError("InUser", s.projectID, ref.String(), err)
	}
	if err := resource.ID(s.projectID).Validate(); err != nil {
		return nil, quotaUserError("InUser", s.projectID, ref.String(), err)
	}
	if err := ref.Validate(); err != nil {
		return nil, quotaUserError("InUser", s.projectID, ref.String(), err)
	}
	config := projectOptions{identity: s.identity}
	for _, apply := range options {
		if apply == nil {
			return nil, quotaUserError("InUser", s.projectID, ref.String(), fmt.Errorf("%w: nil user resolution option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, quotaUserError("InUser", s.projectID, ref.String(), err)
		}
	}
	id := ref.String()
	if ref.IsName() {
		if config.identity == nil {
			return nil, quotaUserError("InUser", s.projectID, id, fmt.Errorf("%w: user names require an Identity v3 client", resource.ErrUnsupported))
		}
		if err := project.ValidateIdentityClient(config.identity); err != nil {
			return nil, quotaUserError("InUser", s.projectID, id, err)
		}
		if config.identity == s.api.client {
			return nil, quotaUserError("InUser", s.projectID, id, fmt.Errorf("%w: the Compute client cannot resolve Keystone user names", resource.ErrInvalidOption))
		}
		var err error
		id, err = users.New(config.identity).Resources.ResolveID(ctx, ref)
		if err != nil {
			return nil, quotaUserError("InUser", s.projectID, ref.String(), err)
		}
	}
	return &UserQuotaScope{api: s.api, projectID: s.projectID, userID: id}, nil
}

// ProjectID returns the fixed project without an HTTP request.
func (s *UserQuotaScope) ProjectID() string {
	if s == nil {
		return ""
	}
	return s.projectID
}

// UserID returns the fixed user without an HTTP request.
func (s *UserQuotaScope) UserID() string {
	if s == nil {
		return ""
	}
	return s.userID
}

func (s *UserQuotaScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: user quota scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validateQuotaClient(ctx); err != nil {
		return err
	}
	if err := resource.ID(s.projectID).Validate(); err != nil {
		return err
	}
	// Never construct a user operation with an empty user_id. In particular,
	// a malformed or zero-valued scope must not reset all project quotas.
	return resource.ID(s.userID).Validate()
}

func quotaUserError(operation, projectID, userID string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "compute user quota", Reference: projectID + "/" + userID, Cause: err}
	}
	return request.Wrap(operation, "compute user quota", err)
}

func (s *UserQuotaScope) wrap(operation string, err error) error {
	return quotaUserError(operation, s.ProjectID(), s.UserID(), err)
}

func (s *UserQuotaScope) get(ctx context.Context, suffix string) (result gophercloud.Result) {
	endpoint := s.api.quotaURL(s.projectID, s.userID, suffix)
	client, err := guardedQuotaClient(s.api.client, http.MethodGet, endpoint)
	if err != nil {
		result.Err = err
		return
	}
	response, err := client.Get(ctx, endpoint, &result.Body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	return
}

// Get reads limits for this fixed project/user pair.
func (s *UserQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, s.wrap("Get", err)
	}
	value, err := decodeQuota(s.get(ctx, ""), s.projectID)
	if value != nil {
		value.UserID = s.userID
	}
	return value, s.wrap("Get", err)
}

// Detail reads limit, in-use and reserved values for this project/user pair.
func (s *UserQuotaScope) Detail(ctx context.Context) (*QuotaDetailResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, s.wrap("Detail", err)
	}
	result := s.get(ctx, "detail")
	body, fields, err := quotaObject(result)
	if err != nil {
		return nil, s.wrap("Detail", err)
	}
	var quota QuotaDetailSet
	if err := json.Unmarshal(body, &quota); err != nil {
		return nil, s.wrap("Detail", err)
	}
	return &QuotaDetailResource{QuotaDetailSet: quota, ProjectID: s.projectID, UserID: s.userID, Body: fields, Header: result.Header.Clone()}, nil
}

// Update uses the same concrete, snapshotted limit/force/extension inputs as
// ProjectQuotaScope.Update and always sends this scope's user_id query.
func (s *UserQuotaScope) Update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, s.wrap("Update", err)
	}
	body, err := prepareQuotaUpdate(opts, options...)
	if err != nil {
		return nil, s.wrap("Update", err)
	}
	endpoint := s.api.quotaURL(s.projectID, s.userID, "")
	client, err := guardedQuotaClient(s.api.client, http.MethodPut, endpoint)
	if err != nil {
		return nil, s.wrap("Update", err)
	}
	var result gophercloud.Result
	response, err := client.Put(ctx, endpoint, body, &result.Body, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	_, result.Header, result.Err = gophercloud.ParseResponse(response, err)
	value, err := decodeQuota(result, s.projectID)
	if value != nil {
		value.UserID = s.userID
	}
	return value, s.wrap("Update", err)
}

// Reset removes only this user's overrides within this fixed project. Native
// DELETE accepts 202/204; no GET or project-wide fallback follows a failure.
func (s *UserQuotaScope) Reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	if err := s.validate(ctx); err != nil {
		return nil, s.wrap("Reset", err)
	}
	var config resetOptions
	for _, apply := range options {
		if apply == nil {
			return nil, s.wrap("Reset", fmt.Errorf("%w: nil reset option", resource.ErrInvalidOption))
		}
		if err := apply(&config); err != nil {
			return nil, s.wrap("Reset", err)
		}
	}
	endpoint := s.api.quotaURL(s.projectID, s.userID, "")
	client, err := guardedQuotaClient(s.api.client, http.MethodDelete, endpoint)
	if err != nil {
		return nil, s.wrap("Reset", err)
	}
	response, err := client.Delete(ctx, endpoint, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted, http.StatusNoContent}})
	_, header, err := gophercloud.ParseResponse(response, err)
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, s.wrap("Reset", err)
	}
	return &ResetResponse{ProjectID: s.projectID, UserID: s.userID, Header: header.Clone()}, nil
}
