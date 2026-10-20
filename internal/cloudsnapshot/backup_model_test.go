package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestBackupModelHasTwentyFourNullableFieldsAndOwnDescriptorOrder(t *testing.T) {
	view, seeded, err := normalizeBackup(json.RawMessage(`{}`), nil, true, resource.CloudLocation{})
	if err != nil || seeded {
		t.Fatalf("empty backup: %s, seeded=%v, err=%v", view, seeded, err)
	}
	fields := snapshotModelContractFields(t, view)
	keys := []string{"availability_zone", "container", "created_at", "data_timestamp", "description", "encryption_key_id", "fail_reason", "force", "has_dependent_backups", "is_incremental", "links", "metadata", "name", "object_count", "project_id", "size", "snapshot_id", "status", "updated_at", "user_id", "volume_id", "volume_name", "id"}
	if len(fields) != 24 {
		t.Fatalf("want 23 Body descriptors and computed location, got %s", view)
	}
	last := -1
	for _, key := range keys {
		snapshotModelContractRaw(t, fields, key, "null")
		at := strings.Index(string(view), `"`+key+`":`)
		if at <= last {
			t.Fatalf("source declaration/inheritance order changed at %q: %s", key, view)
		}
		last = at
	}
	if strings.Index(string(view), `"location":`) <= last {
		t.Fatal("Backup inherited location must follow inherited id", string(view))
	}
	snapshotModelContractRaw(t, fields, "location", `{"cloud":null,"region_name":null,"zone":null,"project":{"id":null,"name":null,"domain_id":null,"domain_name":null}}`)

	row := json.RawMessage(`{"container":[],"created_at":{"literal":9007199254740993},"data_timestamp":false,"description":[null],"encryption_key_id":{},"fail_reason":0,"name":{"raw":"not-stringified"},"snapshot_id":false,"status":["available"],"updated_at":"not-parsed-time","user_id":{},"volume_id":0,"volume_name":[false],"id":{"opaque":9007199254740993},"unknown":9007199254740993,"incremental":true,"display_name":"ignored","display_description":"ignored","SIZE":55}`)
	before := bytes.Clone(row)
	view, seeded, err = normalizeBackup(row, nil, true, resource.CloudLocation{})
	if err != nil || seeded {
		t.Fatal(string(view), seeded, err)
	}
	fields = snapshotModelContractFields(t, view)
	for key, want := range map[string]string{
		"container": "[]", "created_at": `{"literal":9007199254740993}`, "data_timestamp": "false", "description": "[null]",
		"encryption_key_id": "{}", "fail_reason": "0", "name": `{"raw":"not-stringified"}`, "snapshot_id": "false", "status": `["available"]`,
		"updated_at": `"not-parsed-time"`, "user_id": "{}", "volume_id": "0", "volume_name": "[false]", "id": `{"opaque":9007199254740993}`, "is_incremental": "null", "size": "null",
	} {
		snapshotModelContractRaw(t, fields, key, want)
	}
	if len(fields) != 24 || !bytes.Equal(row, before) {
		t.Fatal("unknown read fields leaked or physical row changed", string(view), string(row))
	}
	row[0] = '!'
	if !json.Valid(view) {
		t.Fatal("logical view borrows physical row bytes", string(view))
	}
}

func TestBackupModelOrdinaryBooleanDescriptorsAreNullableAndDoNotUseBoolStr(t *testing.T) {
	for _, key := range []string{"force", "has_dependent_backups", "is_incremental"} {
		for _, tc := range []struct{ raw, want string }{
			{"null", "null"}, {"true", "true"}, {"false", "false"}, {"0", "false"}, {"-1", "true"}, {"0.0", "false"},
			{`""`, "false"}, {`"false"`, "true"}, {`"FALSE"`, "true"}, {`" true "`, "true"}, {`"yes"`, "true"},
			{"[]", "false"}, {"{}", "false"}, {"[false]", "true"}, {`{"x":null}`, "true"}, {"1e400", "true"},
			// Owned JSON truthiness is exact. Python response.json float
			// underflow turns this literal into zero; that is a declared mapping.
			{"1e-9999", "true"},
		} {
			t.Run(key+"/"+tc.raw, func(t *testing.T) {
				view, seeded, err := normalizeBackup(json.RawMessage(`{"`+key+`":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
				if err != nil || seeded {
					t.Fatal(string(view), seeded, err)
				}
				snapshotModelContractRaw(t, snapshotModelContractFields(t, view), key, tc.want)
			})
		}
	}
}

func TestBackupModelLinksWrapSingleValuesAndMetadataUsesOrdinaryNullableDict(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"null", "null"}, {"[]", "[]"}, {`[false,{"n":9007199254740993}]`, `[false,{"n":9007199254740993}]`},
		{"{}", "[{}]"}, {"false", "[false]"}, {"0", "[0]"}, {`"abc"`, `["abc"]`}, {`{"next":"opaque"}`, `[{"next":"opaque"}]`},
	} {
		t.Run("links/"+tc.raw, func(t *testing.T) {
			view, seeded, err := normalizeBackup(json.RawMessage(`{"links":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "links", tc.want)
		})
	}
	for _, tc := range []struct{ raw, want string }{
		{"null", "null"}, {"{}", "{}"}, {"false", "{}"}, {"0", "{}"}, {`"opaque"`, "{}"}, {`[["pair",1]]`, "{}"},
		{`{"self":null,"connection":false,"microversion":{},"_synchronized":[1],"n":9007199254740993,"nested":[false,null]}`, `{"self":null,"connection":false,"microversion":{},"_synchronized":[1],"n":9007199254740993,"nested":[false,null]}`},
	} {
		t.Run("metadata/"+tc.raw, func(t *testing.T) {
			view, seeded, err := normalizeBackup(json.RawMessage(`{"metadata":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "metadata", tc.want)
		})
	}
}

func TestBackupModelBothIntegerDescriptorsConvertBeforeAnySelection(t *testing.T) {
	for _, key := range []string{"object_count", "size"} {
		for _, tc := range []struct{ raw, want string }{
			{"null", "null"}, {"true", "true"}, {"false", "false"}, {"-7", "-7"}, {"3.9", "3"}, {"-3.9", "-3"},
			{"900719925474099312345", "900719925474099312345"}, {"1e30", "1000000000000000019884624838656"}, {"1e-9999", "0"},
			{`"١２৩"`, "123"}, {`"0007"`, "7"}, {`"-3"`, "0"}, {`" 3 "`, "0"}, {`"3.5"`, "0"}, {`""`, "0"}, {"[]", "0"}, {"{}", "0"},
		} {
			t.Run(key+"/"+tc.raw, func(t *testing.T) {
				view, seeded, err := normalizeBackup(json.RawMessage(`{"`+key+`":`+tc.raw+`}`), nil, true, resource.CloudLocation{})
				if err != nil || seeded {
					t.Fatal(string(view), seeded, err)
				}
				snapshotModelContractRaw(t, snapshotModelContractFields(t, view), key, tc.want)
			})
		}
		for _, bad := range []string{`"²"`, "1e400"} {
			row := json.RawMessage(`{"id":"not-selected","name":"mismatch","force":true,"` + key + `":` + bad + `}`)
			before := bytes.Clone(row)
			for _, list := range []bool{false, true} {
				seed := "requested"
				view, seeded, err := normalizeBackup(row, &seed, list, resource.CloudLocation{})
				var policy *locationError
				if err == nil || view != nil || seeded || errors.As(err, &policy) || !strings.Contains(err.Error(), `"`+key+`"`) || !bytes.Equal(row, before) {
					t.Fatalf("invalid descriptor was hidden by later selection or misclassified: %s, %v, %v", view, seeded, err)
				}
			}
		}
	}
}

func TestBackupModelProjectAliasesFollowParsedDuplicateFirstInsertionOrder(t *testing.T) {
	for _, tc := range []struct{ row, want string }{
		{`{"project_id":"first","os-backup-project-attr:project_id":"middle","project_id":"last"}`, `"middle"`},
		{`{"os-backup-project-attr:project_id":"first","project_id":"middle","os-backup-project-attr:project_id":"last"}`, `"middle"`},
		{`{"project_id":"unused","os-backup-project-attr:project_id":null}`, "null"},
		{`{"os-backup-project-attr:project_id":null,"project_id":{"selected":9007199254740993}}`, `{"selected":9007199254740993}`},
		{`{"Project_ID":"unknown","OS-BACKUP-PROJECT-ATTR:PROJECT_ID":"unknown"}`, "null"},
	} {
		t.Run(tc.row, func(t *testing.T) {
			row := json.RawMessage(tc.row)
			view, seeded, err := normalizeBackup(row, nil, true, snapshotModelContractScope())
			if err != nil || seeded || string(row) != tc.row {
				t.Fatal(string(view), seeded, err, string(row))
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "project_id", tc.want)
		})
	}
}

func TestBackupModelMemberSeedsOnlyAbsentInheritedIDAndKeepsPhysicalBytes(t *testing.T) {
	seed := "requested-id"
	row := json.RawMessage(`{"status":"available","unknown":1}`)
	view, seeded, err := normalizeBackup(row, &seed, false, resource.CloudLocation{})
	if err != nil || !seeded {
		t.Fatal(string(view), seeded, err)
	}
	snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", `"requested-id"`)
	if string(row) != `{"status":"available","unknown":1}` || seed != "requested-id" {
		t.Fatal("logical seed changed original raw proof or caller seed", string(row), seed)
	}
	seed = "changed-after-call"
	snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", `"requested-id"`)
	for _, id := range []string{"null", "false", "0", `""`, "[]", "{}", `{"opaque":9007199254740993}`, `"actual-id"`} {
		t.Run(id, func(t *testing.T) {
			row := json.RawMessage(`{"id":` + id + `}`)
			view, seeded, err := normalizeBackup(row, &seed, false, resource.CloudLocation{})
			if err != nil || seeded || string(row) != `{"id":`+id+`}` {
				t.Fatal(string(view), seeded, err, string(row))
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", id)
		})
	}
	for _, list := range []bool{false, true} {
		view, seeded, err := normalizeBackup(json.RawMessage(`{}`), nil, list, resource.CloudLocation{})
		if err != nil || seeded {
			t.Fatal(string(view), seeded, err)
		}
		snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "id", "null")
	}
}

func TestBackupModelListConstructorControlsDoNotBecomeMemberResponseControls(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		for _, value := range []string{"null", "false", "0", "[]", "{}", `"control"`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				row := json.RawMessage(`{"id":"actual","name":"mismatch","` + key + `":` + value + `}`)
				view, seeded, err := normalizeBackup(row, nil, true, resource.CloudLocation{})
				var policy *locationError
				if err == nil || view != nil || seeded || errors.As(err, &policy) || !strings.Contains(err.Error(), key) {
					t.Fatal("list constructor collision did not fail at consumed row", string(view), seeded, err)
				}
				seed := "seed"
				view, seeded, err = normalizeBackup(row, &seed, false, resource.CloudLocation{})
				if err != nil || seeded {
					t.Fatal("member runtime field became a constructor argument", string(view), seeded, err)
				}
				fields := snapshotModelContractFields(t, view)
				snapshotModelContractRaw(t, fields, "id", `"actual"`)
				if _, ok := fields[key]; ok {
					t.Fatal("runtime control leaked into logical view", string(view))
				}
			})
		}
	}
	for _, list := range []bool{false, true} {
		view, seeded, err := normalizeBackup(json.RawMessage(`{"self":{"ignored":true},"Connection":true,"Microversion":true,"_Synchronized":true,"metadata":{"self":true,"connection":{},"microversion":null,"_synchronized":false}}`), nil, list, resource.CloudLocation{})
		if err != nil || seeded {
			t.Fatal("self/case variants/nested metadata incorrectly treated as controls", string(view), seeded, err)
		}
		snapshotModelContractRaw(t, snapshotModelContractFields(t, view), "metadata", `{"self":true,"connection":{},"microversion":null,"_synchronized":false}`)
	}
}

func TestBackupModelLocationUsesBothRowProjectAndLiteralAvailabilityZone(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, project := range []string{"null", "false", "0", `""`, "[]", "{}", `"scope"`} {
			scope := snapshotModelContractScope()
			row := json.RawMessage(`{"project_id":` + project + `,"availability_zone":"wire-zone","location":{"project":{"id":"wire"},"zone":"ignored"}}`)
			view, seeded, err := normalizeBackup(row, nil, list, scope)
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
			snapshotModelContractRaw(t, location, "cloud", `"snapshot-cloud"`)
			snapshotModelContractRaw(t, location, "region_name", `"snapshot-region"`)
			snapshotModelContractRaw(t, location, "zone", `"wire-zone"`)
			snapshotModelContractRaw(t, location, "project", `{"id":"scope","name":"configured-project","domain_id":"domain-id","domain_name":"Domain"}`)
			if string(scope.Zone) != `"caller-zone"` || string(scope.Project.ID) != `"scope"` {
				t.Fatal("row scope computation mutated caller scope")
			}
		}
		for _, project := range []string{`"foreign"`, `{"opaque":9007199254740993}`, "[1,false]", "1e-9999"} {
			view, seeded, err := normalizeBackup(json.RawMessage(`{"os-backup-project-attr:project_id":`+project+`}`), nil, list, snapshotModelContractScope())
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
			snapshotModelContractRaw(t, location, "zone", "null")
			snapshotModelContractRaw(t, location, "project", `{"id":`+project+`,"name":null,"domain_id":null,"domain_name":null}`)
		}
		for _, zone := range []string{"null", "false", "0", `""`, "[]", "{}", `"wire-zone"`, `{"opaque":9007199254740993}`, "[1,false]"} {
			view, seeded, err := normalizeBackup(json.RawMessage(`{"availability_zone":`+zone+`}`), nil, list, snapshotModelContractScope())
			if err != nil || seeded {
				t.Fatal(string(view), seeded, err)
			}
			snapshotModelContractRaw(t, snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"]), "zone", zone)
		}
		// Every connected Backup has a zone descriptor: missing/null zone
		// replaces the caller default, and raw wire location cannot win.
		scope := snapshotModelContractScope()
		scope.Zone = json.RawMessage(`invalid-unused-default`)
		view, seeded, err := normalizeBackup(json.RawMessage(`{"location":"wire-only"}`), nil, list, scope)
		if err != nil || seeded {
			t.Fatal(string(view), seeded, err)
		}
		snapshotModelContractRaw(t, snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"]), "zone", "null")
	}
	// Exact JSON project equality deliberately includes Python's bool/int
	// relation while avoiding float rounding. Foreign underflow above is
	// likewise the documented Go exact-decimal mapping.
	scope := snapshotModelContractScope()
	scope.Project.ID = json.RawMessage(`1`)
	view, _, err := normalizeBackup(json.RawMessage(`{"project_id":true,"availability_zone":false}`), nil, true, scope)
	if err != nil {
		t.Fatal(err)
	}
	location := snapshotModelContractFields(t, snapshotModelContractFields(t, view)["location"])
	snapshotModelContractRaw(t, location, "project", `{"id":1,"name":"configured-project","domain_id":"domain-id","domain_name":"Domain"}`)
	snapshotModelContractRaw(t, location, "zone", "false")
}

func TestBackupModelPhysicalAndLocalLocationFailuresAreAtomicAndOutputIsOwned(t *testing.T) {
	for _, row := range []json.RawMessage{nil, json.RawMessage{}, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`[]`), json.RawMessage(`{} {}`), json.RawMessage(`{"size":`), json.RawMessage(`{"id":"ok"} trailing`), json.RawMessage([]byte{'{', '"', 'i', 'd', '"', ':', '"', 0xff, '"', '}'})} {
		before := json.RawMessage(bytes.Clone(row))
		for _, list := range []bool{false, true} {
			seed := "seed"
			view, seeded, err := normalizeBackup(row, &seed, list, resource.CloudLocation{})
			var policy *locationError
			if err == nil || view != nil || seeded || errors.As(err, &policy) || errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("physical shape/encoding failure misclassified: %q => %s, %v, %v", row, view, seeded, err)
			}
		}
		if !reflect.DeepEqual(row, before) {
			t.Fatal("rejected physical input changed")
		}
	}
	for _, bad := range []json.RawMessage{json.RawMessage{}, json.RawMessage(`invalid`), json.RawMessage(`{} {}`), json.RawMessage{0xff}} {
		scope := snapshotModelContractScope()
		scope.Project.ID = bad
		view, seeded, err := normalizeBackup(json.RawMessage(`{"id":"admitted-row"}`), nil, true, scope)
		var policy *locationError
		if err == nil || view != nil || seeded || !errors.As(err, &policy) || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("owned policy failure lacks local cause: %s, %v, %v", view, seeded, err)
		}
	}
	scope := snapshotModelContractScope()
	row := json.RawMessage(`{"metadata":{"n":9007199254740993},"links":{"opaque":[1]},"created_at":[1],"project_id":null,"availability_zone":{"raw":[false,1]}}`)
	rowBefore, projectBefore, zoneBefore := bytes.Clone(row), bytes.Clone(scope.Project.ID), bytes.Clone(scope.Zone)
	view, seeded, err := normalizeBackup(row, nil, true, scope)
	if err != nil || seeded {
		t.Fatal(string(view), seeded, err)
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
	if !bytes.Equal(row[1:], rowBefore[1:]) {
		t.Fatal("mutating normalized output changed physical row")
	}
}
