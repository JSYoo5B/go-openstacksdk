package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func (s *Service) validateNetworkMutation(ctx context.Context) error {
	if ctx == nil {
		return networkMutationInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.client == nil || s.client.ProviderClient == nil || s.Networks == nil || s.Roles == nil {
		return networkMutationInvalid("network service is required")
	}
	if s.Networks != s.mutationNetworks || s.Roles != s.mutationRoles {
		return networkMutationInvalid("network service collections or role cache were replaced")
	}
	return rest.ValidateTarget(s.client, s.client.ServiceURL("networks"))
}

func wrapNetworkMutation(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &resource.OperationError{Operation: operation, Resource: "network", Cause: err}
}

func (s *Service) networkMutationTarget() (string, func(context.Context) error) {
	client := s.client
	provider, endpoint, base := client.ProviderClient, client.Endpoint, client.ResourceBase
	var sourceError error
	var mu sync.Mutex
	return client.ServiceURL("networks"), func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		if sourceError == nil && (s.client != client || client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || s.Networks != s.mutationNetworks || s.Roles != s.mutationRoles) {
			sourceError = networkMutationInvalid("network service source changed during mutation")
		}
		return sourceError
	}
}

// Mutation lookups use the same owned source guard and response pipeline as
// writes. The public Collection keeps its established native read contract.
func (s *Service) networkMutationLookup(guard func(context.Context) error) *resource.Collection[Network] {
	return rest.Collection(rest.CollectionSpec[Network]{
		Client: s.client, Path: "networks", Kind: "network", SingleKey: "network", PluralKey: "networks",
		ID: func(value *Network) string { return value.ID }, Name: func(value *Network) string { return value.Name },
		NameQuery: func(name string) string { return name },
		Metadata:  func(*Network) *resource.Metadata { return &resource.Metadata{} },
		Validate:  guard, SourceGuard: guard, Get: true, GetCodes: []int{http.StatusOK},
		ListCodes: []int{http.StatusOK, http.StatusNoContent}, Paging: rest.PagePolicy[Network]{HTTPLink: true},
	})
}

// CreateNetwork applies cloud defaults and concrete options, then invalidates
// shared roles after an accepted mutation, including a later decode failure.
// It returns the native network model rather than a mutable Python Resource.
func (s *Service) CreateNetwork(ctx context.Context, input CreateNetworkRequest, options ...NetworkOption) (*Network, error) {
	if err := s.validateNetworkMutation(ctx); err != nil {
		return nil, err
	}
	base, guard := s.networkMutationTarget()
	o, err := prepareNetworkMutation(true, input.Name, options)
	if err != nil {
		return nil, err
	}
	if o.hints {
		if err := s.requireNetworkAvailabilityZone(ctx, guard); err != nil {
			return nil, wrapNetworkMutation("create/availability zones", err)
		}
	}
	return s.executeNetworkMutation(ctx, "create", http.MethodPost, base, "", guard, o, http.StatusCreated, http.StatusAccepted)
}

// UpdateNetwork preserves supplied false/empty values. IDs avoid a lookup when
// fields or a revision condition are supplied; names resolve exactly once.
// With no changes it reads the current network and resets roles without PUT.
func (s *Service) UpdateNetwork(ctx context.Context, ref resource.Ref, options ...NetworkOption) (*Network, error) {
	if err := s.validateNetworkMutation(ctx); err != nil {
		return nil, err
	}
	base, guard := s.networkMutationTarget()
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	o, err := prepareNetworkMutation(false, "", options)
	if err != nil {
		return nil, err
	}
	if len(o.fields) == 0 && o.revision == nil {
		current, err := s.networkMutationLookup(guard).Find(ctx, ref)
		if err == nil {
			err = validateMutationNetwork(current, ref)
		}
		if err != nil {
			return nil, wrapNetworkMutation("update/lookup", err)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := guard(ctx); err != nil {
			return nil, err
		}
		s.mutationRoles.Reset()
		return current, nil
	}
	id, err := s.networkMutationLookup(guard).ResolveID(ctx, ref)
	if err != nil {
		return nil, wrapNetworkMutation("update/lookup", err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, err
	}
	return s.executeNetworkMutation(ctx, "update", http.MethodPut, base, id, guard, o, http.StatusOK, http.StatusCreated)
}

// DeleteNetwork reports false for a missing initial lookup. A successful
// DELETE (202/204), or a clean 404 after finding the resource, reports true and
// resets roles. Accepted-response read/Close errors retain true plus the error.
// The existing Networks.Delete collection method keeps its direct-ID contract.
func (s *Service) DeleteNetwork(ctx context.Context, ref resource.Ref) (bool, error) {
	if err := s.validateNetworkMutation(ctx); err != nil {
		return false, err
	}
	base, guard := s.networkMutationTarget()
	if err := ref.Validate(); err != nil {
		return false, err
	}
	current, err := s.networkMutationLookup(guard).Find(ctx, ref, resource.WithIgnoreMissing())
	if err != nil {
		return false, wrapNetworkMutation("delete/lookup", err)
	}
	if current == nil {
		return false, guard(ctx)
	}
	if err := validateMutationNetwork(current, ref); err != nil {
		return false, wrapNetworkMutation("delete/lookup", err)
	}
	accepted, err := s.deleteNetworkIDGuarded(ctx, base, current.ID, guard)
	if !accepted && ctx.Err() == nil && cleanNetworkDeleteMissing(err) {
		s.mutationRoles.Reset()
		return true, nil
	}
	return accepted, err
}

// Ignore only a direct native missing response, never a joined callback/source
// failure or an accepted status rejected by the SDK's original status policy.
func cleanNetworkDeleteMissing(err error) bool {
	if operation, ok := err.(*resource.OperationError); ok {
		err = operation.Cause
	}
	switch value := err.(type) {
	case gophercloud.ErrUnexpectedResponseCode:
		return value.Actual == http.StatusNotFound
	case *gophercloud.ErrUnexpectedResponseCode:
		return value != nil && value.Actual == http.StatusNotFound
	default:
		return false
	}
}

func validateMutationNetwork(value *Network, requested resource.Ref) error {
	if value == nil {
		return fmt.Errorf("Neutron returned no network")
	}
	if err := resource.ID(value.ID).Validate(); err != nil {
		return err
	}
	if requested != (resource.Ref{}) && !requested.IsName() && value.ID != requested.String() {
		return fmt.Errorf("Neutron returned network %q; requested %q", value.ID, requested.String())
	}
	return nil
}

func (s *Service) executeNetworkMutation(ctx context.Context, operation, method, base, id string, guard func(context.Context) error, o networkMutationOptions, codes ...int) (*Network, error) {
	endpoint := base
	if id != "" {
		endpoint += "/" + id
	}
	headers := map[string]string{}
	if o.revision != nil {
		headers["If-Match"] = fmt.Sprintf("revision_number=%d", *o.revision)
	}
	response, err := rest.DoJSONGuarded(ctx, s.client, guard, method, endpoint, map[string]any{"network": o.fields}, headers, codes...)
	if response != nil {
		// Reset before decoding: Neutron has accepted a state-changing request.
		// A malformed body must not leave a successful topology cache in place.
		s.mutationRoles.Reset()
	}
	if err != nil {
		if operation == "update" && ctx.Err() == nil && cleanNetworkDeleteMissing(err) {
			err = &resource.NotFoundError{Resource: "network", Reference: id, Cause: err}
		}
		return nil, wrapNetworkMutation(operation, err)
	}
	body, err := response.Object("network")
	if err != nil {
		return nil, wrapNetworkMutation(operation+"/decode", err)
	}
	var value Network
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, wrapNetworkMutation(operation+"/decode", response.Fail(err))
	}
	requested := resource.Ref{}
	if id != "" {
		requested = resource.ID(id)
	}
	if err := validateMutationNetwork(&value, requested); err != nil {
		return &value, wrapNetworkMutation(operation+"/verify", response.Fail(err))
	}
	return &value, nil
}

func (s *Service) deleteNetworkID(ctx context.Context, id string) (bool, error) {
	if err := s.validateNetworkMutation(ctx); err != nil {
		return false, err
	}
	if err := resource.ID(id).Validate(); err != nil {
		return false, err
	}
	base, guard := s.networkMutationTarget()
	return s.deleteNetworkIDGuarded(ctx, base, id, guard)
}

func (s *Service) deleteNetworkIDGuarded(ctx context.Context, base, id string, guard func(context.Context) error) (bool, error) {
	response, err := rest.DoJSONGuarded(ctx, s.client, guard, http.MethodDelete, base+"/"+id, nil, nil, http.StatusAccepted, http.StatusNoContent)
	if response != nil {
		s.mutationRoles.Reset()
	}
	return response != nil, wrapNetworkMutation("delete", err)
}

type networkMutationExtension struct {
	Alias    string            `json:"alias"`
	Metadata resource.Metadata `json:"-"`
}

func (s *Service) requireNetworkAvailabilityZone(ctx context.Context, guard func(context.Context) error) error {
	available := false
	spec := rest.CollectionSpec[networkMutationExtension]{
		Client: s.client, Path: "extensions", Kind: "network extension", PluralKey: "extensions",
		Metadata: func(value *networkMutationExtension) *resource.Metadata { return &value.Metadata },
		Validate: guard, SourceGuard: guard, ListCodes: []int{http.StatusOK},
		Paging: rest.PagePolicy[networkMutationExtension]{HTTPLink: true},
	}
	for extension, err := range rest.List(ctx, spec, nil) {
		if err != nil {
			return err
		}
		available = available || extension.Alias == "network_availability_zone"
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := guard(ctx); err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("%w: network_availability_zone extension is unavailable", resource.ErrUnsupported)
	}
	return nil
}
