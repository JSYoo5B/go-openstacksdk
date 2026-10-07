package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/tags"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const tagServerBase = "/compute/v2.1/project/servers/server-id/tags"

func serverTagsClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/compute/v2.1/project")
	client.Microversion = version
	return client
}

func serverTagsScope(t *testing.T, cloud *testcloud.Cloud) *tags.ServerTagScope {
	t.Helper()
	scope, err := tags.New(serverTagsClient(cloud, "2.26")).InServer(context.Background(), resource.ID("server-id"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func checkTagHeaders(t *testing.T, r *http.Request, version string) {
	t.Helper()
	if got := r.Header.Get("X-Auth-Token"); got != "test-token" {
		t.Errorf("auth token=%q", got)
	}
	if got := r.Header.Get("X-OpenStack-Nova-API-Version"); got != version {
		t.Errorf("Nova microversion=%q", got)
	}
	if got := r.Header.Get("OpenStack-API-Version"); got != "compute "+version {
		t.Errorf("OpenStack microversion=%q", got)
	}
	if r.URL.RawQuery != "" {
		t.Errorf("unexpected tag query=%q", r.URL.RawQuery)
	}
}

func TestServerTagsScopeResolvesServerOnceAndUsesNativeHTTPContracts(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, adds, deletes atomic.Int32
	cloud.Mux.HandleFunc("/compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if got := r.URL.Query().Get("name"); got != "^web\\.prod$" {
			t.Errorf("exact server name query=%q", got)
		}
		if r.Header.Get("X-OpenStack-Nova-API-Version") != "2.26" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("lookup headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"servers":[{"id":"server-id","name":"web.prod"},{"id":"wrong-id","name":"webXprod"}]}`)
	})
	cloud.Mux.HandleFunc(tagServerBase+"/managed", func(w http.ResponseWriter, r *http.Request) {
		checkTagHeaders(t, r, "2.26")
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("single-tag %s must have no body: %s", r.Method, body)
		}
		switch r.Method {
		case http.MethodPut:
			if adds.Add(1) == 1 {
				w.WriteHeader(201)
			} else {
				w.WriteHeader(204)
			}
		case http.MethodGet:
			w.WriteHeader(204)
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	cloud.Mux.HandleFunc(tagServerBase, func(w http.ResponseWriter, r *http.Request) {
		checkTagHeaders(t, r, "2.26")
		switch r.Method {
		case http.MethodGet:
			testcloud.JSON(w, 200, `{"tags":["managed","role=db"]}`)
		case http.MethodPut:
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 1 || string(body["tags"]) != `["managed","role=db"]` {
				t.Errorf("replacement body=%s", body)
			}
			testcloud.JSON(w, 200, `{"tags":["role=db","managed"]}`)
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	})
	ctx := context.Background()
	client := serverTagsClient(cloud, "2.26")
	beforeEndpoint, beforeVersion := client.Endpoint, client.Microversion
	scope, err := tags.New(client).InServer(ctx, resource.Name("web.prod"))
	if err != nil || scope.ServerID() != "server-id" {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	for range 2 {
		if err := scope.Add(ctx, "managed"); err != nil {
			t.Fatal(err)
		}
	}
	if present, err := scope.Check(ctx, "managed"); err != nil || !present {
		t.Fatalf("present=%v err=%v", present, err)
	}
	if values, err := scope.List(ctx); err != nil || !reflect.DeepEqual(values, []string{"managed", "role=db"}) {
		t.Fatalf("list=%v err=%v", values, err)
	}
	if values, err := scope.Replace(ctx, []string{"managed", "role=db"}); err != nil || !reflect.DeepEqual(values, []string{"role=db", "managed"}) {
		t.Fatalf("replace=%v err=%v", values, err)
	}
	if err := scope.Remove(ctx, "managed"); err != nil {
		t.Fatal(err)
	}
	if err := scope.RemoveAll(ctx); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 1 || adds.Load() != 2 || deletes.Load() != 2 || client.Endpoint != beforeEndpoint || client.Microversion != beforeVersion {
		t.Fatalf("lookup=%d add=%d delete=%d endpoint=%s microversion=%s", lookups.Load(), adds.Load(), deletes.Load(), client.Endpoint, client.Microversion)
	}
}

func TestServerTagsScopeEscapesAllowedUnicodeAndReservedCharacters(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var expected string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkTagHeaders(t, r, "2.26")
		escaped := url.PathEscape(expected)
		if expected == "." || expected == ".." {
			escaped = strings.Repeat("%2E", len(expected))
		}
		if r.RequestURI != tagServerBase+"/"+escaped || r.URL.Path != tagServerBase+"/"+expected {
			t.Errorf("tag=%q URI=%q path=%q", expected, r.RequestURI, r.URL.Path)
		}
		w.WriteHeader(204)
	})
	scope := serverTagsScope(t, cloud)
	ctx := context.Background()
	for _, value := range []string{"role=db", "two words", "?#%", "한글💾", `back\slash`, ".", "..", strings.Repeat("한", 60)} {
		expected = value
		if err := scope.Add(ctx, value); err != nil {
			t.Fatal(err)
		}
		if present, err := scope.Check(ctx, value); err != nil || !present {
			t.Fatalf("present=%v err=%v", present, err)
		}
		if err := scope.Remove(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 24 {
		t.Fatalf("unexpected requests or redirects: %d", calls.Load())
	}
}

func TestServerTagsScopeEmptyReplacementAndExplicitFalseExtension(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(tagServerBase, func(w http.ResponseWriter, r *http.Request) {
		checkTagHeaders(t, r, "2.26")
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"tags":[]}`)
			return
		}
		if r.Method != http.MethodPut {
			t.Errorf("method=%s", r.Method)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if string(body["tags"]) != "[]" {
			t.Errorf("empty replacement must be an array: %s", body)
		}
		if calls.Add(1) == 4 && string(body["vendor:enabled"]) != "false" {
			t.Errorf("explicit false extension lost: %s", body)
		}
		testcloud.JSON(w, 200, `{"tags":[]}`)
	})
	scope := serverTagsScope(t, cloud)
	ctx := context.Background()
	for _, input := range [][]string{nil, {}} {
		if values, err := scope.Replace(ctx, input); err != nil || values == nil || len(values) != 0 {
			t.Fatalf("replace=%v err=%v", values, err)
		}
	}
	if values, err := scope.Replace(ctx, []string{"original"}, tags.WithReplaceAllOptions(tags.ReplaceAllOpts{})); err != nil || values == nil || len(values) != 0 {
		t.Fatalf("replace selected empty concrete options=%v err=%v", values, err)
	}
	if _, err := scope.Replace(ctx, nil, tags.WithReplaceAllField("vendor:enabled", false)); err != nil {
		t.Fatal(err)
	}
	if values, err := scope.List(ctx); err != nil || values == nil || len(values) != 0 {
		t.Fatalf("empty list=%v err=%v", values, err)
	}
	if calls.Load() != 4 {
		t.Fatal(calls.Load())
	}
}

func TestServerTagsScopeMissingPoliciesPreserveHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var code atomic.Int32
	code.Store(404)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		checkTagHeaders(t, r, "2.26")
		testcloud.JSON(w, int(code.Load()), `{"itemNotFound":{"message":"tag or parent not found"}}`)
	})
	scope := serverTagsScope(t, cloud)
	ctx := context.Background()
	if present, err := scope.Check(ctx, "absent"); err != nil || present {
		t.Fatalf("present=%v err=%v", present, err)
	}
	if err := scope.Remove(ctx, "absent"); err != nil {
		t.Fatal(err)
	}
	if err := scope.RemoveAll(ctx); err != nil {
		t.Fatal(err)
	}
	strict := []tags.MissingOption{tags.WithIgnoreMissing(true), tags.WithIgnoreMissing(false)}
	operations := []struct {
		name string
		run  func() error
	}{
		{"Check", func() error { _, err := scope.Check(ctx, "absent", strict...); return err }},
		{"Remove", func() error { return scope.Remove(ctx, "absent", strict...) }},
		{"RemoveAll", func() error { return scope.RemoveAll(ctx, tags.WithMissingError()) }},
		{"Add", func() error { return scope.Add(ctx, "absent") }},
		{"List", func() error { _, err := scope.List(ctx); return err }},
		{"Replace", func() error { _, err := scope.Replace(ctx, nil); return err }},
	}
	for _, op := range operations {
		if err := op.run(); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
			t.Errorf("%s must preserve not-found and original HTTP cause: %v", op.name, err)
		}
	}
	if err := scope.Remove(ctx, "absent", tags.WithMissingError(), tags.WithIgnoreMissing(true)); err != nil {
		t.Fatal(err)
	}
	for _, status := range []int{403, 409, 500} {
		code.Store(int32(status))
		for _, op := range operations {
			if err := op.run(); !gophercloud.ResponseCodeIs(err, status) || errors.Is(err, resource.ErrNotFound) {
				t.Errorf("%s must preserve HTTP %d: %v", op.name, status, err)
			}
		}
		if present, err := scope.Check(ctx, "absent", tags.WithIgnoreMissing(true)); present || !gophercloud.ResponseCodeIs(err, status) {
			t.Fatalf("Check ignored HTTP %d: present=%v err=%v", status, present, err)
		}
		if err := scope.Remove(ctx, "absent", tags.WithIgnoreMissing(true)); !gophercloud.ResponseCodeIs(err, status) {
			t.Fatal(err)
		}
		if err := scope.RemoveAll(ctx, tags.WithIgnoreMissing(true)); !gophercloud.ResponseCodeIs(err, status) {
			t.Fatal(err)
		}
	}
}

func TestServerTagsScopePreflightRejectsInvalidInputsWithoutHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, "{}")
	})
	ctx := context.Background()
	api := tags.New(serverTagsClient(cloud, "2.26"))
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID(".."), resource.ID("bad/id"), resource.ID("bad%id"), resource.Name(" ")} {
		if _, err := api.InServer(ctx, ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("ref=%v err=%v", ref, err)
		}
	}
	scope, err := api.InServer(ctx, resource.ID("server-id"))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "a/b", "a,b", strings.Repeat("x", 61), strings.Repeat("한", 61), string([]byte{0xff})} {
		if err := scope.Add(ctx, value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
		if _, err := scope.Check(ctx, value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
		if err := scope.Remove(ctx, value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
		if _, err := scope.Replace(ctx, []string{value}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
	}
	if _, err := scope.Replace(ctx, make([]string, 51)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Check(ctx, "ok", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Remove(ctx, "ok", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.RemoveAll(ctx, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, option := range []tags.ReplaceAllOption{
		nil,
		tags.WithReplaceAllField("tags", []string{"overwrite"}),
		request.WithQuery[tags.ReplaceAllOpts]("tags", "one"),
		request.WithHeader[tags.ReplaceAllOpts]("X-Vendor", "value"),
		request.WithArgument[tags.ReplaceAllOpts]("ignored", false),
		tags.WithReplaceAllOptions(tags.ReplaceAllOpts{Tags: []string{"bad/tag"}}),
	} {
		if _, err := scope.Replace(ctx, nil, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	for _, run := range []func() error{
		func() error { _, err := api.InServer(canceled, resource.Name("web")); return err },
		func() error { return scope.Add(canceled, "ok") },
		func() error { _, err := scope.Check(canceled, "ok"); return err },
		func() error { _, err := scope.List(canceled); return err },
		func() error { _, err := scope.Replace(canceled, nil); return err },
		func() error { return scope.Remove(canceled, "ok") },
		func() error { return scope.RemoveAll(canceled) },
	} {
		if err := run(); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight errors performed %d HTTP requests", calls.Load())
	}
}

func TestServerTagsScopeRequiresActualMicroversionWithoutChangingClient(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var expectedVersion string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkTagHeaders(t, r, expectedVersion)
		w.WriteHeader(204)
	})
	ctx := context.Background()
	for _, version := range []string{"", "2.25", "2.9", "1.26", "3.26", "latest", "2", "2.-26", "2.26.extra", " 2.26", "2.999999999999999999999999999999"} {
		client := serverTagsClient(cloud, version)
		if _, err := tags.New(client).InServer(ctx, resource.Name("web")); err == nil || client.Microversion != version {
			t.Errorf("version=%q selected=%q err=%v", version, client.Microversion, err)
		}
	}
	if _, err := tags.New(nil).InServer(ctx, resource.ID("server-id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	for _, version := range []string{"2.26", "2.90", "2.100"} {
		expectedVersion = version
		client := serverTagsClient(cloud, version)
		scope, err := tags.New(client).InServer(ctx, resource.ID("server-id"))
		if err != nil || client.Microversion != version {
			t.Fatalf("version=%s err=%v", version, err)
		}
		if err := scope.Add(ctx, "ok"); err != nil || client.Microversion != version {
			t.Fatalf("version=%s err=%v", version, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestServerTagsScopePreservesInvalidHTTPStatusAndResponseErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
		run  func(*tags.ServerTagScope) error
	}{
		{"Add requires 201 or 204", 200, "{}", func(s *tags.ServerTagScope) error { return s.Add(context.Background(), "ok") }},
		{"Check requires 204", 200, "{}", func(s *tags.ServerTagScope) error { _, err := s.Check(context.Background(), "ok"); return err }},
		{"Remove requires 204", 200, "{}", func(s *tags.ServerTagScope) error { return s.Remove(context.Background(), "ok") }},
		{"RemoveAll requires 204", 200, "{}", func(s *tags.ServerTagScope) error { return s.RemoveAll(context.Background()) }},
		{"List requires 200", 204, "", func(s *tags.ServerTagScope) error { _, err := s.List(context.Background()); return err }},
		{"Replace requires 200", 204, "", func(s *tags.ServerTagScope) error { _, err := s.Replace(context.Background(), nil); return err }},
		{"List malformed JSON", 200, `{"tags":`, func(s *tags.ServerTagScope) error { _, err := s.List(context.Background()); return err }},
		{"Replace wrong response type", 200, `{"tags":42}`, func(s *tags.ServerTagScope) error { _, err := s.Replace(context.Background(), nil); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, tc.code, tc.body)
			})
			if err := tc.run(serverTagsScope(t, cloud)); err == nil {
				t.Fatal("invalid success code or response was accepted")
			}
		})
	}
}

func TestServerTagsScopeNameResolutionReportsMissingAndAmbiguousParents(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"missing", `{"servers":[]}`, resource.ErrNotFound},
		{"ambiguous", `{"servers":[{"id":"a","name":"web"},{"id":"b","name":"web"}]}`, resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/compute/v2.1/project/servers/detail" || r.URL.Query().Get("name") != "^web$" {
					t.Errorf("lookup=%s", r.URL)
				}
				testcloud.JSON(w, 200, tc.body)
			})
			if scope, err := tags.New(serverTagsClient(cloud, "2.26")).InServer(context.Background(), resource.Name("web")); scope != nil || !errors.Is(err, tc.want) || calls.Load() != 1 {
				t.Fatalf("scope=%v err=%v calls=%d", scope, err, calls.Load())
			}
		})
	}
}
