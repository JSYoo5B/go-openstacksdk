package snapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"strings"
	"testing"
)

func TestPythonSnapshotManageRouteCapabilityAndFixedVersion(t *testing.T) {
	for _, version := range []string{"3.7", "3.8", ""} {
		t.Run("selected"+version, func(t *testing.T) {
			api, calls := pythonSnapshotAPI(t, version, func(r *http.Request) *http.Response {
				if r.Method == "GET" {
					return pythonSnapshotWire(200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.90"}}`)
				}
				return pythonSnapshotWire(202, `{"snapshot":{"id":"managed","size":"١٢","metadata":42,"extra":true}}`)
			})
			row, err := api.ManageRecord(context.Background(), "volume", json.RawMessage(`{"source-name":"backend"}`))
			if err != nil || row == nil || len(*calls) != 2 || len(row.Discovery) != 1 {
				t.Fatal(row, err, *calls)
			}
			last := (*calls)[1]
			path := "/manageable_snapshots"
			if version == "3.7" {
				path = "/os-snapshot-manage"
			}
			if !strings.HasSuffix(last.path, path) || last.version != "volume 3.65" || last.method != "POST" {
				t.Fatal(last)
			}
			var body map[string]map[string]json.RawMessage
			if err := json.Unmarshal([]byte(last.body), &body); err != nil {
				t.Fatal(err)
			}
			fields := body["snapshot"]
			if len(fields) != 5 || string(fields["name"]) != "null" || string(fields["metadata"]) != "null" || string(fields["volume_id"]) != `"volume"` {
				t.Fatal(fields)
			}
			snapshotFindField(t, row, "size", "12")
			snapshotFindField(t, row, "metadata", "{}")
			if row.Resource.Body["extra"] != nil || string(row.Wire.Body["extra"]) != "true" {
				t.Fatal(row)
			}
		})
	}
}
func TestPythonSnapshotManageResponseMissingIDAndOptionsOwned(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.90", func(r *http.Request) *http.Response {
		if r.Method == "GET" {
			return pythonSnapshotWire(200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.90"}}`)
		}
		if r.Header.Get("X-Test") != "owned" {
			t.Error(r.Header)
		}
		return pythonSnapshotWire(203, `{"snapshot":{"name":"actual"}}`)
	})
	metadata := json.RawMessage(`{"x":1}`)
	headers := map[string]string{"X-Test": "owned"}
	option := snapshots.WithManageRecordOpts(snapshots.ManageRecordOpts{Metadata: metadata, Headers: headers})
	metadata[5] = '9'
	headers["X-Test"] = "changed"
	row, err := api.ManageRecord(context.Background(), "", json.RawMessage(`null`), option, snapshots.WithManageRecordName(""))
	if err != nil || row == nil || len(*calls) != 2 {
		t.Fatal(row, err, *calls)
	}
	snapshotFindField(t, row, "id", "null")
	if !strings.Contains((*calls)[1].body, `"metadata":{"x":1}`) || !strings.Contains((*calls)[1].body, `"name":""`) {
		t.Fatal(*calls)
	}
	before := len(*calls)
	_, err = api.ManageRecord(context.Background(), "v", json.RawMessage(`{bad`))
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != before {
		t.Fatal(err, *calls)
	}
}
func TestPythonSnapshotUnmanageOpaqueAckFixedHeaderAndMissing(t *testing.T) {
	for _, status := range []int{202, 299, 404} {
		api, calls := pythonSnapshotAPI(t, "3.1", func(*http.Request) *http.Response { return pythonSnapshotWire(status, "opaque acknowledgement") })
		receipt, err := api.UnmanageRecord(context.Background(), "snap/id")
		if len(*calls) != 1 || (*calls)[0].body != `{"os-unmanage":null}` || (*calls)[0].version != "volume 3.65" {
			t.Fatal(*calls)
		}
		if status == 404 {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 404 {
				t.Fatal(err)
			}
		} else if err != nil || receipt == nil || receipt.StatusCode != status || string(receipt.Envelope) != "opaque acknowledgement" {
			t.Fatal(receipt, err)
		}
	}
}
func TestPythonSnapshotManagementGuardsAndAcceptedProjectionError(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.65", func(r *http.Request) *http.Response {
		if r.Method == "GET" {
			return pythonSnapshotWire(200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.90"}}`)
		}
		return pythonSnapshotWire(202, `{"snapshot":{"size":"²"}}`)
	})
	row, err := api.ManageRecord(context.Background(), "v", json.RawMessage(`{}`))
	var proof *resource.ResponseError
	if row == nil || row.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 202 {
		t.Fatal(row, err)
	}
	before := len(*calls)
	_, err = api.UnmanageRecord(context.Background(), "s", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != before {
		t.Fatal(err, *calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = api.UnmanageRecord(ctx, "s")
	if !errors.Is(err, context.Canceled) || len(*calls) != before {
		t.Fatal(err)
	}
}

func TestPythonSnapshotManagementRetriesOwnFixedVersionAndHeaders(t *testing.T) {
	for _, action := range []string{"manage", "unmanage"} {
		for _, mode := range []string{"clean", "replace version", "omit version"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				actions := 0
				api, calls := pythonSnapshotAPI(t, "3.1", func(r *http.Request) *http.Response {
					if r.Method == http.MethodGet {
						return pythonSnapshotWire(200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.90"}}`)
					}
					actions++
					if r.Header.Get("OpenStack-API-Version") != "volume 3.65" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.65" || r.Header.Get("X-Owned") != "selected" {
						t.Error("request escaped fixed header policy", r.Header)
					}
					if actions == 1 {
						return pythonSnapshotWire(503, "first503")
					}
					if action == "manage" {
						return pythonSnapshotWire(202, `{"snapshot":{"id":"accepted"}}`)
					}
					return pythonSnapshotWire(202, "ack")
				})
				callbacks := 0
				api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
					callbacks++
					switch mode {
					case "replace version":
						if opts.MoreHeaders == nil {
							opts.MoreHeaders = map[string]string{}
						}
						opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.1"
					case "omit version":
						opts.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"}
					}
					return nil
				}
				var err error
				if action == "manage" {
					_, err = api.ManageRecord(context.Background(), "v", json.RawMessage(`{}`), snapshots.WithManageRecordHeaders(map[string]string{"X-Owned": "selected"}))
				} else {
					_, err = api.UnmanageRecord(context.Background(), "s", snapshots.WithUnmanageRecordHeaders(map[string]string{"X-Owned": "selected"}))
				}
				if callbacks != 1 {
					t.Fatal(callbacks, err)
				}
				if mode == "clean" {
					if err != nil || actions != 2 {
						t.Fatal(err, actions, *calls)
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "first503" || actions != 1 {
					t.Fatal("retry mutation lost native rejection or transmitted changed headers", err, actions, *calls)
				}
			})
		}
	}
}
