package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"gophercloudsdk/resource"
)

func snapshotMutationModelOverlay(t *testing.T, state *mutationState, raw string) {
	t.Helper()
	if err := state.overlay(json.RawMessage(raw)); err != nil {
		t.Fatalf("overlay %s: %v", raw, err)
	}
}

func snapshotMutationModelView(t *testing.T, state *mutationState, location resource.CloudLocation) map[string]json.RawMessage {
	t.Helper()
	raw, err := state.view(location)
	if err != nil {
		t.Fatalf("build mutation logical view: %v", err)
	}
	return snapshotMutationModelObject(t, raw)
}

func TestSnapshotMutationStateMergesKnownPartialFieldsWithoutChangingRawDescriptorValues(t *testing.T) {
	_, state := snapshotMutationModelCreateBody(t, "volume", CreateOptions{Attributes: CreateAttributes{Name: snapshotMutationModelString("request name"), Description: snapshotMutationModelString("request description")}})
	first := json.RawMessage(`{"id":"first","force":"FaLsE","size":"٧","metadata":"opaque","status":"creating","created_at":900719925474099312345678901,"consumes_quota":[],"group_snapshot_id":{"g":false},"os-extended-snapshot-attributes:progress":0.000,"updated_at":false,"user_id":[null],"unknown":{"nested":true},"location":"untrusted","connection":true,"self":true}`)
	firstBefore := json.RawMessage(bytes.Clone(first))
	if err := state.overlay(first); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, firstBefore) {
		t.Fatal("overlay changed actual response bytes")
	}
	location := resource.CloudLocation{Cloud: snapshotMutationModelString("cloud"), RegionName: snapshotMutationModelString("region"), Zone: json.RawMessage("{"), Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: snapshotMutationModelString("configured"), DomainID: snapshotMutationModelString("domain")}}
	view := snapshotMutationModelView(t, &state, location)
	if len(view) != 16 {
		t.Fatalf("logical field count = %d", len(view))
	}
	snapshotMutationModelRaw(t, view["is_forced"], "false")
	snapshotMutationModelRaw(t, view["size"], "7")
	snapshotMutationModelRaw(t, view["metadata"], "{}")
	snapshotMutationModelRaw(t, view["created_at"], "900719925474099312345678901")
	snapshotMutationModelRaw(t, view["consumes_quota"], "[]")
	snapshotMutationModelRaw(t, view["group_snapshot_id"], `{"g":false}`)
	snapshotMutationModelRaw(t, view["progress"], "0.000")
	snapshotMutationModelRaw(t, view["updated_at"], "false")
	snapshotMutationModelRaw(t, view["user_id"], "[null]")
	rawState := snapshotMutationModelObject(t, state.object())
	snapshotMutationModelRaw(t, rawState["is_forced"], `"FaLsE"`)
	snapshotMutationModelRaw(t, rawState["size"], `"٧"`)
	snapshotMutationModelRaw(t, rawState["metadata"], `"opaque"`)
	for _, key := range []string{"unknown", "location", "connection", "self"} {
		if _, present := rawState[key]; present {
			t.Fatalf("response-only field %q entered known mutation state", key)
		}
	}
	snapshotMutationModelOverlay(t, &state, `{"id":"second","status":"AVAILABLE","description":null,"project_id":{"foreign":1e-9999},"metadata":{"n":900719925474099312345678901}}`)
	view = snapshotMutationModelView(t, &state, location)
	snapshotMutationModelRaw(t, view["id"], `"second"`)
	snapshotMutationModelRaw(t, view["name"], `"request name"`)
	snapshotMutationModelRaw(t, view["description"], "null")
	snapshotMutationModelRaw(t, view["volume_id"], `"volume"`)
	snapshotMutationModelRaw(t, view["size"], "7")
	snapshotMutationModelRaw(t, view["metadata"], `{"n":900719925474099312345678901}`)
	computed := snapshotMutationModelObject(t, view["location"])
	snapshotMutationModelRaw(t, computed["cloud"], `"cloud"`)
	snapshotMutationModelRaw(t, computed["zone"], "null")
	project := snapshotMutationModelObject(t, computed["project"])
	snapshotMutationModelRaw(t, project["id"], `{"foreign":1e-9999}`)
	for _, key := range []string{"name", "domain_id", "domain_name"} {
		snapshotMutationModelRaw(t, project[key], "null")
	}
	if id, err := state.routeID(); err != nil || id != "second" {
		t.Fatalf("latest safe response ID not used: %q %v", id, err)
	}
	returnedID := state.id()
	returnedID[1] = 'X'
	snapshotMutationModelRaw(t, state.id(), `"second"`)
	snapshotMutationModelOverlay(t, &state, `{}`)
	snapshotMutationModelRaw(t, state.id(), `"second"`)
	snapshotMutationModelOverlay(t, &state, `{"name":null,"force":null,"size":null,"metadata":null}`)
	view = snapshotMutationModelView(t, &state, location)
	for _, key := range []string{"name", "is_forced", "size", "metadata"} {
		snapshotMutationModelRaw(t, view[key], "null")
	}
}

func TestSnapshotMutationStateAliasesUseParsedDuplicateInsertionOrder(t *testing.T) {
	cases := []struct{ raw, forced, progress, project string }{
		{`{"force":true,"is_forced":false,"force":true,"progress":"logical-first","os-extended-snapshot-attributes:progress":"wire-middle","progress":"logical-last"}`, "false", `"wire-middle"`, "null"},
		{`{"is_forced":true,"force":false,"is_forced":true,"os-extended-snapshot-attributes:project_id":"wire-first","project_id":"logical-middle","os-extended-snapshot-attributes:project_id":"wire-last"}`, "false", "null", `"logical-middle"`},
		{`{"force":"not a BoolStr","is_forced":"TrUe","Force":false}`, "true", "null", "null"},
		{`{"is_forced":"not a BoolStr","force":"FALSE"}`, "false", "null", "null"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			var state mutationState
			snapshotMutationModelOverlay(t, &state, tc.raw)
			view := snapshotMutationModelView(t, &state, resource.CloudLocation{})
			snapshotMutationModelRaw(t, view["is_forced"], tc.forced)
			snapshotMutationModelRaw(t, view["progress"], tc.progress)
			snapshotMutationModelRaw(t, view["project_id"], tc.project)
		})
	}
	var state mutationState
	snapshotMutationModelOverlay(t, &state, `{"id":"original","force":false}`)
	before := state.object()
	for _, raw := range []json.RawMessage{json.RawMessage(`{"id":"partial",`), json.RawMessage(`{"id":"partial"} {}`), json.RawMessage("null"), json.RawMessage("[]"), json.RawMessage{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		if err := state.overlay(raw); err == nil {
			t.Fatalf("invalid overlay was accepted: %q", raw)
		}
		if !bytes.Equal(state.object(), before) {
			t.Fatalf("invalid overlay changed prior known state: %s", state.object())
		}
	}
}

func TestSnapshotMutationStateConversionIsEagerAndErrorsDoNotRewriteRawState(t *testing.T) {
	for _, raw := range []string{`{"id":"valid","status":"available","force":1}`, `{"id":"valid","status":"available","force":" true "}`, `{"id":"valid","status":"available","size":"²"}`, `{"id":"valid","status":"available","size":1e999}`} {
		t.Run(raw, func(t *testing.T) {
			var state mutationState
			snapshotMutationModelOverlay(t, &state, raw)
			before := state.object()
			value, err := state.view(resource.CloudLocation{})
			if err == nil || value != nil {
				t.Fatalf("ready status deferred a reached invalid descriptor: %s %v", value, err)
			}
			var physical *resource.ResponseError
			if errors.As(err, &physical) {
				t.Fatal("pure logical conversion invented HTTP proof")
			}
			if !bytes.Equal(before, state.object()) {
				t.Fatal("failed descriptor conversion changed raw known values")
			}
		})
	}
	for _, tc := range []struct{ raw, want string }{
		{`"900719925474099312345678901"`, "900719925474099312345678901"},
		{"true", "true"}, {"false", "false"}, {"-1.9", "-1"},
		{`"-7"`, "0"}, {`" 7 "`, "0"}, {"[]", "0"}, {"{}", "0"}, {"null", "null"},
	} {
		var state mutationState
		snapshotMutationModelOverlay(t, &state, `{"size":`+tc.raw+`}`)
		view := snapshotMutationModelView(t, &state, resource.CloudLocation{})
		snapshotMutationModelRaw(t, view["size"], tc.want)
		snapshotMutationModelRaw(t, snapshotMutationModelObject(t, state.object())["size"], tc.raw)
	}
	var state mutationState
	snapshotMutationModelOverlay(t, &state, `{"status":"available"}`)
	value, err := state.view(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage("{")}})
	var local *locationError
	var physical *resource.ResponseError
	if value != nil || !errors.As(err, &local) || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
		t.Fatalf("invalid owned location was not a local atomic failure: %s %v", value, err)
	}
}

func TestSnapshotMutationStateRawIDsAndNullableStatusesValidateOnlyAtReachedPhases(t *testing.T) {
	for _, raw := range []string{"null", "false", "0", `""`, "[]", `{"id":"nested"}`, `"../other"`, `"encoded%2fsegment"`, `"white space"`, `"line\nfeed"`} {
		t.Run(raw, func(t *testing.T) {
			var state mutationState
			snapshotMutationModelOverlay(t, &state, `{"id":`+raw+`,"status":"available"}`)
			view := snapshotMutationModelView(t, &state, resource.CloudLocation{})
			snapshotMutationModelRaw(t, view["id"], raw)
			snapshotMutationModelRaw(t, state.id(), raw)
			if _, err := state.routeID(); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("used invalid route ID accepted: %s %v", state.id(), err)
			}
			snapshotMutationModelRaw(t, state.id(), raw)
		})
	}
	var state mutationState
	snapshotMutationModelOverlay(t, &state, `{"id":"日本語","status":"AvAiLaBlE"}`)
	if id, err := state.routeID(); id != "日本語" || err != nil {
		t.Fatalf("UTF8 single segment rejected: %q %v", id, err)
	}
	if status, err := state.status(false); status != "AvAiLaBlE" || err != nil {
		t.Fatalf("phase status did not preserve casing: %q %v", status, err)
	}
	snapshotMutationModelOverlay(t, &state, `{}`)
	if status, err := state.status(false); status != "AvAiLaBlE" || err != nil {
		t.Fatalf("missing status lost previous logical state: %q %v", status, err)
	}
	for _, raw := range []string{"null", "false", "0", "[]", "{}", `""`} {
		snapshotMutationModelOverlay(t, &state, `{"status":`+raw+`}`)
		if raw == `""` {
			if status, err := state.status(false); status != "" || err != nil {
				t.Fatalf("empty string is a valid nonterminal source status: %q %v", status, err)
			}
			continue
		}
		if _, err := state.status(false); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("delete wait accepted a reached nonstring status: %s %v", raw, err)
		}
		status, err := state.status(true)
		if raw == "null" {
			if status != "" || err != nil {
				t.Fatalf("create nullable status rejected: %q %v", status, err)
			}
		} else if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nullable create phase accepted a nonnull nonstring status: %s %v", raw, err)
		}
	}
	var empty mutationState
	snapshotMutationModelRaw(t, empty.id(), "null")
	if status, err := empty.status(true); status != "" || err != nil {
		t.Fatalf("missing create status was not nullable: %q %v", status, err)
	}
	if _, err := empty.status(false); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing delete wait status did not fail when reached: %v", err)
	}
}

func TestSnapshotMutationStateDeleteLogicalSeedDoesNotManufacturePhysicalFields(t *testing.T) {
	physical := json.RawMessage(`{"name":"chosen","force":"FALSE","size":"٧","metadata":"opaque","status":"available"}`)
	physicalBefore := json.RawMessage(bytes.Clone(physical))
	seed := "requested-member"
	logical, seeded, err := normalize(physical, &seed, false, resource.CloudLocation{})
	if err != nil || !seeded {
		t.Fatalf("member logical seed unavailable: %s %t %v", logical, seeded, err)
	}
	var actual resource.RawResource
	if err := json.Unmarshal(physical, &actual); err != nil {
		t.Fatal(err)
	}
	var state mutationState
	if err := state.overlay(logical); err != nil {
		t.Fatal(err)
	}
	view := snapshotMutationModelView(t, &state, resource.CloudLocation{})
	snapshotMutationModelRaw(t, view["id"], `"requested-member"`)
	snapshotMutationModelRaw(t, view["is_forced"], "false")
	snapshotMutationModelRaw(t, view["size"], "7")
	snapshotMutationModelRaw(t, view["metadata"], "{}")
	if _, present := actual.Body["id"]; present || !bytes.Equal(physical, physicalBefore) {
		t.Fatal("semantic delete seed was inserted in physical response")
	}
	snapshotMutationModelRaw(t, actual.Body["force"], `"FALSE"`)
	snapshotMutationModelRaw(t, actual.Body["size"], `"٧"`)
	snapshotMutationModelRaw(t, actual.Body["metadata"], `"opaque"`)
	snapshotMutationModelOverlay(t, &state, `{"status":"deleting"}`)
	if id, err := state.routeID(); id != seed || err != nil {
		t.Fatalf("missing later ID did not retain seed: %q %v", id, err)
	}
	snapshotMutationModelOverlay(t, &state, `{"id":"new-target"}`)
	if id, err := state.routeID(); id != "new-target" || err != nil {
		t.Fatalf("later explicit ID did not replace seed: %q %v", id, err)
	}
}
