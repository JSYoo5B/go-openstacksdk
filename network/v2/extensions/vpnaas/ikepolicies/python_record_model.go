package ikepolicies

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// IKEPolicyRecord owns the nullable descriptor view and received HTTP proof.
type IKEPolicyRecord struct {
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

// Source derives Resource directly, with no NetworkResource/tag/revision defaults.
// lifetime has no implicit units/value children; separate units/value are Body fields.
var policyRecordProfile = neutronfind.Profile{
	Kind: "ikepolicies", Path: "vpn/ikepolicies", Singular: "ikepolicy", Plural: "ikepolicies", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/vpn/ikepolicies", PreserveUnknownBody: false, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"},
		{Name: "auth_algorithm", Wire: "auth_algorithm", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "encryption_algorithm", Wire: "encryption_algorithm", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "pfs", Wire: "pfs", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null"},
		{Name: "phase1_negotiation_mode", Wire: "phase1_negotiation_mode", Missing: "null"},
		{Name: "units", Wire: "units", Missing: "null"},
		{Name: "ike_version", Wire: "ike_version", Missing: "null"},
		{Name: "lifetime", Wire: "lifetime", Kind: neutronfind.Dictionary, Missing: "null"}, {Name: "value", Wire: "value", Kind: neutronfind.Integer, Missing: "null"}},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "auth_algorithm", Wire: "auth_algorithm"}, {Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "encryption_algorithm", Wire: "encryption_algorithm"}, {Name: "name", Wire: "name"}, {Name: "pfs", Wire: "pfs"}, {Name: "project_id", Wire: "project_id"}, {Name: "phase1_negotiation_mode", Wire: "phase1_negotiation_mode"}, {Name: "ike_version", Wire: "ike_version"}},
}
