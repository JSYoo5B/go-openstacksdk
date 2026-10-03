package metadeftags

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type tagCoreTransport func(*http.Request) (*http.Response, error)

func (f tagCoreTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type tagCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *tagCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *tagCoreBody) Close() error               { b.closes++; return b.closeErr }

type tagCoreReader struct {
	data  string
	err   error
	after func()
}

func (r *tagCoreReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if r.after != nil {
		r.after()
		r.after = nil
	}
	if r.data == "" {
		return n, r.err
	}
	return n, nil
}
func tagCoreClient(transport tagCoreTransport) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "original", HTTPClient: http.Client{Transport: transport}}, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}
func tagCoreHTTP(req *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"original"}}, Body: body}
}
func tagCoreJSON(req *http.Request, status int, body string) *http.Response {
	return tagCoreHTTP(req, status, io.NopCloser(strings.NewReader(body)))
}
func tagCoreScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *NamespaceScope {
	t.Helper()
	scope, err := New(client).InNamespace(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
func tagCoreProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
		t.Fatalf("exact accepted proof missing: %v %+v", err, proof)
	}
	return proof
}

func TestMetadefTagsCoreFixedScopeAndRoutes(t *testing.T) {
	parent, name := "OS::空 白", " child::名字 "
	calls := 0
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		base := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(parent) + "/tags"
		child := base + "/" + url.PathEscape(name)
		if req.Header.Get("X-Auth-Token") != "original" {
			t.Fatal("live auth missing")
		}
		switch calls {
		case 1:
			if req.Method != "POST" || req.URL.EscapedPath() != child || req.Body != nil {
				t.Fatalf("bodyless child create: %s %s", req.Method, req.URL)
			}
			return tagCoreJSON(req, 201, `{"name":"passive","created_at":"literal date","unknown":9007199254740993}`), nil
		case 2:
			if req.Method != "GET" || req.URL.EscapedPath() != child || req.Body != nil {
				t.Fatal("fixed child GET")
			}
			return tagCoreJSON(req, 200, `{"name":"other","links":false}`), nil
		case 3, 4:
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			expected := name
			if calls == 4 {
				expected = "renamed"
			}
			if req.Method != "PUT" || req.URL.EscapedPath() != child || len(body) != 1 || body["name"] != expected {
				t.Fatalf("rename on current route: %v %s", body, req.URL)
			}
			return tagCoreJSON(req, 200, `{"name":"server-choice"}`), nil
		case 5, 6, 7:
			var body struct {
				Tags []map[string]string `json:"tags"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if req.Method != "POST" || req.URL.EscapedPath() != base || body.Tags == nil {
				t.Fatal("Set collection JSON")
			}
			if calls == 5 && (len(body.Tags) != 0 || req.Header.Get("X-Openstack-Append") != "False") {
				t.Fatal("nil Set must send [] and False")
			}
			if calls == 6 && (len(body.Tags) != 0 || req.Header.Get("X-Openstack-Append") != "False") {
				t.Fatal("empty Set must send [] and False")
			}
			if calls == 7 {
				want := []map[string]string{{"name": ""}, {"name": "line\n\x00"}, {"name": strings.Repeat("界", 81)}, {"name": "dup"}, {"name": "dup"}}
				if !reflect.DeepEqual(body.Tags, want) || req.Header.Get("X-Openstack-Append") != "True" {
					t.Fatalf("bulk literals lost: %+v", body.Tags)
				}
			}
			return tagCoreJSON(req, 201, `{"tags":[],"stored_state":"unknown"}`), nil
		case 8:
			if req.Method != "DELETE" || req.URL.EscapedPath() != child || req.Body != nil {
				t.Fatal("child DELETE")
			}
			return tagCoreJSON(req, 204, "opaque child"), nil
		case 9:
			if req.Method != "DELETE" || req.URL.EscapedPath() != base || req.Body != nil {
				t.Fatal("collection DELETE")
			}
			return tagCoreJSON(req, 204, "opaque bulk"), nil
		case 10:
			if req.Method != "GET" || req.URL.EscapedPath() != base || req.URL.RawQuery != "" {
				t.Fatal("default list one uncapped GET")
			}
			return tagCoreJSON(req, 200, `{"tags":[],"next":false}`), nil
		default:
			t.Fatalf("unexpected extra HTTP%d", calls)
			return nil, nil
		}
	})
	api := New(client)
	scope, err := api.InNamespace(context.Background(), parent)
	if err != nil || calls != 0 || scope.NamespaceName() != parent || scope.RawClient() != client || api.RawClient() != client {
		t.Fatalf("zero HTTP fixed scope: %v", err)
	}
	value, err := scope.Create(context.Background(), name)
	if err != nil || *value.Name != "passive" || *value.CreatedAt != "literal date" || string(value.Body["unknown"]) != "9007199254740993" {
		t.Fatalf("create result %v %+v", err, value)
	}
	if _, err = scope.Get(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name, WithUpdateName("renamed")); err != nil {
		t.Fatal(err)
	}
	for _, names := range [][]string{nil, {}, {"", "line\n\x00", strings.Repeat("界", 81), "dup", "dup"}} {
		var opts []SetOption
		if len(names) > 0 {
			opts = append(opts, WithSetAppend(true))
		}
		result, err := scope.Set(context.Background(), names, opts...)
		if err != nil || result.Tags == nil || len(result.Tags) != 0 || result.StatusCode != 201 {
			t.Fatalf("Set response %v %+v", err, result)
		}
	}
	ack, err := scope.Delete(context.Background(), name)
	if err != nil || ack.Namespace != parent || *ack.Name != name || string(ack.Body) != "opaque child" || ack.StatusCode != 204 {
		t.Fatalf("child ack %v %+v", err, ack)
	}
	ack, err = scope.DeleteAll(context.Background())
	if err != nil || ack.Name != nil || string(ack.Body) != "opaque bulk" {
		t.Fatalf("bulk ack %v %+v", err, ack)
	}
	values, err := scope.All(context.Background())
	if err != nil || values == nil || len(values) != 0 || calls != 10 {
		t.Fatalf("empty All %v %#v %d", err, values, calls)
	}
}

func TestMetadefTagsCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	raw := `{"name":"","created_at":"not a time","updated_at":null,"links":false,"self":{"url":"https://foreign.test"},"large":9007199254740993}`
	var value Tag
	if err := json.Unmarshal([]byte(raw), &value); err != nil || value.Name == nil || *value.Name != "" || *value.CreatedAt != "not a time" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["large"]) != "9007199254740993" {
		t.Fatalf("passive model %v %+v", err, value)
	}
	original := *value.Name
	value.Body["name"][1] = 'x'
	if *value.Name != original {
		t.Fatal("typed name aliases raw bytes")
	}
	for _, bad := range []string{`null`, `[]`, `{"name":1}`, `{"created_at":false}`, `{"updated_at":{}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'})} {
		var decoded Tag
		if err := json.Unmarshal([]byte(`{"name":"before"}`), &decoded); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(bad), &decoded); err == nil || *decoded.Name != "before" {
			t.Fatalf("atomic canonical decode %q: %v %+v", bad, err, decoded)
		}
	}
	setRaw := `{"tags":[{"name":"one","created_at":""},{"name":null}],"large":9007199254740993}`
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) { return tagCoreJSON(req, 201, setRaw), nil })
	result, err := tagCoreScope(t, client, "parent").Set(context.Background(), []string{"request"})
	if err != nil || len(result.Tags) != 2 || result.Tags[1].Name != nil || result.Tags[0].StatusCode != 201 || result.Tags[0].Header.Get("X-Proof") != "original" || *result.Tags[0].CreatedAt != "" {
		t.Fatalf("owned Set fields %v %+v", err, result)
	}
	result.Tags[0].Header.Set("X-Proof", "row")
	result.Tags[0].Body["name"][1] = 'x'
	if result.Header.Get("X-Proof") != "original" || string(result.Body["tags"]) != `[{"name":"one","created_at":""},{"name":null}]` || *result.Tags[0].Name != "one" {
		t.Fatal("root/row/typed/header alias")
	}
	for _, bad := range []string{`{}`, `{"tags":null}`, `{"tags":{}}`, `{"tags":[{"name":"valid"},null]}`, `{"tags":[{"name":"valid"},{"name":false}]}`, `{"tags":[],"unknown":"` + string([]byte{255}) + `"}`} {
		client := tagCoreClient(func(req *http.Request) (*http.Response, error) { return tagCoreJSON(req, 201, bad), nil })
		result, err := tagCoreScope(t, client, "parent").Set(context.Background(), nil)
		if result != nil || err == nil {
			t.Fatalf("malformed Set result %q %+v %v", bad, result, err)
		}
		tagCoreProof(t, err, 201, bad)
	}
}

func TestMetadefTagsCoreCompletePreflight(t *testing.T) {
	calls, callbacks := 0, 0
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return tagCoreJSON(req, 200, `{}`), nil })
	scope := tagCoreScope(t, client, "parent")
	for _, name := range []string{"", ".", "..", "slash/", "back\\", "%2f", "?", "#", "control\n", strings.Repeat("界", 81), string([]byte{255})} {
		_, err := scope.Create(context.Background(), name, func(*CreateOpts) error { callbacks++; return nil })
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("unsafe route %q %v", name, err)
		}
	}
	if _, err := scope.Set(context.Background(), []string{string([]byte{255})}, func(*SetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Get(nil, "child", func(*GetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancelled")
	cancel(cause)
	if _, err := scope.Delete(ctx, "child", func(*DeleteOpts) error { callbacks++; return nil }); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatalf("custom cancellation %v", err)
	}
	if callbacks != 0 || calls != 0 {
		t.Fatalf("preflight called callbacks/http %d/%d", callbacks, calls)
	}
	client.MoreHeaders = map[string]string{"x-OPENSTACK-append": "True"}
	operations := []func() error{
		func() error {
			_, e := scope.Create(context.Background(), "child", func(*CreateOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.Get(context.Background(), "child", func(*GetOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.Update(context.Background(), "child", func(*UpdateOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.Delete(context.Background(), "child", func(*DeleteOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.DeleteAll(context.Background(), func(*DeleteAllOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.Set(context.Background(), nil, func(*SetOpts) error { callbacks++; return nil })
			return e
		},
		func() error {
			_, e := scope.All(context.Background(), func(*ListOpts) error { callbacks++; return nil })
			return e
		},
	}
	for _, operation := range operations {
		if err := operation(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("source append header %v", err)
		}
	}
	if callbacks != 0 || calls != 0 {
		t.Fatalf("source protection late %d/%d", callbacks, calls)
	}
	client.MoreHeaders = nil
	sentinel := errors.New("callback")
	if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { callbacks++; return sentinel }); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), "child", WithUpdateName("../unsafe")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Set(context.Background(), nil, WithSetHeader("X-OpenStack-Append", "True")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), WithListLimit(-1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if callbacks != 1 || calls != 0 {
		t.Fatalf("option preflight %d/%d", callbacks, calls)
	}
	var nilAPI *API
	var nilScope *NamespaceScope
	if nilAPI.RawClient() != nil || nilScope.RawClient() != nil || nilScope.NamespaceName() != "" {
		t.Fatal("nil accessors")
	}
	if _, err := nilAPI.InNamespace(context.Background(), "parent"); err == nil {
		t.Fatal("nil API")
	}
	if _, err := nilScope.Get(context.Background(), "child"); err == nil {
		t.Fatal("nil scope")
	}
}

func TestMetadefTagsCorePagedRawMarkersAndConsumption(t *testing.T) {
	t.Run("raw names oversized pages latest headers", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		client = tagCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			q := req.URL.Query()
			if q.Get("limit") != "2" || q.Get("sort_key") != "name" || q.Get("sort_dir") != "asc" {
				t.Fatalf("fixed paging query %s", req.URL)
			}
			switch calls {
			case 1:
				if _, ok := q["marker"]; ok || req.Header.Get("X-Latest") != "one" {
					t.Fatal("initial page")
				}
				return tagCoreJSON(req, 200, `{"tags":[{"name":"a"},{"name":"b"},{"name":"raw/?%#tail"}],"next":"https://foreign.test"}`), nil
			case 2:
				if q.Get("marker") != "raw/?%#tail" || req.Header.Get("X-Latest") != "two" {
					t.Fatalf("raw marker/latest headers %s %v", req.URL, req.Header)
				}
				return tagCoreJSON(req, 200, `{"tags":[{"name":"final"}]}`), nil
			default:
				t.Fatal("short page did not stop")
				return nil, nil
			}
		})
		client.MoreHeaders = map[string]string{"X-Latest": "one"}
		scope := tagCoreScope(t, client, "parent")
		seq := scope.List(context.Background(), WithListLimit(2), WithListSortKey("name"), WithListSortDir("asc"))
		if calls != 0 {
			t.Fatal("List eager")
		}
		names := []string{}
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			names = append(names, *value.Name)
			*value.Name = "mutated"
			value.Body["name"][1] = 'x'
			client.MoreHeaders["X-Latest"] = "two"
		}
		if calls != 2 || !reflect.DeepEqual(names, []string{"a", "b", "raw/?%#tail", "final"}) {
			t.Fatalf("page consumption %v %d", names, calls)
		}
	})
	for _, caseName := range []string{"nil limit", "zero limit", "cap", "break"} {
		t.Run(caseName, func(t *testing.T) {
			calls := 0
			client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if caseName == "zero limit" {
					q := req.URL.Query()
					if q.Get("limit") != "0" || len(q["marker"]) != 1 || q.Get("marker") != "" || len(q["sort_key"]) != 1 {
						t.Fatalf("explicit zero/empty %s", req.URL)
					}
					return tagCoreJSON(req, 200, `{"tags":[]}`), nil
				}
				return tagCoreJSON(req, 200, `{"tags":[{"name":"one"},null],"next":false}`), nil
			})
			scope := tagCoreScope(t, client, "parent")
			var options []ListOption
			switch caseName {
			case "zero limit":
				options = []ListOption{WithListLimit(0), WithListMarker(""), WithListSortKey("")}
			case "cap":
				options = []ListOption{WithListLimit(1), WithListMaxItems(1)}
			case "break":
				options = []ListOption{WithListLimit(1)}
			}
			if caseName == "break" {
				for _, err := range scope.List(context.Background(), options...) {
					if err != nil {
						t.Fatal(err)
					}
					break
				}
			} else if caseName == "nil limit" {
				values, err := scope.All(context.Background(), WithListMaxItems(1))
				if err != nil || len(values) != 1 {
					t.Fatalf("cap nillimit %v", err)
				}
			} else {
				if _, err := scope.All(context.Background(), options...); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 {
				t.Fatalf("extra follow %d", calls)
			}
		})
	}
	for _, tail := range []string{`{}`, `{"name":null}`, `{"name":""}`, `{"name":"line\n"}`, `{"name":"repeat"}`} {
		t.Run(tail, func(t *testing.T) {
			calls, yielded := 0, 0
			raw := `{"tags":[` + tail + `]}`
			client := tagCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return tagCoreJSON(req, 200, raw), nil })
			for _, err := range tagCoreScope(t, client, "parent").List(context.Background(), WithListLimit(1), WithListMarker("repeat")) {
				if err != nil {
					tagCoreProof(t, err, 200, raw)
					if tail == `{"name":"repeat"}` && !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatal(err)
					}
					continue
				}
				yielded++
			}
			if calls != 1 || yielded != 1 {
				t.Fatalf("tail invalid should fail after rows, before HTTP %d/%d", calls, yielded)
			}
		})
	}
	t.Run("All later failure discards rows", func(t *testing.T) {
		calls := 0
		bad := `{"tags":[{"name":false}]}`
		client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return tagCoreJSON(req, 200, `{"tags":[{"name":"first"}]}`), nil
			}
			return tagCoreJSON(req, 200, bad), nil
		})
		values, err := tagCoreScope(t, client, "parent").All(context.Background(), WithListLimit(1))
		if values != nil || calls != 2 {
			t.Fatal("late error kept rows")
		}
		tagCoreProof(t, err, 200, bad)
	})
}

func TestMetadefTagsCoreAcceptedBodiesAndStrictMissingPolicy(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "Update", "Set", "Delete", "DeleteAll"} {
		t.Run(operation, func(t *testing.T) {
			readErr, closeErr, custom := errors.New("read"), errors.New("close"), errors.New("cancel cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			status := 200
			if operation == "Create" || operation == "Set" {
				status = 201
			}
			if operation == "Delete" || operation == "DeleteAll" {
				status = 204
			}
			raw := "partial opaque"
			body := &tagCoreBody{reader: &tagCoreReader{data: raw, err: readErr, after: func() { cancel(custom) }}, closeErr: closeErr}
			calls := 0
			client := tagCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return tagCoreHTTP(req, status, body), nil })
			scope := tagCoreScope(t, client, "parent")
			var err error
			var ack *Acknowledgement
			switch operation {
			case "Create":
				v, e := scope.Create(ctx, "child")
				err = e
				if v != nil {
					t.Fatal("partial typed create")
				}
			case "Get":
				v, e := scope.Get(ctx, "child")
				err = e
				if v != nil {
					t.Fatal("partial typed get")
				}
			case "Update":
				v, e := scope.Update(ctx, "child")
				err = e
				if v != nil {
					t.Fatal("partial typed update")
				}
			case "Set":
				v, e := scope.Set(ctx, nil)
				err = e
				if v != nil {
					t.Fatal("partial typed Set")
				}
			case "Delete":
				ack, err = scope.Delete(ctx, "child")
			case "DeleteAll":
				ack, err = scope.DeleteAll(ctx)
			}
			for _, cause := range []error{readErr, closeErr, custom, context.Canceled} {
				if !errors.Is(err, cause) {
					t.Fatalf("lost cause %v: %v", cause, err)
				}
			}
			proof := tagCoreProof(t, err, status, raw)
			if status == 204 {
				if ack == nil || ack.StatusCode != 204 || string(ack.Body) != raw || ack.Namespace != "parent" {
					t.Fatalf("ack absent %+v", ack)
				}
				ack.Body[0] = 'x'
				ack.Header.Set("X-Proof", "ack")
				if string(proof.Body) != raw || proof.Header.Get("X-Proof") != "original" {
					t.Fatal("ack aliases proof")
				}
			}
			if calls != 1 || body.closes != 1 {
				t.Fatalf("replay/close %d/%d", calls, body.closes)
			}
		})
	}
	for _, bulk := range []bool{false, true} {
		t.Run("strict404", func(t *testing.T) {
			retryCalls := 0
			client := tagCoreClient(func(req *http.Request) (*http.Response, error) { return tagCoreJSON(req, 404, `{"missing":true}`), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retryCalls++
				return errors.New("stop retry")
			}
			scope := tagCoreScope(t, client, "parent")
			var ack *Acknowledgement
			var err error
			if bulk {
				ack, err = scope.DeleteAll(context.Background())
			} else {
				ack, err = scope.Delete(context.Background(), "child")
			}
			if ack != nil || err == nil || !gophercloud.ResponseCodeIs(err, 404) || retryCalls != 1 {
				t.Fatalf("404 not strict/native %v %+v retries%d", err, ack, retryCalls)
			}
		})
	}
}

func TestMetadefTagsCoreLifetimeSourceAndNativeRetryBoundaries(t *testing.T) {
	t.Run("source changes before callbacks and accepted body", func(t *testing.T) {
		calls, callbacks := 0, 0
		var client *gophercloud.ServiceClient
		client = tagCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			body := &tagCoreBody{reader: &tagCoreReader{data: `{"name":"child"}`, err: io.EOF, after: func() { client.Endpoint = "https://changed.test/" }}}
			return tagCoreHTTP(req, 200, body), nil
		})
		scope := tagCoreScope(t, client, "parent")
		endpoint := client.Endpoint
		client.Endpoint = "https://changed.test/"
		if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 || calls != 0 {
			t.Fatalf("changed target %v %d/%d", err, callbacks, calls)
		}
		client.Endpoint = endpoint
		value, err := scope.Get(context.Background(), "child")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatalf("accepted source error %v %+v", err, value)
		}
		tagCoreProof(t, err, 200, `{"name":"child"}`)
	})
	t.Run("live auth normal retry remains owned", func(t *testing.T) {
		calls, retries := 0, 0
		client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Auth-Token") != "new-token" {
				t.Fatal("auth not live")
			}
			if calls == 1 {
				return tagCoreJSON(req, 503, `{"busy":true}`), nil
			}
			return tagCoreJSON(req, 201, `{"name":"server"}`), nil
		})
		scope := tagCoreScope(t, client, "parent")
		client.ProviderClient.TokenID = "new-token"
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		value, err := scope.Create(context.Background(), "child")
		if err != nil || *value.Name != "server" || calls != 2 || retries != 1 {
			t.Fatalf("prebody retry %v %d/%d", err, calls, retries)
		}
	})
	t.Run("retry response ownership and original status", func(t *testing.T) {
		for _, expand := range []bool{false, true} {
			calls := 0
			client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return tagCoreJSON(req, 503, `busy`), nil
				}
				return tagCoreJSON(req, 202, `opaque202`), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				if expand {
					o.OkCodes = append(o.OkCodes, 202)
				} else {
					o.KeepResponseBody = false
				}
				return nil
			}
			value, err := tagCoreScope(t, client, "parent").Create(context.Background(), "child")
			if value != nil || err == nil {
				t.Fatal("changed ownership accepted")
			}
			if expand {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != "opaque202" || calls != 2 {
					t.Fatalf("actual original status %v %+v %d", err, native, calls)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatalf("ownership changed %v %d", err, calls)
			}
		}
	})
	t.Run("postyield source header owned", func(t *testing.T) {
		calls := 0
		client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return tagCoreJSON(req, 200, `{"tags":[{"name":"one"}]}`), nil
		})
		scope := tagCoreScope(t, client, "parent")
		observed := false
		for _, err := range scope.List(context.Background(), WithListLimit(1)) {
			if err != nil {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				tagCoreProof(t, err, 200, `{"tags":[{"name":"one"}]}`)
				observed = true
				continue
			}
			client.MoreHeaders = map[string]string{"X-Openstack-Append": "True"}
		}
		if !observed || calls != 1 {
			t.Fatalf("owned target header changed between pages %v %d", observed, calls)
		}
	})
}
