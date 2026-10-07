// Package quotaclasssets owns Manila's named quota-class GET/PUT API, absent
// from Gophercloud v2.15.0. A class name is explicit and never a Keystone ref.
package quotaclasssets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/manilaversion"
	"github.com/JSYoo5B/gophercloudsdk/internal/project"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/JSYoo5B/gophercloudsdk/sharedfilesystems/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// QuotaClassSet has the same twelve exact optional limits as Manila project
// quota responses. The alias shares only that data model, never scope methods.
type QuotaClassSet = quotasets.QuotaSet

type QuotaClassResource struct {
	QuotaClassSet
	ClassName  string
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

type QuotaClassScope struct {
	api       *API
	className string
}

func (a *API) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil {
		return fmt.Errorf("%w: Manila quota class API is required", resource.ErrInvalidOption)
	}
	if err := project.ValidateClient(ctx, a.client); err != nil {
		return err
	}
	_, err := a.minorVersion()
	return err
}

func (a *API) minorVersion() (int, error) {
	return manilaversion.Minor(a.client)
}

// InClass fixes an exact class name as one unescaped URL path segment. Names
// such as default or UUID-shaped class names are never looked up as projects.
func (a *API) InClass(ctx context.Context, className string) (*QuotaClassScope, error) {
	if err := a.validate(ctx); err != nil {
		return nil, classError("InClass", className, err)
	}
	if _, err := a.minorVersion(); err != nil {
		return nil, classError("InClass", className, err)
	}
	if err := resource.ID(className).Validate(); err != nil {
		return nil, classError("InClass", className, err)
	}
	return &QuotaClassScope{api: a, className: className}, nil
}

func (s *QuotaClassScope) ClassName() string {
	if s == nil {
		return ""
	}
	return s.className
}

func (s *QuotaClassScope) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: quota class scope is required", resource.ErrInvalidOption)
	}
	if err := s.api.validate(ctx); err != nil {
		return err
	}
	return resource.ID(s.className).Validate()
}

func (s *QuotaClassScope) endpoint() (string, error) {
	minor, err := s.api.minorVersion()
	if err != nil {
		return "", err
	}
	root := "quota-class-sets"
	if minor <= 6 {
		root = "os-quota-class-sets"
	}
	return s.api.client.ServiceURL(root, s.className), nil
}

func classError(operation, className string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		err = &resource.NotFoundError{Resource: "shared file system quota class", Reference: className, Cause: err}
	}
	return request.Wrap(operation, "shared file system quota class", err)
}

func decodeClass(response *http.Response, envelope map[string]json.RawMessage, className string, err error) (*QuotaClassResource, error) {
	if err != nil {
		return nil, err
	}
	data := envelope["quota_class_set"]
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("quota_class_set must be a JSON object")
	}
	var quota QuotaClassSet
	if err := json.Unmarshal(data, &quota); err != nil {
		return nil, err
	}
	return &QuotaClassResource{QuotaClassSet: quota, ClassName: className, Body: fields, Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// Get fetches this named class without a lookup or an invented class list.
func (s *QuotaClassScope) Get(ctx context.Context) (*QuotaClassResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, classError("Get", s.ClassName(), err)
	}
	endpoint, err := s.endpoint()
	if err != nil {
		return nil, classError("Get", s.className, err)
	}
	client, err := fixedrequest.New(s.api.client, http.MethodGet, endpoint)
	if err != nil {
		return nil, classError("Get", s.className, err)
	}
	var envelope map[string]json.RawMessage
	response, err := client.Get(ctx, endpoint, &envelope, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}})
	value, err := decodeClass(response, envelope, s.className, err)
	return value, classError("Get", s.className, err)
}
