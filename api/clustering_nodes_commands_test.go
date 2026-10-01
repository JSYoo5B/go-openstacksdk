package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func nodeCommandCall(ctx context.Context, api *nodes.API, command string, ref resource.Ref) (*actions.Submission, error) {
	switch command {
	case "Check":
		return api.Check(ctx, ref)
	case "Recover":
		return api.Recover(ctx, ref, nodes.RecoverOpts{})
	default:
		return api.PerformOperation(ctx, ref, "reboot")
	}
}

func TestClusteringNodeCommandsBaseDefaultsAndSubmissionOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	response := `{"action":"command-action","number":9007199254740993,"unset":null}`
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/controller-name/actions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		call := calls.Add(1)
		expected := `{"check":{}}`
		if call == 2 {
			expected = `{"recover":{}}`
		}
		if string(body) != expected || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(string(body), r.URL, r.Header)
		}
		w.Header().Set("Location", "/reverse/senlin/v1/actions/command-action")
		w.Header().Set("X-Request-ID", "command-request")
		testcloud.JSON(w, 202, response)
	})
	api := nodes.New(cloud.Client("clustering", "/reverse/senlin/v1"))
	var previous *actions.Submission
	for _, command := range []string{"Check", "Recover", "Check"} {
		value, err := nodeCommandCall(context.Background(), api, command, resource.ID("controller-name"))
		if err != nil || value == nil || value.ActionID != "command-action" || value.Location != "/reverse/senlin/v1/actions/command-action" || string(value.Body) != response || value.StatusCode != 202 || value.Header.Get("X-Request-ID") != "command-request" {
			t.Fatal(value, err)
		}
		if previous != nil && (string(previous.Body) == response || previous.Header.Get("X-Request-ID") != "consumer") {
			t.Fatal("submission evidence is shared across requests")
		}
		value.Body[2] = 'X'
		value.Header.Set("X-Request-ID", "consumer")
		previous = value
	}
	if calls.Load() != 3 {
		t.Fatal("command looked up an explicit ID or fetched an action", calls.Load())
	}
}

func TestClusteringNodeCommandsRecoverNullableSnapshotAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.6"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/actions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		expected := `{"recover":{"check":false,"operation":"reboot","operation_params":{"big":9007199254740993,"type":"soft"},"vendor":{"off":false,"zero":0}}}`
		if calls.Add(1) == 3 {
			expected = `{"recover":{"check":null,"operation_params":null}}`
		}
		if string(body) != expected || r.Header.Get("OpenStack-API-Version") != "clustering 1.6" || r.Header.Get("X-Vendor") != "retained" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/recover-action")
		testcloud.JSON(w, 202, `{"action":"recover-action"}`)
	})
	raw := json.RawMessage(`{"big":9007199254740993,"type":"soft"}`)
	options := nodes.WithRecoverOptions(nodes.RecoverOpts{OperationParams: raw, Check: request.Present(false)})
	raw[2] = 'X'
	vendor := map[string]any{"zero": 0, "off": false}
	extension := nodes.WithRecoverField("vendor", vendor)
	vendor["off"] = true
	api := nodes.New(client)
	for range 2 {
		value, err := api.Recover(context.Background(), resource.ID("node-id"), nodes.RecoverOpts{}, options, nodes.WithRecoverOperation("reboot"), extension, nodes.WithRecoverHeader("X-Vendor", "retained"))
		if err != nil || value.ActionID != "recover-action" {
			t.Fatal(value, err)
		}
	}
	if _, err := api.Recover(context.Background(), resource.ID("node-id"), nodes.RecoverOpts{}, nodes.WithRecoverOperationParams(nil), nodes.WithRecoverCheckNull(), nodes.WithRecoverHeader("X-Vendor", "retained")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeCommandsCheckParameterExtensionsSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/actions", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"check":{"id":"plugin-id","params":{"big":9007199254740993,"off":false},"unset":null}}` || r.Header.Get("X-Vendor") != "check-header" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/check-action")
		testcloud.JSON(w, 202, `{"action":"check-action"}`)
	})
	raw := json.RawMessage(`{"big":9007199254740993,"off":false}`)
	option := nodes.WithCheckField("params", raw)
	raw[2] = 'X'
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	for range 2 {
		value, err := api.Check(context.Background(), resource.ID("node-id"), option, nodes.WithCheckField("id", "plugin-id"), nodes.WithCheckField("unset", nil), nodes.WithCheckHeader("X-Vendor", "check-header"))
		if err != nil || value.ActionID != "check-action" {
			t.Fatal(value, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeCommandsRecoverOperationOmittedEmptyAndNull(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	expected := []string{`{"recover":{}}`, `{"recover":{"operation":""}}`, `{"recover":{"operation":null}}`, `{"recover":{"operation":""}}`, `{"recover":{"operation":null}}`}
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/actions", func(w http.ResponseWriter, r *http.Request) {
		index := int(calls.Add(1)) - 1
		body, _ := io.ReadAll(r.Body)
		if index >= len(expected) || string(body) != expected[index] || r.Header.Get("OpenStack-API-Version") != "" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/presence-action")
		testcloud.JSON(w, 202, `{"action":"presence-action"}`)
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	for _, options := range [][]nodes.RecoverOption{
		nil,
		{nodes.WithRecoverOperation("")},
		{nodes.WithRecoverOperationNull()},
		{nodes.WithRecoverOptions(nodes.RecoverOpts{Operation: request.Present("")})},
		{nodes.WithRecoverOptions(nodes.RecoverOpts{Operation: request.Null[string]()})},
	} {
		value, err := api.Recover(context.Background(), resource.ID("node-id"), nodes.RecoverOpts{}, options...)
		if err != nil || value.ActionID != "presence-action" {
			t.Fatal(value, err)
		}
	}
	if calls.Load() != int32(len(expected)) {
		t.Fatal("operation presence changed command count", calls.Load())
	}
}

func TestClusteringNodeCommandsPluginParametersDoNotReplaceTargetOrEnvelope(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.4"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/ops", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		expected := `{"dance":{"action":null,"id":"plugin-identity","location":"plugin-location","params":{"big":9007199254740993,"style":"tango"},"status":false}}`
		if calls.Add(1) == 3 {
			expected = `{"reboot":{}}`
		}
		if string(body) != expected || r.Header.Get("OpenStack-API-Version") != "clustering 1.4" || r.Header.Get("X-Vendor") != "operation-header" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/plugin-action")
		testcloud.JSON(w, 202, `{"action":"plugin-action"}`)
	})
	params := json.RawMessage(`{"big":9007199254740993,"style":"tango"}`)
	option := nodes.WithPerformOperationField("params", params)
	params[2] = 'X'
	api := nodes.New(client)
	for range 2 {
		value, err := api.PerformOperation(context.Background(), resource.ID("node-id"), "dance", option, nodes.WithPerformOperationField("id", "plugin-identity"), nodes.WithPerformOperationField("status", false), nodes.WithPerformOperationField("action", nil), nodes.WithPerformOperationField("location", "plugin-location"), nodes.WithPerformOperationHeader("X-Vendor", "operation-header"))
		if err != nil || value.ActionID != "plugin-action" {
			t.Fatal(value, err)
		}
	}
	if _, err := api.PerformOperation(context.Background(), resource.ID("node-id"), "reboot", nodes.WithPerformOperationHeader("X-Vendor", "operation-header")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeCommandsNameLookupFreezesBodyHeadersAndGate(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.6"
	var reads, posts atomic.Int32
	var captured *request.Config[nodes.RecoverOpts]
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.URL.Query().Get("name") != "selected" {
			t.Error(r.URL)
		}
		captured.Options.Operation = request.Present("mutated")
		captured.Options.OperationParams[2] = 'X'
		captured.Options.Check = request.Optional[bool]{}
		captured.Fields["id"] = json.RawMessage(`"mutated"`)
		captured.Headers["X-Vendor"] = "mutated"
		testcloud.JSON(w, 200, `{"nodes":[{"id":"wrong","name":"other"},{"id":"raw-node-id","name":"selected","physical_id":"physical-id"}]}`)
	})
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/raw-node-id/actions", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"recover":{"check":false,"id":"plugin-id","operation":"reboot","operation_params":{"type":"soft"}}}` || r.Header.Get("X-Vendor") != "original" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/frozen-action")
		testcloud.JSON(w, 202, `{"action":"frozen-action"}`)
	})
	capture := func(config *request.Config[nodes.RecoverOpts]) error { captured = config; return nil }
	value, err := nodes.New(client).Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{OperationParams: json.RawMessage(`{"type":"soft"}`), Check: request.Present(false)}, nodes.WithRecoverOperation("reboot"), nodes.WithRecoverField("id", "plugin-id"), nodes.WithRecoverHeader("X-Vendor", "original"), capture)
	if err != nil || value.ActionID != "frozen-action" || reads.Load() != 1 || posts.Load() != 1 {
		t.Fatal(value, err, reads.Load(), posts.Load())
	}
}

func TestClusteringNodeCommandsVersionPreflightAndRecheck(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("unsupported command reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := nodes.New(client)
	for _, version := range []string{"", "1.5", "latest"} {
		client.Microversion = version
		for _, check := range []request.Optional[bool]{request.Present(false), request.Present(true), request.Null[bool]()} {
			if _, err := api.Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{Check: check}); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, err)
			}
		}
	}
	for _, version := range []string{"", "1.3", "latest"} {
		client.Microversion = version
		if _, err := api.PerformOperation(context.Background(), resource.Name("selected"), "reboot"); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	for _, command := range []string{"Check", "Recover", "PerformOperation"} {
		t.Run(command+"AfterLookup", func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.6"
			var reads, posts atomic.Int32
			var captured *request.Config[nodes.RecoverOpts]
			cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if command == "Check" {
					client.Type = "compute"
				} else if command == "Recover" {
					client.Microversion = "1.5"
					captured.Options.Check = request.Optional[bool]{}
				} else {
					client.Microversion = "1.3"
				}
				testcloud.JSON(w, 200, `{"nodes":[{"id":"node-id","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/{path}", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				t.Error("changed source reached command POST")
			})
			api := nodes.New(client)
			var err error
			if command == "Recover" {
				capture := func(config *request.Config[nodes.RecoverOpts]) error { captured = config; return nil }
				_, err = api.Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{Check: request.Present(false)}, capture)
			} else {
				_, err = nodeCommandCall(context.Background(), api, command, resource.Name("selected"))
			}
			want := resource.ErrUnsupported
			if command == "Check" {
				want = resource.ErrInvalidOption
			}
			if !errors.Is(err, want) || reads.Load() != 1 || posts.Load() != 0 {
				t.Fatal(err, reads.Load(), posts.Load())
			}
		})
	}
}

func TestClusteringNodeCommandsPreflightInvalidInput(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid command reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.6"
	api := nodes.New(client)
	for _, raw := range []json.RawMessage{json.RawMessage(`[]`), json.RawMessage(`false`), json.RawMessage(`{`)} {
		if _, err := api.Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{OperationParams: raw}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(string(raw), err)
		}
	}
	for _, key := range []string{"operation", "operation_params", "check"} {
		if _, err := api.Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{}, nodes.WithRecoverField(key, nil)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	for _, option := range []nodes.CheckOption{nil, nodes.WithCheckHeader("X-Auth-Token", "foreign"), nodes.WithCheckHeader("openstack-api-version", "clustering 1.6"), request.WithQuery[nodes.CheckOpts]("vendor", "unsupported"), request.WithArgument[nodes.CheckOpts]("vendor", false), nodes.WithCheckField("vendor", json.RawMessage(`{`))} {
		if _, err := api.Check(context.Background(), resource.Name("selected"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.PerformOperation(context.Background(), resource.ID("node-id"), "  "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Check(context.Background(), resource.ID("bad/id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, invalid := range []*nodes.API{nil, nodes.New(nil), nodes.New(&gophercloud.ServiceClient{})} {
		if _, err := invalid.Check(context.Background(), resource.ID("node-id")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, command := range []string{"Check", "Recover", "PerformOperation"} {
		if _, err := nodeCommandCall(ctx, api, command, resource.ID("node-id")); !errors.Is(err, context.Canceled) {
			t.Fatal(command, err)
		}
	}
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.7"}
	if _, err := api.Check(context.Background(), resource.Name("selected")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.MoreHeaders = nil
	mutate := func(config *request.Config[nodes.RecoverOpts]) error { client.Type = "compute"; return nil }
	if _, err := api.Recover(context.Background(), resource.Name("selected"), nodes.RecoverOpts{}, mutate); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeCommandsMalformedAcceptedResponsesRetainEvidence(t *testing.T) {
	fixtures := []struct{ name, body, location string }{
		{"missing-location", `{"action":"accepted"}`, ""},
		{"mismatch", `{"action":"accepted"}`, "actions/other"},
		{"missing-action", `{}`, "actions/accepted"},
		{"null-action", `{"action":null}`, "actions/accepted"},
		{"object-action", `{"action":{"id":"accepted"}}`, "actions/accepted"},
		{"malformed", `{`, "actions/accepted"},
		{"empty", "", "actions/accepted"},
		{"foreign", `{"action":"accepted"}`, "https://foreign.example/actions/accepted"},
		{"wrong-collection", `{"action":"accepted"}`, "/senlin/v1/nodes/accepted"},
		{"query", `{"action":"accepted"}`, "actions/accepted?force=true"},
	}
	for _, command := range []string{"Check", "Recover", "PerformOperation"} {
		for _, fixture := range fixtures {
			t.Run(command+"/"+fixture.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/{path}", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if fixture.location != "" {
						w.Header().Set("Location", fixture.location)
					}
					w.Header().Set("X-Request-ID", "accepted-request")
					testcloud.JSON(w, 202, fixture.body)
				})
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.6"
				value, err := nodeCommandCall(context.Background(), nodes.New(client), command, resource.ID("node-id"))
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != fixture.body || evidence.Header.Get("X-Request-ID") != "accepted-request" || calls.Load() != 1 {
					t.Fatal(value, err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringNodeCommandsHTTPFailuresRemainOriginalErrors(t *testing.T) {
	for _, command := range []string{"Check", "Recover", "PerformOperation"} {
		for _, code := range []int{200, 201, 404, 409, 503} {
			t.Run(command+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				body := `{"error":{"message":"original-error"}}`
				cloud.Mux.HandleFunc("POST /senlin/v1/nodes/node-id/{path}", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "failed-request")
					testcloud.JSON(w, code, body)
				})
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.6"
				value, err := nodeCommandCall(context.Background(), nodes.New(client), command, resource.ID("node-id"))
				var evidence *resource.ResponseError
				var original gophercloud.ErrUnexpectedResponseCode
				if value != nil || !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &original) || errors.As(err, &evidence) || string(original.Body) != body || original.ResponseHeader.Get("X-Request-ID") != "failed-request" || calls.Load() != 1 {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringNodeCommandsLookupMissingAndAmbiguousNeverSubmit(t *testing.T) {
	cloud := testcloud.New(t)
	var reads, posts atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		body := `{"nodes":[]}`
		if r.URL.Query().Get("name") == "duplicate" {
			body = `{"nodes":[{"id":"first","name":"duplicate"},{"id":"second","name":"duplicate"}]}`
		}
		testcloud.JSON(w, 200, body)
	})
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/{id}/{path}", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); t.Error("unresolved name submitted") })
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.6"
	for _, command := range []string{"Check", "Recover", "PerformOperation"} {
		for _, name := range []string{"missing", "duplicate"} {
			_, err := nodeCommandCall(context.Background(), nodes.New(client), command, resource.Name(name))
			want := resource.ErrNotFound
			if name == "duplicate" {
				want = resource.ErrAmbiguous
			}
			if !errors.Is(err, want) || !strings.Contains(err.Error(), command) {
				t.Fatal(command, name, err)
			}
		}
	}
	if reads.Load() != 6 || posts.Load() != 0 {
		t.Fatal(reads.Load(), posts.Load())
	}
}
