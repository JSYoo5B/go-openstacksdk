package image

import (
	"context"
	"net/url"
	"strings"

	"gophercloudsdk/internal/query"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type Image = images.Image

type Service struct {
	Images *resource.Collection[Image]
	client *gophercloud.ServiceClient
}

func (s *Service) RawClient() *gophercloud.ServiceClient { return s.client }

func New(client *gophercloud.ServiceClient) *Service {
	return &Service{client: client, Images: resource.NewCollection[Image](resource.Adapter[Image]{
		Kind:    "image",
		Get:     func(ctx context.Context, id string) (*Image, error) { return images.Get(ctx, client, id).Extract() },
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
