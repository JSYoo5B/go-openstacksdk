package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func securityReadClient(t *testing.T, body string) (*gophercloud.ServiceClient, func() []pythonCloudBCall) {
	t.Helper()
	return pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/security-groups": {{203, body}}})
}
func securityReadField(t *testing.T, row *groups.SecurityGroupRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(row, key, want)
	}
}
func TestCloudSecurityGroupSearchDictionaryPushdownThenWholeLocalSelection(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/security-groups": {{203, `{"security_groups":[{"id":"one","name":"web","tenant_id":"tenant","shared":"false","security_group_rules":[],"x_new":{"enabled":1}}],"next":"?marker=tail"}`}, {200, `{"security_groups":[{"id":"two","name":"web","tenant_id":"tenant","shared":true,"security_group_rules":[]}]}`}}})
	result, err := network.New(client).SearchSecurityGroups(context.Background(), "one", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(`{"name":"web","project_id":"tenant","is_shared":true}`)))
	if err != nil || result == nil || len(result.Groups) != 1 || len(result.Inventory) != 2 || result.Expression || len(calls()) != 2 || calls()[0].query.Get("name") != "web" || calls()[0].query.Get("project_id") != "tenant" || calls()[0].query.Get("shared") != "True" || calls()[1].query.Get("marker") != "tail" {
		t.Fatal(result, err, calls())
	}
	securityReadField(t, result.Groups[0], "is_shared", "true")
	securityReadField(t, result.Groups[0], "project_id", `"tenant"`)
	if result.Groups[0].StatusCode != 203 || string(result.Groups[0].Wire.Body["shared"]) != `"false"` {
		t.Fatal(result)
	}
}
func TestCloudSecurityGroupSearchStringJMESHasNoServerQueryAndArbitraryJSON(t *testing.T) {
	for _, tc := range []struct{ expression, want string }{{"[].id", `["one","two"]`}, {"length(@)", "2"}, {"`null`", "null"}, {"`false`", "false"}, {"'한'", `"한"`}, {"`42`", "42"}} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/security-groups": {{200, `{"security_groups":[{"id":"one"}],"next":"?marker=tail"}`}, {200, `{"security_groups":[{"id":"two"}]}`}}})
		result, err := network.New(client).SearchSecurityGroups(context.Background(), "", network.WithSecurityGroupReadExpression(tc.expression))
		if err != nil || string(result.Value) != tc.want || !result.Expression || result.Groups != nil || len(result.Inventory) != 2 || len(calls()) != 2 || len(calls()[0].query) != 0 {
			t.Fatal(tc, result, err, calls())
		}
	}
	client, calls := securityReadClient(t, `{"security_groups":[{"id":"one"}]}`)
	result, err := network.New(client).SearchSecurityGroups(context.Background(), "absent", network.WithSecurityGroupReadExpression("[?"))
	if result == nil || len(result.Inventory) != 1 || result.Value != nil || err == nil || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		rows  string
		count int
	}{{`[]`, 0}, {`[{"id":"one","name":"web"}]`, 1}, {`[{"id":"one","name":"web"},{"id":"two","name":"web"}]`, 2}} {
		client, calls := securityReadClient(t, `{"security_groups":`+tc.rows+`}`)
		result, err := network.New(client).GetSecurityGroup(context.Background(), "web")
		if result == nil || len(calls()) != 1 || calls()[0].path != "/v2.0/security-groups" || len(calls()[0].query) != 0 {
			t.Fatal(result, err, calls())
		}
		if tc.count == 2 {
			if !errors.Is(err, resource.ErrAmbiguous) || result.Value != nil || result.Group != nil {
				t.Fatal(result, err)
			}
		} else if err != nil || tc.count == 0 && (result.Value != nil || result.Group != nil) || tc.count == 1 && (result.Group == nil || string(result.Group.Resource.Body["id"]) != `"one"`) {
			t.Fatal(result, err)
		}
	}
}
func TestCloudSecurityGroupGetExpressionFirstScalarAndFalseySemantics(t *testing.T) {
	for _, tc := range []struct {
		expression, want string
		failure          bool
	}{{"`[]`", "", false}, {"`[null]`", "", false}, {"`[false]`", "false", false}, {"`[0]`", "0", false}, {"`null`", "", false}, {"`1e-400`", "", false}, {"'한'", `"한"`, false}, {"`1`", "", true}, {"`{\"x\":1}`", "", true}, {"`[1,2]`", "", true}} {
		client, calls := securityReadClient(t, `{"security_groups":[{"id":"wire"}]}`)
		result, err := network.New(client).GetSecurityGroup(context.Background(), "", network.WithSecurityGroupReadExpression(tc.expression))
		if result == nil || tc.failure != (err != nil) || string(result.Value) != tc.want || result.Group != nil || len(calls()) != 1 {
			t.Fatal(tc, result, err, calls())
		}
	}
}
func TestCloudSecurityGroupExistingReferenceBypassesFeatureDiscoveryAndCallbacks(t *testing.T) {
	disabled := ""
	cloud := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Source: &disabled})
	for _, ref := range []any{json.RawMessage(`{"id":null,"name":"raw"}`), map[string]any{"id": false, "name": "raw"}, &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage("0")}}}} {
		callbacks := 0
		result, err := cloud.GetSecurityGroup(context.Background(), ref, func(*network.SecurityGroupReadOpts) error { callbacks++; return errors.New("must be ignored") })
		if err != nil || result == nil || result.Backend != "existing" || len(result.Value) == 0 || callbacks != 0 || len(result.Inventory) != 0 {
			t.Fatal(result, err, callbacks)
		}
		existing, handled, err := network.ExistingSecurityGroupRead(ref)
		if err != nil || !handled || existing == nil {
			t.Fatal(existing, handled, err)
		}
	}
	row := &groups.SecurityGroupRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"one"`), "name": json.RawMessage(`"raw"`)}}}, Header: http.Header{"X-Request-Id": []string{"prior"}}, Envelope: json.RawMessage(`{"actual":true}`), StatusCode: 203}
	result, err := cloud.GetSecurityGroup(context.Background(), row)
	if err != nil || result.Group == nil || result.Group.StatusCode != 203 {
		t.Fatal(result, err)
	}
	row.Resource.Body["id"][1] = 'X'
	row.Header.Set("X-Request-Id", "consumer")
	row.Envelope[0] = 'X'
	if string(result.Group.Resource.Body["id"]) != `"one"` || result.Group.Header.Get("X-Request-Id") != "prior" || string(result.Group.Envelope) != `{"actual":true}` {
		t.Fatal(result)
	}
	_, handled, err := network.ExistingSecurityGroupRead(json.RawMessage(`{"name":"not fetched"}`))
	if err != nil || handled {
		t.Fatal(handled, err)
	}
}
func TestCloudSecurityGroupDirectUUIDGetterIgnoresSearchFiltersAndKeeps404(t *testing.T) {
	const id = "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"
	for _, status := range []int{203, 404} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/security-groups/" + id: {{status, `{"security_group":{"id":"wire","name":"actual"}}`}}})
		cloud := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: network.New(client), UseDirectGet: true})
		result, err := cloud.GetSecurityGroup(context.Background(), id, network.WithSecurityGroupReadExpression("[?"))
		if result == nil || len(calls()) != 1 || len(calls()[0].query) != 0 {
			t.Fatal(result, err, calls())
		}
		if status == 404 {
			if !gophercloud.ResponseCodeIs(err, 404) || result.Value != nil {
				t.Fatal(result, err)
			}
		} else if err != nil || result.Group == nil || string(result.Group.Resource.Body["id"]) != `"wire"` {
			t.Fatal(result, err)
		}
	}
}
func TestCloudSecurityGroupOrderedNestedAndLoadedFilters(t *testing.T) {
	for _, tc := range []struct {
		filter  string
		count   int
		failure bool
	}{{`{"x_new":0,"missing":1}`, 0, false}, {`{"missing":1,"x_new":0}`, 0, true}, {`{"x_new":{"number":9007199254740992.0}}`, 1, false}, {`{"x_new":{"nested":{}}}`, 1, false}, {`{"x_new":{"nested":{"a":1}}}`, 0, true}} {
		client, calls := securityReadClient(t, `{"security_groups":[{"id":"one","x_new":{"number":9007199254740993.0,"nested":true}}]}`)
		result, err := network.New(client).SearchSecurityGroups(context.Background(), "", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(tc.filter)))
		if result == nil || tc.failure != (err != nil) || !tc.failure && len(result.Groups) != tc.count || len(calls()) != 1 {
			t.Fatal(tc, result, err, calls())
		}
		if tc.failure {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 203 || result.Value != nil {
				t.Fatal(proof, err, result)
			}
		}
	}
}
func TestCloudSecurityGroupFalseyAndNonmappingFiltersAreLocalAfterWholeList(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		failure bool
	}{{"null", false}, {"false", false}, {"0", false}, {"1e-400", false}, {`""`, false}, {"[]", false}, {"{}", false}, {"true", true}, {"[1]", true}, {"1", true}} {
		client, calls := securityReadClient(t, `{"security_groups":[{"id":"one"}]}`)
		result, err := network.New(client).SearchSecurityGroups(context.Background(), "", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(tc.raw)))
		if result == nil || tc.failure != (err != nil) || len(calls()) != 1 || len(calls()[0].query) != 0 || len(result.Inventory) != 1 {
			t.Fatal(tc, result, err, calls())
		}
	}
	client, calls := securityReadClient(t, `{"security_groups":[]}`)
	result, err := network.New(client).SearchSecurityGroups(context.Background(), "", network.WithSecurityGroupReadFiltersJSON(json.RawMessage("true")))
	if err != nil || string(result.Value) != "[]" || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudSecurityGroupProxyLiteralsAndNestedMembershipDoNotInventRows(t *testing.T) {
	for _, tc := range []struct {
		expression, name, want string
		reads                  int
		failure                bool
	}{{"`[null,false,0,[],{}]`", "", "[]", 0, false}, {"`[null]`", "id", "", 0, true}, {"to_array(@)", "", "", 1, true}, {"to_array(@)", "id", "", 0, true}, {"[*].id", "", "", 0, true}} {
		client, calls := securityReadClient(t, `{"security_groups":[{"id":"one"}]}`)
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expression})
		result, err := network.New(client).SearchSecurityGroups(context.Background(), tc.name, network.WithSecurityGroupReadFiltersJSON(raw))
		if result == nil || tc.failure != (err != nil) || string(result.Value) != tc.want || len(result.Inventory) != tc.reads || len(calls()) != tc.reads {
			t.Fatal(tc, result, err, calls())
		}
	}
}
func securityReadNova(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*gophercloud.ServiceClient, *int) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new(int)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if r.Method != "GET" || r.Header.Get("X-Auth-Token") == "" {
			t.Error(r.Method, r.Header)
		}
		w.Header().Set("X-Request-Id", "nova-actual")
		reply(w, r)
	})
	return cloud.Client("compute", "/nova/v2.1/"), calls
}

const novaSecurityGroup = `{"id":7,"name":"legacy","description":"desc","tenant_id":"old","project_id":"foreign","links":"noise","NAME_ATTR":"noise","x_new":{"kept":true},"stateful":false,"rules":[{"id":8,"parent_group_id":7,"from_port":" +٢_٢ ","to_port":"443","ip_protocol":"tcp","ip_range":{"cidr":"0.0.0.0/0"},"tenant_id":"rule-owner","x_rule":null}]}`

func TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility(t *testing.T) {
	for _, strict := range []bool{false, true} {
		client, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/nova/v2.1/os-security-groups" || r.URL.Query().Get("name") != "legacy" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 203, `{"security_groups":[`+novaSecurityGroup+`]}`)
		})
		networkAvailable := false
		source := "neutron"
		name := "configured"
		location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
		cloud := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: client, Source: &source, NetworkAvailable: &networkAvailable, Strict: strict, CurrentLocation: &location})
		result, err := cloud.SearchSecurityGroups(context.Background(), "legacy", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(`{"name":"legacy"}`)))
		if err != nil || result == nil || result.Backend != "nova" || len(result.Groups) != 1 || *count != 1 || result.NovaResponse.StatusCode != 203 {
			t.Fatal(result, err, *count)
		}
		row := result.Groups[0]
		securityReadField(t, row, "id", "7")
		securityReadField(t, row, "description", `"desc"`)
		if row.Header.Get("X-Request-Id") != "nova-actual" || string(row.Wire.Body["links"]) != `"noise"` || row.Resource.Body["links"] != nil {
			t.Fatal(row)
		}
		var rules []map[string]json.RawMessage
		if e := json.Unmarshal(row.Resource.Body["security_group_rules"], &rules); e != nil || len(rules) != 1 || string(rules[0]["port_range_min"]) != "22" || string(rules[0]["port_range_max"]) != `"443"` || string(rules[0]["remote_ip_prefix"]) != `"0.0.0.0/0"` || string(rules[0]["security_group_id"]) != "7" {
			t.Fatal(rules, e)
		}
		var current resource.CloudLocation
		if e := json.Unmarshal(row.Resource.Body["location"], &current); e != nil || string(current.Project.ID) != `"foreign"` || current.Project.Name != nil {
			t.Fatal(current, e)
		}
		if strict {
			if row.Resource.Body["project_id"] != nil || row.Resource.Body["x_new"] != nil || row.Resource.Body["stateful"] != nil || rules[0]["project_id"] != nil {
				t.Fatal(row, rules)
			}
		} else {
			securityReadField(t, row, "project_id", `"foreign"`)
			securityReadField(t, row, "stateful", "false")
			if string(rules[0]["project_id"]) != `"rule-owner"` || string(rules[0]["x_rule"]) != "null" {
				t.Fatal(rules)
			}
		}
	}
}
func TestCloudSecurityGroupNovaSourceSelectionAndNeutron404DoesNotFallback(t *testing.T) {
	for _, source := range []string{"nova", "NEUTRON", "Nova"} {
		nova, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"security_groups":[]}`) })
		client, calls := pythonCloudBClient(t, nil)
		available := true
		result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: network.New(client), Nova: nova, Source: &source, NetworkAvailable: &available}).SearchSecurityGroups(context.Background(), "")
		if err != nil || result.Backend != "nova" || *count != 1 || len(calls()) != 0 {
			t.Fatal(source, result, err, *count, calls())
		}
	}
	for _, source := range []string{"", "invalid"} {
		nova, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected Nova") })
		result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: nova, Source: &source}).SearchSecurityGroups(context.Background(), "")
		if result != nil || !errors.Is(err, network.ErrUnavailableSecurityGroups) || *count != 0 {
			t.Fatal(result, err, *count)
		}
	}
	nova, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected fallback") })
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/security-groups": {{404, `{"missing":"Neutron resource"}`}}})
	result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: network.New(client), Nova: nova}).SearchSecurityGroups(context.Background(), "")
	if result == nil || !gophercloud.ResponseCodeIs(err, 404) || len(calls()) != 1 || *count != 0 {
		t.Fatal(result, err, calls(), *count)
	}
}
func TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug(t *testing.T) {
	const id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	body := `{"security_group":{"id":7,"name":"legacy","description":"d","security_group_rules":[{"id":1,"port_range_min":"-1","from_port":50,"port_range_max":"-1","to_port":60,"remote_ip_prefix":"canonical","ip_range":{},"ip_protocol":"old","protocol":null}]}}`
	nova, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nova/v2.1/os-security-groups/"+id || len(r.URL.Query()) != 0 {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 203, body)
	})
	source := "nova"
	result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: nova, Source: &source, UseDirectGet: true}).GetSecurityGroup(context.Background(), id, network.WithSecurityGroupReadExpression("[?"))
	if err != nil || result.Group == nil || *count != 1 {
		t.Fatal(result, err, *count)
	}
	var rules []map[string]json.RawMessage
	_ = json.Unmarshal(result.Group.Resource.Body["security_group_rules"], &rules)
	var properties map[string]json.RawMessage
	_ = json.Unmarshal(rules[0]["properties"], &properties)
	if string(rules[0]["port_range_min"]) != "-1" || string(rules[0]["port_range_max"]) != `"-1"` || string(rules[0]["protocol"]) != "null" || string(properties["port_range_min"]) != `"-1"` || properties["from_port"] != nil || properties["ip_range"] != nil {
		t.Fatal(rules, properties)
	}
}
func TestCloudSecurityGroupNovaMalformedRowsKeepActualPartialProof(t *testing.T) {
	for _, bad := range []string{`{"id":2,"name":"bad"}`, `{"id":2,"name":"bad","description":"d","rules":null}`, `{"id":2,"name":"bad","description":"d","rules":[{"id":3,"remote_ip_prefix":"present","ip_range":null}]}`, `{"id":2,"name":"bad","description":"d","rules":[{"id":3,"from_port":"bad"}]}`} {
		good := `{"id":1,"name":"good","description":"d"}`
		body := `{"security_groups":[` + good + `,` + bad + `]}`
		client, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 203, body) })
		result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: client}).SearchSecurityGroups(context.Background(), "")
		var proof *resource.ResponseError
		if result == nil || len(result.Inventory) != 1 || result.Value != nil || result.Groups != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "nova-actual" || *count != 1 {
			t.Fatal(result, err, proof, *count)
		}
	}
}
func TestCloudSecurityGroupReadOwnsOptionsLocationAndExportedCarriers(t *testing.T) {
	raw := json.RawMessage(`{"name":"web"}`)
	option := network.WithSecurityGroupReadFiltersJSON(raw)
	raw[2] = 'X'
	name := "cloud"
	project := json.RawMessage(`"current"`)
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: project, Name: &name}}
	loc := network.WithSecurityGroupReadCurrentLocation(location)
	name = "caller"
	project[1] = 'X'
	client, calls := securityReadClient(t, `{"security_groups":[{"id":"one","name":"web","project_id":1e-400,"location":{"fake":true}}]}`)
	result, err := network.New(client).SearchSecurityGroups(context.Background(), "one", option, loc)
	if err != nil || len(result.Groups) != 1 || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	var current resource.CloudLocation
	_ = json.Unmarshal(result.Groups[0].Resource.Body["location"], &current)
	if current.Cloud == nil || *current.Cloud != "cloud" || string(current.Project.ID) != `"current"` || current.Project.Name == nil {
		t.Fatal(current)
	}
	original := string(result.Value)
	row := result.Groups[0]
	row.Resource.Body["id"][1] = 'X'
	row.Wire.Body["id"][1] = 'Y'
	row.Envelope[0] = 'X'
	row.Header.Set("consumer", "changed")
	if string(result.Value) != original || string(result.Inventory[0].Resource.Body["id"]) != `"one"` || string(result.LastGroup.Wire.Body["id"]) != `"one"` || result.Inventory[0].Header.Get("consumer") != "" {
		t.Fatal(result)
	}
}
func TestCloudSecurityGroupReadSourceGuardCancellationAndPreflight(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, nil)
		service := network.New(client)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionErr := errors.New("cancelled"), errors.New("option")
		result, err := service.SearchSecurityGroups(ctx, "", func(*network.SecurityGroupReadOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				client.ResourceBase += "changed/"
			}
			return optionErr
		})
		cancel(nil)
		if result != nil || !errors.Is(err, optionErr) || cancelled && !errors.Is(err, cause) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
			t.Fatal(result, err, calls())
		}
	}
	client, calls := pythonCloudBClient(t, nil)
	result, err := network.New(client).SearchSecurityGroups(nil, "")
	if result != nil || err == nil || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudSecurityGroupReadLatePhysicalPagesKeepNativeReceiptsAndStopHooks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			raw := `{"actual":"security final"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("marker") == "late" {
					w.Header().Set("Content-Length", "1000")
					w.Header().Set("X-Request-Id", "security-final")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(raw))
					return
				}
				testcloud.JSON(w, 203, `{"security_groups":[{"id":"observed"}],"next":"?marker=late"}`)
			})
			client := cloud.Client("network", "/")
			client.ResourceBase = cloud.Server.URL + "/v2.0/"
			retry, reauth, backoff := 0, 0, 0
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retry++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			result, err := network.New(client).GetSecurityGroup(context.Background(), "observed")
			if result == nil || len(result.Inventory) != 1 || result.Value != nil || result.Group != nil || calls != 2 || !errors.Is(err, io.ErrUnexpectedEOF) || retry != 0 || reauth != 0 || backoff != 0 {
				t.Fatal(result, err, calls, retry, reauth, backoff)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "security-final" {
					t.Fatal(proof, err)
				}
			} else {
				original := err
				if status == 401 {
					var unable *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &unable) || !errors.Is(unable.ErrReauth, io.ErrUnexpectedEOF) {
						t.Fatal(unable, err)
					}
					original = unable.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &native) || native.Actual != status || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "security-final" {
					t.Fatal(native, err)
				}
			}
		})
	}
}
func TestCloudSecurityGroupNovaFalseyContainersAndQueryShapes(t *testing.T) {
	for _, value := range []string{`[]`, `{}`, `""`} {
		client, count := securityReadNova(t, func(w http.ResponseWriter, r *http.Request) {
			if !reflect.DeepEqual(r.URL.Query()["unknown"], []string{"one", "two"}) {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"security_groups":`+value+`}`)
		})
		result, err := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: client}).SearchSecurityGroups(context.Background(), "", network.WithSecurityGroupReadFiltersJSON(json.RawMessage(`{"unknown":["one","two"]}`)))
		if err != nil || string(result.Value) != "[]" || *count != 1 {
			t.Fatal(result, err, *count)
		}
	}
}
