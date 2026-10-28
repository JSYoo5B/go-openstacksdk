package snapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func snapshotFindField(t *testing.T, v *snapshots.SnapshotRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, v, want)
	}
}
func TestPythonSnapshotFindRecordDirectGetAndOwnedDescriptors(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.80", func(*http.Request) *http.Response {
		return pythonSnapshotWire(200, `{"snapshot":{"name":false,"size":"١٢","force":"TrUe","is_forced":"FALSE","project_id":"first","os-extended-snapshot-attributes:project_id":"last","metadata":42,"created_at":"literal","unknown":9007199254740993}}`)
	})
	v, err := api.FindRecord(context.Background(), "selected", snapshots.WithRecordFindDetails(false), snapshots.WithRecordFindAllProjects(true))
	if err != nil || calls.Len() != 1 || pythonSnapshotCalls(calls)[0].method != http.MethodGet || pythonSnapshotCalls(calls)[0].path != pythonSnapshotBase+"/selected" || pythonSnapshotCalls(calls)[0].query != "" || pythonSnapshotCalls(calls)[0].body != "" || pythonSnapshotCalls(calls)[0].version != "volume 3.80" {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	for key, want := range map[string]string{"id": `"selected"`, "name": `false`, "size": `12`, "is_forced": `false`, "project_id": `"last"`, "metadata": `{}`, "created_at": `"literal"`, "location": `null`, "updated_at": `null`, "consumes_quota": `null`} {
		snapshotFindField(t, v, key, want)
	}
	if len(v.Resource.Body) != 16 || v.Wire.Body["id"] != nil || string(v.Wire.Body["metadata"]) != "42" || string(v.Wire.Body["unknown"]) != "9007199254740993" || len(v.Discovery) != 0 {
		t.Fatal(v)
	}
	v.Resource.Body["id"][1] = 'X'
	v.Header.Set("X-Changed", "yes")
	if strings.Contains(string(v.Envelope), "X") || v.Wire.Header.Get("X-Changed") != "" || v.Resource.Header.Get("X-Changed") != "" {
		t.Fatal("proof aliases model")
	}
}

func TestPythonSnapshotFindRecordDefaultFallbackRoutesAndAllProjects(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, detail := range []bool{false, true} {
			for _, all := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d_detail%v_all%v", code, detail, all), func(t *testing.T) {
					path := pythonSnapshotBase
					if detail {
						path += "/detail"
					}
					api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
						if req.URL.Path != path {
							return pythonSnapshotWire(code, `{"missing":true}`)
						}
						return pythonSnapshotWire(200, `{"snapshots":[{"id":"snap","name":"daily","size":true,"metadata":{"k":9007199254740993}}]}`)
					})
					opts := []snapshots.RecordFindOption{snapshots.WithRecordFindAllProjects(all)}
					if !detail {
						opts = append(opts, snapshots.WithRecordFindDetails(false))
					}
					v, err := api.FindRecord(context.Background(), "daily", opts...)
					if err != nil || v == nil || calls.Len() != 2 || pythonSnapshotCalls(calls)[0].path != pythonSnapshotBase+"/daily" || pythonSnapshotCalls(calls)[0].query != "" || pythonSnapshotCalls(calls)[1].path != path {
						t.Fatal(v, err, pythonSnapshotCalls(calls))
					}
					query, _ := url.ParseQuery(pythonSnapshotCalls(calls)[1].query)
					if query.Get("name") != "daily" || query.Has("all_tenants") != all || all && query.Get("all_tenants") != "True" {
						t.Fatal(query)
					}
					snapshotFindField(t, v, "size", `true`)
					if v.StatusCode != 200 || !strings.Contains(string(v.Envelope), `"snapshots"`) {
						t.Fatal(v)
					}
				})
			}
		}
	}
}

func TestPythonSnapshotFindRecordFallbackMatchesIDAndDetectsLateDuplicateOrFailure(t *testing.T) {
	for _, last := range []string{"success", "duplicate", "error"} {
		api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
			if req.URL.Path != pythonSnapshotBase+"/detail" {
				return pythonSnapshotWire(404, `{}`)
			}
			if req.URL.Query().Get("marker") == "next" {
				if last == "error" {
					return pythonSnapshotWire(500, `{"later":true}`)
				}
				if last == "duplicate" {
					return pythonSnapshotWire(200, `{"snapshots":[{"id":"another","name":"wanted"}]}`)
				}
				return pythonSnapshotWire(200, `{"snapshots":[{"id":"other","name":"other"}]}`)
			}
			return pythonSnapshotWire(200, `{"snapshots":[{"id":"wanted","name":"other"}],"snapshots_links":[{"rel":"next","href":"?marker=next"}]}`)
		})
		v, err := api.FindRecord(context.Background(), "wanted")
		if calls.Len() != 3 {
			t.Fatal(pythonSnapshotCalls(calls))
		}
		if last == "success" {
			if err != nil || v == nil {
				t.Fatal(v, err)
			}
			snapshotFindField(t, v, "id", `"wanted"`)
		} else {
			if v != nil || err == nil || last == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || last == "error" && !gophercloud.ResponseCodeIs(err, 500) {
				t.Fatal(v, err)
			}
		}
	}
}

func TestPythonSnapshotFindRecordMissingAndCleanNativeRetry(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
		if req.URL.Path == pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(200, `{"snapshots":[]}`)
		}
		return pythonSnapshotWire(404, `{}`)
	})
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return original
	}
	v, err := api.FindRecord(context.Background(), "missing")
	if err != nil || v != nil || calls.Len() != 2 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	_, err = api.FindRecord(context.Background(), "missing", snapshots.WithRecordFindIgnoreMissing(false))
	if !errors.Is(err, resource.ErrNotFound) || calls.Len() != 4 {
		t.Fatal(err, pythonSnapshotCalls(calls))
	}
	api, calls = pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return pythonSnapshotWire(401, `{"unauthorized":true}`) })
	if v, err = api.FindRecord(context.Background(), "daily"); v != nil || !gophercloud.ResponseCodeIs(err, 401) || calls.Len() != 1 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
}

func TestPythonSnapshotFindRecordQueryAliasesFiltersAndRawRowCap(t *testing.T) {
	fields := map[string]json.RawMessage{"all_projects": json.RawMessage(`true`), "all_tenants": json.RawMessage(`false`), "volume_id": json.RawMessage(`["a",null,"b"]`), "metadata": json.RawMessage(`{"enabled":true}`), "unknown": json.RawMessage(`{bad`)}
	opts := snapshots.WithRecordFindOpts(snapshots.RecordFindOpts{Query: fields})
	fields["volume_id"][2] = 'X'
	api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
		if req.URL.Path != pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(404, `{}`)
		}
		return pythonSnapshotWire(200, `{"snapshots":[{"id":"a","name":"daily","metadata":{"enabled":false}},{"id":"b","name":"daily","metadata":{"enabled":1,"extra":false}},{"id":"c","name":"daily","metadata":{"enabled":true}}],"next":"https://bad.example/snapshots"}`)
	})
	v, err := api.FindRecord(context.Background(), "daily", opts, snapshots.WithRecordFindMaxItems(2), snapshots.WithRecordFindLimit(0))
	if err != nil || v == nil || calls.Len() != 2 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	snapshotFindField(t, v, "id", `"b"`)
	q, _ := url.ParseQuery(pythonSnapshotCalls(calls)[1].query)
	if q.Get("all_tenants") != "True" || q.Get("limit") != "2" || q.Get("name") != "daily" || !reflect.DeepEqual(q["volume_id"], []string{"a", "b"}) || q.Has("metadata") || q.Has("unknown") || pythonSnapshotCalls(calls)[0].query != "" {
		t.Fatal(q, pythonSnapshotCalls(calls))
	}
	calls.Reset()
	v, err = api.FindRecord(context.Background(), "daily", opts, snapshots.WithRecordFindAllProjects(false), snapshots.WithRecordFindMaxItems(2))
	if err != nil || v == nil {
		t.Fatal(v, err)
	}
	q, _ = url.ParseQuery(pythonSnapshotCalls(calls)[1].query)
	if q.Has("all_tenants") {
		t.Fatal(q)
	}
}

func TestPythonSnapshotFindRecordFilterDictionaryAndExplicitNullName(t *testing.T) {
	for _, item := range []struct {
		description, filter string
		valid, match        bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"k":true}`, false, false}, {`""`, `{}`, true, false}, {`null`, `null`, true, true}, {`1`, `true`, true, true}} {
		api, _ := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
			if req.URL.Path != pythonSnapshotBase+"/detail" {
				return pythonSnapshotWire(404, `{}`)
			}
			return pythonSnapshotWire(200, `{"snapshots":[{"id":"snap","name":"daily","description":`+item.description+`}]}`)
		})
		v, err := api.FindRecord(context.Background(), "daily", snapshots.WithRecordFindOpts(snapshots.RecordFindOpts{Query: map[string]json.RawMessage{"description": json.RawMessage(item.filter)}}))
		if (err == nil) != item.valid || (v != nil) != item.match {
			t.Fatal(item, v, err)
		}
	}
	api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
		if req.URL.Path != pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(404, `{}`)
		}
		return pythonSnapshotWire(200, `{"snapshots":[{"id":"daily","name":"other"}]}`)
	})
	v, err := api.FindRecord(context.Background(), "daily", snapshots.WithRecordFindQuery("name", nil))
	q, _ := url.ParseQuery(pythonSnapshotCalls(calls)[1].query)
	if err != nil || v == nil || q.Has("name") {
		t.Fatal(v, err, q)
	}
}

func TestPythonSnapshotFindRecordSinglePageMarkerFallbackAndOffsetPaging(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
		if req.URL.Path != pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(404, `{}`)
		}
		if req.URL.Query().Get("marker") == "a" {
			return pythonSnapshotWire(200, `{"snapshots":[]}`)
		}
		return pythonSnapshotWire(200, `{"snapshots":{"id":"a","name":"daily"}}`)
	})
	v, err := api.FindRecord(context.Background(), "daily", snapshots.WithRecordFindLimit(5))
	if err != nil || v == nil || calls.Len() != 3 || !strings.Contains(pythonSnapshotCalls(calls)[2].query, "marker=a") {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	calls.Reset()
	v, err = api.FindRecord(context.Background(), "daily", snapshots.WithRecordFindLimit(5), snapshots.WithRecordFindPaginated(false))
	if err != nil || v == nil || calls.Len() != 2 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	api, calls = pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
		if req.URL.Path != pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(404, `{}`)
		}
		if req.URL.Query().Get("offset") == "2" {
			return pythonSnapshotWire(200, `{"snapshots":[{"id":"b","name":"daily"}]}`)
		}
		return pythonSnapshotWire(200, `{"snapshots":[{"id":"a","name":"other"}],"next":"?offset=2&limit=2"}`)
	})
	v, err = api.FindRecord(context.Background(), "daily", snapshots.WithRecordFindQuery("offset", 0))
	if err != nil || v == nil || calls.Len() != 3 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	snapshotFindField(t, v, "id", `"b"`)
}

func TestPythonSnapshotFindRecordResponseShapeAndNullableIdentity(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
	}{{`{"snapshot":{"id":null,"size":false,"metadata":[]}}`, true}, {`{"snapshot":{"id":42,"force":null}}`, true}, {`{}`, true}, {"", true}, {"{bad", true}, {`[]`, false}, {`null`, false}, {`{"snapshot":null}`, false}, {`{"snapshot":{"force":1}}`, false}} {
		api, calls := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return pythonSnapshotWire(200, item.body) })
		v, err := api.FindRecord(context.Background(), "seed")
		if (err == nil) != item.valid || v == nil || v.StatusCode != 200 || string(v.Envelope) != item.body || calls.Len() != 1 {
			t.Fatal(item, v, err, pythonSnapshotCalls(calls))
		}
		if !item.valid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
				t.Fatal(err, proof)
			}
		}
	}
	for _, body := range []string{`{"snapshots":[null]}`, `{"snapshots":[{"id":"a","force":1}]}`, `{"snapshots":[{"id":"a","connection":null}]}`, `{"snapshots":null}`, `{"snapshots":[{"id":"a","name":"daily"}],"next":"?name=changed&marker=a"}`} {
		api, _ := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
			if req.URL.Path != pythonSnapshotBase+"/detail" {
				return pythonSnapshotWire(404, `{}`)
			}
			return pythonSnapshotWire(200, body)
		})
		v, err := api.FindRecord(context.Background(), "daily")
		var proof *resource.ResponseError
		if v != nil || err == nil || !errors.As(err, &proof) || string(proof.Body) != body {
			t.Fatal(v, err, proof)
		}
	}
}
func TestPythonSnapshotFindRecordRejectedPhysicalFailuresNeverFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause, closeCause := errors.New("member read"), errors.New("member close")
		closes := 0
		api, calls := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response {
			return testcloud.StreamResponse(code, &testcloud.Body{Data: `{"missing":true}`, ReadErr: readCause, CloseErr: closeCause, OnClose: func() { closes++ }}, http.Header{"X-Request-Id": {"rejected"}})
		})
		v, err := api.FindRecord(context.Background(), "daily")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || calls.Len() != 1 || closes != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"missing":true}` || native.ResponseHeader.Get("X-Request-Id") != "rejected" {
			t.Fatal(v, err, native, pythonSnapshotCalls(calls))
		}
	}
}
func TestPythonSnapshotFindRecordAcceptedFaultAndStickySourceProof(t *testing.T) {
	for _, sticky := range []bool{false, true} {
		readCause, closeCause := errors.New("accepted read"), errors.New("accepted close")
		api, _ := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return nil })
		calls := []pythonSnapshotCall{}
		api.RawClient().HTTPClient.Transport = testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
			calls = append(calls, pythonSnapshotCall{method: http.MethodGet})
			body := &testcloud.Body{Data: `{"snapshot":{"id":"a"}}`, ReadErr: readCause, CloseErr: closeCause}
			if sticky {
				body.ReadErr = nil
				body.CloseErr = nil
				body.OnRead = func() { api.RawClient().Microversion = "3.80" }
				body.OnClose = func() { api.RawClient().Microversion = "3.65" }
			}
			return testcloud.StreamResponse(200, body, http.Header{"X-Request-Id": {"accepted"}}), nil
		})
		v, err := api.FindRecord(context.Background(), "daily")
		var proof *resource.ResponseError
		if v == nil || err == nil || len(calls) != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || v.StatusCode != 200 || v.Resource != nil || v.Header.Get("X-Request-Id") != "accepted" {
			t.Fatal(v, err, proof, calls)
		}
		if sticky {
			if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Microversion != "3.65" {
				t.Fatal(err)
			}
		} else {
			if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
				t.Fatal(err)
			}
		}
	}
}

func TestPythonSnapshotFindRecordDiscoveryAndProtocolHeaderProtection(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "", func(req *http.Request) *http.Response {
		if req.URL.Path == "/cinder/v3/" {
			if req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" {
				t.Fatal(req.Header)
			}
			return pythonSnapshotWire(200, `{"version":{"id":"v3.0","version":"3.80","min_version":"3.0"}}`)
		}
		return pythonSnapshotWire(200, `{"snapshot":{"id":"snap"}}`)
	})
	api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
	v, err := api.FindRecord(context.Background(), "snap")
	if err != nil || v == nil || calls.Len() != 2 || pythonSnapshotCalls(calls)[0].path != "/cinder/v3/" || pythonSnapshotCalls(calls)[1].version != "volume 3.65" || api.RawClient().Microversion != "" || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 || v.StatusCode != 200 {
		t.Fatal(v, err, pythonSnapshotCalls(calls))
	}
	api, calls = pythonSnapshotAPI(t, "3.80", func(*http.Request) *http.Response { return pythonSnapshotWire(409, `{"conflict":true}`) })
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.65"
		return nil
	}
	_, err = api.FindRecord(context.Background(), "daily")
	if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 409) || calls.Len() != 1 {
		t.Fatal(err, pythonSnapshotCalls(calls))
	}
}

func TestPythonSnapshotFindRecordPreflightCancellationAndRetainedOptions(t *testing.T) {
	api, calls := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return pythonSnapshotWire(200, `{"snapshot":{"id":"a"}}`) })
	for _, opts := range [][]snapshots.RecordFindOption{{nil}, {snapshots.WithRecordFindLimit(-1)}, {snapshots.WithRecordFindMaxItems(-1)}, {snapshots.WithRecordFindQuery("id", "different")}, {snapshots.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "bad"})}, {snapshots.WithRecordFindOpts(snapshots.RecordFindOpts{Query: map[string]json.RawMessage{"metadata": json.RawMessage(`{bad`)}})}} {
		_, err := api.FindRecord(context.Background(), "daily", opts...)
		if err == nil || calls.Len() != 0 {
			t.Fatal(err, pythonSnapshotCalls(calls))
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("custom cancel"), errors.New("callback failure")
	_, err := api.FindRecord(ctx, "daily", func(*snapshots.RecordFindOpts) error { cancel(cause); return callback })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) || calls.Len() != 0 {
		t.Fatal(err, pythonSnapshotCalls(calls))
	}
	var retained *snapshots.RecordFindOpts
	_, err = api.FindRecord(context.Background(), "a", func(c *snapshots.RecordFindOpts) error {
		retained = c
		c.Headers = map[string]string{"X-Custom": "original"}
		return nil
	}, func(*snapshots.RecordFindOpts) error { retained.Headers["X-Auth-Token"] = "later mutation"; return nil })
	if err != nil || calls.Len() != 1 {
		t.Fatal(err, pythonSnapshotCalls(calls))
	}
}

func TestPythonSnapshotFindRecordConcurrentOptionReuse(t *testing.T) {
	api, _ := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return nil })
	api.RawClient().HTTPClient.Transport = testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Custom") != "owned" {
			t.Error(req.Header)
		}
		return pythonSnapshotWire(200, `{"snapshot":{"id":"a"}}`), nil
	})
	opts := snapshots.WithRecordFindHeaders(map[string]string{"X-Custom": "owned"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := api.FindRecord(context.Background(), "a", opts); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestPythonSnapshotFindRecordListPhysicalFailureAndDiscoveryOnlyProof(t *testing.T) {
	t.Run("list rejection", func(t *testing.T) {
		readCause, closeCause := errors.New("list read"), errors.New("list close")
		closes := 0
		api, calls := pythonSnapshotAPI(t, "3.65", func(req *http.Request) *http.Response {
			if req.URL.Path != pythonSnapshotBase+"/detail" {
				return pythonSnapshotWire(404, `{}`)
			}
			return testcloud.StreamResponse(403, &testcloud.Body{Data: `{"policy":false}`, ReadErr: readCause, CloseErr: closeCause, OnClose: func() { closes++ }}, http.Header{"X-Request-Id": {"list-denied"}})
		})
		v, err := api.FindRecord(context.Background(), "daily")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || calls.Len() != 2 || closes != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"policy":false}` || native.ResponseHeader.Get("X-Request-Id") != "list-denied" {
			t.Fatal(v, err, native, pythonSnapshotCalls(calls), closes)
		}
	})
	t.Run("accepted discovery fault", func(t *testing.T) {
		readCause, closeCause := errors.New("discovery read"), errors.New("discovery close")
		closes := 0
		body := `{"version":{"id":"v3.0","version":"3.80","min_version":"3.0"}}`
		api, calls := pythonSnapshotAPI(t, "", func(*http.Request) *http.Response {
			return testcloud.StreamResponse(200, &testcloud.Body{Data: body, ReadErr: readCause, CloseErr: closeCause, OnClose: func() { closes++ }}, http.Header{"X-Request-Id": {"discovery"}})
		})
		api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
		v, err := api.FindRecord(context.Background(), "daily")
		var proof *resource.ResponseError
		if v == nil || err == nil || calls.Len() != 1 || closes != 1 || v.StatusCode != 0 || v.Envelope != nil || v.Resource != nil || v.Wire != nil || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 || string(v.Discovery[0].Envelope) != body || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 200 {
			t.Fatal(v, err, proof, pythonSnapshotCalls(calls), closes)
		}
		v.Discovery[0].Envelope[0] = 'X'
		v.Discovery[0].Header.Set("X-Request-Id", "changed")
		if string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "discovery" {
			t.Fatal("discovery proof aliases error", proof)
		}
	})
}
