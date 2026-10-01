package blockstorage

import (
	"context"
	blockstorageapi "gophercloudsdk/blockstorage/v3"
	"net/url"
	"strings"

	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Volume = volumes.Volume

type Service struct {
	API     *blockstorageapi.Service
	Volumes *resource.Collection[Volume]
	client  *gophercloud.ServiceClient
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return &Service{client: client, API: blockstorageapi.New(client), Volumes: resource.NewCollection[Volume](resource.Adapter[Volume]{
		Kind:         "volume",
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*Volume, error) { return volumes.Get(ctx, client, id).Extract() },
		GetIdentityQuery: func(ctx context.Context, id string, q url.Values) (*Volume, error) {
			var result volumes.GetResult
			result.Header, result.Err = nativefind.Get(ctx, client, []string{"volumes", id}, q, []int{200}, &result.Body)
			return result.Extract()
		},
		List:    func(q url.Values) pagination.Pager { return volumes.List(client, query.Adapter(q)) },
		Extract: volumes.ExtractVolumes,
		Delete: func(ctx context.Context, id string) error {
			return volumes.Delete(ctx, client, id, volumes.DeleteOpts{}).ExtractErr()
		},
		ID: func(v *Volume) string { return v.ID }, Name: func(v *Volume) string { return v.Name },
		NameQuery: func(name string) string { return name },
		Status:    func(v *Volume) string { return v.Status },
		Failed:    func(status string) bool { return strings.HasPrefix(strings.ToLower(status), "error") },
	})}
}
