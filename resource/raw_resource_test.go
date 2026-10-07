package resource_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestRawResourceAcceptsFlexibleCanonicalObjectsAndFailedDecodeIsAtomic(t *testing.T) {
	staleCreated, staleUpdated := "old creation", "old update"
	value := resource.RawResource{Metadata: resource.Metadata{Header: http.Header{"X-Proof": {"page"}}, StatusCode: 200, CreatedAt: &staleCreated, UpdatedAt: &staleUpdated, Links: []resource.Link{{Href: "/stale", Rel: "self"}}}}
	body := []byte(`{"id":false,"size":{"server":"owns schema"},"status":null,"created_at":23,"attachments":[{"server_id":true,"device":false}],"vendor":900719925474099312345678901234567890}`)
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	if string(value.Body["id"]) != "false" || string(value.Body["created_at"]) != "23" || string(value.Body["vendor"]) != "900719925474099312345678901234567890" || value.Header.Get("X-Proof") != "page" || value.StatusCode != 200 || value.CreatedAt != nil || value.UpdatedAt != nil || value.Links != nil {
		t.Fatal("new raw response retained stale derived metadata", value)
	}
	before := value.Clone()
	for _, invalid := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`false`), []byte(`{"id":`), []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		if err := json.Unmarshal(invalid, &value); err == nil || !reflect.DeepEqual(&value, before) {
			t.Fatalf("raw object validation was not atomic body=%q value=%+v error=%v", invalid, value, err)
		}
	}
	value.Body["id"] = json.RawMessage(`"edited"`)
	encoded, err := json.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil || string(fields["id"]) != `"edited"` || string(fields["vendor"]) != "900719925474099312345678901234567890" || len(fields) != len(value.Body) {
		t.Fatal(string(encoded), err)
	}
	if _, exists := fields["Header"]; exists {
		t.Fatal("HTTP evidence leaked into current JSON body", fields)
	}
}

func TestRawResourceCloneOwnsRawHeaderNullableMetadataAndLinks(t *testing.T) {
	created, updated := "created literal", "updated literal"
	value := &resource.RawResource{Metadata: resource.Metadata{CreatedAt: &created, UpdatedAt: &updated, Links: []resource.Link{{Href: "/original", Rel: "self"}}, Body: map[string]json.RawMessage{"vendor": json.RawMessage(`{"number":9007199254740993}`)}, Header: http.Header{"X-Proof": {"one", "two"}}, StatusCode: 200}}
	clone := value.Clone()
	if clone == nil || clone == value || !reflect.DeepEqual(clone, value) {
		t.Fatal(clone, value)
	}
	*clone.CreatedAt = "caller"
	*clone.UpdatedAt = "caller"
	clone.Links[0].Href = "/caller"
	clone.Body["vendor"][0] = '!'
	clone.Header["X-Proof"][0] = "caller"
	clone.StatusCode = 202
	if *value.CreatedAt != "created literal" || *value.UpdatedAt != "updated literal" || value.Links[0].Href != "/original" || string(value.Body["vendor"]) != `{"number":9007199254740993}` || !reflect.DeepEqual(value.Header["X-Proof"], []string{"one", "two"}) || value.StatusCode != 200 {
		t.Fatal("clone aliases original response evidence", value)
	}
	empty := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{}, Header: http.Header{}, Links: []resource.Link{}}}
	copy := empty.Clone()
	if copy.Body == nil || copy.Header == nil || copy.Links == nil {
		t.Fatal("clone lost explicit empty containers", copy)
	}
}

func TestRawResourceDecodeProjectsStrictTypedModelAtomicallyWithIndependentMetadata(t *testing.T) {
	var raw resource.RawResource
	if err := json.Unmarshal([]byte(`{"id":"volume","status":"in-use","size":2,"attachments":[{"server_id":"server","device":null}],"vendor":{"number":9007199254740993}}`), &raw); err != nil {
		t.Fatal(err)
	}
	raw.Header = http.Header{"X-Proof": {"actual page"}}
	raw.StatusCode = 200
	target := blockstorage.VolumeInfo{Metadata: resource.Metadata{Header: http.Header{"X-Old": {"old"}}, StatusCode: 201}, ID: func() *string { v := "old"; return &v }()}
	if err := raw.Decode(&target); err != nil {
		t.Fatal(err)
	}
	if target.ID == nil || *target.ID != "volume" || target.Status == nil || *target.Status != "in-use" || target.Header.Get("X-Proof") != "actual page" || target.StatusCode != 200 || string(target.Body["vendor"]) != `{"number":9007199254740993}` {
		t.Fatal(target)
	}
	target.Header["X-Proof"][0] = "caller"
	target.Body["vendor"][0] = '!'
	if raw.Header.Get("X-Proof") != "actual page" || string(raw.Body["vendor"]) != `{"number":9007199254740993}` {
		t.Fatal("projection aliases raw metadata")
	}
	before := target
	raw.Body["size"] = json.RawMessage(`false`)
	if err := raw.Decode(&target); err == nil || !reflect.DeepEqual(target, before) {
		t.Fatal("failed known-type projection partially changed target", target, err)
	}
	var fields map[string]json.RawMessage
	if err := raw.Decode(&fields); err != nil || string(fields["size"]) != "false" || string(fields["vendor"]) != `{"number":9007199254740993}` {
		t.Fatal("explicit raw map projection unnecessarily requires known schema", fields, err)
	}
	fields["vendor"][0] = '!'
	if string(raw.Body["vendor"]) != `{"number":9007199254740993}` {
		t.Fatal("map projection aliases original raw values")
	}
}

func TestRawResourceDecodeRejectsInvalidTargetAndRetainsLocalConversionCauses(t *testing.T) {
	raw := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"volume"`)}}}
	for _, target := range []any{nil, blockstorage.VolumeInfo{}, (*blockstorage.VolumeInfo)(nil)} {
		err := raw.Decode(target)
		var accepted *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &accepted) {
			t.Fatal("invalid projection target error identity", err)
		}
	}
	var target blockstorage.VolumeInfo
	var absent *resource.RawResource
	if err := absent.Decode(&target); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	raw.Body["id"] = json.RawMessage(`{"unfinished":`)
	err := raw.Decode(&target)
	var accepted *resource.ResponseError
	if err == nil || errors.As(err, &accepted) || target.ID != nil {
		t.Fatal("local invalid JSON projection invented HTTP proof or partial model", target, err)
	}
	raw.Body["id"] = json.RawMessage(`"volume"`)
	raw.Body["vendor"] = json.RawMessage([]byte{'"', 0xff, '"'})
	err = raw.Decode(&target)
	if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &accepted) || target.ID != nil {
		t.Fatal("caller RawMessage UTF-8 was replaced or partially decoded", target, err)
	}
	if err := absent.UnmarshalJSON([]byte(`{}`)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil raw receiver should be a local input error", err)
	}
	safe := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"volume"`)}, Header: http.Header{"X-Proof": {"actual"}}, StatusCode: 200}}
	var promoted rawResourcePromotedProjection
	if err := safe.Decode(&promoted); err != nil || promoted.ID != "volume" || promoted.RawResourceNilCarrier != nil {
		t.Fatal("nil promoted Metadata pointer panicked, was synthesized or blocked valid projection", promoted, err)
	}
}

type RawResourceNilCarrier struct{ resource.Metadata }
type rawResourcePromotedProjection struct {
	*RawResourceNilCarrier
	ID string `json:"id"`
}
