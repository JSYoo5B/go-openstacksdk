package senlin

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type mutationOpts struct {
	Name     *string         `json:"name,omitempty"`
	Spec     json.RawMessage `json:"spec,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

func TestMutationSnapshotRetainsPresenceNumbersAndIndependentReuse(t *testing.T) {
	name := "original"
	raw := json.RawMessage(`{"count":9007199254740993,"fraction":1.234567890123456789,"enabled":false}`)
	option := Snapshot(mutationOpts{Name: &name, Spec: raw, Metadata: json.RawMessage(`null`)})
	name = "changed"
	raw[2] = 'X'
	first, err := request.Apply(mutationOpts{}, option)
	if err != nil {
		t.Fatal(err)
	}
	if *first.Options.Name != "original" || !strings.Contains(string(first.Options.Spec), `"count":9007199254740993`) || string(first.Options.Metadata) != "null" {
		t.Fatalf("snapshot changed: %+v", first.Options)
	}
	*first.Options.Name = "consumer"
	first.Options.Spec[2] = 'Y'
	second, err := request.Apply(mutationOpts{}, option)
	if err != nil || *second.Options.Name != "original" || second.Options.Spec[2] != 'c' {
		t.Fatalf("reused snapshot shared input: %+v, %v", second.Options, err)
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`{}`)} {
		config, err := request.Apply(mutationOpts{}, Snapshot(mutationOpts{Metadata: raw}))
		if err != nil || string(config.Options.Metadata) != string(raw) {
			t.Fatalf("presence %q became %q: %v", raw, config.Options.Metadata, err)
		}
	}
}

func TestMutationBodyProtectsOmittedInputsIdentitiesAndOwnedHeaders(t *testing.T) {
	for _, option := range []request.Option[mutationOpts]{
		request.WithField[mutationOpts]("name", "override"),
		request.WithField[mutationOpts]("id", "invented"),
		request.WithHeader[mutationOpts]("oPeNsTaCk-ApI-vErSiOn", "clustering 1.99"),
		request.WithHeader[mutationOpts]("X-Auth-Token", "override"),
		request.WithHeader[mutationOpts]("Content-Type", "text/plain"),
		request.WithQuery[mutationOpts]("project", "other"),
		request.WithArgument[mutationOpts]("target", "other"),
	} {
		config, err := request.Apply(mutationOpts{Spec: json.RawMessage(`{}`)}, option)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Body(config, "profile", "id"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("unprotected input: %v", err)
		}
	}
}

func TestMutationBodyRetainsEmptyAndNullObjectsInsideEnvelope(t *testing.T) {
	for _, metadata := range []string{`{}`, `null`} {
		config, err := request.Apply(mutationOpts{Metadata: json.RawMessage(metadata)},
			request.WithField[mutationOpts]("vendor", map[string]any{"count": json.Number("9007199254740993")}),
			request.WithHeader[mutationOpts]("X-Vendor", "retained"))
		if err != nil {
			t.Fatal(err)
		}
		body, err := Body(config, "profile")
		if err != nil {
			t.Fatal(err)
		}
		var envelope map[string]map[string]json.RawMessage
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatal(err)
		}
		fields := envelope["profile"]
		if string(fields["metadata"]) != metadata || fields["name"] != nil || string(fields["vendor"]) != `{"count":9007199254740993}` {
			t.Fatalf("body lost presence/precision: %s", body)
		}
	}
	config, _ := request.Apply(mutationOpts{})
	if _, err := Body(config, "profile"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty update must be explicit: %v", err)
	}
}

func TestMutationObjectValidationAllowsOnlyDocumentedPresenceShapes(t *testing.T) {
	for _, raw := range []string{`{}`, `{"nested":false}`} {
		if err := Object(json.RawMessage(raw), "spec"); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{"", `null`, `[]`, `false`, `"text"`, `{} {}`} {
		if err := Object(json.RawMessage(raw), "spec"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("spec %q was accepted: %v", raw, err)
		}
	}
	for _, raw := range []string{"", `null`, ` { } `} {
		if err := OptionalObject(json.RawMessage(raw), "metadata"); err != nil {
			t.Fatalf("optional %q rejected: %v", raw, err)
		}
	}
	if _, err := request.Apply(mutationOpts{}, Snapshot(mutationOpts{Spec: json.RawMessage(`{`)})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid snapshot accepted: %v", err)
	}
}

func TestMutationFlatBodyPreservesRootFieldsAndGuards(t *testing.T) {
	raw := json.RawMessage(`{"number":9007199254740993,"off":false}`)
	config, err := request.Apply(mutationOpts{Spec: raw}, request.WithField[mutationOpts]("vendor", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := FlatBody(config)
	if err != nil || string(body) != `{"spec":{"number":9007199254740993,"off":false},"vendor":null}` {
		t.Fatal(string(body), err)
	}
	raw[2] = 'X'
	config.Fields["vendor"] = json.RawMessage(`false`)
	if string(body) != `{"spec":{"number":9007199254740993,"off":false},"vendor":null}` {
		t.Fatal("flat body retained mutable request input", string(body))
	}
	for _, option := range []request.Option[mutationOpts]{
		request.WithField[mutationOpts]("name", "override"),
		request.WithField[mutationOpts]("id", "invented"),
		request.WithHeader[mutationOpts]("X-Auth-Token", "override"),
		request.WithQuery[mutationOpts]("project", "other"),
		request.WithArgument[mutationOpts]("target", "other"),
	} {
		config, err := request.Apply(mutationOpts{Metadata: json.RawMessage(`null`)}, option)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := FlatBody(config, "id"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("unprotected flat input", err)
		}
	}
	if _, err := FlatBody(request.Config[mutationOpts]{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("empty flat body accepted", err)
	}
}
