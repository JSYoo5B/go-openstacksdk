package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func snapshotMutationModelObject(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var result map[string]json.RawMessage
	if err := json.Unmarshal(raw, &result); err != nil || result == nil {
		t.Fatalf("expected nonnull object, got %s: %v", raw, err)
	}
	return result
}

func snapshotMutationModelRaw(t *testing.T, got json.RawMessage, want string) {
	t.Helper()
	if !bytes.Equal(bytes.TrimSpace(got), []byte(want)) {
		t.Fatalf("JSON = %s; want %s", got, want)
	}
}

func snapshotMutationModelString(value string) *string { return &value }

func snapshotMutationModelCreateBody(t *testing.T, volume string, policy CreateOptions) (map[string]json.RawMessage, mutationState) {
	t.Helper()
	wire, state, err := compileCreateBody(volume, policy)
	if err != nil {
		t.Fatalf("compile creation: %v", err)
	}
	envelope := snapshotMutationModelObject(t, wire)
	if len(envelope) != 1 {
		t.Fatalf("unexpected envelope fields: %s", wire)
	}
	return snapshotMutationModelObject(t, envelope["snapshot"]), state
}

func TestSnapshotMutationCreateBodyPreservesOnlyFourCloudAttributesAndDefaultFalse(t *testing.T) {
	body, state := snapshotMutationModelCreateBody(t, "", CreateOptions{})
	if len(body) != 2 {
		t.Fatalf("empty direct create invents fields: %#v", body)
	}
	snapshotMutationModelRaw(t, body["volume_id"], `""`)
	snapshotMutationModelRaw(t, body["force"], "false")
	snapshotMutationModelRaw(t, state.id(), "null")
	view, err := state.view(resource.CloudLocation{})
	if err != nil {
		t.Fatal(err)
	}
	logical := snapshotMutationModelObject(t, view)
	if len(logical) != 16 {
		t.Fatalf("logical field count = %d, want 16", len(logical))
	}
	for _, key := range []string{"consumes_quota", "created_at", "description", "group_snapshot_id", "progress", "project_id", "size", "status", "updated_at", "user_id", "id", "name", "metadata"} {
		snapshotMutationModelRaw(t, logical[key], "null")
	}
	snapshotMutationModelRaw(t, logical["is_forced"], "false")
	snapshotMutationModelRaw(t, logical["volume_id"], `""`)
	location := snapshotMutationModelObject(t, logical["location"])
	for _, key := range []string{"cloud", "region_name", "zone"} {
		snapshotMutationModelRaw(t, location[key], "null")
	}
	project := snapshotMutationModelObject(t, location["project"])
	for _, key := range []string{"id", "name", "domain_id", "domain_name"} {
		snapshotMutationModelRaw(t, project[key], "null")
	}

	for _, key := range []string{"id", "metadata", "size", "status", "is_forced", "force", "volume_id", "project_id", "base_path", "self", "connection", "microversion", "_synchronized", "resource_type", "__conflicting_attrs", "Name", "unknown"} {
		t.Run(key, func(t *testing.T) {
			policy := CreateOptions{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{key: json.RawMessage("null")}}}
			wire, state, err := compileCreateBody("volume", policy)
			if !errors.Is(err, resource.ErrInvalidOption) || wire != nil || string(state.object()) != "{}" {
				t.Fatalf("unknown cloud kwarg produced request/state: %s %s %v", wire, state.object(), err)
			}
		})
	}
	force := true
	body, _ = snapshotMutationModelCreateBody(t, "literal", CreateOptions{Force: &force, Attributes: CreateAttributes{
		Fields: map[string]json.RawMessage{"name": json.RawMessage(`"canonical"`), "display_name": json.RawMessage(`"discarded"`), "description": json.RawMessage(`"description"`), "display_description": json.RawMessage(`"discarded description"`)},
	}})
	if len(body) != 4 {
		t.Fatalf("direct fields became wider attributes: %#v", body)
	}
	snapshotMutationModelRaw(t, body["force"], "true")
	snapshotMutationModelRaw(t, body["name"], `"canonical"`)
	snapshotMutationModelRaw(t, body["description"], `"description"`)
}

func TestSnapshotMutationCreateBodyCanonicalPresenceSuppressesAliasesByJSONTruthiness(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage("false"), json.RawMessage("0"), json.RawMessage("-0.000e99"), json.RawMessage(`""`), json.RawMessage("[]"), json.RawMessage("{}")} {
		t.Run("falsey "+string(raw), func(t *testing.T) {
			body, _ := snapshotMutationModelCreateBody(t, "volume", CreateOptions{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{
				"name": raw, "display_name": json.RawMessage(`"must not win"`), "description": raw, "display_description": json.RawMessage(`"must not win either"`),
			}}})
			if _, present := body["name"]; present {
				t.Fatalf("falsey canonical name was transmitted: %#v", body)
			}
			if _, present := body["description"]; present {
				t.Fatalf("falsey canonical description was transmitted: %#v", body)
			}
		})
	}
	for _, raw := range []string{`"text"`, "true", "900719925474099312345678901", "1e-9999", "[false]", `{"empty":null,"n":900719925474099312345678901}`} {
		t.Run("truthy "+raw, func(t *testing.T) {
			body, state := snapshotMutationModelCreateBody(t, "volume", CreateOptions{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{
				"display_name": json.RawMessage(raw), "display_description": json.RawMessage(raw),
			}}})
			snapshotMutationModelRaw(t, body["name"], raw)
			snapshotMutationModelRaw(t, body["description"], raw)
			view, err := state.view(resource.CloudLocation{})
			if err != nil {
				t.Fatal(err)
			}
			snapshotMutationModelRaw(t, snapshotMutationModelObject(t, view)["name"], raw)
		})
	}
}

func TestSnapshotMutationCreateBodyTypedOverridesAndUnusedAliasValidation(t *testing.T) {
	invalidText := string([]byte{0xff})
	body, _ := snapshotMutationModelCreateBody(t, "volume", CreateOptions{Attributes: CreateAttributes{
		Name: snapshotMutationModelString("typed name"), Description: snapshotMutationModelString(""), DisplayName: &invalidText,
		Fields: map[string]json.RawMessage{"name": json.RawMessage("{"), "display_name": json.RawMessage("{"), "description": json.RawMessage(`"raw description"`), "display_description": json.RawMessage("{")},
	}})
	snapshotMutationModelRaw(t, body["name"], `"typed name"`)
	if _, present := body["description"]; present {
		t.Fatal("empty concrete canonical description did not suppress aliases")
	}
	body, _ = snapshotMutationModelCreateBody(t, "volume", CreateOptions{Attributes: CreateAttributes{
		DisplayName: snapshotMutationModelString("typed display"), Fields: map[string]json.RawMessage{"display_name": json.RawMessage("{")},
	}})
	snapshotMutationModelRaw(t, body["name"], `"typed display"`)
	for _, policy := range []CreateOptions{
		{Attributes: CreateAttributes{Name: &invalidText}},
		{Attributes: CreateAttributes{DisplayDescription: &invalidText}},
		{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{"name": json.RawMessage("{")}}},
		{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{"display_name": json.RawMessage{0xff}}}},
		{Attributes: CreateAttributes{Fields: map[string]json.RawMessage{"name": json.RawMessage(`"valid"`), "description": json.RawMessage("{} {}")}}},
	} {
		wire, state, err := compileCreateBody("volume", policy)
		if !errors.Is(err, resource.ErrInvalidOption) || wire != nil || string(state.object()) != "{}" {
			t.Fatalf("selected invalid value produced partial request/state: %s %s %v", wire, state.object(), err)
		}
	}
}

func TestSnapshotMutationCreateBodyInputValidationIsAtomicAndBodyDataDoesNotUseRoutes(t *testing.T) {
	for _, volume := range []string{"", "../literal/volume?not=a-route#fragment", "with space\n", "日本語"} {
		body, _ := snapshotMutationModelCreateBody(t, volume, CreateOptions{})
		expected, _ := json.Marshal(volume)
		if !bytes.Equal(body["volume_id"], expected) {
			t.Fatalf("literal volume data %q was changed to %s", volume, body["volume_id"])
		}
	}
	wire, state, err := compileCreateBody(string([]byte{'v', 0xff}), CreateOptions{})
	if !errors.Is(err, resource.ErrInvalidOption) || wire != nil || string(state.object()) != "{}" {
		t.Fatalf("invalid UTF8 body data produced a partial request: %s %s %v", wire, state.object(), err)
	}
}

func TestSnapshotMutationCreateBodyOwnsSuppliedFieldsAndIndependentState(t *testing.T) {
	name := json.RawMessage(`{"n":900719925474099312345678901}`)
	fields := map[string]json.RawMessage{"name": name}
	before := json.RawMessage(bytes.Clone(name))
	force := false
	policy := CreateOptions{Force: &force, Attributes: CreateAttributes{Fields: fields}}
	wire, state, err := compileCreateBody("volume", policy)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(name, before) || len(fields) != 1 || force {
		t.Fatal("compilation changed the supplied policy")
	}
	wireBefore := json.RawMessage(bytes.Clone(wire))
	stateBefore := state.object()
	for i := range name {
		name[i] = ' '
	}
	fields["name"] = json.RawMessage(`"changed"`)
	fields["description"] = json.RawMessage(`"late"`)
	force = true
	if !bytes.Equal(wire, wireBefore) || !bytes.Equal(state.object(), stateBefore) {
		t.Fatal("retained caller fields changed the compiled body or logical seed")
	}
	for i := range wire {
		wire[i] = ' '
	}
	if !bytes.Equal(state.object(), stateBefore) {
		t.Fatal("wire body aliases the logical seed")
	}
	if err := state.overlay(json.RawMessage(`{"id":"created","status":"available"}`)); err != nil {
		t.Fatal(err)
	}
	body := snapshotMutationModelObject(t, snapshotMutationModelObject(t, wireBefore)["snapshot"])
	snapshotMutationModelRaw(t, body["name"], string(before))
	snapshotMutationModelRaw(t, body["force"], "false")
	if _, present := body["id"]; present {
		t.Fatal("later logical overlay changed the earlier physical request")
	}
}
