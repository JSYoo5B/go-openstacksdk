package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func collectControlledList(ctx context.Context, spec CollectionSpec[listItem], query url.Values, control ListControl) ([]*listItem, error) {
	var values []*listItem
	for value, err := range ListWithControl(ctx, spec, query, control) {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestListControlCapStopsBeforeTrailingDecodeValidationOrContinuation(t *testing.T) {
	for _, trailing := range []string{`false`, `{"id":"bad","value":"wrong type"}`, `{"id":"bad","value":1}`} {
		t.Run(trailing, func(t *testing.T) {
			var calls, validations, markers atomic.Int32
			body := `{"items":[{"id":"first","value":9007199254740993},` + trailing + `],"next":false,"links":"malformed"}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Link", "malformed HTTP link")
				writeList(w, body)
			})
			spec.ValidateItem = func(value *listItem) error {
				validations.Add(1)
				if value.ID != "first" {
					return errors.New("unconsumed row validation")
				}
				return nil
			}
			spec.Paging.HTTPLink, spec.Paging.MarkerFallback = true, true
			spec.Paging.Marker = func(value *listItem) (string, error) {
				markers.Add(1)
				return "", errors.New("unconsumed marker")
			}
			values, err := collectControlledList(context.Background(), spec, url.Values{"limit": {"1"}}, ListControl{MaxItems: 1})
			if err != nil || len(values) != 1 || values[0].ID != "first" || values[0].Value != 9007199254740993 || values[0].Header.Get("X-Page") != "kept" || calls.Load() != 1 || validations.Load() != 1 || markers.Load() != 0 {
				t.Fatal(values, err, calls.Load(), validations.Load(), markers.Load())
			}
		})
	}
}

func TestListControlCountsValidatedRawRowsBeforeLocalFiltering(t *testing.T) {
	var calls, validations atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"one","name":"excluded"},{"id":"two","name":"excluded"},{"id":"three","name":"match"}],"next":"?marker=unvisited"}`)
	})
	spec.ValidateItem = func(value *listItem) error { validations.Add(1); return nil }
	matched := 0
	for value, err := range ListWithControl(context.Background(), spec, nil, ListControl{MaxItems: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		if value.Name == "match" {
			matched++
		}
	}
	if matched != 0 || validations.Load() != 2 || calls.Load() != 1 {
		t.Fatal("cap refilled locally filtered rows", matched, validations.Load(), calls.Load())
	}
}

func TestListControlSinglePageSkipsNextButValidatesAllConsumedRows(t *testing.T) {
	for _, malformedRow := range []bool{false, true} {
		t.Run(fmt.Sprint(malformedRow), func(t *testing.T) {
			var calls, validations atomic.Int32
			second := `{"id":"second"}`
			if malformedRow {
				second = `{"id":"second","value":"not an integer"}`
			}
			body := `{"items":[{"id":"first"},` + second + `],"next":{},"items_links":false}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Link", "malformed")
				writeList(w, body)
			})
			spec.Paging.HTTPLink = true
			spec.ValidateItem = func(value *listItem) error { validations.Add(1); return nil }
			values, err := collectControlledList(context.Background(), spec, nil, ListControl{SinglePage: true})
			if malformedRow {
				var proof *resource.ResponseError
				if len(values) != 1 || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" || validations.Load() != 1 {
					t.Fatal(values, err, proof, validations.Load())
				}
			} else if err != nil || len(values) != 2 || validations.Load() != 2 {
				t.Fatal(values, err, validations.Load())
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestListControlLimitHintOwnsInitialQueryAndPreservesExplicitLimit(t *testing.T) {
	for _, item := range []struct {
		name  string
		limit string
		hint  bool
		want  string
	}{{name: "hint", hint: true, want: "3"}, {name: "explicit", limit: "8", hint: true, want: "8"}, {name: "opt-in-only", want: ""}} {
		t.Run(item.name, func(t *testing.T) {
			var calls, initialChecks, pageChecks, sourceChecks atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != item.want || r.URL.Query().Get("owner") != "fixed" || fmt.Sprint(r.URL.Query()["vendor"]) != "[a b]" {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					writeList(w, `{"items":[{"id":"one"}],"next":"?marker=first"}`)
				} else if r.URL.Query().Get("marker") == "first" {
					writeList(w, `{"items":[{"id":"two"},{"id":"three"},{"value":"unconsumed"}],"next":false}`)
				} else {
					t.Error(r.URL)
				}
			})
			spec.Validate = func(context.Context) error { sourceChecks.Add(1); return nil }
			spec.ValidateInitialQuery = func(_ context.Context, query url.Values) error {
				initialChecks.Add(1)
				if query.Get("limit") != item.want {
					return errors.New("initial validation did not receive the effective hint")
				}
				query.Set("limit", "99")
				query["vendor"][0] = "validator"
				return nil
			}
			spec.ValidateQuery = func(_ context.Context, query url.Values) error {
				pageChecks.Add(1)
				query.Set("owner", "validator")
				return nil
			}
			query := url.Values{"owner": {"fixed"}, "vendor": {"a", "b"}}
			if item.limit != "" {
				query.Set("limit", item.limit)
			}
			control := ListControl{MaxItems: 3, LimitHint: item.hint}
			stream := ListWithControl(context.Background(), spec, query, control)
			if query.Get("limit") != item.limit {
				t.Fatal("hint mutated caller query", query)
			}
			query.Set("owner", "caller")
			query["vendor"][0] = "caller"
			control.MaxItems = 1
			if calls.Load() != 0 || initialChecks.Load() != 0 || sourceChecks.Load() != 0 {
				t.Fatal("iterator was eager")
			}
			for repetition := 1; repetition <= 2; repetition++ {
				count := 0
				for value, err := range stream {
					if err != nil {
						t.Fatal(err)
					}
					count++
					value.Header.Set("X-Page", "consumer")
				}
				if count != 3 || calls.Load() != int32(2*repetition) || sourceChecks.Load() != int32(2*repetition) || pageChecks.Load() != int32(2*repetition) || initialChecks.Load() != int32(repetition) {
					t.Fatal(count, calls.Load(), sourceChecks.Load(), pageChecks.Load(), initialChecks.Load())
				}
			}
		})
	}
}

func TestListControlZeroPreservesLegacyPagingAndEmptyStopIsOptIn(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "" {
					t.Error("unbounded control emitted a limit", r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					writeList(w, `{"items":[],"next":"?marker=next"}`)
				} else {
					writeList(w, `{"items":[{"id":"next"}]}`)
				}
			})
			spec.Paging.StopOnEmptyPage = stop
			values, err := collectControlledList(context.Background(), spec, nil, ListControl{LimitHint: true})
			wantCalls, wantItems := int32(2), 1
			if stop {
				wantCalls, wantItems = 1, 0
			}
			if err != nil || len(values) != wantItems || calls.Load() != wantCalls {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
	for _, stop := range []bool{false, true} {
		spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { writeList(w, `{"items":[],"next":false}`) })
		spec.Paging.StopOnEmptyPage = stop
		_, err := collectControlledList(context.Background(), spec, nil, ListControl{})
		if stop && err != nil || !stop && err == nil {
			t.Fatal(stop, err)
		}
	}
}

func TestListControlPreflightAndHintValidationMakeNoHTTP(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid control reached HTTP", r.URL)
	})
	for _, item := range []struct {
		query   url.Values
		control ListControl
	}{{control: ListControl{MaxItems: -1}}, {query: url.Values{"limit": {""}}, control: ListControl{MaxItems: 1, LimitHint: true}}, {query: url.Values{"limit": {"2", "3"}}, control: ListControl{MaxItems: 1, LimitHint: true}}} {
		values, err := collectControlledList(context.Background(), spec, item.query, item.control)
		if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(values, err)
		}
	}
	cause := errors.New("limit hints unsupported by this service")
	spec.ValidateInitialQuery = func(_ context.Context, query url.Values) error {
		if query.Has("limit") {
			return cause
		}
		return nil
	}
	if _, err := collectControlledList(context.Background(), spec, nil, ListControl{MaxItems: 1, LimitHint: true}); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := collectControlledList(ctx, spec, nil, ListControl{MaxItems: 1}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestListControlCapStillValidatesConsumedRowsAndKeepsWholePageEvidence(t *testing.T) {
	for _, malformed := range []string{`{"id":"second","value":"bad"}`, `{"id":"rejected","value":1}`} {
		t.Run(malformed, func(t *testing.T) {
			var calls atomic.Int32
			body := `{"items":[{"id":"first"},` + malformed + `],"next":false}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); writeList(w, body) })
			cause := errors.New("wrong scoped identity")
			spec.ValidateItem = func(value *listItem) error {
				if value.ID == "rejected" {
					value.Header.Set("X-Page", "consumer")
					value.Body["id"][0] = '['
					return cause
				}
				return nil
			}
			values, err := collectControlledList(context.Background(), spec, nil, ListControl{MaxItems: 2})
			var proof *resource.ResponseError
			if len(values) != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Page") != "kept" || calls.Load() != 1 {
				t.Fatal(values, err, proof, calls.Load())
			}
			if values[0].ID != "first" || malformed == `{"id":"rejected","value":1}` && !errors.Is(err, cause) {
				t.Fatal(values, err)
			}
		})
	}
}

func TestListControlAcrossPagesRetainsLateHTTPErrorsAndSourceGuards(t *testing.T) {
	for _, mode := range []string{"late-http", "source-change", "cap-before-late-http"} {
		t.Run(mode, func(t *testing.T) {
			var calls, checks atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					writeList(w, `{"items":[{"id":"first"}],"next":"?marker=next"}`)
				} else {
					w.Header().Set("X-Page", "native")
					w.WriteHeader(403)
					fmt.Fprint(w, `{"error":"denied"}`)
				}
			})
			cause := errors.New("selected source version changed")
			spec.Validate = func(context.Context) error {
				if checks.Add(1) == 2 && mode == "source-change" {
					return cause
				}
				return nil
			}
			control := ListControl{MaxItems: 2}
			if mode == "cap-before-late-http" {
				control.MaxItems = 1
			}
			values, err := collectControlledList(context.Background(), spec, nil, control)
			if len(values) != 1 || values[0].ID != "first" {
				t.Fatal(values, err)
			}
			if mode == "late-http" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || !gophercloud.ResponseCodeIs(err, 403) || string(native.Body) != `{"error":"denied"}` || native.ResponseHeader.Get("X-Page") != "native" || calls.Load() != 2 || checks.Load() != 2 {
					t.Fatal(err, calls.Load(), checks.Load())
				}
			} else if mode == "source-change" {
				if !errors.Is(err, cause) || calls.Load() != 1 || checks.Load() != 2 {
					t.Fatal(err, calls.Load(), checks.Load())
				}
			} else if err != nil || calls.Load() != 1 || checks.Load() != 1 {
				t.Fatal(err, calls.Load(), checks.Load())
			}
		})
	}
}

func TestListControlCancellationAtCapPreservesAcceptedPageAndBreakStaysLazy(t *testing.T) {
	for _, breakEarly := range []bool{false, true} {
		t.Run(fmt.Sprint(breakEarly), func(t *testing.T) {
			var calls, validations atomic.Int32
			body := `{"items":[{"id":"first"},{"value":"unconsumed"}],"next":false}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); writeList(w, body) })
			spec.ValidateItem = func(value *listItem) error { validations.Add(1); return nil }
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stream := ListWithControl(ctx, spec, nil, ListControl{MaxItems: 1})
			if calls.Load() != 0 || validations.Load() != 0 {
				t.Fatal("creating stream fetched or validated rows")
			}
			seen := 0
			var failure error
			for value, err := range stream {
				if err != nil {
					failure = err
					continue
				}
				if value.ID != "first" {
					t.Fatal(value)
				}
				seen++
				cancel()
				if breakEarly {
					break
				}
			}
			if !breakEarly {
				var proof *resource.ResponseError
				if !errors.Is(failure, context.Canceled) || !errors.As(failure, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" {
					t.Fatal(failure, proof)
				}
			} else if failure != nil {
				t.Fatal(failure)
			}
			if seen != 1 || calls.Load() != 1 || validations.Load() != 1 {
				t.Fatal(seen, calls.Load(), validations.Load())
			}
		})
	}
}

func TestCollectionListControlCapsRawRowsBeforeNameFiltersAndHintsOptIn(t *testing.T) {
	for _, hint := range []bool{false, true} {
		t.Run(fmt.Sprint(hint), func(t *testing.T) {
			var calls, validations atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				limit := ""
				if hint {
					limit = "2"
				}
				if r.URL.Query().Get("limit") != limit || r.URL.Query().Get("name") != "match" {
					t.Error(r.URL)
				}
				writeList(w, `{"items":[{"id":"one","name":"excluded"},{"id":"two","name":"excluded"},{"id":"three","name":"match","value":"unconsumed"}],"next":false}`)
			})
			spec.Paging.MaxItemsLimitHint = hint
			spec.NameQuery = func(name string) string { return name }
			spec.ValidateItem = func(value *listItem) error { validations.Add(1); return nil }
			values, err := Collection(spec).All(context.Background(), resource.WithMaxItems(2), resource.WithName("match"))
			if err != nil || len(values) != 0 || calls.Load() != 1 || validations.Load() != 2 {
				t.Fatal(values, err, calls.Load(), validations.Load())
			}
		})
	}
}

func TestCollectionListControlForwardsSinglePageAndPreservesExplicitLimit(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "8" {
			t.Error("collection overwrote explicit wire limit", r.URL)
		}
		writeList(w, `{"items":[{"id":"one"},{"id":"two"}],"next":false}`)
	})
	spec.Paging.MaxItemsLimitHint = true
	values, err := Collection(spec).All(context.Background(), resource.WithMaxItems(5), resource.WithPaginated(false), resource.WithPageSize(8))
	if err != nil || len(values) != 2 || calls.Load() != 1 || values[0].Header.Get("X-Page") != "kept" || values[1].StatusCode != 200 {
		t.Fatal(values, err, calls.Load())
	}
}
