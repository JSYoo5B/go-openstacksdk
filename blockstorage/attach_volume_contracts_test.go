package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
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

const attachVolumeContractNovaBase = "/reverse/nova/v2.1/project/"
const attachVolumeContractCinderBase = "/reverse/cinder/v3/project/"
const attachVolumeContractCreatePath = attachVolumeContractNovaBase + "servers/server-1/os-volume_attachments"
const attachVolumeContractVolumePath = attachVolumeContractCinderBase + "volumes/vol-1"

type attachVolumeContractTransport func(*http.Request) (*http.Response, error)

func (f attachVolumeContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func attachVolumeContractClients(cloud *testcloud.Cloud) (*gophercloud.ServiceClient, *gophercloud.ServiceClient) {
	nova, cinder := cloud.Client("compute", "/unused/nova/"), cloud.Client("block-storage", "/unused/cinder/")
	nova.ResourceBase, cinder.ResourceBase = cloud.Server.URL+attachVolumeContractNovaBase, cloud.Server.URL+attachVolumeContractCinderBase
	nova.Microversion, cinder.Microversion = "2.79", "3.60"
	nova.MoreHeaders, cinder.MoreHeaders = map[string]string{"x-source": "nova-entry"}, map[string]string{"x-source": "cinder-entry"}
	return nova, cinder
}
func attachVolumeContractInput() blockstorage.AttachVolumeRequest {
	return blockstorage.AttachVolumeRequest{Server: resource.ID("server-1"), Volume: resource.ID("vol-1")}
}
func attachVolumeContractObservation(status, attachments string) string {
	return fmt.Sprintf(`{"volume":{"id":"vol-1","name":"data","status":%q,"attachments":%s,"created_at":"literal service time","vendor":{"number":9007199254740993}}}`, status, attachments)
}
func attachVolumeContractCreated() string {
	return `{"volumeAttachment":{"id":"nova-response-id","volumeId":"vol-1","serverId":"server-1","device":"/dev/vdb","attachment_id":"cinder-response-id","bdm_uuid":"block-device-id","delete_on_termination":false,"vendor":9007199254740993}}`
}
func attachVolumeContractWire(t *testing.T, r *http.Request, method, path, token string) {
	t.Helper()
	source, version := "cinder-entry", "volume 3.60"
	if strings.HasPrefix(path, attachVolumeContractNovaBase) {
		source, version = "nova-entry", "compute 2.79"
	}
	if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != source || r.Header.Get("OpenStack-API-Version") != version || r.Header.Get("X-Auth-Token") != token {
		t.Errorf("request method=%s URL=%s headers=%v", r.Method, r.URL, r.Header)
	}
}
func attachVolumeContractPostBody(t *testing.T, r *http.Request, device string) {
	t.Helper()
	var envelope map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		t.Error(err)
		return
	}
	want := map[string]map[string]json.RawMessage{"volumeAttachment": {"volumeId": json.RawMessage(`"vol-1"`)}}
	if device != "" {
		encoded, _ := json.Marshal(device)
		want["volumeAttachment"]["device"] = encoded
	}
	if !reflect.DeepEqual(envelope, want) {
		t.Errorf("attachment body=%v want=%v", envelope, want)
	}
}
func attachVolumeContractResponse(status int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {proof}}, Body: body}
}
func attachVolumeContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "AttachVolume" || operation.Cause == nil {
		t.Fatalf("workflow context missing: %v", err)
	}
}
func attachVolumeContractCreatedProof(t *testing.T, result *blockstorage.AttachVolumeResult) {
	t.Helper()
	if result == nil || result.ServerID != "server-1" || result.VolumeID != "vol-1" || result.Checked == nil || result.Checked.Volume == nil || result.Created == nil || result.Created.StatusCode != 200 || result.Created.Attachment == nil || result.Created.Attachment.VolumeID == nil || *result.Created.Attachment.VolumeID != "vol-1" || string(result.Created.Body) != attachVolumeContractCreated() || result.Created.Header.Get("X-Proof") != "created" {
		t.Fatalf("completed creation proof missing: %+v", result)
	}
}

func TestAttachVolumeContractsDefaultSequenceAndIndependentProof(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := attachVolumeContractClients(cloud)
	checked, ready, created := attachVolumeContractObservation("available", `[]`), attachVolumeContractObservation("IN-USE", `null`), attachVolumeContractCreated()
	var steps atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		step := steps.Add(1)
		switch step {
		case 1:
			attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
			w.Header().Set("X-Proof", "checked")
			testcloud.JSON(w, 200, checked)
		case 2:
			attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
			attachVolumeContractPostBody(t, r, "")
			w.Header().Set("X-Proof", "created")
			w.Header().Set("Location", "https://foreign.invalid/not-followed")
			testcloud.JSON(w, 200, created)
		case 3:
			attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
			w.Header().Set("X-Proof", "ready")
			testcloud.JSON(w, 200, ready)
		default:
			t.Error("unexpected workflow phase", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput())
	if err != nil || steps.Load() != 3 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, steps.Load())
	}
	attachVolumeContractCreatedProof(t, result)
	if result.Checked.StatusCode != 200 || result.Checked.Header.Get("X-Proof") != "checked" || string(result.Checked.Body) != checked || result.LastAccepted == nil || result.LastAccepted.Volume == nil || result.LastAccepted.Header.Get("X-Proof") != "ready" || string(result.LastAccepted.Body) != ready || result.Ready == nil || result.Ready.Status == nil || *result.Ready.Status != "IN-USE" || result.Ready.Attachments != nil || result.Ready.CreatedAt == nil || *result.Ready.CreatedAt != "literal service time" || string(result.Ready.Body["vendor"]) != `{"number":9007199254740993}` {
		t.Fatalf("response ownership/status readiness=%+v", result)
	}
	if result.Ready == result.LastAccepted.Volume {
		t.Fatal("Ready aliases LastAccepted.Volume")
	}
	result.Created.Body[0] = '!'
	result.Created.Header.Set("X-Proof", "caller")
	result.Created.Attachment.Body["vendor"][0] = '!'
	result.Checked.Volume.Body["status"][0] = '!'
	*result.Ready.Status = "caller"
	result.Ready.Body["vendor"][0] = '!'
	result.Ready.Header.Set("X-Proof", "caller")
	if string(result.Checked.Body) != checked || string(result.LastAccepted.Body) != ready || result.LastAccepted.Header.Get("X-Proof") != "ready" || *result.LastAccepted.Volume.Status != "IN-USE" || string(result.LastAccepted.Volume.Body["vendor"]) != `{"number":9007199254740993}` || result.LastAccepted.Volume.Header.Get("X-Proof") != "ready" {
		t.Fatal("phase or model evidence aliases", result)
	}
}

func TestAttachVolumeContractsExplicitNoWaitAndLiteralDevice(t *testing.T) {
	for _, device := range []string{"", "/dev/disk/by-id/volume data"} {
		t.Run(device, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
				} else {
					attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
					attachVolumeContractPostBody(t, r, device)
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 200, attachVolumeContractCreated())
				}
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeDevice(device), blockstorage.WithAttachVolumeWait(false))
			attachVolumeContractCreatedProof(t, result)
			if err != nil || calls.Load() != 2 || result.LastAccepted != nil || result.Ready != nil {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestAttachVolumeContractsFreshPreconditionForbidsUnsafeMutation(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"already associated missing device", attachVolumeContractObservation("available", `[{"server_id":"server-1"}]`)},
		{"already associated empty device", attachVolumeContractObservation("available", `[{"server_id":"server-1","device":""}]`)},
		{"already associated populated device", attachVolumeContractObservation("available", `[{"server_id":"server-1","device":"/dev/vdb"}]`)},
		{"in use including multiattach", attachVolumeContractObservation("in-use", `[{"server_id":"other-server"}]`)},
		{"exact available", attachVolumeContractObservation("AVAILABLE", `[]`)},
		{"missing attachments", `{"volume":{"id":"vol-1","status":"available"}}`},
		{"null attachments", attachVolumeContractObservation("available", `null`)},
		{"null attachment row", attachVolumeContractObservation("available", `[null]`)},
		{"nonarray attachments", attachVolumeContractObservation("available", `{}`)},
		{"missing canonical identity", `{"volume":{"ID":"vol-1","status":"available","attachments":[]}}`},
		{"contradictory identity", `{"volume":{"id":"response-decoy","status":"available","attachments":[]}}`},
		{"missing status", `{"volume":{"id":"vol-1","attachments":[]}}`},
		{"typed status", `{"volume":{"id":"vol-1","status":false,"attachments":[]}}`},
		{"null volume", `{"volume":null}`},
		{"incomplete JSON", `{"volume":`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
				w.Header().Set("X-Proof", "checked-failure")
				testcloud.JSON(w, 200, tc.body)
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWait(false))
			var accepted *resource.ResponseError
			if err == nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != tc.body || result == nil || result.Checked == nil || result.Checked.StatusCode != 200 || string(result.Checked.Body) != tc.body || result.Checked.Header.Get("X-Proof") != "checked-failure" || result.Created != nil || result.LastAccepted != nil || result.Ready != nil || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			attachVolumeContractOperation(t, err)
		})
	}
}

func TestAttachVolumeContractsPreflightBeforeCallbacksOrHTTP(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, tc := range []struct {
		name                                        string
		change                                      func(*gophercloud.ServiceClient, *gophercloud.ServiceClient, *blockstorage.AttachVolumeRequest)
		option                                      blockstorage.AttachVolumeOption
		nilContext, nilNova, nilCinder, unsupported bool
		callbacks                                   int
	}{
		{name: "nil context", nilContext: true}, {name: "nil Nova", nilNova: true}, {name: "nil Cinder", nilCinder: true},
		{name: "missing provider", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) { n.ProviderClient = nil }},
		{name: "wrong service", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) { n.Type = "image" }, unsupported: true},
		{name: "base query", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			c.ResourceBase += "?unsafe=1"
		}},
		{name: "base fragment", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			n.ResourceBase += "#fragment"
		}},
		{name: "relative endpoint", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) { n.Endpoint = "/relative/" }},
		{name: "foreign base", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			c.ResourceBase = "https://foreign.invalid/v3/"
		}},
		{name: "missing reference", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.AttachVolumeRequest) { i.Server = resource.Ref{} }},
		{name: "unsafe explicit ID", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.AttachVolumeRequest) {
			i.Volume = resource.ID("../other")
		}},
		{name: "control in Name", change: func(_, _ *gophercloud.ServiceClient, i *blockstorage.AttachVolumeRequest) {
			i.Server = resource.Name("name\n")
		}},
		{name: "authorization override", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			n.MoreHeaders["Authorization"] = "Bearer other"
		}},
		{name: "cookie override", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			c.MoreHeaders["Cookie"] = "session=other"
		}},
		{name: "host override", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			n.MoreHeaders["Host"] = "foreign.invalid"
		}},
		{name: "framing override", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			n.MoreHeaders["Content-Length"] = "1"
		}},
		{name: "noncanonical duplicate headers", change: func(n, _ *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			n.MoreHeaders["X-Source"] = "duplicate"
		}},
		{name: "version override", change: func(_, c *gophercloud.ServiceClient, _ *blockstorage.AttachVolumeRequest) {
			c.MoreHeaders["OpenStack-API-Version"] = "volume 3.99"
		}},
		{name: "invalid device", option: blockstorage.WithAttachVolumeDevice("device\n"), callbacks: 1},
		{name: "inactive negative timeout", option: blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{Timeout: &negative}), callbacks: 1},
		{name: "inactive zero interval", option: blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{PollInterval: &zero}), callbacks: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			input := attachVolumeContractInput()
			if tc.change != nil {
				tc.change(nova, cinder, &input)
			}
			if tc.nilNova {
				nova = nil
			}
			if tc.nilCinder {
				cinder = nil
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid workflow reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			ctx := context.Background()
			if tc.nilContext {
				ctx = nil
			}
			options := []blockstorage.AttachVolumeOption{func(*blockstorage.AttachVolumeOpts) error { callbacks.Add(1); return nil }, blockstorage.WithAttachVolumeWait(false)}
			if tc.option != nil {
				options = append(options, tc.option)
			}
			result, err := blockstorage.AttachVolume(ctx, nova, cinder, input, options...)
			expected := resource.ErrInvalidOption
			if tc.unsupported {
				expected = resource.ErrUnsupported
			}
			if result != nil || !errors.Is(err, expected) || calls.Load() != 0 || callbacks.Load() != int32(tc.callbacks) {
				t.Fatalf("result=%+v error=%v HTTP=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
			attachVolumeContractOperation(t, err)
		})
	}
	t.Run("canceled caller cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		cause := errors.New("caller canceled before workflow")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		callbacks := 0
		result, err := blockstorage.AttachVolume(ctx, nova, cinder, attachVolumeContractInput(), func(*blockstorage.AttachVolumeOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || callbacks != 0 {
			t.Fatalf("result=%+v error=%v callbacks=%d", result, err, callbacks)
		}
	})
	t.Run("nil option", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), nil)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	})
}

func TestAttachVolumeContractsNamesResolveOnceBeforeFreshCheck(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := attachVolumeContractClients(cloud)
	name := "worker?selected#literal"
	var steps atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch steps.Add(1) {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != attachVolumeContractNovaBase+"servers/detail" || r.URL.Query().Get("name") != "^worker\\?selected#literal$" {
				t.Error("server Name binding", r.Method, r.URL)
			}
			testcloud.JSON(w, 200, fmt.Sprintf(`{"servers":[{"id":"server-1","name":%q}]}`, name))
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != attachVolumeContractCinderBase+"volumes/detail" || r.URL.Query().Get("name") != "data" {
				t.Error("volume Name binding", r.Method, r.URL)
			}
			testcloud.JSON(w, 200, `{"volumes":[{"id":"vol-1","name":"data","attachments":[]}]}`)
		case 3:
			attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
			testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[{"server_id":"other-server","device":"/dev/vdc"}]`))
		case 4:
			attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
			attachVolumeContractPostBody(t, r, "")
			w.Header().Set("X-Proof", "created")
			testcloud.JSON(w, 200, attachVolumeContractCreated())
		default:
			t.Error("Name resolved repeatedly or unrelated operation", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, blockstorage.AttachVolumeRequest{Server: resource.Name(name), Volume: resource.Name("data")}, blockstorage.WithAttachVolumeWait(false))
	attachVolumeContractCreatedProof(t, result)
	if err != nil || steps.Load() != 4 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, steps.Load())
	}
	for _, tc := range []struct {
		name, rows string
		cause      error
	}{
		{"missing", `[]`, resource.ErrNotFound},
		{"ambiguous", `[{"id":"one","name":"worker"},{"id":"two","name":"worker"}]`, resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != attachVolumeContractNovaBase+"servers/detail" {
					t.Error("unexpected phase", r.URL)
				}
				testcloud.JSON(w, 200, `{"servers":`+tc.rows+`}`)
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, blockstorage.AttachVolumeRequest{Server: resource.Name("worker"), Volume: resource.ID("vol-1")})
			if result != nil || !errors.Is(err, tc.cause) || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func TestAttachVolumeContractsCreatedProofSurvivesWaitFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body  string
		code        int
		failedState bool
	}{
		{"exact failure", attachVolumeContractObservation("ErRoR", `[]`), 200, true},
		{"malformed JSON", `{"volume":`, 200, false},
		{"null envelope", `{"volume":null}`, 200, false},
		{"missing status", `{"volume":{"id":"vol-1"}}`, 200, false},
		{"wrong identity", `{"volume":{"id":"decoy","status":"in-use"}}`, 200, false},
		{"native not found", `{"error":"missing volume"}`, 404, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
				case 2:
					attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 200, attachVolumeContractCreated())
				case 3:
					attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
					w.Header().Set("X-Proof", "failed-poll")
					testcloud.JSON(w, tc.code, tc.body)
				default:
					t.Error("failure caused retry or cleanup", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput())
			attachVolumeContractCreatedProof(t, result)
			if err == nil || result.Ready != nil || calls.Load() != 3 || errors.Is(err, resource.ErrFailedState) != tc.failedState {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			attachVolumeContractOperation(t, err)
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if tc.code == 200 {
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != tc.body || result.LastAccepted == nil || string(result.LastAccepted.Body) != tc.body || result.LastAccepted.Header.Get("X-Proof") != "failed-poll" {
					t.Fatalf("accepted proof=%+v error=%v", result, err)
				}
			} else if result.LastAccepted != nil || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != tc.code || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Proof") != "failed-poll" {
				t.Fatalf("rejected poll masked: result=%+v error=%v", result, err)
			}
		})
	}
}

func TestAttachVolumeContractsExactWaitStatesTargetFirstAndCallback(t *testing.T) {
	for _, tc := range []struct {
		name          string
		states        []string
		polls         []string
		callbackError error
		callbacks     int
	}{
		{name: "target before configured failure", states: []string{"in-use"}, polls: []string{"IN-USE"}},
		{name: "empty failures disable default", states: []string{}, polls: []string{"error", "in-use"}, callbacks: 1},
		{name: "default exact failure not prefix", polls: []string{"error_extending", "in-use"}, callbacks: 1},
		{name: "progress error stops without cleanup", polls: []string{"attaching"}, callbackError: errors.New("progress callback stopped"), callbacks: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				count := int(calls.Add(1))
				if count == 1 {
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
					return
				}
				if count == 2 {
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 200, attachVolumeContractCreated())
					return
				}
				attachVolumeContractWire(t, r, http.MethodGet, attachVolumeContractVolumePath, "test-token")
				index := count - 3
				if index >= len(tc.polls) {
					t.Error("unexpected extra poll or cleanup", r.Method, r.URL)
					w.WriteHeader(500)
					return
				}
				testcloud.JSON(w, 200, attachVolumeContractObservation(tc.polls[index], `null`))
			})
			interval := time.Millisecond
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{PollInterval: &interval, FailureStates: tc.states, ProgressCallback: func(progress int) error {
				callbacks.Add(1)
				if progress != 0 {
					t.Errorf("invented Cinder progress=%d", progress)
				}
				return tc.callbackError
			}}))
			attachVolumeContractCreatedProof(t, result)
			if calls.Load() != int32(2+len(tc.polls)) || callbacks.Load() != int32(tc.callbacks) || errors.Is(err, resource.ErrFailedState) || tc.callbackError == nil && (err != nil || result.Ready == nil) || tc.callbackError != nil && (!errors.Is(err, tc.callbackError) || result.Ready != nil) {
				t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
			}
		})
	}
}

func TestAttachVolumeContractsWaitTimeoutAndCancellationKeepLastAccepted(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller stopped after accepted poll")
			var calls atomic.Int32
			pollBody := attachVolumeContractObservation("attaching", `[]`)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
				case 2:
					w.Header().Set("X-Proof", "created")
					testcloud.JSON(w, 200, attachVolumeContractCreated())
				case 3:
					w.Header().Set("X-Proof", "last-accepted")
					testcloud.JSON(w, 200, pollBody)
				default:
					t.Error("cancellation issued more requests", r.Method, r.URL)
					w.WriteHeader(500)
				}
			})
			interval, timeout := time.Hour, 200*time.Millisecond
			policy := blockstorage.AttachVolumeWaitOpts{PollInterval: &interval, Timeout: &timeout}
			want := error(context.DeadlineExceeded)
			if canceled {
				policy.Timeout = nil
				policy.ProgressCallback = func(int) error { cancel(cause); return nil }
				want = context.Canceled
			}
			result, err := blockstorage.AttachVolume(ctx, nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(policy))
			attachVolumeContractCreatedProof(t, result)
			var accepted *resource.ResponseError
			if !errors.Is(err, want) || canceled && !errors.Is(err, cause) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != pollBody || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != pollBody || result.LastAccepted.Header.Get("X-Proof") != "last-accepted" || result.Ready != nil || calls.Load() != 3 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
	t.Run("rejected later poll does not borrow older HTTP proof", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := attachVolumeContractClients(cloud)
		var calls atomic.Int32
		pollBody := attachVolumeContractObservation("attaching", `[]`)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
			case 2:
				w.Header().Set("X-Proof", "created")
				testcloud.JSON(w, 200, attachVolumeContractCreated())
			case 3:
				w.Header().Set("X-Proof", "last-accepted")
				testcloud.JSON(w, 200, pollBody)
			case 4:
				w.Header().Set("X-Proof", "rejected")
				testcloud.JSON(w, 403, `{"error":"later denial"}`)
			default:
				t.Error("unexpected later poll", r.URL)
				w.WriteHeader(500)
			}
		})
		interval := time.Millisecond
		result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput(), blockstorage.WithAttachVolumeWaitPolicy(blockstorage.AttachVolumeWaitOpts{PollInterval: &interval}))
		attachVolumeContractCreatedProof(t, result)
		var native gophercloud.ErrUnexpectedResponseCode
		var accepted *resource.ResponseError
		if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Proof") != "rejected" || errors.As(err, &accepted) || result.LastAccepted == nil || string(result.LastAccepted.Body) != pollBody || result.Ready != nil || calls.Load() != 4 {
			t.Fatalf("result=%+v native=%+v error=%v calls=%d", result, native, err, calls.Load())
		}
	})
}

func TestAttachVolumeContractsNovaStatusAndCreatedDecodeFailuresAreTerminal(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"native conflict", `{"error":"already attached"}`, 409},
		{"unexpected created code", attachVolumeContractCreated(), 201},
		{"unexpected accepted code", attachVolumeContractCreated(), 202},
		{"malformed accepted JSON", `{"volumeAttachment":`, 200},
		{"missing canonical envelope", `{"VolumeAttachment":{"volumeId":"vol-1"}}`, 200},
		{"null envelope", `{"volumeAttachment":null}`, 200},
		{"wrong canonical field type", `{"volumeAttachment":{"volumeId":"vol-1","device":false}}`, 200},
		{"missing canonical volume", `{"volumeAttachment":{"VolumeId":"vol-1"}}`, 200},
		{"wrong volume", `{"volumeAttachment":{"volumeId":"other"}}`, 200},
		{"wrong optional server", `{"volumeAttachment":{"volumeId":"vol-1","serverId":"other"}}`, 200},
		{"invalid UTF8", "{\"volumeAttachment\":{\"volumeId\":\"vol-1\",\"device\":\"\xff\"}}", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := attachVolumeContractClients(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 200, attachVolumeContractObservation("available", `[]`))
					return
				}
				attachVolumeContractWire(t, r, http.MethodPost, attachVolumeContractCreatePath, "test-token")
				w.Header().Set("X-Proof", "creation-failure")
				testcloud.JSON(w, tc.status, tc.body)
			})
			result, err := blockstorage.AttachVolume(context.Background(), nova, cinder, attachVolumeContractInput())
			var accepted *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || result == nil || result.Checked == nil || result.LastAccepted != nil || result.Ready != nil || calls.Load() != 2 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			attachVolumeContractOperation(t, err)
			if tc.status == 200 {
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || !bytes.Equal(accepted.Body, []byte(tc.body)) || result.Created == nil || result.Created.Attachment != nil || !bytes.Equal(result.Created.Body, []byte(tc.body)) || result.Created.Header.Get("X-Proof") != "creation-failure" || retries.Load() != 0 {
					t.Fatalf("accepted failure proof=%+v error=%v retries=%d", result, err, retries.Load())
				}
			} else if result.Created != nil || errors.As(err, &accepted) || !errors.As(err, &native) || native.Actual != tc.status || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Proof") != "creation-failure" || retries.Load() != 1 {
				t.Fatalf("native mutation failure=%+v error=%v retries=%d", result, err, retries.Load())
			}
		})
	}
}
