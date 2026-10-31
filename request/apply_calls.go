package request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Capabilities names the extension mechanisms a generated native operation
// supports: extension JSON body fields, extension query values, extension
// headers and the keys of its optional secondary arguments.
type Capabilities struct {
	Body      bool
	Query     bool
	Headers   bool
	Arguments []string
	// Unchecked skips the final ValidateCapabilities step, like Apply alone,
	// for hand-written operations that validate the Config themselves. Body,
	// Query, Headers and Arguments are then ignored.
	Unchecked bool
}

// ApplyCalls builds an operation's Config from base and shared calls, in
// caller order. It is the call-based form of Apply followed by
// ValidateCapabilities, and every input it shares with those functions keeps
// their result and error text:
//
//   - Field behaves like WithField, Header like WithHeader, Query with a
//     string like WithQuery and Query with a []string like WithQueryValues.
//   - Custom over Config[T] runs like an Option[T]; operation-specific
//     options of generated packages are built this way.
//   - A nil call, or Custom with a nil function, fails like a nil Option.
//   - After all calls, a nil typed options pointer and caps are checked with
//     the messages of Apply and ValidateCapabilities.
//
// Query with any other value type fails with resource.ErrInvalidOption,
// because native query serialization only carries text. Custom over another
// state type, and controls that native operations do not implement
// (Headers, Microversion, BasePath, IgnoreMissing and paging), fail with
// UnsupportedCall's error. Every failure happens before HTTP.
func ApplyCalls[T any, C Call](base T, caps Capabilities, calls ...C) (Config[T], error) {
	c := Config[T]{Options: base, Fields: make(map[string]json.RawMessage), Query: make(url.Values), Arguments: make(map[string]any), Headers: make(map[string]string)}
	jsonQuery := false
	for _, call := range calls {
		if any(call) == nil {
			return c, fmt.Errorf("%w: nil request option", resource.ErrInvalidOption)
		}
		entry, err := Entry(call)
		if err != nil {
			return c, err
		}
		if err := applyEntry(&c, entry, caps, &jsonQuery); err != nil {
			return c, err
		}
	}
	v := reflect.ValueOf(c.Options)
	if v.IsValid() && v.Kind() == reflect.Pointer && v.IsNil() {
		return c, fmt.Errorf("%w: nil typed request options", resource.ErrInvalidOption)
	}
	if caps.Unchecked {
		return c, nil
	}
	check := c
	if jsonQuery {
		// A JSON query on an operation without query support counts as a
		// query extension, in ValidateCapabilities' order.
		check.Query = url.Values{"": nil}
	}
	return c, ValidateCapabilities(check, caps.Body, caps.Query, caps.Headers, caps.Arguments...)
}

func applyEntry[T any](c *Config[T], entry CallEntry, caps Capabilities, jsonQuery *bool) error {
	switch entry.Kind {
	case CallCustom:
		fn, ok := CustomFunc[Config[T]](entry)
		if !ok {
			return UnsupportedCall(entry)
		}
		if fn == nil {
			return fmt.Errorf("%w: nil request option", resource.ErrInvalidOption)
		}
		return fn(c)
	case CallField:
		// WithField: key check first, then the construction-time marshal error.
		if strings.TrimSpace(entry.Key) == "" {
			return fmt.Errorf("%w: empty extension key", resource.ErrInvalidOption)
		}
		if entry.JSONErr != nil {
			return fmt.Errorf("%w: extension %q: %v", resource.ErrInvalidOption, entry.Key, entry.JSONErr)
		}
		c.Fields[entry.Key] = append(json.RawMessage(nil), entry.JSON...)
		return nil
	case CallHeader:
		// WithHeader.
		if strings.TrimSpace(entry.Key) == "" || strings.ContainsAny(entry.Key, " \t\r\n:") || strings.ContainsAny(entry.Text, "\r\n") {
			return fmt.Errorf("%w: invalid extension header", resource.ErrInvalidOption)
		}
		c.Headers[http.CanonicalHeaderKey(entry.Key)] = entry.Text
		return nil
	case CallQuery:
		if strings.TrimSpace(entry.Key) == "" {
			return fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		switch entry.Form {
		case QueryText:
			// WithQuery.
			c.Query.Set(entry.Key, entry.Text)
		case QueryList:
			// WithQueryValues.
			if len(entry.Values) == 0 {
				c.Query.Del(entry.Key)
				return nil
			}
			c.Query[entry.Key] = slices.Clone(entry.Values)
		default:
			if !caps.Query && !caps.Unchecked {
				// Reported after all calls, like other query extensions.
				*jsonQuery = true
				return nil
			}
			return fmt.Errorf("%w: query %q needs a string or []string value for this operation", resource.ErrInvalidOption, entry.Key)
		}
		return nil
	}
	return UnsupportedCall(entry)
}
