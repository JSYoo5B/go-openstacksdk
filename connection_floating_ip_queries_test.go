package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type floatingQueryState struct {
	locators, paths []string
	queries         []url.Values
	deadlines       []time.Time
	reply           func(*http.Request) (int, string)
	afterClose      func() error
	catalogError    error
}

func floatingQueryFixture(t *testing.T, source compute.FloatingIPSource) (*testcloud.Cloud, *sdk.Connection, *floatingQueryState) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &floatingQueryState{}
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		state.locators = append(state.locators, o.Type)
		if o.Type == "network" && state.catalogError != nil {
			return "", state.catalogError
		}
		if o.Type != "network" && o.Type != "compute" {
			t.Error("unexpected locator", o.Type)
		}
		return cloud.Server.URL + "/query-" + o.Type + "/", nil
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Error("query mutation", r.Method, r.URL)
		}
		state.paths = append(state.paths, r.URL.Path)
		state.queries = append(state.queries, r.URL.Query())
		deadline, _ := r.Context().Deadline()
		state.deadlines = append(state.deadlines, deadline)
		if strings.HasPrefix(r.URL.Path, "/query-compute/") && r.Header.Get("X-OpenStack-Nova-API-Version") != "2.35" {
			t.Error("version", r.Header)
		}
		if state.reply == nil {
			return nil, errors.New("missing query fixture")
		}
		code, body := state.reply(r)
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
		if state.afterClose != nil {
			reader = readyConnectionCloseBody{ReadCloser: reader, close: state.afterClose}
		}
		return &http.Response{StatusCode: code, Body: reader, Header: http.Header{"X-Query-Proof": {r.URL.Path}, "If-Match": {"actual-tag"}}, Request: r}, nil
	})
	name := "scope-name"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source)), sdk.WithCloudLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}))
	if err != nil {
		t.Fatal(err)
	}
	return cloud, conn, state
}
func queryRaw(t *testing.T, row *resource.RawResource, key, want string) {
	t.Helper()
	if row == nil || string(row.Body[key]) != want {
		t.Fatalf("%s: row=%+v want=%s", key, row, want)
	}
}
func TestFloatingIPQueryNeutronProjectionAndWireOwnership(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		return 200, `{"floatingips":[{"id":9007199254740993,"floating_ip_address":"203.0.113.10","tenant_id":"foreign","revision_number":"12","port_details":"bad","tags":"tag","self":"link","extension":{"n":9007199254740993}}]}`
	}
	result, err := conn.ListFloatingIPs(context.Background())
	if err != nil || len(result.FloatingIPs) != 1 || len(result.Pages) != 1 || result.Backend != compute.FloatingIPNeutron || !reflect.DeepEqual(state.locators, []string{"network"}) || !reflect.DeepEqual(state.paths, []string{"/query-network/v2.0/floatingips"}) {
		t.Fatal(result, err, state)
	}
	row := result.FloatingIPs[0]
	if row.Normalized {
		t.Fatal(row)
	}
	for key, want := range map[string]string{"id": "9007199254740993", "name": `"203.0.113.10"`, "project_id": `"foreign"`, "revision_number": "12", "port_details": "{}", "tags": `["tag"]`, "port_id": "null", "if_match": "null"} {
		queryRaw(t, row.Resource, key, want)
	}
	if _, ok := row.Resource.Body["self"]; ok {
		t.Fatal("self leaked")
	}
	queryRaw(t, row.Wire, "self", `"link"`)
	queryRaw(t, row.Wire, "revision_number", `"12"`)
	var location resource.CloudLocation
	if err := json.Unmarshal(row.Resource.Body["location"], &location); err != nil || string(location.Project.ID) != `"foreign"` || location.Project.Name != nil {
		t.Fatal(location, err)
	}
	row.Resource.Body["extension"][0] = '['
	row.Resource.Header.Set("X-Query-Proof", "changed")
	if string(row.Wire.Body["extension"]) != `{"n":9007199254740993}` || row.Wire.Header.Get("X-Query-Proof") == "changed" || result.Pages[0].Header.Get("X-Query-Proof") == "changed" {
		t.Fatal("unowned response")
	}
}
func TestFloatingIPQueryNeutronParametersAndLocalDescriptors(t *testing.T) {
	for _, filters := range []string{
		`{"project_id":"canonical","tenant_id":"alias","tags":["a","b"],"any_tags":"c","fields":["id","port_details"],"sort_key":false,"limit":null,"marker":[],"unknown":"discard","port_details":{"missing":null},"revision_number":12}`,
		`{"tenant_id":"alias","project_id":"canonical","tags":["a","b"],"tags-any":"c","fields":["id","port_details"],"sort_key":false,"limit":[],"marker":null,"unknown":"discard","port_details":{"missing":null},"revision_number":12}`,
	} {
		t.Run(filters, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				return 200, `{"floatingips":[{"id":"a","revision_number":"12","port_details":{"other":1}},{"id":"b","revision_number":13,"port_details":{"other":2}}]}`
			}
			result, err := conn.ListFloatingIPs(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
			want := url.Values{"project_id": {"alias"}, "tags": {"a", "b"}, "tags-any": {"c"}, "fields": {"id", "port_details"}, "sort_key": {"False"}}
			if err != nil || len(result.FloatingIPs) != 1 || !reflect.DeepEqual(state.queries, []url.Values{want}) {
				t.Fatal(result, err, state.queries)
			}
			queryRaw(t, result.FloatingIPs[0].Resource, "id", `"a"`)
		})
	}
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) { return 200, `{"floatingips":[]}` }
	_, err := conn.ListFloatingIPs(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(`{"project_id":"canonical","tenant_id":null}`)))
	if err != nil || len(state.queries[0]) != 0 {
		t.Fatal(err, state.queries)
	}
}
func TestFloatingIPQueryListFilterSpecificNotFound(t *testing.T) {
	for _, filters := range []string{"omitted", "{}", "null", "false", "0", "[]", `{"unknown":"discard"}`, `{"status":"ACTIVE"}`} {
		t.Run(filters, func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "network") {
					return 404, `{"missing":"neutron"}`
				}
				return 200, `{"floating_ips":[{"id":7,"ip":"198.51.100.4","instance_id":"instance"}]}`
			}
			var opts []compute.FloatingIPQueryOption
			if filters != "omitted" {
				opts = append(opts, compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
			}
			result, err := conn.ListFloatingIPs(context.Background(), opts...)
			filtered := strings.Contains(filters, "unknown") || strings.Contains(filters, "status")
			if err != nil || result.Failure == nil || result.Failure.StatusCode != 404 {
				t.Fatal(result, err)
			}
			if filtered {
				if result.Backend != compute.FloatingIPNeutron || string(result.Value) != "[]" || result.SuppressedNotFound == nil || len(state.paths) != 1 {
					t.Fatal(result, state)
				}
			} else {
				if result.Backend != compute.FloatingIPNova || result.FallbackError == nil || len(result.FloatingIPs) != 1 || len(state.paths) != 2 {
					t.Fatal(result, state)
				}
				row := result.FloatingIPs[0]
				if row.NormalizationSource != compute.FloatingIPNeutron || !row.Normalized {
					t.Fatal(row)
				}
				queryRaw(t, row.Resource, "status", `"UNKNOWN"`)
				queryRaw(t, row.Resource, "attached", "false")
			}
		})
	}
}
func TestFloatingIPQueryNovaAndNonePermissiveRowsAndNotFound(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		t.Run(string(source), func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, source)
			state.reply = func(r *http.Request) (int, string) {
				return 200, `{"floating_ips":[{"id":null,"ip":null,"status":"ERROR","instance_id":[]},{"id":false,"ip":"not-an-ip","pool":null}]}`
			}
			result, err := conn.ListFloatingIPs(context.Background())
			if err != nil || len(result.FloatingIPs) != 2 || !reflect.DeepEqual(state.locators, []string{"compute"}) {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.FloatingIPs[0].Resource, "status", `"ACTIVE"`)
			queryRaw(t, result.FloatingIPs[0].Wire, "status", `"ERROR"`)
			state.reply = func(r *http.Request) (int, string) { return 404, `{"missing":"nova"}` }
			result, err = conn.ListFloatingIPs(context.Background())
			if err != nil || string(result.Value) != "[]" || result.SuppressedNotFound == nil || result.Failure.StatusCode != 404 {
				t.Fatal(result, err)
			}
		})
	}
}
func TestFloatingIPQueryListRejectsFiltersBeforeNovaDiscovery(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone, compute.FloatingIPNeutron} {
		t.Run(string(source), func(t *testing.T) {
			_, conn, state := floatingQueryFixture(t, source)
			for _, filters := range []string{`"expression"`, `[1]`, `{"status":"ACTIVE"}`} {
				if source == compute.FloatingIPNeutron && strings.HasPrefix(filters, "{") {
					continue
				}
				_, err := conn.ListFloatingIPs(context.Background(), compute.WithFloatingIPQueryFilters(json.RawMessage(filters)))
				if !errors.Is(err, resource.ErrInvalidOption) || len(state.paths) != 0 || (source != compute.FloatingIPNeutron && len(state.locators) != 0) {
					t.Fatal(err, state)
				}
			}
		})
	}
}
func TestFloatingIPQueryNeutronDictionarySearchIgnoresID(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) { return 200, `{"floatingips":[{"id":"a"},{"id":"b"}]}` }
	result, err := conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{ID: "absent"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{}`)))
	if err != nil || len(result.FloatingIPs) != 2 || len(state.paths) != 1 {
		t.Fatal(result, err, state)
	}
	_, err = conn.GetFloatingIP(context.Background(), compute.GetFloatingIPRequest{ID: "absent"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{}`)))
	var multiple *compute.FloatingIPSelectionError
	if !errors.As(err, &multiple) || multiple.Length != 2 || !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	state.reply = func(r *http.Request) (int, string) { return 404, `{"missing":true}` }
	result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{ID: "a"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{}`)))
	if !gophercloud.ResponseCodeIs(err, 404) || result.FallbackError != nil || len(state.locators) != 1 || result.Failure.StatusCode != 404 {
		t.Fatal(result, err, state)
	}
}
func TestFloatingIPQueryLocalSearchAndArbitraryExpressions(t *testing.T) {
	_, conn, state := floatingQueryFixture(t, compute.FloatingIPNova)
	state.reply = func(r *http.Request) (int, string) {
		return 200, `{"floating_ips":[{"id":"a","ip":"one","vendor":1},{"id":"b","ip":"two","vendor":2}]}`
	}
	result, err := conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{ID: "b*"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{"vendor":2}`)))
	if err != nil || len(result.FloatingIPs) != 1 {
		t.Fatal(result, err)
	}
	queryRaw(t, result.FloatingIPs[0].Resource, "id", `"b"`)
	for expression, want := range map[string]string{"[].id": `["a","b"]`, "length(@)": "2", "`false`": "false", "`null`": "null", "{ids: [].id}": `{"ids":["a","b"]}`} {
		result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{}, compute.WithFloatingIPQueryExpression(expression))
		if err != nil || result.FloatingIPs != nil || string(result.Value) != want || len(result.Pages) != 1 {
			t.Fatal(expression, result, err)
		}
	}
	result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{ID: "absent"}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{"missing":{"nested":1}}`)))
	if err != nil || string(result.Value) != "[]" {
		t.Fatal(result, err)
	}
	result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{}, compute.WithFloatingIPQueryFilters(json.RawMessage(`{"missing":{"nested":1}}`)))
	if err == nil || !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.FloatingIPs != nil {
		t.Fatal(result, err)
	}
	result, err = conn.SearchFloatingIPs(context.Background(), compute.SearchFloatingIPsRequest{}, compute.WithFloatingIPQueryExpression("["))
	if err == nil || result.Value != nil || len(result.Pages) != 1 {
		t.Fatal(result, err)
	}
}
