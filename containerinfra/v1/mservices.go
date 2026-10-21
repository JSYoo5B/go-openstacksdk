package v1

import (
	"context"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// MagnumService describes one Magnum control-plane service record returned by
// GET /mservices. Gophercloud v2.15.0 has no counterpart, so this model is
// SDK-owned. Nullable wire values stay pointers: a null disabled_reason and a
// null updated_at remain distinguishable from an empty string. Unknown response
// fields are not retained.
type MagnumService struct {
	ID             int     `json:"id"`
	Binary         string  `json:"binary"`
	Host           string  `json:"host"`
	State          string  `json:"state"`
	ReportCount    int     `json:"report_count"`
	DisabledReason *string `json:"disabled_reason"`
	CreatedAt      *string `json:"created_at"`
	UpdatedAt      *string `json:"updated_at"`
}

// ListMagnumServices issues one GET /mservices request and yields the records
// in response order. The route has no query options in this API, 200 is the
// only accepted status, and a continuation link, if a server ever returned one,
// is not followed. The OpenStack-API-Version header follows the client
// Microversion. A canceled context prevents the request and failures keep the
// native error as the cause; the read is never replayed by the SDK.
func (s *Service) ListMagnumServices(ctx context.Context) iter.Seq2[*MagnumService, error] {
	return func(yield func(*MagnumService, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(nil, request.Wrap("List", "mservices", err))
			return
		}
		var body struct {
			Services []MagnumService `json:"mservices"`
		}
		_, err := s.client.Get(ctx, s.client.ServiceURL("mservices"), &body, &gophercloud.RequestOpts{OkCodes: []int{200}})
		if err != nil {
			yield(nil, request.Wrap("List", "mservices", err))
			return
		}
		for i := range body.Services {
			if err := ctx.Err(); err != nil {
				yield(nil, request.Wrap("List", "mservices", err))
				return
			}
			if !yield(&body.Services[i], nil) {
				return
			}
		}
	}
}

// AllMagnumServices collects ListMagnumServices and stops at the first error.
func (s *Service) AllMagnumServices(ctx context.Context) ([]*MagnumService, error) {
	var out []*MagnumService
	for value, err := range s.ListMagnumServices(ctx) {
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}
