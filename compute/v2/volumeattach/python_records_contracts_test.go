package volumeattach_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func attachmentRecordIDs(t *testing.T, api *volumeattach.API, options ...volumeattach.AttachmentListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for value, err := range api.ListRecords(context.Background(), "s1", options...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(value.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func TestPythonAttachmentRecordsListIsLazyAndPagesLimitOffset(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(req *http.Request) *http.Response {
		marker := req.URL.Query().Get("marker")
		switch marker {
		case "":
			return pythonVolumeAttachWire(200, `{"volumeAttachments":[{"volumeId":"v1","bdm_uuid":"b1","tag":false,"delete_on_termination":0,"opaque":99999999999999999999}]}`)
		case "v1":
			return pythonVolumeAttachWire(200, `{"volumeAttachments":[{"id":"v2","volumeId":"other"}]}`)
		default:
			return pythonVolumeAttachWire(200, `{"volumeAttachments":[]}`)
		}
	})
	sequence := api.ListRecords(context.Background(), "s1", volumeattach.WithAttachmentLimit(3), volumeattach.WithAttachmentOffset(2))
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var ids []string
	for v, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		var id string
		_ = json.Unmarshal(v.Resource.Body["id"], &id)
		ids = append(ids, id)
		if id == "v1" {
			if string(v.Resource.Body["bdm_id"]) != `"b1"` || string(v.Resource.Body["tag"]) != "false" || string(v.Resource.Body["delete_on_termination"]) != "0" || string(v.Resource.Body["name"]) != "null" || v.ServerID != "s1" || string(v.Wire.Body["opaque"]) != "99999999999999999999" {
				t.Fatal(v)
			}
			v.Resource.Body["id"] = json.RawMessage(`"consumer-mutated"`)
			v.Wire.Body["volumeId"] = json.RawMessage(`"consumer-mutated"`)
		}
	}
	if !reflect.DeepEqual(ids, []string{"v1", "v2"}) || len(*calls) != 3 || (*calls)[0].query != "limit=3&offset=2" || (*calls)[1].query != "limit=3&marker=v1&offset=2" || (*calls)[2].query != "limit=3&marker=v2&offset=2" {
		t.Fatal(ids, *calls)
	}
}
func TestPythonAttachmentRecordsControlCountsRawRowsBeforeFilters(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response {
		return pythonVolumeAttachWire(200, `{"volumeAttachments":[{"id":"v1","tag":"skip"},{"id":"v2","tag":"wanted"}],"next":"/nova/v2.1/servers/s1/os-volume_attachments?marker=v2&limit=2"}`)
	})
	ids, err := attachmentRecordIDs(t, api, volumeattach.WithAttachmentMaxItems(1), volumeattach.WithAttachmentQuery("tag", "wanted"), volumeattach.WithAttachmentQuery("not_a_field", make(chan int)))
	if err != nil || len(ids) != 0 || len(*calls) != 1 || (*calls)[0].query != "limit=1" {
		t.Fatal(ids, err, *calls)
	}
	ids, err = attachmentRecordIDs(t, api, volumeattach.WithAttachmentPaginated(false), volumeattach.WithAttachmentQuery("tag", "wanted"), volumeattach.WithAttachmentQuery("device", nil), volumeattach.WithAttachmentQuery("offset", nil), volumeattach.WithAttachmentLimit(0))
	if err != nil || !reflect.DeepEqual(ids, []string{"v2"}) || len(*calls) != 2 || (*calls)[1].query != "limit=0" {
		t.Fatal(ids, err, *calls)
	}
	// Early break never follows a server next link.
	for _, err := range api.ListRecords(context.Background(), "s1") {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(*calls) != 3 {
		t.Fatal(*calls)
	}
}
func TestPythonAttachmentRecordsLinksAndLocalAliasProjection(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "v1" {
			return pythonVolumeAttachWire(200, `{"volumeAttachments":[]}`)
		}
		return pythonVolumeAttachWire(200, `{"volumeAttachments":{"id":"v1","bdm_id":"canonical","bdm_uuid":"last","volume_id":"canonical-volume","volumeId":"last-volume"},"volumeAttachments_links":[{"rel":"next","href":"?marker=v1"}]}`)
	})
	var records []*volumeattach.AttachmentRecord
	for v, err := range api.ListRecords(context.Background(), "s1", volumeattach.WithAttachmentQuery("bdm_id", "last"), volumeattach.WithAttachmentQuery("volume_id", "last-volume")) {
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, v)
	}
	if len(records) != 1 || len(*calls) != 2 || string(records[0].Resource.Body["bdm_id"]) != `"last"` {
		t.Fatal(records, *calls)
	}
}
func TestPythonAttachmentRecordUpdateDefaultsSwapAndExplicitValues(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(req *http.Request) *http.Response {
		r := pythonVolumeAttachWire(200, `{"volumeAttachment":{"volumeId":"returned","bdm_uuid":"b","opaque":false}}`)
		r.Header.Set("X-Request-Id", "actual")
		return r
	})
	value, err := api.UpdateRecord(context.Background(), "s1", "old")
	if err != nil || value == nil || string(value.Resource.Body["id"]) != `"old"` || string(value.Resource.Body["volume_id"]) != `"returned"` || string(value.Resource.Body["device"]) != "null" || value.Header.Get("X-Request-Id") != "actual" || value.Wire.Body["id"] != nil {
		t.Fatal(value, err)
	}
	_, err = api.UpdateRecord(context.Background(), "s1", "old", volumeattach.WithAttachmentSwapTo("new"), volumeattach.WithAttachmentAttribute("delete_on_termination", false), volumeattach.WithAttachmentAttribute("device", nil), volumeattach.WithAttachmentAttribute("tag", ""), volumeattach.WithAttachmentAttribute("bdm_id", 0), volumeattach.WithAttachmentAttribute("unknown", make(chan int)))
	if err != nil || len(*calls) != 2 || (*calls)[0].method != "PUT" || (*calls)[0].path != pythonVolumeAttachBase+"/old" || (*calls)[0].body != `{"volumeAttachment":{"volumeId":"old"}}` || (*calls)[1].body != `{"volumeAttachment":{"bdm_uuid":0,"delete_on_termination":false,"device":null,"tag":"","volumeId":"new"}}` {
		t.Fatal(err, *calls)
	}
	_, err = api.UpdateRecord(context.Background(), "s1", "old", volumeattach.WithAttachmentAttribute("volumeId", nil))
	if err != nil || (*calls)[2].body != `{"volumeAttachment":{"volumeId":null}}` {
		t.Fatal(err, *calls)
	}
	_, err = api.UpdateRecord(context.Background(), "s1", "old", volumeattach.WithAttachmentAttribute("volume_id", nil))
	if err != nil || (*calls)[3].body != `{"volumeAttachment":{"volumeId":"old"}}` {
		t.Fatal(err, *calls)
	}
}
func TestPythonAttachmentRecordUpdateSnapshotsAndScope(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response { return pythonVolumeAttachWire(204, "") })
	payload := map[string]any{"enabled": false}
	attribute := volumeattach.WithAttachmentAttribute("tag", payload)
	payload["enabled"] = true
	var retained *volumeattach.AttachmentUpdateOpts
	scope, err := api.InServer(context.Background(), resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scope.UpdateRecord(context.Background(), "old", attribute, func(v *volumeattach.AttachmentUpdateOpts) error {
		retained = v
		v.Headers = map[string]string{"X-Owned": "old"}
		return nil
	}, func(*volumeattach.AttachmentUpdateOpts) error {
		retained.Fields["tag"] = json.RawMessage(`"changed"`)
		return nil
	})
	if err != nil || value == nil || value.Wire != nil || value.StatusCode != 204 || string(value.Resource.Body["volume_id"]) != `"old"` || (*calls)[0].body != `{"volumeAttachment":{"tag":{"enabled":false},"volumeId":"old"}}` {
		t.Fatal(value, err, *calls)
	}
	ids, err := attachmentRecordIDs(t, api, volumeattach.WithAttachmentPaginated(false))
	var receipt *resource.ResponseError
	if len(ids) != 0 || !errors.As(err, &receipt) || receipt.StatusCode != 204 {
		t.Fatal(ids, err, receipt)
	}
}
func TestPythonAttachmentRecordsRejectInvalidInputsBeforeHTTP(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response { return pythonVolumeAttachWire(200, `{"volumeAttachments":[]}`) })
	for _, test := range []struct {
		name string
		call func() error
	}{
		{"negative limit", func() error { _, err := attachmentRecordIDs(t, api, volumeattach.WithAttachmentLimit(-1)); return err }},
		{"negative offset", func() error { _, err := attachmentRecordIDs(t, api, volumeattach.WithAttachmentOffset(-1)); return err }},
		{"negative cap", func() error {
			_, err := attachmentRecordIDs(t, api, volumeattach.WithAttachmentMaxItems(-1))
			return err
		}},
		{"nil list option", func() error { _, err := attachmentRecordIDs(t, api, nil); return err }},
		{"nil update option", func() error { _, err := api.UpdateRecord(context.Background(), "s1", "v1", nil); return err }},
		{"target attr", func() error {
			_, err := api.UpdateRecord(context.Background(), "s1", "v1", volumeattach.WithAttachmentAttribute("id", "other"))
			return err
		}},
		{"bad target", func() error { _, err := api.UpdateRecord(context.Background(), "..", "v1"); return err }},
		{"auth header", func() error {
			_, err := api.UpdateRecord(context.Background(), "s1", "v1", volumeattach.WithAttachmentUpdateHeaders(map[string]string{"X-Auth-Token": "bad"}))
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
		})
	}
}
func TestPythonAttachmentRecordsPreservePhysicalErrorsAndCanceledOptions(t *testing.T) {
	api, _ := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response { return pythonVolumeAttachWire(200, `{}`) })
	readErr, closeErr := errors.New("accepted read"), errors.New("accepted close")
	attempts := 0
	api.RawClient().ProviderClient.HTTPClient.Transport = pythonVolumeAttachTransport(func(*http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"partial"}}, Body: &attachmentFaultBody{data: []byte(`{"volumeAttachment":`), read: readErr, close: closeErr}}, nil
	})
	value, err := api.UpdateRecord(context.Background(), "s1", "v1")
	var receipt *resource.ResponseError
	if value == nil || attempts != 1 || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(value.Envelope) != `{"volumeAttachment":` {
		t.Fatal(value, err, receipt, attempts)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cause"), errors.New("option")
	_, err = api.UpdateRecord(ctx, "s1", "v1", func(*volumeattach.AttachmentUpdateOpts) error { cancel(cause); return callback })
	if attempts != 1 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) {
		t.Fatal(err, attempts)
	}
	original := api.RawClient().Microversion
	defer func() { api.RawClient().Microversion = original }()
	_, err = api.UpdateRecord(context.Background(), "s1", "v1", func(*volumeattach.AttachmentUpdateOpts) error { api.RawClient().Microversion = "2.90"; return callback })
	if attempts != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callback) {
		t.Fatal(err, attempts)
	}
}
func TestPythonAttachmentRecordsDiscoverCeilingWithoutChangingClient(t *testing.T) {
	api, calls := pythonVolumeAttachAPI(t, func(*http.Request) *http.Response { return pythonVolumeAttachWire(200, `{}`) })
	client := api.RawClient()
	client.Endpoint = client.ResourceBase
	client.Microversion = ""
	versions := []string{}
	client.ProviderClient.HTTPClient.Transport = pythonVolumeAttachTransport(func(req *http.Request) (*http.Response, error) {
		versions = append(versions, req.Header.Get("X-OpenStack-Nova-API-Version"))
		if req.URL.Path == "/nova/v2.1" || req.URL.Path == "/nova/v2.1/" {
			return pythonVolumeAttachWire(200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"`+client.ResourceBase+`"}]}}`), nil
		}
		return pythonVolumeAttachWire(200, `{"volumeAttachment":{"volumeId":"v1"}}`), nil
	})
	_, err := api.UpdateRecord(context.Background(), "s1", "v1")
	if err != nil || client.Microversion != "" || !reflect.DeepEqual(versions, []string{"", "2.89"}) || len(*calls) != 0 {
		t.Fatal(err, versions, client.Microversion)
	}
	// An explicitly selected lower version is respected; no discovery or silent promotion.
	client.Microversion = "2.1"
	versions = nil
	_, err = api.UpdateRecord(context.Background(), "s1", "v1")
	if err != nil || !reflect.DeepEqual(versions, []string{"2.1"}) {
		t.Fatal(err, versions)
	}
}

type attachmentFaultBody struct {
	data        []byte
	read, close error
}

func (b *attachmentFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		return n, nil
	}
	return 0, b.read
}
func (b *attachmentFaultBody) Close() error { return b.close }

var _ io.ReadCloser = (*attachmentFaultBody)(nil)
