package resource_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type metadataModel struct {
	resource.Metadata
	ID string `json:"id"`
}

func (m *metadataModel) UnmarshalJSON(data []byte) error {
	type plain metadataModel
	return resource.DecodeObject(data, (*plain)(m), &m.Metadata)
}

func TestMetadataRetainsOriginalFields(t *testing.T) {
	raw := []byte(`{"id":"uuid","created_at":"2026-10-01 12:34:56","updated_at":null,"links":[{"rel":"self","href":"https://service/v1/uuid"}],"quota":9007199254740993,"extension":null,"empty":""}`)
	var model metadataModel
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	if model.ID != "uuid" || model.CreatedAt == nil || *model.CreatedAt != "2026-10-01 12:34:56" || model.UpdatedAt != nil {
		t.Fatalf("typed fields: %#v", model)
	}
	if string(model.Body["quota"]) != "9007199254740993" || string(model.Body["extension"]) != "null" || string(model.Body["empty"]) != `""` {
		t.Fatalf("raw fields: %s", model.Body)
	}
	if _, exists := model.Body["omitted"]; exists {
		t.Fatal("invented omitted field")
	}
	if len(model.Links) != 1 || model.Links[0].Rel != "self" {
		t.Fatalf("links: %#v", model.Links)
	}
	for i := range raw {
		raw[i] = 'x'
	}
	if string(model.Body["quota"]) != "9007199254740993" {
		t.Fatal("raw metadata aliases input")
	}
}

func TestMetadataRejectsNonObjectsAndTypedDecodeErrors(t *testing.T) {
	for _, raw := range []string{"", "null", "[]", `"string"`, "123", "{", `{"id":123}`} {
		t.Run(raw, func(t *testing.T) {
			var model metadataModel
			if err := model.UnmarshalJSON([]byte(raw)); err == nil {
				t.Fatalf("accepted %q", raw)
			}
		})
	}
	if err := resource.DecodeObject([]byte(`{}`), &metadataModel{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil metadata: %v", err)
	}
}
