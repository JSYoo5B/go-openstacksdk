package blockstorage_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/resource"
)

func TestCreateVolumeModelsCanonicalNullableTypesAndLiteralMetadataPrecision(t *testing.T) {
	var value blockstorage.VolumeInfo
	value.Header = http.Header{"X-Proof": {"actual"}}
	value.StatusCode = 202
	body := `{"id":"new-1","ID":"decoy","status":"creating","Status":"ERROR","name":null,"description":"","size":0,"bootable":"FALSE","encrypted":true,"multiattach":false,"shared_targets":null,"consumes_quota":false,"created_at":"literal Cinder time","updated_at":null,"imageRef":"image","ImageRef":"decoy","source_volid":"source","consistencygroup_id":"group","metadata":{"number":9007199254740993,"null":null,"nested":{"enabled":false}},"attachments":[{"server_id":"server","device":null,"attached_at":"literal attach time","vendor":9007199254740993}],"links":[{"href":"https://example.test/volume","rel":"self"}],"vendor":9007199254740993}`
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID == nil || *value.ID != "new-1" || value.Status == nil || *value.Status != "creating" || value.Name != nil || value.Description == nil || *value.Description != "" || value.Size == nil || *value.Size != 0 || value.IsBootable == nil || *value.IsBootable || value.IsEncrypted == nil || !*value.IsEncrypted || value.Multiattach == nil || *value.Multiattach || value.SharedTargets != nil || value.ConsumesQuota == nil || *value.ConsumesQuota || value.CreatedAt == nil || *value.CreatedAt != "literal Cinder time" || value.UpdatedAt != nil || value.ImageID == nil || *value.ImageID != "image" || value.SourceVolumeID == nil || *value.SourceVolumeID != "source" || value.ConsistencyGroupID == nil || *value.ConsistencyGroupID != "group" || string(value.MetadataFields["number"]) != "9007199254740993" || string(value.MetadataFields["null"]) != "null" || string(value.Body["vendor"]) != "9007199254740993" || value.Header.Get("X-Proof") != "actual" || value.StatusCode != 202 || len(value.Attachments) != 1 || value.Attachments[0].Device != nil || *value.Attachments[0].AttachedAt != "literal attach time" || len(value.Links) != 1 || value.Links[0].Rel != "self" {
		t.Fatalf("typed/raw model=%+v", value)
	}
	for _, literal := range []string{`true`, `false`, `"TRUE"`, `"false"`, `null`} {
		var parsed blockstorage.VolumeInfo
		if err := json.Unmarshal([]byte(`{"bootable":`+literal+`,"encrypted":`+literal+`}`), &parsed); err != nil {
			t.Fatalf("supported BoolStr response=%s error=%v", literal, err)
		}
	}
	var wrongCase blockstorage.VolumeInfo
	if err := json.Unmarshal([]byte(`{"ID":"decoy","Status":"error","Size":2,"Bootable":true,"Metadata":{"number":9007199254740993},"Attachments":[]}`), &wrongCase); err != nil || wrongCase.ID != nil || wrongCase.Status != nil || wrongCase.Size != nil || wrongCase.IsBootable != nil || wrongCase.MetadataFields != nil || wrongCase.Attachments != nil || len(wrongCase.Body) != 6 {
		t.Fatalf("case aliases became canonical model fields: %+v error=%v", wrongCase, err)
	}
}

func TestCreateVolumeModelsKnownFieldFailuresAreAtomicAndNullEmptyRemainDistinct(t *testing.T) {
	valid := `{"id":"before","status":"creating","size":2,"bootable":false,"metadata":{"number":9007199254740993},"attachments":[],"links":[]}`
	for _, body := range []string{`null`, `[]`, `{"id":"` + "\xff" + `"}`, `{"id":"after","status":false}`, `{"id":"after","size":1.25}`, `{"id":"after","bootable":"yes"}`, `{"id":"after","encrypted":23}`, `{"id":"after","multiattach":"false"}`, `{"id":"after","metadata":[]}`, `{"id":"after","attachments":[null]}`, `{"id":"after","links":[null]}`, `{"id":"after","links":[{"href":true}]}`} {
		before, value := blockstorage.VolumeInfo{Metadata: resource.Metadata{Header: http.Header{"X-Keep": {"keep"}}, StatusCode: 201}}, blockstorage.VolumeInfo{Metadata: resource.Metadata{Header: http.Header{"X-Keep": {"keep"}}, StatusCode: 201}}
		if err := json.Unmarshal([]byte(valid), &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(valid), &value); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(body), &value); err == nil || !reflect.DeepEqual(value, before) {
			t.Fatalf("invalid=%s before=%+v after=%+v error=%v", body, before, value, err)
		}
	}
	for _, tc := range []struct {
		body    string
		nilMaps bool
	}{{`{"metadata":null,"attachments":null,"links":null}`, true}, {`{"metadata":{},"attachments":[],"links":[]}`, false}} {
		var value blockstorage.VolumeInfo
		if err := json.Unmarshal([]byte(tc.body), &value); err != nil || (value.MetadataFields == nil) != tc.nilMaps || (value.Attachments == nil) != tc.nilMaps || (value.Links == nil) != tc.nilMaps {
			t.Fatalf("empty/null distinctions=%+v error=%v", value, err)
		}
	}
}
