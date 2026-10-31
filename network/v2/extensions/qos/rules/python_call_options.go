package rules

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// qosCallFields points at the shared controls an operation state supports.
// A nil pointer means the operation rejects that control.
type qosCallFields struct {
	headers      *map[string]string
	microversion **string
	ignore       **bool
	basePath     **string
}

// qosCalls translates one call's options into per-operation callbacks over the
// operation's exported options struct T with calls.Translate, in caller order.
// Operation-specific With* options and request.Custom over T yield their own
// function; any other Custom is rejected. A nil option (or a Custom with a nil
// function) stays nil so each operation's apply loop reports its own
// nil-option error at the same position. value interprets Query and Field
// entries (and paging entries, as the limit/marker/max_items/paginated kwargs,
// when paging is true) with the operation's own key rules; a nil value rejects
// them.
func qosCalls[T any, C request.Call](options []C, fields func(*T) qosCallFields, value func(string, json.RawMessage, error) func(*T) error, paging bool) []func(*T) error {
	probe := fields(new(T))
	var spec calls.Spec[T]
	if probe.headers != nil {
		spec.Header, spec.Headers = calls.HeaderMap(func(s *T) *map[string]string { return fields(s).headers })
	}
	if probe.microversion != nil {
		spec.Microversion = calls.Pointer(func(s *T) **string { return fields(s).microversion })
	}
	if probe.basePath != nil {
		spec.BasePath = calls.Pointer(func(s *T) **string { return fields(s).basePath })
	}
	if probe.ignore != nil {
		spec.IgnoreMissing = calls.Pointer(func(s *T) **bool { return fields(s).ignore })
	}
	if value != nil {
		kw := func(s *T, key string, raw json.RawMessage, err error) error { return value(key, raw, err)(s) }
		spec.Query = calls.JSONQuery(kw)
		spec.Field = spec.Query
		if paging {
			spec.Paging = calls.JSONPaging(kw)
		}
	}
	return calls.Translate(options, spec)
}
