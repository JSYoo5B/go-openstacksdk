package compute

import (
	"context"
	computeapi "gophercloudsdk/compute/v2"
	"net/url"
	"regexp"
	"strings"

	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Response models reuse Gophercloud's field definitions in the initial version.
type Server = servers.Server
type Flavor = flavors.Flavor

type Service struct {
	API     *computeapi.Service
	Servers *Servers
	Flavors *resource.Collection[Flavor]
	client  *gophercloud.ServiceClient
}

// RawClient is an escape hatch for APIs not covered by the SDK yet.
// Treat its configuration as immutable once requests run concurrently.
func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

// Dependencies connects named references to other service collections.
// Connection supplies these resolvers automatically.
type Dependencies struct {
	Image   func(context.Context, resource.Ref) (string, error)
	Network func(context.Context, resource.Ref) (string, error)
	Volume  func(context.Context, resource.Ref) (string, error)
}

type Servers struct {
	*resource.Collection[Server]
	dependencies Dependencies
	flavors      *resource.Collection[Flavor]
	client       *gophercloud.ServiceClient
}

func New(client *gophercloud.ServiceClient, dependencies Dependencies) *Service {
	service := &Service{
		API:    computeapi.New(client),
		client: client,
		Servers: &Servers{
			dependencies: dependencies, client: client,
			Collection: resource.NewCollection[Server](resource.Adapter[Server]{
				Kind:         "server",
				IdentityFind: true,
				Get:          func(ctx context.Context, id string) (*Server, error) { return servers.Get(ctx, client, id).Extract() },
				GetIdentityQuery: func(ctx context.Context, id string, q url.Values) (*Server, error) {
					var result servers.GetResult
					result.Header, result.Err = nativefind.Get(ctx, client, []string{"servers", id}, q, []int{200, 203}, &result.Body)
					return result.Extract()
				},
				List:    func(q url.Values) pagination.Pager { return servers.List(client, query.Adapter(q)) },
				Extract: servers.ExtractServers,
				Delete:  func(ctx context.Context, id string) error { return servers.Delete(ctx, client, id).ExtractErr() },
				ID:      func(s *Server) string { return s.ID }, Name: func(s *Server) string { return s.Name },
				NameQuery: func(name string) string { return "^" + regexp.QuoteMeta(name) + "$" },
				Status:    func(s *Server) string { return s.Status },
				Failed:    func(status string) bool { return strings.EqualFold(status, "ERROR") },
			}),
		},
		Flavors: resource.NewCollection[Flavor](resource.Adapter[Flavor]{
			Kind:    "flavor",
			Get:     func(ctx context.Context, id string) (*Flavor, error) { return flavors.Get(ctx, client, id).Extract() },
			List:    func(q url.Values) pagination.Pager { return flavors.ListDetail(client, query.Adapter(q)) },
			Extract: flavors.ExtractFlavors,
			ID:      func(f *Flavor) string { return f.ID }, Name: func(f *Flavor) string { return f.Name },
		}),
	}

	service.Servers.flavors = service.Flavors
	return service
}
