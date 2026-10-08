package image

import (
	"context"
	imageapi "github.com/JSYoo5B/go-openstacksdk/image/v2"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/query"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Image = images.Image

// Dependencies supplies Connection facts without coupling the image service to
// authentication or other service construction. Each operation owns its snapshot.
type Dependencies struct {
	CloudLocation func() (resource.CloudLocation, error)
}

type Service struct {
	API          *imageapi.Service
	Images       *resource.Collection[Image]
	client       *gophercloud.ServiceClient
	dependencies Dependencies
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return NewWithDependencies(client, Dependencies{})
}

func NewWithDependencies(client *gophercloud.ServiceClient, dependencies Dependencies) *Service {
	api := imageapi.NewWithDependencies(client, imageapi.Dependencies{CloudLocation: dependencies.CloudLocation})
	return &Service{client: client, dependencies: dependencies, API: api, Images: resource.NewCollection[Image](resource.Adapter[Image]{
		Kind:                     "image",
		IdentityFind:             true,
		IdentityMissingListQuery: url.Values{"os_hidden": {"true"}},
		Get:                      func(ctx context.Context, id string) (*Image, error) { return images.Get(ctx, client, id).Extract() },
		GetIdentityQuery: func(ctx context.Context, id string, q url.Values) (*Image, error) {
			var result images.GetResult
			result.Header, result.Err = nativefind.Get(ctx, client, []string{"images", id}, q, []int{200}, &result.Body)
			return result.Extract()
		},
		List:    func(q url.Values) pagination.Pager { return images.List(client, query.Adapter(q)) },
		Extract: images.ExtractImages,
		Delete:  func(ctx context.Context, id string) error { return images.Delete(ctx, client, id).ExtractErr() },
		ID:      func(i *Image) string { return i.ID }, Name: func(i *Image) string { return i.Name },
		NameQuery: func(name string) string { return name },
		Status:    func(i *Image) string { return string(i.Status) },
		Failed: func(status string) bool {
			return strings.EqualFold(status, "killed") || strings.EqualFold(status, "deleted")
		},
	})}
}
