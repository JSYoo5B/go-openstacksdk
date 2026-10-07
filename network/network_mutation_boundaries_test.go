package network_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	nativeNetworks "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
)

func mutationCall(ctx context.Context, s *network.Service, operation string) (*network.Network, bool, error) {
	switch operation {
	case "create":
		row, err := s.CreateNetwork(ctx, network.CreateNetworkRequest{Name: "chosen"})
		return row, false, err
	case "update":
		row, err := s.UpdateNetwork(ctx, resource.ID("chosen"), network.WithNetworkName("changed"))
		return row, false, err
	case "delete":
		deleted, err := s.DeleteNetwork(ctx, resource.ID("chosen"))
		return nil, deleted, err
	default:
		return nil, false, s.Networks.Delete(ctx, resource.ID("chosen"))
	}
}

func mutationChosenLookup(cloud *testcloud.Cloud) {
	cloud.Mux.HandleFunc("GET /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"network":{"id":"chosen"}}`)
	})
}

func TestNetworkMutationRejectsInvalidServicesContextsAndReplacedPublicParts(t *testing.T) {
	for _, scenario := range []string{"nil service", "nil client", "nil provider", "invalid origin", "nil context", "canceled", "replaced networks", "replaced roles"} {
		for _, operation := range []string{"create", "update", "delete"} {
			t.Run(scenario+"/"+operation, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("invalid preflight sent %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				s := mutationService(t, cloud)
				ctx := context.Background()
				switch scenario {
				case "nil service":
					s = nil
				case "nil client":
					s = network.New(nil)
				case "nil provider":
					s.RawClient().ProviderClient = nil
				case "invalid origin":
					s.RawClient().Endpoint = "https://user:pass@example.com/v2.0/"
				case "nil context":
					ctx = nil
				case "canceled":
					canceled, cancel := context.WithCancel(ctx)
					cancel()
					ctx = canceled
				case "replaced networks":
					s.Networks = network.New(cloud.Client("network", "/other")).Networks
				case "replaced roles":
					s.Roles = network.New(cloud.Client("network", "/other")).Roles
				}
				row, deleted, err := mutationCall(ctx, s, operation)
				want := resource.ErrInvalidOption
				if scenario == "canceled" {
					want = context.Canceled
				}
				if row != nil || deleted || !errors.Is(err, want) {
					t.Fatal(row, deleted, err)
				}
			})
		}
	}
}

func TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, payload := range []string{`{`, `null`, `{}`, `{"network":null}`, `{"network":[]}`, `{"network":{"id":"chosen","shared":"invalid"}}`, `{"network":{"id":"bad/id"}}`, `{"network":{"id":"other"}}`} {
			t.Run(operation+"/"+payload, func(t *testing.T) {
				cloud := testcloud.New(t)
				var inventory, mutations, callbacks atomic.Int32
				mutationInventory(cloud, &inventory)
				code, method, path := 201, "POST", "/v2.0/networks"
				if operation == "update" {
					code, method, path = 200, "PUT", "/v2.0/networks/chosen"
				}
				cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
					mutations.Add(1)
					w.Header().Set("X-Request-Id", "accepted-mutation")
					testcloud.JSON(w, code, payload)
				})
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					callbacks.Add(1)
					return original
				}
				s := mutationService(t, cloud)
				requireMutationCache(t, s, &inventory, 1)
				row, _, err := mutationCall(context.Background(), s, operation)
				if payload == `{"network":{"id":"other"}}` && operation == "create" {
					if err != nil || row == nil || row.ID != "other" {
						t.Fatal(row, err)
					}
				} else {
					var proof *resource.ResponseError
					if err == nil || !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "accepted-mutation" || string(proof.Body) != payload {
						t.Fatal(row, err, proof)
					}
					partial := payload == `{"network":{"id":"bad/id"}}` || payload == `{"network":{"id":"other"}}`
					if (row != nil) != partial {
						t.Fatal("unexpected partial model", row, err)
					}
				}
				requireMutationCache(t, s, &inventory, 2)
				if mutations.Load() != 1 || callbacks.Load() != 0 {
					t.Fatal(mutations.Load(), callbacks.Load())
				}
			})
		}
	}
}

type mutationTransport func(*http.Request) (*http.Response, error)

func (f mutationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type mutationErrorReader struct {
	payload string
	cause   error
}

func (r *mutationErrorReader) Read(buffer []byte) (int, error) {
	n := copy(buffer, r.payload)
	r.payload = r.payload[n:]
	return n, r.cause
}

type mutationBody struct {
	io.Reader
	onClose func() error
	closes  *atomic.Int32
}

func (b *mutationBody) Close() error { b.closes.Add(1); return b.onClose() }

func TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete", "collection delete"} {
		for _, mode := range []string{"read", "close", "cancel", "roles nil", "networks replacement", "endpoint replacement"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var inventory, mutations, callbacks, closes atomic.Int32
				mutationInventory(cloud, &inventory)
				mutationChosenLookup(cloud)
				s := mutationService(t, cloud)
				requireMutationCache(t, s, &inventory, 1)
				originalRoles, originalNetworks, originalEndpoint := s.Roles, s.Networks, s.RawClient().Endpoint
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted read failed"), errors.New("accepted close failed"), errors.New("caller canceled")
				code, payload := 201, `{"network":{"id":"chosen"}}`
				if operation == "update" {
					code = 200
				}
				if strings.Contains(operation, "delete") {
					code, payload = 204, ""
				}
				originalTransport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = mutationTransport(func(r *http.Request) (*http.Response, error) {
					if r.Method == http.MethodGet {
						return originalTransport.RoundTrip(r)
					}
					mutations.Add(1)
					var reader io.Reader = strings.NewReader(payload)
					if mode == "read" {
						reader = &mutationErrorReader{payload: payload, cause: readCause}
					}
					body := &mutationBody{Reader: reader, closes: &closes, onClose: func() error {
						switch mode {
						case "close":
							return closeCause
						case "cancel":
							cancel(cancelCause)
						case "roles nil":
							s.Roles = nil
						case "networks replacement":
							s.Networks = network.New(cloud.Client("network", "/other")).Networks
						case "endpoint replacement":
							s.RawClient().Endpoint = cloud.Server.URL + "/other/"
						}
						return nil
					}}
					return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"actual-mutation"}}, Body: body}, nil
				})
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					callbacks.Add(1)
					return original
				}
				row, deleted, err := mutationCall(ctx, s, operation)
				var proof *resource.ResponseError
				if row != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "actual-mutation" {
					t.Fatal(row, deleted, err, proof)
				}
				if mode == "read" && !errors.Is(err, readCause) || mode == "close" && !errors.Is(err, closeCause) || mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(err)
				}
				if strings.Contains(mode, "replacement") || mode == "roles nil" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
				if operation == "delete" && !deleted || mutations.Load() != 1 || closes.Load() != 1 || callbacks.Load() != 0 {
					t.Fatal(deleted, mutations.Load(), closes.Load(), callbacks.Load())
				}
				s.Roles, s.Networks, s.RawClient().Endpoint = originalRoles, originalNetworks, originalEndpoint
				requireMutationCache(t, s, &inventory, 2)
			})
		}
	}
}

func TestNetworkMutationRetryFailuresAndExpanded404DoNotBecomeSuccessOrMissing(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete", "collection delete"} {
		for _, mode := range []string{"source", "body", "callback", "cancel", "expanded404"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var inventory, mutations, callbacks atomic.Int32
				mutationInventory(cloud, &inventory)
				mutationChosenLookup(cloud)
				s := mutationService(t, cloud)
				originalEndpoint := s.RawClient().Endpoint
				requireMutationCache(t, s, &inventory, 1)
				handler := func(w http.ResponseWriter, r *http.Request) {
					call := mutations.Add(1)
					code := 404
					if mode == "expanded404" && call == 1 {
						code = 503
					}
					w.Header().Set("X-Request-Id", "rejected-mutation")
					testcloud.JSON(w, code, `{"error":"actual rejection"}`)
				}
				cloud.Mux.HandleFunc("POST /v2.0/networks", handler)
				cloud.Mux.HandleFunc("PUT /v2.0/networks/chosen", handler)
				cloud.Mux.HandleFunc("DELETE /v2.0/networks/chosen", handler)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				callbackCause := errors.New("callback refused")
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, count uint) error {
					callbacks.Add(1)
					if count > 1 {
						return original
					}
					switch mode {
					case "source":
						s.RawClient().Endpoint = cloud.Server.URL + "/other/"
					case "body":
						options.JSONBody = map[string]any{"network": map[string]any{"name": "changed by callback"}}
					case "callback":
						return callbackCause
					case "cancel":
						cancel()
					case "expanded404":
						options.OkCodes = append(options.OkCodes, 404)
					}
					return nil
				}
				row, deleted, err := mutationCall(ctx, s, operation)
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || deleted || err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "rejected-mutation" {
					t.Fatal(row, deleted, err, native)
				}
				if (mode == "source" || mode == "body") && !errors.Is(err, resource.ErrInvalidOption) || mode == "callback" && !errors.Is(err, callbackCause) || mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				want := int32(1)
				if mode == "expanded404" {
					want = 2
				}
				if mutations.Load() != want || callbacks.Load() != 1 {
					t.Fatal(mutations.Load(), callbacks.Load())
				}
				s.RawClient().Endpoint = originalEndpoint
				requireMutationCache(t, s, &inventory, 1)
			})
		}
	}
}

func TestCreateNetworkAvailabilityZonePagesAndSourceGuard(t *testing.T) {
	for _, scenario := range []string{"body next", "empty next", "header next", "late403", "cycle", "foreign origin", "source retry"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var pages, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/extensions", func(w http.ResponseWriter, r *http.Request) {
				pages.Add(1)
				if scenario == "source retry" {
					testcloud.JSON(w, 503, `{}`)
					return
				}
				if r.URL.Query().Get("marker") == "last" {
					if scenario == "late403" {
						testcloud.JSON(w, 403, `{}`)
					} else {
						testcloud.JSON(w, 200, `{"extensions":[{"alias":"network_availability_zone"}]}`)
					}
					return
				}
				link := cloud.Server.URL + "/v2.0/extensions?marker=last"
				if scenario == "cycle" {
					link = cloud.Server.URL + "/v2.0/extensions"
				}
				if scenario == "foreign origin" {
					link = "http://foreign.invalid/v2.0/extensions?marker=last"
				}
				rows := `[{"alias":"other"}]`
				if scenario == "late403" {
					rows = `[{"alias":"network_availability_zone"}]`
				}
				if scenario == "empty next" {
					rows = `[]`
				}
				if scenario == "header next" {
					w.Header().Set("Link", "<"+link+">; rel=\"next\"")
					testcloud.JSON(w, 200, `{"extensions":`+rows+`}`)
				} else {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"extensions":%s,"extensions_links":[{"rel":"next","href":%q}]}`, rows, link))
				}
			})
			cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				testcloud.JSON(w, 201, `{"network":{"id":"created"}}`)
			})
			s := mutationService(t, cloud)
			if scenario == "source retry" {
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					s.RawClient().Endpoint = cloud.Server.URL + "/other/"
					return nil
				}
			}
			row, err := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{}, network.WithNetworkAvailabilityZoneHints())
			success := scenario == "body next" || scenario == "empty next" || scenario == "header next"
			if success {
				if err != nil || row == nil || posts.Load() != 1 {
					t.Fatal(row, err, posts.Load())
				}
			} else {
				if err == nil || row != nil || posts.Load() != 0 || errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(row, err, posts.Load())
				}
				if scenario == "source retry" && !errors.Is(err, resource.ErrInvalidOption) || scenario == "late403" && !gophercloud.ResponseCodeIs(err, 403) {
					t.Fatal(err)
				}
			}
			wantPages := int32(2)
			if scenario == "cycle" || scenario == "foreign origin" || scenario == "source retry" {
				wantPages = 1
			}
			if pages.Load() != wantPages {
				t.Fatal(pages.Load())
			}
		})
	}
}

func TestNetworkMutationOwnedProviderExtensionAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var posts atomic.Int32
	cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body := networkMutationBody(t, r)
		want := map[string]any{"name": "owned", "admin_state_up": true, "provider:segmentation_id": map[string]any{"values": []any{float64(0)}}, "vendor:values": []any{"original"}}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body=%v want=%v", body, want)
		}
		testcloud.JSON(w, 201, `{"network":{"id":"created"}}`)
	})
	segmentation := map[string]any{"values": []int{0}}
	extension := []string{"original"}
	provider := network.WithNetworkProvider(network.ProviderNetwork{SegmentationID: request.Present[any](segmentation)})
	field := network.WithNetworkField("vendor:values", extension)
	segmentation["values"].([]int)[0] = 99
	extension[0] = "changed"
	s := mutationService(t, cloud)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			row, err := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "owned"},
				network.WithNetworkProvider(network.ProviderNetwork{NetworkType: request.Present("first"), PhysicalNetwork: request.Present("first")}), provider, field)
			if err != nil || row == nil {
				t.Error(row, err)
			}
		}()
	}
	wait.Wait()
	if posts.Load() != 8 {
		t.Fatal(posts.Load())
	}
}

func TestNetworkMutationResetCannotPublishEarlierInflightDiscovery(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var inventory atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		call := inventory.Add(1)
		if call == 1 {
			close(started)
			<-release
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"cache-%d","name":"private"}]}`, call))
	})
	cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 201, `{"network":{"id":"created"}}`) })
	s := mutationService(t, cloud)
	type discovery struct {
		row *network.NetworkRoleSnapshot
		err error
	}
	finished := make(chan discovery, 1)
	go func() { row, err := s.Roles.Discover(context.Background()); finished <- discovery{row, err} }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("discovery did not start")
	}
	if row, err := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{}); err != nil || row == nil {
		t.Fatal(row, err)
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case old := <-finished:
		if old.err != nil || old.row.DefaultNetwork.ID != "cache-1" {
			t.Fatal(old)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("old discovery did not finish")
	}
	requireMutationCache(t, s, &inventory, 2)
	requireMutationCache(t, s, &inventory, 2)
}

func TestDeleteNetworkInitial404PreservesArbitraryCallbackFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var inventory, gets, deletes atomic.Int32
	mutationInventory(cloud, &inventory)
	cloud.Mux.HandleFunc("GET /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("DELETE /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
	cause := errors.New("lookup callback failed")
	cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return errors.Join(original, cause)
	}
	s := mutationService(t, cloud)
	requireMutationCache(t, s, &inventory, 1)
	deleted, err := s.DeleteNetwork(context.Background(), resource.ID("chosen"))
	if deleted || !errors.Is(err, cause) || errors.Is(err, resource.ErrNotFound) || gets.Load() != 1 || deletes.Load() != 0 {
		t.Fatal(deleted, err, gets.Load(), deletes.Load())
	}
	requireMutationCache(t, s, &inventory, 1)
}

func TestNetworkMutationSourceGuardRemainsFailedAfterCallbackRestoresSource(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete", "collection delete", "AZ probe"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventory, requests, backoffs, retries atomic.Int32
			mutationInventory(cloud, &inventory)
			mutationChosenLookup(cloud)
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventory, 1)
			originalEndpoint := s.RawClient().Endpoint
			handler := func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				testcloud.JSON(w, 429, `{"error":"rate limit"}`)
			}
			cloud.Mux.HandleFunc("POST /v2.0/networks", handler)
			cloud.Mux.HandleFunc("PUT /v2.0/networks/chosen", handler)
			cloud.Mux.HandleFunc("DELETE /v2.0/networks/chosen", handler)
			cloud.Mux.HandleFunc("GET /v2.0/extensions", handler)
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoffs.Add(1)
				s.RawClient().Endpoint = cloud.Server.URL + "/changed/"
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				s.RawClient().Endpoint = originalEndpoint
				return nil
			}
			var err error
			if operation == "AZ probe" {
				_, err = s.CreateNetwork(context.Background(), network.CreateNetworkRequest{}, network.WithNetworkAvailabilityZoneHints())
			} else {
				_, _, err = mutationCall(context.Background(), s, operation)
			}
			if !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 {
				t.Fatal(err, requests.Load(), backoffs.Load(), retries.Load())
			}
			requireMutationCache(t, s, &inventory, 1)
		})
	}
}

func TestNetworkMutationRawAndGeneratedLanesRequireExplicitReset(t *testing.T) {
	cloud := testcloud.New(t)
	var inventory, posts atomic.Int32
	mutationInventory(cloud, &inventory)
	cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, 201, `{"network":{"id":"created"}}`)
	})
	s := mutationService(t, cloud)
	requireMutationCache(t, s, &inventory, 1)
	if row, err := s.API.Networks.Create(context.Background(), nativeNetworks.CreateOpts{Name: "generated"}); err != nil || row == nil {
		t.Fatal(row, err)
	}
	requireMutationCache(t, s, &inventory, 1)
	if row, err := nativeNetworks.Create(context.Background(), s.RawClient(), nativeNetworks.CreateOpts{Name: "native"}).Extract(); err != nil || row == nil {
		t.Fatal(row, err)
	}
	requireMutationCache(t, s, &inventory, 1)
	s.Roles.Reset()
	requireMutationCache(t, s, &inventory, 2)
	if posts.Load() != 2 {
		t.Fatal(posts.Load())
	}
}

func TestNetworkMutationLookupSourceChangeStopsBeforeResendOrWrite(t *testing.T) {
	for _, operation := range []string{"name update", "name delete", "ID delete", "empty update"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventories, lookups, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("name") != "" {
					lookups.Add(1)
					testcloud.JSON(w, 503, `{"error":"lookup unavailable"}`)
					return
				}
				call := inventories.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"cache-%d","name":"private"}]}`, call))
			})
			cloud.Mux.HandleFunc("GET /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				testcloud.JSON(w, 503, `{"error":"lookup unavailable"}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				testcloud.JSON(w, 200, `{"network":{"id":"chosen"}}`)
			})
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventories, 1)
			endpoint := s.RawClient().Endpoint
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				s.RawClient().Endpoint = cloud.Server.URL + "/other/"
				return nil
			}
			var err error
			switch operation {
			case "name update":
				_, err = s.UpdateNetwork(context.Background(), resource.Name("chosen"), network.WithNetworkName("changed"))
			case "name delete":
				_, err = s.DeleteNetwork(context.Background(), resource.Name("chosen"))
			case "ID delete":
				_, err = s.DeleteNetwork(context.Background(), resource.ID("chosen"))
			default:
				_, err = s.UpdateNetwork(context.Background(), resource.ID("chosen"))
			}
			if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || lookups.Load() != 1 || writes.Load() != 0 {
				t.Fatal(err, lookups.Load(), writes.Load())
			}
			s.RawClient().Endpoint = endpoint
			requireMutationCache(t, s, &inventories, 1)
		})
	}
}

func TestNetworkMutationNoContentNameLookupKeepsMissingPolicyAndCache(t *testing.T) {
	for _, operation := range []string{"update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventories, lookups, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("name") == "missing" {
					lookups.Add(1)
					w.WriteHeader(204)
					return
				}
				call := inventories.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"cache-%d","name":"private"}]}`, call))
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { writes.Add(1); w.WriteHeader(500) })
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventories, 1)
			if operation == "update" {
				row, err := s.UpdateNetwork(context.Background(), resource.Name("missing"), network.WithNetworkName("changed"))
				if row != nil || !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(row, err)
				}
			} else {
				deleted, err := s.DeleteNetwork(context.Background(), resource.Name("missing"))
				if deleted || err != nil {
					t.Fatal(deleted, err)
				}
			}
			if lookups.Load() != 1 || writes.Load() != 0 {
				t.Fatal(lookups.Load(), writes.Load())
			}
			requireMutationCache(t, s, &inventories, 1)
		})
	}
}
