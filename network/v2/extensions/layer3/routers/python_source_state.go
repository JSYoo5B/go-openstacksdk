package routers

import "encoding/json"

// SourceBody returns the pinned Python Resource body attributes of this
// lookup as raw JSON keyed by wire name, before Body casts or missing
// defaults. Canonical-only members (a response member named by its attribute
// name rather than its wire key) are stored under the wire key, and
// canonical/wire collisions follow Python's ordered last-member-wins
// consumption. Unknown extension members are kept. It is nil when the record
// has no Source view (for example a metadata-only failure record).
func (v *RouterActionRecord) SourceBody() map[string]json.RawMessage {
	if v == nil {
		return nil
	}
	return v.source.Body()
}

// TranslateSourceBody applies the same pinned Resource._translate_response
// body mapping to one selected response object, such as an accepted update
// response member. It returns only the wire-keyed attributes the object
// contributes.
func (v *RouterActionRecord) TranslateSourceBody(object json.RawMessage) (map[string]json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	return v.source.Translate(object)
}
