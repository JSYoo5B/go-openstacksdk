package senlin_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func listRequestConfig(t *testing.T, options ...senlin.ListOption) request.Config[senlin.ListOpts] {
	t.Helper()
	config, err := request.Apply(senlin.ListOpts{}, options...)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func listRequestSpec(client *gophercloud.ServiceClient) rest.CollectionSpec[filterRow] {
	spec := filterCollectionSpec(client)
	spec.Validate = func(ctx context.Context) error { return senlin.RequireVersion(ctx, client, 7) }
	return spec
}

func TestListRequestClientOwnsControlsAndPreservesSharedSource(t *testing.T) {
	source := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Type: "clustering", Endpoint: "https://service.test/v1/project/", Microversion: "1.2",
		MoreHeaders: map[string]string{"x-trace": "shared", "X-Source": "kept", "Accept": "source/json"}}
	plain, err := senlin.PrepareListClient(context.Background(), source, listRequestConfig(t))
	if err != nil || plain != source {
		t.Fatal("uncontrolled list changed source ownership", plain, err)
	}
	config := listRequestConfig(t,
		senlin.WithListMicroversion[senlin.ListOpts]("1.7"),
		senlin.WithListHeader[senlin.ListOpts]("x-trace", "earlier"),
		senlin.WithListHeader[senlin.ListOpts]("X-TRACE", "last"),
		senlin.WithListHeader[senlin.ListOpts]("accept", "application/vnd.test+json"),
		request.WithQuery[senlin.ListOpts]("vendor", "kept"),
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "schema", map[string]any{"n": json.Number("9007199254740993")}))
	if err := senlin.ValidateListCapabilities(config, bodyFilterDescriptor().Namespace); err != nil {
		t.Fatal(err)
	}
	effective, err := senlin.PrepareListClient(context.Background(), source, config)
	if err != nil || effective == source || effective.ProviderClient != source.ProviderClient || effective.Endpoint != source.Endpoint || effective.Microversion != "1.7" {
		t.Fatal("incorrect request clone", effective, err)
	}
	want := map[string]string{"X-Trace": "last", "X-Source": "kept", "Accept": "application/vnd.test+json"}
	if !reflect.DeepEqual(effective.MoreHeaders, want) {
		t.Fatal("case-insensitive caller precedence", effective.MoreHeaders)
	}
	config.Headers["X-Trace"] = "changed-config"
	source.MoreHeaders["x-trace"] = "changed-source"
	if effective.MoreHeaders["X-Trace"] != "last" || source.Microversion != "1.2" || source.MoreHeaders["Accept"] != "source/json" {
		t.Fatal("request controls alias shared headers", source, effective)
	}
	effective.MoreHeaders["X-Source"] = "changed-request"
	if source.MoreHeaders["X-Source"] != "kept" {
		t.Fatal("effective headers alias source")
	}
	data := senlin.ListQueryConfig(config)
	if len(data.Headers) != 0 || len(data.Arguments) != 1 || len(config.Arguments) != 2 || len(config.Headers) != 2 {
		t.Fatal("request controls were not separated on a copy", data, config)
	}
	query, err := senlin.Query(data, bodyFilterDescriptor().Namespace)
	if err != nil || query.Get("vendor") != "kept" {
		t.Fatal(query, err)
	}
	filters, err := senlin.PrepareBodyFilters(data, bodyFilterDescriptor())
	if err != nil || string(filters["schema"]) != `{"n":9007199254740993}` {
		t.Fatal("separated config changed raw filters", filters, err)
	}
	delete(data.Arguments, bodyFilterDescriptor().Namespace)
	if len(config.Arguments) != 2 {
		t.Fatal("separated arguments alias original config")
	}
}

func TestListRequestMicroversionSelectionGatesAndHeaderConflicts(t *testing.T) {
	for _, tc := range []struct {
		name, source, selected, header, service string
		option                                  senlin.ListOption
		want                                    error
	}{
		{name: "numeric-upgrade", source: "1.2", selected: "1.7", service: "clustering"},
		{name: "explicit-server-default", source: "1.7", selected: "", service: "clustering"},
		{name: "matching-source-header", source: "1.7", selected: "1.7", header: "clustering 1.7", service: "clustering"},
		{name: "stale-source-header", source: "1.2", selected: "1.7", header: "clustering 1.2", service: "clustering", want: resource.ErrInvalidOption},
		{name: "reset-conflicts-with-source-header", source: "1.7", selected: "", header: "clustering 1.7", service: "clustering", want: resource.ErrInvalidOption},
		{name: "symbolic-version", selected: "latest", service: "clustering", want: resource.ErrUnsupported},
		{name: "major-version", selected: "2.0", service: "clustering", want: resource.ErrInvalidOption},
		{name: "leading-zero", selected: "1.07", service: "clustering", want: resource.ErrInvalidOption},
		{name: "missing-minor", selected: "1.", service: "clustering", want: resource.ErrInvalidOption},
		{name: "negative-minor", selected: "1.-1", service: "clustering", want: resource.ErrInvalidOption},
		{name: "numeric-requires-service-type", selected: "1.7", want: resource.ErrInvalidOption},
		{name: "wrong-argument-type", service: "clustering", option: request.WithArgument[senlin.ListOpts]("senlin.list.microversion", 7), want: resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Type: tc.service, Microversion: tc.source}
			if tc.header != "" {
				source.MoreHeaders = map[string]string{"openstack-api-version": tc.header}
			}
			option := tc.option
			if option == nil {
				option = senlin.WithListMicroversion[senlin.ListOpts](tc.selected)
			}
			effective, err := senlin.PrepareListClient(context.Background(), source, listRequestConfig(t, option))
			if tc.want != nil {
				if effective != nil || !errors.Is(err, tc.want) {
					t.Fatal(effective, err)
				}
				return
			}
			if err != nil || effective.Microversion != tc.selected || source.Microversion != tc.source {
				t.Fatal(effective, err)
			}
			gate := senlin.ListSpec(source, effective, listRequestSpec)
			err = gate.Validate(context.Background())
			if tc.selected == "" {
				if !errors.Is(err, resource.ErrUnsupported) || len(effective.MoreHeaders) != 0 {
					t.Fatal("server default bypassed the effective minimum", effective, err)
				}
			} else if err != nil {
				t.Fatal("source minimum incorrectly blocked effective upgrade", err)
			}
		})
	}
}

func TestListRequestPreflightKeepsKindAndRejectsControlBypasses(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"items":[]}`)
	})
	options := []senlin.ListOption{
		request.WithField[senlin.ListOpts]("name", "body"),
		request.WithArgument[senlin.ListOpts]("other.local_filters", map[string]json.RawMessage{}),
		request.WithArgument[senlin.ListOpts]("senlin.list.microversion", 7),
		request.WithQuery[senlin.ListOpts]("schema", "wire"),
		senlin.WithMaxItems(-1),
		func(config *request.Config[senlin.ListOpts]) error {
			config.Headers = map[string]string{"x-vendor": "one", "X-Vendor": "two"}
			return nil
		},
		nil,
	}
	for _, key := range []string{"headers", "microversion", "base_path", "jmespath_filters", "allow_unknown_params", "max_items", "paginated"} {
		if err := senlin.RejectListControlQuery(url.Values{key: {"value"}}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("request control remained in query", key, err)
		}
		options = append(options, request.WithQuery[senlin.ListOpts](key, "wire"))
	}
	for _, key := range []string{"openstack-api-version", "X-AUTH-TOKEN", "x-service-token", "authorization", "host", "Cookie", "Content-Type", "Content-Length", "bad:key", "bad key"} {
		options = append(options, senlin.WithListHeader[senlin.ListOpts](key, "value"))
	}
	options = append(options, senlin.WithListHeader[senlin.ListOpts]("X-Vendor", "one\r\ntwo"))
	for i, option := range options {
		source := cloud.Client("clustering", "/senlin")
		source.Microversion = "1.7"
		stream := senlin.ListWithClientBodyFilters(context.Background(), source, listRequestSpec, bodyFilterDescriptor(), option)
		if calls.Load() != 0 {
			t.Fatal("eager HTTP", calls.Load())
		}
		rows, err := collectBodyFiltered(t, stream)
		var operation *resource.OperationError
		if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "List" || operation.Resource != "filter-test" || calls.Load() != 0 {
			t.Fatal(i, rows, err, operation, calls.Load())
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {Type: "clustering"}, {ProviderClient: cloud.Provider, Type: "compute"}} {
		_, err := collectBodyFiltered(t, senlin.ListWithClientBodyFilters(context.Background(), source, listRequestSpec, bodyFilterDescriptor()))
		var operation *resource.OperationError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Resource != "filter-test" || calls.Load() != 0 {
			t.Fatal("invalid source preflight lost kind or cause", err, calls.Load())
		}
	}
	_, err := collectBodyFiltered(t, senlin.ListWithClientBodyFilters[filterRow](context.Background(), cloud.Client("clustering", "/senlin"), nil, bodyFilterDescriptor()))
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("nil factory was not a lazy preflight error", err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = collectBodyFiltered(t, senlin.ListWithClientBodyFilters(ctx, cloud.Client("clustering", "/senlin"), listRequestSpec, bodyFilterDescriptor(), senlin.WithListHeader[senlin.ListOpts]("X-Vendor", "kept")))
	if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled controls dispatched HTTP", err, calls.Load())
	}
}

func TestListRequestHeadersVersionAndLiveAuthRemainFixedAcrossPagesAndReauth(t *testing.T) {
	cloud := testcloud.New(t)
	source := cloud.Client("clustering", "/senlin")
	source.Microversion = "1.2"
	source.MoreHeaders = map[string]string{"x-trace": "source", "X-Source": "kept"}
	var attempts, reauth atomic.Int32
	var captured request.Config[senlin.ListOpts]
	cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		wantToken := "next-page-token"
		if attempt == 1 {
			wantToken = "test-token"
		} else if attempt == 2 {
			wantToken = "fresh-token"
		}
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-Trace") != "request" || r.Header.Get("X-Source") != "kept" || r.Header.Get("Accept") != "application/vnd.test+json" || r.Header.Get("X-Auth-Token") != wantToken {
			t.Error("page or retry changed owned request controls", attempt, r.Header)
		}
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("vendor") != "kept" || len(r.URL.Query()) != 2+boolCount(r.URL.Query().Has("marker")) {
			t.Error("local controls or Body filters leaked to wire", r.URL)
		}
		if attempt == 1 {
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		w.Header().Set("X-Request-ID", "list-proof")
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"items":[{"id":"one","name":"other"},{"id":"two","name":"wanted","large":9007199254740993}],"links":[{"rel":"next","href":"?marker=two&limit=2"}]}`)
		} else {
			if r.URL.Query().Get("marker") != "two" {
				t.Error("unexpected continuation", r.URL)
			}
			testcloud.JSON(w, 200, `{"items":[{"id":"three","name":"wanted"},false],"links":false}`)
		}
	})
	cloud.Provider.ReauthFunc = func(ctx context.Context) error {
		reauth.Add(1)
		captured.Headers["X-Trace"] = "mutated-during-reauth"
		captured.Query.Set("vendor", "mutated-during-reauth")
		cloud.Provider.SetToken("fresh-token")
		return nil
	}
	options := []senlin.ListOption{
		request.WithOptions(senlin.ListOpts{Limit: 2}), senlin.WithMaxItems(3),
		senlin.WithListHeader[senlin.ListOpts]("x-trace", "request"),
		senlin.WithListHeader[senlin.ListOpts]("Accept", "application/vnd.test+json"),
		senlin.WithListMicroversion[senlin.ListOpts]("1.7"),
		request.WithQuery[senlin.ListOpts]("vendor", "kept"),
		senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "name", "wanted"),
		func(config *request.Config[senlin.ListOpts]) error { captured = *config; return nil },
	}
	stream := senlin.ListWithClientBodyFilters(context.Background(), source, listRequestSpec, bodyFilterDescriptor(), options...)
	options[2], options[4] = nil, nil
	if attempts.Load() != 0 {
		t.Fatal("list preparation performed eager HTTP")
	}
	for iteration := range 2 {
		var ids []string
		for value, err := range stream {
			if err != nil {
				t.Fatal(iteration, err)
			}
			ids = append(ids, value.ID)
			if value.ID == "two" {
				if string(value.Body["large"]) != "9007199254740993" || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "list-proof" {
					t.Fatal("response evidence or exact number lost", value)
				}
				source.MoreHeaders["x-trace"] = "source-changed-after-page"
				cloud.Provider.SetToken("next-page-token")
			}
		}
		if !reflect.DeepEqual(ids, []string{"two", "three"}) {
			t.Fatal("raw cap or repeated iteration changed", iteration, ids)
		}
	}
	if attempts.Load() != 5 || reauth.Load() != 1 || source.Microversion != "1.2" || len(source.MoreHeaders) != 2 || source.MoreHeaders["X-Source"] != "kept" {
		t.Fatal("shared configuration changed or request resent", attempts.Load(), reauth.Load(), source)
	}
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestListRequestRevalidatesSourceBeforeContinuationAndGatesEffectiveVersion(t *testing.T) {
	for _, mode := range []string{"type", "provider", "version-header", "clone-source-downshift", "uncontrolled-downshift"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			source := cloud.Client("clustering", "/senlin")
			source.Microversion = "1.7"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				version := "clustering 1.8"
				if mode == "uncontrolled-downshift" {
					version = "clustering 1.7"
				}
				if r.Header.Get("OpenStack-API-Version") != version {
					t.Error("effective selection changed", r.Header)
				}
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, `{"items":[{"id":"one"}],"links":[{"rel":"next","href":"?marker=one"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"items":[{"id":"two"}]}`)
				}
			})
			var options []senlin.ListOption
			if mode != "uncontrolled-downshift" {
				options = append(options, senlin.WithListMicroversion[senlin.ListOpts]("1.8"))
			}
			stream := senlin.ListWithClientBodyFilters(context.Background(), source, listRequestSpec, bodyFilterDescriptor(), options...)
			var ids []string
			var failure error
			for value, err := range stream {
				if err != nil {
					failure = err
					continue
				}
				ids = append(ids, value.ID)
				if value.ID == "one" {
					switch mode {
					case "type":
						source.Type = "compute"
					case "provider":
						source.ProviderClient = nil
					case "version-header":
						source.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.9"}
					default:
						source.Microversion = "1.0"
					}
				}
			}
			if mode == "clone-source-downshift" {
				if failure != nil || calls.Load() != 2 || !reflect.DeepEqual(ids, []string{"one", "two"}) {
					t.Fatal("valid source's lower version blocked effective gate", ids, failure, calls.Load())
				}
				return
			}
			want := resource.ErrInvalidOption
			if mode == "uncontrolled-downshift" {
				want = resource.ErrUnsupported
			}
			if !reflect.DeepEqual(ids, []string{"one"}) || !errors.Is(failure, want) || calls.Load() != 1 {
				t.Fatal("invalidated source reached continuation", ids, failure, calls.Load())
			}
		})
	}
}

func TestListRequestControlsKeepRawCapSinglePageAndAcceptedEvidence(t *testing.T) {
	for _, mode := range []string{"filtered-cap", "single-page", "malformed-row", "foreign-link", "filtered-cancellation", "consumer-break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"items":[{"id":"one","name":"wanted"}],"links":false}`
			if mode == "filtered-cap" || mode == "filtered-cancellation" {
				body = `{"items":[{"id":"one","name":"other"},false],"links":false}`
			} else if mode == "malformed-row" {
				body = `{"items":[{"id":"one","name":"wanted"},false],"links":false}`
			} else if mode == "foreign-link" {
				body = `{"items":[{"id":"one","name":"wanted"}],"links":[{"rel":"next","href":"https://foreign.invalid/items"}]}`
			}
			cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-Vendor") != "kept" {
					t.Error("request controls lost", r.Header)
				}
				w.Header().Set("X-Request-ID", "accepted-proof")
				testcloud.JSON(w, 200, body)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			factory := func(client *gophercloud.ServiceClient) rest.CollectionSpec[filterRow] {
				spec := listRequestSpec(client)
				if mode == "filtered-cancellation" {
					spec.ValidateItem = func(*filterRow) error { cancel(); return nil }
				}
				return spec
			}
			options := []senlin.ListOption{senlin.WithListMicroversion[senlin.ListOpts]("1.7"), senlin.WithListHeader[senlin.ListOpts]("X-Vendor", "kept"),
				senlin.WithBodyFilter[senlin.ListOpts](bodyFilterDescriptor(), "name", "wanted")}
			if mode == "filtered-cap" || mode == "filtered-cancellation" {
				options = append(options, senlin.WithMaxItems(1))
			} else if mode == "single-page" {
				options = append(options, senlin.WithPaginated(false))
			}
			stream := senlin.ListWithClientBodyFilters(ctx, cloud.Client("clustering", "/senlin"), factory, bodyFilterDescriptor(), options...)
			if mode == "consumer-break" {
				for value, err := range stream {
					if err != nil || value.ID != "one" {
						t.Fatal(value, err)
					}
					cancel()
					break
				}
				if calls.Load() != 1 {
					t.Fatal("consumer break requested continuation", calls.Load())
				}
				return
			}
			ids, err := collectBodyFiltered(t, stream)
			if mode == "single-page" || mode == "filtered-cap" {
				want := []string{"one"}
				if mode == "filtered-cap" {
					want = []string{}
				}
				if err != nil || !reflect.DeepEqual(ids, want) || calls.Load() != 1 {
					t.Fatal("unused row or next link evaluated after local stop", ids, err, calls.Load())
				}
				return
			}
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "accepted-proof" || string(proof.Body) != body || calls.Load() != 1 {
				t.Fatal("accepted failure lost evidence or resent", ids, err, proof, calls.Load())
			}
			if mode == "filtered-cancellation" && (!errors.Is(err, context.Canceled) || len(ids) != 0) {
				t.Fatal("filtered terminal cancellation disappeared", ids, err)
			}
		})
	}
}

func TestListRequestConcurrentCallsKeepHeadersVersionsAndSharedConfiguration(t *testing.T) {
	cloud := testcloud.New(t)
	source := cloud.Client("clustering", "/senlin")
	source.Microversion = "1.2"
	source.MoreHeaders = map[string]string{"x-call": "source", "X-Shared": "kept"}
	versions := map[string]string{"first": "1.1", "second": "1.7", "default": ""}
	var firstPages, calls atomic.Int32
	allStarted := make(chan struct{})
	cloud.Mux.HandleFunc("GET /senlin/items", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		label := r.Header.Get("X-Call")
		version, exists := versions[label]
		wantVersion := ""
		if version != "" {
			wantVersion = "clustering " + version
		}
		if !exists || r.Header.Get("OpenStack-API-Version") != wantVersion || r.Header.Get("X-Shared") != "kept" || r.Header.Get("Accept") != "application/vnd."+label+"+json" || r.Header.Get("X-Auth-Token") != "test-token" || r.URL.Query().Get("vendor") != label {
			t.Error("concurrent per-call settings crossed request boundaries", r.URL, r.Header)
		}
		if r.URL.Query().Get("marker") == "" {
			if firstPages.Add(1) == int32(len(versions)) {
				close(allStarted)
			}
			select {
			case <-allStarted:
			case <-r.Context().Done():
				return
			}
			testcloud.JSON(w, 200, `{"items":[{"id":"`+label+`-one"}],"links":[{"rel":"next","href":"?marker=first"}]}`)
		} else {
			testcloud.JSON(w, 200, `{"items":[{"id":"`+label+`-two"}]}`)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var workers sync.WaitGroup
	for label, version := range versions {
		workers.Add(1)
		go func() {
			defer workers.Done()
			stream := senlin.ListWithClientBodyFilters(ctx, source, filterCollectionSpec, bodyFilterDescriptor(),
				senlin.WithListHeader[senlin.ListOpts]("X-Call", label),
				senlin.WithListHeader[senlin.ListOpts]("Accept", "application/vnd."+label+"+json"),
				senlin.WithListMicroversion[senlin.ListOpts](version),
				request.WithQuery[senlin.ListOpts]("vendor", label))
			var ids []string
			for value, err := range stream {
				if err != nil {
					t.Error(label, err)
					return
				}
				ids = append(ids, value.ID)
			}
			if !reflect.DeepEqual(ids, []string{label + "-one", label + "-two"}) {
				t.Error("concurrent iteration lost its own rows", label, ids)
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 6 || source.Microversion != "1.2" || !reflect.DeepEqual(source.MoreHeaders, map[string]string{"x-call": "source", "X-Shared": "kept"}) {
		t.Fatal("per-call settings mutated shared source", calls.Load(), source)
	}
}
