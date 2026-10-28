package backups_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func backupFindAPI(t *testing.T, reply func(*http.Request) *http.Response) (*backups.API, *testcloud.Recorder) {
	t.Helper()
	api, calls := pythonBackupAPI(t, reply)
	api.RawClient().Microversion = "3.64"
	return api, calls
}
func backupFindField(t *testing.T, value *backups.BackupResourceRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s got %#v want %s", key, value, want)
	}
}

func TestPythonBackupFindRecordGetFirstOwnsSourceViewAndRawEvidence(t *testing.T) {
	body := `{"backup":{"id":9007199254740993,"name":"nightly","size":"١٢","object_count":true,"force":"False","has_dependent_backups":{},"is_incremental":[],"links":"self","metadata":false,"project_id":"first","os-backup-project-attr:project_id":"last","created_at":"2026-10-11T01:02:03+09:00","unknown":[null,false]}}`
	api, calls := backupFindAPI(t, func(*http.Request) *http.Response { return pythonBackupWire(200, body) })
	value, err := api.FindRecord(context.Background(), "9007199254740993")
	if err != nil || calls.Len() != 1 || pythonBackupCalls(calls)[0].path != pythonBackupBase+"/9007199254740993" || pythonBackupCalls(calls)[0].query != "" || pythonBackupCalls(calls)[0].body != "" || pythonBackupCalls(calls)[0].version != "volume 3.64" {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
	for key, want := range map[string]string{"id": "9007199254740993", "size": "12", "object_count": "true", "force": "true", "has_dependent_backups": "false", "is_incremental": "false", "links": `["self"]`, "metadata": "{}", "project_id": `"last"`, "created_at": `"2026-10-11T01:02:03+09:00"`, "snapshot_id": "null", "location": "null"} {
		backupFindField(t, value, key, want)
	}
	if len(value.Resource.Body) != 24 || string(value.Wire.Body["unknown"]) != `[null,false]` || string(value.Wire.Body["size"]) != `"١٢"` || string(value.Envelope) != body || value.StatusCode != 200 || len(value.Discovery) != 0 {
		t.Fatal(value)
	}
	value.Resource.Body["id"][0] = '1'
	value.Header.Set("X-Changed", "yes")
	if string(value.Wire.Body["id"]) != "9007199254740993" || !strings.Contains(string(value.Envelope), "9007199254740993") || value.Wire.Header.Get("X-Changed") != "" || value.Resource.Header.Get("X-Changed") != "" {
		t.Fatal("record channels alias")
	}
}

func TestPythonBackupFindRecordFallbackCodesAndDetailsDefaults(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(fmt.Sprint(status)+fmt.Sprint(details), func(t *testing.T) {
				api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
					if strings.HasSuffix(req.URL.Path, "/nightly") {
						return pythonBackupWire(status, `{"error":"absent"}`)
					}
					return pythonBackupWire(200, `{"backups":[{"id":"b1","name":"nightly"}]}`)
				})
				var options []backups.RecordFindOption
				if !details {
					options = append(options, backups.WithRecordFindDetails(false))
				}
				value, err := api.FindRecord(context.Background(), "nightly", options...)
				want := pythonBackupBase
				if details {
					want += "/detail"
				}
				if err != nil || calls.Len() != 2 || pythonBackupCalls(calls)[0].path != pythonBackupBase+"/nightly" || pythonBackupCalls(calls)[0].query != "" || pythonBackupCalls(calls)[1].path != want || pythonBackupCalls(calls)[1].query != "name=nightly" {
					t.Fatal(value, err, pythonBackupCalls(calls))
				}
				backupFindField(t, value, "id", `"b1"`)
			})
		}
	}
}

func TestPythonBackupFindRecordExactIdentityAndLateDuplicateOrError(t *testing.T) {
	for _, mode := range []string{"exact ID", "late duplicate", "late HTTP", "numeric ID is distinct"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/nightly") {
					return pythonBackupWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "after" {
					switch mode {
					case "late duplicate":
						return pythonBackupWire(200, `{"backups":[{"id":"b2","name":"nightly"}]}`)
					case "late HTTP":
						return pythonBackupWire(500, `{"error":"late unavailable"}`)
					}
					return pythonBackupWire(200, `{"backups":[]}`)
				}
				if mode == "numeric ID is distinct" {
					return pythonBackupWire(200, `{"backups":[{"id":9007199254740993,"name":"different"}]}`)
				}
				return pythonBackupWire(200, `{"backups":[{"id":"nightly","name":"different"}],"backups_links":[{"rel":"next","href":"?marker=after"}]}`)
			})
			value, err := api.FindRecord(context.Background(), "nightly")
			switch mode {
			case "exact ID":
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				backupFindField(t, value, "id", `"nightly"`)
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
				t.Fatal(pythonBackupCalls(calls))
			}
		})
	}
	api, _ := backupFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/9007199254740993") {
			return pythonBackupWire(404, `{}`)
		}
		return pythonBackupWire(200, `{"backups":[{"id":9007199254740993,"name":"different"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "9007199254740993")
	if value != nil || err != nil {
		t.Fatal("numeric Source ID must not equal string identity", value, err)
	}
}

func TestPythonBackupFindRecordMissingAndAcceptedTranslation(t *testing.T) {
	api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/absent") {
			return pythonBackupWire(404, `{}`)
		}
		return pythonBackupWire(200, `{"backups":[]}`)
	})
	value, err := api.FindRecord(context.Background(), "absent")
	if value != nil || err != nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
	value, err = api.FindRecord(context.Background(), "absent", backups.WithRecordFindIgnoreMissing(false))
	if value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
	for _, body := range []string{"", "{bad", `{}`, `{"backup":{}}`, `{"backup":null}`, `[]`, `{"backup":{"size":"²"}}`} {
		t.Run(body, func(t *testing.T) {
			api, calls := backupFindAPI(t, func(*http.Request) *http.Response { return pythonBackupWire(200, body) })
			value, err := api.FindRecord(context.Background(), "literal")
			invalid := body == `{"backup":null}` || body == `[]` || strings.Contains(body, "²")
			if invalid {
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || calls.Len() != 1 {
					t.Fatal(value, err, proof, pythonBackupCalls(calls))
				}
			} else {
				if err != nil || value == nil || value.StatusCode != 200 || string(value.Envelope) != body || calls.Len() != 1 {
					t.Fatal(value, err, pythonBackupCalls(calls))
				}
				backupFindField(t, value, "id", `"literal"`)
			}
		})
	}
}

func TestPythonBackupFindRecordQueryClassesOwnedOptsAndRawCap(t *testing.T) {
	api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/nightly") {
			return pythonBackupWire(404, `{}`)
		}
		return pythonBackupWire(200, `{"backups":[{"id":"b1","name":"nightly","metadata":{"count":true,"extra":"kept"},"container":"one"},{"id":"b2","name":"nightly","metadata":{"count":false},"container":"two"}]}`)
	})
	query := map[string]json.RawMessage{"project_id": json.RawMessage(`["p1",null,"p2"]`), "all_projects": json.RawMessage(`true`), "metadata": json.RawMessage(`{"count":1}`), "container": json.RawMessage(`"one"`), "unknown": json.RawMessage(`"value"`)}
	option := backups.WithRecordFindOpts(backups.RecordFindOpts{Query: query})
	query["container"] = json.RawMessage(`"changed"`)
	value, err := api.FindRecord(context.Background(), "nightly", option)
	if err != nil || value == nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
	backupFindField(t, value, "id", `"b1"`)
	get, _ := url.ParseQuery(pythonBackupCalls(calls)[0].query)
	list, _ := url.ParseQuery(pythonBackupCalls(calls)[1].query)
	if get.Get("container") != "one" || get.Get("metadata") != "count" || get.Get("unknown") != "value" || get.Has("all_projects") || !reflect.DeepEqual(get["project_id"], []string{"p1", "p2"}) || !reflect.DeepEqual(list, url.Values{"name": {"nightly"}, "project_id": {"p1", "p2"}, "all_tenants": {"True"}}) {
		t.Fatal(get, list)
	}
	calls.Reset()
	value, err = api.FindRecord(context.Background(), "nightly", backups.WithRecordFindQuery("container", "two"), backups.WithRecordFindMaxItems(1))
	if err != nil || value != nil || calls.Len() != 2 || pythonBackupCalls(calls)[1].query != "limit=1&name=nightly" {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
	calls.Reset()
	value, err = api.FindRecord(context.Background(), "nightly", backups.WithRecordFindQuery("name", nil), backups.WithRecordFindQuery("container", "one"))
	if err != nil || value == nil || pythonBackupCalls(calls)[1].query != "" {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
}

func TestPythonBackupFindRecordPagingShortMarkersAndOffset(t *testing.T) {
	for _, offset := range []bool{false, true} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/nightly") {
					return pythonBackupWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "b1" || req.URL.Query().Get("offset") == "1" {
					return pythonBackupWire(200, `{"backups":[]}`)
				}
				if offset {
					return pythonBackupWire(200, `{"backups":[{"id":"b1","name":"nightly"}],"next":"?offset=1&limit=2"}`)
				}
				return pythonBackupWire(200, `{"backups":[{"id":"b1","name":"nightly"}]}`)
			})
			options := []backups.RecordFindOption{backups.WithRecordFindQuery("limit", 2)}
			if offset {
				options = append(options, backups.WithRecordFindQuery("offset", 0))
			}
			value, err := api.FindRecord(context.Background(), "nightly", options...)
			if err != nil || value == nil || calls.Len() != 3 {
				t.Fatal(value, err, pythonBackupCalls(calls))
			}
			want := "limit=2&marker=b1&name=nightly"
			if offset {
				want = "limit=2&name=nightly&offset=1"
			}
			if pythonBackupCalls(calls)[2].query != want {
				t.Fatal(pythonBackupCalls(calls))
			}
			calls.Reset()
			value, err = api.FindRecord(context.Background(), "nightly", append(options, backups.WithRecordFindPaginated(false))...)
			if err != nil || value == nil || calls.Len() != 2 {
				t.Fatal(value, err, pythonBackupCalls(calls))
			}
		})
	}
}

func TestPythonBackupFindRecordMarkerUsesRetainedUntypedWireID(t *testing.T) {
	api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/nightly") {
			return pythonBackupWire(404, `{}`)
		}
		if req.URL.Query().Get("marker") == "9007199254740993" {
			return pythonBackupWire(200, `{"backups":[]}`)
		}
		return pythonBackupWire(200, `{"backups":[{"id":9007199254740993,"name":"nightly"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "nightly", backups.WithRecordFindQuery("limit", 2))
	if err != nil || value == nil || calls.Len() != 3 || pythonBackupCalls(calls)[2].query != "limit=2&marker=9007199254740993&name=nightly" {
		t.Fatal(value, err, pythonBackupCalls(calls))
	}
	backupFindField(t, value, "id", "9007199254740993")
	if string(value.Wire.Body["id"]) != "9007199254740993" {
		t.Fatal("marker conversion mutated untyped ID", value)
	}
}

func TestPythonBackupFindRecordRejectedAndAcceptedPhysicalCauses(t *testing.T) {
	for _, phase := range []string{"member404", "member500", "list500", "accepted list"} {
		t.Run(phase, func(t *testing.T) {
			readCause, closeCause := errors.New("backup read"), errors.New("backup close")
			var body *testcloud.Body
			api, calls := backupFindAPI(t, func(req *http.Request) *http.Response {
				if (phase == "list500" || phase == "accepted list") && strings.HasSuffix(req.URL.Path, "/nightly") {
					return pythonBackupWire(404, `{}`)
				}
				status, raw := 500, `{"error":"actual rejected"}`
				if phase == "member404" {
					status = 404
				}
				if phase == "accepted list" {
					status = 200
					raw = `{"backups":[{"id":"b1","name":"nightly"}]}`
				}
				body = &testcloud.Body{Data: raw, ReadErr: readCause, CloseErr: closeCause}
				response := pythonBackupWire(status, "")
				response.Header.Set("X-Request-Id", "backup-find-proof")
				response.Body = body
				return response
			})
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				return original
			}
			value, err := api.FindRecord(context.Background(), "nightly")
			if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || body == nil || body.Closes() != 1 {
				t.Fatal(value, err, body)
			}
			want := 1
			if phase == "list500" || phase == "accepted list" {
				want = 2
			}
			if calls.Len() != want {
				t.Fatal("faulty observation resent or allowed fallback", pythonBackupCalls(calls))
			}
			if phase == "accepted list" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "backup-find-proof" || string(proof.Body) != `{"backups":[{"id":"b1","name":"nightly"}]}` {
					t.Fatal(err, proof)
				}
			} else {
				var proof gophercloud.ErrUnexpectedResponseCode
				expected := 500
				if phase == "member404" {
					expected = 404
				}
				if !errors.As(err, &proof) || proof.Actual != expected || proof.ResponseHeader.Get("X-Request-Id") != "backup-find-proof" || string(proof.Body) != `{"error":"actual rejected"}` {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestPythonBackupFindRecordSelectionAndOptionFailures(t *testing.T) {
	api, calls := backupFindAPI(t, func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
	cases := []struct {
		identity string
		options  []backups.RecordFindOption
	}{
		{"..", nil}, {"nightly", []backups.RecordFindOption{nil}}, {"nightly", []backups.RecordFindOption{backups.WithRecordFindMaxItems(-1)}},
		{"nightly", []backups.RecordFindOption{backups.WithRecordFindQuery("id", "collision")}},
		{"nightly", []backups.RecordFindOption{backups.WithRecordFindOpts(backups.RecordFindOpts{Query: map[string]json.RawMessage{"metadata": json.RawMessage(`{bad`)}})}},
	}
	for _, item := range cases {
		if _, err := api.FindRecord(context.Background(), item.identity, item.options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Len() != 0 {
		t.Fatal(pythonBackupCalls(calls))
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("user cancelled lookup")
	_, err := api.FindRecord(ctx, "nightly", func(*backups.RecordFindOpts) error { cancel(cause); return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Len() != 0 {
		t.Fatal(err, pythonBackupCalls(calls))
	}
	_, err = api.FindRecord(context.Background(), "nightly", func(*backups.RecordFindOpts) error { api.RawClient().Microversion = "3.43"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, pythonBackupCalls(calls))
	}
}

func TestPythonBackupFindRecordDiscoveryDefaultAndExplicitVersion(t *testing.T) {
	for _, item := range []struct{ selected, maximum, want string }{{"", "3.71", "3.64"}, {"", "3.43", "3.43"}, {"3.71", "3.43", "3.71"}} {
		t.Run(item.selected+item.maximum, func(t *testing.T) {
			cloud := testcloud.New(t)
			var paths, versions []string
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				paths = append(paths, req.URL.Path)
				versions = append(versions, req.Header.Get("X-OpenStack-Volume-API-Version"))
				if req.URL.Path == "/cinder/v3/" {
					return pythonBackupWire(200, `{"version":{"id":"v3.0","max_version":"`+item.maximum+`","min_version":"3.0"}}`), nil
				}
				return pythonBackupWire(200, `{"backup":{"id":"b1"}}`), nil
			}))
			client := cloud.Client("volumev3", "/cinder/v3/project/")
			client.Microversion = item.selected
			value, err := backups.New(client).FindRecord(context.Background(), "b1")
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

func TestPythonBackupFindRecordDiscoveryRejectionPreservesPhysicalCauses(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	readCause, closeCause := errors.New("discovery read"), errors.New("discovery close")
	body := &testcloud.Body{Data: `{"error":"discovery unavailable"}`, ReadErr: readCause, CloseErr: closeCause}
	cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
		calls++
		r := pythonBackupWire(500, "")
		r.Body = body
		r.Header.Set("X-Request-Id", "discovery-proof")
		return r, nil
	}))
	value, err := backups.New(cloud.Client("volumev3", "/cinder/v3/project/")).FindRecord(context.Background(), "b1")
	var proof gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.Actual != 500 || proof.ResponseHeader.Get("X-Request-Id") != "discovery-proof" || string(proof.Body) != `{"error":"discovery unavailable"}` || calls != 1 || body.Closes() != 1 {
		t.Fatal(value, err, proof, calls, body)
	}
}
