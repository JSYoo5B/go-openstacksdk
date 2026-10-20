package containers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordView(t *testing.T, record *containers.ContainerRecord, key, want string) {
	t.Helper()
	if record == nil || record.Resource == nil || string(record.Resource.Body[key]) != want {
		t.Fatalf("record=%+v key=%s want=%s", record, key, want)
	}
}
func recordOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "containers" {
		t.Fatalf("err=%v wrapper=%+v", err, wrapped)
	}
}

func TestPythonContainerRecordCreatesHeadersAndDeclaredProjection(t *testing.T) {
	calls := 0
	attrs := map[string]any{"count": false, "bytes": "۱۲", "is_newest": false, "is_content_type_detected": false, "read_ACL": []string{".r:*", "user:read"}, "READ_ACL": make(chan int), "OBJECT_COUNT": make(chan int), "x-container-write": []string{"user:write"}, "X-CoNtAiNeR-ByTeS-UsEd": "0005", "versions_location": nil, "content_type": "input/type", "id": "ignored", "unknown": make(chan int), "metadata": map[string]any{"Book": true, "content_type": nil, "X-Container-Meta-Exact": []any{false, 0, nil}, "x-container-meta-low": "double-prefix"}}
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		containerMetadataNoBody(t, r)
		if r.Method != "PUT" || r.Header.Get("X-Newest") != "False" || r.Header.Get("X-Detect-Content-Type") != "False" || r.Header.Get("X-Container-Read") != ".r:*, user:read" || r.Header.Get("X-Container-Write") != "user:write" || r.Header.Get("X-Container-Bytes-Used") != "0005" || r.Header.Get("X-Versions-Location") != "None" || r.Header.Get("Content-Type") != "input/type" || r.Header.Get("X-Container-Meta-Book") != "True" || r.Header.Get("X-Container-Meta-Content_type") != "None" || r.Header.Get("X-Container-Meta-Exact") != "[False, 0, None]" || r.Header.Get("X-Container-Meta-X-Container-Meta-Low") != "double-prefix" {
			t.Errorf("headers=%v", r.Header)
		}
		response := containerMetadataResponse(204, `{"name":"response-name","count":"7","bytes":null,"unknown":{"opaque":true}}`)
		response.Header.Set("X-Container-Object-Count", "9")
		response.Header.Set("X-Container-Bytes-Used", "12")
		response.Header.Set("X-Detect-Content-Type", "false")
		response.Header.Set("Content-Type", "response/type")
		response.Header.Set("X-Container-Meta-Observed", "physical")
		response.Header.Set("X-Container-Meta-Temp-URL-Key", "actual-secret")
		return response, nil
	})
	result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, attrs)
	if err != nil || result == nil || result.StatusCode != 204 || calls != 1 || result.Metadata["observed"] != "physical" || len(result.Metadata) != 1 || client.MoreHeaders != nil {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
	}
	for key, want := range map[string]string{"name": `"response-name"`, "id": `"response-name"`, "count": "7", "object_count": "9", "bytes": "null", "bytes_used": "12", "content_type": `"response/type"`, "is_content_type_detected": "true", "is_newest": "false", "versions_location": "null", "meta_temp_url_key": `"actual-secret"`, "location": "null"} {
		recordView(t, result, key, want)
	}
	if string(result.SeedBody["count"]) != "false" || string(result.SeedHeaders["bytes_used"]) != `"0005"` || string(result.Wire.Body["unknown"]) != `{"opaque":true}` || result.Resource.Body["unknown"] != nil {
		t.Fatal(result)
	}
	result.Resource.Body["name"][1] = 'X'
	if string(result.Wire.Body["name"]) != `"response-name"` || string(result.SeedBody["name"]) == string(result.Resource.Body["name"]) {
		t.Fatal("record carriers alias")
	}
}

func TestPythonContainerRecordSeedDescriptorsFailBeforeOptionsAndHTTP(t *testing.T) {
	for _, key := range []string{"count", "bytes", "object_count", "bytes_used", "x-container-object-count", "X-Container-Bytes-Used"} {
		t.Run(key, func(t *testing.T) {
			calls, options := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("invalid constructor descriptor sent HTTP")
				return nil, errors.New("unexpected HTTP")
			})
			result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName,
				map[string]any{key: "²"}, func(*containers.RecordCreateOpts) error { options++; return nil })
			var physical *resource.ResponseError
			if result != nil || err == nil || calls != 0 || options != 0 || errors.As(err, &physical) {
				t.Fatalf("result=%+v err=%v calls=%d options=%d physical=%+v", result, err, calls, options, physical)
			}
			recordOperation(t, err, "CreateRecord")
		})
	}
	// Seed conversion is validation only: the caller's raw header spelling is
	// sent and retained even though the result's descriptor getter is an int.
	calls := 0
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Container-Object-Count") != "0010" || r.Header.Get("X-Newest") != "false" {
			t.Fatal(r.Header)
		}
		return containerMetadataResponse(201, ""), nil
	})
	attrs := map[string]any{"count": "۱۲", "object_count": "0010", "is_newest": "false"}
	result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, attrs)
	if err != nil || calls != 1 || attrs["object_count"] != "0010" || string(result.SeedHeaders["object_count"]) != `"0010"` {
		t.Fatal(result, err, calls, attrs)
	}
	recordView(t, result, "count", "12")
	recordView(t, result, "object_count", "10")
	recordView(t, result, "is_newest", "true")
}

func TestPythonContainerRecordOpaqueSuccessRetainsSeedsAndAliasDefaults(t *testing.T) {
	for _, body := range []string{"", "opaque", `{invalid`} {
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return containerMetadataResponse(202, body), nil })
		result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, map[string]any{"count": true, "bytes": "-2", "history_location": "", "object_count": nil})
		if err != nil || result == nil || string(result.Envelope) != body {
			t.Fatal(result, err)
		}
		for key, want := range map[string]string{"id": `"` + containerMetadataName + `"`, "count": "true", "object_count": "null", "bytes": "0", "bytes_used": "0", "history_location": `""`, "timestamp": "null"} {
			recordView(t, result, key, want)
		}
	}
	client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
		return containerMetadataResponse(201, `{"id":null,"count":null}`), nil
	})
	result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, nil)
	if err != nil {
		t.Fatal(err)
	}
	recordView(t, result, "id", "null")
	recordView(t, result, "count", "null")
	recordView(t, result, "object_count", "null")
	for _, body := range []string{`null`, `false`, `[]`, `"string"`} {
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return containerMetadataResponse(203, body), nil })
		result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, nil)
		if result == nil || result.Resource != nil || err == nil {
			t.Fatal(result, err)
		}
		containerMetadataProof(t, err, 203, body)
	}
}

func TestPythonContainerRecordGenericStatusesAndCleanMissingPolicy(t *testing.T) {
	for _, status := range []int{199, 200, 201, 202, 204, 299, 300, 399, 400, 403, 404, 409, 500} {
		for _, create := range []bool{true, false} {
			calls, hooks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return containerMetadataResponse(status, "opaque"), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				hooks++
				return original
			}
			api := containers.New(client)
			var err error
			if create {
				_, err = api.CreateRecord(context.Background(), containerMetadataName, nil)
			} else {
				result, failure := api.DeleteRecord(context.Background(), containerMetadataName)
				err = failure
				if status == 404 && (err != nil || result == nil || !result.IgnoredMissing || result.StatusCode != 404 || string(result.Body) != "opaque" || hooks != 1) {
					t.Fatalf("result=%+v err=%v hooks=%d", result, err, hooks)
				}
			}
			wantError := status >= 400 && (create || status != 404)
			if (err != nil) != wantError || calls != 1 {
				t.Fatalf("create=%t status=%d err=%v calls=%d", create, status, err, calls)
			}
			if wantError {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status {
					t.Fatal(err, native)
				}
			}
		}
	}
	client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return containerMetadataResponse(404, "missing"), nil })
	result, err := containers.New(client).DeleteRecord(context.Background(), containerMetadataName, containers.WithRecordDeleteIgnoreMissing(false))
	if result != nil || err == nil {
		t.Fatal(result, err)
	}
	recordOperation(t, err, "DeleteRecord")
}

type containerRecordMarshal func() ([]byte, error)

func (f containerRecordMarshal) MarshalJSON() ([]byte, error) { return f() }

func TestPythonContainerRecordSourceCaptureSnapshotAndOptions(t *testing.T) {
	calls := 0
	attrs := map[string]any{"metadata": map[string]any{"Book": "owned"}, "read_ACL": []string{"first", "second"}}
	factoryHeaders := map[string]string{"X-Probe": "owned"}
	option := containers.WithRecordCreateHeaders(factoryHeaders)
	factoryHeaders["X-Probe"] = "changed"
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Probe") != "owned" || r.Header.Get("X-Container-Meta-Book") != "owned" || r.Header.Get("X-Container-Read") != "first, second" {
			t.Error(r.Header)
		}
		return containerMetadataResponse(201, ""), nil
	})
	_, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, attrs, option, func(cfg *containers.RecordCreateOpts) error {
		attrs["metadata"].(map[string]any)["Book"] = "changed"
		attrs["read_ACL"].([]string)[0] = "changed"
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"marshal source", "marshal cancel", "option source", "option cancel", "nil option", "bad header"} {
		calls = 0
		cause := errors.New("callback cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(201, ""), nil })
		input := map[string]any{}
		var options []containers.RecordCreateOption
		switch mode {
		case "marshal source":
			input["read_ACL"] = containerRecordMarshal(func() ([]byte, error) { client.Endpoint += "changed/"; return []byte(`"read"`), cause })
		case "marshal cancel":
			input["read_ACL"] = containerRecordMarshal(func() ([]byte, error) { cancel(cause); return nil, cause })
		case "option source":
			options = []containers.RecordCreateOption{func(*containers.RecordCreateOpts) error { client.Microversion = "changed"; return cause }}
		case "option cancel":
			options = []containers.RecordCreateOption{func(*containers.RecordCreateOpts) error { cancel(cause); return cause }}
		case "nil option":
			options = []containers.RecordCreateOption{nil}
		case "bad header":
			options = []containers.RecordCreateOption{containers.WithRecordCreateHeader("X-Auth-Token", "caller")}
		}
		_, err := containers.New(client).CreateRecord(ctx, containerMetadataName, input, options...)
		cancel(nil)
		if err == nil || calls != 0 {
			t.Fatal(mode, err, calls)
		}
		if strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
			t.Fatal(mode, err)
		}
		if strings.Contains(mode, "source") && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause)) {
			t.Fatal(mode, err)
		}
	}
	for _, attrs := range []map[string]any{{"name": "duplicate"}, {"read_ACL": []any{nil}}, {"read_ACL": "a", "X-Container-Read": "b"}, {"metadata": []any{}}, {"metadata": map[string]any{"Book": "a", "X-Container-Meta-book": "b"}}} {
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(201, ""), nil })
		before := calls
		if _, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, attrs); err == nil || calls != before {
			t.Fatal(attrs, err, calls)
		}
	}
}

func TestPythonContainerRecordPhysicalFaultsPreserveReceiptsAndDirty404(t *testing.T) {
	for _, create := range []bool{true, false} {
		for _, status := range []int{203, 404} {
			for _, mode := range []string{"read close", "cancel read", "sticky source"} {
				calls := 0
				readCause, closeCause, cancelCause := io.ErrUnexpectedEOF, errors.New("Close cause"), errors.New("cancel cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				var body *containerMetadataBody
				var client *gophercloud.ServiceClient
				original := containerMetadataBase
				client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
					calls++
					reader := strings.NewReader("physical proof")
					body = &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) {
						n, err := reader.Read(p)
						switch mode {
						case "read close":
							return n, readCause
						case "cancel read":
							cancel(cancelCause)
						case "sticky source":
							client.Endpoint = original + "changed/"
						}
						return n, err
					})}
					if mode == "read close" {
						body.closeErr = closeCause
					}
					if mode == "sticky source" {
						body.onClose = func() { client.Endpoint = original }
					}
					return containerMetadataWire(status, body), nil
				})
				api := containers.New(client)
				var err error
				if create {
					result, failure := api.CreateRecord(ctx, containerMetadataName, nil)
					err = failure
					if status < 400 && (result == nil || result.Resource != nil || result.StatusCode != status || string(result.Envelope) != "physical proof") {
						t.Fatal(result, err)
					}
				} else {
					result, failure := api.DeleteRecord(ctx, containerMetadataName)
					err = failure
					if result != nil && result.IgnoredMissing {
						t.Fatal("dirty404 ignored", result)
					}
				}
				if err == nil || calls != 1 || body.closes.Load() != 1 {
					t.Fatal(create, status, mode, err, calls, body.closes.Load())
				}
				if mode == "read close" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause)) {
					t.Fatal(err)
				}
				if mode == "cancel read" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(err)
				}
				if mode == "sticky source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if status < 400 {
					containerMetadataProof(t, err, status, "physical proof")
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "physical proof" {
						t.Fatal(err, native)
					}
				}
				cancel(nil)
			}
		}
	}
}

func TestPythonContainerRecordRetryRetainsTypedHeadersAndOriginalCauses(t *testing.T) {
	for _, mode := range []string{"clean", "header changed", "body added"} {
		calls, hooks := 0, 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Container-Read") != ".r:*" {
				t.Error(r.Header)
			}
			if calls == 1 {
				return containerMetadataResponse(503, "first rejected"), nil
			}
			return containerMetadataResponse(204, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks++
			if mode == "header changed" {
				opts.MoreHeaders["X-Container-Read"] = "changed"
			}
			if mode == "body added" {
				opts.JSONBody = map[string]any{"unexpected": true}
			}
			return nil
		}
		result, err := containers.New(client).CreateRecord(context.Background(), containerMetadataName, map[string]any{"read_ACL": ".r:*"})
		if mode == "clean" {
			if err != nil || result == nil || calls != 2 || hooks != 1 {
				t.Fatal(result, err, calls, hooks)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 || hooks != 1 {
				t.Fatal(result, err, calls, hooks)
			}
		}
	}
}

func TestPythonContainerRecordPublicFixtureAndIndependentDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []string
	cloud.Mux.HandleFunc("/swift/a", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.RequestURI())
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		if len(calls) == 1 && r.Header.Get("X-Container-Meta-Book") != "first" {
			t.Error(r.Header)
		}
		if len(calls) == 2 && r.Header.Get("X-Container-Meta-Book") != "" {
			t.Error("shared Source class metadata leaked", r.Header)
		}
		w.WriteHeader(204)
	})
	api := containers.New(cloud.Client("object-store", "/swift/"))
	first, err := api.CreateRecord(context.Background(), "a", map[string]any{"metadata": map[string]any{"Book": "first"}})
	if err != nil || first.Resource == nil {
		t.Fatal(first, err)
	}
	second, err := api.CreateRecord(context.Background(), "a", nil)
	if err != nil || second.Resource == nil {
		t.Fatal(second, err)
	}
	if !reflect.DeepEqual(calls, []string{"PUT /swift/a", "PUT /swift/a"}) {
		t.Fatal(calls)
	}
	encoded, _ := json.Marshal(second.Resource)
	if !strings.Contains(string(encoded), `"name":"a"`) {
		t.Fatal(string(encoded))
	}
}
