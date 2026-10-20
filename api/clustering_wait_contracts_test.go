package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type clusteringWaitAPI struct {
	path, key string
	hasStatus bool
	status    func(context.Context, *gophercloud.ServiceClient, resource.Ref, string, ...resource.WaitOption) (any, error)
	deleted   func(context.Context, *gophercloud.ServiceClient, resource.Ref, ...resource.WaitOption) error
}

var clusteringWaitAPIs = []clusteringWaitAPI{
	{"actions", "action", true, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return actions.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return actions.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"clusters", "cluster", true, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return clusters.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return clusters.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"nodes", "node", true, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return nodes.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return nodes.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"events", "event", true, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return events.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return events.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"profiles", "profile", false, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return profiles.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return profiles.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"policies", "policy", false, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return policies.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return policies.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"receivers", "receiver", false, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return receivers.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return receivers.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"profile-types", "profile_type", false, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return profiletypes.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return profiletypes.New(c).WaitForDelete(ctx, ref, opts...)
	}},
	{"policy-types", "policy_type", false, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, target string, opts ...resource.WaitOption) (any, error) {
		return policytypes.New(c).WaitForStatus(ctx, ref, target, opts...)
	}, func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref, opts ...resource.WaitOption) error {
		return policytypes.New(c).WaitForDelete(ctx, ref, opts...)
	}},
}

func clusteringWaitMetadata(value any) *resource.Metadata {
	switch value := value.(type) {
	case *actions.Action:
		return &value.Metadata
	case *clusters.Cluster:
		return &value.Metadata
	case *nodes.Node:
		return &value.Metadata
	case *events.Event:
		return &value.Metadata
	case *profiles.Profile:
		return &value.Metadata
	case *policies.Policy:
		return &value.Metadata
	case *receivers.Receiver:
		return &value.Metadata
	case *profiletypes.ProfileType:
		return &value.Metadata
	case *policytypes.PolicyType:
		return &value.Metadata
	default:
		return nil
	}
}

func clusteringWaitBody(key, state string) string {
	return fmt.Sprintf(`{%q:{"id":"different-response-id","name":%q,"status":%q,"future":{"big":9007199254740993,"off":false}}}`, key, state, state)
}

func clusteringWaitOptions(api clusteringWaitAPI) []resource.WaitOption {
	opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
	if !api.hasStatus {
		opts = append(opts, resource.WithStatusAttribute("name"))
	}
	return opts
}

func TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence(t *testing.T) {
	for _, api := range clusteringWaitAPIs {
		t.Run(api.path, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/reverse/senlin/v1/"+api.path+"/controller-name", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
					t.Error(r.Method, r.URL, r.Header)
				}
				if calls.Add(1) == 1 {
					w.Header().Set("Location", "https://foreign.example/do-not-follow")
					w.Header().Set("X-Request-ID", "wait-evidence")
					testcloud.JSON(w, 200, clusteringWaitBody(api.key, "READY"))
				} else {
					testcloud.JSON(w, 404, `{"error":"gone"}`)
				}
			})
			client := cloud.Client("clustering", "/reverse/senlin/v1")
			client.Microversion = "1.13"
			value, err := api.status(context.Background(), client, resource.ID("controller-name"), "ready", clusteringWaitOptions(api)...)
			if err != nil {
				t.Fatal(err)
			}
			metadata := clusteringWaitMetadata(value)
			if metadata == nil || metadata.StatusCode != 200 || metadata.Header.Get("X-Request-ID") != "wait-evidence" || metadata.Header.Get("Location") != "https://foreign.example/do-not-follow" || string(metadata.Body["future"]) != `{"big":9007199254740993,"off":false}` {
				t.Fatal(value)
			}
			if err := api.deleted(context.Background(), client, resource.ID("controller-name")); err != nil || calls.Load() != 2 {
				t.Fatal("wait submitted DELETE, followed Location, or failed on 404", err, calls.Load())
			}
		})
	}
}

func TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP(t *testing.T) {
	for _, api := range clusteringWaitAPIs {
		t.Run(api.path, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			client := cloud.Client("clustering", "/senlin/v1")
			for _, target := range []string{"", " "} {
				if _, err := api.status(context.Background(), client, resource.ID("fixed"), target); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(target, err)
				}
			}
			for _, opts := range [][]resource.WaitOption{{nil}, {resource.WithTimeout(0)}, {resource.WithPollInterval(0)}, {resource.WithFailureStates(" ")}} {
				if _, err := api.status(context.Background(), client, resource.ID("fixed"), "READY", opts...); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if err := api.deleted(context.Background(), client, resource.ID("fixed"), opts...); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			for _, attr := range []string{"missing_field", "Body"} {
				if _, err := api.status(context.Background(), client, resource.ID("fixed"), "READY", resource.WithStatusAttribute(attr)); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(attr, err)
				}
				if err := api.deleted(context.Background(), client, resource.ID("fixed"), resource.WithStatusAttribute(attr)); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(attr, err)
				}
			}
			if !api.hasStatus {
				if _, err := api.status(context.Background(), client, resource.ID("fixed"), "READY"); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("invented a default status field", err)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("invalid policy reached HTTP", calls.Load())
			}
		})
	}
}

func TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence(t *testing.T) {
	tests := []struct {
		name, state, target string
		opts                []resource.WaitOption
		failure             bool
		gets                int32
	}{
		{"default-error", "eRrOr", "READY", nil, true, 1},
		{"target-error-wins", "ERROR", "error", nil, false, 1},
		{"custom-error-disabled", "ERROR", "READY", []resource.WaitOption{resource.WithFailureStates("BROKEN")}, false, 2},
		{"custom-state-fails", "broken", "READY", []resource.WaitOption{resource.WithFailureStates("BROKEN")}, true, 1},
		{"empty-failures", "ERROR", "READY", []resource.WaitOption{resource.WithFailureStates()}, false, 2},
		{"failed-is-not-default", "FAILED", "READY", nil, false, 2},
		{"cancelled-is-not-default", "CANCELLED", "READY", nil, false, 2},
	}
	for _, api := range clusteringWaitAPIs {
		for _, test := range tests {
			t.Run(api.path+"/"+test.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+api.path+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					state := test.state
					if calls.Add(1) > 1 {
						state = "READY"
					}
					testcloud.JSON(w, 200, clusteringWaitBody(api.key, state))
				})
				opts := append(clusteringWaitOptions(api), test.opts...)
				_, err := api.status(context.Background(), cloud.Client("clustering", "/senlin/v1"), resource.ID("fixed"), test.target, opts...)
				if errors.Is(err, resource.ErrFailedState) != test.failure || (!test.failure && err != nil) || calls.Load() != test.gets {
					t.Fatal(err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringWaitContractsDeletionStatusAndExplicitAttribute(t *testing.T) {
	for _, api := range clusteringWaitAPIs {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/explicit=%t", api.path, explicit), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+api.path+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					if calls.Add(1) == 1 {
						testcloud.JSON(w, 200, clusteringWaitBody(api.key, "DeLeTeD"))
					} else {
						testcloud.JSON(w, 404, `{}`)
					}
				})
				opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
				if explicit {
					attr := "name"
					if api.path == "events" {
						attr = "status"
					}
					opts = append(opts, resource.WithStatusAttribute(attr))
				}
				err := api.deleted(context.Background(), cloud.Client("clustering", "/senlin/v1"), resource.ID("fixed"), opts...)
				wantGets := int32(1)
				if !api.hasStatus && !explicit {
					wantGets = 2
				}
				if err != nil || calls.Load() != wantGets {
					t.Fatal("incidental raw status changed deletion semantics", err, calls.Load(), wantGets)
				}
			})
		}
	}
}

type clusteringWaitTransport func(*http.Request) (*http.Response, error)

func (run clusteringWaitTransport) RoundTrip(r *http.Request) (*http.Response, error) { return run(r) }

func TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides(t *testing.T) {
	tests := []struct {
		name    string
		deleted bool
		common  bool
		parent  time.Duration
		opts    []resource.WaitOption
		want    time.Duration
	}{
		{"status-unlimited", false, false, 0, nil, 0},
		{"status-bounded", false, false, 0, []resource.WaitOption{resource.WithTimeout(15 * time.Second)}, 15 * time.Second},
		{"status-last-unlimited", false, false, 0, []resource.WaitOption{resource.WithTimeout(time.Minute), resource.WithUnlimitedWait()}, 0},
		{"status-last-bounded", false, false, 0, []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithTimeout(time.Minute)}, time.Minute},
		{"delete-default", true, false, 0, nil, 120 * time.Second},
		{"delete-unlimited", true, false, 0, []resource.WaitOption{resource.WithUnlimitedWait()}, 0},
		{"delete-last-bounded", true, false, 0, []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithTimeout(13 * time.Second)}, 13 * time.Second},
		{"status-parent-deadline", false, false, 5 * time.Second, nil, 5 * time.Second},
		{"delete-parent-deadline", true, false, 5 * time.Second, nil, 5 * time.Second},
		{"common-policy-unchanged", false, true, 0, nil, 5 * time.Minute},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/actions/fixed", func(w http.ResponseWriter, r *http.Request) {
				if test.deleted {
					testcloud.JSON(w, 404, `{}`)
				} else {
					testcloud.JSON(w, 200, clusteringWaitBody("action", "READY"))
				}
			})
			transport := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = clusteringWaitTransport(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				deadline, ok := r.Context().Deadline()
				if test.want == 0 {
					if ok {
						t.Error("unexpected SDK deadline", time.Until(deadline))
					}
				} else if !ok || time.Until(deadline) < test.want-time.Second || time.Until(deadline) > test.want {
					t.Error("wrong request deadline", ok, time.Until(deadline), test.want)
				}
				return transport.RoundTrip(r)
			})
			ctx := context.Background()
			if test.parent > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, test.parent)
				defer cancel()
			}
			api := actions.New(cloud.Client("clustering", "/senlin/v1"))
			var err error
			if test.deleted {
				err = api.WaitForDelete(ctx, resource.ID("fixed"), test.opts...)
			} else if test.common {
				_, err = api.Resources.Wait(ctx, resource.ID("fixed"), "READY", test.opts...)
			} else {
				_, err = api.WaitForStatus(ctx, resource.ID("fixed"), "READY", test.opts...)
			}
			if err != nil || requests.Load() != 1 {
				t.Fatal(err, requests.Load())
			}
		})
	}
}

func TestClusteringWaitContractsNameOnceStableIDAndFreshAuth(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprintf("deleted=%t", deleted), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, progress atomic.Int32
			name := "550e8400-e29b-41d4-a716-446655440000"
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != name || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"nodes":[{"id":"wrong","name":"other","status":"ACTIVE"},{"id":"resolved","name":"`+name+`","status":"BUILDING"}]}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/nodes/resolved", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Auth-Token") != "rotated-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" || r.URL.RawQuery != "" {
					t.Error(r.URL, r.Header)
				}
				state := "BUILDING"
				if gets.Add(1) == 2 {
					state = "ACTIVE"
					if deleted {
						state = "DELETED"
					}
				}
				testcloud.JSON(w, 200, clusteringWaitBody("node", state))
			})
			client := cloud.Client("clustering", "/reverse/senlin/v1")
			client.Microversion = "1.13"
			api := nodes.New(client)
			opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(value int) {
				if value != 0 {
					t.Error("invented Senlin progress", value)
				}
				progress.Add(1)
				client.ProviderClient.SetToken("rotated-token")
			})}
			var err error
			if deleted {
				// Delete wait does not report the name-list result. Rotate before its first GET.
				transport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = clusteringWaitTransport(func(r *http.Request) (*http.Response, error) {
					response, err := transport.RoundTrip(r)
					if r.URL.Path == "/reverse/senlin/v1/nodes" {
						client.ProviderClient.SetToken("rotated-token")
					}
					return response, err
				})
				err = api.WaitForDelete(context.Background(), resource.Name(name), opts...)
			} else {
				value, waitErr := api.WaitForStatus(context.Background(), resource.Name(name), "ACTIVE", opts...)
				err = waitErr
				if err == nil && (value.ID != "different-response-id" || value.Status != "ACTIVE") {
					t.Fatal(value)
				}
			}
			wantProgress := int32(2)
			if deleted {
				wantProgress = 1
			}
			if err != nil || lists.Load() != 1 || gets.Load() != 2 || progress.Load() != wantProgress {
				t.Fatal("name resolved repeatedly, ID changed, or terminal callback ran", err, lists.Load(), gets.Load(), progress.Load())
			}
		})
	}
}

func TestClusteringWaitContractsSourceRecheckedBeforeEachPoll(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		for _, mode := range []string{"type", "version", "version-header"} {
			t.Run(fmt.Sprintf("%s/deleted=%t", mode, deleted), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, clusteringWaitBody("node", "BUILDING"))
				})
				client := cloud.Client("clustering", "/senlin/v1")
				api := nodes.New(client)
				opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(int) {
					switch mode {
					case "type":
						client.Type = "compute"
					case "version":
						client.Microversion = "latest"
					case "version-header":
						client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.99"}
					}
				})}
				var err error
				if deleted {
					err = api.WaitForDelete(context.Background(), resource.ID("fixed"), opts...)
				} else {
					_, err = api.WaitForStatus(context.Background(), resource.ID("fixed"), "ACTIVE", opts...)
				}
				want := resource.ErrInvalidOption
				if mode == "version" {
					want = resource.ErrUnsupported
				}
				if !errors.Is(err, want) || gets.Load() != 1 {
					t.Fatal("changed service source reached HTTP", err, gets.Load())
				}
			})
		}
	}
}

func TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
		state := "ACTIVE"
		if gets.Add(1) > 1 {
			state = "ERROR"
		}
		testcloud.JSON(w, 200, clusteringWaitBody("node", state))
	})
	api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
	cached, err := api.Get(context.Background(), "fixed")
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.WaitForStatus(context.Background(), resource.ID("fixed"), "ACTIVE")
	if !errors.Is(err, resource.ErrFailedState) || gets.Load() != 2 || cached.Status != "ACTIVE" {
		t.Fatal("reused or mutated caller's old model", err, gets.Load(), cached)
	}
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel-deleted=%t", deleted), func(t *testing.T) {
			fixture := testcloud.New(t)
			var polls, callbacks atomic.Int32
			fixture.Mux.HandleFunc("GET /senlin/v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
				polls.Add(1)
				testcloud.JSON(w, 200, clusteringWaitBody("node", "BUILDING"))
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			nodeAPI := nodes.New(fixture.Client("clustering", "/senlin/v1"))
			opts := []resource.WaitOption{resource.WithProgressCallback(func(value int) {
				callbacks.Add(1)
				if value != 0 {
					t.Error(value)
				}
				cancel()
			})}
			var err error
			if deleted {
				err = nodeAPI.WaitForDelete(ctx, resource.ID("fixed"), opts...)
			} else {
				_, err = nodeAPI.WaitForStatus(ctx, resource.ID("fixed"), "ACTIVE", opts...)
			}
			if !errors.Is(err, context.Canceled) || polls.Load() != 1 || callbacks.Load() != 1 {
				t.Fatal(err, polls.Load(), callbacks.Load())
			}
			if deleted {
				err = nodeAPI.WaitForDelete(ctx, resource.ID("fixed"))
			} else {
				_, err = nodeAPI.WaitForStatus(ctx, resource.ID("fixed"), "ACTIVE")
			}
			if !errors.Is(err, context.Canceled) || polls.Load() != 1 {
				t.Fatal("already cancelled context reached HTTP", err, polls.Load())
			}
		})
	}
}

func TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		for _, code := range []int{201, 403, 404, 409, 503} {
			t.Run(fmt.Sprintf("deleted=%t/code=%d", deleted, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, callbacks atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, code, `{"error":{"message":"native-error"}}`)
				})
				api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
				opts := []resource.WaitOption{resource.WithProgressCallback(func(int) { callbacks.Add(1) })}
				var err error
				if deleted {
					err = api.WaitForDelete(context.Background(), resource.ID("fixed"), opts...)
				} else {
					_, err = api.WaitForStatus(context.Background(), resource.ID("fixed"), "ACTIVE", opts...)
				}
				if code == 404 && deleted {
					if err != nil {
						t.Fatal(err)
					}
				} else if !gophercloud.ResponseCodeIs(err, code) || (code == 404 && !errors.Is(err, resource.ErrNotFound)) {
					t.Fatal("native HTTP error lost", err)
				}
				if gets.Load() != 1 || callbacks.Load() != 0 {
					t.Fatal("HTTP failure retried or called progress", gets.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, api := range clusteringWaitAPIs {
		for _, deleted := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/wrong-envelope/deleted=%t", api.path, deleted), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets atomic.Int32
				body := `{"wrong":{"status":"ACTIVE"}}`
				cloud.Mux.HandleFunc("GET /senlin/v1/"+api.path+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					w.Header().Set("X-Request-ID", "malformed-wait")
					testcloud.JSON(w, 200, body)
				})
				var err error
				client := cloud.Client("clustering", "/senlin/v1")
				if deleted {
					err = api.deleted(context.Background(), client, resource.ID("fixed"))
				} else {
					_, err = api.status(context.Background(), client, resource.ID("fixed"), "ACTIVE", clusteringWaitOptions(api)...)
				}
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "malformed-wait" || gets.Load() != 1 {
					t.Fatal("accepted response lost or resent", err, evidence, gets.Load())
				}
			})
		}
	}
	for _, body := range []string{`{`, `{"node":null}`, `{"node":false}`, `{"node":{"id":123}}`} {
		t.Run("malformed="+body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/nodes/fixed", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, body)
			})
			_, err := nodes.New(cloud.Client("clustering", "/senlin/v1")).WaitForStatus(context.Background(), resource.ID("fixed"), "ACTIVE")
			var evidence *resource.ResponseError
			if !errors.As(err, &evidence) || string(evidence.Body) != body || gets.Load() != 1 {
				t.Fatal(err, evidence, gets.Load())
			}
		})
	}
}

func TestClusteringWaitContractsTransportAndInFlightContextFailures(t *testing.T) {
	for _, mode := range []string{"transport", "parent-timeout", "sdk-timeout"} {
		for _, deleted := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deleted=%t", mode, deleted), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				failure := errors.New("original transport failure")
				cloud.Provider.HTTPClient.Transport = clusteringWaitTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if mode == "transport" {
						return nil, failure
					}
					<-r.Context().Done()
					return nil, r.Context().Err()
				})
				ctx := context.Background()
				opts := []resource.WaitOption(nil)
				if mode == "parent-timeout" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, 10*time.Millisecond)
					defer cancel()
				} else if mode == "sdk-timeout" {
					opts = []resource.WaitOption{resource.WithTimeout(10 * time.Millisecond)}
				}
				api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
				var err error
				if deleted {
					err = api.WaitForDelete(ctx, resource.ID("fixed"), opts...)
				} else {
					_, err = api.WaitForStatus(ctx, resource.ID("fixed"), "ACTIVE", opts...)
				}
				want := failure
				if mode != "transport" {
					want = context.DeadlineExceeded
				}
				if !errors.Is(err, want) || calls.Load() != 1 {
					t.Fatal("transport error or context lost/resent", err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringWaitContractsNameLookupFailuresNeverPollDetail(t *testing.T) {
	fixtures := []struct {
		name, body string
		cause      error
		laterError bool
	}{
		{"missing", `{"nodes":[]}`, resource.ErrNotFound, false},
		{"ambiguous", `{"nodes":[{"id":"first","name":"selected","status":"ACTIVE"},{"id":"second","name":"selected","status":"ACTIVE"}]}`, resource.ErrAmbiguous, false},
		{"missing-response-id", `{"nodes":[{"id":null,"name":"selected","status":"BUILDING"}]}`, resource.ErrInvalidOption, false},
		{"later-page-error", `{"nodes":[{"id":"first","name":"selected","status":"ACTIVE"}]}`, nil, true},
	}
	for _, fixture := range fixtures {
		for _, deleted := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deleted=%t", fixture.name, deleted), func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, gets atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("marker") == "next" {
						testcloud.JSON(w, 403, `{"error":"later-page"}`)
						return
					}
					if fixture.laterError {
						w.Header().Set("Link", `</senlin/v1/nodes?marker=next>; rel="next"`)
					}
					testcloud.JSON(w, 200, fixture.body)
				})
				cloud.Mux.HandleFunc("GET /senlin/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					t.Error("failed name lookup reached detail GET", r.URL)
					w.WriteHeader(500)
				})
				api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
				var err error
				if deleted {
					err = api.WaitForDelete(context.Background(), resource.Name("selected"))
				} else {
					_, err = api.WaitForStatus(context.Background(), resource.Name("selected"), "ACTIVE")
				}
				wantLists := int32(1)
				if fixture.laterError {
					wantLists = 2
					if !gophercloud.ResponseCodeIs(err, 403) {
						t.Fatal("early list match hid later failure", err)
					}
				} else if deleted && fixture.name == "missing" {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, fixture.cause) {
					t.Fatal(err)
				}
				if lists.Load() != wantLists || gets.Load() != 0 {
					t.Fatal(lists.Load(), gets.Load())
				}
			})
		}
	}
}
