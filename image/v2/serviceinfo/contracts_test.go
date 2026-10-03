package serviceinfo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	nativeimport "gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/image/v2/serviceinfo"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const infoPrefix = "/reverse/glance/v2/"

type infoTransport func(*http.Request) (*http.Response, error)

func (f infoTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type infoBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *infoBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type infoReadFailure struct {
	prefix []byte
	cause  error
}

func (b *infoReadFailure) Read(p []byte) (int, error) {
	n := copy(p, b.prefix)
	b.prefix = b.prefix[n:]
	return n, b.cause
}

func infoWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"Content-Type": {"application/json"}, "X-Request-Id": {"actual-info"},
	}, Body: body}
}

func infoClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + infoPrefix
	return client
}

func infoProof(t *testing.T, err error, raw string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-info" {
		t.Fatalf("accepted evidence lost: %v %#v", err, proof)
	}
	return proof
}

func TestServiceInfoDefaultRoutesAndEvidence(t *testing.T) {
	for _, mode := range []string{"stores", "details", "empty stores", "import"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := infoClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.10"
			cloud.Provider.SetToken("live-token")
			var calls atomic.Int32
			path, raw := infoPrefix+"info/stores", `{"stores":[{"id":"second"},{"id":"first","default":true},{"id":"second"}]}`
			if mode == "details" {
				path = infoPrefix + "info/stores/detail"
			} else if mode == "empty stores" {
				raw = `{"stores":[]}`
			} else if mode == "import" {
				path, raw = infoPrefix+"info/import", `{"import-methods":{"description":"available","type":"array","value":["future-method","glance-direct","future-method"]}}`
			}
			body := &infoBody{Reader: strings.NewReader(raw)}
			wire := infoWire(200, body)
			wire.Header.Set("Location", "https://foreign.invalid/not-a-resource")
			cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != path || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(r.Method, r.URL, r.Header, r.Body)
				}
				return wire, nil
			})
			api := serviceinfo.New(client)
			if api.RawClient() != client {
				t.Fatal("API changed original client")
			}
			if mode == "import" {
				v, err := api.GetImportInfo(context.Background(), serviceinfo.WithGetImportInfoHeader("X-Option", "ordinary"))
				if err != nil || v == nil || v.ImportMethods == nil || !reflect.DeepEqual(v.ImportMethods.Value, []string{"future-method", "glance-direct", "future-method"}) || v.StatusCode != 200 || v.Header.Get("X-Request-Id") != "actual-info" || string(v.Body["import-methods"]) == "" {
					t.Fatal(v, err)
				}
				wire.Header.Set("X-Request-Id", "wire-later")
				if v.Header.Get("X-Request-Id") != "actual-info" {
					t.Fatal("metadata aliases wire")
				}
			} else {
				v, err := api.AllStores(context.Background(), serviceinfo.WithListStoresDetails(mode == "details"), serviceinfo.WithListStoresHeader("X-Option", "ordinary"))
				if err != nil || v == nil || mode == "empty stores" && len(v) != 0 || mode != "empty stores" && (len(v) != 3 || v[0].ID != "second" || v[1].ID != "first" || v[2].ID != "second" || v[1].IsDefault == nil || !*v[1].IsDefault) {
					t.Fatal(v, err)
				}
				for _, store := range v {
					if store.StatusCode != 200 || store.Header.Get("X-Request-Id") != "actual-info" || store.Header.Get("Location") == "" {
						t.Fatal(store)
					}
				}
				if len(v) > 1 {
					v[0].Header.Set("X-Request-Id", "caller")
					wire.Header.Set("X-Request-Id", "wire-later")
					if v[1].Header.Get("X-Request-Id") != "actual-info" {
						t.Fatal("row metadata aliases another row/wire")
					}
				}
			}
			if calls.Load() != 1 || body.closes.Load() != 1 || client.MoreHeaders["X-Source"] != "captured" || client.ResourceBase != cloud.Server.URL+infoPrefix {
				t.Fatal(calls.Load(), body.closes.Load(), client)
			}
			for _, name := range []string{"Find", "Delete", "Wait", "GetStore", "Create", "Update"} {
				if _, exists := reflect.TypeOf(api).MethodByName(name); exists {
					t.Fatal("discovery invented capability", name)
				}
			}
		})
	}
}

func TestServiceInfoCanonicalModelsAndPresence(t *testing.T) {
	t.Run("store exact flags optional values and raw precision", func(t *testing.T) {
		cloud := testcloud.New(t)
		raw := `{"stores":[{"id":"fixed","ID":12,"description":"canonical","Description":12,"default":"false","is_default":true,"DEFAULT":12,"read-only":"true","Read-Only":12,"type":"rbd","TYPE":12,"weight":0,"WEIGHT":"decoy","properties":{"large":9223372036854775808,"fraction":1.000000000000000000001},"unknown":{"future":true}},{"id":null,"description":null,"default":null,"read-only":null,"type":null,"weight":null,"properties":null},{"id":"omitted"},{"id":"empty","default":false,"read-only":true,"type":"","weight":-3,"properties":{}}]}`
		cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) {
			return infoWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		v, err := serviceinfo.New(infoClient(cloud)).AllStores(context.Background())
		if err != nil || len(v) != 4 || v[0].ID != "fixed" || v[0].Description != "canonical" || v[0].IsDefault == nil || *v[0].IsDefault || v[0].ReadOnly == nil || !*v[0].ReadOnly || v[0].Type == nil || *v[0].Type != "rbd" || v[0].Weight == nil || *v[0].Weight != 0 || string(v[0].Properties["large"]) != "9223372036854775808" || string(v[0].Properties["fraction"]) != "1.000000000000000000001" || string(v[0].Body["ID"]) != "12" || string(v[0].Body["is_default"]) != "true" || string(v[0].Body["unknown"]) != `{"future":true}` {
			t.Fatal(v, err)
		}
		if v[1].ID != "" || v[1].Description != "" || v[1].IsDefault != nil || v[1].ReadOnly != nil || v[1].Type != nil || v[1].Weight != nil || v[1].Properties != nil || string(v[1].Body["default"]) != "null" || v[2].IsDefault != nil || v[2].Properties != nil || v[3].Properties == nil || len(v[3].Properties) != 0 || v[3].Weight == nil || *v[3].Weight != -3 {
			t.Fatal("presence collapsed", v)
		}
		if _, exists := v[2].Body["default"]; exists {
			t.Fatal("omission was synthesized")
		}
	})
	for _, raw := range []string{`{}`, `{"import-methods":null}`, `{"Import-Methods":17}`, `{"import":{"import-methods":{"value":["decoy"]}}}`, `{"import-methods":{}}`, `{"import-methods":{"description":null,"type":null,"value":null}}`, `{"import-methods":{"description":"available","Description":12,"type":"array","Type":12,"value":[],"Value":12,"future":{"large":9223372036854775808}}}`} {
		t.Run("import presence "+raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) {
				return infoWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			v, err := serviceinfo.New(infoClient(cloud)).GetImportInfo(context.Background())
			if err != nil || v == nil || v.StatusCode != 200 {
				t.Fatal(v, err)
			}
			if strings.Contains(raw, `"import-methods":{`) && !strings.HasPrefix(raw, `{"import":`) {
				if v.ImportMethods == nil {
					t.Fatal("present object lost", v)
				}
				if strings.Contains(raw, `"value":[]`) {
					if v.ImportMethods.Value == nil || len(v.ImportMethods.Value) != 0 || v.ImportMethods.Description != "available" || v.ImportMethods.Type != "array" || string(v.ImportMethods.Body["future"]) != `{"large":9223372036854775808}` || string(v.ImportMethods.Body["Value"]) != "12" {
						t.Fatal(v.ImportMethods)
					}
				} else if v.ImportMethods.Value != nil || v.ImportMethods.Description != "" || v.ImportMethods.Type != "" {
					t.Fatal("null or omission changed", v.ImportMethods)
				}
			} else if v.ImportMethods != nil {
				t.Fatal("unknown envelope/case variant unwrapped", v)
			}
		})
	}
	for _, raw := range []string{"", "null", "[]", `{"stores":null}`, `{"stores":{}}`, `{"stores":[null]}`, `{"stores":[1]}`, `{"stores":[{"id":1}]}`, `{"stores":[{"description":false}]}`, `{"stores":[{"default":"False"}]}`, `{"stores":[{"default":1}]}`, `{"stores":[{"read-only":[]}]}`, `{"stores":[{"type":1}]}`, `{"stores":[{"weight":"1"}]}`, `{"stores":[{"weight":1.5}]}`, `{"stores":[{"weight":9223372036854775808}]}`, `{"stores":[{"properties":[]}]}`, `{"stores":[]} {}`, string([]byte("{\"stores\":[],\"unknown\":\"\xff\"}")), string([]byte("{\"stores\":[{\"id\":\"\xff\"}]}"))} {
		t.Run("invalid stores "+raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			body := &infoBody{Reader: strings.NewReader(raw)}
			cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) { return infoWire(200, body), nil })
			v, err := serviceinfo.New(infoClient(cloud)).AllStores(context.Background())
			if v != nil || err == nil || body.closes.Load() != 1 {
				t.Fatal(v, err, body.closes.Load())
			}
			infoProof(t, err, raw)
		})
	}
	for _, raw := range []string{"null", "[]", `{"import-methods":1}`, `{"import-methods":{"description":1}}`, `{"import-methods":{"type":[]}}`, `{"import-methods":{"value":{}}}`, `{"import-methods":{"value":[1]}}`, string([]byte("{\"unknown\":\"\xff\"}"))} {
		t.Run("invalid import "+raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) {
				return infoWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			v, err := serviceinfo.New(infoClient(cloud)).GetImportInfo(context.Background())
			if v != nil || err == nil {
				t.Fatal(v, err)
			}
			infoProof(t, err, raw)
		})
	}
}

func TestServiceInfoCompletePreflight(t *testing.T) {
	for _, mode := range []string{"nil API", "nil client", "nil provider", "wrong type", "foreign base", "query base", "owned source header", "invalid source header", "nil context", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := infoClient(cloud)
			var calls, callbacks atomic.Int32
			cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return infoWire(500, http.NoBody), nil })
			ctx := context.Background()
			cause := errors.New("preflight cancellation")
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = cloud.Server.URL + infoPrefix + "?bad=base"
			case "owned source header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "override"}
			case "invalid source header":
				client.MoreHeaders = map[string]string{"X-Source": "bad\r\nheader"}
			case "nil context":
				ctx = nil
			case "canceled":
				parent, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = parent, context.Canceled
			}
			api := serviceinfo.New(client)
			if mode == "nil API" {
				api = nil
			}
			v, err := api.GetImportInfo(ctx, func(*request.Config[serviceinfo.GetImportInfoOpts]) error { callbacks.Add(1); return nil })
			if v != nil || !errors.Is(err, want) {
				t.Fatal(v, err)
			}
			stores, err := api.AllStores(ctx, func(*request.Config[serviceinfo.ListStoresOpts]) error { callbacks.Add(1); return nil })
			if stores != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != 0 || mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal(stores, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, mode := range []string{"nil option", "negative limit", "negative cap", "empty marker query", "details query", "base_path query", "paginated query", "headers query", "microversion query", "JSON field", "argument", "owned header", "bad header", "callback error", "callback provider", "callback type"} {
		t.Run("option "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := infoClient(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return infoWire(500, http.NoBody), nil })
			cause := errors.New("option failed")
			want := error(resource.ErrInvalidOption)
			var option serviceinfo.ListStoresOption
			switch mode {
			case "negative limit":
				option = serviceinfo.WithListStoresOptions(serviceinfo.ListStoresOpts{Limit: -1})
			case "negative cap":
				option = serviceinfo.WithListStoresMaxItems(-1)
			case "empty marker query":
				option = serviceinfo.WithListStoresQuery("marker", "")
			case "details query", "base_path query", "paginated query", "headers query", "microversion query":
				option = serviceinfo.WithListStoresQuery(strings.TrimSuffix(mode, " query"), "value")
			case "JSON field":
				option = request.WithField[serviceinfo.ListStoresOpts]("unexpected", true)
			case "argument":
				option = request.WithArgument[serviceinfo.ListStoresOpts]("microversion", "2.10")
			case "owned header":
				option = serviceinfo.WithListStoresHeader("Authorization", "override")
			case "bad header":
				option = serviceinfo.WithListStoresHeader("X-Bad", "bad\nvalue")
			case "callback error", "callback provider", "callback type":
				option = func(*request.Config[serviceinfo.ListStoresOpts]) error {
					if mode == "callback error" {
						return cause
					}
					if mode == "callback provider" {
						client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
					} else {
						client.Type = "compute"
					}
					return nil
				}
				if mode == "callback error" {
					want = cause
				} else if mode == "callback type" {
					want = resource.ErrUnsupported
				}
			}
			v, err := serviceinfo.New(client).AllStores(context.Background(), option)
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, option := range []serviceinfo.GetImportInfoOption{nil, request.WithField[serviceinfo.GetImportInfoOpts]("unexpected", true), request.WithQuery[serviceinfo.GetImportInfoOpts]("id", "not-supported"), request.WithArgument[serviceinfo.GetImportInfoOpts]("microversion", "2.10"), serviceinfo.WithGetImportInfoHeader("X-Auth-Token", "override")} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return infoWire(500, http.NoBody), nil })
		v, err := serviceinfo.New(infoClient(cloud)).GetImportInfo(context.Background(), option)
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(v, err, calls.Load())
		}
	}
}

func TestServiceInfoPreparedOptionsAndLazyParallelReuse(t *testing.T) {
	t.Run("retained configs and caller slice cannot change prepared policy", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := infoClient(cloud)
		paginated := false
		bulk := serviceinfo.WithListStoresOptions(serviceinfo.ListStoresOpts{Details: true, Paginated: &paginated})
		paginated = true
		var retained *request.Config[serviceinfo.ListStoresOpts]
		var calls, first, second atomic.Int32
		options := []serviceinfo.ListStoresOption{bulk, func(c *request.Config[serviceinfo.ListStoresOpts]) error {
			first.Add(1)
			c.Query.Set("vendor:mode", "original")
			c.Headers["X-Option"] = "original"
			retained = c
			return nil
		}, func(c *request.Config[serviceinfo.ListStoresOpts]) error {
			second.Add(1)
			retained.Options.Details, *retained.Options.Paginated = false, true
			retained.Query["vendor:mode"][0] = "changed"
			retained.Headers["X-Option"] = "changed"
			if !c.Options.Details || c.Options.Paginated == nil || *c.Options.Paginated || c.Query.Get("vendor:mode") != "original" || c.Headers["X-Option"] != "original" {
				t.Error("previous retained callback changed next config", c)
			}
			return nil
		}}
		api := serviceinfo.New(client)
		list := api.ListStores(context.Background(), options...)
		options[1] = nil
		if calls.Load() != 0 || first.Load() != 0 || second.Load() != 0 {
			t.Fatal("list was not lazy")
		}
		cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			retained.Query["vendor:mode"][0], retained.Headers["X-Option"] = "late", "late"
			if r.URL.Path != infoPrefix+"info/stores/detail" || r.URL.Query().Get("vendor:mode") != "original" || r.URL.Query().Has("details") || r.URL.Query().Has("paginated") || r.Header.Get("X-Option") != "original" {
				t.Error(r.URL, r.Header)
			}
			return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"fixed"}],"next":"https://foreign.invalid/unused"}`))), nil
		})
		for iteration := 0; iteration < 2; iteration++ {
			count := 0
			for store, err := range list {
				if err != nil || store == nil || store.ID != "fixed" {
					t.Fatal(store, err)
				}
				count++
			}
			if count != 1 {
				t.Fatal(count)
			}
		}
		if calls.Load() != 2 || first.Load() != 2 || second.Load() != 2 {
			t.Fatal(calls.Load(), first.Load(), second.Load())
		}
	})
	t.Run("standard options and list can be reused in parallel", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, callbacks atomic.Int32
		cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.Path != infoPrefix+"info/stores" || r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("name") != "wire-only" || r.Header.Get("X-Option") != "ordinary" {
				t.Error(r.URL, r.Header)
			}
			return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"unfiltered","name":"different"},{"id":12}],"next":"https://foreign.invalid/unused"}`))), nil
		})
		options := []serviceinfo.ListStoresOption{serviceinfo.WithListStoresDetails(true), serviceinfo.WithListStoresOptions(serviceinfo.ListStoresOpts{MaxItems: 1}), serviceinfo.WithListStoresQuery("name", "wire-only"), serviceinfo.WithListStoresHeader("X-Option", "ordinary"), func(*request.Config[serviceinfo.ListStoresOpts]) error { callbacks.Add(1); return nil }}
		list := serviceinfo.New(infoClient(cloud)).ListStores(context.Background(), options...)
		var workers sync.WaitGroup
		for i := 0; i < 4; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				count := 0
				for store, err := range list {
					if err != nil || store == nil || store.ID != "unfiltered" || string(store.Body["name"]) != `"different"` {
						t.Error(store, err)
					}
					count++
				}
				if count != 1 {
					t.Error(count)
				}
			}()
		}
		workers.Wait()
		if calls.Load() != 4 || callbacks.Load() != 4 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
}

func TestServiceInfoPaginationCapsAndRawMarkers(t *testing.T) {
	t.Run("short-page fallback uses owned literal raw ID", func(t *testing.T) {
		cloud := testcloud.New(t)
		id := "wire%literal /?#"
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if r.URL.Path != infoPrefix+"info/stores/detail" || r.URL.Query().Get("limit") != "3" || r.URL.Query().Get("vendor") != "original" {
				t.Error(r.URL)
			}
			if n == 1 {
				return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"wire%literal /?#","ID":"decoy"}]}`))), nil
			}
			if n != 2 || r.URL.Query().Get("marker") != id {
				t.Error("marker came from caller-mutated model", r.URL, n)
			}
			return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[],"next":"https://foreign.invalid/ignored-empty-page"}`))), nil
		})
		count := 0
		for store, err := range serviceinfo.New(infoClient(cloud)).ListStores(context.Background(), serviceinfo.WithListStoresOptions(serviceinfo.ListStoresOpts{Details: true, Limit: 3}), serviceinfo.WithListStoresQuery("vendor", "original")) {
			if err != nil || store == nil || store.ID != id {
				t.Fatal(store, err)
			}
			store.ID, store.Body["id"] = "caller", json.RawMessage(`"caller"`)
			count++
		}
		if count != 1 || calls.Load() != 2 {
			t.Fatal(count, calls.Load())
		}
	})
	for _, representation := range []string{"next", "links", "stores_links", "HTTP Link"} {
		t.Run("advertised "+representation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.URL.Path != infoPrefix+"info/stores" || r.URL.Query().Get("vendor") != "a&b" {
					t.Error(r.URL)
				}
				if n == 2 {
					if r.URL.Query().Get("marker") != "second" {
						t.Error(r.URL)
					}
					return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"second"}]}`))), nil
				}
				raw := `{"stores":[{"id":"first"}]`
				if representation == "next" {
					raw += `,"next":"?marker=second"`
				} else if representation != "HTTP Link" {
					raw += fmt.Sprintf(`,%q:[{"rel":"next","href":"?marker=second"}]`, representation)
				}
				wire := infoWire(200, io.NopCloser(strings.NewReader(raw+"}")))
				if representation == "HTTP Link" {
					wire.Header.Set("Link", `<?marker=second>; rel="next"`)
				}
				return wire, nil
			})
			v, err := serviceinfo.New(infoClient(cloud)).AllStores(context.Background(), serviceinfo.WithListStoresQuery("vendor", "a&b"))
			if err != nil || len(v) != 2 || v[0].ID != "first" || v[1].ID != "second" || calls.Load() != 2 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, mode := range []string{"foreign origin", "path switch", "version-root prefix", "changed filter", "conflicting links", "cycle", "malformed unused row", "first page only", "later malformed row"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			var options []serviceinfo.ListStoresOption
			raw := `{"stores":[{"id":"first"}],"next":"?marker=second"}`
			switch mode {
			case "foreign origin":
				raw = `{"stores":[{"id":"first"}],"next":"https://foreign.invalid/info/stores"}`
			case "path switch":
				raw = `{"stores":[{"id":"first"}],"next":"` + infoPrefix + `info/stores/detail?marker=second"}`
			case "version-root prefix":
				raw = `{"stores":[{"id":"first"}],"next":"/v2/info/stores?marker=second"}`
			case "changed filter":
				raw = `{"stores":[{"id":"first"}],"next":"?marker=second&vendor=changed"}`
				options = []serviceinfo.ListStoresOption{serviceinfo.WithListStoresQuery("vendor", "original")}
			case "conflicting links":
				raw = `{"stores":[{"id":"first"}],"next":"?marker=second","links":[{"rel":"next","href":"?marker=other"}]}`
			case "cycle":
				raw = `{"stores":[{"id":"first"}],"next":"?marker=repeat"}`
			case "malformed unused row":
				raw = `{"stores":[{"id":"first"},{"id":12}],"next":"https://foreign.invalid/unused"}`
				options = []serviceinfo.ListStoresOption{serviceinfo.WithListStoresMaxItems(1)}
			case "first page only":
				raw = `{"stores":[{"id":"first"}],"next":"https://foreign.invalid/unused"}`
				options = []serviceinfo.ListStoresOption{serviceinfo.WithListStoresPaginated(false)}
			}
			cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if !strings.HasPrefix(r.URL.Path, infoPrefix) {
					t.Error("unsafe continuation reached HTTP", r.URL)
				}
				if mode == "malformed unused row" && r.URL.Query().Get("limit") != "1" {
					t.Error("maxitems hint absent", r.URL)
				}
				if mode == "later malformed row" && n%2 == 0 {
					return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":12}]}`))), nil
				}
				return infoWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			api := serviceinfo.New(infoClient(cloud))
			v, err := api.AllStores(context.Background(), options...)
			if mode == "malformed unused row" || mode == "first page only" {
				if err != nil || len(v) != 1 || v[0].ID != "first" || calls.Load() != 1 {
					t.Fatal(v, err, calls.Load())
				}
				return
			}
			if v != nil || err == nil || mode != "cycle" && mode != "later malformed row" && calls.Load() != 1 || mode == "cycle" && (!errors.Is(err, resource.ErrPaginationCycle) || calls.Load() != 2) || mode == "later malformed row" && calls.Load() != 2 {
				t.Fatal(v, err, calls.Load())
			}
			if mode == "later malformed row" {
				infoProof(t, err, `{"stores":[{"id":12}]}`)
				count := 0
				for store, err := range api.ListStores(context.Background()) {
					if err != nil {
						infoProof(t, err, `{"stores":[{"id":12}]}`)
						break
					}
					if store.ID != "first" {
						t.Fatal(store)
					}
					count++
				}
				if count != 1 {
					t.Fatal("List did not retain prior yielded row", count)
				}
			} else {
				infoProof(t, err, raw)
			}
		})
	}
}

func TestServiceInfoAcceptedFailuresAndNativeCompatibility(t *testing.T) {
	for _, phase := range []string{"stores", "import"} {
		for _, mode := range []string{"read", "close", "read close cancel"} {
			t.Run(phase+" "+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("info read failed"), errors.New("info close failed"), errors.New("info custom cancellation")
				raw := `{"stores":[{"id":"fixed"}]}`
				if phase == "import" {
					raw = `{"import-methods":{"value":["future"]}}`
				}
				body := &infoBody{Reader: strings.NewReader(raw)}
				if strings.Contains(mode, "read") {
					raw = raw[:8]
					body.Reader = &infoReadFailure{prefix: []byte(raw), cause: readCause}
				}
				if strings.Contains(mode, "close") {
					body.closeErr = closeCause
				}
				if strings.Contains(mode, "cancel") {
					body.onClose = func() { cancel(cancelCause) }
				}
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted replay")
				}
				cloud.Provider.HTTPClient.Transport = infoTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return infoWire(200, body), nil })
				api := serviceinfo.New(infoClient(cloud))
				var err error
				if phase == "stores" {
					v, failure := api.AllStores(ctx)
					if v != nil {
						t.Fatal(v)
					}
					err = failure
				} else {
					v, failure := api.GetImportInfo(ctx)
					if v != nil {
						t.Fatal(v)
					}
					err = failure
				}
				proof := infoProof(t, err, raw)
				if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(proof, err, calls.Load(), retries.Load(), body.closes.Load())
				}
			})
		}
	}
	for _, code := range []int{202, 204, 300, 403, 404, 500} {
		for _, phase := range []string{"details", "import"} {
			t.Run(fmt.Sprintf("%s rejects%d", phase, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				body := &infoBody{Reader: strings.NewReader("native error proof")}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					want := infoPrefix + "info/stores/detail"
					if phase == "import" {
						want = infoPrefix + "info/import"
					}
					if r.URL.Path != want {
						t.Error("fallback occurred", r.URL)
					}
					return infoWire(code, body), nil
				})
				api := serviceinfo.New(infoClient(cloud))
				var err error
				if phase == "details" {
					v, failure := api.AllStores(context.Background(), serviceinfo.WithListStoresDetails(true))
					if v != nil {
						t.Fatal(v)
					}
					err = failure
				} else {
					v, failure := api.GetImportInfo(context.Background())
					if v != nil {
						t.Fatal(v)
					}
					err = failure
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "native error proof" || native.ResponseHeader.Get("X-Request-Id") != "actual-info" || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(err, native, calls.Load(), body.closes.Load())
				}
			})
		}
	}
	t.Run("native ImageImport.Get ABI and route remain", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodGet || r.URL.Path != infoPrefix+"info/import" || r.URL.RawQuery != "" {
				t.Error(r.Method, r.URL)
			}
			return infoWire(200, io.NopCloser(strings.NewReader(`{"import-methods":{"description":"native","type":"array","value":["web-download"]}}`))), nil
		})
		v, err := nativeimport.New(infoClient(cloud)).Get(context.Background())
		if err != nil || v == nil || v.ImportMethods.Description != "native" || !reflect.DeepEqual(v.ImportMethods.Value, []string{"web-download"}) || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
}

func TestServiceInfoProviderSourceAndRetryBoundaries(t *testing.T) {
	for _, change := range []string{"valid prefix and headers", "provider", "type"} {
		t.Run("source changes between pages "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := infoClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.URL.Path != infoPrefix+"info/stores" || r.Header.Get("X-Source") != "captured" {
					t.Error("source change retargeted request", r.URL, r.Header)
				}
				if n == 2 {
					if r.Header.Get("X-Auth-Token") != "after-page" {
						t.Error(r.Header)
					}
					return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"second"}]}`))), nil
				}
				switch change {
				case "valid prefix and headers":
					client.ResourceBase = cloud.Server.URL + "/later/v2/"
					client.MoreHeaders = map[string]string{"X-Source": "later"}
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
				case "type":
					client.Type = "compute"
				}
				cloud.Provider.SetToken("after-page")
				return infoWire(200, io.NopCloser(strings.NewReader(`{"stores":[{"id":"first"}],"next":"?marker=second"}`))), nil
			})
			v, err := serviceinfo.New(client).AllStores(context.Background())
			if change == "valid prefix and headers" {
				if err != nil || len(v) != 2 || calls.Load() != 2 {
					t.Fatal(v, err, calls.Load())
				}
			} else {
				want := error(resource.ErrInvalidOption)
				if change == "type" {
					want = resource.ErrUnsupported
				}
				if v != nil || !errors.Is(err, want) || calls.Load() != 1 {
					t.Fatal(v, err, calls.Load())
				}
			}
		})
	}
	t.Run("configured prebody hooks use live auth and fixed singleton", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, reauth, backoff, retry atomic.Int32
		var bodies []*infoBody
		cloud.Provider.SetToken("initial")
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauth"); return nil }
		cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			cloud.Provider.SetToken("backoff")
			return nil
		}
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retry.Add(1)
			if method != http.MethodGet || target != cloud.Server.URL+infoPrefix+"info/import" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			cloud.Provider.SetToken("retry")
			return nil
		}
		cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
			n := int(calls.Add(1)) - 1
			codes, tokens := []int{401, 429, 503, 200}, []string{"initial", "reauth", "backoff", "retry"}
			if n >= len(codes) {
				return nil, errors.New("unexpected replay")
			}
			if r.Method != http.MethodGet || r.URL.Path != infoPrefix+"info/import" || r.Body != nil || r.Header.Get("X-Auth-Token") != tokens[n] {
				t.Error(r.Method, r.URL, r.Header, r.Body)
			}
			body := &infoBody{Reader: strings.NewReader(`{"import-methods":{"value":[]}}`)}
			bodies = append(bodies, body)
			return infoWire(codes[n], body), nil
		})
		v, err := serviceinfo.New(infoClient(cloud)).GetImportInfo(context.Background())
		if err != nil || v == nil || v.ImportMethods == nil || v.ImportMethods.Value == nil || calls.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retry.Load() != 1 {
			t.Fatal(v, err, calls.Load(), reauth.Load(), backoff.Load(), retry.Load())
		}
		for _, body := range bodies {
			if body.closes.Load() != 1 {
				t.Fatal(body.closes.Load())
			}
		}
	})
	for _, change := range []string{"KeepResponseBody", "JSONResponse", "JSONBody", "RawBody", "OkCodes"} {
		t.Run("shared retry guard "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, callbacks atomic.Int32
			var bodies []*infoBody
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks.Add(1)
				switch change {
				case "KeepResponseBody":
					options.KeepResponseBody = false
				case "JSONResponse":
					options.JSONResponse = new(any)
				case "JSONBody":
					options.JSONBody = json.RawMessage("null")
				case "RawBody":
					options.RawBody = strings.NewReader("unexpected")
				case "OkCodes":
					options.OkCodes = []int{200, 202}
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = infoTransport(func(r *http.Request) (*http.Response, error) {
				if r.Body != nil || r.URL.Path != infoPrefix+"info/import" {
					t.Error(r.Body, r.URL)
				}
				code := 503
				if calls.Add(1) == 2 {
					code = 202
				}
				body := &infoBody{Reader: strings.NewReader("native prebody evidence")}
				bodies = append(bodies, body)
				return infoWire(code, body), nil
			})
			v, err := serviceinfo.New(infoClient(cloud)).GetImportInfo(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			wantCode, wantCalls := 503, int32(1)
			if change == "OkCodes" {
				wantCode, wantCalls = 202, 2
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("shared ownership guard bypassed", err)
			}
			if v != nil || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != "native prebody evidence" || calls.Load() != wantCalls || callbacks.Load() != 1 {
				t.Fatal(v, err, native, calls.Load(), callbacks.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
}
