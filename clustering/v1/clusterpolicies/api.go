// Package clusterpolicies reads policy bindings in a fixed Senlin cluster scope.
package clusterpolicies

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// Scope retains the canonical parent identity independently of returned models.
// Get identities refer to a policy, not the binding's separate ID.
type Scope struct {
	client    *gophercloud.ServiceClient
	clusterID string
	Resources *resource.Collection[ClusterPolicy]
}

func (s *Scope) RawClient() *gophercloud.ServiceClient {
	if s == nil {
		return nil
	}
	return s.client
}

func (s *Scope) ClusterID() string {
	if s == nil {
		return ""
	}
	return s.clusterID
}

// InCluster fetches a direct identity once, or performs one exact Name lookup.
// The response Body ID becomes the fixed parent for all subsequent operations.
func (a *API) InCluster(ctx context.Context, ref resource.Ref) (*Scope, error) {
	client := a.RawClient()
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterpolicies", err)
	}
	parent, err := rest.Collection(parentSpec(client)).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterpolicies", err)
	}
	if err := senlin.Validate(ctx, client); err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterpolicies", err)
	}
	scope := &Scope{client: client, clusterID: parent.ID}
	scope.Resources = scope.collection()
	return scope, nil
}

func parentSpec(client *gophercloud.ServiceClient) rest.CollectionSpec[clusters.Cluster] {
	return rest.CollectionSpec[clusters.Cluster]{
		Client: client, Path: "clusters", Kind: "clustering.clusters", SingleKey: "cluster", PluralKey: "clusters", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID: func(value *clusters.Cluster) string { return value.ID }, Name: func(value *clusters.Cluster) string { return value.Name },
		NameQuery: func(name string) string { return name }, Metadata: func(value *clusters.Cluster) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.Validate(ctx, client) }, ValidateID: senlin.Identifier,
		ValidateQuery: func(_ context.Context, query url.Values) error { return senlin.SortGrammar(query.Get("sort")) },
		ValidateItem: func(value *clusters.Cluster) error {
			identity, err := bodyIdentity(value.Body, "id")
			if err != nil {
				return err
			}
			value.ID = identity
			value.Name, err = bodyName(value.Body, "name")
			return err
		},
		Paging: rest.PagePolicy[clusters.Cluster]{HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *clusters.Cluster) (string, error) { return value.ID, senlin.Identifier(value.ID) }},
	}
}

func bodyIdentity(body map[string]json.RawMessage, field string) (string, error) {
	var identity string
	if err := json.Unmarshal(body[field], &identity); err != nil {
		return "", fmt.Errorf("%w: Senlin response requires %s string: %v", resource.ErrInvalidOption, field, err)
	}
	if err := senlin.Identifier(identity); err != nil {
		return "", fmt.Errorf("Senlin response %s: %w", field, err)
	}
	return identity, nil
}

func bodyName(body map[string]json.RawMessage, field string) (string, error) {
	if len(body[field]) == 0 {
		return "", nil
	}
	var name string
	if err := json.Unmarshal(body[field], &name); err != nil {
		return "", fmt.Errorf("%w: Senlin response %s must be a string or null: %v", resource.ErrInvalidOption, field, err)
	}
	return name, nil
}

func (s *Scope) spec() rest.CollectionSpec[ClusterPolicy] {
	return rest.CollectionSpec[ClusterPolicy]{
		Client: s.RawClient(), Path: "clusters/" + url.PathEscape(s.ClusterID()) + "/policies", Kind: "clustering.clusterpolicies",
		SingleKey: "cluster_policy", PluralKey: "cluster_policies", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID: func(value *ClusterPolicy) string { return value.PolicyID }, Name: func(value *ClusterPolicy) string { return value.PolicyName },
		NameQuery: func(name string) string { return name }, NameQueryKey: "policy_name",
		Metadata: func(value *ClusterPolicy) *resource.Metadata { return &value.Metadata }, ValidateID: senlin.Identifier,
		Validate: func(ctx context.Context) error {
			if err := senlin.Validate(ctx, s.RawClient()); err != nil {
				return err
			}
			return senlin.Identifier(s.ClusterID())
		},
		ValidateQuery:        func(_ context.Context, query url.Values) error { return validateQuery(query) },
		ValidateInitialQuery: func(_ context.Context, query url.Values) error { return validateInitialQuery(query) },
		ValidateItem: func(value *ClusterPolicy) error {
			identities := make(map[string]string, 3)
			for _, field := range []string{"id", "policy_id", "cluster_id"} {
				identity, err := bodyIdentity(value.Body, field)
				if err != nil {
					return err
				}
				identities[field] = identity
			}
			if identities["cluster_id"] != s.ClusterID() {
				return fmt.Errorf("%w: cluster policy response belongs to a different cluster", resource.ErrInvalidOption)
			}
			// Exact wire keys own identity; case-folded extension keys must not
			// override the parent or the policy used by collection polling.
			value.ID, value.PolicyID, value.ClusterID = identities["id"], identities["policy_id"], identities["cluster_id"]
			var err error
			value.PolicyName, err = bodyName(value.Body, "policy_name")
			if err != nil {
				return err
			}
			value.URIClusterID = s.ClusterID()
			return nil
		},
		Paging: rest.PagePolicy[ClusterPolicy]{HTTPLink: true, MaxItemsLimitHint: false, StopOnEmptyPage: true},
	}
}

func (s *Scope) collection() *resource.Collection[ClusterPolicy] {
	return rest.Collection(s.spec())
}

// Get sends a policy name, short ID or UUID directly to the scoped controller.
func (s *Scope) Get(ctx context.Context, policyIdentity string) (*ClusterPolicy, error) {
	return rest.Collection(s.spec()).Get(ctx, policyIdentity)
}

func (s *Scope) List(ctx context.Context, options ...ListOption) iter.Seq2[*ClusterPolicy, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*ClusterPolicy, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(config)
		}
		if err == nil {
			filters, err = senlin.PrepareBodyFilters(config, filterSpec)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.clusterpolicies", err))
			return
		}
		control := rest.ListControl{MaxItems: config.Options.MaxItems,
			SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated, LimitHint: false}
		for value, err := range rest.ListWithControl(ctx, s.spec(), query, control) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.clusterpolicies", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.clusterpolicies", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (s *Scope) All(ctx context.Context, options ...ListOption) ([]*ClusterPolicy, error) {
	return senlin.All(s.List(ctx, options...))
}
