package users

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"gophercloudsdk/db/v1/instances"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"

	db "github.com/gophercloud/gophercloud/v2/openstack/db/v1/databases"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/db/v1/users"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// UserResource preserves the account's host in addition to Gophercloud's User.
// Password is populated only if Trove includes it in the response.
type UserResource struct {
	User
	Host string `json:"host"`
}

// UserScope fixes a Trove instance. The pinned SDKs expose no usable per-user
// fetch, so Get searches the complete list for an exact user name.
type UserScope struct {
	api        *API
	instanceID string
	resources  *resource.Collection[UserResource]
	deletions  *resource.Collection[UserResource]
	host       *string
}

type scopeOptions struct{ host *string }

// ScopeOption selects a library-owned user scope policy.
type ScopeOption func(*scopeOptions) error

// WithHost fixes one account host. Without it, lookup lists all account hosts
// and duplicate literal names fail; explicit ID deletion targets the '%' host.
func WithHost(host string) ScopeOption {
	return func(options *scopeOptions) error {
		if err := validateUserHost(host); err != nil {
			return err
		}
		options.host = &host
		return nil
	}
}

// InInstance resolves the instance once. Explicit instance IDs avoid lookup.
func (a *API) InInstance(ctx context.Context, parent resource.Ref, options ...ScopeOption) (*UserScope, error) {
	var config scopeOptions
	for _, apply := range options {
		if apply == nil {
			return nil, userInvalid("nil scope option")
		}
		if err := apply(&config); err != nil {
			return nil, err
		}
	}
	id, err := instances.New(a.client).Resources.ResolveID(ctx, parent)
	if err != nil {
		return nil, err
	}
	scope := &UserScope{api: a, instanceID: id, host: config.host}
	binding := resource.Adapter[UserResource]{
		Kind:       "users",
		ValidateID: validateUserName,
		ID:         func(value *UserResource) string { return value.Name },
		Name:       func(value *UserResource) string { return value.Name },
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*UserResource, error] {
			if len(query) != 0 {
				return func(yield func(*UserResource, error) bool) { yield(nil, resource.ErrUnsupported) }
			}
			return func(yield func(*UserResource, error) bool) {
				for value, err := range resource.Stream(ctx, upstream.List(a.client, id), extractScopedUsers) {
					if err != nil {
						yield(nil, err)
						return
					}
					if scope.host != nil && effectiveHost(value.Host) != *scope.host {
						continue
					}
					if !yield(value, nil) {
						return
					}
				}
			}
		},
	}
	binding.Get = func(ctx context.Context, name string) (*UserResource, error) {
		return scope.getAccount(ctx, userAccountID(name, scope.targetHost()))
	}
	scope.resources = resource.NewCollection(binding)
	// Keep the host-qualified identifier inside deletion operations. Public
	// names remain literal, including '@'; named deletion carries the resolved
	// account host and deletion waits keep that same account across polls.
	binding.ID = func(value *UserResource) string { return userAccountID(value.Name, value.Host) }
	binding.ValidateID = validateUserAccountID
	binding.Get = scope.getAccount
	binding.Delete = func(ctx context.Context, account string) error {
		if err := validateUserAccountID(account); err != nil {
			return err
		}
		return a.Delete(ctx, id, escapeUserAccount(account))
	}
	scope.deletions = resource.NewCollection(binding)
	return scope, nil
}

// Get lists for an exact literal name at the scope's host, or '%' by default.
// It does not interpret '@' as a hostname in the caller's name. Find with Name
// searches all hosts unless WithHost fixes one.
func (s *UserScope) Get(ctx context.Context, name string) (*UserResource, error) {
	return s.resources.Get(ctx, name)
}

func (s *UserScope) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*UserResource, error) {
	return s.resources.Find(ctx, ref, options...)
}

func (s *UserScope) List(ctx context.Context, options ...resource.ListOption) iter.Seq2[*UserResource, error] {
	return s.resources.List(ctx, options...)
}

func (s *UserScope) All(ctx context.Context, options ...resource.ListOption) ([]*UserResource, error) {
	return s.resources.All(ctx, options...)
}

// ResolveID returns a literal name that can be reused in this scope. A lookup
// of a non-default account requires WithHost, so its host cannot be lost.
func (s *UserScope) ResolveID(ctx context.Context, ref resource.Ref) (string, error) {
	if !ref.IsName() {
		return s.resources.ResolveID(ctx, ref)
	}
	value, err := s.Find(ctx, ref)
	if err != nil {
		return "", err
	}
	if s.host == nil && effectiveHost(value.Host) != "%" {
		return "", &resource.OperationError{Operation: "resolve", Resource: "users", Cause: fmt.Errorf("%w: fix the account host with users.WithHost before resolving its ID", resource.ErrUnsupported)}
	}
	if err := validateUserName(value.Name); err != nil {
		return "", err
	}
	return value.Name, nil
}

// Delete ignores missing accounts by default. Name resolves a unique account
// and retains its host; an explicit ID is a literal name at the scope's host,
// or the default '%' host when no host scope was selected.
func (s *UserScope) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	account, err := s.deletionRef(ref)
	if err != nil {
		return err
	}
	return s.deletions.Delete(ctx, account, options...)
}

// WaitDeleted uses list lookup for the resolved account, retaining its host.
func (s *UserScope) WaitDeleted(ctx context.Context, ref resource.Ref, options ...resource.WaitOption) error {
	account, err := s.deletionRef(ref)
	if err != nil {
		return err
	}
	return s.deletions.WaitDeleted(ctx, account, options...)
}

// Wait returns ErrUnsupported because the user model has no status field.
func (s *UserScope) Wait(ctx context.Context, ref resource.Ref, status string, options ...resource.WaitOption) (*UserResource, error) {
	return s.resources.Wait(ctx, ref, status, options...)
}

func (s *UserScope) deletionRef(ref resource.Ref) (resource.Ref, error) {
	if ref.IsName() {
		return ref, nil
	}
	if err := validateUserName(ref.String()); err != nil {
		return resource.Ref{}, err
	}
	return resource.ID(userAccountID(ref.String(), s.targetHost())), nil
}

func (s *UserScope) targetHost() string {
	if s.host != nil {
		return *s.host
	}
	return "%"
}

func (s *UserScope) getAccount(ctx context.Context, account string) (*UserResource, error) {
	separator := strings.LastIndexByte(account, '@')
	name, host := account[:separator], account[separator+1:]
	var found *UserResource
	for value, err := range s.resources.List(ctx, resource.WithName(name)) {
		if err != nil {
			return nil, err
		}
		if effectiveHost(value.Host) != host {
			continue
		}
		if found != nil {
			return nil, &resource.AmbiguousError{Resource: "users", Name: name, IDs: []string{account, account}}
		}
		found = value
	}
	if found == nil {
		return nil, &resource.NotFoundError{Resource: "users", Reference: account}
	}
	return found, nil
}

func effectiveHost(host string) string {
	if host == "" {
		return "%"
	}
	return host
}

func userAccountID(name, host string) string {
	host = effectiveHost(host)
	if validateUserHost(host) != nil {
		return ""
	}
	return name + "@" + host
}

func validateUserHost(host string) error {
	if strings.TrimSpace(host) == "" || strings.ContainsAny(host, "@/\\\x00\r\n") {
		return userInvalid("user host must be a non-empty path segment without '@'")
	}
	return nil
}

func validateUserAccountID(account string) error {
	separator := strings.LastIndexByte(account, '@')
	if separator < 0 || separator == len(account)-1 {
		return userInvalid("user account must include a host")
	}
	if err := validateUserName(account[:separator]); err != nil {
		return err
	}
	return validateUserHost(account[separator+1:])
}

// WSGI decodes PATH_INFO and Trove unquotes again before splitting the last
// '@'. Protect literal percent signs across both decodes. Escape dots too,
// which Trove's router may treat as a file type suffix.
func escapeUserAccount(account string) string {
	protected := strings.ReplaceAll(account, "%", "%25")
	protected = strings.ReplaceAll(protected, ".", "%2E")
	return url.PathEscape(protected)
}

// Gophercloud's nested Database model lacks a character_set tag. Keep its
// public User fields while owning the wire conversion for the scoped list.
func extractScopedUsers(page pagination.Page) ([]UserResource, error) {
	var wire struct {
		Users []struct {
			Name      string `json:"name"`
			Host      string `json:"host"`
			Password  string `json:"password"`
			Databases []struct {
				Name    string `json:"name"`
				CharSet string `json:"character_set"`
				Collate string `json:"collate"`
			} `json:"databases"`
		} `json:"users"`
	}
	if err := page.(upstream.UserPage).ExtractInto(&wire); err != nil {
		return nil, err
	}
	values := make([]UserResource, len(wire.Users))
	for i, value := range wire.Users {
		values[i] = UserResource{User: User{Name: value.Name, Password: value.Password}, Host: value.Host}
		if value.Databases != nil {
			values[i].Databases = make([]db.Database, len(value.Databases))
			for j, database := range value.Databases {
				values[i].Databases[j] = db.Database{Name: database.Name, CharSet: database.CharSet, Collate: database.Collate}
			}
		}
	}
	return values, nil
}

// Create submits one user in Trove's users array. The asynchronous API returns
// no user body, so success is reported as a nil error.
func (s *UserScope) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) error {
	return s.create(ctx, BatchCreateOpts{opts}, true, options...)
}

// CreateBatch validates the entire batch before POST. Trove determines the
// request's atomicity and completion; password or access updates are separate.
func (s *UserScope) CreateBatch(ctx context.Context, opts BatchCreateOpts, options ...CreateOption) error {
	return s.create(ctx, opts, false, options...)
}

func (s *UserScope) create(ctx context.Context, opts BatchCreateOpts, single bool, options ...CreateOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg, err := request.Apply(opts, options...)
	if err != nil {
		return request.Wrap("create", "users", err)
	}
	if len(cfg.Options) == 0 {
		return userInvalid("create batch must not be empty")
	}
	if single && len(cfg.Options) != 1 {
		return userInvalid("single create requires exactly one user")
	}
	if err := request.ValidateCapabilities(cfg, true, false, false); err != nil {
		return request.Wrap("create", "users", err)
	}
	// A host scope supplies its host when omitted and rejects conflicting input.
	// Copy the options slice so applying the scope does not mutate caller input.
	cfg.Options = append(BatchCreateOpts(nil), cfg.Options...)
	if s.host != nil {
		for i := range cfg.Options {
			if cfg.Options[i].Host == "" {
				cfg.Options[i].Host = *s.host
			} else if cfg.Options[i].Host != *s.host {
				return userInvalid("create host conflicts with the fixed scope host")
			}
		}
	}
	seen := make(map[string]bool, len(cfg.Options))
	for _, user := range cfg.Options {
		if err := validateUserName(user.Name); err != nil {
			return err
		}
		if user.Name == "root" {
			return userInvalid("root is a reserved user name")
		}
		if user.Password == "" {
			return userInvalid("user password is required")
		}
		account := userAccountID(user.Name, user.Host)
		if err := validateUserAccountID(account); err != nil {
			return err
		}
		if seen[account] {
			return userInvalid("duplicate user account in create batch")
		}
		seen[account] = true
		databases := make(map[string]bool, len(user.Databases))
		for _, database := range user.Databases {
			if strings.TrimSpace(database.Name) == "" || database.Name == "." || database.Name == ".." || strings.ContainsAny(database.Name, "/\\\x00\r\n") || len(database.Name) > 64 {
				return userInvalid("database name must be one non-empty path segment of at most 64 bytes")
			}
			if databases[database.Name] {
				return userInvalid("duplicate database name for user")
			}
			databases[database.Name] = true
		}
	}
	builder := createOptsBuilder{base: cfg.Options, config: cfg}
	if _, err := builder.ToUserCreateMap(); err != nil {
		return request.Wrap("create", "users", err)
	}
	return request.Wrap("create", "users", upstream.Create(ctx, s.api.client, s.instanceID, builder).ExtractErr())
}

func validateUserName(name string) error {
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00\r\n") {
		return userInvalid("user name must identify one non-empty path segment")
	}
	return nil
}

func userInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
