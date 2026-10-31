package compute

import (
	"encoding/json"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverview"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CloudServerReadOpts owns the local controls of the cloud server reads.
// Filters follow the cloud filter rules: nil or null selects only the
// name-or-ID phase, a JSON object is an ordered dictionary filter and a JSON
// string is a JMESPath expression. List options reach servers.API.ListCloudRecords.
type CloudServerReadOpts struct {
	Filters *json.RawMessage
	List    []servers.RecordFindOption
	// Bare skips the address expansion of the typed server getters. Python
	// expands unless bare is set; the record readers never expand.
	Bare    bool
	Address []ServerAddressOption
	// Detailed replaces the interface expansion with GetOpenStackVars
	// hostvars (Python detailed=True). Bare still skips every expansion.
	Detailed bool
}

// CloudServerReadOption mutates CloudServerReadOpts.
type CloudServerReadOption func(*CloudServerReadOpts) error

// CloudServerSearchResult separates the complete consumed inventory from the
// ordinary name/dictionary matches. Value is the filtered JSON array, or the
// arbitrary JMESPath output when Expression is true; an expression result has
// no source-row association and Records is then empty.
type CloudServerSearchResult struct {
	Value      json.RawMessage
	Records    []*servers.ServerFindRecord
	Inventory  []*servers.ServerFindRecord
	Expression bool
}

// CloudServerResult is the outcome of the typed server getters and actions.
// Interfaces is nil when expansion was skipped or failed before a view existed.
type CloudServerResult struct {
	Server     *servers.Server
	Interfaces *ServerAddressView
	// AdminPassword is the password Nova returned or the caller supplied.
	AdminPassword string
	// Record is the Source-shaped lookup row, when the operation found one.
	Record *servers.ServerFindRecord
	// Sent reports whether a mutating request was sent.
	Sent bool
	// Hostvars is the detailed expansion; nil unless Detailed was requested.
	Hostvars *CloudServerHostvars
}

// CloudServerUpdate lists the only attributes update_server accepts (name and
// description). An unset attribute is omitted, an explicit Null description is
// sent as JSON null, and an empty or null Name is rejected before HTTP because
// the typed update builder cannot send it.
type CloudServerUpdate struct {
	Name        request.Optional[string]
	Description request.Optional[string]
}

// CloudServerSearch is the SearchServers result. Inventory is every expanded
// server; Value is the filtered JSON array of their views, or the arbitrary
// JMESPath output when Expression is true (Servers is then empty).
type CloudServerSearch struct {
	Value      json.RawMessage
	Servers    []*CloudServerResult
	Inventory  []*CloudServerResult
	Expression bool
}

// CloudServerResultView renders the JSON mapping Python filters and returns
// for one server: the hostvars mapping when detailed, else the record's
// Source Resource view (or the typed server JSON) with the interface
// fields of the address expansion set over it.
func CloudServerResultView(result *CloudServerResult) (json.RawMessage, error) {
	if result == nil {
		return nil, invalid("server result is required")
	}
	if result.Hostvars != nil {
		return json.Marshal(result.Hostvars)
	}
	body, err := serverview.Base(result.Server, result.Record, resource.CloudLocation{})
	if err != nil {
		return nil, err
	}
	if err := serverview.Overlay(body, serverViewInterfaces(result.Interfaces)); err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

// CloudServerActionOpts owns the wait, address and floating-IP controls shared
// by DeleteServer and RebuildServer. Timeout defaults to Python's 180 seconds.
type CloudServerActionOpts struct {
	Wait      bool
	Timeout   time.Duration
	Unlimited bool
	// WaitOptions apply after the timeout (for example a poll interval).
	WaitOptions []resource.WaitOption

	// DeleteIPs and DeleteIPRetry map delete_ips and delete_ip_retry
	// (default one retry). FloatingIPQuery and FloatingIPDelete tune the two
	// floating-IP helpers used by DeleteIPs.
	DeleteIPs        bool
	DeleteIPRetry    *int
	FloatingIPQuery  []FloatingIPQueryOption
	FloatingIPDelete []FloatingIPDeleteOption

	// AdminPassword and Rebuild tune RebuildServer; Bare and Address expand
	// the result like the typed getters.
	AdminPassword string
	Rebuild       []servers.RebuildOption
	Bare          bool
	Address       []ServerAddressOption
	// Detailed expands the rebuild result into GetOpenStackVars hostvars.
	Detailed bool
}

// CloudServerActionOption mutates CloudServerActionOpts.
type CloudServerActionOption func(*CloudServerActionOpts) error
