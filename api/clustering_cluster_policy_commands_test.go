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

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/actions"
	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/clusters"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var clusterPolicyCommands = []string{"AttachPolicy", "DetachPolicy", "UpdatePolicy"}

func clusterPolicyCommandCall(ctx context.Context, api *clusters.API, command string, ref resource.Ref, policyID string) (*actions.Submission, error) {
	switch command {
	case "AttachPolicy":
		return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{PolicyID: policyID})
	case "DetachPolicy":
		return api.DetachPolicy(ctx, ref, clusters.DetachPolicyOpts{PolicyID: policyID})
	default:
		return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{PolicyID: policyID})
	}
}

func TestClusteringClusterPolicyCommandsWireDefaultsPresenceAndSubmission(t *testing.T) {
	ctx := context.Background()
	ref := resource.ID("controller-name")
	policyID := "policy/name ?#%"
	plugin := map[string]any{"exact": json.Number("9007199254740993"), "off": false, "zero": 0}
	attachPlugin := clusters.WithAttachPolicyField("plugin", plugin)
	updatePlugin := clusters.WithUpdatePolicyField("plugin", plugin)
	detachPlugin := clusters.WithDetachPolicyField("plugin", plugin)
	plugin["exact"], plugin["off"] = 0, true
	tests := []struct {
		name, body string
		call       func(*clusters.API) (*actions.Submission, error)
	}{
		{"attach-default", `{"policy_attach":{"policy_id":"policy/name ?#%"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{PolicyID: policyID})
		}},
		{"detach-default", `{"policy_detach":{"policy_id":"short-id"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.DetachPolicy(ctx, ref, clusters.DetachPolicyOpts{PolicyID: "short-id"})
		}},
		{"update-default", `{"policy_update":{"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{PolicyID: "named-policy"})
		}},
		{"attach-false", `{"policy_attach":{"enabled":false,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{PolicyID: "named-policy"}, clusters.WithAttachPolicyEnabled(false))
		}},
		{"attach-true", `{"policy_attach":{"enabled":true,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{PolicyID: "named-policy", Enabled: request.Present(true)})
		}},
		{"attach-null", `{"policy_attach":{"enabled":null,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{PolicyID: "named-policy", Enabled: request.Present(true)}, clusters.WithAttachPolicyEnabledNull())
		}},
		{"attach-snapshot-false", `{"policy_attach":{"enabled":false,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{}, clusters.WithAttachPolicyOptions(clusters.AttachPolicyOpts{PolicyID: "named-policy", Enabled: request.Present(false)}))
		}},
		{"update-false", `{"policy_update":{"enabled":false,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{PolicyID: "named-policy"}, clusters.WithUpdatePolicyEnabled(false))
		}},
		{"update-true", `{"policy_update":{"enabled":true,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{PolicyID: "named-policy"}, clusters.WithUpdatePolicyEnabled(true))
		}},
		{"update-null", `{"policy_update":{"enabled":null,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{PolicyID: "named-policy", Enabled: request.Present(false)}, clusters.WithUpdatePolicyEnabledNull())
		}},
		{"update-snapshot-null", `{"policy_update":{"enabled":null,"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{}, clusters.WithUpdatePolicyOptions(clusters.UpdatePolicyOpts{PolicyID: "named-policy", Enabled: request.Null[bool]()}))
		}},
		{"attach-plugin", `{"policy_attach":{"action":null,"id":"plugin-id","location":"plugin-location","plugin":{"exact":9007199254740993,"off":false,"zero":0},"policy_id":"named-policy","status":false}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AttachPolicy(ctx, ref, clusters.AttachPolicyOpts{}, clusters.WithAttachPolicyOptions(clusters.AttachPolicyOpts{PolicyID: "named-policy"}), attachPlugin, clusters.WithAttachPolicyField("id", "plugin-id"), clusters.WithAttachPolicyField("action", nil), clusters.WithAttachPolicyField("location", "plugin-location"), clusters.WithAttachPolicyField("status", false))
		}},
		{"update-plugin", `{"policy_update":{"plugin":{"exact":9007199254740993,"off":false,"zero":0},"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.UpdatePolicy(ctx, ref, clusters.UpdatePolicyOpts{}, clusters.WithUpdatePolicyOptions(clusters.UpdatePolicyOpts{PolicyID: "named-policy"}), updatePlugin)
		}},
		{"detach-go-extension", `{"policy_detach":{"plugin":{"exact":9007199254740993,"off":false,"zero":0},"policy_id":"named-policy"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.DetachPolicy(ctx, ref, clusters.DetachPolicyOpts{}, clusters.WithDetachPolicyOptions(clusters.DetachPolicyOpts{PolicyID: "named-policy"}), detachPlugin)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			response := `{"action":"policy-action","extension":{"exact":9007199254740993,"off":false}}`
			cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/controller-name/actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != test.body || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(string(body), r.URL, r.Header)
				}
				w.Header().Set("Location", "/reverse/senlin/v1/actions/policy-action")
				w.Header().Set("X-Request-ID", "policy-command-evidence")
				testcloud.JSON(w, 202, response)
			})
			api := clusters.New(cloud.Client("clustering", "/reverse/senlin/v1"))
			var previous *actions.Submission
			for range 2 {
				value, err := test.call(api)
				if err != nil || value == nil || value.ActionID != "policy-action" || value.Location != "/reverse/senlin/v1/actions/policy-action" || value.StatusCode != 202 || string(value.Body) != response || value.Header.Get("X-Request-ID") != "policy-command-evidence" {
					t.Fatal(value, err)
				}
				if previous != nil && (string(previous.Body) == response || previous.Header.Get("X-Request-ID") != "consumer-change") {
					t.Fatal("submission evidence shares mutable storage")
				}
				value.Body[2] = 'X'
				value.Header.Set("X-Request-ID", "consumer-change")
				previous = value
			}
			if calls.Load() != 2 {
				t.Fatal("command fetched a policy, followed Location or resent", calls.Load())
			}
		})
	}
}

func TestClusteringClusterPolicyCommandsNameLookupFreezesBodyAndHeaders(t *testing.T) {
	for _, command := range clusterPolicyCommands {
		t.Run(command, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, posts atomic.Int32
			var mutate func()
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "selected" {
					t.Error(r.URL)
				}
				mutate()
				testcloud.JSON(w, 200, `{"clusters":[{"id":"wrong","name":"other"},{"id":"resolved-cluster","name":"selected"}]}`)
			})
			key, enabled := "policy_attach", `"enabled":false,`
			if command == "UpdatePolicy" {
				key = "policy_update"
			} else if command == "DetachPolicy" {
				key, enabled = "policy_detach", ""
			}
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/resolved-cluster/actions", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"`+key+`":{`+enabled+`"policy_id":"original-policy","vendor":{"exact":9007199254740993}}}` || r.Header.Get("X-Vendor") != "original-header" {
					t.Error(string(body), r.Header)
				}
				w.Header().Set("Location", "actions/frozen-action")
				testcloud.JSON(w, 202, `{"action":"frozen-action"}`)
			})
			api := clusters.New(cloud.Client("clustering", "/senlin/v1"))
			var value *actions.Submission
			var err error
			switch command {
			case "AttachPolicy":
				value, err = api.AttachPolicy(context.Background(), resource.Name("selected"), clusters.AttachPolicyOpts{PolicyID: "original-policy", Enabled: request.Present(false)}, clusters.WithAttachPolicyField("vendor", json.RawMessage(`{"exact":9007199254740993}`)), clusters.WithAttachPolicyHeader("X-Vendor", "original-header"), func(config *request.Config[clusters.AttachPolicyOpts]) error {
					mutate = func() {
						config.Options.PolicyID = "changed-policy"
						config.Options.Enabled = request.Present(true)
						config.Fields["vendor"][2] = 'X'
						config.Headers["X-Vendor"] = "changed-header"
					}
					return nil
				})
			case "DetachPolicy":
				value, err = api.DetachPolicy(context.Background(), resource.Name("selected"), clusters.DetachPolicyOpts{PolicyID: "original-policy"}, clusters.WithDetachPolicyField("vendor", json.RawMessage(`{"exact":9007199254740993}`)), clusters.WithDetachPolicyHeader("X-Vendor", "original-header"), func(config *request.Config[clusters.DetachPolicyOpts]) error {
					mutate = func() {
						config.Options.PolicyID = "changed-policy"
						config.Fields["vendor"][2] = 'X'
						config.Headers["X-Vendor"] = "changed-header"
					}
					return nil
				})
			default:
				value, err = api.UpdatePolicy(context.Background(), resource.Name("selected"), clusters.UpdatePolicyOpts{PolicyID: "original-policy", Enabled: request.Present(false)}, clusters.WithUpdatePolicyField("vendor", json.RawMessage(`{"exact":9007199254740993}`)), clusters.WithUpdatePolicyHeader("X-Vendor", "original-header"), func(config *request.Config[clusters.UpdatePolicyOpts]) error {
					mutate = func() {
						config.Options.PolicyID = "changed-policy"
						config.Options.Enabled = request.Present(true)
						config.Fields["vendor"][2] = 'X'
						config.Headers["X-Vendor"] = "changed-header"
					}
					return nil
				})
			}
			if err != nil || value == nil || value.ActionID != "frozen-action" || lists.Load() != 1 || posts.Load() != 1 {
				t.Fatal(value, err, lists.Load(), posts.Load())
			}
		})
	}
}

func clusterPolicyInvalidOption[T any](mode string) request.Option[T] {
	switch mode {
	case "nil":
		return nil
	case "policy_id", "enabled":
		return request.WithField[T](mode, nil)
	case "query":
		return request.WithQuery[T]("cluster_id", "other-parent")
	case "argument":
		return request.WithArgument[T]("target", "other-parent")
	case "malformed-field":
		return request.WithField[T]("vendor", json.RawMessage(`{`))
	default:
		return request.WithHeader[T](mode, "forbidden")
	}
}

func TestClusteringClusterPolicyCommandsInputPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid policy command reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := clusters.New(client)
	for _, command := range clusterPolicyCommands {
		for _, policyID := range []string{"", " \t"} {
			if _, err := clusterPolicyCommandCall(context.Background(), api, command, resource.Name("selected"), policyID); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(command, err)
			}
		}
		for _, ref := range []resource.Ref{resource.ID(""), resource.Name(""), resource.ID("../other"), resource.ID("target?query")} {
			if _, err := clusterPolicyCommandCall(context.Background(), api, command, ref, "policy"); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(command, err)
			}
		}
		for _, mode := range []string{"nil", "policy_id", "enabled", "query", "argument", "malformed-field", "X-Auth-Token", "oPeNsTaCk-ApI-vErSiOn", "Authorization", "Host", "Content-Type", "bad header"} {
			if command == "DetachPolicy" && mode == "enabled" {
				continue
			}
			var err error
			switch command {
			case "AttachPolicy":
				_, err = api.AttachPolicy(context.Background(), resource.Name("selected"), clusters.AttachPolicyOpts{PolicyID: "policy"}, clusterPolicyInvalidOption[clusters.AttachPolicyOpts](mode))
			case "DetachPolicy":
				_, err = api.DetachPolicy(context.Background(), resource.Name("selected"), clusters.DetachPolicyOpts{PolicyID: "policy"}, clusterPolicyInvalidOption[clusters.DetachPolicyOpts](mode))
			default:
				_, err = api.UpdatePolicy(context.Background(), resource.Name("selected"), clusters.UpdatePolicyOpts{PolicyID: "policy"}, clusterPolicyInvalidOption[clusters.UpdatePolicyOpts](mode))
			}
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(command, mode, err)
			}
		}
		for _, invalidAPI := range []*clusters.API{nil, clusters.New(nil), clusters.New(&gophercloud.ServiceClient{})} {
			if _, err := clusterPolicyCommandCall(context.Background(), invalidAPI, command, resource.ID("cluster"), "policy"); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(command, err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := clusterPolicyCommandCall(ctx, api, command, resource.ID("cluster"), "policy"); !errors.Is(err, context.Canceled) {
			t.Fatal(command, err)
		}
		client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.99"}
		if _, err := clusterPolicyCommandCall(context.Background(), api, command, resource.ID("cluster"), "policy"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(command, err)
		}
		client.MoreHeaders = nil
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClusterPolicyCommandsSourceRecheckAfterLookup(t *testing.T) {
	for _, command := range clusterPolicyCommands {
		for _, mode := range []string{"type", "version", "header", "context"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/senlin/v1")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var lists, posts atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					switch mode {
					case "type":
						client.Type = "compute"
					case "version":
						client.Microversion = "latest"
					case "header":
						client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.99"}
					case "context":
						cancel()
					}
					testcloud.JSON(w, 200, `{"clusters":[{"id":"cluster-id","name":"selected"}]}`)
				})
				cloud.Mux.HandleFunc("POST /senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); t.Error("changed source reached mutation") })
				_, err := clusterPolicyCommandCall(ctx, clusters.New(client), command, resource.Name("selected"), "policy")
				want := resource.ErrInvalidOption
				if mode == "version" {
					want = resource.ErrUnsupported
				} else if mode == "context" {
					want = context.Canceled
				}
				if !errors.Is(err, want) || lists.Load() != 1 || posts.Load() != 0 {
					t.Fatal(err, lists.Load(), posts.Load())
				}
			})
		}
	}
}

func TestClusteringClusterPolicyCommandsLookupFailuresNeverSubmit(t *testing.T) {
	fixtures := []struct {
		name, body string
		cause      error
	}{
		{"missing", `{"clusters":[]}`, resource.ErrNotFound},
		{"ambiguous", `{"clusters":[{"id":"first","name":"selected"},{"id":"second","name":"selected"}]}`, resource.ErrAmbiguous},
		{"null-id", `{"clusters":[{"id":null,"name":"selected"}]}`, resource.ErrInvalidOption},
	}
	for _, command := range clusterPolicyCommands {
		for _, fixture := range fixtures {
			t.Run(command+"/"+fixture.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, posts atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); testcloud.JSON(w, 200, fixture.body) })
				cloud.Mux.HandleFunc("POST /senlin/v1/clusters/", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); t.Error("failed lookup reached mutation") })
				_, err := clusterPolicyCommandCall(context.Background(), clusters.New(cloud.Client("clustering", "/senlin/v1")), command, resource.Name("selected"), "policy")
				if !errors.Is(err, fixture.cause) || lists.Load() != 1 || posts.Load() != 0 {
					t.Fatal(err, lists.Load(), posts.Load())
				}
			})
		}
	}
}

func TestClusteringClusterPolicyCommandsMalformedAcceptedResponseEvidence(t *testing.T) {
	fixtures := []struct{ name, body, location string }{
		{"empty-body", "", "actions/action-id"},
		{"malformed-body", `{`, "actions/action-id"},
		{"missing-action", `{}`, "actions/action-id"},
		{"null-action", `{"action":null}`, "actions/action-id"},
		{"object-action", `{"action":{}}`, "actions/action-id"},
		{"empty-action", `{"action":""}`, "actions/action-id"},
		{"missing-location", `{"action":"action-id"}`, ""},
		{"mismatch-location", `{"action":"action-id"}`, "actions/other-id"},
		{"foreign-location", `{"action":"action-id"}`, "https://foreign.example/v1/actions/action-id"},
		{"wrong-path", `{"action":"action-id"}`, "clusters/action-id"},
		{"query-location", `{"action":"action-id"}`, "actions/action-id?follow=true"},
	}
	for _, command := range clusterPolicyCommands {
		for _, fixture := range fixtures {
			t.Run(command+"/"+fixture.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("POST /senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if fixture.location != "" {
						w.Header().Set("Location", fixture.location)
					}
					w.Header().Set("X-Request-ID", "accepted-evidence")
					testcloud.JSON(w, 202, fixture.body)
				})
				value, err := clusterPolicyCommandCall(context.Background(), clusters.New(cloud.Client("clustering", "/senlin/v1")), command, resource.ID("cluster-id"), "policy")
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != fixture.body || evidence.Header.Get("X-Request-ID") != "accepted-evidence" || evidence.Header.Get("Location") != fixture.location || calls.Load() != 1 {
					t.Fatal(value, err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClusterPolicyCommandsHTTPFailuresRemainNative(t *testing.T) {
	for _, command := range clusterPolicyCommands {
		for _, code := range []int{200, 201, 400, 403, 404, 409, 503} {
			t.Run(command+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				body := `{"error":{"message":"native-error"}}`
				cloud.Mux.HandleFunc("POST /senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "original-request")
					testcloud.JSON(w, code, body)
				})
				value, err := clusterPolicyCommandCall(context.Background(), clusters.New(cloud.Client("clustering", "/senlin/v1")), command, resource.ID("cluster-id"), "policy")
				var native gophercloud.ErrUnexpectedResponseCode
				var evidence *resource.ResponseError
				var operation *resource.OperationError
				if value != nil || !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &native) || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "original-request" || errors.As(err, &evidence) || !errors.As(err, &operation) || operation.Operation != command || calls.Load() != 1 {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClusterPolicyCommandsOptionErrorsRemainOriginal(t *testing.T) {
	cause := errors.New("custom-policy-option")
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("option failure reached HTTP", r.URL)
	})
	api := clusters.New(cloud.Client("clustering", "/senlin/v1"))
	for _, command := range clusterPolicyCommands {
		var err error
		switch command {
		case "AttachPolicy":
			_, err = api.AttachPolicy(context.Background(), resource.Name("selected"), clusters.AttachPolicyOpts{PolicyID: "policy"}, func(*request.Config[clusters.AttachPolicyOpts]) error { return cause })
		case "DetachPolicy":
			_, err = api.DetachPolicy(context.Background(), resource.Name("selected"), clusters.DetachPolicyOpts{PolicyID: "policy"}, func(*request.Config[clusters.DetachPolicyOpts]) error { return cause })
		default:
			_, err = api.UpdatePolicy(context.Background(), resource.Name("selected"), clusters.UpdatePolicyOpts{PolicyID: "policy"}, func(*request.Config[clusters.UpdatePolicyOpts]) error { return cause })
		}
		if !errors.Is(err, cause) || !strings.Contains(err.Error(), command) {
			t.Fatal(command, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}
