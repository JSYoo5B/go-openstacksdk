package drivers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ListVendorPassthruRaw follows the pinned list_driver_vendor_passthru: it first
// reads the driver with GET /drivers/{name}, so a missing driver fails with the
// native 404 attributed to that read, and then reads
// GET /drivers/{name}/vendor_passthru/methods. The result maps each method name
// to its unchanged JSON document, so unknown per-method fields are kept. Both
// reads accept only 200, are sent once and use the configured client
// microversion.
func (a *API) ListVendorPassthruRaw(ctx context.Context, name string) (map[string]json.RawMessage, error) {
	if name == "" {
		return nil, request.Wrap("ListVendorPassthruRaw", "drivers", fmt.Errorf("%w: driver name must not be empty", resource.ErrInvalidOption))
	}
	if _, err := a.GetDriverDetails(ctx, name); err != nil {
		return nil, request.Wrap("ListVendorPassthruRaw", "drivers", err)
	}
	var methods map[string]json.RawMessage
	if _, err := a.client.Get(ctx, a.client.ServiceURL("drivers", url.PathEscape(name), "vendor_passthru", "methods"), &methods, &gophercloud.RequestOpts{OkCodes: []int{200}}); err != nil {
		return nil, request.Wrap("ListVendorPassthruRaw", "drivers", err)
	}
	if methods == nil {
		methods = map[string]json.RawMessage{}
	}
	return methods, nil
}
