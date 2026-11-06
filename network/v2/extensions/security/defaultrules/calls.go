package defaultrules

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

var createKeyword = neutronfind.OrderedJSON(func(c *CreateOpts) (*map[string]json.RawMessage, *[]string) { return &c.Fields, &c.order },
	func(key string) error { return invalid("create attribute %q is not JSON", key) })

var listKeyword = neutronfind.OrderedJSON(func(c *ListOpts) (*map[string]json.RawMessage, *[]string) { return &c.Query, &c.order },
	func(key string) error { return invalid("list query %q is not JSON", key) })

var findKeyword = neutronfind.OrderedJSON(func(c *FindOpts) (*map[string]json.RawMessage, *[]string) { return &c.Query, &c.order },
	func(key string) error { return invalid("find query %q is not JSON", key) })

// listQuery drops the kwargs Resource.list discards, with their marshal error.
func listQuery(c *ListOpts, key string, raw json.RawMessage, err error) error {
	if listIgnores(key) {
		return nil
	}
	return listKeyword(c, key, raw, err)
}

// Shared call handlers. request.Headers replaces the headers; request.Header
// and request.BasePath are unsupported.
var (
	createCalls = calls.Spec[CreateOpts]{
		Field:        neutronfind.FieldJSON(createKeyword),
		Headers:      calls.Value(func(c *CreateOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *CreateOpts) **string { return &c.Microversion }),
	}
	deleteCalls = calls.Spec[DeleteOpts]{
		IgnoreMissing: calls.Pointer(func(c *DeleteOpts) **bool { return &c.IgnoreMissing }),
		Headers:       calls.Value(func(c *DeleteOpts) *map[string]string { return &c.Headers }),
		Microversion:  calls.Pointer(func(c *DeleteOpts) **string { return &c.Microversion }),
	}
	getCalls = calls.Spec[GetOpts]{
		Headers:      calls.Value(func(c *GetOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *GetOpts) **string { return &c.Microversion }),
	}
	listCalls = calls.Spec[ListOpts]{
		Query:        calls.JSONQuery(listQuery),
		Paging:       calls.JSONPaging(listQuery),
		Headers:      calls.Value(func(c *ListOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *ListOpts) **string { return &c.Microversion }),
	}
	findCalls = calls.Spec[FindOpts]{
		IgnoreMissing: calls.Pointer(func(c *FindOpts) **bool { return &c.IgnoreMissing }),
		Query:         calls.JSONQuery(findKeyword),
		Headers:       calls.Value(func(c *FindOpts) *map[string]string { return &c.Headers }),
		Microversion:  calls.Pointer(func(c *FindOpts) **string { return &c.Microversion }),
	}
)
