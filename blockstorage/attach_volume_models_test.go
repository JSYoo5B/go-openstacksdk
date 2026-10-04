package blockstorage

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"gophercloudsdk/resource"
)

func TestAttachVolumeModelsCanonicalNullableFieldsAndRawExtensions(t *testing.T) {
	var observed AttachVolumeObservation
	body := `{"id":"vol-1","ID":"decoy","name":null,"status":"available","STATUS":"error","created_at":"2026-10-04 01:02:03.000001","updated_at":null,"vendor":{"number":9007199254740993,"enabled":false},"attachments":[{"id":null,"attachment_id":"cinder-attachment","volume_id":"vol-1","server_id":"server-1","device":"","host_name":null,"attached_at":"literal service time","vendor":9007199254740993}]}`
	observed.Header = http.Header{"X-Proof": {"actual"}}
	observed.StatusCode = 200
	if err := json.Unmarshal([]byte(body), &observed); err != nil {
		t.Fatal(err)
	}
	if observed.ID == nil || *observed.ID != "vol-1" || observed.Name != nil || observed.Status == nil || *observed.Status != "available" || observed.CreatedAt == nil || *observed.CreatedAt != "2026-10-04 01:02:03.000001" || observed.UpdatedAt != nil || observed.StatusCode != 200 || observed.Header.Get("X-Proof") != "actual" || string(observed.Body["vendor"]) != `{"number":9007199254740993,"enabled":false}` || string(observed.Body["name"]) != "null" || string(observed.Body["ID"]) != `"decoy"` || len(observed.Attachments) != 1 {
		t.Fatalf("observation=%+v raw=%v", observed, observed.Body)
	}
	row := observed.Attachments[0]
	if row.ID != nil || row.AttachmentID == nil || *row.AttachmentID != "cinder-attachment" || row.ServerID == nil || *row.ServerID != "server-1" || row.Device == nil || *row.Device != "" || row.HostName != nil || row.AttachedAt == nil || *row.AttachedAt != "literal service time" || string(row.Body["vendor"]) != "9007199254740993" {
		t.Fatalf("attachment=%+v", row)
	}
	var created VolumeAttachmentInfo
	if err := json.Unmarshal([]byte(`{"id":"nova-id","volumeId":"vol-1","VolumeId":"decoy","serverId":"server-1","device":null,"tag":"","attachment_id":"cinder-id","bdm_uuid":"block-device-id","delete_on_termination":false,"vendor":9007199254740993}`), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == nil || *created.ID != "nova-id" || created.VolumeID == nil || *created.VolumeID != "vol-1" || created.ServerID == nil || *created.ServerID != "server-1" || created.Device != nil || created.Tag == nil || *created.Tag != "" || created.AttachmentID == nil || *created.AttachmentID != "cinder-id" || created.BDMID == nil || *created.BDMID != "block-device-id" || created.DeleteOnTermination == nil || *created.DeleteOnTermination || string(created.Body["vendor"]) != "9007199254740993" {
		t.Fatalf("created=%+v", created)
	}
	var wrongCase VolumeAttachmentInfo
	if err := json.Unmarshal([]byte(`{"VolumeId":"decoy","ServerId":"decoy","Delete_On_Termination":true}`), &wrongCase); err != nil || wrongCase.VolumeID != nil || wrongCase.ServerID != nil || wrongCase.DeleteOnTermination != nil || len(wrongCase.Body) != 3 {
		t.Fatalf("case aliases became canonical fields: %+v error=%v", wrongCase, err)
	}
}

func TestAttachVolumeModelsDistinguishAbsentNullAndEmptyAttachments(t *testing.T) {
	for _, tc := range []struct {
		name, body              string
		present, null, nilSlice bool
	}{
		{"absent", `{}`, false, false, true},
		{"null", `{"attachments":null}`, true, true, true},
		{"empty", `{"attachments":[]}`, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var value AttachVolumeObservation
			err := json.Unmarshal([]byte(tc.body), &value)
			raw, present := value.Body["attachments"]
			if err != nil || present != tc.present || (string(raw) == "null") != tc.null || (value.Attachments == nil) != tc.nilSlice || len(value.Attachments) != 0 {
				t.Fatalf("value=%+v raw=%q error=%v", value, raw, err)
			}
		})
	}
}

func TestAttachVolumeModelsRejectInvalidFieldsAtomically(t *testing.T) {
	fixtures := []struct {
		name     string
		newValue func() any
		valid    string
		invalid  []string
	}{
		{"observation", func() any {
			return &AttachVolumeObservation{Metadata: resource.Metadata{Header: http.Header{"X-Existing": {"keep"}}, StatusCode: 201}}
		}, `{"id":"before","status":"available","attachments":[{"server_id":"before"}],"vendor":9007199254740993}`, []string{`null`, `[]`, `"scalar"`, `{`, `{"id":"after","status":false}`, `{"id":"after","attachments":{}}`, `{"id":"after","attachments":[null]}`, `{"id":"after","attachments":[{"server_id":4}]}`, "{\"name\":\"\xff\"}"}},
		{"record", func() any { return new(AttachVolumeRecord) }, `{"id":"before","device":null,"vendor":9007199254740993}`, []string{`null`, `[]`, `{"id":"after","server_id":false}`, `{"id":"after","attached_at":23}`}},
		{"creation", func() any { return new(VolumeAttachmentInfo) }, `{"id":"before","volumeId":"vol-1","delete_on_termination":false,"vendor":9007199254740993}`, []string{`null`, `[]`, `{"id":"after","volumeId":4}`, `{"id":"after","delete_on_termination":"false"}`, `{"id":"after","created_at":true}`}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			for _, body := range fixture.invalid {
				value, before := fixture.newValue(), fixture.newValue()
				if err := json.Unmarshal([]byte(fixture.valid), value); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(fixture.valid), before); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(body), value); err == nil || !reflect.DeepEqual(value, before) {
					t.Fatalf("invalid=%q changed=%+v before=%+v error=%v", body, value, before, err)
				}
			}
		})
	}
}
