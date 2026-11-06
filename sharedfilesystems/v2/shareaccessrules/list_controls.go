package shareaccessrules

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/manilaown"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const listControlArgument = "shareaccessrules.list_control"

// listCalls handles request.Query, request.Header, request.MaxItems and
// request.Paginated. MaxItems caps consumed access rules locally: zero is
// unlimited and a negative value is rejected before HTTP; it never adds a wire
// limit or marker. Paginated(false) selects one response; the modern access
// endpoint already returns a single collection, so neither value follows
// advertised links.
var listCalls = calls.Spec[request.Config[ListOpts]]{
	Header: calls.ConfigHeader[ListOpts],
	Query:  calls.ConfigQuery[ListOpts],
	Paging: calls.Paging[request.Config[ListOpts]]{
		MaxItems: func(c *request.Config[ListOpts], n int) error {
			return manilaown.EditControl(c, listControlArgument, func(control *resource.ListControl) { control.MaxItems = n })
		},
		Paginated: func(c *request.Config[ListOpts], paginated bool) error {
			return manilaown.EditControl(c, listControlArgument, func(control *resource.ListControl) { control.SinglePage = !paginated })
		},
	},
}
