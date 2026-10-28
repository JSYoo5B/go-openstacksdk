package snapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordMetadataFixture(t *testing.T, reply func(*http.Request) *http.Response) (*snapshots.API, *testcloud.Recorder) {
	return pythonSnapshotAPI(t, "", reply)
}
func recordMetadataSeed(metadataJSON string) *snapshots.SnapshotRecord {
	return &snapshots.SnapshotRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
		"id": json.RawMessage(`"seed-id"`), "name": json.RawMessage(`"cached-name"`), "size": json.RawMessage(`"۱۲"`), "metadata": json.RawMessage(metadataJSON),
	}}}}
}
func recordMetadataField(t *testing.T, value *snapshots.SnapshotRecord, key, expected string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != expected {
		t.Fatalf("value=%+v key=%s want=%s", value, key, expected)
	}
}
func recordMetadataProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body {
		t.Fatalf("err=%v proof=%+v", err, proof)
	}
}

func TestPythonSnapshotMetadataRecordFetchAliasesSeedsAndRawValues(t *testing.T) {
	for _, tc := range []struct{ body, metadata string }{
		{`{"unknown":9007199254740993,"id":"wrong","name":"wrong"}`, `{"cached":false}`},
		{`{"metadata":null}`, `null`}, {`{"metadata":42}`, `{}`}, {`{"metadata":[]}`, `{}`},
		{`{"metadata":{"nested":[false,0,null],"large":9007199254740993},"extra":true}`, `{"nested":[false,0,null],"large":9007199254740993}`},
	} {
		t.Run(tc.body, func(t *testing.T) {
			api, calls := recordMetadataFixture(t, func(r *http.Request) *http.Response {
				if r.Header.Get("OpenStack-API-Version") != "" {
					t.Fatal("metadata started microversion negotiation", r.Header)
				}
				response := pythonSnapshotWire(203, tc.body)
				response.Header.Set("X-Receipt", "actual-metadata")
				return response
			})
			seed := recordMetadataSeed(`{"cached":false}`)
			option := snapshots.WithRecordMetadataSeed(seed)
			seed.Resource.Body["metadata"] = json.RawMessage(`{"mutated":true}`)
			for _, fetch := range []func(context.Context, string, ...snapshots.RecordMetadataOption) (*snapshots.SnapshotRecord, error){api.FetchMetadataRecord, api.GetMetadataRecord} {
				value, err := fetch(context.Background(), "literal/%雪", option)
				if err != nil || calls.Len() == 0 || value.StatusCode != 203 || value.Header.Get("X-Receipt") != "actual-metadata" || string(value.Envelope) != tc.body {
					t.Fatal(value, err, pythonSnapshotCalls(calls))
				}
				recordMetadataField(t, value, "metadata", tc.metadata)
				recordMetadataField(t, value, "id", `"literal/%雪"`)
				recordMetadataField(t, value, "name", `"cached-name"`)
				recordMetadataField(t, value, "size", `12`)
				if value.Resource.Body["unknown"] != nil || value.Wire == nil {
					t.Fatal(value)
				}
				value.Resource.Body["metadata"][0] = 'X'
				if string(value.Envelope) != tc.body || value.Wire.Body["unknown"] != nil && string(value.Wire.Body["unknown"]) != "9007199254740993" {
					t.Fatal("metadata carriers alias", value)
				}
			}
			if calls.Len() != 2 || pythonSnapshotCalls(calls)[0].method != "GET" || pythonSnapshotCalls(calls)[0].query != "" || pythonSnapshotCalls(calls)[0].body != "" {
				t.Fatal(pythonSnapshotCalls(calls))
			}
		})
	}
}

func TestPythonSnapshotMetadataRecordSetCachesRequestDeltaAndOwnsInputs(t *testing.T) {
	input := map[string]any{"nested": map[string]any{"false": false, "zero": 0, "null": nil}, "large": json.Number("9007199254740993")}
	raw, _ := json.Marshal(input)
	headers := map[string]string{"X-Probe": "owned"}
	option := snapshots.WithRecordMetadataHeaders(headers)
	headers["X-Probe"] = "changed"
	seed := recordMetadataSeed(`{"old":"cached"}`)
	api, calls := recordMetadataFixture(t, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Probe") != "owned" || r.Header.Get("X-Owned") != "first" {
			t.Fatal(r.Header)
		}
		return pythonSnapshotWire(299, `{"metadata":{"old":"server","extra":true},"unknown":9007199254740993}`)
	})
	var retained *snapshots.RecordMetadataOpts
	value, err := api.SetMetadataRecord(context.Background(), "r1", input, snapshots.WithRecordMetadataSeed(seed), option,
		func(cfg *snapshots.RecordMetadataOpts) error {
			input["nested"].(map[string]any)["false"] = true
			cfg.Headers["X-Owned"] = "first"
			retained = cfg
			return nil
		},
		func(*snapshots.RecordMetadataOpts) error {
			retained.Headers["X-Owned"] = "changed"
			seed.Resource.Body["name"] = json.RawMessage(`"changed"`)
			return nil
		})
	expected, _ := json.Marshal(map[string]any{"metadata": json.RawMessage(raw)})
	if err != nil || calls.Len() != 1 || pythonSnapshotCalls(calls)[0].method != "POST" || pythonSnapshotCalls(calls)[0].path != pythonSnapshotBase+"/r1/metadata" || pythonSnapshotCalls(calls)[0].body != string(expected) {
		t.Fatal(value, err, pythonSnapshotCalls(calls))
	}
	recordMetadataField(t, value, "metadata", string(raw))
	recordMetadataField(t, value, "name", `"cached-name"`)
	if value.Wire == nil || string(value.Wire.Body["metadata"]) != `{"old":"server","extra":true}` || string(value.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(value)
	}
	for _, body := range []string{"", "not JSON", `[]`, `null`, `false`} {
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(204, body) })
		value, err := api.SetMetadataRecord(context.Background(), "r1", nil)
		if err != nil || calls.Len() != 1 || pythonSnapshotCalls(calls)[0].body != `{"metadata":{}}` || string(value.Envelope) != body {
			t.Fatal(value, err, pythonSnapshotCalls(calls))
		}
		recordMetadataField(t, value, "metadata", `{}`)
	}
}

func TestPythonSnapshotMetadataRecordDeleteNilEmptyOrderedAndCachedPop(t *testing.T) {
	api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(204, "opaque ACK") })
	cleared, err := api.DeleteMetadataRecord(context.Background(), "r1", nil, snapshots.WithRecordMetadataSeed(recordMetadataSeed(`{"old":false}`)))
	if err != nil || !cleared.Cleared || len(cleared.Responses) != 1 || cleared.Responses[0].Method != "PUT" || calls.Len() != 1 || pythonSnapshotCalls(calls)[0].body != `{"metadata":{}}` {
		t.Fatal(cleared, err, pythonSnapshotCalls(calls))
	}
	recordMetadataField(t, cleared.Record, "metadata", `{}`)
	empty, err := api.DeleteMetadataRecord(context.Background(), "r1", []string{}, snapshots.WithRecordMetadataSeed(recordMetadataSeed(`{"old":false}`)))
	if err != nil || empty.Cleared || len(empty.Responses) != 0 || empty.Record.StatusCode != 0 || calls.Len() != 1 {
		t.Fatal(empty, err, pythonSnapshotCalls(calls))
	}
	recordMetadataField(t, empty.Record, "metadata", `{"old":false}`)
	ordered, err := api.DeleteMetadataRecord(context.Background(), "r1", []string{"literal /%雪", "literal /%雪"})
	if err != nil || len(ordered.Responses) != 2 || !reflect.DeepEqual(ordered.DeletedKeys, []string{"literal /%雪", "literal /%雪"}) || calls.Len() != 3 {
		t.Fatal(ordered, err, pythonSnapshotCalls(calls))
	}
	if pythonSnapshotCalls(calls)[1].method != "DELETE" || pythonSnapshotCalls(calls)[1].body != "" || *ordered.Responses[0].Key != "literal /%雪" {
		t.Fatal(ordered, pythonSnapshotCalls(calls))
	}
	seed := recordMetadataSeed(`{"first":false,"second":0}`)
	failure, err := api.DeleteMetadataRecord(context.Background(), "r1", []string{"first", "missing", "second"}, snapshots.WithRecordMetadataSeed(seed))
	var cache *snapshots.MetadataRecordCacheKeyError
	if !errors.As(err, &cache) || cache.Key != "missing" || len(failure.Responses) != 2 || !reflect.DeepEqual(failure.DeletedKeys, []string{"first"}) || calls.Len() != 5 {
		t.Fatal(failure, err, pythonSnapshotCalls(calls))
	}
	recordMetadataProof(t, err, 204, "opaque ACK")
	recordMetadataField(t, failure.Record, "metadata", `{"second":0}`)
	if string(seed.Resource.Body["metadata"]) != `{"first":false,"second":0}` {
		t.Fatal("seed cache mutated", seed)
	}
}

func TestPythonSnapshotMetadataRecordHTTPPoliciesAndPartialDeletion(t *testing.T) {
	for _, status := range []int{199, 200, 201, 204, 299, 302, 399, 400, 403, 404, 409, 429, 503} {
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(status, `{"metadata":null}`) })
		value, err := api.FetchMetadataRecord(context.Background(), "r1")
		if calls.Len() != 1 || (err != nil) != (status >= 400) {
			t.Fatal(status, value, err, pythonSnapshotCalls(calls))
		}
		if status >= 400 && !gophercloud.ResponseCodeIs(err, status) {
			t.Fatal(status, err)
		}
	}
	for _, body := range []string{"", "{bad", `null`, `[]`, `42`} {
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(204, body) })
		value, err := api.FetchMetadataRecord(context.Background(), "r1")
		if value == nil || value.Resource != nil || err == nil || calls.Len() != 1 {
			t.Fatal(value, err, pythonSnapshotCalls(calls))
		}
		recordMetadataProof(t, err, 204, body)
	}
	count := 0
	api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response {
		count++
		if count == 2 {
			return pythonSnapshotWire(404, "missing key")
		}
		return pythonSnapshotWire(204, "")
	})
	value, err := api.DeleteMetadataRecord(context.Background(), "r1", []string{"first", "missing", "third"})
	if !gophercloud.ResponseCodeIs(err, 404) || calls.Len() != 2 || len(value.Responses) != 1 || !reflect.DeepEqual(value.DeletedKeys, []string{"first"}) {
		t.Fatal(value, err, pythonSnapshotCalls(calls))
	}
}

type metadataRecordMarshal func() ([]byte, error)

func (f metadataRecordMarshal) MarshalJSON() ([]byte, error) { return f() }
func TestPythonSnapshotMetadataRecordSourceCaptureCallbacksAndPreflight(t *testing.T) {
	for _, mode := range []string{"source marshal", "cancel marshal", "source option", "cancel option", "nil option", "auth header", "bad id"} {
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("callback cause")
		input := map[string]any{"k": false}
		options := []snapshots.RecordMetadataOption{}
		id := "r1"
		switch mode {
		case "source marshal":
			input["k"] = metadataRecordMarshal(func() ([]byte, error) { api.RawClient().ResourceBase = "https://changed.test/v3/"; return nil, cause })
		case "cancel marshal":
			input["k"] = metadataRecordMarshal(func() ([]byte, error) { cancel(cause); return nil, cause })
		case "source option":
			options = append(options, func(*snapshots.RecordMetadataOpts) error { api.RawClient().Microversion = "changed"; return cause })
		case "cancel option":
			options = append(options, func(*snapshots.RecordMetadataOpts) error { cancel(cause); return cause })
		case "nil option":
			options = append(options, nil)
		case "auth header":
			options = append(options, snapshots.WithRecordMetadataHeader("X-Auth-Token", "caller"))
		case "bad id":
			id = ".."
		}
		value, err := api.SetMetadataRecord(ctx, id, input, options...)
		if value != nil || err == nil || calls.Len() != 0 {
			t.Fatal(mode, value, err, pythonSnapshotCalls(calls))
		}
		if strings.Contains(mode, "source") && (!errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption)) {
			t.Fatal(err)
		}
		if strings.Contains(mode, "cancel") && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
			t.Fatal(err)
		}
		cancel(nil)
	}
	var nilAPI *snapshots.API
	if _, err := nilAPI.FetchMetadataRecord(context.Background(), "r1"); err == nil {
		t.Fatal("nil API accepted")
	}
}

func TestPythonSnapshotMetadataRecordPhysicalFaultsNeverReplayOrSuppress(t *testing.T) {
	for _, status := range []int{203, 404, 503} {
		for _, operation := range []string{"fetch", "set", "delete"} {
			t.Run(operation+string(rune(status)), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("Read cause"), errors.New("Close cause"), errors.New("cancel cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				var client *gophercloud.ServiceClient
				closes := atomic.Int32{}
				original := ""
				body := `{"metadata":{"k":false}}`
				api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response {
					response := pythonSnapshotWire(status, body)
					response.Header.Set("X-Receipt", "physical")
					response.Body = &testcloud.Body{Data: body, ReadErr: readCause, CloseErr: closeCause,
						OnRead: func() { client.ResourceBase = original + "changed/" }, OnClose: func() { closes.Add(1); client.ResourceBase = original; cancel(cancelCause) }}
					return response
				})
				client = api.RawClient()
				original = client.ResourceBase
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
				var err error
				switch operation {
				case "fetch":
					value, failure := api.FetchMetadataRecord(ctx, "r1")
					err = failure
					if status < 400 && (value == nil || value.Resource != nil || string(value.Envelope) != body) {
						t.Fatal(value, err)
					}
				case "set":
					value, failure := api.SetMetadataRecord(ctx, "r1", map[string]any{"k": false})
					err = failure
					if status < 400 && (value == nil || value.Resource != nil || string(value.Envelope) != body) {
						t.Fatal(value, err)
					}
				case "delete":
					value, failure := api.DeleteMetadataRecord(ctx, "r1", []string{"k"})
					err = failure
					if value == nil || len(value.DeletedKeys) != 0 {
						t.Fatal(value, err)
					}
				}
				if calls.Len() != 1 || closes.Load() != 1 {
					t.Fatal("physical fault replayed", pythonSnapshotCalls(calls), closes.Load(), err)
				}
				for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
					if !errors.Is(err, cause) {
						t.Fatal("missing cause", cause, err)
					}
				}
				if status < 400 {
					recordMetadataProof(t, err, status, body)
				} else if !gophercloud.ResponseCodeIs(err, status) {
					t.Fatal(err)
				}
				cancel(nil)
			})
		}
	}
}

func TestPythonSnapshotMetadataRecordRetrySnapshotAndNativeCauses(t *testing.T) {
	for _, mode := range []string{"input changed", "body changed", "denied"} {
		input := map[string]any{"k": false}
		count := 0
		cause := errors.New("retry callback cause")
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response {
			count++
			if count == 1 {
				return pythonSnapshotWire(503, "first rejected")
			}
			return pythonSnapshotWire(204, "")
		})
		api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			input["k"] = true
			if mode == "body changed" {
				opts.JSONBody = json.RawMessage(`{"metadata":{"k":true}}`)
			}
			if mode == "denied" {
				return cause
			}
			return nil
		}
		value, err := api.SetMetadataRecord(context.Background(), "r1", input)
		if mode == "input changed" {
			if err != nil || calls.Len() != 2 || pythonSnapshotCalls(calls)[0].body != `{"metadata":{"k":false}}` || pythonSnapshotCalls(calls)[1].body != pythonSnapshotCalls(calls)[0].body {
				t.Fatal(value, err, pythonSnapshotCalls(calls))
			}
			recordMetadataField(t, value, "metadata", `{"k":false}`)
		} else {
			if calls.Len() != 1 || !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(value, err, pythonSnapshotCalls(calls))
			}
			if mode == "body changed" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "denied" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		}
	}
	api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(204, "") })
	if _, err := api.DeleteMetadataRecord(context.Background(), "r1", []string{}, snapshots.WithRecordMetadataHeader("X-Auth-Token", "bad")); !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal("no-op bypassed header validation", err, pythonSnapshotCalls(calls))
	}
}

func TestPythonSnapshotMetadataRecordSelectedMicroversionRetryAndSeedPreflight(t *testing.T) {
	count := 0
	api, calls := recordMetadataFixture(t, func(r *http.Request) *http.Response {
		count++
		if r.Header.Get("OpenStack-API-Version") != "volume 3.15" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.15" {
			t.Fatal(r.Header)
		}
		if count == 1 {
			return pythonSnapshotWire(503, "try again")
		}
		return pythonSnapshotWire(200, `{"metadata":{"k":false}}`)
	})
	client := api.RawClient()
	client.Microversion = "3.15"
	client.MoreHeaders = map[string]string{"Openstack-Api-Version": "volume 3.15", "X-Openstack-Volume-Api-Version": "3.15"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
	value, err := api.FetchMetadataRecord(context.Background(), "r1")
	if err != nil || calls.Len() != 2 || client.Microversion != "3.15" || client.MoreHeaders["Openstack-Api-Version"] != "volume 3.15" {
		t.Fatal(value, err, pythonSnapshotCalls(calls), client.MoreHeaders)
	}
	recordMetadataField(t, value, "metadata", `{"k":false}`)
	for _, tc := range []struct{ key, raw string }{{"metadata", "{"}, {"size", `"²"`}, {"name", ""}} {
		seed := recordMetadataSeed(`{}`)
		seed.Resource.Body[tc.key] = json.RawMessage(tc.raw)
		api, calls := recordMetadataFixture(t, func(*http.Request) *http.Response { t.Fatal("seed reached HTTP"); return pythonSnapshotWire(200, `{}`) })
		value, err := api.FetchMetadataRecord(context.Background(), "r1", snapshots.WithRecordMetadataSeed(seed))
		var proof *resource.ResponseError
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Len() != 0 {
			t.Fatal(tc, value, err, pythonSnapshotCalls(calls))
		}
	}
	keys := []string{"original"}
	api, calls = recordMetadataFixture(t, func(*http.Request) *http.Response { return pythonSnapshotWire(204, "") })
	deleted, err := api.DeleteMetadataRecord(context.Background(), "r1", keys, func(*snapshots.RecordMetadataOpts) error { keys[0] = "mutated"; return nil })
	if err != nil || calls.Len() != 1 || len(deleted.Responses) != 1 || *deleted.Responses[0].Key != "original" || !strings.HasSuffix(pythonSnapshotCalls(calls)[0].path, "/original") {
		t.Fatal(deleted, err, pythonSnapshotCalls(calls))
	}
}
