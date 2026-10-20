package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const detachVolumeContractNovaBase = "/reverse/nova/v2.1/project/"
const detachVolumeContractCinderBase = "/reverse/cinder/v3/project/"
const detachVolumeContractDeletePath = detachVolumeContractNovaBase + "servers/server-1/os-volume_attachments/vol-1"
const detachVolumeContractVolumePath = detachVolumeContractCinderBase + "volumes/vol-1"

func detachVolumeContractClients(cloud *testcloud.Cloud) (*gophercloud.ServiceClient, *gophercloud.ServiceClient) {
	nova, cinder := cloud.Client("compute", "/unused/nova/"), cloud.Client("volumev3", "/unused/cinder/")
	nova.ResourceBase, cinder.ResourceBase = cloud.Server.URL+detachVolumeContractNovaBase, cloud.Server.URL+detachVolumeContractCinderBase
	nova.Microversion, cinder.Microversion = "2.79", "3.60"
	nova.MoreHeaders, cinder.MoreHeaders = map[string]string{"x-source": "nova-entry"}, map[string]string{"x-source": "cinder-entry"}
	return nova, cinder
}
func detachVolumeContractInput() blockstorage.DetachVolumeRequest {
	return blockstorage.DetachVolumeRequest{Server: resource.ID("server-1"), Volume: resource.ID("vol-1")}
}
func detachVolumeContractObservation(status, attachments string) string {
	fields := ""
	if attachments != "" {
		fields = `,"attachments":` + attachments
	}
	return fmt.Sprintf(`{"volume":{"id":"vol-1","name":"data","status":%q%s,"created_at":"literal Cinder time","vendor":{"number":9007199254740993}}}`, status, fields)
}
func detachVolumeContractWire(t *testing.T, r *http.Request, method, path, token string) {
	t.Helper()
	source, version := "cinder-entry", "volume 3.60"
	if strings.HasPrefix(path, detachVolumeContractNovaBase) {
		source, version = "nova-entry", "compute 2.79"
	}
	if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != source || r.Header.Get("OpenStack-API-Version") != version || r.Header.Get("X-Auth-Token") != token {
		t.Errorf("method=%s URL=%s headers=%v", r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("bodyless detach phase sent %q error=%v", body, err)
		}
	}
}
func detachVolumeContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "DetachVolume" || operation.Cause == nil {
		t.Fatalf("detach operation context missing: %v", err)
	}
}
func detachVolumeContractDeleted(t *testing.T, result *blockstorage.DetachVolumeResult, code int, body []byte) {
	t.Helper()
	if result == nil || result.ServerID != "server-1" || result.VolumeID != "vol-1" || result.Deleted == nil || result.Deleted.StatusCode != code || !bytes.Equal(result.Deleted.Body, body) || result.Deleted.Header.Get("X-Proof") != "deleted" {
		t.Fatalf("accepted deletion acknowledgement missing: %+v", result)
	}
}

func TestDetachVolumeContractsDefaultDeleteThenFreshAvailableAndIndependentProof(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := detachVolumeContractClients(cloud)
	opaque := []byte{0, 0xff, 'a', '\n'}
	ready := detachVolumeContractObservation("AVAILABLE", `[{"server_id":"server-1","device":null,"vendor":9007199254740993}]`)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
			w.Header().Set("X-Proof", "deleted")
			w.Header().Set("Location", "https://foreign.invalid/not-followed")
			w.WriteHeader(202)
			_, _ = w.Write(opaque)
		case 2:
			detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
			w.Header().Set("X-Proof", "ready")
			testcloud.JSON(w, 200, ready)
		default:
			t.Error("unexpected precondition, lookup or cleanup", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput())
	detachVolumeContractDeleted(t, result, 202, opaque)
	if err != nil || calls.Load() != 2 || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != ready || result.LastAccepted.Header.Get("X-Proof") != "ready" || result.Ready == nil || *result.Ready.Status != "AVAILABLE" || result.Ready.CreatedAt == nil || *result.Ready.CreatedAt != "literal Cinder time" || len(result.Ready.Attachments) != 1 || string(result.Ready.Attachments[0].Body["vendor"]) != "9007199254740993" || result.Ready == result.LastAccepted.Volume {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	result.Deleted.Body[0] = '!'
	result.Deleted.Header.Set("X-Proof", "caller")
	*result.Ready.Status = "caller"
	result.Ready.Header.Set("X-Proof", "caller")
	result.Ready.Body["vendor"][0] = '!'
	result.Ready.Attachments[0].Body["vendor"][0] = '!'
	*result.Ready.Attachments[0].ServerID = "caller"
	if string(result.LastAccepted.Body) != ready || result.LastAccepted.Header.Get("X-Proof") != "ready" || *result.LastAccepted.Volume.Status != "AVAILABLE" || string(result.LastAccepted.Volume.Body["vendor"]) != `{"number":9007199254740993}` || string(result.LastAccepted.Volume.Attachments[0].Body["vendor"]) != "9007199254740993" || *result.LastAccepted.Volume.Attachments[0].ServerID != "server-1" {
		t.Fatal("readiness or phase evidence aliases", result)
	}
}

func TestDetachVolumeContractsNoWaitIDsNeedOnlyNovaAndOpaqueAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body []byte
	}{
		{"opaque binary", 202, []byte{0, 0xff, 'x'}},
		{"incomplete JSON", 202, []byte(`{"unfinished":`)},
		{"empty accepted", 202, nil},
		{"no content", 204, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, _ := detachVolumeContractClients(cloud)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
				w.Header().Set("X-Proof", "deleted")
				w.WriteHeader(tc.code)
				_, _ = w.Write(tc.body)
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, nil, detachVolumeContractInput(), blockstorage.WithDetachVolumeWait(false), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{ProgressCallback: func(int) error { callbacks.Add(1); return nil }}))
			detachVolumeContractDeleted(t, result, tc.code, tc.body)
			if err != nil || result.LastAccepted != nil || result.Ready != nil || calls.Load() != 1 || callbacks.Load() != 0 {
				t.Fatalf("result=%+v error=%v HTTP=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
		})
	}
}

func TestDetachVolumeContractsPreflightAndRequiredCinderBeforeNameLookup(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, tc := range []struct {
		name                                              string
		change                                            func(*gophercloud.ServiceClient, *gophercloud.ServiceClient, *blockstorage.DetachVolumeRequest)
		nilContext, nilNova, nilCinder, wait, unsupported bool
		option                                            blockstorage.DetachVolumeOption
		callbacks                                         int32
	}{
		{name: "nil context", nilContext: true}, {name: "nil Nova", nilNova: true},
		{name: "required waiting Cinder", nilCinder: true, wait: true, callbacks: 1},
		{name: "required volume Name Cinder before server Name", nilCinder: true, callbacks: 1, change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.DetachVolumeRequest) {
			i.Server = resource.Name("worker")
			i.Volume = resource.Name("data")
		}},
		{name: "invalid unused Cinder provider", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) { c.ProviderClient = nil }},
		{name: "invalid unused Cinder service", unsupported: true, change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) { c.Type = "image" }},
		{name: "invalid Nova service", unsupported: true, change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) { n.Type = "network" }},
		{name: "relative Nova endpoint", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) { n.Endpoint = "/relative/" }},
		{name: "foreign effective base", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) {
			c.ResourceBase = "https://foreign.invalid/v3/"
		}},
		{name: "reserved authentication header", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.DetachVolumeRequest) {
			n.MoreHeaders["X-Auth-Token"] = "other"
		}},
		{name: "unsafe explicit volume", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.DetachVolumeRequest) {
			i.Volume = resource.ID("other/volume")
		}},
		{name: "invalid server Name", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.DetachVolumeRequest) {
			i.Server = resource.Name("name\n")
		}},
		{name: "inactive negative timeout", callbacks: 1, option: blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &negative})},
		{name: "inactive zero interval", callbacks: 1, option: blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{PollInterval: &zero})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			input := detachVolumeContractInput()
			if tc.change != nil {
				tc.change(nova, cinder, &input)
			}
			if tc.nilNova {
				nova = nil
			}
			if tc.nilCinder {
				cinder = nil
			}
			ctx := context.Background()
			if tc.nilContext {
				ctx = nil
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid detach reached lookup or mutation", r.Method, r.URL)
				w.WriteHeader(500)
			})
			options := []blockstorage.DetachVolumeOption{func(*blockstorage.DetachVolumeOpts) error { callbacks.Add(1); return nil }, blockstorage.WithDetachVolumeWait(tc.wait)}
			if tc.option != nil {
				options = append(options, tc.option)
			}
			result, err := blockstorage.DetachVolume(ctx, nova, cinder, input, options...)
			want := resource.ErrInvalidOption
			if tc.unsupported {
				want = resource.ErrUnsupported
			}
			if result != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != tc.callbacks {
				t.Fatalf("result=%+v error=%v HTTP=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
			detachVolumeContractOperation(t, err)
		})
	}
	t.Run("canceled entry retains custom cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := detachVolumeContractClients(cloud)
		cause := errors.New("caller canceled detach")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		calls := 0
		result, err := blockstorage.DetachVolume(ctx, nova, cinder, detachVolumeContractInput(), func(*blockstorage.DetachVolumeOpts) error { calls++; return nil })
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
			t.Fatalf("result=%+v error=%v callbacks=%d", result, err, calls)
		}
	})
}

func TestDetachVolumeContractsNamesResolveOnceServerThenVolumeWithoutSwap(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := detachVolumeContractClients(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			if r.URL.Path != detachVolumeContractNovaBase+"servers/detail" || r.URL.Query().Get("name") != "^worker\\?literal$" {
				t.Error("server Name lookup", r.URL)
			}
			testcloud.JSON(w, 200, `{"servers":[{"id":"server-1","name":"worker?literal"}]}`)
		case 2:
			if r.URL.Path != detachVolumeContractCinderBase+"volumes/detail" || r.URL.Query().Get("name") != "data" {
				t.Error("volume Name lookup", r.URL)
			}
			testcloud.JSON(w, 200, `{"volumes":[{"id":"vol-1","name":"data","attachments":[]}]}`)
		case 3:
			detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
			w.Header().Set("X-Proof", "deleted")
			w.WriteHeader(204)
		default:
			t.Error("unexpected pre-delete lookup, ID swap or Cinder guard", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, blockstorage.DetachVolumeRequest{Server: resource.Name("worker?literal"), Volume: resource.Name("data")}, blockstorage.WithDetachVolumeWait(false))
	detachVolumeContractDeleted(t, result, 204, nil)
	if err != nil || calls.Load() != 3 || result.LastAccepted != nil {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	t.Run("server Name only requires Nova without waiting", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, _ := detachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				if r.URL.Path != detachVolumeContractNovaBase+"servers/detail" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"servers":[{"id":"server-1","name":"worker"}]}`)
				return
			}
			detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
			w.Header().Set("X-Proof", "deleted")
			w.WriteHeader(204)
		})
		result, err := blockstorage.DetachVolume(context.Background(), nova, nil, blockstorage.DetachVolumeRequest{Server: resource.Name("worker"), Volume: resource.ID("vol-1")}, blockstorage.WithDetachVolumeWait(false))
		detachVolumeContractDeleted(t, result, 204, nil)
		if err != nil || calls.Load() != 2 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
	for _, tc := range []struct {
		name, rows string
		cause      error
	}{{"missing", `[]`, resource.ErrNotFound}, {"ambiguous", `[{"id":"one","name":"worker"},{"id":"two","name":"worker"}]`, resource.ErrAmbiguous}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != detachVolumeContractNovaBase+"servers/detail" {
					t.Error(r.Method, r.URL)
				}
				testcloud.JSON(w, 200, `{"servers":`+tc.rows+`}`)
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, blockstorage.DetachVolumeRequest{Server: resource.Name("worker"), Volume: resource.ID("vol-1")})
			if result != nil || !errors.Is(err, tc.cause) || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestDetachVolumeContractsMissingAndUnexpectedDeleteCodesKeepNativeFailure(t *testing.T) {
	for _, code := range []int{200, 201, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			body := `{"error":"attachment missing or rejected"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
				w.Header().Set("X-Proof", "rejected-delete")
				testcloud.JSON(w, code, body)
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput())
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+detachVolumeContractDeletePath || !reflect.DeepEqual(native.Expected, []int{202, 204}) || string(native.Body) != body || native.ResponseHeader.Get("X-Proof") != "rejected-delete" || errors.As(err, &accepted) || result == nil || result.Deleted != nil || result.LastAccepted != nil || result.Ready != nil || calls.Load() != 1 {
				t.Fatalf("result=%+v native=%+v error=%v calls=%d", result, native, err, calls.Load())
			}
			detachVolumeContractOperation(t, err)
		})
	}
}

func TestDetachVolumeContractsAvailableBeforeExactFailuresAndStatusOnlyReadiness(t *testing.T) {
	stopped := errors.New("stop on nonterminal status")
	for _, tc := range []struct {
		name          string
		states, polls []string
		callbackError error
		callbacks     int32
		failed        bool
	}{
		{name: "target before failure", states: []string{"available"}, polls: []string{"AVAILABLE"}},
		{name: "exact default failure", polls: []string{"ErRoR"}, failed: true},
		{name: "default failure not prefix", polls: []string{"error_deleting", "available"}, callbacks: 1},
		{name: "empty failures disable default", states: []string{}, polls: []string{"error", "available"}, callbacks: 1},
		{name: "other attachments still in use", polls: []string{"in-use"}, callbackError: stopped, callbacks: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				step := int(calls.Add(1))
				if step == 1 {
					detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
					w.Header().Set("X-Proof", "deleted")
					w.WriteHeader(204)
					return
				}
				detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
				index := step - 2
				if index >= len(tc.polls) {
					t.Error("extra poll or cleanup", r.URL)
					w.WriteHeader(500)
					return
				}
				testcloud.JSON(w, 200, detachVolumeContractObservation(tc.polls[index], ""))
			})
			interval := time.Millisecond
			result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{PollInterval: &interval, FailureStates: tc.states, ProgressCallback: func(progress int) error {
				callbacks.Add(1)
				if progress != 0 {
					t.Errorf("invented progress=%d", progress)
				}
				return tc.callbackError
			}}))
			detachVolumeContractDeleted(t, result, 204, nil)
			if calls.Load() != int32(1+len(tc.polls)) || callbacks.Load() != tc.callbacks || errors.Is(err, resource.ErrFailedState) != tc.failed || tc.callbackError != nil && !errors.Is(err, tc.callbackError) || !tc.failed && tc.callbackError == nil && (err != nil || result.Ready == nil) || (tc.failed || tc.callbackError != nil) && result.Ready != nil {
				t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
		})
	}
}

func TestDetachVolumeContractsAcceptedInvalidPollsAnd404PreserveAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
	}{
		{"missing identity", `{"volume":{"status":"available"}}`, 200},
		{"wrong canonical identity", `{"volume":{"ID":"vol-1","status":"available"}}`, 200},
		{"contradictory identity", `{"volume":{"id":"other","status":"available"}}`, 200},
		{"null status", `{"volume":{"id":"vol-1","status":null}}`, 200},
		{"null attachment row", detachVolumeContractObservation("available", `[null]`), 200},
		{"malformed JSON", `{"volume":`, 200},
		{"volume disappeared", `{"error":"volume absent"}`, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("X-Proof", "deleted")
					w.WriteHeader(204)
					return
				}
				detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
				w.Header().Set("X-Proof", "poll-failure")
				testcloud.JSON(w, tc.code, tc.body)
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput())
			detachVolumeContractDeleted(t, result, 204, nil)
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if err == nil || result.Ready != nil || calls.Load() != 2 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			if tc.code == 200 {
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != tc.body || result.LastAccepted == nil || string(result.LastAccepted.Body) != tc.body || result.LastAccepted.Header.Get("X-Proof") != "poll-failure" {
					t.Fatalf("accepted proof=%+v error=%v", result, err)
				}
			} else if !errors.As(err, &native) || native.Actual != 404 || errors.As(err, &accepted) || result.LastAccepted != nil {
				t.Fatalf("disappearance claimed success or borrowed proof: result=%+v error=%v", result, err)
			}
			detachVolumeContractOperation(t, err)
		})
	}
}

func TestDetachVolumeContractsWaitTimeoutCancellationAndLaterRejectionKeepLastAccepted(t *testing.T) {
	for _, mode := range []string{"timeout", "cancel", "later rejection"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			pollBody := detachVolumeContractObservation("detaching", `null`)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller stopped detach")
			callbackCause := errors.New("callback also stopped")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					w.Header().Set("X-Proof", "deleted")
					w.WriteHeader(204)
				case 2:
					w.Header().Set("X-Proof", "last-accepted")
					testcloud.JSON(w, 200, pollBody)
				case 3:
					if mode != "later rejection" {
						t.Error("wait cancellation issued extra HTTP", r.Method, r.URL)
					}
					w.Header().Set("X-Proof", "rejected")
					testcloud.JSON(w, 403, `{"error":"later denial"}`)
				default:
					t.Error("extra poll or cleanup", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			interval, timeout := time.Hour, 200*time.Millisecond
			policy := blockstorage.DetachVolumeWaitOpts{PollInterval: &interval, Timeout: &timeout}
			if mode == "cancel" {
				policy.Timeout = nil
				policy.ProgressCallback = func(int) error { cancel(cause); return callbackCause }
			}
			if mode == "later rejection" {
				interval = time.Millisecond
				policy.Timeout = nil
			}
			result, err := blockstorage.DetachVolume(ctx, nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(policy))
			detachVolumeContractDeleted(t, result, 204, nil)
			if result.LastAccepted == nil || string(result.LastAccepted.Body) != pollBody || result.LastAccepted.Header.Get("X-Proof") != "last-accepted" || result.Ready != nil {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if mode == "later rejection" {
				if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "rejected" || errors.As(err, &accepted) || calls.Load() != 3 {
					t.Fatalf("later native error masked: %v calls=%d", err, calls.Load())
				}
			} else {
				want := error(context.DeadlineExceeded)
				if mode == "cancel" {
					want = context.Canceled
				}
				if !errors.Is(err, want) || mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, callbackCause)) || !errors.As(err, &accepted) || string(accepted.Body) != pollBody || calls.Load() != 2 {
					t.Fatalf("accepted wait error=%v calls=%d", err, calls.Load())
				}
			}
		})
	}
}
