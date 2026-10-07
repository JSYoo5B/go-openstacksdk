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

func TestClusteringClusterCommandsWireDefaultsAndPresence(t *testing.T) {
	ctx := context.Background()
	ids := []string{"node-name", "short-node-id"}
	add := clusters.WithAddNodesOptions(clusters.AddNodesOpts{Nodes: ids})
	remove := clusters.WithRemoveNodesOptions(clusters.RemoveNodesOpts{Nodes: ids, DestroyAfterDeletion: request.Present(false)})
	pairs := map[string]string{"old-name": "replacement-short-id"}
	replace := clusters.WithReplaceNodesOptions(clusters.ReplaceNodesOpts{Nodes: pairs})
	plugin := map[string]any{"precision": json.Number("9007199254740993")}
	resizePlugin := clusters.WithResizeField("plugin", plugin)
	ids[0], pairs["old-name"], plugin["precision"] = "caller-changed", "caller-changed", 0
	cases := []struct {
		name, body string
		call       func(*clusters.API) (*actions.Submission, error)
	}{
		{"scale-in-default", `{"scale_in":{"count":null}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleIn(ctx, resource.ID("controller-name"), clusters.ScaleInOpts{})
		}},
		{"scale-out-default", `{"scale_out":{"count":null}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleOut(ctx, resource.ID("controller-name"), clusters.ScaleOutOpts{})
		}},
		{"scale-in-zero", `{"scale_in":{"count":0,"vendor":false}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleIn(ctx, resource.ID("controller-name"), clusters.ScaleInOpts{}, clusters.WithScaleInCount(0), clusters.WithScaleInField("vendor", false))
		}},
		{"scale-out-count", `{"scale_out":{"action":"plugin-action","count":2,"id":"plugin-id","location":"plugin-location","status":"plugin-status"}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleOut(ctx, resource.ID("controller-name"), clusters.ScaleOutOpts{}, clusters.WithScaleOutCount(2), clusters.WithScaleOutField("action", "plugin-action"), clusters.WithScaleOutField("id", "plugin-id"), clusters.WithScaleOutField("location", "plugin-location"), clusters.WithScaleOutField("status", "plugin-status"))
		}},
		{"scale-in-explicit-null", `{"scale_in":{"count":null}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleIn(ctx, resource.ID("controller-name"), clusters.ScaleInOpts{Count: request.Present(2)}, clusters.WithScaleInCountNull())
		}},
		{"scale-out-explicit-null", `{"scale_out":{"count":null}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ScaleOut(ctx, resource.ID("controller-name"), clusters.ScaleOutOpts{Count: request.Present(2)}, clusters.WithScaleOutCountNull())
		}},
		{"resize-empty", `{"resize":{}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.Resize(ctx, resource.ID("controller-name"), clusters.ResizeOpts{})
		}},
		{"resize-exact-fields", `{"resize":{"adjustment_type":"CHANGE_IN_PERCENTAGE","max_size":-1,"min_size":0,"min_step":0,"number":9007199254740993.125,"plugin":{"precision":9007199254740993},"strict":false}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.Resize(ctx, resource.ID("controller-name"), clusters.ResizeOpts{}, clusters.WithResizeAdjustmentType(clusters.ChangeInPercentage), clusters.WithResizeNumber("9007199254740993.125"), clusters.WithResizeMinSize(0), clusters.WithResizeMaxSize(-1), clusters.WithResizeMinStep(0), clusters.WithResizeStrict(false), resizePlugin)
		}},
		{"resize-null", `{"resize":{"number":null,"strict":null}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.Resize(ctx, resource.ID("controller-name"), clusters.ResizeOpts{}, clusters.WithResizeOptions(clusters.ResizeOpts{Number: request.Null[json.Number](), Strict: request.Null[bool]()}))
		}},
		{"add-nodes-snapshot", `{"add_nodes":{"nodes":["node-name","short-node-id"]}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.AddNodes(ctx, resource.ID("controller-name"), clusters.AddNodesOpts{}, add)
		}},
		{"remove-nodes-snapshot-false", `{"del_nodes":{"destroy_after_deletion":false,"nodes":["node-name","short-node-id"]}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.RemoveNodes(ctx, resource.ID("controller-name"), clusters.RemoveNodesOpts{}, remove)
		}},
		{"remove-nodes-default", `{"del_nodes":{"nodes":["node-id"]}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.RemoveNodes(ctx, resource.ID("controller-name"), clusters.RemoveNodesOpts{Nodes: []string{"node-id"}})
		}},
		{"remove-nodes-null", `{"del_nodes":{"destroy_after_deletion":null,"nodes":["node-id"]}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.RemoveNodes(ctx, resource.ID("controller-name"), clusters.RemoveNodesOpts{Nodes: []string{"node-id"}, DestroyAfterDeletion: request.Null[bool]()})
		}},
		{"replace-nodes-snapshot", `{"replace_nodes":{"nodes":{"old-name":"replacement-short-id"}}}`, func(api *clusters.API) (*actions.Submission, error) {
			return api.ReplaceNodes(ctx, resource.ID("controller-name"), clusters.ReplaceNodesOpts{}, replace)
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			version, versionHeader := "", ""
			switch test.name {
			case "replace-nodes-snapshot":
				version, versionHeader = "1.3", "clustering 1.3"
			case "remove-nodes-snapshot-false", "remove-nodes-null":
				version, versionHeader = "1.4", "clustering 1.4"
			}
			var calls atomic.Int32
			response := `{"action":"command-action","plugin":{"exact":9007199254740993}}`
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/controller-name/actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != test.body || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != versionHeader {
					t.Error(string(body), r.Header)
				}
				w.Header().Set("Location", "actions/command-action")
				w.Header().Set("X-Request-ID", "command-evidence")
				testcloud.JSON(w, 202, response)
			})
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = version
			api := clusters.New(client)
			for range 2 {
				value, err := test.call(api)
				if err != nil || value == nil || value.ActionID != "command-action" || value.Location != "actions/command-action" || value.StatusCode != 202 || string(value.Body) != response || value.Header.Get("X-Request-ID") != "command-evidence" {
					t.Fatal(value, err)
				}
			}
			if calls.Load() != 2 {
				t.Fatal("command fetched, polled or resent an action", calls.Load())
			}
		})
	}
}

func TestClusteringClusterCommandsNameLookupFreezesInputsAndVersions(t *testing.T) {
	for _, mode := range []string{"snapshot", "replace-gate", "remove-gate", "source-type"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.4"
			var lists, posts atomic.Int32
			var addConfig *request.Config[clusters.AddNodesOpts]
			var removeConfig *request.Config[clusters.RemoveNodesOpts]
			var replaceConfig *request.Config[clusters.ReplaceNodesOpts]
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "selected" {
					t.Error(r.URL)
				}
				switch mode {
				case "snapshot":
					addConfig.Options.Nodes[0] = "changed-during-lookup"
					addConfig.Headers["X-Vendor"] = "changed-during-lookup"
					addConfig.Fields["vendor"][0] = '!'
				case "replace-gate":
					client.Microversion = "1.2"
					replaceConfig.Options.Nodes["old-node"] = "changed-during-lookup"
				case "remove-gate":
					client.Microversion = "1.3"
					removeConfig.Options.DestroyAfterDeletion = request.Optional[bool]{}
				case "source-type":
					client.Type = "network"
				}
				testcloud.JSON(w, 200, `{"clusters":[{"id":"resolved-cluster","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/resolved-cluster/actions", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"add_nodes":{"nodes":["old-node"],"vendor":{"flag":false}}}` || r.Header.Get("X-Vendor") != "original" {
					t.Error(string(body), r.Header)
				}
				w.Header().Set("Location", "actions/lookup-action")
				testcloud.JSON(w, 202, `{"action":"lookup-action"}`)
			})
			api := clusters.New(client)
			var value *actions.Submission
			var err error
			switch mode {
			case "replace-gate":
				capture := func(config *request.Config[clusters.ReplaceNodesOpts]) error { replaceConfig = config; return nil }
				value, err = api.ReplaceNodes(context.Background(), resource.Name("selected"), clusters.ReplaceNodesOpts{Nodes: map[string]string{"old-node": "new-node"}}, capture)
			case "remove-gate":
				capture := func(config *request.Config[clusters.RemoveNodesOpts]) error { removeConfig = config; return nil }
				value, err = api.RemoveNodes(context.Background(), resource.Name("selected"), clusters.RemoveNodesOpts{Nodes: []string{"old-node"}}, clusters.WithRemoveNodesDestroyAfterDeletion(false), capture)
			default:
				capture := func(config *request.Config[clusters.AddNodesOpts]) error { addConfig = config; return nil }
				value, err = api.AddNodes(context.Background(), resource.Name("selected"), clusters.AddNodesOpts{Nodes: []string{"old-node"}}, clusters.WithAddNodesField("vendor", map[string]bool{"flag": false}), clusters.WithAddNodesHeader("X-Vendor", "original"), capture)
			}
			if mode == "snapshot" {
				if err != nil || value == nil || value.ActionID != "lookup-action" || posts.Load() != 1 {
					t.Fatal(value, err, posts.Load())
				}
			} else {
				want := resource.ErrUnsupported
				if mode == "source-type" {
					want = resource.ErrInvalidOption
				}
				if value != nil || !errors.Is(err, want) || posts.Load() != 0 {
					t.Fatal("lookup weakened the frozen command requirements", value, err, posts.Load())
				}
			}
			if lists.Load() != 1 {
				t.Fatal("name was not resolved exactly once", lists.Load())
			}
		})
	}
}

func TestClusteringClusterCommandsVersionPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("gated command made HTTP", r.URL) })
	client := cloud.Client("clustering", "/senlin/v1")
	api := clusters.New(client)
	for _, version := range []string{"", "1.2", "latest"} {
		client.Microversion = version
		if _, err := api.ReplaceNodes(context.Background(), resource.Name("selected"), clusters.ReplaceNodesOpts{Nodes: map[string]string{"old": "new"}}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	for _, version := range []string{"", "1.3", "latest"} {
		client.Microversion = version
		for _, destroy := range []request.Optional[bool]{request.Present(false), request.Present(true), request.Null[bool]()} {
			if _, err := api.RemoveNodes(context.Background(), resource.Name("selected"), clusters.RemoveNodesOpts{Nodes: []string{"node"}, DestroyAfterDeletion: destroy}); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClusterCommandsAcceptedResponseEvidenceAndNoResend(t *testing.T) {
	cases := []struct {
		name, body string
		locations  []string
	}{
		{"missing-action", `{}`, []string{"actions/accepted"}},
		{"null-action", `{"action":null}`, []string{"actions/accepted"}},
		{"number-action", `{"action":1}`, []string{"actions/accepted"}},
		{"empty-action", `{"action":""}`, []string{"actions/accepted"}},
		{"invalid-action", `{"action":"unsafe/action"}`, []string{"actions/accepted"}},
		{"array", `[]`, []string{"actions/accepted"}},
		{"malformed-json", `{"action":`, []string{"actions/accepted"}},
		{"missing-location", `{"action":"accepted"}`, nil},
		{"different-location", `{"action":"accepted"}`, []string{"actions/other"}},
		{"foreign-location", `{"action":"accepted"}`, []string{"https://foreign.invalid/v1/actions/accepted"}},
		{"multiple-locations", `{"action":"accepted"}`, []string{"actions/accepted", "actions/accepted"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/controller-id/actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				for _, location := range test.locations {
					w.Header().Add("Location", location)
				}
				w.Header().Set("X-Request-ID", "accepted-command")
				testcloud.JSON(w, 202, test.body)
			})
			value, err := clusters.New(cloud.Client("clustering", "/senlin/v1")).Resize(context.Background(), resource.ID("controller-id"), clusters.ResizeOpts{})
			var evidence *resource.ResponseError
			if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != test.body || evidence.Header.Get("X-Request-ID") != "accepted-command" || calls.Load() != 1 {
				t.Fatal(value, err, evidence, calls.Load())
			}
		})
	}
}

func TestClusteringClusterCommandsPreflightErrorsAndCancellation(t *testing.T) {
	t.Run("invalid-input", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			t.Error("invalid command reached HTTP", r.URL)
		})
		api := clusters.New(cloud.Client("clustering", "/senlin/v1"))
		ctx := context.Background()
		ref := resource.Name("selected")
		cases := []struct {
			name string
			call func() error
		}{
			{"nil-add-nodes", func() error { _, e := api.AddNodes(ctx, ref, clusters.AddNodesOpts{}); return e }},
			{"empty-add-nodes", func() error { _, e := api.AddNodes(ctx, ref, clusters.AddNodesOpts{Nodes: []string{}}); return e }},
			{"invalid-add-node", func() error {
				_, e := api.AddNodes(ctx, ref, clusters.AddNodesOpts{Nodes: []string{"bad/node"}})
				return e
			}},
			{"empty-remove-nodes", func() error { _, e := api.RemoveNodes(ctx, ref, clusters.RemoveNodesOpts{}); return e }},
			{"invalid-remove-node", func() error {
				_, e := api.RemoveNodes(ctx, ref, clusters.RemoveNodesOpts{Nodes: []string{""}})
				return e
			}},
			{"nil-replacement", func() error { _, e := api.ReplaceNodes(ctx, ref, clusters.ReplaceNodesOpts{}); return e }},
			{"empty-replacement", func() error {
				_, e := api.ReplaceNodes(ctx, ref, clusters.ReplaceNodesOpts{Nodes: map[string]string{}})
				return e
			}},
			{"invalid-replacement-key", func() error {
				_, e := api.ReplaceNodes(ctx, ref, clusters.ReplaceNodesOpts{Nodes: map[string]string{"": "new"}})
				return e
			}},
			{"invalid-replacement-value", func() error {
				_, e := api.ReplaceNodes(ctx, ref, clusters.ReplaceNodesOpts{Nodes: map[string]string{"old": "bad/node"}})
				return e
			}},
			{"core-count", func() error {
				_, e := api.ScaleIn(ctx, ref, clusters.ScaleInOpts{}, clusters.WithScaleInField("count", 2))
				return e
			}},
			{"core-number", func() error {
				_, e := api.Resize(ctx, ref, clusters.ResizeOpts{}, clusters.WithResizeField("number", 2))
				return e
			}},
			{"core-nodes", func() error {
				_, e := api.AddNodes(ctx, ref, clusters.AddNodesOpts{Nodes: []string{"node"}}, clusters.WithAddNodesField("nodes", []string{"other"}))
				return e
			}},
			{"invalid-number", func() error {
				_, e := api.Resize(ctx, ref, clusters.ResizeOpts{}, clusters.WithResizeNumber("NaN"))
				return e
			}},
			{"auth-header", func() error {
				_, e := api.Resize(ctx, ref, clusters.ResizeOpts{}, clusters.WithResizeHeader("X-Auth-Token", "replace"))
				return e
			}},
			{"unsupported-query", func() error {
				_, e := api.ScaleIn(ctx, ref, clusters.ScaleInOpts{}, request.WithQuery[clusters.ScaleInOpts]("vendor", "ignored"))
				return e
			}},
			{"unsupported-argument", func() error {
				_, e := api.ScaleOut(ctx, ref, clusters.ScaleOutOpts{}, request.WithArgument[clusters.ScaleOutOpts]("vendor", true))
				return e
			}},
			{"nil-option", func() error { _, e := api.Resize(ctx, ref, clusters.ResizeOpts{}, nil); return e }},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				if err := test.call(); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := api.ScaleOut(canceled, resource.ID("id"), clusters.ScaleOutOpts{}); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if _, err := clusters.New(nil).ScaleIn(ctx, resource.ID("id"), clusters.ScaleInOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if calls.Load() != 0 {
			t.Fatal(calls.Load())
		}
	})
	for _, code := range []int{200, 201, 204, 400, 403, 404, 409, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/id/actions", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "actions/native-error")
				testcloud.JSON(w, code, `{"action":"native-error","error":"server decision"}`)
			})
			value, err := clusters.New(cloud.Client("clustering", "/senlin/v1")).ScaleIn(context.Background(), resource.ID("id"), clusters.ScaleInOpts{})
			var accepted *resource.ResponseError
			if value != nil || !gophercloud.ResponseCodeIs(err, code) || errors.As(err, &accepted) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

type clusterCommandTransport func(*http.Request) (*http.Response, error)

func (f clusterCommandTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClusteringClusterCommandsCustomTransportSubmissionOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	header := http.Header{"location": {"actions/transport-action"}, "X-Request-Id": {"transport-evidence"}}
	responseBody := `{"action":"transport-action","extra":{"exact":9007199254740993}}`
	var calls atomic.Int32
	client.HTTPClient.Transport = clusterCommandTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/senlin/v1/clusters/id/actions" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Method, r.URL, r.Header)
		}
		return &http.Response{StatusCode: 202, Header: header, Body: io.NopCloser(strings.NewReader(responseBody)), Request: r}, nil
	})
	value, err := clusters.New(client).ScaleOut(context.Background(), resource.ID("id"), clusters.ScaleOutOpts{}, clusters.WithScaleOutCount(1))
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	header["location"][0], header["X-Request-Id"][0] = "changed", "changed"
	if value.ActionID != "transport-action" || value.Location != "actions/transport-action" || value.Header["location"][0] != "actions/transport-action" || value.Header.Get("X-Request-ID") != "transport-evidence" || string(value.Body) != responseBody || value.StatusCode != 202 || calls.Load() != 1 {
		t.Fatal("submission did not retain independent source transport evidence", value, calls.Load())
	}
}
