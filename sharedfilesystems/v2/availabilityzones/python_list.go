package availabilityzones

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/octaviarest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// PythonListMicroversion is the Manila microversion sent by ListZones. The
// pinned Python resource declares it as _max_microversion, and the
// /availability-zones route replaced the legacy os-availability-zone route in
// that release.
const PythonListMicroversion = "2.7"

// Zone is the declared view of one /availability-zones row. Timestamps stay
// strings exactly as sent (Python keeps them as str). Body preserves every
// wire field, including absent and null values.
type Zone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// ListZones reads GET /availability-zones with microversion 2.7, as Python
// openstacksdk does, and follows availability_zones_links if a deployment
// emits them. It is separate from List, which keeps Gophercloud's legacy
// /os-availability-zone route. The client's configured Microversion is not
// modified; the request always carries 2.7.
func (a *API) ListZones(ctx context.Context) iter.Seq2[*Zone, error] {
	return func(yield func(*Zone, error) bool) {
		fail := func(err error) { yield(nil, request.Wrap("ListZones", "availability zones", err)) }
		if a == nil || a.client == nil || ctx == nil {
			fail(fmt.Errorf("%w: client and context are required", resource.ErrInvalidOption))
			return
		}
		pinned := *a.client
		pinned.Microversion = PythonListMicroversion
		for row, err := range octaviarest.Pages(ctx, &pinned, pinned.ServiceURL("availability-zones"), "availability_zones", false) {
			if err != nil {
				fail(err)
				return
			}
			var fields map[string]json.RawMessage
			var zone Zone
			if err := json.Unmarshal(row.Raw, &fields); err == nil && fields == nil {
				err = gophercloud.ErrUnexpectedType{Expected: "availability zone JSON object", Actual: "null"}
				fail(err)
				return
			} else if err != nil {
				fail(err)
				return
			}
			if err := json.Unmarshal(row.Raw, &zone); err != nil {
				fail(err)
				return
			}
			zone.Body, zone.Header, zone.StatusCode = fields, row.Header.Clone(), row.Status
			if !yield(&zone, nil) {
				return
			}
		}
	}
}

// AllZones collects ListZones. The empty result is a non-nil slice.
func (a *API) AllZones(ctx context.Context) ([]*Zone, error) {
	zones := make([]*Zone, 0)
	for zone, err := range a.ListZones(ctx) {
		if err != nil {
			return nil, err
		}
		zones = append(zones, zone)
	}
	return zones, nil
}
