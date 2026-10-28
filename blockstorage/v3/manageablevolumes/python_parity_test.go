package manageablevolumes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonManageCall records the wire request openstacksdk would also send.
type pythonManageCall struct{ method, path, query, body, version string }

func pythonManageWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonManageAPI(t *testing.T, microversion string) (*manageablevolumes.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) {
		return pythonManageWire(202, `{"volume":{"id":"vol-9","name":"imported","status":"creating","size":0,"os-vol-host-attr:host":"node@lvm#pool","availability_zone":"nova","bootable":"true","metadata":{"k":"v"},"volume_type":"lvm","created_at":"2026-10-11T01:02:03.000000"}}`), nil
	})
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = microversion
	return manageablevolumes.New(client), calls
}

// pythonManageCalls projects recorded calls onto the local call fields.
func pythonManageCalls(rec *testcloud.Recorder) []pythonManageCall {
	calls := []pythonManageCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonManageCall{call.Method, call.Path, call.RawQuery, call.Body, call.Header.Get("OpenStack-API-Version")})
	}
	return calls
}

func pythonManageBody(t *testing.T, raw string) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(raw, err)
	}
	return body
}

func TestPythonManageVolumeRequest(t *testing.T) {
	ctx := context.Background()
	t.Run("all arguments", func(t *testing.T) {
		// Volume.manage always posts at the fixed Volume._max_microversion 3.71.
		api, calls := pythonManageAPI(t, "3.71")
		got, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{
			Host: "node@lvm#pool", Cluster: "c1", Ref: map[string]string{"source-name": "vol-1"}, Name: "imported", Description: "d",
			VolumeType: "lvm", AvailabilityZone: "nova", Metadata: map[string]string{"k": "v"}, Bootable: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if calls.Len() != 1 {
			t.Fatal(pythonManageCalls(calls))
		}
		call := pythonManageCalls(calls)[0]
		if call.method != http.MethodPost || call.path != "/cinder/v3/project/manageable_volumes" || call.query != "" || call.version != "volume 3.71" {
			t.Fatalf("%+v", call)
		}
		// manage_volume(host=..., ref=..., name=..., description=..., volume_type=..., availability_zone=..., metadata=..., bootable=True, cluster=...)
		python := pythonManageBody(t, `{"volume":{"host":"node@lvm#pool","ref":{"source-name":"vol-1"},"name":"imported","description":"d","volume_type":"lvm","availability_zone":"nova","metadata":{"k":"v"},"bootable":true,"cluster":"c1"}}`)
		if body := pythonManageBody(t, call.body); !reflect.DeepEqual(body, python) {
			t.Fatalf("%v", body)
		}
		if !(got.ID == "vol-9" && got.Name == "imported" && got.Status == "creating" && got.Host == "node@lvm#pool" && got.AvailabilityZone == "nova" &&
			got.Bootable == "true" && got.Metadata["k"] == "v" && got.VolumeType == "lvm") {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("python defaults", func(t *testing.T) {
		api, calls := pythonManageAPI(t, "")
		if _, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "node@lvm#pool", Ref: map[string]string{"source-name": "vol-1"}}); err != nil {
			t.Fatal(err)
		}
		// Python also sends null name/description/volume_type/availability_zone/metadata and bootable=false; Cinder reads omitted keys the same way.
		want := pythonManageBody(t, `{"volume":{"host":"node@lvm#pool","ref":{"source-name":"vol-1"}}}`)
		if calls.Len() != 1 || !reflect.DeepEqual(pythonManageBody(t, pythonManageCalls(calls)[0].body), want) || pythonManageCalls(calls)[0].version != "" {
			t.Fatalf("%+v", pythonManageCalls(calls))
		}
	})
	t.Run("explicit null is not reachable", func(t *testing.T) {
		api, calls := pythonManageAPI(t, "")
		_, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "node@lvm#pool", Ref: map[string]string{"source-name": "vol-1"}},
			manageablevolumes.WithManageExistingField("name", nil))
		if err == nil || calls.Len() != 0 {
			t.Fatal(err, pythonManageCalls(calls))
		}
	})
}
