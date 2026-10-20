package consistencygroupsnapshots_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonConsistencyGroupSnapshotFindDefaultsCleanFallbackAndNoAutomaticNameFilter(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(fmt.Sprintf("%d/%t", code, details), func(t *testing.T) {
				api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
					if strings.HasSuffix(r.URL.Path, "/literal /%雪") {
						return snapshotWire(code, "member rejected")
					}
					return snapshotWire(203, `{"cgsnapshots":[{"id":"unrelated","name":"unrelated"},{"id":"literal /%雪","name":"different"}]}`)
				})
				api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source"}
				options := []consistencygroupsnapshots.RecordFindOption{}
				if !details {
					options = append(options, consistencygroupsnapshots.WithRecordFindDetails(false))
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
				if value.StatusCode != 203 || value.Wire == nil || value.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
					t.Fatal(value)
				}
			})
		}
	}
}
func TestPythonConsistencyGroupSnapshotFindDirectSuccessUntypedIDAndSeed(t *testing.T) {
	for _, body := range []string{`{"cgsnapshot":{"id":null,"name":false,"cls":false}}`, `{"id":0,"name":[null]}`, `{}`, `opaque`} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(299, body) })
		value, err := api.FindRecord(context.Background(), "selected", consistencygroupsnapshots.WithRecordFindQuery("description", map[string]any{"zero": 0, "null": nil, "enabled": false}))
		if err != nil || value == nil || value.Resource == nil || value.StatusCode != 299 || string(value.Envelope) != body || len(*calls) != 1 {
			t.Fatal(body, value, err, *calls)
		}
		snapshotField(t, value, "description", `{"enabled":false,"null":null,"zero":0}`)
		if strings.Contains(body, `"cls"`) && (value.Wire == nil || string(value.Wire.Body["cls"]) != "false" || value.Resource.Body["cls"] != nil) {
			t.Fatal(value)
		}
	}
}
func TestPythonConsistencyGroupSnapshotFindCompletePagesDuplicateAndLateFailure(t *testing.T) {
	for _, mode := range []string{"unique", "duplicate", "late HTTP", "late row", "unsafe link"} {
		t.Run(mode, func(t *testing.T) {
			first := `{"cgsnapshots":[{"id":"first","name":"data"}],"cgsnapshots_links":[{"rel":"next","href":"?marker=first"}]}`
			if mode == "unsafe link" {
				first = `{"cgsnapshots":[{"id":"first","name":"data"}],"next":"https://other.test/cgsnapshots/detail?marker=first"}`
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
					return snapshotWire(200, `{"cgsnapshots":[{"id":"second","name":"data"}]}`)
				case "late HTTP":
					return snapshotWire(503, "late native")
				case "late row":
					return snapshotWire(200, `{"cgsnapshots":[{"id":"second","cls":null}]}`)
				default:
					return snapshotWire(200, `{"cgsnapshots":[]}`)
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
func TestPythonConsistencyGroupSnapshotFindDefaultMissingStrictAndListRejection(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, strict := range []bool{false, true} {
			api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/absent") {
					return snapshotWire(code, "suppressed GET")
				}
				return snapshotWire(200, `{"cgsnapshots":[]}`)
			})
			opts := []consistencygroupsnapshots.RecordFindOption{}
			if strict {
				opts = append(opts, consistencygroupsnapshots.WithRecordFindIgnoreMissing(false))
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
func TestPythonConsistencyGroupSnapshotFindGetOnlyMicroversionWithoutDiscovery(t *testing.T) {
	for _, mode := range []string{"no default", "GET override", "GET absent", "service aliases"} {
		selected := ""
		if mode == "GET absent" {
			selected = "3.80"
		}
		if mode == "service aliases" {
			selected = "3.71"
		}
		api, calls := snapshotAPI(t, selected, func(r *http.Request) *http.Response {
			if r.URL.Path == snapshotBase+"/data" {
				return snapshotWire(404, "member missing")
			}
			if r.URL.Path != snapshotBase+"/detail" {
				t.Fatal("unexpected discovery/list route", r.URL)
			}
			return snapshotWire(200, `{"cgsnapshots":[{"id":"v","name":"data"}]}`)
		})
		options := []consistencygroupsnapshots.RecordFindOption{}
		if mode == "GET override" || mode == "service aliases" {
			options = append(options, consistencygroupsnapshots.WithRecordFindMicroversion("3.80"))
		}
		if mode == "GET absent" {
			options = append(options, consistencygroupsnapshots.WithRecordFindMicroversion(""))
		}
		if mode == "service aliases" {
			api.RawClient().MoreHeaders = map[string]string{"openstack-api-version": "volume 3.71", "x-openstack-volume-api-version": "3.71"}
		}
		value, err := api.FindRecord(context.Background(), "data", options...)
		if err != nil || value == nil || len(*calls) != 2 || api.RawClient().Microversion != selected {
			t.Fatal(mode, value, err, *calls)
		}
		getVersion := selected
		if mode == "GET override" || mode == "service aliases" {
			getVersion = "3.80"
		}
		if mode == "GET absent" {
			getVersion = ""
		}
		for index, call := range *calls {
			want := selected
			if index == 0 {
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
func TestPythonConsistencyGroupSnapshotFindQuerySeedFieldsScopeAndOwnedSnapshots(t *testing.T) {
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","name"]`), "name": json.RawMessage(`"data"`), "description": json.RawMessage(`{"enabled":false}`), "all_projects": json.RawMessage(`false`), "sort": json.RawMessage(`"name:asc"`), "unknown": json.RawMessage(`null`)}
	headers := map[string]string{"Accept": "application/operation", "X-Owned": "before"}
	option := consistencygroupsnapshots.WithRecordFindOpts(consistencygroupsnapshots.RecordFindOpts{Query: query, Headers: headers})
	query["name"] = json.RawMessage(`"after"`)
	headers["X-Owned"] = "after"
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return snapshotWire(403, "fallback")
		}
		return snapshotWire(200, `{"cgsnapshots":[{"id":"first","name":"data","description":{"enabled":false,"extra":0}},{"id":"second","name":"other"}]}`)
	})
	var retained *consistencygroupsnapshots.RecordFindOpts
	value, err := api.FindRecord(context.Background(), "data", option, func(c *consistencygroupsnapshots.RecordFindOpts) error {
		c.Headers["X-Next"] = "before"
		retained = c
		return nil
	}, func(*consistencygroupsnapshots.RecordFindOpts) error {
		retained.Headers["X-Next"] = "after"
		retained.Query["name"] = json.RawMessage(`"changed"`)
		return nil
	})
	if err != nil || value == nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	get, _ := url.ParseQuery((*calls)[0].query)
	list, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(get["fields"], []string{"id", "name"}) || get.Get("name") != "data" || get.Has("all_projects") || get.Has("unknown") || !reflect.DeepEqual(get["description"], []string{"enabled"}) || list.Has("all_tenants") || list.Has("all_projects") || list.Get("sort") != "name:asc" || list.Has("fields") || list.Has("name") || list.Has("description") {
		t.Fatal(get, list, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("X-Owned") != "before" || call.headers.Get("X-Next") != "before" || call.headers.Get("Accept") != "application/operation" {
			t.Fatal(call)
		}
	}
	snapshotField(t, value, "id", `"first"`)
	api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, "") })
	value, err = api.FindRecord(context.Background(), "id", consistencygroupsnapshots.WithRecordFindQuery("name", false), consistencygroupsnapshots.WithRecordFindQuery("description", nil))
	if err != nil || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
	snapshotField(t, value, "name", "false")
	snapshotField(t, value, "description", "null")
}
func TestPythonConsistencyGroupSnapshotFindReservedAndMalformedInputPrecedesHTTP(t *testing.T) {
	for _, key := range []string{"id", "cls", "self", "connection", "_synchronized", "base_path", "microversion", "paginated", "max_items"} {
		for _, input := range []any{nil, false, 0, snapshotMarshal(func() ([]byte, error) { t.Fatal("reserved input marshaled"); return nil, nil })} {
			api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return snapshotWire(200, `{}`) })
			value, err := api.FindRecord(context.Background(), "data", consistencygroupsnapshots.WithRecordFindQuery(key, input))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(key, value, err, *calls)
			}
		}
	}
	for _, cfg := range []consistencygroupsnapshots.RecordFindOpts{{MaxItems: -1}, {Query: map[string]json.RawMessage{"name": json.RawMessage(`{"broken"`)}}, {Query: map[string]json.RawMessage{"all_projects": json.RawMessage(`0`)}}, {Query: map[string]json.RawMessage{"limit": json.RawMessage(`[1,2]`)}}, {Query: map[string]json.RawMessage{"cls": json.RawMessage(`null`)}}} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return snapshotWire(200, `{}`) })
		value, err := api.FindRecord(context.Background(), "data", consistencygroupsnapshots.WithRecordFindOpts(cfg))
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
	var absent *consistencygroupsnapshots.API
	if v, err := absent.FindRecord(context.Background(), "data"); v != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(v, err)
	}
}
func TestPythonConsistencyGroupSnapshotFindCapturedSourceOptionCancellationAndNil(t *testing.T) {
	for _, mode := range []string{"source", "cancel", "nil", "nil context"} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { t.Fatal("callback reached HTTP"); return snapshotWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("option cause")
		opts := []consistencygroupsnapshots.RecordFindOption{}
		switch mode {
		case "source":
			opts = append(opts, func(*consistencygroupsnapshots.RecordFindOpts) error {
				api.RawClient().Microversion = "3.80"
				return cause
			})
		case "cancel":
			opts = append(opts, func(*consistencygroupsnapshots.RecordFindOpts) error { cancel(cause); return cause })
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
func TestPythonConsistencyGroupSnapshotFindPhysicalRejectionsCannotFallbackReplayOrIgnore(t *testing.T) {
	for _, code := range []int{203, 400, 403, 404, 503} {
		readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		var closes atomic.Int32
		callbacks := 0
		var client *gophercloud.ServiceClient
		original := ""
		body := `{"cgsnapshot":{"id":"data"}}`
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
func TestPythonConsistencyGroupSnapshotFindLatePhysicalListFailureKeepsActualPage(t *testing.T) {
	readCause, closeCause := errors.New("late read"), errors.New("late close")
	var closes atomic.Int32
	callbacks := 0
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/data") {
			return snapshotWire(404, "GET")
		}
		if r.URL.Query().Get("marker") == "" {
			return snapshotWire(200, `{"cgsnapshots":[{"id":"first","name":"data"}],"next":"?marker=first"}`)
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
func TestPythonConsistencyGroupSnapshotFindCleanRetryAndOwnedHeaders(t *testing.T) {
	for _, mode := range []string{"clean", "header mutation", "source mutation"} {
		requests := 0
		cause := errors.New("retry cause")
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			requests++
			if requests == 1 {
				return snapshotWire(503, "first rejection")
			}
			return snapshotWire(200, `{"cgsnapshot":{"id":"data"}}`)
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
		value, err := api.FindRecord(context.Background(), "data", consistencygroupsnapshots.WithRecordFindHeaders(map[string]string{"X-Owned": "original"}))
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
func TestPythonConsistencyGroupSnapshotFindPagingControlsOffsetAndRawCap(t *testing.T) {
	for _, single := range []bool{false, true} {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			if strings.HasSuffix(r.URL.Path, "/data") {
				return snapshotWire(404, "GET")
			}
			if r.URL.Query().Get("offset") == "1" {
				return snapshotWire(200, `{"cgsnapshots":[]}`)
			}
			return snapshotWire(200, `{"cgsnapshots":[{"id":"first","name":"data"}],"next":"?offset=1"}`)
		})
		opts := []consistencygroupsnapshots.RecordFindOption{consistencygroupsnapshots.WithRecordFindQuery("sort", "name:asc")}
		if single {
			opts = append(opts, consistencygroupsnapshots.WithRecordFindPaginated(false))
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
		return snapshotWire(200, `{"cgsnapshots":[{"id":"first","name":"other"},{"id":"second","name":"data"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "data", consistencygroupsnapshots.WithRecordFindMaxItems(1))
	if value != nil || err != nil || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "limit=1") {
		t.Fatal(value, err, *calls)
	}
}
func TestPythonConsistencyGroupSnapshotFindDeclaredLocationAndEvidenceDoNotAlias(t *testing.T) {
	name := "cloud"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	option := consistencygroupsnapshots.WithRecordFindLocation(location)
	*location.Cloud = "changed"
	body := `{"cgsnapshot":{"id":"server","project_id":"other","unknown":9007199254740993}}`
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
	value, err := api.FindRecord(context.Background(), "input", option)
	if err != nil || value == nil || len(*calls) != 1 || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(value, err, *calls)
	}
	var observed resource.CloudLocation
	if err := json.Unmarshal(value.Resource.Body["location"], &observed); err != nil || observed.Cloud == nil || *observed.Cloud != "cloud" || string(observed.Project.ID) != `"current"` {
		t.Fatal(observed, err)
	}
	value.Resource.Body["id"][1] = 'x'
	value.Header.Set("X-Request-Id", "changed")
	value.Envelope[0] = '['
	if string(value.Wire.Body["id"]) != `"server"` || value.Wire.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
		t.Fatal(value)
	}
}

func TestPythonConsistencyGroupSnapshotFindPreparationCannotBecomeMemberFallback(t *testing.T) {
	for _, mode := range []string{"version", "location", "headers"} {
		api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
			t.Fatal("preparation reached member/list", r.URL)
			return snapshotWire(200, `{}`)
		})
		options := []consistencygroupsnapshots.RecordFindOption{}
		switch mode {
		case "version":
			options = append(options, consistencygroupsnapshots.WithRecordFindMicroversion("invalid"))
		case "location":
			options = append(options, consistencygroupsnapshots.WithRecordFindLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`bad`)}}))
		case "headers":
			options = append(options, consistencygroupsnapshots.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "wrong"}))
		}
		value, err := api.FindRecord(context.Background(), "data", options...)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(mode, value, err, *calls)
		}
		var native gophercloud.ErrUnexpectedResponseCode
		var response *resource.ResponseError
		if errors.As(err, &native) || errors.As(err, &response) {
			t.Fatal("preparation invented HTTP proof", err)
		}
	}
}
