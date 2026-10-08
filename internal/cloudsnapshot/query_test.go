package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func snapshotQueryContractRaw(text string) *json.RawMessage {
	raw := json.RawMessage(text)
	return &raw
}
func snapshotQueryContractCompile(t *testing.T, options ListOptions) *listPolicy {
	t.Helper()
	policy, err := compileList(options)
	if err != nil || policy == nil {
		t.Fatal(policy, err)
	}
	return policy
}
func snapshotQueryContractValues(t *testing.T, policy *listPolicy) url.Values {
	t.Helper()
	values, err := encodeQuery(policy.query)
	if err != nil {
		t.Fatal(err)
	}
	return values
}

func TestSnapshotListCompileSplitsServerQueryFromOrderedBodyPredicates(t *testing.T) {
	policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"name":"server-name","metadata":{"owner":1},"description":"keep","is_forced":1,"id":null,"size":3,"project_id":[[1,2],[],null],"status":false,"volume_id":{"z":1,"a":2},"offset":0,"sort_dir":["asc","desc"],"sort_key":"size","sort":"name:asc","force":"ignored wire alias","os-extended-snapshot-attributes:progress":"ignored wire alias","unknown":"ignored","location":{"ignored":true}}`)})
	expected := url.Values{"name": {"server-name"}, "project_id": {"1", "2"}, "status": {"False"}, "volume_id": {"z", "a"}, "offset": {"0"}, "sort_dir": {"asc", "desc"}, "sort_key": {"size"}, "sort": {"name:asc"}}
	if got := snapshotQueryContractValues(t, policy); !reflect.DeepEqual(got, expected) {
		t.Fatal(got, expected)
	}
	var keys []string
	for _, predicate := range policy.local {
		keys = append(keys, predicate.Key)
	}
	if !reflect.DeepEqual(keys, []string{"metadata", "description", "is_forced", "id", "size"}) {
		t.Fatal(keys)
	}
	matched, err := matchLocal(context.Background(), json.RawMessage(`{"name":"unrelated","project_id":"foreign","metadata":{"owner":true,"extra":"kept"},"description":"keep","is_forced":true,"size":3}`), policy.local, nil)
	if err != nil || !matched {
		t.Fatal("local predicates coerced or server fields rechecked", matched, err)
	}
}

func TestSnapshotListAllProjectsIsBoundBeforeLateAliasOverlay(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`{"all_projects":false}`, ""}, {`{"all_projects":null}`, ""}, {`{"all_projects":[]}`, ""},
		{`{"all_projects":false,"all_tenants":"wire"}`, "wire"},
		{`{"all_projects":"False","all_tenants":false}`, "True"},
		{`{"all_projects":{"truthy":1},"all_tenants":"wire"}`, "True"},
		{`{"all_projects":true,"all_tenants":"wire","__conflicting_attrs":{"all_projects":false}}`, "False"},
		{`{"all_tenants":"wire","__conflicting_attrs":{"all_projects":null}}`, ""},
		{`{"all_projects":true,"__conflicting_attrs":{"all_tenants":"late-wire"}}`, "True"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(tc.raw)})
			values := snapshotQueryContractValues(t, policy)
			if values.Get("all_tenants") != tc.want || len(policy.local) != 0 {
				t.Fatal(values, policy.local)
			}
		})
	}
}

func TestSnapshotListTypedControlsOverrideFinalRawValuesButKeepProxyStage(t *testing.T) {
	detailed, paginated, maximum, version, expression, allowUnknown := false, false, 0, "", "", false
	policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"paginated":"false","jmespath_filters":17,"max_items":"wrong-original","microversion":false,"headers":[1],"name":"original","__conflicting_attrs":{"max_items":2.5,"microversion":"3.99","headers":{"X-Probe":"late"},"name":"late","jmespath_filters":"ignored-late"}}`), Detailed: &detailed, Paginated: &paginated, MaxItems: &maximum, Microversion: &version, Headers: map[string]string{}, Expression: &expression, AllowUnknownParams: &allowUnknown})
	if policy.detailed || policy.paginated || string(policy.maximum) != "0" || policy.limit != nil || policy.microversion == nil || *policy.microversion != "" || policy.expression == nil || *policy.expression != "" || !reflect.DeepEqual(policy.headers, map[string]string{"Accept": "application/json"}) || snapshotQueryContractValues(t, policy).Get("name") != "late" {
		t.Fatal(policy)
	}
	plain := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"paginated":"false","jmespath_filters":"[].id","microversion":null,"headers":{"accept":"application/custom","X-Probe":"entry"},"__conflicting_attrs":{"details":false,"resource_type":false,"self":false,"jmespath_filters":"ignored-late","microversion":"3.99","max_items":1.5,"allow_unknown_params":false}}`)})
	if !plain.detailed || !plain.paginated || plain.expression == nil || *plain.expression != "[].id" || plain.microversion == nil || *plain.microversion != "3.99" || string(plain.maximum) != "1.5" || string(plain.limit) != "1.5" || plain.headers["Accept"] != "application/custom" || plain.headers["X-Probe"] != "entry" {
		t.Fatal(plain)
	}
}

func TestSnapshotListControlCollisionsDependOnActualBindingStage(t *testing.T) {
	paginated := false
	for _, key := range []string{"details", "base_path", "resource_type", "self", "session", "cls"} {
		encoded, _ := json.Marshal(map[string]any{key: nil})
		raw := json.RawMessage(encoded)
		policy, err := compileList(ListOptions{Filters: &raw, Paginated: &paginated})
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, policy, err)
		}
	}
	for _, key := range []string{"paginated", "base_path", "session", "cls"} {
		encoded, _ := json.Marshal(map[string]any{"__conflicting_attrs": map[string]any{key: nil}})
		raw := json.RawMessage(encoded)
		policy, err := compileList(ListOptions{Filters: &raw, Paginated: &paginated})
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("late", key, policy, err)
		}
	}
	policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"__conflicting_attrs":{"details":false,"resource_type":"ignored","self":"ignored","jmespath_filters":17,"__conflicting_attrs":{"base_path":"not-recursive"},"description":"selected","unknown":17}}`)})
	matched, err := matchLocal(context.Background(), json.RawMessage(`{"description":"selected"}`), policy.local, nil)
	if err != nil || !matched || !policy.detailed || !policy.paginated || policy.expression != nil || len(policy.query) != 0 {
		t.Fatal(policy, matched, err)
	}
}

func TestSnapshotListCompileRejectsMalformedRawAndUnsupportedDynamicControlsAtomically(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage{}, json.RawMessage(" "), json.RawMessage(`{} []`), json.RawMessage(`{"x":}`), json.RawMessage([]byte{'"', 0xff, '"'}), json.RawMessage(`true`), json.RawMessage(`[1]`), json.RawMessage(`"truthy"`)} {
		policy, err := compileList(ListOptions{Filters: &raw})
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(string(raw), policy, err)
		}
	}
	for _, raw := range []string{`{"__conflicting_attrs":true}`, `{"__conflicting_attrs":[1]}`, `{"microversion":false}`, `{"headers":false}`, `{"headers":{"X-Probe":null}}`, `{"headers":{"X-Probe":1}}`, `{"headers":{"X-Probe":"a","x-probe":"b"}}`, `{"max_items":"2"}`, `{"max_items":[]}`, `{"jmespath_filters":true}`} {
		policy, err := compileList(ListOptions{Filters: snapshotQueryContractRaw(raw)})
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(raw, policy, err)
		}
	}
	raw := json.RawMessage{}
	policy, err := compileList(ListOptions{ConflictingAttrs: &raw})
	if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(policy, err)
	}
	bad := string([]byte{0xff})
	for _, options := range []ListOptions{{Microversion: &bad}, {Expression: &bad}, {Headers: map[string]string{"X-Probe": bad}}, {Headers: map[string]string{bad: "value"}}} {
		policy, err := compileList(options)
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(policy, err)
		}
	}
}

func TestSnapshotListFalseyFiltersAndLegacyExpressionKeepExplicitProjectionSeparate(t *testing.T) {
	defaults := snapshotQueryContractCompile(t, ListOptions{})
	if !defaults.detailed || !defaults.paginated || len(defaults.query) != 0 || len(defaults.local) != 0 || defaults.expression != nil || defaults.maximum != nil || defaults.microversion != nil || !reflect.DeepEqual(defaults.headers, map[string]string{"Accept": "application/json"}) {
		t.Fatal(defaults)
	}
	for _, raw := range []string{`null`, `false`, `0`, `-0.0e999999`, `[]`, `{}`, `""`} {
		policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(raw)})
		if !policy.detailed || !policy.paginated || len(policy.query) != 0 || len(policy.local) != 0 || policy.expression != nil || policy.maximum != nil || policy.microversion != nil || !reflect.DeepEqual(policy.headers, map[string]string{"Accept": "application/json"}) {
			t.Fatal(raw, policy)
		}
		controls := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"jmespath_filters":` + raw + `,"__conflicting_attrs":` + raw + `,"allow_unknown_params":` + raw + `,"unknown":"source-dropped"}`)})
		if controls.expression != nil || len(controls.query) != 0 || len(controls.local) != 0 {
			t.Fatal(raw, controls)
		}
	}
	empty := ""
	policy := snapshotQueryContractCompile(t, ListOptions{Expression: &empty})
	if policy.expression == nil || *policy.expression != "" {
		t.Fatal("typed empty expression must reach engine rather than disappearing", policy)
	}
	for _, raw := range []string{`null`, `false`, `0`, `[]`, `""`} {
		policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"paginated":` + raw + `}`)})
		if policy.paginated {
			t.Fatal(raw, policy)
		}
	}
	ignored := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(`{"allow_unknown_params":{"truthy":"still unused"},"unknown":"source-dropped"}`)})
	if len(ignored.query) != 0 || len(ignored.local) != 0 {
		t.Fatal("allow_unknown no-op changed unknown query behavior", ignored)
	}
}

func TestSnapshotListCompiledPolicyOwnsRawQueryLocalControlsAndHeadersIndependently(t *testing.T) {
	raw := json.RawMessage(`{"limit":0,"max_items":2.5,"description":{"nested":true},"headers":{"X-Probe":"original"},"microversion":"3.60"}`)
	before := bytes.Clone(raw)
	policy := snapshotQueryContractCompile(t, ListOptions{Filters: &raw})
	if !bytes.Equal(raw, before) || string(policy.limit) != "2.5" || string(policy.maximum) != "2.5" {
		t.Fatal(policy)
	}
	for i := range raw {
		raw[i] = '!'
	}
	if policy.headers["X-Probe"] != "original" || *policy.microversion != "3.60" || string(policy.local[0].Value) != `{"nested":true}` || snapshotQueryContractValues(t, policy).Get("limit") != "2.5" {
		t.Fatal("input aliases compiled policy", policy)
	}
	policy.query["limit"][0] = '9'
	if string(policy.limit) != "2.5" || string(policy.maximum) != "2.5" {
		t.Fatal("query limit aliases policy maximum or retained limit")
	}
	policy.limit[0] = '8'
	if string(policy.maximum) != "2.5" {
		t.Fatal("retained limit aliases maximum")
	}
	repeated := snapshotQueryContractCompile(t, ListOptions{Filters: (func() *json.RawMessage { copy := json.RawMessage(bytes.Clone(before)); return &copy })()})
	policy.headers["X-Probe"] = "changed"
	(*policy.microversion) = "3.99"
	policy.local[0].Value[0] = '!'
	if repeated.headers["X-Probe"] != "original" || *repeated.microversion != "3.60" || string(repeated.local[0].Value) != `{"nested":true}` {
		t.Fatal("independent compiles share policy", repeated)
	}
}

func TestSnapshotListQueryEncoderUsesTwoIterableLevelsAndFailsWithoutPartialValues(t *testing.T) {
	query := map[string]json.RawMessage{"project_id": json.RawMessage(`[[1,2],[],null,[null],{"z":1,"a":2},[[null,false]]]`), "status": json.RawMessage(`false`), "name": json.RawMessage(`"a /?&= Ω"`), "omit": json.RawMessage(`null`), "empty": json.RawMessage(`""`), "large": json.RawMessage(`900719925474099312345`)}
	values, err := encodeQuery(query)
	expected := url.Values{"project_id": {"1", "2", "None", "z", "a", "[None, False]"}, "status": {"False"}, "name": {"a /?&= Ω"}, "empty": {""}, "large": {"900719925474099312345"}}
	if err != nil || !reflect.DeepEqual(values, expected) {
		t.Fatal(values, err)
	}
	values["project_id"][0] = "mutated"
	again, err := encodeQuery(query)
	if err != nil || !reflect.DeepEqual(again, expected) {
		t.Fatal("encoded values reused earlier storage", again, err)
	}
	for _, bad := range []map[string]json.RawMessage{{"good": json.RawMessage(`1`), "bad": json.RawMessage(``)}, {"good": json.RawMessage(`1`), "bad": json.RawMessage(`{} []`)}, {string([]byte{0xff}): json.RawMessage(`1`)}} {
		got, err := encodeQuery(bad)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(got, err)
		}
	}
}

func TestSnapshotListMaximumKeepsBooleanFractionNegativeAndHugeExponentCounterSemantics(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		count int
		want  bool
	}{
		{"null", 99, false}, {"false", 99, false}, {"0", 99, false}, {"-0e99999999999999999999999999", 99, false},
		{"true", 0, false}, {"true", 1, true}, {"1.5", 1, false}, {"1.5", 2, true}, {"1.0000000000000000000001", 1, false}, {"1.0000000000000000000001", 2, true},
		{"-0.1", 0, true}, {"-1", 0, true}, {"1e99999999999999999999999999", 99, false}, {"1e-99999999999999999999999999", 1, true}, {"10.00", 9, false}, {"100e-1", 10, true}, {"9.999999999999999999", 10, true},
	} {
		got, err := maxReached(tc.count, json.RawMessage(tc.raw))
		if err != nil || got != tc.want {
			t.Fatal(tc, got, err)
		}
	}
	if got, err := maxReached(99, nil); err != nil || got {
		t.Fatal(got, err)
	}
	for _, raw := range []json.RawMessage{json.RawMessage{}, json.RawMessage(`"2"`), json.RawMessage(`[]`), json.RawMessage(`{}`), json.RawMessage(`1 true`)} {
		got, err := maxReached(0, raw)
		if got || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(string(raw), got, err)
		}
	}
	for _, tc := range []struct{ raw, limit string }{
		{`{"max_items":true}`, "true"}, {`{"max_items":-1}`, "-1"}, {`{"max_items":1.5,"limit":null}`, "1.5"}, {`{"max_items":2,"limit":[]}`, "2"}, {`{"max_items":2,"limit":false}`, "2"}, {`{"max_items":2,"limit":""}`, "2"}, {`{"max_items":2,"limit":3}`, "3"},
	} {
		policy := snapshotQueryContractCompile(t, ListOptions{Filters: snapshotQueryContractRaw(tc.raw)})
		if string(policy.limit) != tc.limit {
			t.Fatal(tc, policy)
		}
	}
}
