package api_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type typedListControlInput struct {
	ctx              context.Context
	max, limit       int
	paginated        *bool
	maxSequence      []int
	paginateSequence []bool
	filter           bool
	queryKey         string
}

func typedListContext(input typedListControlInput) context.Context {
	if input.ctx != nil {
		return input.ctx
	}
	return context.Background()
}

type typedListControlCase struct {
	plural string
	filter string
	list   func(*gophercloud.ServiceClient, typedListControlInput) iter.Seq2[*resource.Metadata, error]
}

func typedListMetadata[T any](sequence iter.Seq2[*T, error], metadata func(*T) *resource.Metadata) iter.Seq2[*resource.Metadata, error] {
	return func(yield func(*resource.Metadata, error) bool) {
		for value, err := range sequence {
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(metadata(value), nil) {
				return
			}
		}
	}
}

func typedListCases() []typedListControlCase {
	return []typedListControlCase{
		{"profiles", "metadata", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []profiles.ListOption{profiles.WithListOptions(profiles.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, profiles.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, profiles.WithListPaginated(value))
			}
			if input.filter {
				options = append(options, profiles.WithListFilter("metadata", map[string]any{"select": true}))
			}
			if input.queryKey != "" {
				options = append(options, profiles.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(profiles.New(client).List(typedListContext(input), options...), func(value *profiles.Profile) *resource.Metadata { return &value.Metadata })
		}},
		{"policies", "data", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []policies.ListOption{policies.WithListOptions(policies.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, policies.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, policies.WithListPaginated(value))
			}
			if input.filter {
				options = append(options, policies.WithListFilter("data", map[string]any{"select": true}))
			}
			if input.queryKey != "" {
				options = append(options, policies.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(policies.New(client).List(typedListContext(input), options...), func(value *policies.Policy) *resource.Metadata { return &value.Metadata })
		}},
		{"clusters", "metadata", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []clusters.ListOption{clusters.WithListOptions(clusters.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, clusters.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, clusters.WithListPaginated(value))
			}
			if input.filter {
				options = append(options, clusters.WithListFilter("metadata", map[string]any{"select": true}))
			}
			if input.queryKey != "" {
				options = append(options, clusters.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(clusters.New(client).List(typedListContext(input), options...), func(value *clusters.Cluster) *resource.Metadata { return &value.Metadata })
		}},
		{"nodes", "metadata", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []nodes.ListOption{nodes.WithListOptions(nodes.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, nodes.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, nodes.WithListPaginated(value))
			}
			if input.filter {
				options = append(options, nodes.WithListFilter("metadata", map[string]any{"select": true}))
			}
			if input.queryKey != "" {
				options = append(options, nodes.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(nodes.New(client).List(typedListContext(input), options...), func(value *nodes.Node) *resource.Metadata { return &value.Metadata })
		}},
		{"receivers", "params", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []receivers.ListOption{receivers.WithListOptions(receivers.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, receivers.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, receivers.WithListPaginated(value))
			}
			if input.filter {
				options = append(options, receivers.WithListFilter("params", map[string]any{"select": true}))
			}
			if input.queryKey != "" {
				options = append(options, receivers.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(receivers.New(client).List(typedListContext(input), options...), func(value *receivers.Receiver) *resource.Metadata { return &value.Metadata })
		}},
		{"actions", "", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []actions.ListOption{actions.WithListOptions(actions.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, actions.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, actions.WithListPaginated(value))
			}
			if input.queryKey != "" {
				options = append(options, actions.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(actions.New(client).List(typedListContext(input), options...), func(value *actions.Action) *resource.Metadata { return &value.Metadata })
		}},
		{"events", "", func(client *gophercloud.ServiceClient, input typedListControlInput) iter.Seq2[*resource.Metadata, error] {
			options := []events.ListOption{events.WithListOptions(events.ListOpts{MaxItems: input.max, Paginated: input.paginated, Limit: input.limit})}
			for _, value := range input.maxSequence {
				options = append(options, events.WithListMaxItems(value))
			}
			for _, value := range input.paginateSequence {
				options = append(options, events.WithListPaginated(value))
			}
			if input.queryKey != "" {
				options = append(options, events.WithListQuery(input.queryKey, "override"))
			}
			return typedListMetadata(events.New(client).List(typedListContext(input), options...), func(value *events.Event) *resource.Metadata { return &value.Metadata })
		}},
	}
}

func typedListCollect(sequence iter.Seq2[*resource.Metadata, error]) ([]*resource.Metadata, error) {
	values := make([]*resource.Metadata, 0)
	for value, err := range sequence {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func typedListRows(plural, tail string) string {
	return fmt.Sprintf(`{%q:[{"id":"one","name":"one","metadata":{"select":false},"data":{"select":false},"params":{"select":false},"vendor":9007199254740993},{"id":"two","name":"two","metadata":{"select":true},"data":{"select":true},"params":{"select":true}}]%s}`, plural, tail)
}

func TestClusteringTypedListControlsSevenFacadesCountRawRowsBeforeFilters(t *testing.T) {
	for _, facade := range typedListCases() {
		t.Run(facade.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Query().Get("limit") != "10" || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
					t.Error(r.URL)
				}
				w.Header().Set("X-Request-ID", "raw-page")
				w.Header().Set("Link", "invalid continuation must not be parsed")
				body := fmt.Sprintf(`{%q:[{"id":"one",%q:{"select":false},"vendor":9007199254740993},{"id":"two",%q:{"select":true}},{"id":[]}],"next":false}`, facade.plural, facade.filter, facade.filter)
				testcloud.JSON(w, 200, body)
			})
			values, err := typedListCollect(facade.list(cloud.Client("clustering", "/reverse/senlin/v1"), typedListControlInput{max: 2, limit: 10, filter: facade.filter != ""}))
			want := 2
			if facade.filter != "" {
				want = 1
			}
			if err != nil || len(values) != want || requests.Load() != 1 {
				t.Fatal("cap counted yielded rather than raw rows or decoded past it", err, len(values), requests.Load())
			}
			for _, value := range values {
				if value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "raw-page" {
					t.Fatal(value)
				}
			}
		})
	}
}

func TestClusteringTypedListControlsSinglePageIgnoresUnneededContinuation(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, mode := range []string{"body", "http", "foreign"} {
			t.Run(facade.plural+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.Query().Get("limit") != "1" || r.URL.Query().Has("paginated") {
						t.Error(r.URL)
					}
					tail := `,"next":false`
					if mode == "http" {
						tail = ""
						w.Header().Set("Link", "malformed")
					} else if mode == "foreign" {
						tail = `,"links":[{"rel":"next","href":"https://foreign.invalid/other"}]`
					}
					testcloud.JSON(w, 200, typedListRows(facade.plural, tail))
				})
				paged := false
				values, err := typedListCollect(facade.list(cloud.Client("clustering", "/senlin/v1"), typedListControlInput{limit: 1, paginated: &paged}))
				if err != nil || len(values) != 2 || requests.Load() != 1 {
					t.Fatal("single page validated or followed continuation", err, len(values), requests.Load())
				}
			})
		}
	}
}

func TestClusteringTypedListControlsLimitHintLastWinsAndExactBoundary(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, limit := range []int{0, 5} {
			t.Run(facade.plural+"/limit="+strconv.Itoa(limit), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					want := limit
					if want == 0 {
						want = 2
					}
					if r.URL.Query().Get("limit") != strconv.Itoa(want) || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, typedListRows(facade.plural, `,"next":false`))
				})
				sequence := facade.list(cloud.Client("clustering", "/senlin/v1"), typedListControlInput{limit: limit, maxSequence: []int{-1, 99, 2}})
				for iteration := 0; iteration < 2; iteration++ {
					values, err := typedListCollect(sequence)
					if err != nil || len(values) != 2 || requests.Load() != int32(iteration+1) {
						t.Fatal("boundary fetched another page or option reuse leaked counters", err, len(values), requests.Load())
					}
				}
			})
		}
	}
}

func TestClusteringTypedListControlsOwnPaginatedPointersAndOverrideOrder(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, override := range []bool{false, true} {
			t.Run(facade.plural+"/override="+strconv.FormatBool(override), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.Query().Get("limit") != "2" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"only"}],"next":false}`, facade.plural))
				})
				paged := false
				input := typedListControlInput{limit: 2, paginated: &paged}
				if override {
					paged = true
					input.paginateSequence = []bool{false, true, false}
				}
				sequence := facade.list(cloud.Client("clustering", "/senlin/v1"), input)
				paged = true // Neither a constructor snapshot nor a reusable option may alias this pointer.
				for iteration := 0; iteration < 2; iteration++ {
					values, err := typedListCollect(sequence)
					if err != nil || len(values) != 1 || requests.Load() != int32(iteration+1) {
						t.Fatal("pagination input mutated existing iterator", err, len(values), requests.Load())
					}
				}
			})
		}
	}
}

func TestClusteringTypedListControlsZeroUnlimitedTrueOverridesFalseAndEmptyStops(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, override := range []bool{false, true} {
			t.Run(facade.plural+"/override="+strconv.FormatBool(override), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.Query().Get("limit") != "1" || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
						t.Error(r.URL)
					}
					marker := r.URL.Query().Get("marker")
					switch marker {
					case "":
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"one"}]}`, facade.plural))
					case "one":
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"two"}]}`, facade.plural))
					case "two":
						w.Header().Set("Link", "malformed empty continuation")
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[],"next":false}`, facade.plural))
					default:
						t.Error(marker)
						w.WriteHeader(500)
					}
				})
				input := typedListControlInput{limit: 1}
				if override {
					input.maxSequence, input.paginateSequence = []int{1, 0}, []bool{false, true}
				}
				values, err := typedListCollect(facade.list(cloud.Client("clustering", "/senlin/v1"), input))
				if err != nil || len(values) != 2 || requests.Load() != 3 {
					t.Fatal("zero stayed capped, true stayed single page, or empty followed next", err, len(values), requests.Load())
				}
			})
		}
	}
}

func TestClusteringTypedListControlsParentCancellation(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, preflight := range []bool{false, true} {
			t.Run(facade.plural+"/preflight="+strconv.FormatBool(preflight), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"one"}],"next":"?marker=two"}`, facade.plural))
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if preflight {
					cancel()
				}
				sequence := facade.list(cloud.Client("clustering", "/senlin/v1"), typedListControlInput{ctx: ctx, max: 3, limit: 1})
				var terminal error
				var yielded int
				for _, err := range sequence {
					if err != nil {
						terminal = err
						break
					}
					yielded++
					cancel()
				}
				want := 1
				if preflight {
					want = 0
				}
				if !errors.Is(terminal, context.Canceled) || requests.Load() != int32(want) || yielded != want {
					t.Fatal("cancellation fetched another page or lost parent cause", terminal, requests.Load(), yielded)
				}
			})
		}
	}
}

func TestClusteringTypedListControlsNegativeAndReservedQueryFailBeforeHTTP(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, mode := range []string{"negative", "max_items", "paginated"} {
			t.Run(facade.plural+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(500) })
				input := typedListControlInput{queryKey: mode}
				if mode == "negative" {
					input.queryKey, input.max = "", -1
				}
				_, err := typedListCollect(facade.list(cloud.Client("clustering", "/senlin/v1"), input))
				if !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 0 {
					t.Fatal(err, requests.Load())
				}
			})
		}
	}
}

func TestClusteringTypedListControlsConsumerBreakAndNextPageSourceGuard(t *testing.T) {
	for _, facade := range typedListCases() {
		for _, mode := range []string{"break", "version", "type"} {
			t.Run(facade.plural+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/senlin/v1")
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+facade.plural, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"one"}],"next":"?marker=two"}`, facade.plural))
				})
				sequence := facade.list(client, typedListControlInput{max: 3, limit: 1})
				var terminal error
				var yielded int
				for value, err := range sequence {
					if err != nil {
						terminal = err
						break
					}
					yielded++
					if value.StatusCode != 200 {
						t.Fatal(value)
					}
					switch mode {
					case "break":
					case "version":
						client.Microversion = "latest"
					case "type":
						client.Type = "compute"
					}
					if mode == "break" {
						break
					}
				}
				if mode == "version" && !errors.Is(terminal, resource.ErrUnsupported) || mode == "type" && !errors.Is(terminal, resource.ErrInvalidOption) || mode == "break" && terminal != nil || requests.Load() != 1 || yielded != 1 {
					t.Fatal("fetched after break or lost next-page source validation", terminal, requests.Load(), yielded)
				}
			})
		}
	}
}
