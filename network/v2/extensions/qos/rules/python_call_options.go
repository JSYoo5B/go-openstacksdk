package rules

import (
	"encoding/json"
	"maps"

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

// qosCalls translates shared request calls into per-operation state callbacks
// in caller order. A nil call stays nil so each operation's apply loop reports
// its own nil-option error at the same position. value interprets Query and
// Field entries (and paging entries when paging is true) with the operation's
// own key rules; a nil value rejects them.
func qosCalls[T any, C request.Call](calls []C, fields func(*T) qosCallFields, value func(string, json.RawMessage, error) func(*T) error, paging bool) []func(*T) error {
	probe := fields(new(T))
	out := make([]func(*T) error, len(calls))
	for i, call := range calls {
		if any(call) == nil {
			continue
		}
		entry, err := request.Entry(call)
		if err != nil {
			out[i] = func(*T) error { return err }
			continue
		}
		out[i] = qosCall(entry, probe, fields, value, paging)
	}
	return out
}

func qosCall[T any](entry request.CallEntry, probe qosCallFields, fields func(*T) qosCallFields, value func(string, json.RawMessage, error) func(*T) error, paging bool) func(*T) error {
	unsupported := func(*T) error { return request.UnsupportedCall(entry) }
	switch entry.Kind {
	case request.CallHeader:
		if probe.headers == nil {
			return unsupported
		}
		key, text := entry.Key, entry.Text
		return func(s *T) error {
			target := fields(s).headers
			headers := maps.Clone(*target)
			if headers == nil {
				headers = make(map[string]string)
			}
			headers[key] = text
			*target = headers
			return nil
		}
	case request.CallHeaders:
		if probe.headers == nil {
			return unsupported
		}
		owned := entry.Headers
		return func(s *T) error { *fields(s).headers = maps.Clone(owned); return nil }
	case request.CallMicroversion:
		if probe.microversion == nil {
			return unsupported
		}
		text := entry.Text
		return func(s *T) error { x := text; *fields(s).microversion = &x; return nil }
	case request.CallBasePath:
		if probe.basePath == nil {
			return unsupported
		}
		text := entry.Text
		return func(s *T) error { x := text; *fields(s).basePath = &x; return nil }
	case request.CallIgnoreMissing:
		if probe.ignore == nil {
			return unsupported
		}
		ignore := entry.Bool
		return func(s *T) error { x := ignore; *fields(s).ignore = &x; return nil }
	case request.CallQuery, request.CallField:
		if value == nil {
			return unsupported
		}
		return value(entry.Key, entry.JSON, entry.JSONErr)
	case request.CallLimit, request.CallMarker, request.CallMaxItems, request.CallPaginated:
		if !paging || value == nil {
			return unsupported
		}
		// Paging controls are the Source limit/marker/max_items/paginated kwargs.
		var key string
		var raw []byte
		var err error
		switch entry.Kind {
		case request.CallLimit:
			key = "limit"
			raw, err = json.Marshal(entry.Int)
		case request.CallMarker:
			key = "marker"
			raw, err = json.Marshal(entry.Text)
		case request.CallMaxItems:
			key = "max_items"
			raw, err = json.Marshal(entry.Int)
		default:
			key = "paginated"
			raw, err = json.Marshal(entry.Bool)
		}
		return value(key, raw, err)
	}
	return unsupported
}
