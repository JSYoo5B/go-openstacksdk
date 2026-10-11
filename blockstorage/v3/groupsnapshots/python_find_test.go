package groupsnapshots_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonGroupSnapshotFindDefaultsCleanFallbackAndNoAutomaticNameFilter(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(fmt.Sprintf("%d/%t", code, details), func(t *testing.T) {
				api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
					if strings.HasSuffix(r.URL.Path, "/literal /%雪") {
						return snapshotWire(code, "member rejected")
					}
					return snapshotWire(203, `{"group_snapshots":[{"id":"unrelated","name":"unrelated"},{"id":"literal /%雪","name":"different"}]}`)
				})
				api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source"}
				options := []groupsnapshots.RecordFindOption{}
				if !details {
					options = append(options, groupsnapshots.WithRecordFindDetails(false))
				}
				value, err := api.FindRecord(context.Background(), "literal /%雪", options...)
				path := snapshotBase + "/detail"
				if !details {
					path = snapshotBase
				}
				if err != nil || value == nil || len(*calls) != 2 || (*calls)[0].escaped != snapshotBase+"/literal%20%2F%25%E9%9B%AA" || (*calls)[1].path != path || (*calls)[1].query != "" || (*calls)[0].headers.Get("Accept") != "application/source" || (*calls)[1].headers.Get("Accept") != "application/json" {
					t.Fatal(value, err, *calls)
				}
				snapshotField(t, value, "id", `"literal /%雪"`)
				if value.StatusCode != 203 || value.Wire == nil || value.Header.Get("X-Request-Id") != "actual-group-snapshot" {
					t.Fatal(value)
				}
			})
		}
	}
}
func TestPythonGroupSnapshotFindDirectSuccessUntypedIDAndSeed(t *testing.T) {
	for _, body := range []string{`{"group_snapshot":{"id":null,"name":false,"cls":false}}`, `{"id":0,"name":[null]}`, `{}`, `opaque`} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(299, body) })
		value, err := api.FindRecord(context.Background(), "selected", groupsnapshots.WithRecordFindQuery("description", map[string]any{"zero": 0, "null": nil, "enabled": false}))
		if err != nil || value == nil || value.Resource == nil || value.StatusCode != 299 || string(value.Envelope) != body || len(*calls) != 1 {
			t.Fatal(body, value, err, *calls)
		}
		snapshotField(t, value, "description", `{"enabled":false,"null":null,"zero":0}`)
		if strings.Contains(body, `"cls"`) && (value.Wire == nil || string(value.Wire.Body["cls"]) != "false" || value.Resource.Body["cls"] != nil) {
			t.Fatal(value)
		}
	}
}
func TestPythonGroupSnapshotFindCompletePagesDuplicateAndLateFailure(t *testing.T) {
	for _, mode := range []string{"unique", "duplicate", "late HTTP", "late row", "unsafe link"} {
		t.Run(mode, func(t *testing.T) {
			first := `{"group_snapshots":[{"id":"first","name":"data"}],"group_snapshots_links":[{"rel":"next","href":"?marker=first"}]}`
			if mode == "unsafe link" {
				first = `{"group_snapshots":[{"id":"first","name":"data"}],"next":"https://other.test/group_snapshots/detail?marker=first"}`
			}
			api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/data") {
					return snapshotWire(404, "missing")
				}
				if r.URL.Query().Get("marker") == "" {
					return snapshotWire(200, first)
				}
				switch mode {
				case "duplicate":
					return snapshotWire(200, `{"group_snapshots":[{"id":"second","name":"data"}]}`)
				case "late HTTP":
					return snapshotWire(503, "late native")
				case "late row":
					return snapshotWire(200, `{"group_snapshots":[{"id":"second","cls":null}]}`)
				default:
					return snapshotWire(200, `{"group_snapshots":[]}`)
				}
			})
			value, err := api.FindRecord(context.Background(), "data")
			if mode == "unique" {
				if err != nil || value == nil || len(*calls) != 3 {
					t.Fatal(value, err, *calls)
				}
				snapshotField(t, value, "id", `"first"`)
				if string(value.Envelope) != first {
					t.Fatal(value)
				}
				return
			}
			if err == nil {
				t.Fatal(mode, value, *calls)
			}
			switch mode {
			case "duplicate":
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) || len(*calls) != 3 {
					t.Fatal(value, err, *calls)
				}
			case "late HTTP":
				if value == nil || value.Resource != nil || value.StatusCode != 503 || string(value.Envelope) != "late native" || !gophercloud.ResponseCodeIs(err, 503) {
					t.Fatal(value, err, *calls)
				}
			case "late row":
				if value == nil || value.Resource != nil || value.StatusCode != 200 || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(string(value.Envelope), `"cls":null`) {
					t.Fatal(value, err, *calls)
				}
			case "unsafe link":
				if value == nil || value.Resource != nil || len(*calls) != 2 || !errors.Is(err, resource.ErrInvalidOption) || string(value.Envelope) != first {
					t.Fatal(value, err, *calls)
				}
			}
		})
	}
}
func TestPythonGroupSnapshotFindDefaultMissingStrictAndListRejection(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, strict := range []bool{false, true} {
			api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/absent") {
					return snapshotWire(code, "suppressed GET")
				}
				return snapshotWire(200, `{"group_snapshots":[]}`)
			})
			opts := []groupsnapshots.RecordFindOption{}
			if strict {
				opts = append(opts, groupsnapshots.WithRecordFindIgnoreMissing(false))
			}
			value, err := api.FindRecord(context.Background(), "absent", opts...)
			if value != nil || len(*calls) != 2 || (err != nil) != strict {
				t.Fatal(code, strict, value, err, *calls)
			}
			if strict && (!errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, code)) {
				t.Fatal(err)
			}
		}
	}
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/absent") {
			return snapshotWire(404, "GET")
		}
		return snapshotWire(404, "LIST")
	})
	value, err := api.FindRecord(context.Background(), "absent")
	if value == nil || value.Resource != nil || value.StatusCode != 404 || string(value.Envelope) != "LIST" || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
}
func TestPythonGroupSnapshotFindGetOnlyMicroversionAndIndependentDiscovery(t *testing.T) {
	for _, mode := range []string{"both discovery", "GET override", "GET absent", "service aliases"} {
		selected := ""
		if mode == "GET absent" {
			selected = "3.80"
		}
		if mode == "service aliases" {
			selected = "3.29"
		}
		api, calls := snapshotAPI(t, selected, func(r *http.Request) *http.Response {
			if r.URL.Path == "/cinder/v3/" {
				return snapshotWire(200, `{"version":{"id":"v3.0","status":"CURRENT","min_version":"3.0","version":"3.80"}}`)
			}
			if strings.HasSuffix(r.URL.Path, "/data") {
				return snapshotWire(404, "missing")
			}
			return snapshotWire(200, `{"group_snapshots":[{"id":"v","name":"data"}]}`)
		})
		opts := []groupsnapshots.RecordFindOption{}
		if mode == "GET override" || mode == "service aliases" {
			opts = append(opts, groupsnapshots.WithRecordFindMicroversion("3.80"))
		}
		if mode == "GET absent" {
			opts = append(opts, groupsnapshots.WithRecordFindMicroversion(""))
		}
		if mode == "service aliases" {
			api.RawClient().MoreHeaders = map[string]string{"openstack-api-version": "volume 3.29", "x-openstack-volume-api-version": "3.29"}
		}
		value, err := api.FindRecord(context.Background(), "data", opts...)
		if err != nil || value == nil || api.RawClient().Microversion != selected {
			t.Fatal(mode, value, err, *calls)
		}
		getVersion, listVersion := "3.29", "3.29"
		discoveries := 2
		if mode == "GET override" {
			getVersion = "3.80"
			discoveries = 1
		}
		if mode == "GET absent" {
			getVersion = ""
			listVersion = "3.80"
			discoveries = 0
		}
		if mode == "service aliases" {
			getVersion = "3.80"
			discoveries = 0
		}
		if len(value.Discovery) != discoveries || len(*calls) != discoveries+2 {
			t.Fatal(mode, value, err, *calls)
		}
		for _, call := range *calls {
			if call.path == "/cinder/v3/" {
				continue
			}
			want := listVersion
			if strings.HasSuffix(call.path, "/data") {
				want = getVersion
			}
			generic := ""
			if want != "" {
				generic = "volume " + want
			}
			if call.headers.Get("OpenStack-API-Version") != generic || call.headers.Get("X-OpenStack-Volume-API-Version") != want {
				t.Fatal(mode, call, want)
			}
		}
	}
}
func TestPythonGroupSnapshotFindQuerySeedFieldsScopeAndOwnedSnapshots(t *testing.T) {
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","name"]`), "name": json.RawMessage(`"data"`), "description": json.RawMessage(`{"enabled":false}`), "all_projects": json.RawMessage(`false`), "sort": json.RawMessage(`"name:asc"`), "unknown": json.RawMessage(`null`)}
	headers := map[string]string{"Accept": "application/operation", "X-Owned": "before"}
	option := groupsnapshots.WithRecordFindOpts(groupsnapshots.RecordFindOpts{Query: query, Headers: headers})
	query["name"] = json.RawMessage(`"after"`)
	headers["X-Owned"] = "after"
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return snapshotWire(403, "fallback")
		}
		return snapshotWire(200, `{"group_snapshots":[{"id":"first","name":"data","description":{"enabled":false,"extra":0}},{"id":"second","name":"other"}]}`)
	})
	var retained *groupsnapshots.RecordFindOpts
	value, err := api.FindRecord(context.Background(), "data", option, func(c *groupsnapshots.RecordFindOpts) error { c.Headers["X-Next"] = "before"; retained = c; return nil }, func(*groupsnapshots.RecordFindOpts) error {
		retained.Headers["X-Next"] = "after"
		retained.Query["name"] = json.RawMessage(`"changed"`)
		return nil
	})
	if err != nil || value == nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	get, _ := url.ParseQuery((*calls)[0].query)
	list, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(get["fields"], []string{"id", "name"}) || get.Get("name") != "data" || get.Has("all_projects") || get.Has("unknown") || !reflect.DeepEqual(get["description"], []string{"enabled"}) || list.Get("all_tenants") != "False" || list.Get("sort") != "name:asc" || list.Has("fields") || list.Has("name") || list.Has("description") {
		t.Fatal(get, list, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("X-Owned") != "before" || call.headers.Get("X-Next") != "before" || call.headers.Get("Accept") != "application/operation" {
			t.Fatal(call)
		}
	}
	snapshotField(t, value, "id", `"first"`)
	api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, "") })
	value, err = api.FindRecord(context.Background(), "id", groupsnapshots.WithRecordFindQuery("name", false), groupsnapshots.WithRecordFindQuery("description", nil))
	if err != nil || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
	snapshotField(t, value, "name", "false")
	snapshotField(t, value, "description", "null")
}
func TestPythonGroupSnapshotFindReservedAndMalformedInputPrecedesHTTP(t *testing.T) {
	for _, key := range []string{"id", "cls", "self", "connection", "_synchronized", "base_path", "microversion", "paginated", "max_items"} {
		for _, input := range []any{nil, false, 0, snapshotMarshal(func() ([]byte, error) { t.Fatal("reserved input marshaled"); return nil, nil })} {
			api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return snapshotWire(200, `{}`) })
			value, err := api.FindRecord(context.Background(), "data", groupsnapshots.WithRecordFindQuery(key, input))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(key, value, err, *calls)
			}
		}
	}
	for _, cfg := range []groupsnapshots.RecordFindOpts{{MaxItems: -1}, {Query: map[string]json.RawMessage{"name": json.RawMessage(`{"broken"`)}}, {Query: map[string]json.RawMessage{"all_projects": json.RawMessage(`0`)}}, {Query: map[string]json.RawMessage{"limit": json.RawMessage(`[1,2]`)}}, {Query: map[string]json.RawMessage{"cls": json.RawMessage(`null`)}}} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return snapshotWire(200, `{}`) })
		value, err := api.FindRecord(context.Background(), "data", groupsnapshots.WithRecordFindOpts(cfg))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(cfg, value, err, *calls)
		}
	}
	for _, identity := range []string{"", " ", ".", "..", "bad\n", string([]byte{0xff})} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { t.Fatal("invalid identity HTTP"); return snapshotWire(200, `{}`) })
		value, err := api.FindRecord(context.Background(), identity)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(identity, value, err, *calls)
		}
	}
	var absent *groupsnapshots.API
	if v, err := absent.FindRecord(context.Background(), "data"); v != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(v, err)
	}
}
func TestPythonGroupSnapshotFindCapturedSourceOptionCancellationAndNil(t *testing.T) {
	for _, mode := range []string{"source", "cancel", "nil", "nil context"} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { t.Fatal("callback reached HTTP"); return snapshotWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("option cause")
		opts := []groupsnapshots.RecordFindOption{}
		switch mode {
		case "source":
			opts = append(opts, func(*groupsnapshots.RecordFindOpts) error { api.RawClient().Microversion = "3.80"; return cause })
		case "cancel":
			opts = append(opts, func(*groupsnapshots.RecordFindOpts) error { cancel(cause); return cause })
		case "nil":
			opts = append(opts, nil)
		case "nil context":
			ctx = nil
		}
		value, err := api.FindRecord(ctx, "data", opts...)
		if value != nil || err == nil || len(*calls) != 0 {
			t.Fatal(mode, value, err, *calls)
		}
		if mode == "source" && (!errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption)) {
			t.Fatal(err)
		}
		if mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
			t.Fatal(err)
		}
		cancel(nil)
	}
}
func TestPythonGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore(t *testing.T) {
	for _, code := range []int{203, 400, 403, 404, 503} {
		readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		var closes atomic.Int32
		callbacks := 0
		var client *gophercloud.ServiceClient
		original := ""
		body := `{"group_snapshot":{"id":"data"}}`
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			response := snapshotWire(code, body)
			response.Body = &snapshotFaultBody{data: []byte(body), read: readCause, close: closeCause, closes: &closes, onRead: func() { client.ResourceBase = original + "changed/" }, onClose: func() { client.ResourceBase = original; cancel(cancelCause) }}
			return response
		})
		client = api.RawClient()
		original = client.ResourceBase
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks++
			return nil
		}
		value, err := api.FindRecord(ctx, "data")
		if value == nil || value.Resource != nil || value.StatusCode != code || string(value.Envelope) != body || len(*calls) != 1 || closes.Load() != 1 || callbacks != 0 {
			t.Fatal(code, value, err, *calls, closes.Load(), callbacks)
		}
		for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, cause) {
				t.Fatal(code, cause, err)
			}
		}
		cancel(nil)
	}
}
func TestPythonGroupSnapshotFindLatePhysicalListFailureKeepsActualPage(t *testing.T) {
	readCause, closeCause := errors.New("late read"), errors.New("late close")
	var closes atomic.Int32
	callbacks := 0
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return snapshotWire(404, "GET")
		}
		if r.URL.Query().Get("marker") == "" {
			return snapshotWire(200, `{"group_snapshots":[{"id":"first","name":"data"}],"next":"?marker=first"}`)
		}
		rsp := snapshotWire(404, "partial LIST")
		rsp.Body = &snapshotFaultBody{data: []byte("partial LIST"), read: readCause, close: closeCause, closes: &closes}
		return rsp
	})
	// The initial clean404 could invoke retry policy; leave clean4xx terminal
	// at the native retry hook while still checking late faults before callbacks.
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
		if gophercloud.ResponseCodeIs(err, 404) && len(*calls) == 1 {
			return err
		}
		callbacks++
		return err
	}
	value, err := api.FindRecord(context.Background(), "data")
	if value == nil || value.Resource != nil || value.StatusCode != 404 || string(value.Envelope) != "partial LIST" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || closes.Load() != 1 || len(*calls) != 3 || callbacks != 0 {
		t.Fatal(value, err, *calls, callbacks, closes.Load())
	}
}
func TestPythonGroupSnapshotFindCleanRetryAndOwnedHeaders(t *testing.T) {
	for _, mode := range []string{"clean", "header mutation", "source mutation"} {
		requests := 0
		cause := errors.New("retry cause")
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			requests++
			if requests == 1 {
				return snapshotWire(503, "first rejection")
			}
			return snapshotWire(200, `{"group_snapshot":{"id":"data"}}`)
		})
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			switch mode {
			case "header mutation":
				opts.MoreHeaders["X-Owned"] = "changed"
			case "source mutation":
				api.RawClient().Microversion = "3.80"
				return cause
			}
			return nil
		}
		value, err := api.FindRecord(context.Background(), "data", groupsnapshots.WithRecordFindHeaders(map[string]string{"X-Owned": "original"}))
		if mode == "clean" {
			if err != nil || value == nil || value.Resource == nil || len(*calls) != 2 {
				t.Fatal(value, err, *calls)
			}
			for _, call := range *calls {
				if call.headers.Get("X-Owned") != "original" || call.headers.Get("OpenStack-API-Version") != "volume 3.29" {
					t.Fatal(call)
				}
			}
		} else if value == nil || value.Resource != nil || value.StatusCode != 503 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || len(*calls) != 1 {
			t.Fatal(mode, value, err, *calls)
		}
		if mode == "source mutation" && !errors.Is(err, cause) {
			t.Fatal(err)
		}
	}
}
func TestPythonGroupSnapshotFindPagingControlsOffsetAndRawCap(t *testing.T) {
	for _, single := range []bool{false, true} {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			if strings.HasSuffix(r.URL.Path, "/data") {
				return snapshotWire(404, "GET")
			}
			if r.URL.Query().Get("offset") == "1" {
				return snapshotWire(200, `{"group_snapshots":[]}`)
			}
			return snapshotWire(200, `{"group_snapshots":[{"id":"first","name":"data"}],"next":"?offset=1"}`)
		})
		opts := []groupsnapshots.RecordFindOption{groupsnapshots.WithRecordFindQuery("sort", "name:asc")}
		if single {
			opts = append(opts, groupsnapshots.WithRecordFindPaginated(false))
		}
		value, err := api.FindRecord(context.Background(), "data", opts...)
		count := 3
		if single {
			count = 2
		}
		if err != nil || value == nil || len(*calls) != count {
			t.Fatal(single, value, err, *calls)
		}
		if !single {
			q, _ := url.ParseQuery((*calls)[2].query)
			if q.Get("offset") != "1" || q.Get("sort") != "name:asc" || q.Has("marker") {
				t.Fatal(q, *calls)
			}
		}
	}
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return snapshotWire(404, "GET")
		}
		return snapshotWire(200, `{"group_snapshots":[{"id":"first","name":"other"},{"id":"second","name":"data"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "data", groupsnapshots.WithRecordFindMaxItems(1))
	if value != nil || err != nil || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "limit=1") {
		t.Fatal(value, err, *calls)
	}
}
func TestPythonGroupSnapshotFindDeclaredLocationAndEvidenceDoNotAlias(t *testing.T) {
	name := "cloud"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	option := groupsnapshots.WithRecordFindLocation(location)
	*location.Cloud = "changed"
	body := `{"group_snapshot":{"id":"server","project_id":"other","unknown":9007199254740993}}`
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
	value, err := api.FindRecord(context.Background(), "input", option)
	if err != nil || value == nil || len(*calls) != 1 || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(value, err, *calls)
	}
	var observed resource.CloudLocation
	if err := json.Unmarshal(value.Resource.Body["location"], &observed); err != nil || observed.Cloud == nil || *observed.Cloud != "cloud" || string(observed.Project.ID) != `"other"` {
		t.Fatal(observed, err)
	}
	value.Resource.Body["id"][1] = 'x'
	value.Header.Set("X-Request-Id", "changed")
	value.Envelope[0] = '['
	if string(value.Wire.Body["id"]) != `"server"` || value.Wire.Header.Get("X-Request-Id") != "actual-group-snapshot" {
		t.Fatal(value)
	}
}
