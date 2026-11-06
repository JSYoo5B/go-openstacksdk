package policies

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Spec is a required JSON object. RawMessage preserves exact numbers and
// explicit values; WithCreateSpec accepts maps and other JSON object inputs.
type CreateOpts struct {
	Name string          `json:"name"`
	Spec json.RawMessage `json:"spec"`
}

// CreateOption is one option of Create: WithCreateOptions, WithCreateSpec,
// request.Header, request.Field or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = senlin.WriteCalls[CreateOpts]()

func WithCreateOptions(value CreateOpts) CreateOption { return request.Custom(senlin.Snapshot(value)) }
func WithCreateSpec(value any) CreateOption {
	raw, err := json.Marshal(value)
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: policy spec: %v", resource.ErrInvalidOption, err)
		}
		if err := senlin.Object(raw, "spec"); err != nil {
			return err
		}
		config.Options.Spec = append(json.RawMessage(nil), raw...)
		return nil
	})
}

// Update changes only the policy name. A nil name and no extension is an
// invalid stateless update; this API does not keep Python's cached dirty state.
type UpdateOpts struct {
	Name *string `json:"name,omitempty"`
}

// UpdateOption is one option of Update, TrackedPolicy.Edit and
// TrackedPolicy.Commit: WithUpdateOptions, WithUpdateName, request.Header,
// request.Field or request.Custom over *request.Config[UpdateOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = senlin.WriteCalls[UpdateOpts]()

func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.Custom(senlin.Snapshot(value)) }
func WithUpdateName(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error { copy := value; config.Options.Name = &copy; return nil })
}

type ValidateOpts struct {
	Spec json.RawMessage `json:"spec"`
}

// ValidateOption is one option of Validate: WithValidateOptions,
// WithValidateSpec, request.Header, request.Field or request.Custom over
// *request.Config[ValidateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ValidateOption = request.WriteCall

var validateCalls = senlin.WriteCalls[ValidateOpts]()

func WithValidateOptions(value ValidateOpts) ValidateOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithValidateSpec(value any) ValidateOption {
	raw, err := json.Marshal(value)
	return request.Custom(func(config *request.Config[ValidateOpts]) error {
		if err != nil {
			return fmt.Errorf("%w: policy spec: %v", resource.ErrInvalidOption, err)
		}
		if err := senlin.Object(raw, "spec"); err != nil {
			return err
		}
		config.Options.Spec = append(json.RawMessage(nil), raw...)
		return nil
	})
}

func (a *API) decode(ctx context.Context, method, path string, body json.RawMessage, extraHeaders map[string]string, code int) (*Policy, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, err
	}
	response, err := rest.DoJSON(ctx, client, method, client.ServiceURL(path), body, extraHeaders, code)
	if err != nil {
		return nil, err
	}
	return rest.Decode(response, "policy", func(value *Policy) *resource.Metadata { return &value.Metadata })
}

var policyName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func validateName(value string) error {
	if err := senlin.Required(value); err != nil {
		return err
	}
	if len(value) >= 255 || !policyName.MatchString(value) {
		return fmt.Errorf("%w: policy name must start with an ASCII letter and contain fewer than 255 ASCII letters, digits, underscores, periods or hyphens", resource.ErrInvalidOption)
	}
	return nil
}

var responseFields = []string{"id", "type", "data", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at", "policy"}

func (a *API) Create(ctx context.Context, value CreateOpts, options ...CreateOption) (*Policy, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Create", "clustering.policies", err)
	}
	config, err := calls.ApplyConfig(value, createCalls, options...)
	if err == nil {
		err = validateName(config.Options.Name)
	}
	if err == nil {
		err = senlin.Object(config.Options.Spec, "spec")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "policy", responseFields...)
	}
	if err != nil {
		return nil, request.Wrap("Create", "clustering.policies", err)
	}
	result, err := a.decode(ctx, http.MethodPost, "policies", body, config.Headers, http.StatusCreated)
	return result, request.Wrap("Create", "clustering.policies", err)
}

func (a *API) Update(ctx context.Context, ref resource.Ref, value UpdateOpts, options ...UpdateOption) (*Policy, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("Update", "clustering.policies", err)
	}
	if err := ref.Validate(); err != nil {
		return nil, request.Wrap("Update", "clustering.policies", err)
	}
	config, err := calls.ApplyConfig(value, updateCalls, options...)
	if err == nil && config.Options.Name != nil {
		err = validateName(*config.Options.Name)
	}
	if err == nil && config.Options.Name == nil && len(config.Fields) == 0 {
		err = fmt.Errorf("%w: policy update has no fields", resource.ErrInvalidOption)
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "policy", append(append([]string(nil), responseFields...), "spec")...)
	}
	if err != nil {
		return nil, request.Wrap("Update", "clustering.policies", err)
	}
	extraHeaders := maps.Clone(config.Headers)
	identity := ref.String()
	if ref.IsName() {
		found, err := rest.Collection(spec(client)).Find(ctx, ref)
		if err != nil {
			return nil, request.Wrap("Update", "clustering.policies", err)
		}
		identity = found.ID
	}
	if err := senlin.Identifier(identity); err != nil {
		return nil, request.Wrap("Update", "clustering.policies", err)
	}
	result, err := a.decode(ctx, http.MethodPatch, "policies/"+url.PathEscape(identity), body, extraHeaders, http.StatusOK)
	return result, request.Wrap("Update", "clustering.policies", err)
}

// Validate requires an explicitly selected numeric Senlin version 1.2 or newer.
// It requests server validation without creating or storing a policy locally.
func (a *API) Validate(ctx context.Context, value ValidateOpts, options ...ValidateOption) (*Policy, error) {
	if err := senlin.RequireVersion(ctx, a.RawClient(), 2); err != nil {
		return nil, request.Wrap("Validate", "clustering.policies", err)
	}
	config, err := calls.ApplyConfig(value, validateCalls, options...)
	if err == nil {
		err = senlin.Object(config.Options.Spec, "spec")
	}
	var body json.RawMessage
	if err == nil {
		body, err = senlin.Body(config, "policy", responseFields...)
	}
	if err != nil {
		return nil, request.Wrap("Validate", "clustering.policies", err)
	}
	if err := senlin.RequireVersion(ctx, a.RawClient(), 2); err != nil {
		return nil, request.Wrap("Validate", "clustering.policies", err)
	}
	result, err := a.decode(ctx, http.MethodPost, "policies/validate", body, config.Headers, http.StatusOK)
	return result, request.Wrap("Validate", "clustering.policies", err)
}
