package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

type listItem struct {
	resource.Metadata
	ID         string `json:"id"`
	WireMarker string `json:"wire_marker"`
	Name       string `json:"name"`
	Value      int64  `json:"value"`
}

func (v *listItem) UnmarshalJSON(data []byte) error {
	type plain listItem
	var value plain
	if err := resource.DecodeObject(data, &value, &value.Metadata); err != nil {
		return err
	}
	*v = listItem(value)
	return nil
}

func listSpec(t *testing.T, handler http.HandlerFunc) CollectionSpec[listItem] {
	t.Helper()
	fake := testhelper.SetupHTTP()
	t.Cleanup(fake.Teardown)
	fake.Mux.Handle("/", handler)
	server := fake.Server
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("shared-token")
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: server.URL + "/v1/"}
	return CollectionSpec[listItem]{Client: client, Path: "items", Kind: "items", SingleKey: "item", PluralKey: "items",
		ID: func(v *listItem) string { return v.ID }, Name: func(v *listItem) string { return v.Name },
		Metadata: func(v *listItem) *resource.Metadata { return &v.Metadata }, Get: true, Delete: true}
}

func writeList(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Page", "kept")
	fmt.Fprint(w, body)
}

func collectList(ctx context.Context, spec CollectionSpec[listItem], query url.Values) ([]*listItem, error) {
	var values []*listItem
	for value, err := range List(ctx, spec, query) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestListLazyExactObjectsAndFiltersSurviveEmptyContinuation(t *testing.T) {
	var calls, validates, queryValidates atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/items" || r.Header.Get("X-Auth-Token") != "shared-token" || r.URL.Query().Get("project_id") != "fixed" || r.URL.Query().Get("sort") != "name:asc" || r.URL.Query().Get("limit") != "2" || strings.Join(r.URL.Query()["vendor"], ",") != "a,b" {
			t.Errorf("page changed request: %s %v", r.URL, r.Header)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			writeList(w, `{"items":[{"id":"one","wire_marker":"wire-one","value":9007199254740993,"vendor":{"counter":9007199254740995},"nullable":null}],"items_links":[{"rel":"next","href":"?marker=wire-one"}]}`)
		case "wire-one":
			writeList(w, `{"items":[],"next":"?marker=wire-two"}`)
		case "wire-two":
			writeList(w, `{"items":[{"id":"two","value":7}]}`)
		default:
			t.Errorf("unexpected marker: %s", r.URL)
		}
	})
	spec.Validate = func(context.Context) error { validates.Add(1); return nil }
	spec.ValidateQuery = func(_ context.Context, query url.Values) error {
		queryValidates.Add(1)
		query.Set("project_id", "mutated validation input")
		return nil
	}
	query := url.Values{"project_id": {"fixed"}, "sort": {"name:asc"}, "limit": {"2"}, "vendor": {"a", "b"}}
	stream := List(context.Background(), spec, query)
	query.Set("project_id", "later mutation")
	if calls.Load() != 0 || validates.Load() != 0 {
		t.Fatal("list was eager")
	}
	var values []*listItem
	for value, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 2 || values[0].Value != 9007199254740993 || string(values[0].Body["nullable"]) != "null" || !strings.Contains(string(values[0].Body["vendor"]), "9007199254740995") || values[0].Body["missing"] != nil || values[0].StatusCode != 200 || calls.Load() != 3 || validates.Load() != 3 || queryValidates.Load() != 3 {
		t.Fatalf("values=%+v calls/validation=%d/%d/%d", values, calls.Load(), validates.Load(), queryValidates.Load())
	}
	values[0].Header.Set("X-Page", "changed")
	if values[1].Header.Get("X-Page") != "kept" {
		t.Fatal("page metadata shares headers")
	}
}

func TestListBreakDoesNotValidateOrFetchAnotherPage(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"first"},{"id":"second"}],"next":"https://foreign.invalid/items"}`)
	})
	for value, err := range List(context.Background(), spec, nil) {
		if err != nil || value.ID != "first" {
			t.Fatalf("value=%v err=%v", value, err)
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal("break fetched another page")
	}
}

func TestListDictionaryLinksAreExplicitAndGuarded(t *testing.T) {
	for _, tc := range []struct {
		name, continuation  string
		enabled, wantError  bool
		wantCalls, wantRows int
	}{
		{"opt-in", `"links":{"next":"?marker=next","self":{"ignored":true}}`, true, false, 2, 2},
		{"disabled", `"links":{"next":"?marker=next"}`, false, true, 1, 1},
		{"array-compatible", `"links":[{"rel":"next","href":"?marker=next"}]`, true, false, 2, 2},
		{"null-next", `"links":{"next":null}`, true, false, 1, 1},
		{"empty-next", `"links":{"next":""}`, true, false, 1, 1},
		{"malformed-next", `"links":{"next":[]}`, true, true, 1, 1},
		{"conflicting-next", `"links":{"next":"?marker=next"},"next":"?marker=other"`, true, true, 1, 1},
		{"foreign-path", `"links":{"next":"/v1/other?marker=next"}`, true, true, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					writeList(w, `{"items":[{"id":"first"}],`+tc.continuation+`}`)
				} else {
					writeList(w, `{"items":[{"id":"second"}]}`)
				}
			})
			spec.Paging.DictionaryLinks = tc.enabled
			rows, err := collectList(context.Background(), spec, nil)
			if (err != nil) != tc.wantError || int(calls.Load()) != tc.wantCalls || len(rows) != tc.wantRows {
				t.Fatalf("rows=%d calls=%d err=%v", len(rows), calls.Load(), err)
			}
			if tc.wantError {
				var response *resource.ResponseError
				if !errors.As(err, &response) || response.StatusCode != http.StatusOK || response.Header.Get("X-Page") != "kept" {
					t.Fatalf("continuation error lost HTTP evidence: %v", err)
				}
			}
		})
	}
}

func TestListContinuationCannotWidenCollectionFiltersOrMarkers(t *testing.T) {
	for name, next := range map[string]string{
		"origin": "https://foreign.invalid/v1/items?marker=next", "path": "/v1/other?marker=next", "userinfo": "http://user@placeholder/v1/items?marker=next",
		"project": "?marker=next&project_id=other", "sort": "?marker=next&sort=name:desc", "newfilter": "?marker=next&all_projects=true", "duplicatedproject": "?marker=next&project_id=fixed&project_id=other",
		"limitincrease": "?marker=next&limit=3", "limitreductionstrict": "?marker=next&limit=1", "emptymarker": "?marker=", "duplicatemarker": "?marker=next&marker=other", "fragment": "?marker=next#fragment", "malformedquery": "?marker=next&vendor=%zz",
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				writeList(w, fmt.Sprintf(`{"items":[],"next":%q}`, next))
			})
			_, err := collectList(context.Background(), spec, url.Values{"project_id": {"fixed"}, "sort": {"name:asc"}, "limit": {"2"}})
			var response *resource.ResponseError
			if err == nil || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v response=%v", calls.Load(), err, response)
			}
		})
	}
}

func TestListFirstServerLimitIsOptInAndFixedAcrossContinuations(t *testing.T) {
	for _, check := range []struct {
		name, firstNext, secondNext string
		enabled, wantError          bool
		wantCalls, wantRows         int
	}{
		{"first positive", "?marker=one&limit=25", "", true, false, 2, 2},
		{"fixed later", "?marker=one&limit=25", "?marker=two&limit=25", true, false, 3, 3},
		{"omitted limit retains", "?marker=one&limit=25", "?marker=two", true, false, 3, 3},
		{"opt out", "?marker=one&limit=25", "", false, true, 1, 1},
		{"negative", "?marker=one&limit=-1", "", true, true, 1, 1},
		{"zero", "?marker=one&limit=0", "", true, true, 1, 1},
		{"duplicate", "?marker=one&limit=25&limit=25", "", true, true, 1, 1},
		{"later introduced", "?marker=one", "?marker=two&limit=25", true, true, 2, 2},
		{"later changed", "?marker=one&limit=25", "?marker=two&limit=24", true, true, 2, 2},
		{"nonpagination filter", "?marker=one&limit=25&name=injected", "", true, true, 1, 1},
	} {
		t.Run(check.name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				call := int(calls.Add(1))
				testhelper.TestMethod(t, r, http.MethodGet)
				testhelper.TestHeader(t, r, "X-Auth-Token", "shared-token")
				want := url.Values{}
				if call > 1 {
					want.Set("marker", "one")
					if call == 3 {
						want.Set("marker", "two")
					}
					if strings.Contains(check.firstNext, "limit=25") {
						want.Set("limit", "25")
					}
				}
				if r.URL.Path != "/v1/items" || r.URL.RawQuery != want.Encode() || call > check.wantCalls {
					t.Errorf("continuation changed captured limit/filter: call=%d URL=%s want=%s", call, r.URL, want.Encode())
				}
				next := ""
				if call == 1 {
					next = check.firstNext
				} else if call == 2 {
					next = check.secondNext
				}
				writeList(w, fmt.Sprintf(`{"items":[{"id":"row%d"}],"next":%q}`, call, next))
			})
			spec.Paging.AllowFirstServerLimit = check.enabled
			rows, err := collectList(context.Background(), spec, nil)
			if (err != nil) != check.wantError || int(calls.Load()) != check.wantCalls || len(rows) != check.wantRows {
				t.Fatal("rows/calls/error", len(rows), calls.Load(), err, check.wantRows, check.wantCalls)
			}
			for index, row := range rows {
				if row.ID != fmt.Sprintf("row%d", index+1) || row.StatusCode != 200 || row.Header.Get("X-Page") != "kept" {
					t.Fatal("earlier row/receipt lost", index, row)
				}
			}
			if check.wantError {
				var proof *resource.ResponseError
				next := check.firstNext
				if check.wantCalls == 2 {
					next = check.secondNext
				}
				wantBody := fmt.Sprintf(`{"items":[{"id":"row%d"}],"next":%q}`, check.wantCalls, next)
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" || string(proof.Body) != wantBody {
					t.Fatal("rejected continuation lost actual page proof", err, proof)
				}
			}
		})
	}
}

func TestListCyclesRejectRepeatedMarkerBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("marker") == "" {
			writeList(w, `{"items":[],"next":"?marker=repeated"}`)
		} else {
			writeList(w, `{"items":[],"next":"?marker=repeated&limit=2"}`)
		}
	})
	_, err := collectList(context.Background(), spec, url.Values{"limit": {"2"}})
	if !errors.Is(err, resource.ErrPaginationCycle) || calls.Load() != 2 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestListHTTPLinksAreOptInAndConflictingNextLinksFail(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					w.Header().Set("Link", `</v1/items?marker=next>; rel="next"; title="a,b", </v1/items>; rel="previous"`)
					writeList(w, `{"items":[{"id":"first"}]}`)
				} else {
					writeList(w, `{"items":[{"id":"second"}]}`)
				}
			})
			spec.Paging.HTTPLink = enabled
			values, err := collectList(context.Background(), spec, nil)
			want := int32(1)
			if enabled {
				want = 2
			}
			if err != nil || calls.Load() != want || len(values) != int(want) {
				t.Fatalf("values=%v calls=%d err=%v", values, calls.Load(), err)
			}
		})
	}
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<?marker=two>; rel="next"`)
		writeList(w, `{"items":[],"next":"?marker=one"}`)
	})
	spec.Paging.HTTPLink = true
	if _, err := collectList(context.Background(), spec, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestListEquivalentNextSourcesInheritFiltersBeforeComparison(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("project_id") != "fixed" || r.URL.Query().Get("limit") != "2" {
			t.Errorf("continuation omitted sticky query: %s", r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			w.Header().Set("Link", `<?marker=next&project_id=fixed&limit=2>; rel="next"`)
			writeList(w, `{"items":[],"next":"?marker=next"}`)
		} else {
			writeList(w, `{"items":[{"id":"second"}]}`)
		}
	})
	spec.Paging.HTTPLink = true
	values, err := collectList(context.Background(), spec, url.Values{"project_id": {"fixed"}, "limit": {"2"}})
	if err != nil || len(values) != 1 || calls.Load() != 2 {
		t.Fatalf("equivalent continuation sources conflicted: calls=%d values=%v err=%v", calls.Load(), values, err)
	}
}

func TestListMarkerFallbackUsesWireMarkerAndRequiresExplicitPolicy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					writeList(w, `{"items":[{"id":"sdk-alternate-id","wire_marker":"raw-database-id"}]}`)
				} else {
					if r.URL.Query().Get("marker") != "raw-database-id" || r.URL.Query().Get("project_id") != "fixed" {
						t.Errorf("invented marker or dropped filter: %s", r.URL)
					}
					writeList(w, `{"items":[]}`)
				}
			})
			spec.Paging.MarkerFallback = enabled
			spec.Paging.Marker = func(v *listItem) (string, error) { return v.WireMarker, nil }
			values, err := collectList(context.Background(), spec, url.Values{"limit": {"1"}, "project_id": {"fixed"}})
			want := int32(1)
			if enabled {
				want = 2
			}
			if err != nil || len(values) != 1 || calls.Load() != want {
				t.Fatalf("values=%v calls=%d err=%v", values, calls.Load(), err)
			}
			if enabled {
				values, err := collectList(context.Background(), spec, nil)
				if err != nil || len(values) != 1 || calls.Load() != want+1 {
					t.Fatalf("unbounded list invented continuation: calls=%d values=%v err=%v", calls.Load(), values, err)
				}
				spec.Paging.Marker = nil
				if _, err := collectList(context.Background(), spec, nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != want+1 {
					t.Fatalf("missing wire marker callback made HTTP: %v", err)
				}
			}
		})
	}
}

func TestListFirstLimitReductionCannotChangeAgain(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Query().Get("marker") {
		case "":
			writeList(w, `{"items":[],"next":"?marker=one&limit=2"}`)
		case "one":
			if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("project_id") != "fixed" {
				t.Errorf("reduced limit/filter changed: %s", r.URL)
			}
			writeList(w, `{"items":[],"next":"?marker=two&limit=1"}`)
		default:
			t.Error("second reduction reached HTTP")
		}
	})
	spec.Paging.AllowFirstLimitReduction = true
	_, err := collectList(context.Background(), spec, url.Values{"limit": {"4"}, "project_id": {"fixed"}})
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestListFallbackUsesRetainedWireDataAndFreshSourceToken(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("marker") == "" {
			if r.Header.Get("X-Auth-Token") != "shared-token" {
				t.Errorf("first request lost source token: %v", r.Header)
			}
			writeList(w, `{"items":[{"id":"sdk-id","wire_marker":"raw-id"}]}`)
			return
		}
		if r.URL.Query().Get("marker") != "raw-id" || r.Header.Get("X-Auth-Token") != "refreshed-token" {
			t.Errorf("pagination used changed model or stale token: %s %v", r.URL, r.Header)
		}
		writeList(w, `{"items":[]}`)
	})
	spec.Paging.MarkerFallback = true
	spec.Paging.Marker = func(v *listItem) (string, error) { return v.WireMarker, nil }
	stream := List(context.Background(), spec, url.Values{"limit": {"1"}})
	for value, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
		value.WireMarker = "caller-changed-id"
		value.Body["wire_marker"] = []byte(`"caller-changed-id"`)
		spec.Client.SetToken("refreshed-token")
	}
	if calls.Load() != 2 {
		t.Fatalf("fallback requests=%d", calls.Load())
	}
	spec.Client.SetToken("shared-token")
	for _, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls.Load() != 3 {
		t.Fatalf("iterator was not a fresh request: calls=%d", calls.Load())
	}
}

func TestListResponseAndTransportErrorsKeepEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"syntax", 200, `{"items":`}, {"wrongEnvelope", 200, `{"other":[]}`}, {"nullList", 200, `{"items":null}`}, {"badItem", 200, `{"items":[{"value":"bad"}]}`}, {"nullItem", 200, `{"items":[null]}`}, {"wrongStatus", 202, `{"items":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Evidence", "kept")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			})
			_, err := collectList(context.Background(), spec, nil)
			if tc.status == 200 {
				var response *resource.ResponseError
				if !errors.As(err, &response) || string(response.Body) != tc.body || response.StatusCode != 200 || response.Header.Get("X-Evidence") != "kept" {
					t.Fatalf("evidence lost: %v", err)
				}
			} else {
				var cause gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &cause) || cause.Actual != tc.status || string(cause.Body) != tc.body {
					t.Fatalf("HTTP cause lost: %v", err)
				}
			}
		})
	}
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { writeList(w, `{"items":[]}`) })
	cause := errors.New("source transport failed")
	spec.Client.HTTPClient.Transport = listTransport(func(*http.Request) (*http.Response, error) { return nil, cause })
	if _, err := collectList(context.Background(), spec, nil); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	spec.Client.HTTPClient.Transport = listTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Evidence": {"kept"}}, Body: &failedListBody{prefix: []byte(`{"items":`), cause: cause}, Request: r}, nil
	})
	_, err := collectList(context.Background(), spec, nil)
	var response *resource.ResponseError
	if !errors.Is(err, cause) || !errors.As(err, &response) || string(response.Body) != `{"items":` || response.Header.Get("X-Evidence") != "kept" {
		t.Fatalf("partial read lost: %v", err)
	}
}

type listTransport func(*http.Request) (*http.Response, error)

func (f listTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failedListBody struct {
	prefix []byte
	cause  error
}

func (b *failedListBody) Read(p []byte) (int, error) {
	n := copy(p, b.prefix)
	b.prefix = b.prefix[n:]
	if len(b.prefix) == 0 {
		return n, b.cause
	}
	return n, nil
}
func (*failedListBody) Close() error { return nil }

func TestListPreflightEveryPageAndCancellationStopsHTTP(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"first"}],"next":"?marker=second"}`)
	})
	valid := true
	spec.Validate = func(context.Context) error {
		if !valid {
			return resource.ErrUnsupported
		}
		return nil
	}
	var err error
	for value, e := range List(context.Background(), spec, nil) {
		if e != nil {
			err = e
			break
		}
		if value.ID == "first" {
			valid = false
		}
	}
	if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
	valid = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := collectList(ctx, spec, nil); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("canceled preflight made HTTP: %v", err)
	}
	spec.ValidateQuery = func(_ context.Context, q url.Values) error {
		if q.Has("enabled") {
			return resource.ErrUnsupported
		}
		return nil
	}
	if _, err := Collection(spec).All(context.Background(), resource.WithQuery("enabled", "false")); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
		t.Fatalf("query gate bypassed: %v", err)
	}
	slow := listSpec(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	deadline, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	if _, err := collectList(deadline, slow, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestListRedirectPolicyCannotReplaceVirtualHost(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Redirect(w, r, r.URL.String(), 307) })
	spec.Client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error { r.Host = "foreign.invalid"; return nil }
	if _, err := collectList(context.Background(), spec, nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestCollectionCapabilitiesExactNamesAndDeleteCodes(t *testing.T) {
	var deletes atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deletes.Add(1)
			w.WriteHeader(202)
			return
		}
		if r.URL.Path == "/v1/items/fixed" {
			writeList(w, `{"item":{"id":"fixed","name":"target"}}`)
			return
		}
		if r.URL.Query().Get("marker") == "" {
			writeList(w, `{"items":[{"id":"other","name":"target-other"}],"next":"?marker=next"}`)
		} else {
			writeList(w, `{"items":[{"id":"fixed","name":"target"}]}`)
		}
	})
	spec.DeleteCodes = []int{202}
	collection := Collection(spec)
	value, err := collection.Find(context.Background(), resource.Name("target"))
	if err != nil || value.ID != "fixed" {
		t.Fatalf("value=%v err=%v", value, err)
	}
	value, err = collection.Get(context.Background(), "fixed")
	if err != nil || value.ID != "fixed" || value.StatusCode != 200 || value.Header.Get("X-Page") != "kept" || string(value.Body["name"]) != `"target"` {
		t.Fatalf("singleton evidence lost: value=%+v err=%v", value, err)
	}
	if err := collection.Delete(context.Background(), resource.ID("fixed")); err != nil || deletes.Load() != 1 {
		t.Fatalf("deletes=%d err=%v", deletes.Load(), err)
	}
	spec.Get, spec.Delete = false, false
	if _, err := Collection(spec).Get(context.Background(), "fixed"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := Collection(spec).Delete(context.Background(), resource.ID("fixed")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	// Explicit invalid pagination inputs fail without executing the transport.
	spec.Client.HTTPClient.Transport = listTransport(func(*http.Request) (*http.Response, error) { t.Error("invalid query reached HTTP"); return nil, io.EOF })
	for _, q := range []url.Values{{"marker": {""}}, {"marker": {"a", "b"}}, {"limit": {"0"}}, {"limit": {"1", "2"}}} {
		if _, err := collectList(context.Background(), spec, q); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
}
