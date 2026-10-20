package containers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSwiftPythonContainerMetadataSetterRefreshesObservedStateAndSourceValues(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/swift/v1/AUTH_selected/")
	client.MoreHeaders = map[string]string{"X-Source": "selected"}
	var methods []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
			t.Error("nonempty request body", string(body), err)
		}
		if r.URL.Path != "/swift/v1/AUTH_selected/container" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Source") != "selected" {
			t.Error(r.Method, r.URL, r.Header)
		}
		if r.Method == "POST" {
			for key, want := range map[string]string{"Owner": "requested", "Enabled": "False", "Unset": "None", "Large": "900719925474099312345", "List": "['x', True]", "Already": "prefixed"} {
				if r.Header.Get("X-Container-Meta-"+key) != want {
					t.Error(key, r.Header)
				}
			}
			if r.Header.Get("X-Container-Read") != ".r:*" || r.Header.Get("X-History-Location") != "archive" {
				t.Error("missing audited system aliases", r.Header)
			}
			w.Header().Set("X-Proof", "ack")
			w.WriteHeader(204)
			return
		}
		if r.Header.Get("X-Container-Meta-Owner") != "" {
			t.Error("POST fields leaked into refresh HEAD", r.Header)
		}
		w.Header().Set("X-Container-Meta-Owner", "actually observed")
		w.Header().Set("X-Container-Object-Count", "7")
		w.Header().Set("X-Proof", "head")
		w.WriteHeader(204)
	})
	result, err := containers.New(client).SetMetadataAndRefresh(context.Background(), "container", map[string]any{"Owner": "requested", "read_ACL": ".r:*", "history_location": "archive", "Enabled": false, "Unset": nil, "Large": json.Number("900719925474099312345"), "List": []any{"x", true}, "X-Container-Meta-Already": "prefixed"})
	if err != nil || result.Acknowledgement.StatusCode != 204 || result.Acknowledgement.Header.Get("X-Proof") != "ack" || result.Refreshed.StatusCode != 204 || result.Refreshed.Header.Get("X-Proof") != "head" || result.Refreshed.Metadata.Values["owner"] != "actually observed" || *result.Refreshed.Metadata.ObjectCount != 7 || !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) {
		t.Fatal(result, err, methods)
	}
	if !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "selected"}) {
		t.Fatal(client.MoreHeaders)
	}
}

func TestSwiftPythonContainerMetadataEmptyAndPrefixedDeleteFollowSourcePOST(t *testing.T) {
	for _, keys := range [][]string{nil, {}, {"Owner", "Owner", "X-Container-Meta-Prefixed"}} {
		t.Run(fmt.Sprint(keys), func(t *testing.T) {
			var calls int
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.String() != containerMetadataBase+"container" || r.Body != nil {
					t.Error(r.Method, r.URL)
				}
				if len(keys) > 0 {
					for _, key := range []string{"X-Container-Meta-Owner", "X-Container-Meta-Prefixed"} {
						values, present := r.Header[http.CanonicalHeaderKey(key)]
						if !present || !reflect.DeepEqual(values, []string{""}) {
							t.Error(key, r.Header)
						}
					}
				}
				return containerMetadataResponse(299, "opaque deletion ack"), nil
			})
			result, err := containers.New(client).DeleteMetadataKeys(context.Background(), "container", keys)
			if err != nil || result.StatusCode != 299 || string(result.Body) != "opaque deletion ack" || calls != 1 {
				t.Fatal(result, err, calls)
			}
		})
	}
	var methods []string
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		return containerMetadataResponse(204, ""), nil
	})
	if _, err := containers.New(client).SetMetadataAndRefresh(context.Background(), "container", nil); err != nil || !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) {
		t.Fatal(err, methods)
	}
}

func TestSwiftPythonContainerMetadataRefreshFailuresKeepAcknowledgementWithoutReplay(t *testing.T) {
	for _, phase := range []string{"POST rejected", "HEAD rejected", "POST Close", "HEAD Close", "cancel after POST"} {
		t.Run(phase, func(t *testing.T) {
			calls := 0
			closeCause := errors.New("physical Close failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cancelCause := errors.New("after acknowledged POST")
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				code := 204
				if phase == r.Method+" rejected" {
					code = 404
				}
				body := &containerMetadataBody{Reader: strings.NewReader("physical")}
				if phase == r.Method+" Close" {
					body.closeErr = closeCause
				}
				if phase == "cancel after POST" && r.Method == "POST" {
					body.onClose = func() { cancel(cancelCause) }
				}
				return containerMetadataWire(code, body), nil
			})
			result, err := containers.New(client).SetMetadataAndRefresh(ctx, "container", map[string]any{"owner": "input"})
			if err == nil || result == nil {
				t.Fatal(result, err)
			}
			if strings.HasPrefix(phase, "POST") || phase == "cancel after POST" {
				if calls != 1 || result.Refreshed != nil {
					t.Fatal("failed POST was replayed or refreshed", calls, result)
				}
			} else if calls != 2 || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 204 {
				t.Fatal("lost prior acknowledged write", calls, result)
			}
			if strings.HasSuffix(phase, "rejected") {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "physical" {
					t.Fatal(err, native)
				}
			}
			if strings.HasSuffix(phase, "Close") && !errors.Is(err, closeCause) {
				t.Fatal(err)
			}
			if phase == "cancel after POST" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal(err)
			}
		})
	}
}

func TestSwiftPythonContainerMetadataInputsCallbacksAndConcurrentReuseAreOwned(t *testing.T) {
	values := map[string]any{"owner": "original", "nested": []any{"before"}}
	var kept *containers.MetadataRefreshOpts
	option := func(cfg *containers.MetadataRefreshOpts) error {
		values["owner"] = "caller"
		values["nested"].([]any)[0] = "changed"
		cfg.Headers["X-Probe"] = "snapshot"
		kept = cfg
		return nil
	}
	var calls int
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		kept.Headers = map[string]string{"X-Probe": "late"}
		if r.Header.Get("X-Probe") != "snapshot" {
			t.Error(r.Header)
		}
		if r.Method == "POST" && (r.Header.Get("X-Container-Meta-Owner") != "original" || r.Header.Get("X-Container-Meta-Nested") != "['before']") {
			t.Error(r.Header)
		}
		return containerMetadataResponse(204, ""), nil
	})
	if _, err := containers.New(client).SetMetadataAndRefresh(context.Background(), "container", values, option); err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	cloud := testcloud.New(t)
	api := containers.New(cloud.Client("object-store", "/account/"))
	headers := map[string]string{"X-Probe": "snapshot"}
	reusable := containers.WithMetadataRefreshHeaders(headers)
	headers["X-Probe"] = "caller"
	var concurrentCalls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		concurrentCalls.Add(1)
		if r.Header.Get("X-Probe") != "snapshot" {
			t.Error(r.Header)
		}
		w.WriteHeader(204)
	})
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := api.SetMetadataAndRefresh(context.Background(), "container", map[string]any{"owner": "same"}, reusable); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	if concurrentCalls.Load() != 16 {
		t.Fatal(concurrentCalls.Load())
	}
}

func TestSwiftPythonContainerMetadataPreflightAndSourceStatusPolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399} {
		var calls int
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return containerMetadataResponse(code, "opaque"), nil
		})
		result, err := containers.New(client).SetMetadataAndRefresh(context.Background(), "container", map[string]any{})
		if err != nil || result.Acknowledgement.StatusCode != code || result.Refreshed.StatusCode != code || calls != 2 {
			t.Fatal(code, result, err, calls)
		}
	}
	var calls int
	client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(204, ""), nil })
	api := containers.New(client)
	for _, values := range []map[string]any{{"owner": "x", "X-Container-Meta-Owner": "collision"}, {"bad\r\nkey": "x"}, {"key": func() {}}} {
		if _, err := api.SetMetadataAndRefresh(context.Background(), "container", values); err == nil {
			t.Fatal(values)
		}
	}
	cause := errors.New("precanceled caller")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if _, err := api.SetMetadataAndRefresh(ctx, "container", nil); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if _, err := api.DeleteMetadataKeys(context.Background(), "container", nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal(calls)
	}
}

func TestSwiftPythonContainerMetadataRefreshFalseLaterOptionsAndReusableSnapshots(t *testing.T) {
	refresh := false
	input := containers.MetadataRefreshOpts{Refresh: &refresh, Headers: map[string]string{"X-Probe": "owned"}}
	reusable := containers.WithMetadataRefreshOpts(input)
	refresh = true
	input.Headers["X-Probe"] = "caller"
	var methods []string
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		if r.Header.Get("X-Probe") != "owned" {
			t.Error(r.Header)
		}
		return containerMetadataResponse(204, ""), nil
	})
	api := containers.New(client)
	for i := 0; i < 2; i++ {
		result, err := api.SetMetadataAndRefresh(context.Background(), "container", map[string]any{}, reusable)
		if err != nil || result.Acknowledgement == nil || result.Refreshed != nil {
			t.Fatal(result, err)
		}
	}
	if !reflect.DeepEqual(methods, []string{"POST", "POST"}) {
		t.Fatal(methods)
	}
	methods = nil
	result, err := api.SetMetadataAndRefresh(context.Background(), "container", nil, reusable, containers.WithMetadataRefresh(true))
	if err != nil || result.Refreshed == nil || !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) {
		t.Fatal(result, err, methods)
	}
	methods = nil
	result, err = api.SetMetadataAndRefresh(context.Background(), "container", nil, reusable, containers.WithMetadataRefreshOpts(containers.MetadataRefreshOpts{Headers: map[string]string{"X-Probe": "owned"}}))
	if err != nil || result.Refreshed == nil || !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) {
		t.Fatal(result, err, methods)
	}
}

func TestSwiftPythonContainerMetadataEverySystemAliasAndPrefixedConvention(t *testing.T) {
	aliases := map[string]string{
		"content_type": "Content-Type", "is_content_type_detected": "X-Detect-Content-Type", "versions_location": "X-Versions-Location", "history_location": "X-History-Location",
		"read_ACL": "X-Container-Read", "write_ACL": "X-Container-Write", "sync_to": "X-Container-Sync-To", "sync_key": "X-Container-Sync-Key",
	}
	for alias, wire := range aliases {
		for _, key := range []string{alias, strings.ToLower(wire)} {
			t.Run(key, func(t *testing.T) {
				var calls int
				client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Header.Get(wire) != "value" || r.Header.Get("X-Container-Meta-"+key) != "" {
						t.Error(r.Header)
					}
					return containerMetadataResponse(204, ""), nil
				})
				if _, err := containers.New(client).SetMetadataAndRefresh(context.Background(), "container", map[string]any{key: "value"}, containers.WithMetadataRefresh(false)); err != nil || calls != 1 {
					t.Fatal(err, calls)
				}
			})
		}
	}
	var calls int
	client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		for _, key := range []string{"X-Container-Read", "X-History-Location", "X-Container-Meta-Owner"} {
			if value, present := r.Header[http.CanonicalHeaderKey(key)]; !present || !reflect.DeepEqual(value, []string{""}) {
				t.Error(key, r.Header)
			}
		}
		return containerMetadataResponse(204, ""), nil
	})
	if _, err := containers.New(client).DeleteMetadataKeys(context.Background(), "container", []string{"read_ACL", "history_location", "X-Container-Meta-Owner"}); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
}

type swiftPythonContainerMetadataMarshaler func() ([]byte, error)

func (marshal swiftPythonContainerMetadataMarshaler) MarshalJSON() ([]byte, error) { return marshal() }

func TestSwiftPythonContainerMetadataCapturesSourceBeforeCustomJSONMarshaler(t *testing.T) {
	var calls int
	client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(204, ""), nil })
	api := containers.New(client)
	input := swiftPythonContainerMetadataMarshaler(func() ([]byte, error) {
		client.Endpoint = "https://swift.invalid/changed/AUTH_other/"
		return []byte(`"value"`), nil
	})
	_, err := api.SetMetadataAndRefresh(context.Background(), "container", map[string]any{"custom": input})
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("marshal retargeted selected Source", err, calls)
	}
}
