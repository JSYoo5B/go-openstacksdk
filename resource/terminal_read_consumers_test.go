package resource_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute/v2/servers"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// These are consumers of the shared classification policy, including a real
// native SDK binding. Classifier implementation and synthetic cause tables have
// independent internal tests; no DELETE missing policy is changed by this unit.
func TestTerminalReadConsumersOrdinaryHTTP404RemainsMissing(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	const body = `{"itemNotFound":{"message":"actual missing resource"}}`
	cloud.Mux.HandleFunc("GET /reverse/v2/servers/missing", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		w.Header().Set("X-Read", "native404")
		testcloud.JSON(w, 404, body)
	})
	cloud.Mux.HandleFunc("GET /reverse/v2/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") != "^missing$" {
			t.Error("native name query changed", r.URL)
		}
		testcloud.JSON(w, 200, `{"servers":[]}`)
	})
	collection := servers.New(cloud.Client("compute", "/reverse/v2/")).Resources
	value, err := collection.Get(context.Background(), "missing")
	var missing *resource.NotFoundError
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &missing) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Read") != "native404" {
		t.Fatal("ordinary 404 lost missing or native evidence", value, err, missing, native)
	}
	value, err = collection.Find(context.Background(), resource.ID("missing"), resource.WithIgnoreMissing())
	if value != nil || err != nil {
		t.Fatal("explicit ignore missing changed", value, err)
	}
	var progress atomic.Int32
	if err := collection.WaitDeleted(context.Background(), resource.ID("missing"), resource.WithProgressCallback(func(int) { progress.Add(1) })); err != nil || progress.Load() != 0 {
		t.Fatal("ordinary missing no longer terminates delete wait", err, progress.Load())
	}
	value, err = collection.FindIdentity(context.Background(), "missing")
	if value != nil || err != nil || lists.Load() != 1 {
		t.Fatal("compatible missing lookup did not finish its empty list", value, err, lists.Load())
	}
	for _, query := range []bool{false, true} {
		options := []resource.IdentityFindOption{resource.WithIdentityFindFallback(resource.FindFallbackNever)}
		if query {
			options = append(options, resource.WithIdentityFindQuery("project_id", "chosen-project"))
		}
		value, err = collection.FindIdentity(context.Background(), "missing", options...)
		if value != nil || err != nil || lists.Load() != 1 {
			t.Fatal("GET-only ordinary missing no longer respects default ignore", value, err, lists.Load())
		}
		value, err = collection.FindIdentity(context.Background(), "missing", append(options, resource.WithIdentityFindIgnoreMissing(false))...)
		if value != nil || !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Read") != "native404" || lists.Load() != 1 {
			t.Fatal("strict missing no longer retains direct/query native cause", value, err, native, lists.Load())
		}
	}
	if gets.Load() != 8 {
		t.Fatal("ordinary missing made extra requests", gets.Load(), lists.Load())
	}
}

func TestTerminalReadConsumersNativeTransport404IsNeverMissing(t *testing.T) {
	for _, prewrapped := range []bool{false, true} {
		t.Run(fmt.Sprintf("source-not-found-wrapper=%t", prewrapped), func(t *testing.T) {
			cloud := testcloud.New(t)
			var attempts, lists atomic.Int32
			native := &gophercloud.ErrUnexpectedResponseCode{Method: "GET", Actual: 404, Body: []byte(`{"transport":"not a resource response"}`), ResponseHeader: http.Header{"X-Transport": {"original"}}}
			var cause error = native
			var originalMissing *resource.NotFoundError
			if prewrapped {
				originalMissing = &resource.NotFoundError{Resource: "source", Reference: "fixed", Cause: native}
				cause = originalMissing
			}
			transportFailure := &url.Error{Op: "GET", URL: "http://transport.invalid/original", Err: cause}
			cloud.Provider.HTTPClient.Transport = terminalConsumerRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/reverse/v2/servers/detail" {
					lists.Add(1)
					return nil, errors.New("fallback must not run")
				}
				attempts.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/reverse/v2/servers/fixed" {
					t.Error("native GET route changed", r.Method, r.URL)
				}
				if r.URL.RawQuery != "" && !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) {
					t.Error("query hook changed caller values", r.URL)
				}
				return nil, transportFailure
			})
			collection := servers.New(cloud.Client("compute", "/reverse/v2/")).Resources
			check := func(value *servers.Server, err error) {
				t.Helper()
				var operation *resource.OperationError
				var wrappedTransport *url.Error
				if value != nil || err == nil || !errors.Is(err, transportFailure) || !errors.Is(err, native) || !errors.As(err, &operation) || !errors.As(err, &wrappedTransport) || lists.Load() != 0 {
					t.Fatal("terminal transport was suppressed or replaced", value, err, operation, wrappedTransport, attempts.Load(), lists.Load())
				}
				if !prewrapped && errors.Is(err, resource.ErrNotFound) {
					t.Fatal("transport 404 acquired resource missing classification", err)
				}
				if prewrapped {
					var preserved *resource.NotFoundError
					if !errors.As(err, &preserved) || preserved != originalMissing {
						t.Fatal("original source missing cause was replaced", err, preserved)
					}
				}
			}
			value, err := collection.Get(context.Background(), "fixed")
			check(value, err)
			value, err = collection.Find(context.Background(), resource.ID("fixed"), resource.WithIgnoreMissing())
			check(value, err)
			var progress atomic.Int32
			err = collection.WaitDeleted(context.Background(), resource.ID("fixed"), resource.WithProgressCallback(func(int) { progress.Add(1) }))
			check(nil, err)
			if progress.Load() != 0 {
				t.Fatal("failed read invoked deletion progress", progress.Load())
			}
			for _, query := range []bool{false, true} {
				for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
					options := []resource.IdentityFindOption{resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(true)}
					if query {
						options = append(options, resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"tag": {"first", "second"}}}))
						// WithIdentityFindOptions is a full replacement. Restore the
						// selected fallback after adding the frozen bulk query.
						options = append(options, resource.WithIdentityFindFallback(policy))
					}
					value, err = collection.FindIdentity(context.Background(), "fixed", options...)
					check(value, err)
				}
			}
			if attempts.Load() != 9 || lists.Load() != 0 {
				t.Fatal("terminal read retried, listed, or failed to call native GET", attempts.Load(), lists.Load())
			}
		})
	}
}

type terminalConsumerItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func terminalConsumerAdapter(get func(context.Context, string, url.Values) (*terminalConsumerItem, error), lists *atomic.Int32) resource.Adapter[terminalConsumerItem] {
	return resource.Adapter[terminalConsumerItem]{
		Kind: "terminal-items", IdentityFind: true,
		Get:              func(ctx context.Context, id string) (*terminalConsumerItem, error) { return get(ctx, id, nil) },
		GetIdentityQuery: get,
		ID:               func(v *terminalConsumerItem) string { return v.ID },
		Name:             func(v *terminalConsumerItem) string { return v.Name },
		NameQuery:        func(name string) string { return name },
		Iterate: func(context.Context, url.Values) iter.Seq2[*terminalConsumerItem, error] {
			return func(yield func(*terminalConsumerItem, error) bool) {
				lists.Add(1)
				yield(&terminalConsumerItem{ID: "fixed", Name: "fixed"}, nil)
			}
		},
	}
}

func TestTerminalReadConsumersAcceptedDecodeEvidenceWinsOverNested404(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	const body = `{"item":{"id":false,"name":"fixed"},"large":9007199254740993}`
	cloud.Mux.HandleFunc("GET /reverse/v1/items/fixed", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		w.Header().Set("X-Accepted", "actual")
		testcloud.JSON(w, 200, body)
	})
	client := cloud.Client("test", "/reverse/v1/")
	nestedStatus := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
	get := func(ctx context.Context, id string, query url.Values) (*terminalConsumerItem, error) {
		target := client.ServiceURL("items", id)
		if encoded := query.Encode(); encoded != "" {
			target += "?" + encoded
		}
		response, err := rest.DoJSON(ctx, client, http.MethodGet, target, nil, nil, 200)
		if err != nil {
			return nil, err
		}
		var decoded struct {
			Item terminalConsumerItem `json:"item"`
		}
		err = json.Unmarshal(response.Body, &decoded)
		// A decoder/translator can carry an older status cause as well as an
		// accepted-response error. It must retain the actual 200 evidence.
		return nil, response.Fail(errors.Join(err, nestedStatus))
	}
	collection := resource.NewCollection(terminalConsumerAdapter(get, &lists))
	check := func(value *terminalConsumerItem, err error) {
		t.Helper()
		var accepted *resource.ResponseError
		var decode *json.UnmarshalTypeError
		if value != nil || err == nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Accepted") != "actual" || !errors.As(err, &decode) || !errors.Is(err, nestedStatus) || errors.Is(err, resource.ErrNotFound) || lists.Load() != 0 {
			t.Fatal("accepted decode became missing or lost evidence", value, err, accepted, decode, lists.Load())
		}
	}
	value, err := collection.Get(context.Background(), "fixed")
	check(value, err)
	value, err = collection.Find(context.Background(), resource.ID("fixed"), resource.WithIgnoreMissing())
	check(value, err)
	check(nil, collection.WaitDeleted(context.Background(), resource.ID("fixed")))
	for _, query := range []bool{false, true} {
		var options []resource.IdentityFindOption
		if query {
			options = append(options, resource.WithIdentityFindQuery("project_id", "chosen"))
		}
		value, err = collection.FindIdentity(context.Background(), "fixed", options...)
		check(value, err)
	}
	if gets.Load() != 5 {
		t.Fatal("accepted decode triggered extra HTTP", gets.Load())
	}
}

func TestTerminalReadConsumersJoinedTerminalCausesKeepSourceNotFound(t *testing.T) {
	for _, terminal := range []error{context.Canceled, context.DeadlineExceeded, io.ErrUnexpectedEOF, &json.SyntaxError{Offset: 1}} {
		t.Run(fmt.Sprintf("%T/%v", terminal, terminal), func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/fixed", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-Source", "actual404")
				testcloud.JSON(w, 404, `{"missing":"fixed"}`)
			})
			client := cloud.Client("test", "/reverse/v1/")
			var original *resource.NotFoundError
			get := func(ctx context.Context, id string, query url.Values) (*terminalConsumerItem, error) {
				target := client.ServiceURL("items", id)
				if encoded := query.Encode(); encoded != "" {
					target += "?" + encoded
				}
				_, err := client.Get(ctx, target, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
				original = &resource.NotFoundError{Resource: "source", Reference: id, Cause: err}
				return nil, errors.Join(original, terminal)
			}
			collection := resource.NewCollection(terminalConsumerAdapter(get, &lists))
			check := func(value *terminalConsumerItem, err error) {
				t.Helper()
				var missing *resource.NotFoundError
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || err == nil || !errors.Is(err, terminal) || !errors.As(err, &missing) || missing != original || !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Source") != "actual404" || string(native.Body) != `{"missing":"fixed"}` || lists.Load() != 0 {
					t.Fatal("joined terminal cause was hidden or replaced", value, err, missing, native, lists.Load())
				}
			}
			value, err := collection.Get(context.Background(), "fixed")
			check(value, err)
			value, err = collection.Find(context.Background(), resource.ID("fixed"), resource.WithIgnoreMissing())
			check(value, err)
			check(nil, collection.WaitDeleted(context.Background(), resource.ID("fixed")))
			for _, query := range []bool{false, true} {
				var options []resource.IdentityFindOption
				if query {
					options = append(options, resource.WithIdentityFindQuery("project_id", "chosen"))
				}
				value, err = collection.FindIdentity(context.Background(), "fixed", options...)
				check(value, err)
			}
			if gets.Load() != 5 {
				t.Fatal("terminal cause triggered extra HTTP", gets.Load())
			}
		})
	}
}

type terminalConsumerRoundTrip func(*http.Request) (*http.Response, error)

func (f terminalConsumerRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
