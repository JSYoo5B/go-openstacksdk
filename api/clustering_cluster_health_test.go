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
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringClusterHealthWirePresenceAndRoutes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, version, path, body string
		call                      func(*clusters.API) (*actions.Submission, error)
	}{
		{"check", "", "actions", `{"check":{}}`, func(a *clusters.API) (*actions.Submission, error) { return a.Check(ctx, resource.ID("controller-id")) }},
		{"check-plugin", "", "actions", `{"check":{"id":"plugin-id","status":false}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Check(ctx, resource.ID("controller-id"), clusters.WithCheckField("id", "plugin-id"), clusters.WithCheckField("status", false))
		}},
		{"recover", "", "actions", `{"recover":{}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("controller-id"), clusters.RecoverOpts{})
		}},
		{"recover-check-only-false", "1.6", "actions", `{"recover":{"check":false}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("controller-id"), clusters.RecoverOpts{}, clusters.WithRecoverCheck(false))
		}},
		{"recover-check-only-null", "1.6", "actions", `{"recover":{"check":null}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("controller-id"), clusters.RecoverOpts{}, clusters.WithRecoverCheckNull())
		}},
		{"recover-false", "1.7", "actions", `{"recover":{"check":false,"check_capacity":false,"operation":"","operation_params":{}}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("controller-id"), clusters.RecoverOpts{}, clusters.WithRecoverOperation(""), clusters.WithRecoverOperationParams(map[string]any{}), clusters.WithRecoverCheck(false), clusters.WithRecoverCheckCapacity(false))
		}},
		{"recover-null", "1.7", "actions", `{"recover":{"check":null,"check_capacity":null,"operation":null,"operation_params":null}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("controller-id"), clusters.RecoverOpts{}, clusters.WithRecoverOperationNull(), clusters.WithRecoverOperationParams(nil), clusters.WithRecoverCheckNull(), clusters.WithRecoverCheckCapacityNull())
		}},
		{"operation-empty", "1.4", "ops", `{"reboot":{}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.PerformOperation(ctx, resource.ID("controller-id"), "reboot", clusters.PerformOperationOpts{})
		}},
		{"operation-fields", "1.4", "ops", `{"reboot":{"filters":{"role":"worker"},"params":{"precision":9007199254740993.125},"status":"plugin-status"}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.PerformOperation(ctx, resource.ID("controller-id"), "reboot", clusters.PerformOperationOpts{}, clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}), clusters.WithPerformOperationParams(map[string]any{"precision": json.Number("9007199254740993.125")}), clusters.WithPerformOperationField("status", "plugin-status"))
		}},
		{"operation-null", "1.4", "ops", `{"reboot":{"filters":null,"params":null}}`, func(a *clusters.API) (*actions.Submission, error) {
			return a.PerformOperation(ctx, resource.ID("controller-id"), "reboot", clusters.PerformOperationOpts{}, clusters.WithPerformOperationFilters(nil), clusters.WithPerformOperationParams(nil))
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/controller-id/"+test.path, func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				body, _ := io.ReadAll(r.Body)
				version := ""
				if test.version != "" {
					version = "clustering " + test.version
				}
				if string(body) != test.body || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != version {
					t.Error(string(body), r.Header)
				}
				w.Header().Set("Location", "/reverse/senlin/v1/actions/health-action")
				w.Header().Set("X-Request-ID", "accepted-evidence")
				testcloud.JSON(w, 202, `{"action":"health-action","vendor":9007199254740993}`)
			})
			client := cloud.Client("clustering", "/reverse/senlin/v1")
			client.Microversion = test.version
			for range 2 {
				value, err := test.call(clusters.New(client))
				if err != nil || value == nil || value.ActionID != "health-action" || value.StatusCode != 202 || value.Header.Get("X-Request-ID") != "accepted-evidence" || !strings.Contains(string(value.Body), "9007199254740993") {
					t.Fatal(value, err)
				}
			}
			if posts.Load() != 2 {
				t.Fatal("command implicitly fetched or resent", posts.Load())
			}
		})
	}
}

func TestClusteringClusterHealthSnapshotsBeforeNameLookup(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.4"
	params := json.RawMessage(`{"exact":9007199254740993}`)
	filters := map[string]string{"role": "worker"}
	value := clusters.WithPerformOperationOptions(clusters.PerformOperationOpts{Params: params})
	filter := clusters.WithPerformOperationFilters(filters)
	plugin := map[string]any{"flag": false}
	extra := clusters.WithPerformOperationField("vendor", plugin)
	params[0], filters["role"], plugin["flag"] = '!', "changed", true
	var config *request.Config[clusters.PerformOperationOpts]
	var lists, posts atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") != "selected" {
			t.Error(r.URL)
		}
		config.Options.Filters[0], config.Options.Params[0], config.Fields["vendor"][0] = '!', '!', '!'
		config.Headers["X-Vendor"] = "changed"
		testcloud.JSON(w, 200, `{"clusters":[{"id":"resolved","name":"selected"}]}`)
	})
	cloud.Mux.HandleFunc("POST /senlin/v1/clusters/resolved/ops", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"reboot":{"filters":{"role":"worker"},"params":{"exact":9007199254740993},"vendor":{"flag":false}}}` || r.Header.Get("X-Vendor") != "original" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "actions/snapshot")
		testcloud.JSON(w, 202, `{"action":"snapshot"}`)
	})
	for range 2 {
		capture := func(c *request.Config[clusters.PerformOperationOpts]) error { config = c; return nil }
		got, err := clusters.New(client).PerformOperation(context.Background(), resource.Name("selected"), "reboot", clusters.PerformOperationOpts{}, value, filter, extra, clusters.WithPerformOperationHeader("X-Vendor", "original"), capture)
		if err != nil || got == nil || got.ActionID != "snapshot" {
			t.Fatal(got, err)
		}
	}
	if lists.Load() != 2 || posts.Load() != 2 {
		t.Fatal(lists.Load(), posts.Load())
	}
}

func TestClusteringClusterHealthVersionPreflightAndRecheck(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("preflight made HTTP", r.URL) })
	a := clusters.New(client)
	for _, version := range []string{"", "1.3", "latest"} {
		client.Microversion = version
		if _, err := a.PerformOperation(context.Background(), resource.Name("selected"), "reboot", clusters.PerformOperationOpts{}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	for _, version := range []string{"", "1.5", "latest"} {
		client.Microversion = version
		for _, check := range []request.Optional[bool]{request.Present(false), request.Present(true), request.Null[bool]()} {
			if _, err := a.Recover(context.Background(), resource.Name("selected"), clusters.RecoverOpts{Check: check}); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, check, err)
			}
		}
	}
	for _, version := range []string{"", "1.6", "latest"} {
		client.Microversion = version
		for _, check := range []request.Optional[bool]{request.Present(false), request.Present(true), request.Null[bool]()} {
			if _, err := a.Recover(context.Background(), resource.Name("selected"), clusters.RecoverOpts{CheckCapacity: check}); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, check, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	for _, mode := range []string{"capacity", "check", "operation", "source"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.7"
			var posts, lists atomic.Int32
			var config *request.Config[clusters.RecoverOpts]
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				switch mode {
				case "capacity":
					client.Microversion = "1.6"
					config.Options.CheckCapacity = request.Optional[bool]{}
				case "check":
					client.Microversion = "1.5"
					config.Options.Check = request.Optional[bool]{}
				case "operation":
					client.Microversion = "1.3"
				case "source":
					client.Type = "network"
				}
				testcloud.JSON(w, 200, `{"clusters":[{"id":"resolved","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("POST /senlin/v1/clusters/resolved/", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); t.Error(r.URL) })
			a := clusters.New(client)
			var err error
			capture := func(c *request.Config[clusters.RecoverOpts]) error { config = c; return nil }
			switch mode {
			case "capacity":
				_, err = a.Recover(context.Background(), resource.Name("selected"), clusters.RecoverOpts{}, clusters.WithRecoverCheckCapacity(false), capture)
			case "check":
				_, err = a.Recover(context.Background(), resource.Name("selected"), clusters.RecoverOpts{}, clusters.WithRecoverCheck(false), capture)
			case "operation":
				_, err = a.PerformOperation(context.Background(), resource.Name("selected"), "reboot", clusters.PerformOperationOpts{})
			case "source":
				_, err = a.Check(context.Background(), resource.Name("selected"))
			}
			want := resource.ErrUnsupported
			if mode == "source" {
				want = resource.ErrInvalidOption
			}
			if !errors.Is(err, want) || lists.Load() != 1 || posts.Load() != 0 {
				t.Fatal(err, lists.Load(), posts.Load())
			}
		})
	}
}

func TestClusteringClusterHealthInvalidInputsNeverRequest(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.7"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error(r.URL) })
	a := clusters.New(client)
	ctx := context.Background()
	ref := resource.Name("selected")
	cases := []func() error{
		func() error { _, e := a.Check(ctx, ref, nil); return e },
		func() error {
			_, e := a.Check(ctx, ref, clusters.WithCheckHeader("x-auth-token", "replacement"))
			return e
		},
		func() error {
			_, e := a.Check(ctx, ref, request.WithQuery[clusters.CheckOpts]("vendor", "value"))
			return e
		},
		func() error {
			_, e := a.Recover(ctx, ref, clusters.RecoverOpts{OperationParams: json.RawMessage(`[]`)})
			return e
		},
		func() error {
			_, e := a.Recover(ctx, ref, clusters.RecoverOpts{}, clusters.WithRecoverField("check_capacity", false))
			return e
		},
		func() error { _, e := a.PerformOperation(ctx, ref, " ", clusters.PerformOperationOpts{}); return e },
		func() error {
			_, e := a.PerformOperation(ctx, ref, "reboot", clusters.PerformOperationOpts{Filters: json.RawMessage(`[]`)})
			return e
		},
		func() error {
			_, e := a.PerformOperation(ctx, ref, "reboot", clusters.PerformOperationOpts{Params: json.RawMessage(`1`)})
			return e
		},
		func() error {
			_, e := a.PerformOperation(ctx, ref, "reboot", clusters.PerformOperationOpts{}, clusters.WithPerformOperationField("filters", map[string]any{}))
			return e
		},
	}
	for i, call := range cases {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(i, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClusterHealthAcceptedEvidenceAndNativeErrors(t *testing.T) {
	ctx := context.Background()
	calls := []struct {
		name, path string
		call       func(context.Context, *clusters.API) (*actions.Submission, error)
	}{
		{"check", "actions", func(ctx context.Context, a *clusters.API) (*actions.Submission, error) {
			return a.Check(ctx, resource.ID("target"))
		}},
		{"recover", "actions", func(ctx context.Context, a *clusters.API) (*actions.Submission, error) {
			return a.Recover(ctx, resource.ID("target"), clusters.RecoverOpts{})
		}},
		{"operation", "ops", func(ctx context.Context, a *clusters.API) (*actions.Submission, error) {
			return a.PerformOperation(ctx, resource.ID("target"), "reboot", clusters.PerformOperationOpts{})
		}},
	}
	for _, command := range calls {
		for _, status := range []int{202, 200, 409} {
			t.Run(command.name+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.7"
				var posts atomic.Int32
				body := `{"action":"accepted"}`
				cloud.Mux.HandleFunc("POST /senlin/v1/clusters/target/"+command.path, func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					w.Header().Set("X-Evidence", "original")
					w.Header().Set("Location", "actions/different")
					testcloud.JSON(w, status, body)
				})
				got, err := command.call(ctx, clusters.New(client))
				if got != nil || err == nil || posts.Load() != 1 {
					t.Fatal(got, err, posts.Load())
				}
				if status == 202 {
					var e *resource.ResponseError
					if !errors.As(err, &e) || e.StatusCode != 202 || string(e.Body) != body || e.Header.Get("X-Evidence") != "original" {
						t.Fatal(err)
					}
				} else if !gophercloud.ResponseCodeIs(err, status) {
					t.Fatal(err)
				}
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				if _, err := command.call(cancelled, clusters.New(client)); !errors.Is(err, context.Canceled) || posts.Load() != 1 {
					t.Fatal(err, posts.Load())
				}
			})
		}
	}
}
