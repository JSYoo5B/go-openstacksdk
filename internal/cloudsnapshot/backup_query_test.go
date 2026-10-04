package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"gophercloudsdk/resource"
)

func backupQueryContractRaw(text string) *json.RawMessage {
	value := json.RawMessage(text)
	return &value
}
func backupQueryContractCompile(t *testing.T, value ListOptions) *listPolicy {
	t.Helper()
	policy, err := compileListFor(value, backupReadSchema())
	if err != nil || policy == nil {
		t.Fatal(policy, err)
	}
	return policy
}
func backupQueryContractValues(t *testing.T, value *listPolicy) url.Values {
	t.Helper()
	values, err := encodeQuery(value.query)
	if err != nil {
		t.Fatal(values, err)
	}
	return values
}
func backupQueryContractLocalKeys(value *listPolicy) []string {
	keys := make([]string, len(value.local))
	for i, predicate := range value.local {
		keys[i] = predicate.Key
	}
	return keys
}

func TestBackupListAllProjectsPreservesRawValuesInsteadOfSnapshotProxyTruthiness(t *testing.T) {
	for _, tc := range []struct {
		raw              string
		backup, snapshot []string
	}{
		{`false`, []string{"False"}, nil}, {`0`, []string{"0"}, nil}, {`null`, nil, nil}, {`[]`, nil, nil}, {`{}`, nil, nil},
		{`"false"`, []string{"false"}, []string{"True"}}, {`[false,1]`, []string{"False", "1"}, []string{"True"}},
		{`{"z":1,"a":2}`, []string{"z", "a"}, []string{"True"}}, {`true`, []string{"True"}, []string{"True"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			options := ListOptions{Filters: backupQueryContractRaw(`{"all_projects":` + tc.raw + `}`)}
			backup := backupQueryContractCompile(t, options)
			snapshot, err := compileList(options)
			if err != nil || snapshot == nil {
				t.Fatal(snapshot, err)
			}
			if string(backup.query["all_tenants"]) != tc.raw || len(backup.local) != 0 || !reflect.DeepEqual(backupQueryContractValues(t, backup)["all_tenants"], tc.backup) || !reflect.DeepEqual(backupQueryContractValues(t, snapshot)["all_tenants"], tc.snapshot) {
				t.Fatal(backup.query, snapshot.query, tc)
			}
			if _, leaked := backup.query["all_projects"]; leaked {
				t.Fatal("client alias reached physical query", backup.query)
			}
		})
	}
}

func TestBackupListCanonicalAllProjectsPresenceWinsWireAliasEvenWhenFalsey(t *testing.T) {
	for _, tc := range []struct {
		raw              string
		backup, snapshot []string
	}{
		{`{"all_projects":false,"all_tenants":"wire"}`, []string{"False"}, []string{"wire"}},
		{`{"all_projects":null,"all_tenants":"wire"}`, nil, []string{"wire"}},
		{`{"all_projects":[],"all_tenants":"wire"}`, nil, []string{"wire"}},
		{`{"all_projects":"false","all_tenants":"wire"}`, []string{"false"}, []string{"True"}},
		{`{"all_tenants":0}`, []string{"0"}, []string{"0"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			options := ListOptions{Filters: backupQueryContractRaw(tc.raw)}
			backup := backupQueryContractCompile(t, options)
			snapshot, err := compileList(options)
			if err != nil || snapshot == nil {
				t.Fatal(snapshot, err)
			}
			if !reflect.DeepEqual(backupQueryContractValues(t, backup)["all_tenants"], tc.backup) || !reflect.DeepEqual(backupQueryContractValues(t, snapshot)["all_tenants"], tc.snapshot) || len(backup.local) != 0 {
				t.Fatal(backup.query, snapshot.query, tc)
			}
		})
	}
}

func TestBackupListLateAliasOverlayDoesNotPassThroughEitherProxyAgain(t *testing.T) {
	for _, tc := range []struct {
		raw, canonical string
		encoded        []string
	}{
		{`{"all_projects":true,"all_tenants":"wire","__conflicting_attrs":{"all_projects":false}}`, `false`, []string{"False"}},
		{`{"all_projects":true,"__conflicting_attrs":{"all_projects":[null,false]}}`, `[null,false]`, []string{"False"}},
		{`{"all_tenants":"wire","__conflicting_attrs":{"all_projects":null}}`, `null`, nil},
		{`{"__conflicting_attrs":{"all_projects":"false","all_tenants":"wire"}}`, `"false"`, []string{"false"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			options := ListOptions{Filters: backupQueryContractRaw(tc.raw)}
			for _, schema := range []readSchema{backupReadSchema(), snapshotReadSchema()} {
				policy, err := compileListFor(options, schema)
				if err != nil || policy == nil || string(policy.query["all_tenants"]) != tc.canonical || !reflect.DeepEqual(backupQueryContractValues(t, policy)["all_tenants"], tc.encoded) || len(policy.local) != 0 {
					t.Fatal(schema.kind, policy, err)
				}
			}
		})
	}
}

func TestBackupListQueryUsesItsOwnBodyPredicatesAndPreservesServerOnlyParameters(t *testing.T) {
	raw := backupQueryContractRaw(`{"name":"server-name","project_id":[[1,2],[],null],"status":false,"volume_id":{"z":1,"a":2},"offset":0,"sort_dir":["asc","desc"],"sort_key":"size","sort":"name:asc","force":true,"has_dependent_backups":false,"is_incremental":true,"links":[{"href":"next"}],"object_count":3,"size":3,"snapshot_id":null,"availability_zone":"wire-zone","container":"container","metadata":{"owner":1},"is_forced":"unused snapshot","consumes_quota":false,"group_snapshot_id":"unused snapshot","progress":false,"os-backup-project-attr:project_id":"unused wire alias","unknown":"ignored","location":{"ignored":true}}`)
	policy := backupQueryContractCompile(t, ListOptions{Filters: raw})
	values := url.Values{"name": {"server-name"}, "project_id": {"1", "2"}, "status": {"False"}, "volume_id": {"z", "a"}, "offset": {"0"}, "sort_dir": {"asc", "desc"}, "sort_key": {"size"}, "sort": {"name:asc"}}
	if got := backupQueryContractValues(t, policy); !reflect.DeepEqual(got, values) {
		t.Fatal(got, values)
	}
	expected := []string{"force", "has_dependent_backups", "is_incremental", "links", "object_count", "size", "snapshot_id", "availability_zone", "container", "metadata"}
	if got := backupQueryContractLocalKeys(policy); !reflect.DeepEqual(got, expected) {
		t.Fatal(got, expected)
	}
	view, _, err := normalizeBackup(json.RawMessage(`{"name":"unrelated","project_id":"unrelated","status":"error","force":"false","has_dependent_backups":{},"is_incremental":[1],"links":{"href":"next"},"object_count":3.9,"size":"003","snapshot_id":null,"availability_zone":"wire-zone","container":"container","metadata":{"owner":true,"extra":null}}`), nil, true, resource.CloudLocation{})
	if err != nil {
		t.Fatal(string(view), err)
	}
	matched, err := matchLocal(context.Background(), view, policy.local, nil)
	if err != nil || !matched {
		t.Fatal("server fields rechecked, predicate values coerced, or backup descriptors misplaced", string(view), matched, err)
	}
	var normalized map[string]json.RawMessage
	if err := json.Unmarshal(view, &normalized); err != nil {
		t.Fatal(err)
	}
	if _, leaked := normalized["is_forced"]; leaked {
		t.Fatal("snapshot normalized field entered backup view", string(view))
	}
}

func TestBackupListCallerPredicatesAreNeverCoercedLikeBackupResponseDescriptors(t *testing.T) {
	view, _, err := normalizeBackup(json.RawMessage(`{"force":"false","has_dependent_backups":0,"is_incremental":[1]}`), nil, true, resource.CloudLocation{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		raw   string
		match bool
	}{
		{`{"force":true}`, true}, {`{"force":1}`, true}, {`{"force":"false"}`, false},
		{`{"has_dependent_backups":false}`, true}, {`{"has_dependent_backups":null}`, false},
		{`{"is_incremental":true}`, true}, {`{"is_incremental":[1]}`, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			policy := backupQueryContractCompile(t, ListOptions{Filters: backupQueryContractRaw(tc.raw)})
			matched, err := matchLocal(context.Background(), view, policy.local, nil)
			if err != nil || matched != tc.match || len(policy.local) != 1 || len(policy.query) != 0 {
				t.Fatal(tc, matched, err, policy)
			}
		})
	}
	both := ListOptions{Filters: backupQueryContractRaw(`{"force":true,"is_forced":false}`)}
	backup := backupQueryContractCompile(t, both)
	snapshot, err := compileList(both)
	if err != nil || !reflect.DeepEqual(backupQueryContractLocalKeys(backup), []string{"force"}) || !reflect.DeepEqual(backupQueryContractLocalKeys(snapshot), []string{"is_forced"}) {
		t.Fatal(backup.local, snapshot, err)
	}
}

func TestBackupListControlsKeepBindingStageAndTypedOverridesIndependentOfResourceModel(t *testing.T) {
	detailed, paginated, maximum, version, expression := false, false, 0, "", ""
	policy := backupQueryContractCompile(t, ListOptions{Filters: backupQueryContractRaw(`{"paginated":"false","jmespath_filters":17,"max_items":"bad original","microversion":false,"headers":[1],"all_projects":false,"__conflicting_attrs":{"max_items":2.5,"microversion":"3.99","headers":{"X-Probe":"late"},"name":"late","force":true,"all_projects":[1,2],"jmespath_filters":"ignored late"}}`), Detailed: &detailed, Paginated: &paginated, MaxItems: &maximum, Microversion: &version, Headers: map[string]string{}, Expression: &expression})
	if policy.detailed || policy.paginated || string(policy.maximum) != "0" || policy.limit != nil || policy.microversion == nil || *policy.microversion != "" || policy.expression == nil || *policy.expression != "" || !reflect.DeepEqual(policy.headers, map[string]string{"Accept": "application/json"}) || !reflect.DeepEqual(backupQueryContractValues(t, policy), url.Values{"name": {"late"}, "all_tenants": {"1", "2"}}) || !reflect.DeepEqual(backupQueryContractLocalKeys(policy), []string{"force"}) {
		t.Fatal(policy)
	}
	for _, key := range []string{"details", "base_path", "resource_type", "self", "session", "cls"} {
		raw, _ := json.Marshal(map[string]any{key: nil})
		value := json.RawMessage(raw)
		policy, err := compileListFor(ListOptions{Filters: &value, Paginated: &paginated}, backupReadSchema())
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, policy, err)
		}
	}
	for _, key := range []string{"paginated", "base_path", "session", "cls"} {
		raw, _ := json.Marshal(map[string]any{"__conflicting_attrs": map[string]any{key: nil}})
		value := json.RawMessage(raw)
		policy, err := compileListFor(ListOptions{Filters: &value, Paginated: &paginated}, backupReadSchema())
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("late", key, policy, err)
		}
	}
	late := backupQueryContractCompile(t, ListOptions{Filters: backupQueryContractRaw(`{"__conflicting_attrs":{"details":false,"resource_type":"unused","self":false,"jmespath_filters":17,"force":true,"unknown":"dropped"}}`)})
	if !late.detailed || late.expression != nil || len(late.query) != 0 || !reflect.DeepEqual(backupQueryContractLocalKeys(late), []string{"force"}) {
		t.Fatal(late)
	}
}

func TestBackupListPolicyOwnsQueryLocalHeadersAndResourceSelectionAcrossReuse(t *testing.T) {
	raw := json.RawMessage(`{"all_projects":false,"max_items":2.5,"force":true,"metadata":{"nested":1},"headers":{"X-Probe":"original"},"microversion":"3.60"}`)
	before := bytes.Clone(raw)
	policy := backupQueryContractCompile(t, ListOptions{Filters: &raw})
	repeated := backupQueryContractCompile(t, ListOptions{Filters: &raw})
	if !bytes.Equal(raw, before) || string(policy.limit) != "2.5" || string(policy.maximum) != "2.5" {
		t.Fatal(policy, string(raw))
	}
	for i := range raw {
		raw[i] = '!'
	}
	policy.query["all_tenants"][0] = '!'
	policy.query["limit"][0] = '9'
	policy.limit[0] = '8'
	policy.local[1].Value[0] = '!'
	policy.headers["X-Probe"] = "changed"
	*policy.microversion = "3.99"
	if string(policy.maximum) != "2.5" || string(repeated.query["all_tenants"]) != "false" || string(repeated.limit) != "2.5" || string(repeated.local[1].Value) != `{"nested":1}` || repeated.headers["X-Probe"] != "original" || *repeated.microversion != "3.60" {
		t.Fatal("compiled policies share owned memory", policy, repeated)
	}
	shared := json.RawMessage(`{"all_projects":false,"force":true,"is_forced":false}`)
	var wg sync.WaitGroup
	failures := make(chan string, 32)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(backup bool) {
			defer wg.Done()
			schema := snapshotReadSchema()
			if backup {
				schema = backupReadSchema()
			}
			got, err := compileListFor(ListOptions{Filters: &shared}, schema)
			if err != nil || got == nil {
				failures <- "compile failed"
				return
			}
			values, err := encodeQuery(got.query)
			want := []string{"is_forced"}
			if backup {
				want = []string{"force"}
				if values.Get("all_tenants") != "False" {
					failures <- "backup became snapshot truthiness"
				}
			} else if values.Has("all_tenants") {
				failures <- "snapshot became backup literalfalse"
			}
			if err != nil || !reflect.DeepEqual(backupQueryContractLocalKeys(got), want) {
				failures <- "resource predicates or query mutated"
			}
		}(i%2 == 0)
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

func TestBackupListFalseyAndInvalidRawPoliciesHaveAtomicResourceSpecificCompilation(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `-0.0e99999`, `[]`, `{}`, `""`} {
		policy := backupQueryContractCompile(t, ListOptions{Filters: backupQueryContractRaw(raw)})
		if !policy.detailed || !policy.paginated || len(policy.query) != 0 || len(policy.local) != 0 || policy.expression != nil || policy.maximum != nil || policy.microversion != nil || !reflect.DeepEqual(policy.headers, map[string]string{"Accept": "application/json"}) {
			t.Fatal(raw, policy)
		}
		unused := backupQueryContractCompile(t, ListOptions{Filters: backupQueryContractRaw(`{"jmespath_filters":` + raw + `,"allow_unknown_params":` + raw + `,"__conflicting_attrs":` + raw + `,"unknown":"ignored","is_forced":"ignored snapshot"}`)})
		if unused.expression != nil || len(unused.query) != 0 || len(unused.local) != 0 {
			t.Fatal("falsey raw control or snapshot-only predicate leaked", raw, unused)
		}
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage{}, json.RawMessage(" "), json.RawMessage(`{} []`), json.RawMessage(`{"x":}`), json.RawMessage{0xff}, json.RawMessage(`true`), json.RawMessage(`[1]`), json.RawMessage(`"truthy"`)} {
		policy, err := compileListFor(ListOptions{Filters: &raw}, backupReadSchema())
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(string(raw), policy, err)
		}
	}
	for _, raw := range []string{`{"__conflicting_attrs":true}`, `{"max_items":"2"}`, `{"microversion":false}`, `{"headers":{"X-Probe":null}}`, `{"jmespath_filters":true}`} {
		policy, err := compileListFor(ListOptions{Filters: backupQueryContractRaw(raw)}, backupReadSchema())
		if policy != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(raw, policy, err)
		}
	}
	empty := ""
	projection := backupQueryContractCompile(t, ListOptions{Expression: &empty})
	if projection.expression == nil || *projection.expression != "" {
		t.Fatal("typed empty projection disappeared", projection)
	}
	// Input values may retain the snapshot helper's internal error wording;
	// public Backup wrappers supply its operation/kind. The sentinel is stable.
	_, err := compileListFor(ListOptions{Filters: backupQueryContractRaw(`{"headers":false}`)}, backupReadSchema())
	if err == nil || !strings.Contains(err.Error(), "headers") {
		t.Fatal(err)
	}
}
