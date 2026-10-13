package extensions

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ExtensionRecord owns the Source attribute/default view and actual evidence.
type ExtensionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Compute Extension derives from Resource directly and discards unknown Body fields.
// id uses alias only when the actual id key is absent; explicit null wins.
var extensionRecordProfile = neutronfind.Profile{
	ServiceRole: "compute", Kind: "extensions", Path: "extensions", Singular: "extension", Plural: "extensions", IDKey: "id", NameKey: "name", AlternateIDKey: "alias", VersionedPath: "/v2.1/extensions", PreserveUnknownBody: false, SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, SourceLinkPolicy: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null", Alias: "alias"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "alias", Wire: "alias", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "links", Wire: "links", Kind: neutronfind.List, ListItemKind: neutronfind.Dictionary, Missing: "null"},
		{Name: "namespace", Wire: "namespace", Missing: "null"},
		{Name: "updated_at", Wire: "updated", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}
