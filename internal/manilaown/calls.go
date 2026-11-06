package manilaown

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// The handlers below translate the shared calls of package request for the
// hand-written Manila operations of the sharedfilesystems packages. Each
// operation keeps its request.Config apply loop and capability check, so
// calls.ApplyConfig turns the calls into the same options the removed
// per-operation With* functions built. A control without a handler fails with
// request.UnsupportedCall before HTTP.

// QueryCalls handles reads and lists that take only a vendor query:
// request.Query with a string or []string value.
func QueryCalls[O any]() calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{Query: calls.ConfigQuery[O]}
}

// ListCalls handles the single-response lists of ListFor: request.Query and
// request.MaxItems, a local consumption cap stored under ControlArgument.
// These lists never took Header, Headers, Microversion, Limit, Marker or
// Paginated calls.
func ListCalls[O any]() calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{
		Query: calls.ConfigQuery[O],
		Paging: calls.Paging[request.Config[O]]{
			MaxItems: func(c *request.Config[O], n int) error {
				return EditControl(c, ControlArgument, func(l *resource.ListControl) { l.MaxItems = n })
			},
		},
	}
}

// FieldCalls handles creates, updates and actions that take only extension
// body members: request.Field.
func FieldCalls[O any]() calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{Field: calls.ConfigField[O]}
}

// HeaderFieldCalls handles writes that take request.Header and request.Field.
func HeaderFieldCalls[O any]() calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{Header: calls.ConfigHeader[O], Field: calls.ConfigField[O]}
}

// HeaderCalls handles reads that take only request.Header.
func HeaderCalls[O any]() calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{Header: calls.ConfigHeader[O]}
}

// IgnoreMissingCalls handles deletes that take only request.IgnoreMissing,
// stored through the given pointer field (nil keeps the operation default).
func IgnoreMissingCalls[O any](ignoreMissing func(*O) **bool) calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{IgnoreMissing: optionPointer(ignoreMissing)}
}

// FindCalls handles FindByNameOrID: request.Query (a list-fallback filter, a
// string or []string value) and request.IgnoreMissing through the given
// pointer field.
func FindCalls[O any](ignoreMissing func(*O) **bool) calls.Spec[request.Config[O]] {
	return calls.Spec[request.Config[O]]{Query: calls.ConfigQuery[O], IgnoreMissing: optionPointer(ignoreMissing)}
}

func optionPointer[O any](field func(*O) **bool) func(*request.Config[O], bool) error {
	return calls.Pointer(func(c *request.Config[O]) **bool { return field(&c.Options) })
}

// EditControl edits the local list control stored under key in the
// configuration arguments.
func EditControl[O any](cfg *request.Config[O], key string, edit func(*resource.ListControl)) error {
	control, _, err := request.Argument[resource.ListControl](*cfg, key)
	if err != nil {
		return err
	}
	edit(&control)
	if cfg.Arguments == nil {
		cfg.Arguments = make(map[string]any)
	}
	cfg.Arguments[key] = control
	return nil
}
