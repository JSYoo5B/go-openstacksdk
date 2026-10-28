package volumes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func volumeFindAPI(t *testing.T, reply func(*http.Request) *http.Response) (*volumes.API, *testcloud.Recorder) {
	t.Helper()
	api, calls := pythonVolumeAPI(t, reply)
	api.RawClient().Microversion = "3.71"
	return api, calls
}
func volumeFindField(t *testing.T, value *volumes.VolumeResourceRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s got %#v want %s", key, value, want)
	}
}

func TestPythonVolumeFindRecordGetFirstOwnsSourceViewAndRawEvidence(t *testing.T) {
	body := `{"volume":{"id":9007199254740993,"name":"data","size":"١٢","bootable":"FALSE","encrypted":true,"multiattach":"False","shared_targets":{},"attachments":{"device":false},"metadata":false,"scheduler_hints":[],"project_id":"first","os-vol-tenant-attr:tenant_id":"last","availability_zone":false,"created_at":"2026-10-11T01:02:03+09:00","unknown":[null,false],"location":{"cloud":"untrusted"}}}`
	api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
		if req.Header.Get("OpenStack-API-Version") != "volume 3.71" || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.71" {
			t.Fatal(req.Header)
		}
		return pythonVolumeWire(200, body)
	})
	value, err := api.FindRecord(context.Background(), "9007199254740993")
	if err != nil || calls.Len() != 1 || pythonVolumeCalls(calls)[0].path != pythonVolumeBase+"/9007199254740993" || pythonVolumeCalls(calls)[0].query != "" || pythonVolumeCalls(calls)[0].body != "" {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	for key, want := range map[string]string{"id": "9007199254740993", "size": "12", "is_bootable": "false", "is_encrypted": "true", "is_multiattach": "true", "shared_targets": "false", "attachments": `[{"device":false}]`, "metadata": "{}", "scheduler_hints": "{}", "project_id": `"last"`, "created_at": `"2026-10-11T01:02:03+09:00"`, "snapshot_id": "null"} {
		volumeFindField(t, value, key, want)
	}
	if len(value.Resource.Body) != 38 || string(value.Wire.Body["unknown"]) != `[null,false]` || string(value.Wire.Body["size"]) != `"١٢"` || string(value.Envelope) != body || value.StatusCode != 200 || len(value.Discovery) != 0 {
		t.Fatal(value)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(value.Resource.Body["location"], &location); err != nil || location.Cloud != nil || string(location.Project.ID) != `"last"` || string(location.Zone) != "false" {
		t.Fatal(location, err)
	}
	value.Resource.Body["id"][0] = '1'
	value.Header.Set("X-Changed", "yes")
	if string(value.Wire.Body["id"]) != "9007199254740993" || !strings.Contains(string(value.Envelope), "9007199254740993") || value.Wire.Header.Get("X-Changed") != "" || value.Resource.Header.Get("X-Changed") != "" {
		t.Fatal("record channels alias")
	}
}

func TestPythonVolumeFindRecordFallbackCodesAndDetailsDefaults(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(fmt.Sprint(status)+fmt.Sprint(details), func(t *testing.T) {
				api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
					if strings.HasSuffix(req.URL.Path, "/data") {
						return pythonVolumeWire(status, `{"error":"absent"}`)
					}
					return pythonVolumeWire(200, `{"volumes":[{"id":"v1","name":"data"}]}`)
				})
				var options []volumes.RecordFindOption
				if !details {
					options = append(options, volumes.WithRecordFindDetails(false))
				}
				value, err := api.FindRecord(context.Background(), "data", options...)
				want := pythonVolumeBase
				if details {
					want += "/detail"
				}
				if err != nil || calls.Len() != 2 || pythonVolumeCalls(calls)[0].path != pythonVolumeBase+"/data" || pythonVolumeCalls(calls)[0].query != "" || pythonVolumeCalls(calls)[1].path != want || pythonVolumeCalls(calls)[1].query != "name=data" {
					t.Fatal(value, err, pythonVolumeCalls(calls))
				}
				volumeFindField(t, value, "id", `"v1"`)
			})
		}
	}
}

func TestPythonVolumeFindRecordExactIdentityAndLateDuplicateOrError(t *testing.T) {
	for _, mode := range []string{"exact ID", "late duplicate", "late HTTP", "numeric ID is distinct"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "after" {
					switch mode {
					case "late duplicate":
						return pythonVolumeWire(200, `{"volumes":[{"id":"v2","name":"data"}]}`)
					case "late HTTP":
						return pythonVolumeWire(500, `{"error":"late unavailable"}`)
					}
					return pythonVolumeWire(200, `{"volumes":[]}`)
				}
				if mode == "numeric ID is distinct" {
					return pythonVolumeWire(200, `{"volumes":[{"id":9007199254740993,"name":"different"}]}`)
				}
				return pythonVolumeWire(200, `{"volumes":[{"id":"data","name":"different"}],"volumes_links":[{"rel":"next","href":"?marker=after"}]}`)
			})
			value, err := api.FindRecord(context.Background(), "data")
			switch mode {
			case "exact ID":
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				volumeFindField(t, value, "id", `"data"`)
			case "late duplicate":
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(value, err)
				}
			case "late HTTP":
				if value != nil || !gophercloud.ResponseCodeIs(err, 500) {
					t.Fatal(value, err)
				}
			case "numeric ID is distinct":
				if value != nil || err != nil {
					t.Fatal(value, err)
				}
			}
			wantCalls := 3
			if mode == "numeric ID is distinct" {
				wantCalls = 2
			}
			if calls.Len() != wantCalls {
				t.Fatal(pythonVolumeCalls(calls))
			}
		})
	}
	api, _ := volumeFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/9007199254740993") {
			return pythonVolumeWire(404, `{}`)
		}
		return pythonVolumeWire(200, `{"volumes":[{"id":9007199254740993,"name":"different"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "9007199254740993")
	if value != nil || err != nil {
		t.Fatal("numeric Source ID must not equal string identity", value, err)
	}
}

func TestPythonVolumeFindRecordMissingAndAcceptedTranslation(t *testing.T) {
	api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/absent") {
			return pythonVolumeWire(404, `{}`)
		}
		return pythonVolumeWire(200, `{"volumes":[]}`)
	})
	value, err := api.FindRecord(context.Background(), "absent")
	if value != nil || err != nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	value, err = api.FindRecord(context.Background(), "absent", volumes.WithRecordFindIgnoreMissing(false))
	if value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
	for _, body := range []string{"", "{bad", `{}`, `{"volume":{}}`, `{"volume":null}`, `[]`, `{"volume":{"size":"²"}}`, `{"volume":{"bootable":"yes"}}`} {
		t.Run(body, func(t *testing.T) {
			api, calls := volumeFindAPI(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, body) })
			value, err := api.FindRecord(context.Background(), "literal")
			invalid := body == `{"volume":null}` || body == `[]` || strings.Contains(body, "²") || strings.Contains(body, "yes")
			if invalid {
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || calls.Len() != 1 {
					t.Fatal(value, err, proof, pythonVolumeCalls(calls))
				}
			} else {
				if err != nil || value == nil || value.StatusCode != 200 || string(value.Envelope) != body || calls.Len() != 1 {
					t.Fatal(value, err, pythonVolumeCalls(calls))
				}
				volumeFindField(t, value, "id", `"literal"`)
			}
		})
	}
}

func TestPythonVolumeFindRecordAllProjectsIsFallbackOnlyAndFalseOmitted(t *testing.T) {
	for _, mode := range []string{"default", "false", "true"} {
		t.Run(mode, func(t *testing.T) {
			var options []volumes.RecordFindOption
			if mode != "default" {
				options = append(options, volumes.WithRecordFindAllProjects(mode == "true"))
			}
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				return pythonVolumeWire(200, `{"volumes":[{"id":"v1","name":"data"}]}`)
			})
			value, err := api.FindRecord(context.Background(), "data", options...)
			want := "name=data"
			if mode == "true" {
				want = "all_tenants=True&name=data"
			}
			if err != nil || value == nil || calls.Len() != 2 || pythonVolumeCalls(calls)[0].query != "" || pythonVolumeCalls(calls)[1].query != want {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
			api, calls = volumeFindAPI(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, `{"volume":{"id":"v1"}}`) })
			value, err = api.FindRecord(context.Background(), "data", options...)
			if err != nil || value == nil || calls.Len() != 1 || pythonVolumeCalls(calls)[0].query != "" {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
		})
	}
}

func TestPythonVolumeFindRecordQueryClassesOwnedOptsAndRawCap(t *testing.T) {
	api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/data") {
			return pythonVolumeWire(404, `{}`)
		}
		return pythonVolumeWire(200, `{"volumes":[{"id":"v1","name":"data","metadata":{"count":true,"extra":"kept"},"description":"one"},{"id":"v2","name":"data","metadata":{"count":false},"description":"two"}]}`)
	})
	query := map[string]json.RawMessage{"project_id": json.RawMessage(`["p1",null,"p2"]`), "all_projects": json.RawMessage(`true`), "metadata": json.RawMessage(`{"count":1}`), "description": json.RawMessage(`"one"`), "unknown": json.RawMessage(`"value"`), "offset": json.RawMessage(`0`)}
	option := volumes.WithRecordFindOpts(volumes.RecordFindOpts{Query: query})
	query["description"] = json.RawMessage(`"changed"`)
	value, err := api.FindRecord(context.Background(), "data", option)
	if err != nil || value == nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	volumeFindField(t, value, "id", `"v1"`)
	get, _ := url.ParseQuery(pythonVolumeCalls(calls)[0].query)
	list, _ := url.ParseQuery(pythonVolumeCalls(calls)[1].query)
	if get.Get("description") != "one" || get.Get("metadata") != "count" || get.Get("unknown") != "value" || get.Has("all_projects") || get.Get("offset") != "0" || !reflect.DeepEqual(get["project_id"], []string{"p1", "p2"}) || !reflect.DeepEqual(list, url.Values{"name": {"data"}, "project_id": {"p1", "p2"}, "all_tenants": {"True"}, "metadata": {"count"}}) {
		t.Fatal(get, list)
	}
	// metadata is both a mapped wire query and a canonical local Body filter.
	// properties wins the server alias while metadata still filters locally.
	calls.Reset()
	value, err = api.FindRecord(context.Background(), "data", volumes.WithRecordFindQuery("properties", map[string]any{"server": "hint"}), volumes.WithRecordFindQuery("metadata", map[string]any{"count": 1}))
	if err != nil || value == nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	list, _ = url.ParseQuery(pythonVolumeCalls(calls)[1].query)
	if list.Get("metadata") != "server" {
		t.Fatal(list)
	}
	volumeFindField(t, value, "id", `"v1"`)
	calls.Reset()
	value, err = api.FindRecord(context.Background(), "data", volumes.WithRecordFindQuery("description", "two"), volumes.WithRecordFindMaxItems(1))
	if err != nil || value != nil || calls.Len() != 2 || pythonVolumeCalls(calls)[1].query != "limit=1&name=data" {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	calls.Reset()
	value, err = api.FindRecord(context.Background(), "data", volumes.WithRecordFindQuery("name", nil), volumes.WithRecordFindQuery("description", "one"))
	if err != nil || value == nil || pythonVolumeCalls(calls)[1].query != "" {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
}

func TestPythonVolumeFindRecordOwnedLocationUsesRecordedScopeAndRowProjectZone(t *testing.T) {
	cloudName, region, projectName := "source", "RegionOne", "configured"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	option := volumes.WithRecordFindLocation(location)
	cloudName = "changed"
	location.Project.ID[0] = '!'
	for _, project := range []string{`"scope"`, `"foreign"`, `false`} {
		api, _ := volumeFindAPI(t, func(*http.Request) *http.Response {
			return pythonVolumeWire(200, `{"volume":{"id":"v1","project_id":`+project+`,"availability_zone":0,"location":{"cloud":"untrusted"}}}`)
		})
		value, err := api.FindRecord(context.Background(), "v1", option)
		if err != nil || value == nil {
			t.Fatal(value, err)
		}
		var computed resource.CloudLocation
		if err := json.Unmarshal(value.Resource.Body["location"], &computed); err != nil || computed.Cloud == nil || *computed.Cloud != "source" || computed.RegionName == nil || *computed.RegionName != "RegionOne" || string(computed.Zone) != "0" {
			t.Fatal(computed, err)
		}
		if project == `"foreign"` {
			if string(computed.Project.ID) != project || computed.Project.Name != nil {
				t.Fatal(computed)
			}
		} else if string(computed.Project.ID) != `"scope"` || computed.Project.Name == nil || *computed.Project.Name != "configured" {
			t.Fatal(computed)
		}
	}
}

func TestPythonVolumeFindRecordPagingShortMarkerAndHTTPLinkKeepScope(t *testing.T) {
	for _, httpLink := range []bool{false, true} {
		t.Run(fmt.Sprint(httpLink), func(t *testing.T) {
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "v1" {
					return pythonVolumeWire(200, `{"volumes":[]}`)
				}
				response := pythonVolumeWire(200, `{"volumes":[{"id":"v1","name":"data"}]}`)
				if httpLink {
					response.Header.Set("Link", "<"+req.URL.Path+`?marker=v1>; rel="next"`)
				}
				return response
			})
			options := []volumes.RecordFindOption{volumes.WithRecordFindQuery("limit", 2), volumes.WithRecordFindAllProjects(true)}
			value, err := api.FindRecord(context.Background(), "data", options...)
			if err != nil || value == nil || calls.Len() != 3 || pythonVolumeCalls(calls)[2].query != "all_tenants=True&limit=2&marker=v1&name=data" {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
			calls.Reset()
			value, err = api.FindRecord(context.Background(), "data", append(options, volumes.WithRecordFindPaginated(false))...)
			if err != nil || value == nil || calls.Len() != 2 {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
		})
	}
}

func TestPythonVolumeFindRecordMarkerUsesRetainedUntypedWireID(t *testing.T) {
	api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/data") {
			return pythonVolumeWire(404, `{}`)
		}
		if req.URL.Query().Get("marker") == "9007199254740993" {
			return pythonVolumeWire(200, `{"volumes":[]}`)
		}
		return pythonVolumeWire(200, `{"volumes":[{"id":9007199254740993,"name":"data"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "data", volumes.WithRecordFindQuery("limit", 2))
	if err != nil || value == nil || calls.Len() != 3 || pythonVolumeCalls(calls)[2].query != "limit=2&marker=9007199254740993&name=data" {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	volumeFindField(t, value, "id", "9007199254740993")
	if string(value.Wire.Body["id"]) != "9007199254740993" {
		t.Fatal("marker conversion mutated untyped ID", value)
	}
}

func TestPythonVolumeFindRecordRejectedAndAcceptedPhysicalCauses(t *testing.T) {
	for _, phase := range []string{"member400", "member403", "member404", "member500", "list500", "accepted list"} {
		for _, fault := range []string{"read", "close", "mixed"} {
			t.Run(phase+"/"+fault, func(t *testing.T) {
				readCause, closeCause := errors.New("volume read"), errors.New("volume close")
				var body *connectorFaultBody
				var closes atomic.Int32
				api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
					if (phase == "list500" || phase == "accepted list") && strings.HasSuffix(req.URL.Path, "/data") {
						return pythonVolumeWire(404, `{}`)
					}
					status, raw := 500, `{"error":"actual rejected"}`
					switch phase {
					case "member400":
						status = 400
					case "member403":
						status = 403
					case "member404":
						status = 404
					case "accepted list":
						status = 200
						raw = `{"volumes":[{"id":"v1","name":"data"}]}`
					}
					response := pythonVolumeWire(status, raw)
					body = &connectorFaultBody{ReadCloser: response.Body, closes: &closes}
					if fault != "close" {
						body.readErr = readCause
					}
					if fault != "read" {
						body.closeErr = closeCause
					}
					response.Header.Set("X-Request-Id", "volume-find-proof")
					response.Body = body
					return response
				})
				api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					return original
				}
				value, err := api.FindRecord(context.Background(), "data")
				if value != nil || errors.Is(err, readCause) != (fault != "close") || errors.Is(err, closeCause) != (fault != "read") || body == nil || closes.Load() != 1 {
					t.Fatal(value, err, body)
				}
				want := 1
				if phase == "list500" || phase == "accepted list" {
					want = 2
				}
				if calls.Len() != want {
					t.Fatal("faulty observation resent or allowed fallback", pythonVolumeCalls(calls))
				}
				if phase == "accepted list" {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "volume-find-proof" || string(proof.Body) != `{"volumes":[{"id":"v1","name":"data"}]}` {
						t.Fatal(err, proof)
					}
				} else {
					var proof gophercloud.ErrUnexpectedResponseCode
					expected := 500
					switch phase {
					case "member400":
						expected = 400
					case "member403":
						expected = 403
					case "member404":
						expected = 404
					}
					if !errors.As(err, &proof) || proof.Actual != expected || proof.ResponseHeader.Get("X-Request-Id") != "volume-find-proof" || string(proof.Body) != `{"error":"actual rejected"}` {
						t.Fatal(err, proof)
					}
				}
			})
		}
	}
}

func TestPythonVolumeFindRecordResponseCloseSourceChangeNeverFallsBack(t *testing.T) {
	for _, status := range []int{200, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("volumev3", "/cinder/v3/project/")
			client.Microversion = "3.71"
			calls := 0
			var closes atomic.Int32
			cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
				calls++
				response := pythonVolumeWire(status, `{"volume":{"id":"v1"}}`)
				response.Header.Set("X-Request-Id", "source-proof")
				response.Body = &connectorFaultBody{ReadCloser: response.Body, closes: &closes, afterClose: func() { client.ResourceBase = cloud.Server.URL + "/changed/" }}
				return response, nil
			}))
			value, err := volumes.New(client).FindRecord(context.Background(), "v1")
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || closes.Load() != 1 {
				t.Fatal(value, err, calls, closes.Load())
			}
			if status == 200 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Request-Id") != "source-proof" {
					t.Fatal(err, proof)
				}
			} else {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != status || proof.ResponseHeader.Get("X-Request-Id") != "source-proof" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestPythonVolumeFindRecordSelectionAndOptionFailures(t *testing.T) {
	api, calls := volumeFindAPI(t, func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
	cases := []struct {
		identity string
		options  []volumes.RecordFindOption
	}{
		{"..", nil}, {"data", []volumes.RecordFindOption{nil}}, {"data", []volumes.RecordFindOption{volumes.WithRecordFindMaxItems(-1)}},
		{"data", []volumes.RecordFindOption{volumes.WithRecordFindQuery("id", "collision")}},
		{"data", []volumes.RecordFindOption{volumes.WithRecordFindAllProjects(false), volumes.WithRecordFindQuery("all_tenants", nil)}},
		{"data", []volumes.RecordFindOption{volumes.WithRecordFindQuery("all_projects", true), volumes.WithRecordFindAllProjects(true)}},
		{"data", []volumes.RecordFindOption{volumes.WithRecordFindLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{bad`)}})}},
		{"data", []volumes.RecordFindOption{volumes.WithRecordFindOpts(volumes.RecordFindOpts{Query: map[string]json.RawMessage{"metadata": json.RawMessage(`{bad`)}})}},
	}
	for _, item := range cases {
		if _, err := api.FindRecord(context.Background(), item.identity, item.options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Len() != 0 {
		t.Fatal(pythonVolumeCalls(calls))
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("user cancelled lookup")
	_, err := api.FindRecord(ctx, "data", func(*volumes.RecordFindOpts) error { cancel(cause); return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Len() != 0 {
		t.Fatal(err, pythonVolumeCalls(calls))
	}
	_, err = api.FindRecord(context.Background(), "data", func(*volumes.RecordFindOpts) error { api.RawClient().Microversion = "3.43"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, pythonVolumeCalls(calls))
	}
}

func TestPythonVolumeFindRecordDiscoveryDefaultAndExplicitVersion(t *testing.T) {
	for _, item := range []struct{ selected, maximum, want string }{{"", "3.80", "3.71"}, {"", "3.43", "3.43"}, {"3.80", "3.43", "3.80"}} {
		t.Run(item.selected+item.maximum, func(t *testing.T) {
			cloud := testcloud.New(t)
			var paths, versions []string
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				paths = append(paths, req.URL.Path)
				versions = append(versions, req.Header.Get("X-OpenStack-Volume-API-Version"))
				if req.URL.Path == "/cinder/v3/" {
					return pythonVolumeWire(200, `{"version":{"id":"v3.0","max_version":"`+item.maximum+`","min_version":"3.0"}}`), nil
				}
				return pythonVolumeWire(200, `{"volume":{"id":"v1"}}`), nil
			}))
			client := cloud.Client("volumev3", "/cinder/v3/project/")
			client.Microversion = item.selected
			value, err := volumes.New(client).FindRecord(context.Background(), "v1")
			if err != nil || value == nil || versions[len(versions)-1] != item.want || client.Microversion != item.selected {
				t.Fatal(value, err, paths, versions, client)
			}
			if item.selected == "" {
				if len(paths) != 2 || paths[0] != "/cinder/v3/" || versions[0] != "" || len(value.Discovery) != 1 || value.Discovery[0].StatusCode != 200 || value.StatusCode != 200 {
					t.Fatal(value, paths, versions)
				}
			} else if len(paths) != 1 || len(value.Discovery) != 0 {
				t.Fatal("explicit default must bypass discovery", paths, value)
			}
		})
	}
}

func TestPythonVolumeFindRecordDiscoveryRejectionPreservesPhysicalCauses(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	readCause, closeCause := errors.New("discovery read"), errors.New("discovery close")
	var closes atomic.Int32
	body := &connectorFaultBody{ReadCloser: pythonVolumeWire(500, `{"error":"discovery unavailable"}`).Body, readErr: readCause, closeErr: closeCause, closes: &closes}
	cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
		calls++
		r := pythonVolumeWire(500, "")
		r.Body = body
		r.Header.Set("X-Request-Id", "discovery-proof")
		return r, nil
	}))
	value, err := volumes.New(cloud.Client("volumev3", "/cinder/v3/project/")).FindRecord(context.Background(), "v1")
	var proof gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.Actual != 500 || proof.ResponseHeader.Get("X-Request-Id") != "discovery-proof" || string(proof.Body) != `{"error":"discovery unavailable"}` || calls != 1 || closes.Load() != 1 {
		t.Fatal(value, err, proof, calls, body)
	}
}

func TestPythonVolumeFindRecordQuerySeedCanonicalPriorityAndNullOmission(t *testing.T) {
	api, calls := volumeFindAPI(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, `{"volume":{}}`) })
	option := volumes.WithRecordFindOpts(volumes.RecordFindOpts{Query: map[string]json.RawMessage{"image_id": json.RawMessage(`"canonical"`), "imageRef": json.RawMessage(`"wire"`), "shared_targets": json.RawMessage(`false`), "size": json.RawMessage(`0`), "name": json.RawMessage(`null`)}})
	value, err := api.FindRecord(context.Background(), "v1", option)
	if err != nil || value == nil || calls.Len() != 1 {
		t.Fatal(value, err, pythonVolumeCalls(calls))
	}
	volumeFindField(t, value, "id", `"v1"`)
	volumeFindField(t, value, "image_id", `"canonical"`)
	volumeFindField(t, value, "shared_targets", `false`)
	volumeFindField(t, value, "size", `0`)
	volumeFindField(t, value, "name", `null`)
	query, _ := url.ParseQuery(pythonVolumeCalls(calls)[0].query)
	if query.Has("name") || query.Get("shared_targets") != "False" || query.Get("size") != "0" || query.Get("image_id") != "canonical" || query.Get("imageRef") != "wire" {
		t.Fatal(query)
	}
	if len(value.Wire.Body) != 0 {
		t.Fatal("seed was presented as HTTP evidence", value.Wire)
	}
}

func TestPythonVolumeFindRecordConstructorSeedDescriptorsBeforeDiscoveryOrHTTP(t *testing.T) {
	for _, selected := range []string{"", "3.71"} {
		for _, item := range []struct {
			key   string
			value any
		}{{"is_bootable", "yes"}, {"bootable", "yes"}, {"is_encrypted", 1}, {"size", "²"}} {
			t.Run(selected+"/"+item.key, func(t *testing.T) {
				api, calls := volumeFindAPI(t, func(*http.Request) *http.Response {
					t.Fatal("invalid constructor seed reached discovery/member HTTP")
					return pythonVolumeWire(200, `{"volume":{"id":"v1","bootable":"false","encrypted":false,"size":1}}`)
				})
				api.RawClient().Microversion = selected
				value, err := api.FindRecord(context.Background(), "v1", volumes.WithRecordFindQuery(item.key, item.value))
				var response *resource.ResponseError
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &response) || calls.Len() != 0 {
					t.Fatal(value, err, pythonVolumeCalls(calls))
				}
			})
		}
	}
}

func TestPythonVolumeFindRecordAcceptsSourceInformationalStatus199(t *testing.T) {
	for _, phase := range []string{"member", "list"} {
		t.Run(phase, func(t *testing.T) {
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if phase == "list" && strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				body := `{"volume":{"id":"v1","name":"data"}}`
				if phase == "list" {
					body = `{"volumes":[{"id":"v1","name":"data"}]}`
				}
				return pythonVolumeWire(199, body)
			})
			value, err := api.FindRecord(context.Background(), "data")
			want := 1
			if phase == "list" {
				want = 2
			}
			if err != nil || value == nil || value.StatusCode != 199 || calls.Len() != want || value.Resource.StatusCode != 199 || value.Wire.StatusCode != 199 {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
			volumeFindField(t, value, "id", `"v1"`)
		})
	}
}

func TestPythonVolumeFindRecordRejectedMixedFaultsNilRetryStickySourceAndCancel(t *testing.T) {
	for _, phase := range []string{"member", "list"} {
		for _, status := range []int{429, 503} {
			for _, mixed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/sticky-cancel-%v", phase, status, mixed), func(t *testing.T) {
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readCause, closeCause, cancelCause := errors.New("mixed volume read"), errors.New("mixed volume close"), errors.New("custom volume cancellation")
					var closes atomic.Int32
					var api *volumes.API
					original := ""
					api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
						if phase == "list" && strings.HasSuffix(req.URL.Path, "/data") {
							return pythonVolumeWire(404, `{}`)
						}
						response := pythonVolumeWire(status, `{"failure":"original rejected volume"}`)
						response.Header.Set("X-Request-Id", "mixed-original-proof")
						body := &connectorFaultBody{ReadCloser: response.Body, readErr: readCause, closeErr: closeCause, closes: &closes}
						if mixed {
							body.afterRead = func() { api.RawClient().ResourceBase = original + "changed/" }
							body.afterClose = func() { api.RawClient().ResourceBase = original; cancel(cancelCause) }
						}
						response.Body = body
						return response
					})
					original = api.RawClient().ResourceBase
					retryCalls := 0
					hook := func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, originalErr error, _ uint) error {
						// Permit the initial clean404 to enter Source fallback;
						// the actual physical rejection's native hook says retry.
						if gophercloud.ResponseCodeIs(originalErr, 404) {
							return originalErr
						}
						retryCalls++
						return nil
					}
					api.RawClient().RetryFunc = hook
					value, err := api.FindRecord(ctx, "data")
					var native gophercloud.ErrUnexpectedResponseCode
					var accepted *resource.ResponseError
					target := original + "volumes/data"
					want := 1
					if phase == "list" {
						target = original + "volumes/detail?name=data"
						want = 2
					}
					if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || errors.As(err, &accepted) || native.Actual != status || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.Method != http.MethodGet || native.URL != target || string(native.Body) != `{"failure":"original rejected volume"}` || native.ResponseHeader.Get("X-Request-Id") != "mixed-original-proof" || calls.Len() != want || closes.Load() != 1 || retryCalls > 1 || reflect.ValueOf(api.RawClient().RetryFunc).Pointer() != reflect.ValueOf(hook).Pointer() {
						t.Fatal(value, err, native, pythonVolumeCalls(calls), closes.Load(), retryCalls)
					}
					if mixed {
						if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || api.RawClient().ResourceBase != original {
							t.Fatal("joined sticky/cancel cause lost", err)
						}
					}
				})
			}
		}
	}
}

func TestPythonVolumeFindRecordListConstructorCollisionsKeepActualPage(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		for _, raw := range []string{"null", "false"} {
			for _, late := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/late-%v", key, raw, late), func(t *testing.T) {
					body := `{"volumes":[{"id":"v2","name":"data","` + key + `":` + raw + `}]}`
					api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
						if strings.HasSuffix(req.URL.Path, "/data") {
							return pythonVolumeWire(404, `{}`)
						}
						if late && !req.URL.Query().Has("marker") {
							return pythonVolumeWire(200, `{"volumes":[{"id":"v1","name":"data"}],"next":"?marker=second"}`)
						}
						response := pythonVolumeWire(200, body)
						response.Header.Set("X-Request-Id", "constructor-page-proof")
						return response
					})
					value, err := api.FindRecord(context.Background(), "data")
					var actual *resource.ResponseError
					var native gophercloud.ErrUnexpectedResponseCode
					wantCalls := 2
					if late {
						wantCalls = 3
					}
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &actual) || errors.As(err, &native) || actual.StatusCode != 200 || actual.Header.Get("X-Request-Id") != "constructor-page-proof" || string(actual.Body) != body || calls.Len() != wantCalls || !strings.Contains(err.Error(), key) {
						t.Fatal(value, err, actual, pythonVolumeCalls(calls))
					}
				})
			}
		}
	}
	// Source drops self before list construction, but fetch merges responses
	// without invoking the list constructor. All literal fields remain Wire.
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprintf("self-and-member/direct-%v", direct), func(t *testing.T) {
			row := `{"id":"v1","name":"data","self":null}`
			if direct {
				row = `{"id":"v1","name":"data","self":null,"connection":null,"microversion":false,"_synchronized":null,"cls":false}`
			}
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if !direct && strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				if direct {
					return pythonVolumeWire(200, `{"volume":`+row+`}`)
				}
				return pythonVolumeWire(200, `{"volumes":[`+row+`]}`)
			})
			value, err := api.FindRecord(context.Background(), "data")
			wantCalls := 2
			if direct {
				wantCalls = 1
			}
			if err != nil || value == nil || calls.Len() != wantCalls || string(value.Wire.Body["self"]) != "null" {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
			if _, present := value.Resource.Body["self"]; present {
				t.Fatal("self leaked into declared Resource")
			}
		})
	}
}

func TestPythonVolumeFindRecordExactDecimalEqualityAndUnderflowAreExplicitGoPolicy(t *testing.T) {
	// Source json.loads rounds 9007199254740993.0 to binary64's
	// 9007199254740992.0, equal to the integer below. The shared Go helper
	// intentionally compares exact decimals, so that rounded match is absent.
	// Source bool(json.loads("1e-400")) is false; Go's response bool translator
	// retains exact nonzero truthiness and projects multiattach as true.
	for _, tc := range []struct {
		name, row, key, filter string
		found                  bool
	}{
		{"rounded-float-does-not-match", `"metadata":{"count":9007199254740992}`, "metadata", `{"count":9007199254740993.0}`, false},
		{"same-exact-decimal-matches", `"metadata":{"count":9007199254740992}`, "metadata", `{"count":9007199254740992.0}`, true},
		{"ordinary-bool-number-matches", `"metadata":{"count":true}`, "metadata", `{"count":1}`, true},
		{"underflow-does-not-match-false", `"multiattach":1e-400`, "is_multiattach", `false`, false},
		{"underflow-matches-true", `"multiattach":1e-400`, "is_multiattach", `true`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"volumes":[{"id":"v1","name":"data",` + tc.row + `}]}`
			api, calls := volumeFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/data") {
					return pythonVolumeWire(404, `{}`)
				}
				return pythonVolumeWire(200, body)
			})
			value, err := api.FindRecord(context.Background(), "data", volumes.WithRecordFindQuery(tc.key, json.RawMessage(tc.filter)))
			if err != nil || (value != nil) != tc.found || calls.Len() != 2 {
				t.Fatal(value, err, pythonVolumeCalls(calls))
			}
			query, _ := url.ParseQuery(pythonVolumeCalls(calls)[1].query)
			if query.Get("name") != "data" || query.Has("is_multiattach") || (tc.key == "metadata" && query.Get("metadata") != "count") {
				t.Fatal("local/server classification changed", query)
			}
			if value != nil && (string(value.Envelope) != body || value.StatusCode != 200) {
				t.Fatal("actual response changed", value)
			}
			if tc.key == "is_multiattach" && value != nil {
				volumeFindField(t, value, "is_multiattach", "true")
				if string(value.Wire.Body["multiattach"]) != "1e-400" {
					t.Fatal("literal underflow value lost", value.Wire)
				}
			}
		})
	}
}
