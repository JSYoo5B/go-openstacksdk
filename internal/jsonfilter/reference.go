package jsonfilter

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ReferenceLastComponent projects an HREFToUUID-style response value without
// using the reference as a request URL. Null bypasses formatting. Original path
// spelling, including literal Unicode/space and percent escapes, is retained.
// Go's URL parser remains stricter than Python urlsplit for some authorities.
func ReferenceLastComponent(raw json.RawMessage) (json.RawMessage, error) {
	var ref *string
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, err
	}
	if ref == nil {
		return json.RawMessage("null"), nil
	}
	// Escape only '%' for parsing, preventing Path from decoding original
	// escapes. Query and fragment remain separate and are never returned.
	parsed, err := url.Parse(strings.ReplaceAll(*ref, "%", "%25"))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" && parsed.User == nil || parsed.Path == "" {
		return nil, fmt.Errorf("reference requires scheme, authority and path")
	}
	return json.Marshal(parsed.Path[strings.LastIndexByte(parsed.Path, '/')+1:])
}
