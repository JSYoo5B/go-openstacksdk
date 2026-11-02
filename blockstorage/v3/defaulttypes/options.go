package defaulttypes

import (
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Opts carries per-request headers for Get and Set, a Go extension.
type Opts = cinderowned.ReadOpts

// Option is one option of Get and Set: request.Headers (replaces the
// headers) or request.Custom over *Opts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type Option = request.GetCall

// DeleteOpts selects ignore_missing for Unset; nil keeps the Python
// behavior of ignoring a 404.
type DeleteOpts = cinderowned.DeleteOpts

// DeleteOption is one option of Unset: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over *DeleteOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

// ListOpts carries headers and the Resource.list max_items and paginated
// knobs. Python default_types() takes no query, so none is offered.
type ListOpts = cinderowned.ListOpts

// ListOption is one option of List and All: request.MaxItems,
// request.Paginated, request.Headers (replaces the headers) or
// request.Custom over *ListOpts. Other shared calls, including request.Query,
// request.Limit and request.Marker, fail with resource.ErrInvalidOption
// before HTTP.
type ListOption = request.ListCall

var (
	readCalls   = calls.Spec[Opts]{Headers: cinderowned.ReadHeadersCall}
	deleteCalls = calls.Spec[DeleteOpts]{Headers: cinderowned.DeleteHeadersCall, IgnoreMissing: cinderowned.DeleteIgnoreMissingCall}
	listCalls   = calls.Spec[ListOpts]{
		Headers: cinderowned.ListHeadersCall(cinderowned.Self[ListOpts]),
		Paging: calls.Paging[ListOpts]{
			MaxItems:  cinderowned.ListPaging(cinderowned.Self[ListOpts]).MaxItems,
			Paginated: cinderowned.ListPaging(cinderowned.Self[ListOpts]).Paginated,
		},
	}
)
