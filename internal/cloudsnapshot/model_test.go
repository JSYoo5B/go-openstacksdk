package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func snapshotModelContractFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		t.Fatalf("expected normalized object, got %s: %v", raw, err)
	}
	return fields
}

func snapshotModelContractRaw(t *testing.T, fields map[string]json.RawMessage, key, want string) {
	t.Helper()
	if got, ok := fields[key]; !ok || string(got) != want {
		t.Fatalf("%s: got %s (present=%v), want %s", key, got, ok, want)
	}
}

func snapshotModelContractScope() resource.CloudLocation {
	cloud, region := "snapshot-cloud", "snapshot-region"
	name, domainID, domainName := "configured-project", "domain-id", "Domain"
	return resource.CloudLocation{
		Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`"caller-zone"`),
		Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name, DomainID: &domainID, DomainName: &domainName},
	}
}

func TestSnapshotModelHasSixteenNullableFieldsAndPreservesUntypedValues(t *testing.T) {
	empty, seeded, err := normalize(json.RawMessage(`{}`), nil, true, resource.CloudLocation{})
	if err != nil || seeded {
		t.Fatalf("empty snapshot: %s, seeded=%v, err=%v", empty, seeded, err)
	}
	fields := snapshotModelContractFields(t, empty)
	keys := []string{"consumes_quota", "created_at", "description", "group_snapshot_id", "is_forced", "progress", "project_id", "size", "status", "updated_at", "user_id", "volume_id", "id", "name", "metadata"}
	if len(fields) != 16 {
		t.Fatalf("want 15 Body descriptors plus location, got %s", empty)
	}
	for _, key := range keys {
		snapshotModelContractRaw(t, fields, key, "null")
	}
	snapshotModelContractRaw(t, fields, "location", `{"cloud":null,"region_name":null,"zone":null,"project":{"id":null,"name":null,"domain_id":null,"domain_name":null}}`)

	row := json.RawMessage(`{"consumes_quota":"false","created_at":[false,9007199254740993],"description":{"opaque":null},"group_snapshot_id":false,"progress":[1],"project_id":null,"status":0,"updated_at":{"timestamp":"not-parsed"},"user_id":[],"volume_id":{},"id":{"opaque":9007199254740993},"name":[null],"unknown":9007199254740993,"SIZE":77,"display_name":"ignored","display_description":"ignored"}`)
	before := bytes.Clone(row)
	view, seeded, err := normalize(row, nil, true, resource.CloudLocation{})
	if err != nil || seeded {
		t.Fatal(string(view), seeded, err)
	}
	fields = snapshotModelContractFields(t, view)
	for key, want := range map[string]string{
		"consumes_quota": `"false"`, "created_at": `[false,9007199254740993]`, "description": `{"opaque":null}`,
		"group_snapshot_id": "false", "progress": "[1]", "project_id": "null", "status": "0", "updated_at": `{"timestamp":"not-parsed"}`,
		"user_id": "[]", "volume_id": "{}", "id": `{"opaque":9007199254740993}`, "name": "[null]", "size": "null",
	} {
		snapshotModelContractRaw(t, fields, key, want)
	}
	if len(fields) != 16 || !bytes.Equal(row, before) {
		t.Fatal("unknown attrs leaked or raw snapshot changed", string(view), string(row))
	}
	row[0] = '!'
	if !json.Valid(view) || !bytes.Equal(before[1:], row[1:]) {
		t.Fatal("normalized values borrow caller bytes", string(view))
	}
}

func TestSnapshotModelBoolStrIsNullableWithoutAnImplicitFalseDefault(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"null", "null"}, {"true", "true"}, {"false", "false"},
		{`"true"`, "true"}, {`"TrUe"`, "true"}, {`"FALSE"`, "false"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			view, seeded, err := normalize(json.RawMessage(`{"force":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "is_forced", tc.want)
		})
	}
	for _, bad := range []string{"0", "1", `"yes"`, `" true "`, `""`, "[]", "{}"} {
		t.Run("reject/"+bad, func(t *testing.T) {
			view, seeded, err := normalize(json.RawMessage(`{"description":"not-selected","force":`+bad+`}`), nil, true, resource.CloudLocation{})
			var policy *locationError
			if err == nil || view != nil || seeded || errors.As(err, &policy) || !strings.Contains(err.Error(), `"is_forced"`) {
				t.Fatalf("invalid descriptor was hidden or misclassified: %s, %v, %v", view, seeded, err)
			}
		})
	}
}

func TestSnapshotModelIntegerAndMetadataDescriptorsMatchOrdinaryResourceConversion(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"null", "null"}, {"true", "true"}, {"false", "false"}, {"-7", "-7"}, {"3.9", "3"}, {"-3.9", "-3"},
		{"900719925474099312345", "900719925474099312345"}, {"1e30", "1000000000000000019884624838656"}, {"1e-9999", "0"},
		{`"١２৩"`, "123"}, {`"0007"`, "7"}, {`"-3"`, "0"}, {`" 3 "`, "0"}, {`"3.5"`, "0"}, {`""`, "0"}, {"[]", "0"}, {"{}", "0"},
	} {
		t.Run("size/"+tc.raw, func(t *testing.T) {
			view, seeded, err := normalize(json.RawMessage(`{"size":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "size", tc.want)
		})
	}
	for _, bad := range []string{`"²"`, "1e400"} {
		view, seeded, err := normalize(json.RawMessage(`{"id":"other","description":"mismatch","size":`+bad+`}`), nil, true, resource.CloudLocation{})
		if err == nil || view != nil || seeded || !strings.Contains(err.Error(), `"size"`) {
			t.Fatal("ordinary constructor must eagerly reject this descriptor", string(view), seeded, err)
		}
	}
	for _, tc := range []struct{ raw, want string }{
		{"null", "null"}, {"{}", "{}"}, {"false", "{}"}, {"0", "{}"}, {`"x"`, "{}"}, {`[["pair",1]]`, "{}"},
		{`{"self":null,"connection":false,"microversion":{},"_synchronized":[1],"n":9007199254740993,"nested":[null,false]}`, `{"self":null,"connection":false,"microversion":{},"_synchronized":[1],"n":9007199254740993,"nested":[null,false]}`},
	} {
		t.Run("metadata/"+tc.raw, func(t *testing.T) {
			view, seeded, err := normalize(json.RawMessage(`{"metadata":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "metadata", tc.want)
		})
	}
}

func TestSnapshotModelAliasesUseParsedDuplicateCollapseAndFirstInsertionOrder(t *testing.T) {
	for _, tc := range []struct{ row, key, want string }{
		{`{"force":true,"is_forced":false}`, "is_forced", "false"},
		{`{"force":true,"is_forced":false,"force":true}`, "is_forced", "false"},
		{`{"is_forced":false,"force":true,"is_forced":false}`, "is_forced", "true"},
		{`{"force":"bad-unused","is_forced":true}`, "is_forced", "true"},
		{`{"is_forced":true,"force":"bad-unused","is_forced":false,"force":null}`, "is_forced", "null"},
		{`{"progress":1,"os-extended-snapshot-attributes:progress":2,"progress":3}`, "progress", "2"},
		{`{"os-extended-snapshot-attributes:progress":2,"progress":3,"os-extended-snapshot-attributes:progress":4}`, "progress", "3"},
		{`{"project_id":"first","os-extended-snapshot-attributes:project_id":"middle","project_id":"last"}`, "project_id", `"middle"`},
		{`{"Force":true,"IS_FORCED":true}`, "is_forced", "null"},
	} {
		t.Run(tc.row, func(t *testing.T) {
			row := json.RawMessage(tc.row)
			view, seeded, err := normalize(row, nil, true, snapshotModelContractScope())
			if err != nil || seeded || string(row) != tc.row {
				t.Fatal(string(view), seeded, err, string(row))
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), tc.key, tc.want)
		})
	}
}

func TestSnapshotModelMemberSeedsOnlyAbsentIDsWithoutChangingRawProof(t *testing.T) {
	seed := "requested-id"
	row := json.RawMessage(`{"status":"available","unknown":1}`)
	view, seeded, err := normalize(row, &seed, false, resource.CloudLocation{})
	if err != nil || !seeded {
		t.Fatal(string(view), seeded, err)
	}
	snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", `"requested-id"`)
	if string(row) != `{"status":"available","unknown":1}` || seed != "requested-id" {
		t.Fatal("seed was written into physical response or caller input", string(row), seed)
	}
	seed = "changed-after-call"
	snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", `"requested-id"`)
	for _, id := range []string{"null", "false", "0", `""`, "[]", "{}", `{"opaque":9007199254740993}`, `"actual-id"`} {
		t.Run(id, func(t *testing.T) {
			row := json.RawMessage(`{"id":` + id + `}`)
			view, seeded, err := normalize(row, &seed, false, resource.CloudLocation{})
			if err != nil || seeded || string(row) != `{"id":`+id+`}` {
				t.Fatal(string(view), seeded, err, string(row))
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", id)
		})
	}
	for _, list := range []bool{false, true} {
		view, seeded, err := normalize(json.RawMessage(`{}`), nil, list, resource.CloudLocation{})
		if err != nil || seeded {
			t.Fatal(string(view), seeded, err)
		}
		snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", "null")
	}
}

func TestSnapshotModelListConstructorCollisionsDoNotBecomeMemberResponseControls(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		for _, value := range []string{"null", "false", "0", "[]", "{}", `"control"`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				row := json.RawMessage(`{"id":"other","description":"mismatch","` + key + `":` + value + `}`)
				view, seeded, err := normalize(row, nil, true, resource.CloudLocation{})
				var policy *locationError
				if err == nil || view != nil || seeded || errors.As(err, &policy) || !strings.Contains(err.Error(), key) {
					t.Fatal("consumed list constructor collision was hidden", string(view), seeded, err)
				}
				seed := "seed"
				view, seeded, err = normalize(row, &seed, false, resource.CloudLocation{})
				if err != nil || seeded {
					t.Fatal("response field became a runtime argument", string(view), seeded, err)
				}
				fields := snapshotModelContractFields(t, view)
				snapshotModelContractRaw(t, fields, "id", `"other"`)
				if _, ok := fields[key]; ok {
					t.Fatal("runtime field leaked into normalized view", string(view))
				}
			})
		}
	}
	for _, list := range []bool{false, true} {
		row := json.RawMessage(`{"self":{"any":"value"},"Connection":true,"Microversion":true,"_Synchronized":true,"metadata":{"self":true,"connection":{},"microversion":null,"_synchronized":false}}`)
		view, seeded, err := normalize(row, nil, list, resource.CloudLocation{})
		if err != nil || seeded {
			t.Fatal("self should be discarded and nested metadata left ordinary", string(view), seeded, err)
		}
		snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "metadata", `{"self":true,"connection":{},"microversion":null,"_synchronized":false}`)
	}
}

func TestSnapshotModelRecomputesLocationFromProjectAndAlwaysUsesNullZone(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, project := range []string{"null", "false", "0", `""`, "[]", "{}", `"scope"`} {
			t.Run(project, func(t *testing.T) {
				scope := snapshotModelContractScope()
				row := json.RawMessage(`{"project_id":` + project + `,"availability_zone":"wire-zone","location":{"project":{"id":"wire"},"zone":"wire-zone"}}`)
				view, seeded, err := normalize(row, nil, list, scope)
				if err != nil || seeded {
					t.Fatal(string(view), seeded, err)
				}
				location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
				snapshotModelContractRaw(t, location, "cloud", `"snapshot-cloud"`)
				snapshotModelContractRaw(t, location, "region_name", `"snapshot-region"`)
				snapshotModelContractRaw(t, location, "zone", "null")
				snapshotModelContractRaw(t, location, "project", `{"id":"scope","name":"configured-project","domain_id":"domain-id","domain_name":"Domain"}`)
				if string(scope.Zone) != `"caller-zone"` || string(scope.Project.ID) != `"scope"` {
					t.Fatal("computed location mutated supplied scope")
				}
			})
		}
		for _, project := range []string{`"foreign"`, `{"opaque":9007199254740993}`, `[1,false]`, "1e-9999"} {
			view, seeded, err := normalize(json.RawMessage(`{"os-extended-snapshot-attributes:project_id":`+project+`}`), nil, list, snapshotModelContractScope())
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
			snapshotModelContractRaw(t, location, "zone", "null")
			snapshotModelContractRaw(t, location, "project", `{"id":`+project+`,"name":null,"domain_id":null,"domain_name":null}`)
		}
	}
	// The existing owned Location policy uses exact JSON truthiness. The tiny
	// nonzero exponent above intentionally differs from Python float underflow.
	scope := snapshotModelContractScope()
	scope.Project.ID = json.RawMessage(`1`)
	view, _, err := normalize(json.RawMessage(`{"project_id":true}`), nil, true, scope)
	if err != nil {
		t.Fatal(err)
	}
	location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
	snapshotModelContractRaw(t, location, "project", `{"id":1,"name":"configured-project","domain_id":"domain-id","domain_name":"Domain"}`)

	// Snapshots have no zone descriptor. An unused caller Zone never becomes
	// the source's computed zone, including an invalid raw unused value.
	scope.Zone = json.RawMessage(`not-used-json`)
	view, _, err = normalize(json.RawMessage(`{"location":"wire-only"}`), nil, false, scope)
	if err != nil {
		t.Fatal(err)
	}
	snapshotModelContractRaw(t, snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"]), "zone", "null")
}

func TestSnapshotModelRejectsPhysicalShapeAndLocalLocationFailuresAtomicallyAndOwnsResult(t *testing.T) {
	for _, row := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`[]`), json.RawMessage(`{} {}`), json.RawMessage(`{"size":`), json.RawMessage(`{"id":"ok"} trailing`), json.RawMessage([]byte{'{', '"', 'i', 'd', '"', ':', '"', 0xff, '"', '}'})} {
		before := json.RawMessage(bytes.Clone(row))
		for _, list := range []bool{false, true} {
			seed := "seed"
			view, seeded, err := normalize(row, &seed, list, resource.CloudLocation{})
			var policy *locationError
			if err == nil || view != nil || seeded || errors.As(err, &policy) || errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("physical row failure misclassified: %q => %s, %v, %v", row, view, seeded, err)
			}
		}
		if !reflect.DeepEqual(row, before) {
			t.Fatal("rejected input changed")
		}
	}
	for _, bad := range []json.RawMessage{json.RawMessage{}, json.RawMessage(`invalid`), json.RawMessage(`{} {}`), json.RawMessage{0xff}} {
		scope := snapshotModelContractScope()
		scope.Project.ID = bad
		view, seeded, err := normalize(json.RawMessage(`{"id":"accepted-row"}`), nil, true, scope)
		var policy *locationError
		if err == nil || view != nil || seeded || !errors.As(err, &policy) || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("owned policy failure lacks its local cause: %s, %v, %v", view, seeded, err)
		}
	}
	scope := snapshotModelContractScope()
	row := json.RawMessage(`{"metadata":{"n":9007199254740993},"created_at":[1],"project_id":null}`)
	rowBefore, projectBefore, zoneBefore := bytes.Clone(row), bytes.Clone(scope.Project.ID), bytes.Clone(scope.Zone)
	view, _, err := normalize(row, nil, true, scope)
	if err != nil {
		t.Fatal(err)
	}
	viewBefore := bytes.Clone(view)
	if !bytes.Equal(row, rowBefore) || !bytes.Equal(scope.Project.ID, projectBefore) || !bytes.Equal(scope.Zone, zoneBefore) {
		t.Fatal("normalization changed caller-owned state")
	}
	row[0], scope.Project.ID[0], scope.Zone[0] = '!', '!', '!'
	*scope.Cloud, *scope.RegionName, *scope.Project.Name = "changed", "changed", "changed"
	if !bytes.Equal(view, viewBefore) || !json.Valid(view) {
		t.Fatal("normalized result borrows input storage", string(view))
	}
	view[0] = '!'
	if row[0] != '!' || !bytes.Equal(row[1:], rowBefore[1:]) {
		t.Fatal("mutating output changed the caller row")
	}
}
