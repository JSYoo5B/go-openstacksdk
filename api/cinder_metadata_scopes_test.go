package api_test

import (
	"context"
	"encoding/json"
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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage/metadata"
	snapshots2 "github.com/JSYoo5B/gophercloudsdk/blockstorage/v2/snapshots"
	volumes2 "github.com/JSYoo5B/gophercloudsdk/blockstorage/v2/volumes"
	snapshots3 "github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/snapshots"
	volumes3 "github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Source: pinned MetadataMixin and the Cinder stock-server metadata controllers.
// Explicit Name binding, owned HTTP evidence and safe key escaping are Go policies;
// these fixtures are HTTP contract proofs, not live-cloud or Resource-cache proofs.
type cinderMetadataScope interface {
	ID() string
	RawClient() *gophercloud.ServiceClient
	Get(context.Context, ...metadata.Option) (*metadata.Result, error)
	Merge(context.Context, map[string]string, ...metadata.Option) (*metadata.Result, error)
	Replace(context.Context, map[string]string, ...metadata.Option) (*metadata.Result, error)
	DeleteKeys(context.Context, []string, ...metadata.Option) (*metadata.DeleteResult, error)
}

type cinderMetadataFixture struct {
	version, collection string
	bind                func(context.Context, *gophercloud.ServiceClient, resource.Ref) (cinderMetadataScope, error)
	nativeGet           func(context.Context, *gophercloud.ServiceClient, string) (string, error)
}

func cinderMetadataFixtures() []cinderMetadataFixture {
	return []cinderMetadataFixture{
		{"v2", "volumes", func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref) (cinderMetadataScope, error) {
			v, err := volumes2.New(c).MetadataIn(ctx, ref)
			if v == nil {
				return nil, err
			}
			return v, err
		}, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (string, error) {
			v, err := volumes2.New(c).Get(ctx, id)
			if v == nil {
				return "", err
			}
			return v.ID, err
		}},
		{"v2", "snapshots", func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref) (cinderMetadataScope, error) {
			v, err := snapshots2.New(c).MetadataIn(ctx, ref)
			if v == nil {
				return nil, err
			}
			return v, err
		}, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (string, error) {
			v, err := snapshots2.New(c).Get(ctx, id)
			if v == nil {
				return "", err
			}
			return v.ID, err
		}},
		{"v3", "volumes", func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref) (cinderMetadataScope, error) {
			v, err := volumes3.New(c).MetadataIn(ctx, ref)
			if v == nil {
				return nil, err
			}
			return v, err
		}, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (string, error) {
			v, err := volumes3.New(c).Get(ctx, id)
			if v == nil {
				return "", err
			}
			return v.ID, err
		}},
		{"v3", "snapshots", func(ctx context.Context, c *gophercloud.ServiceClient, ref resource.Ref) (cinderMetadataScope, error) {
			v, err := snapshots3.New(c).MetadataIn(ctx, ref)
			if v == nil {
				return nil, err
			}
			return v, err
		}, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (string, error) {
			v, err := snapshots3.New(c).Get(ctx, id)
			if v == nil {
				return "", err
			}
			return v.ID, err
		}},
	}
}

func (f cinderMetadataFixture) name() string   { return f.version + "/" + f.collection }
func (f cinderMetadataFixture) prefix() string { return "/reverse/cinder/" + f.version + "/project/" }
func (f cinderMetadataFixture) endpoint(id string) string {
	return f.prefix() + f.collection + "/" + id + "/metadata"
}
func (f cinderMetadataFixture) listPath() string {
	if f.collection == "volumes" || f.version == "v3" {
		return f.prefix() + f.collection + "/detail"
	}
	return f.prefix() + f.collection
}

func cinderMetadataBind(t *testing.T, f cinderMetadataFixture, c *gophercloud.ServiceClient) cinderMetadataScope {
	t.Helper()
	s, err := f.bind(context.Background(), c, resource.ID("selected"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCinderMetadataScopesRoutesAndOwnedResponse(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		t.Run(f.name(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			const response = `{"metadata":{"owner":"server","日本語":"値"},"vendor":{"large":9007199254740993,"fraction":1.2300},"METADATA":false}`
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.URL, r.Header)
				}
				if r.Method == http.MethodGet {
					if body, _ := io.ReadAll(r.Body); len(body) != 0 {
						t.Error("GET body", string(body))
					}
				} else {
					var body map[string]map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]map[string]string{"metadata": {"owner": "request"}}) {
						t.Error(body, err)
					}
				}
				w.Header().Add("X-Proof", "first")
				w.Header().Add("X-Proof", "second")
				testcloud.JSON(w, 200, response)
			})
			client := cloud.Client("block-storage", f.prefix())
			s := cinderMetadataBind(t, f, client)
			if calls.Load() != 0 || s.ID() != "selected" || s.RawClient() != client {
				t.Fatal("constructor changed identity or performed HTTP", calls.Load(), s.ID())
			}
			input := map[string]string{"owner": "request"}
			for _, call := range []func() (*metadata.Result, error){
				func() (*metadata.Result, error) { return s.Get(context.Background()) },
				func() (*metadata.Result, error) { return s.Merge(context.Background(), input) },
				func() (*metadata.Result, error) { return s.Replace(context.Background(), input) },
			} {
				v, err := call()
				if err != nil || v == nil || v.StatusCode != 200 || string(v.Body) != response || !reflect.DeepEqual(v.Metadata, map[string]string{"owner": "server", "日本語": "値"}) || !reflect.DeepEqual(v.Header.Values("X-Proof"), []string{"first", "second"}) {
					t.Fatal(v, err)
				}
				v.Metadata["owner"], v.Header["X-Proof"][0], v.Body[0] = "caller", "caller", '!'
			}
			if input["owner"] != "request" || calls.Load() != 3 {
				t.Fatal("response ownership or request count", input, calls.Load())
			}
		})
	}
}

func TestCinderMetadataScopesEmptyMapsAndDeleteBranches(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		t.Run(f.name(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			var deleted []string
			var deletedMu sync.Mutex
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, _ := io.ReadAll(r.Body)
				if (r.Method != http.MethodPost && r.Method != http.MethodPut) || string(body) != `{"metadata":{}}` {
					t.Error(r.Method, string(body))
				}
				w.Header().Set("X-Clear", "actual")
				testcloud.JSON(w, 200, `{"metadata":{},"cleared":false}`)
			})
			cloud.Mux.HandleFunc(f.endpoint("selected")+"/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				key := strings.TrimPrefix(r.URL.Path, f.endpoint("selected")+"/")
				deletedMu.Lock()
				deleted = append(deleted, key)
				deletedMu.Unlock()
				if r.Method != http.MethodDelete {
					t.Error(r.Method)
				}
				w.Header().Set("X-Key", key)
				if key == "bad" {
					testcloud.JSON(w, 404, `{"missing":"bad"}`)
					return
				}
				w.WriteHeader(200)
			})
			s := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix()))
			for _, values := range []map[string]string{nil, {}} {
				if v, err := s.Merge(context.Background(), values); err != nil || v == nil || len(v.Metadata) != 0 {
					t.Fatal(v, err)
				}
				if v, err := s.Replace(context.Background(), values); err != nil || v == nil || len(v.Metadata) != 0 {
					t.Fatal(v, err)
				}
			}
			cleared, err := s.DeleteKeys(context.Background(), nil)
			if err != nil || cleared == nil || cleared.Cleared == nil || cleared.Cleared.Header.Get("X-Clear") != "actual" || cleared.Cleared.StatusCode != 200 || len(cleared.Deleted) != 0 {
				t.Fatal(cleared, err)
			}
			before := calls.Load()
			empty, err := s.DeleteKeys(context.Background(), []string{})
			if err != nil || empty == nil || empty.Cleared != nil || len(empty.Deleted) != 0 || calls.Load() != before {
				t.Fatal("empty keys fabricated HTTP evidence", empty, err, calls.Load())
			}
			result, err := s.DeleteKeys(context.Background(), []string{"first", "first", "bad", "unvisited"})
			var native gophercloud.ErrUnexpectedResponseCode
			deletedMu.Lock()
			observed := append([]string(nil), deleted...)
			deletedMu.Unlock()
			if !errors.As(err, &native) || native.Actual != 404 || result == nil || result.Cleared != nil || len(result.Deleted) != 2 || !reflect.DeepEqual(observed, []string{"first", "first", "bad"}) {
				t.Fatal(result, observed, err)
			}
			for _, ack := range result.Deleted {
				if ack.Key != "first" || ack.StatusCode != 200 || len(ack.Body) != 0 || ack.Header.Get("X-Key") != "first" {
					t.Fatal("deletion acknowledgement was synthesized", ack)
				}
			}
		})
	}
	// The SDK escapes a literal key once. This fixture does not assert support by
	// every WSGI deployment or override server-specific metadata key validation.
	t.Run("literal key", func(t *testing.T) {
		f := cinderMetadataFixtures()[3]
		cloud := testcloud.New(t)
		key := "rack/zone 位置"
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.EscapedPath() != f.endpoint("selected")+"/"+url.PathEscape(key) || r.URL.RawQuery != "" {
				t.Error(r.Method, r.URL)
			}
			w.WriteHeader(200)
		})
		v, err := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix())).DeleteKeys(context.Background(), []string{key})
		if err != nil || v == nil || len(v.Deleted) != 1 || v.Deleted[0].Key != key {
			t.Fatal(v, err)
		}
	})
}

func TestCinderMetadataScopesNameBindingAndRequestSnapshots(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		t.Run(f.name(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, mutations atomic.Int32
			wantLists := int32(2)
			if f.version == "v2" && f.collection == "snapshots" {
				// The existing native v2 SnapshotPage is a SinglePageBase. Name
				// binding deliberately preserves that collection's pagination ABI.
				wantLists = 1
			}
			cloud.Mux.HandleFunc(f.listPath(), func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "same name" {
					t.Error("name lookup changed", r.URL)
				}
				if wantLists == 1 {
					testcloud.JSON(w, 200, `{"snapshots":[{"id":"other","name":"decoy"},{"id":"selected","name":"same name"}]}`)
					return
				}
				if r.URL.Query().Get("marker") == "next" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":"selected","name":"same name"}],"%s_links":[]}`, f.collection, f.collection))
				} else {
					next := cloud.Server.URL + f.listPath() + "?name=same+name&marker=next"
					testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":"other","name":"decoy"}],"%s_links":[{"rel":"next","href":%q}]}`, f.collection, f.collection, next))
				}
			})
			values := map[string]string{"owner": "before-options"}
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				mutations.Add(1)
				var body map[string]map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["metadata"]["owner"] != "before-options" || r.Header.Get("X-Snapshot") != "before" {
					t.Error(body, err, r.Header)
				}
				testcloud.JSON(w, 200, `{"metadata":{"owner":"server"}}`)
			})
			client := cloud.Client("block-storage", f.prefix())
			s, err := f.bind(context.Background(), client, resource.Name("same name"))
			if err != nil || s.ID() != "selected" || lists.Load() != wantLists {
				t.Fatal(s, err, lists.Load())
			}
			headers := map[string]string{"X-Snapshot": "before"}
			option := metadata.WithHeaders(headers)
			headers["X-Snapshot"] = "after"
			mutate := metadata.Option(func(cfg *request.Config[metadata.Opts]) error { values["owner"] = "after-options"; return nil })
			v, err := s.Merge(context.Background(), values, option, mutate)
			if err != nil || v == nil || v.Metadata["owner"] != "server" || mutations.Load() != 1 || lists.Load() != wantLists || values["owner"] != "after-options" {
				t.Fatal(v, err, mutations.Load(), lists.Load(), values)
			}
		})
	}
	for _, mode := range []string{"missing", "duplicate", "invalid resolved ID", "late error"} {
		t.Run(mode, func(t *testing.T) {
			f := cinderMetadataFixtures()[2]
			cloud := testcloud.New(t)
			var mutation atomic.Int32
			cloud.Mux.HandleFunc(f.listPath(), func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("marker") != "" {
					testcloud.JSON(w, 403, `{}`)
					return
				}
				switch mode {
				case "missing":
					testcloud.JSON(w, 200, `{"volumes":[]}`)
				case "duplicate":
					testcloud.JSON(w, 200, `{"volumes":[{"id":"one","name":"target"},{"id":"two","name":"target"}]}`)
				case "invalid resolved ID":
					testcloud.JSON(w, 200, `{"volumes":[{"id":"unsafe/id","name":"target"}]}`)
				case "late error":
					testcloud.JSON(w, 200, fmt.Sprintf(`{"volumes":[{"id":"selected","name":"target"}],"volumes_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+f.listPath()+"?marker=next"))
				}
			})
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				mutation.Add(1)
				testcloud.JSON(w, 200, `{"metadata":{}}`)
			})
			_, err := f.bind(context.Background(), cloud.Client("block-storage", f.prefix()), resource.Name("target"))
			if err == nil || mutation.Load() != 0 {
				t.Fatal("name binding hid an observation", mode, err, mutation.Load())
			}
			switch mode {
			case "missing":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "invalid resolved ID":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "late error":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 {
					t.Fatal(err)
				}
			}
		})
	}
	t.Run("source validation before and after name lookup", func(t *testing.T) {
		f := cinderMetadataFixtures()[2]
		cloud, foreign := testcloud.New(t), testcloud.New(t)
		var localCalls, foreignCalls atomic.Int32
		cloud.Mux.HandleFunc(f.listPath(), func(w http.ResponseWriter, r *http.Request) {
			localCalls.Add(1)
			testcloud.JSON(w, 200, `{"volumes":[{"id":"selected","name":"target"}]}`)
		})
		foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			foreignCalls.Add(1)
			testcloud.JSON(w, 200, `{"volumes":[{"id":"selected","name":"target"}]}`)
		})
		client := cloud.Client("block-storage", f.prefix())
		client.ResourceBase = foreign.Server.URL + f.prefix()
		if scope, err := f.bind(context.Background(), client, resource.Name("target")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) || foreignCalls.Load() != 0 || localCalls.Load() != 0 {
			t.Fatal("cross-origin base was used for parent resolution", scope, err, localCalls.Load(), foreignCalls.Load())
		}
		client.ResourceBase = ""
		transport := cloud.Provider.HTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		cloud.Provider.HTTPClient.Transport = cinderMetadataRoundTrip(func(r *http.Request) (*http.Response, error) {
			response, err := transport.RoundTrip(r)
			client.Type = "compute" // same call goroutine, before resolution returns
			return response, err
		})
		if scope, err := f.bind(context.Background(), client, resource.Name("target")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) || localCalls.Load() != 1 {
			t.Fatal("lookup changed source without revalidation", scope, err, localCalls.Load())
		}
	})
}

func TestCinderMetadataScopesMalformedAcceptedAndOriginalErrors(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		for _, body := range []string{`{"metadata":{"nullable":null}}`, `{"metadata":{"number":1}}`, `{"metadata":null}`, `{"METADATA":{}}`, `{"metadata":[]}`, `{"metadata":`, "{\"metadata\":{\"invalid\":\"\xff\"}}"} {
			t.Run(f.name()+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("must not retry accepted decode")
				}
				cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Evidence", "original")
					testcloud.JSON(w, 200, body)
				})
				v, err := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix())).Get(context.Background())
				var accepted *resource.ResponseError
				if v != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Evidence") != "original" || calls.Load() != 1 || retries.Load() != 0 {
					t.Fatal(v, accepted, err, calls.Load(), retries.Load())
				}
			})
		}
		for _, code := range []int{201, 204, 400, 404} {
			t.Run(fmt.Sprintf("%s/code%d", f.name(), code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Native", "kept")
					testcloud.JSON(w, code, `{"metadata":{}}`)
				})
				v, err := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix())).Replace(context.Background(), map[string]string{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Native") != "kept" || calls.Load() != 1 || errors.Is(err, resource.ErrNotFound) {
					t.Fatal(v, native, err, calls.Load())
				}
			})
		}
	}
	t.Run("accepted read failure", func(t *testing.T) {
		f := cinderMetadataFixtures()[2]
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cause := errors.New("metadata stream interrupted")
		cloud.Provider.HTTPClient.Transport = cinderMetadataRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: 200, Header: http.Header{"X-Read": {"partial"}}, Body: &cinderMetadataReadFailure{cause: cause}, Request: r}, nil
		})
		v, err := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix())).Get(context.Background())
		var accepted *resource.ResponseError
		if v != nil || !errors.Is(err, cause) || !errors.As(err, &accepted) || string(accepted.Body) != `{"metadata":` || accepted.StatusCode != 200 || accepted.Header.Get("X-Read") != "partial" || calls.Load() != 1 {
			t.Fatal(v, accepted, err, calls.Load())
		}
	})
}

func TestCinderMetadataScopesHeaderVersionAndETagPolicies(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		t.Run(f.name(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-Trace") != "configured" || r.Header.Get("X-Option") != "last" {
					t.Error(r.Header)
				}
				if f.version == "v3" && (r.Header.Get("X-OpenStack-Volume-API-Version") != "3.15" || r.Header.Get("OpenStack-API-Version") != "volume 3.15") {
					t.Error("selected version changed", r.Header)
				}
				if r.Header.Get("If-Match") == `"stale"` {
					testcloud.JSON(w, 412, `{"precondition":"failed"}`)
					return
				}
				w.Header().Set("ETag", `"current"`)
				testcloud.JSON(w, 200, `{"metadata":{"server":"value"}}`)
			})
			client := cloud.Client("block-storage", f.prefix())
			client.MoreHeaders = map[string]string{"X-Trace": "configured"}
			if f.version == "v3" {
				client.Microversion = "3.15"
			}
			s := cinderMetadataBind(t, f, client)
			options := []metadata.Option{metadata.WithHeader("X-Trace", "caller"), metadata.WithHeader("X-Option", "first"), metadata.WithHeader("x-option", "last")}
			v, err := s.Get(context.Background(), options...)
			if err != nil || v == nil || v.Header.Get("ETag") != `"current"` {
				t.Fatal(v, err)
			}
			v, err = s.Replace(context.Background(), map[string]string{"server": "next"}, append(options, metadata.WithHeader("If-Match", `"stale"`))...)
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.As(err, &native) || native.Actual != 412 || calls.Load() != 2 {
				t.Fatal("If-Match was interpreted or retried locally", v, err, calls.Load())
			}
			client.MoreHeaders["If-Match"] = `"configured"`
			before := calls.Load()
			_, err = s.Get(context.Background(), metadata.WithHeader("If-Match", `"different"`))
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("conditional source conflict sent HTTP", err, calls.Load())
			}
		})
	}
	t.Run("closed options and case conflicts", func(t *testing.T) {
		f := cinderMetadataFixtures()[2]
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{"metadata":{}}`) })
		client := cloud.Client("block-storage", f.prefix())
		s := cinderMetadataBind(t, f, client)
		for _, option := range []metadata.Option{
			metadata.WithHeader("X-Auth-Token", "different"), metadata.WithHeader("OpenStack-API-Version", "volume 3.99"), metadata.WithHeader("Host", "elsewhere"),
			func(c *request.Config[metadata.Opts]) error { c.Fields["metadata"] = json.RawMessage(`{}`); return nil },
			func(c *request.Config[metadata.Opts]) error { c.Query.Set("target", "elsewhere"); return nil },
			func(c *request.Config[metadata.Opts]) error { c.Arguments["base_path"] = "other"; return nil },
			func(c *request.Config[metadata.Opts]) error {
				c.Headers["X-Case"], c.Headers["x-case"] = "first", "different"
				return nil
			},
		} {
			_, err := s.Get(context.Background(), option)
			if err == nil || calls.Load() != 0 {
				t.Fatal("closed metadata option sent HTTP", err, calls.Load())
			}
		}
		client.MoreHeaders = map[string]string{"X-Case": "first", "x-case": "different"}
		if _, err := s.DeleteKeys(context.Background(), []string{}); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(err, calls.Load())
		}
		client.MoreHeaders = map[string]string{"X-Case": "same", "x-case": "same"}
		if _, err := s.Get(context.Background()); err != nil || calls.Load() != 1 {
			t.Fatal("equal aliases rejected", err, calls.Load())
		}
		for _, values := range []map[string]string{{"invalid": string([]byte{0xff})}, {string([]byte{0xff}): "invalid key"}} {
			before := calls.Load()
			if _, err := s.Merge(context.Background(), values); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("invalid Unicode map was silently repaired or sent", err, calls.Load())
			}
		}
	})
}

func TestCinderMetadataScopesFixedTargetLiveProviderAndContext(t *testing.T) {
	f := cinderMetadataFixtures()[2]
	t.Run("resource base token middleware retry and reauth", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, middleware, reauth, retries atomic.Int32
		client := cloud.Client("block-storage", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + f.prefix()
		client.MoreHeaders = map[string]string{"X-Source": "original"}
		client.Microversion = "3.15"
		transport := cloud.Provider.HTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		type contextKey struct{}
		cloud.Provider.HTTPClient.Transport = cinderMetadataRoundTrip(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			if r.Context().Value(contextKey{}) != "owned" {
				t.Error("caller context was replaced")
			}
			return transport.RoundTrip(r)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != cloud.Server.URL+f.endpoint("selected") || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			return nil
		}
		cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"metadata":{"owner":"fixed"}}` || r.Header.Get("X-Source") != "original" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.15" {
				t.Error(string(body), r.Header)
			}
			if n == 1 {
				if r.Header.Get("X-Auth-Token") != "live-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 401, `{}`)
				return
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Error("provider token was detached", r.Header)
			}
			if n == 2 {
				testcloud.JSON(w, 503, `{}`)
				return
			}
			testcloud.JSON(w, 200, `{"metadata":{"owner":"server"}}`)
		})
		s := cinderMetadataBind(t, f, client)
		client.ResourceBase = cloud.Server.URL + "/changed/"
		cloud.Provider.SetToken("live-token")
		v, err := s.Merge(context.WithValue(context.Background(), contextKey{}, "owned"), map[string]string{"owner": "fixed"})
		if err != nil || v == nil || v.Metadata["owner"] != "server" || calls.Load() != 3 || middleware.Load() != 3 || reauth.Load() != 1 || retries.Load() != 1 || s.RawClient() != client || client.ResourceBase != cloud.Server.URL+"/changed/" {
			t.Fatal(v, err, calls.Load(), middleware.Load(), reauth.Load(), retries.Load())
		}
	})
	t.Run("redirect cannot change fixed parent", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, wrong atomic.Int32
		cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Redirect(w, r, f.endpoint("other"), 307)
		})
		cloud.Mux.HandleFunc(f.endpoint("other"), func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); testcloud.JSON(w, 200, `{"metadata":{}}`) })
		_, err := cinderMetadataBind(t, f, cloud.Client("block-storage", f.prefix())).Replace(context.Background(), map[string]string{"owner": "fixed"})
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || wrong.Load() != 0 {
			t.Fatal(err, calls.Load(), wrong.Load())
		}
	})
	t.Run("cancellation and clean validation", func(t *testing.T) {
		cloud := testcloud.New(t)
		arrived := make(chan struct{}, 1)
		cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) { arrived <- struct{}{}; <-r.Context().Done() })
		client := cloud.Client("block-storage", f.prefix())
		s := cinderMetadataBind(t, f, client)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { _, err := s.Get(ctx); done <- err }()
		<-arrived
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if _, err := s.DeleteKeys(ctx, []string{}); !errors.Is(err, context.Canceled) {
			t.Fatal("empty delete skipped context validation", err)
		}
		client.Type = "compute"
		if _, err := s.DeleteKeys(context.Background(), []string{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("empty delete skipped source validation", err)
		}
		if _, err := f.bind(nil, client, resource.ID("selected")); err == nil {
			t.Fatal("nil context accepted")
		}
		if _, err := f.bind(context.Background(), nil, resource.ID("selected")); err == nil {
			t.Fatal("nil service client accepted")
		}
		var nilAPI *volumes3.API
		if _, err := nilAPI.MetadataIn(context.Background(), resource.ID("selected")); err == nil {
			t.Fatal("nil API accepted")
		}
		var nilScope *volumes3.MetadataScope
		if nilScope.ID() != "" || nilScope.RawClient() != nil {
			t.Fatal("nil getters")
		}
		if _, err := nilScope.Get(context.Background()); err == nil {
			t.Fatal("nil scope accepted")
		}
	})
}

func TestCinderMetadataScopesConcurrentReuseAndNativeSurfaceIsolation(t *testing.T) {
	for _, f := range cinderMetadataFixtures() {
		t.Run(f.name(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var body map[string]map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["metadata"]["owner"] != "input" || r.Header.Get("X-Reuse") != "frozen" {
					t.Error(body, err, r.Header)
				}
				testcloud.JSON(w, 200, `{"metadata":{"owner":"server"}}`)
			})
			singular := strings.TrimSuffix(f.collection, "s")
			cloud.Mux.HandleFunc(f.prefix()+f.collection+"/selected", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"id":"native-parent","name":"native","metadata":{"owner":"native"}}}`, singular))
			})
			client := cloud.Client("block-storage", f.prefix())
			s := cinderMetadataBind(t, f, client)
			headers := map[string]string{"X-Reuse": "frozen"}
			option := metadata.WithHeaders(headers)
			headers["X-Reuse"] = "caller-change"
			values := map[string]string{"owner": "input"}
			var wait sync.WaitGroup
			for range 4 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					v, err := s.Merge(context.Background(), values, option)
					if err != nil || v == nil || v.Metadata["owner"] != "server" {
						t.Error(v, err)
						return
					}
					v.Metadata["owner"] = "owned"
					v.Header.Set("X-Reuse", "owned")
					v.Body[0] = '!'
				}()
			}
			wait.Wait()
			if calls.Load() != 4 || values["owner"] != "input" || client.MoreHeaders != nil {
				t.Fatal(calls.Load(), values, client.MoreHeaders)
			}
			id, err := f.nativeGet(context.Background(), client, "selected")
			if err != nil || id != "native-parent" || s.ID() != "selected" {
				t.Fatal("metadata scope changed native parent getter", id, err, s.ID())
			}
		})
	}
	t.Run("native Snapshot map result remains broader", func(t *testing.T) {
		for _, f := range cinderMetadataFixtures() {
			if f.collection != "snapshots" {
				continue
			}
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(f.endpoint("selected"), func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"metadata":{"number":9007199254740993}}`)
			})
			client := cloud.Client("block-storage", f.prefix())
			var value map[string]any
			var err error
			if f.version == "v2" {
				value, err = snapshots2.New(client).UpdateMetadata(context.Background(), "selected", snapshots2.UpdateMetadataOpts{Metadata: map[string]any{"number": 1}})
			} else {
				value, err = snapshots3.New(client).UpdateMetadata(context.Background(), "selected", snapshots3.UpdateMetadataOpts{Metadata: map[string]any{"number": 1}})
			}
			if err != nil || value["number"] != json.Number("9007199254740993") {
				t.Fatal("native Snapshot ABI or number decoder changed", value, err)
			}
			owned, err := cinderMetadataBind(t, f, client).Replace(context.Background(), map[string]string{"number": "1"})
			var accepted *resource.ResponseError
			if owned != nil || !errors.As(err, &accepted) || string(accepted.Body) != `{"metadata":{"number":9007199254740993}}` {
				t.Fatal("owned strict string schema changed", owned, err)
			}
		}
	})
}

type cinderMetadataRoundTrip func(*http.Request) (*http.Response, error)

func (f cinderMetadataRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type cinderMetadataReadFailure struct {
	cause error
	read  bool
}

func (r *cinderMetadataReadFailure) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		return copy(p, `{"metadata":`), nil
	}
	return 0, r.cause
}
func (*cinderMetadataReadFailure) Close() error { return nil }
