package backups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const pythonBackupMetadataPath = pythonBackupBase + "/bk-1/metadata"

func TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON(t *testing.T) {
	const body = `{"metadata":{"owner":"wire","large":900719925474099312345,"null":null,"array":[false,1]},"extension":null}`
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		response := pythonBackupWire(203, body)
		response.Header.Set("X-Request-Id", "physical-fetch")
		if req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("X-Probe") != "selected" || req.Header.Get("OpenStack-API-Version") != "volume 3.42" {
			t.Error("lost selected headers", req.Header)
		}
		return response
	})
	api.RawClient().Microversion = "3.42" // MetadataMixin adds no 3.43 gate.
	profile := api.PythonMetadata()
	for _, get := range []func(context.Context, string, ...backups.MetadataOption) (*backups.MetadataResult, error){profile.FetchMetadata, profile.GetMetadata} {
		result, err := get(context.Background(), "bk-1", backups.WithMetadataHeader("X-Probe", "selected"))
		if err != nil || result.BackupID != "bk-1" || string(result.Metadata) != `{"owner":"wire","large":900719925474099312345,"null":null,"array":[false,1]}` || result.Response.StatusCode != 203 || string(result.Response.Body) != body || result.Response.Header.Get("X-Request-Id") != "physical-fetch" {
			t.Fatal(result, err)
		}
		// Metadata value, physical body and subsequent results own their bytes.
		result.Metadata[0] = '['
		if string(result.Response.Body) != body {
			t.Fatal("observed value aliases physical body")
		}
		result.Response.Header.Set("X-Request-Id", "caller")
	}
	if len(*calls) != 2 {
		t.Fatal(*calls)
	}
	for _, call := range *calls {
		if call.method != "GET" || call.path != pythonBackupMetadataPath || call.query != "" || call.body != "" || call.version != "volume 3.42" {
			t.Fatal(call)
		}
	}
}

func TestPythonBackupMetadataProfileFetchOmissionNullAndDecodeFailures(t *testing.T) {
	for _, check := range []struct {
		body, value string
		valid       bool
	}{
		{`{}`, "", true}, {`{"Metadata":{"decoy":1}}`, "", true},
		{`{"metadata":null}`, "null", true}, {`{"metadata":[]}`, "[]", true},
		{`{"metadata":"arbitrary"}`, `"arbitrary"`, true}, {`{"metadata":{"duplicate":1,"duplicate":2}}`, `{"duplicate":1,"duplicate":2}`, true},
		{`{"metadata":{}}`, `{}`, true}, {`{"metadata":`, "", false}, {`null`, "", false}, {`[]`, "", false}, {`{} {}`, "", false},
		{"{\"metadata\":\"" + string([]byte{0xff}) + "\"}", "", false},
	} {
		t.Run(check.body, func(t *testing.T) {
			api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response {
				response := pythonBackupWire(200, check.body)
				response.Header.Set("X-Proof", "decode")
				return response
			})
			result, err := api.PythonMetadata().FetchMetadata(context.Background(), "bk-1")
			if result == nil || result.Response == nil || result.Response.StatusCode != 200 || string(result.Response.Body) != check.body || result.Response.Header.Get("X-Proof") != "decode" || len(*calls) != 1 {
				t.Fatal(result, err, *calls)
			}
			if check.valid {
				if err != nil || string(result.Metadata) != check.value || (check.value == "" && result.Metadata != nil) {
					t.Fatal(result, err)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != check.body || result.Metadata != nil {
					t.Fatal(result, err, proof)
				}
			}
		})
	}
}

func TestPythonBackupMetadataProfileSetOwnsInputAndOpaqueAcknowledgement(t *testing.T) {
	for _, values := range []map[string]string{nil, {}, {"owner": "requested", "empty": "", "Unicode": "서울"}} {
		t.Run(fmt.Sprint(values), func(t *testing.T) {
			const ack = "opaque acknowledgement: not JSON"
			api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response { return pythonBackupWire(299, ack) })
			expected := `{"metadata":{}}`
			if len(values) != 0 {
				expected = `{"metadata":{"Unicode":"서울","empty":"","owner":"requested"}}`
			}
			mutate := func(config *request.Config[metadata.Opts]) error {
				if values != nil {
					values["owner"] = "mutated"
					values["new"] = "caller"
				}
				return nil
			}
			result, err := api.PythonMetadata().SetMetadata(context.Background(), "bk-1", values, mutate)
			if err != nil || result.BackupID != "bk-1" || result.Metadata != nil || result.Response.StatusCode != 299 || string(result.Response.Body) != ack || len(*calls) != 1 {
				t.Fatal(result, err, *calls)
			}
			call := (*calls)[0]
			if call.method != "POST" || call.path != pythonBackupMetadataPath || call.query != "" || call.body != expected {
				t.Fatal(call)
			}
		})
	}
}

func TestPythonBackupMetadataProfileDeleteNilEmptyLiteralKeysAndPartialFailure(t *testing.T) {
	t.Run("nil clears with opaque PUT", func(t *testing.T) {
		api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response { return pythonBackupWire(202, "clear ack") })
		result, err := api.PythonMetadata().DeleteMetadata(context.Background(), "bk-1", nil)
		if err != nil || result.Cleared == nil || result.Cleared.StatusCode != 202 || string(result.Cleared.Body) != "clear ack" || result.Cleared.Metadata != nil || len(result.Deleted) != 0 || !reflect.DeepEqual(*calls, []pythonBackupCall{{"PUT", pythonBackupMetadataPath, "", `{"metadata":{}}`, ""}}) {
			t.Fatal(result, err, *calls)
		}
	})
	t.Run("explicit empty never sends HTTP", func(t *testing.T) {
		api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response { t.Error("unexpected request"); return pythonBackupWire(200, "") })
		result, err := api.PythonMetadata().DeleteMetadata(context.Background(), "bk-1", []string{})
		if err != nil || result.Cleared != nil || result.Deleted == nil || len(result.Deleted) != 0 || len(*calls) != 0 {
			t.Fatal(result, err, *calls)
		}
	})
	t.Run("literal keys duplicated and partial proof", func(t *testing.T) {
		var paths []string
		api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
			paths = append(paths, req.URL.EscapedPath())
			code := 200
			if len(paths) == 3 {
				code = 404
			}
			response := pythonBackupWire(code, fmt.Sprintf("physical-ack-%d", len(paths)))
			response.Header.Set("X-Ack", fmt.Sprint(len(paths)))
			return response
		})
		keys := []string{"literal/ %é?fragment#", "duplicate", "duplicate", "never"}
		mutate := func(*request.Config[metadata.Opts]) error { keys[0] = "mutated"; keys[1] = "caller"; return nil }
		result, err := api.PythonMetadata().DeleteMetadata(context.Background(), "bk-1", keys, mutate)
		var native gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Cleared != nil || len(result.Deleted) != 2 || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "physical-ack-3" || native.ResponseHeader.Get("X-Ack") != "3" || len(*calls) != 3 {
			t.Fatal(result, err, native, *calls)
		}
		expected := []string{pythonBackupMetadataPath + "/" + url.PathEscape("literal/ %é?fragment#"), pythonBackupMetadataPath + "/duplicate", pythonBackupMetadataPath + "/duplicate"}
		if !reflect.DeepEqual(paths, expected) || result.Deleted[0].Key != "literal/ %é?fragment#" || result.Deleted[1].Key != "duplicate" {
			t.Fatal(paths, result)
		}
		for i, ack := range result.Deleted {
			if ack.StatusCode != 200 || string(ack.Body) != fmt.Sprintf("physical-ack-%d", i+1) || ack.Header.Get("X-Ack") != fmt.Sprint(i+1) || (*calls)[i].method != "DELETE" || (*calls)[i].body != "" || (*calls)[i].query != "" {
				t.Fatal(ack, (*calls)[i])
			}
		}
	})
}

// Existing Python backup fixtures and the public Gophercloud testhelper supply
// HTTP. This wrapper adds only response-boundary faults to the same fixture.
type backupMetadataFaultBody struct {
	io.ReadCloser
	readErr, closeErr error
	afterRead         func()
	closes            *atomic.Int32
}

func (body *backupMetadataFaultBody) Read(value []byte) (int, error) {
	n, err := body.ReadCloser.Read(value)
	if body.afterRead != nil {
		body.afterRead()
	}
	if body.readErr != nil {
		return n, body.readErr
	}
	return n, err
}
func (body *backupMetadataFaultBody) Close() error {
	if body.closes != nil {
		body.closes.Add(1)
	}
	return errors.Join(body.ReadCloser.Close(), body.closeErr)
}

func backupMetadataOperations() []struct {
	name string
	call func(context.Context, *backups.PythonMetadataProfile) error
} {
	return []struct {
		name string
		call func(context.Context, *backups.PythonMetadataProfile) error
	}{
		{"FetchMetadata", func(ctx context.Context, p *backups.PythonMetadataProfile) error {
			_, err := p.FetchMetadata(ctx, "bk-1")
			return err
		}},
		{"GetMetadata", func(ctx context.Context, p *backups.PythonMetadataProfile) error {
			_, err := p.GetMetadata(ctx, "bk-1")
			return err
		}},
		{"SetMetadata", func(ctx context.Context, p *backups.PythonMetadataProfile) error {
			_, err := p.SetMetadata(ctx, "bk-1", map[string]string{"owner": "input"})
			return err
		}},
		{"DeleteMetadata", func(ctx context.Context, p *backups.PythonMetadataProfile) error {
			_, err := p.DeleteMetadata(ctx, "bk-1", []string{"key"})
			return err
		}},
	}
}

func TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback(t *testing.T) {
	for _, operation := range backupMetadataOperations() {
		for _, status := range []int{200, 201, 202, 203, 299, 300, 399, 403, 404, 412, 500} {
			t.Run(fmt.Sprintf("%s/%d", operation.name, status), func(t *testing.T) {
				api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response {
					response := pythonBackupWire(status, `{"metadata":{"actual":"value"}}`)
					response.Header.Set("X-Proof", "physical-status")
					return response
				})
				err := operation.call(context.Background(), api.PythonMetadata())
				if len(*calls) != 1 {
					t.Fatal("child route fallback or replay", *calls)
				}
				if status < 400 {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				var wrapped *resource.OperationError
				if !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"metadata":{"actual":"value"}}` || native.ResponseHeader.Get("X-Proof") != "physical-status" || !errors.As(err, &wrapped) || wrapped.Operation != operation.name || wrapped.Resource != "backups.metadata" {
					t.Fatal(err, native, wrapped)
				}
				if !strings.Contains(native.URL, "/backups/bk-1/metadata") || (*calls)[0].query != "" {
					t.Fatal(native, *calls)
				}
			})
		}
	}
}

func TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses(t *testing.T) {
	for _, operation := range backupMetadataOperations() {
		for _, status := range []int{200, 404} {
			for _, mode := range []string{"read", "close", "both", "cancel"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation.name, status, mode), func(t *testing.T) {
					readFailure, closeFailure := io.ErrUnexpectedEOF, errors.New("physical Close failure")
					cancelCause := errors.New("body-boundary cancel cause")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var closes atomic.Int32
					api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response {
						response := pythonBackupWire(status, `{"metadata":{"actual":"value"}}`)
						response.Header.Set("X-Proof", "physical-body")
						body := &backupMetadataFaultBody{ReadCloser: response.Body, closes: &closes}
						if mode == "read" || mode == "both" {
							body.readErr = readFailure
						}
						if mode == "close" || mode == "both" {
							body.closeErr = closeFailure
						}
						if mode == "cancel" {
							body.afterRead = func() { cancel(cancelCause) }
						}
						response.Body = body
						return response
					})
					err := operation.call(ctx, api.PythonMetadata())
					if err == nil || closes.Load() != 1 || len(*calls) != 1 {
						t.Fatal(err, closes.Load(), *calls)
					}
					if (mode == "read" || mode == "both") && !errors.Is(err, readFailure) {
						t.Fatal("lost Read cause", err)
					}
					if (mode == "close" || mode == "both") && !errors.Is(err, closeFailure) {
						t.Fatal("lost Close cause", err)
					}
					if mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
						t.Fatal("lost cancellation cause", err)
					}
					if status == 200 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"metadata":{"actual":"value"}}` || proof.Header.Get("X-Proof") != "physical-body" {
							t.Fatal(err, proof)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"metadata":{"actual":"value"}}` || native.ResponseHeader.Get("X-Proof") != "physical-body" {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}

func TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses(t *testing.T) {
	for _, operation := range backupMetadataOperations() {
		t.Run(operation.name, func(t *testing.T) {
			api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response { t.Error("preflight sent HTTP"); return pythonBackupWire(200, `{}`) })
			cause := errors.New("before HTTP cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(cause)
			err := operation.call(ctx, api.PythonMetadata())
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
			var nilProfile *backups.PythonMetadataProfile
			if err := operation.call(context.Background(), nilProfile); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if err := operation.call(nil, api.PythonMetadata()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response {
		t.Error("invalid option sent HTTP")
		return pythonBackupWire(200, `{}`)
	})
	profile := api.PythonMetadata()
	for _, option := range []backups.MetadataOption{nil, backups.WithMetadataHeader("X-Auth-Token", "foreign"), backups.WithMetadataHeader("Content-Type", "text/plain"), backups.WithMetadataHeader("OpenStack-API-Version", "volume 3.43"), request.WithQuery[metadata.Opts]("limit", "1"), request.WithField[metadata.Opts]("metadata", map[string]string{})} {
		if _, err := profile.FetchMetadata(context.Background(), "bk-1", option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"", "..", "a/b", "escaped%20id"} {
		if _, err := profile.FetchMetadata(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	for _, key := range []string{"", ".", "..", "nul\x00"} {
		if _, err := profile.DeleteMetadata(context.Background(), "bk-1", []string{key}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	wireCause := errors.New("physical metadata transport failure")
	api.RawClient().ProviderClient.HTTPClient.Transport = pythonBackupTransport(func(*http.Request) (*http.Response, error) { return nil, wireCause })
	for _, operation := range backupMetadataOperations() {
		if err := operation.call(context.Background(), profile); !errors.Is(err, wireCause) {
			t.Fatal(err)
		}
	}
}

func TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("volume", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.MoreHeaders = map[string]string{"X-Source": "original"}
	client.Microversion = "3.43"
	api := backups.New(client)
	profile := api.PythonMetadata()
	headers := map[string]string{"X-Probe": "snapshot"}
	option := backups.WithMetadataHeaders(headers)
	headers["X-Probe"] = "caller changed"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.URL.Path != pythonBackupMetadataPath || req.Method != "GET" || req.Header.Get("X-Probe") != "snapshot" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != "volume 3.43" {
			t.Error(req.Method, req.URL, req.Header)
		}
		testcloud.JSON(w, 200, `{"metadata":{"wire":"value"}}`)
	})
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := profile.FetchMetadata(context.Background(), "bk-1", option); err != nil {
				t.Error(err)
			}
		}()
	}
	wait.Wait()
	if calls.Load() != 8 || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "original"}) {
		t.Fatal(calls.Load(), client.MoreHeaders)
	}
	var kept *request.Config[metadata.Opts]
	keep := func(config *request.Config[metadata.Opts]) error {
		config.Headers["X-Probe"] = "snapshot"
		kept = config
		return nil
	}
	transport := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = pythonBackupTransport(func(req *http.Request) (*http.Response, error) {
		kept.Headers = map[string]string{"X-Probe": "late mutation"}
		return transport.RoundTrip(req)
	})
	if _, err := profile.FetchMetadata(context.Background(), "bk-1", keep); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 9 {
		t.Fatal(calls.Load())
	}
	cloud.Provider.HTTPClient.Transport = pythonBackupTransport(func(req *http.Request) (*http.Response, error) {
		response, err := transport.RoundTrip(req)
		client.ResourceBase = cloud.Server.URL + "/changed/"
		return response, err
	})
	result, err := profile.FetchMetadata(context.Background(), "bk-1", option)
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || result.Metadata != nil || calls.Load() != 10 {
		t.Fatal(result, err, proof, calls.Load())
	}
}
