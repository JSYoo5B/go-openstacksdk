package compute

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func queryWire(t *testing.T, text string) *resource.RawResource {
	t.Helper()
	var wire resource.RawResource
	if err := json.Unmarshal([]byte(text), &wire); err != nil {
		t.Fatal(err)
	}
	wire.Header, wire.StatusCode = http.Header{"X-Query-Proof": {"original"}}, 200
	return &wire
}
func queryField(t *testing.T, fields map[string]json.RawMessage, name, want string) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal(fields[name], &actual); err != nil {
		t.Fatal(name, err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal(name, string(fields[name]), want)
	}
}

func TestFloatingIPQueryNormalizationCanonicalNullPropertiesAndStrictAliases(t *testing.T) {
	for _, strict := range []bool{false, true} {
		wire := queryWire(t, `{"id":9007199254740993,"fixed_ip_address":null,"fixed_ip":"legacy","floating_ip_address":null,"ip":"legacy","floating_network_id":null,"network":"legacy","pool":"public","project_id":null,"tenant_id":"foreign","instance_id":"server","port_id":"","router_id":null,"description":null,"status":"ERROR","attached":false,"owner":"other-owner","vendor":{"nested":null}}`)
		name := "current"
		location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
		got, err := normalizeQueryNovaIP(wire, false, strict, location)
		if err != nil || got.Backend != FloatingIPNova || got.NormalizationSource != FloatingIPNova || !got.Normalized {
			t.Fatal(got, err)
		}
		fields := got.Resource.Body
		if string(fields["id"]) != "9007199254740993" {
			t.Fatal(string(fields["id"]))
		}
		for _, key := range []string{"fixed_ip_address", "floating_ip_address", "network", "router", "description", "revision_number", "created_at", "updated_at"} {
			queryField(t, fields, key, "null")
		}
		queryField(t, fields, "status", `"ACTIVE"`)
		queryField(t, fields, "attached", "true")
		queryField(t, fields, "location", `{"cloud":null,"region_name":null,"zone":null,"project":{"id":"scope","name":"current","domain_id":null,"domain_name":null}}`)
		queryField(t, fields, "properties", `{"status":"ERROR","attached":false,"owner":"other-owner","vendor":{"nested":null}}`)
		_, alias := fields["project_id"]
		_, vendor := fields["vendor"]
		if alias == strict || vendor == strict {
			t.Fatal(strict, fields)
		}
		if !strict {
			queryField(t, fields, "project_id", "null")
			queryField(t, fields, "tenant_id", "null")
			queryField(t, fields, "floating_network_id", "null")
		}
		for _, key := range []string{"fixed_ip", "ip", "pool", "instance_id"} {
			if _, exists := fields[key]; exists {
				t.Fatal(key, fields)
			}
		}
		if wire.Header.Get("X-Query-Proof") != "original" || got.Resource.StatusCode != 200 || got.Wire.StatusCode != 200 {
			t.Fatal(got)
		}
		got.Resource.Header.Set("X-Query-Proof", "view")
		got.Wire.Body["id"][0] = '8'
		if wire.Header.Get("X-Query-Proof") != "original" || string(wire.Body["id"]) != "9007199254740993" || string(got.Resource.Body["id"]) != "9007199254740993" {
			t.Fatal(wire, got)
		}
	}
}

func TestFloatingIPQueryNormalizationUsesConfiguredNeutronAfterNovaFallback(t *testing.T) {
	for _, status := range []string{"missing", "null", "value"} {
		wire := queryWire(t, `{"id":12,"ip":"198.51.100.12","pool":"public","instance_id":"server","port_id":[],"project_id":"foreign","tenant_id":"ignored","owner":"also-ignored"}`)
		wantStatus := `"UNKNOWN"`
		if status != "missing" {
			wantStatus = "null"
			if status == "value" {
				wantStatus = `"DOWN"`
			}
			wire.Body["status"] = json.RawMessage(wantStatus)
		}
		name := "current"
		got, err := normalizeQueryNovaIP(wire, true, false, resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}})
		if err != nil || got.NormalizationSource != FloatingIPNeutron || got.Backend != FloatingIPNova {
			t.Fatal(got, err)
		}
		queryField(t, got.Resource.Body, "attached", "false")
		queryField(t, got.Resource.Body, "status", wantStatus)
		queryField(t, got.Resource.Body, "location", `{"cloud":null,"region_name":null,"zone":null,"project":{"id":"foreign","name":null,"domain_id":null,"domain_name":null}}`)
		queryField(t, got.Resource.Body, "properties", `{"owner":"also-ignored"}`)
	}
}

func TestFloatingIPQueryNormalizationReadRowsAreNotMutationCandidates(t *testing.T) {
	for _, raw := range []string{`{"id":false,"floating_ip_address":{},"pool":[]}`, `{"id":null}`, `{"id":1.5,"ip":null}`, `{"id":"unsafe/id","pool":""}`} {
		wire := queryWire(t, raw)
		got, err := normalizeQueryNovaIP(wire, false, true, resource.CloudLocation{})
		if err != nil || got == nil || string(got.Resource.Body["id"]) != string(wire.Body["id"]) {
			t.Fatal(raw, got, err)
		}
	}
	if _, err := normalizeQueryNovaIP(queryWire(t, `{"ip":"198.51.100.10"}`), false, false, resource.CloudLocation{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestFloatingIPQueryOptionsOwnBulkJSONSourceAndLocation(t *testing.T) {
	filters := json.RawMessage(`{"status":"ACTIVE"}`)
	source := FloatingIPNova
	name := "current"
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
	option := WithFloatingIPQueryOptions(FloatingIPQueryOpts{Filters: &filters, Source: &source, Location: &location, Strict: true})
	filters[2], source, name, location.Project.ID[1] = 'x', FloatingIPNone, "changed", 'x'
	got, err := PrepareFloatingIPQueryOptions(context.Background(), option)
	if err != nil || *got.Source != FloatingIPNova || string(*got.Filters) != `{"status":"ACTIVE"}` || *got.Location.Project.Name != "current" || string(got.Location.Project.ID) != `"scope"` || !got.Strict {
		t.Fatal(got, err)
	}
	(*got.Filters)[2] = 'x'
	got.Location.Project.ID[1] = 'x'
	again, err := PrepareFloatingIPQueryOptions(context.Background(), option)
	if err != nil || string(*again.Filters) != `{"status":"ACTIVE"}` || string(again.Location.Project.ID) != `"scope"` {
		t.Fatal(again, err)
	}
}

func TestFloatingIPQueryOptionsPreflightAndUnlimitedDefaults(t *testing.T) {
	for _, options := range [][]FloatingIPQueryOption{{nil}, {WithFloatingIPQuerySource("invalid")}, {WithFloatingIPQueryTimeout(0)}, {WithFloatingIPQueryOptions(FloatingIPQueryOpts{Timeout: -1})}} {
		if _, err := PrepareFloatingIPQueryOptions(context.Background(), options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	value, err := PrepareFloatingIPQueryOptions(context.Background(), WithFloatingIPQueryTimeout(5), WithUnlimitedFloatingIPQueryTimeout())
	if err != nil || value.Timeout != 0 {
		t.Fatal(value, err)
	}
	if _, err := PrepareFloatingIPQueryOptions(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller stopped")
	cancel(cause)
	if _, err := PrepareFloatingIPQueryOptions(ctx); !errors.Is(err, cause) {
		t.Fatal(err)
	}
}
