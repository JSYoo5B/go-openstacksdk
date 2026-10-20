package shareattachments

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API is a library-owned collection scoped by an explicit server argument.
// New performs no HTTP and does not change the selected service client.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

// Record separates the declared Source Resource view from actual response JSON.
// ServerID records the URI argument; response ids never become request routes.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ServerID       string
	Discovery      []DiscoveryObservation
}
type DiscoveryObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}
type DeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
	Discovery      []DiscoveryObservation
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func unsupportedControl(key string) error {
	return fmt.Errorf("%w: share attachment control %q requires a typed SDK option", resource.ErrUnsupported, key)
}
func control(key string) bool {
	switch key {
	case "cls", "self", "session", "resource_type", "connection", "server_id", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "jmespath_filters", "paginated", "max_items", "allow_unknown_params":
		return true
	}
	return false
}
func knownField(key string) bool {
	switch key {
	case "export_location", "share_id", "share_proto", "status", "tag", "uuid", "id", "name":
		return true
	}
	return false
}
func recordFields() []string {
	return []string{"export_location", "share_id", "share_proto", "status", "tag", "uuid", "id", "name"}
}
func recordMetadata(v *Record) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func (v *Record) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, v.Wire); err != nil {
		return err
	}
	v.Envelope = bytes.Clone(data)
	return nil
}
func receipt(response *rest.Response) *Record {
	if response == nil {
		return nil
	}
	return &Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func nativeResponse(err error) *rest.Response {
	var physical *resource.ResponseError
	if errors.As(err, &physical) {
		return &rest.Response{Body: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode}
	}
	var renewed *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &renewed) {
		return nativeResponse(renewed.ErrOriginal)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &native) {
		return &rest.Response{Body: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func project(fields map[string]json.RawMessage, header http.Header, status int) *resource.RawResource {
	out := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage), Header: header.Clone(), StatusCode: status}}
	for _, key := range recordFields() {
		raw, present := fields[key]
		if !present {
			raw = json.RawMessage("null")
			if key == "export_location" {
				raw = json.RawMessage(`""`)
			}
		}
		if key == "id" && !present {
			if alternate, exists := fields["share_id"]; exists {
				raw = alternate
			}
		}
		out.Body[key] = bytes.Clone(raw)
	}
	// A raw ServiceClient does not supply a Connection current_location.
	out.Body["location"] = json.RawMessage("null")
	return out
}
func (p *prepared) translate(response *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	result := receipt(response)
	result.ServerID = p.server
	result.Discovery = p.discoveryCopy()
	if err := p.guard(p.ctx); err != nil {
		return result, response.Fail(err)
	}
	fields := copyFields(seed)
	// Resource.fetch/create ignores JSON ValueError, retaining the initial seed.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		members, err := cloudfilter.ObjectMembers(response.Body)
		if err != nil {
			return result, response.Fail(err)
		}
		raw := response.Body
		for _, member := range members {
			if member.Key == "share" {
				raw = member.Value
				break
			}
		}
		var wire resource.RawResource
		if err := json.Unmarshal(raw, &wire); err != nil {
			return result, response.Fail(err)
		}
		wire.Header = response.Header.Clone()
		wire.StatusCode = response.StatusCode
		result.Wire = &wire
		for key, value := range wire.Body {
			if key != "self" && knownField(key) {
				fields[key] = bytes.Clone(value)
			}
		}
	}
	result.Resource = project(fields, response.Header, response.StatusCode)
	if err := p.guard(p.ctx); err != nil {
		result.Resource = nil
		return result, response.Fail(cloudread.ContextError(p.ctx, err))
	}
	return result, nil
}
func (p *prepared) discoveryCopy() []DiscoveryObservation {
	out := make([]DiscoveryObservation, len(p.discovery))
	for i, v := range p.discovery {
		out[i] = DiscoveryObservation{Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
	}
	return out
}
func (p *prepared) proof(response *rest.Response) *Record {
	result := receipt(response)
	if result == nil {
		result = &Record{}
	}
	result.ServerID = p.server
	result.Discovery = p.discoveryCopy()
	return result
}
func mergeFields(a, b map[string]json.RawMessage) map[string]json.RawMessage {
	out := copyFields(a)
	if out == nil {
		out = make(map[string]json.RawMessage)
	}
	maps.Copy(out, copyFields(b))
	return out
}
func seedID(id string) map[string]json.RawMessage {
	raw, _ := json.Marshal(id)
	return map[string]json.RawMessage{"id": raw}
}
func (p *prepared) discoveryFailure(err error) error {
	if len(p.discovery) == 0 {
		return err
	}
	last := p.discovery[len(p.discovery)-1]
	return (&rest.Response{Body: last.Envelope, Header: last.Header, StatusCode: last.StatusCode}).Fail(err)
}
func validateRaw(key string, raw json.RawMessage) (json.RawMessage, error) {
	if raw == nil {
		return json.RawMessage("null"), nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, invalid("attribute %q must contain complete UTF-8 JSON", key)
	}
	return bytes.Clone(raw), nil
}
