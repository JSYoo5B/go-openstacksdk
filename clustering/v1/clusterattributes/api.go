// Package clusterattributes collects raw node attributes in a fixed cluster scope.
package clusterattributes

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strings"

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

// Scope fixes the canonical parent and normalized JSONPath independently of
// returned rows. JSONPath syntax belongs to the server's parser.
type Scope struct {
	client    *gophercloud.ServiceClient
	clusterID string
	path      string
	Resources *resource.Collection[ClusterAttribute]
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

func (s *Scope) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func normalizedPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "None" {
		return "", fmt.Errorf("%w: cluster attribute JSONPath must not be empty or None", resource.ErrInvalidOption)
	}
	return path, nil
}

func pathSegment(path string) string {
	// PathEscape deliberately leaves dots unchanged. Encode complete dot
	// segments so JSONPath input cannot become URI traversal at a router.
	if path == "." || path == ".." {
		return strings.ReplaceAll(path, ".", "%2E")
	}
	return url.PathEscape(path)
}

// InCluster performs one parent GET or exact Name lookup before fixing the URI.
// The 1.2 gate and JSONPath preflight run before that lookup and again afterward.
func (a *API) InCluster(ctx context.Context, ref resource.Ref, path string) (*Scope, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 2); err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterattributes", err)
	}
	path, err := normalizedPath(path)
	if err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterattributes", err)
	}
	parent, err := rest.Collection(parentSpec(client)).Find(ctx, ref)
	if err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterattributes", err)
	}
	if err := senlin.RequireVersion(ctx, client, 2); err != nil {
		return nil, request.Wrap("InCluster", "clustering.clusterattributes", err)
	}
	scope := &Scope{client: client, clusterID: parent.ID, path: path}
	scope.Resources = rest.Collection(scope.spec())
	return scope, nil
}

func requiredIdentity(body map[string]json.RawMessage) (string, error) {
	var identity string
	if err := json.Unmarshal(body["id"], &identity); err != nil {
		return "", fmt.Errorf("%w: Senlin response requires an id string: %v", resource.ErrInvalidOption, err)
	}
	if err := senlin.Identifier(identity); err != nil {
		return "", err
	}
	return identity, nil
}

func parentSpec(client *gophercloud.ServiceClient) rest.CollectionSpec[clusters.Cluster] {
	return rest.CollectionSpec[clusters.Cluster]{
		Client: client, Path: "clusters", Kind: "clustering.clusters", SingleKey: "cluster", PluralKey: "clusters", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID: func(value *clusters.Cluster) string { return value.ID }, Name: func(value *clusters.Cluster) string { return value.Name },
		NameQuery: func(name string) string { return name }, Metadata: func(value *clusters.Cluster) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.RequireVersion(ctx, client, 2) }, ValidateID: senlin.Identifier,
		ValidateQuery: func(_ context.Context, query url.Values) error { return senlin.SortGrammar(query.Get("sort")) },
		ValidateItem: func(value *clusters.Cluster) error {
			identity, err := requiredIdentity(value.Body)
			if err != nil {
				return err
			}
			value.ID = identity
			value.Name = ""
			if name, present := value.Body["name"]; present {
				if err := json.Unmarshal(name, &value.Name); err != nil {
					return fmt.Errorf("%w: Senlin response name must be a string or null: %v", resource.ErrInvalidOption, err)
				}
			}
			return nil
		},
		Paging: rest.PagePolicy[clusters.Cluster]{HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *clusters.Cluster) (string, error) { return value.ID, senlin.Identifier(value.ID) }},
	}
}

func (s *Scope) spec() rest.CollectionSpec[ClusterAttribute] {
	return rest.CollectionSpec[ClusterAttribute]{
		Client: s.RawClient(), Path: "clusters/" + url.PathEscape(s.ClusterID()) + "/attrs/" + pathSegment(s.Path()),
		Kind: "clustering.clusterattributes", PluralKey: "cluster_attributes",
		ListCodes: []int{http.StatusOK, http.StatusAccepted},
		ID:        func(value *ClusterAttribute) string { return value.NodeID }, ValidateID: senlin.Identifier,
		Metadata: func(value *ClusterAttribute) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error {
			if err := senlin.RequireVersion(ctx, s.RawClient(), 2); err != nil {
				return err
			}
			if err := senlin.Identifier(s.ClusterID()); err != nil {
				return err
			}
			_, err := normalizedPath(s.Path())
			return err
		},
		ValidateInitialQuery: func(_ context.Context, query url.Values) error {
			if len(query) != 0 {
				return fmt.Errorf("%w: cluster attribute query options have no pinned proxy or published server contract", resource.ErrUnsupported)
			}
			return nil
		},
		ValidateItem: func(value *ClusterAttribute) error {
			identity, err := requiredIdentity(value.Body)
			if err != nil {
				return err
			}
			value.NodeID = identity
			value.URIClusterID, value.URIPath = s.ClusterID(), s.Path()
			return nil
		},
		Paging: rest.PagePolicy[ClusterAttribute]{HTTPLink: true},
	}
}

// List is lazy and follows only explicit links which retain the fixed URI.
func (s *Scope) List(ctx context.Context) iter.Seq2[*ClusterAttribute, error] {
	return func(yield func(*ClusterAttribute, error) bool) {
		for value, err := range rest.List(ctx, s.spec(), nil) {
			if !yield(value, request.Wrap("List", "clustering.clusterattributes", err)) {
				return
			}
		}
	}
}
func (s *Scope) All(ctx context.Context) ([]*ClusterAttribute, error) { return senlin.All(s.List(ctx)) }
