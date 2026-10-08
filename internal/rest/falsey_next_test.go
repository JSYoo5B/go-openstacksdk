package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestListFalseyNextIsOptInAndAllowsHeaderOrMarkerFallback(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `-0`, `0.000`, `-0e999999999999`, `0E-999999999999`, `""`, `[]`, `{}`} {
		for _, fallback := range []string{"none", "header", "marker"} {
			t.Run(raw+"/"+fallback, func(t *testing.T) {
				var calls atomic.Int32
				body := fmt.Sprintf(`{"items":[{"id":"one","wire_marker":"wire-one"}],"next":%s}`, raw)
				spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Get("marker") == "" {
						if fallback == "header" {
							w.Header().Set("Link", `<?marker=wire-one>; rel="next"`)
						}
						writeList(w, body)
					} else {
						if r.URL.Query().Get("marker") != "wire-one" {
							t.Error(r.URL)
						}
						writeList(w, `{"items":[]}`)
					}
				})
				spec.Paging.IgnoreFalseyNext = true
				spec.Paging.HTTPLink = true
				query := url.Values(nil)
				if fallback == "marker" {
					spec.Paging.MarkerFallback = true
					spec.Paging.MarkerOnShortPage = true
					spec.Paging.Marker = func(value *listItem) (string, error) { return value.WireMarker, nil }
					query = url.Values{"limit": {"1"}}
				}
				values, err := collectList(context.Background(), spec, query)
				want := int32(1)
				if fallback != "none" {
					want = 2
				}
				if err != nil || len(values) != 1 || calls.Load() != want || values[0].Header.Get("X-Page") != "kept" {
					t.Fatal(values, err, calls.Load())
				}
			})
		}
	}
}

func TestListTruthyNonstringNextAndDefaultFalseyTypeKeepExactPageEvidence(t *testing.T) {
	for _, check := range []struct {
		raw     string
		enabled bool
	}{
		{`true`, true}, {`1`, true}, {`1e-999999999`, true}, {`[false]`, true}, {`{"x":null}`, true},
		{`false`, false}, {`0`, false}, {`[]`, false}, {`{}`, false},
	} {
		t.Run(fmt.Sprint(check.enabled)+"/"+check.raw, func(t *testing.T) {
			var calls atomic.Int32
			body := fmt.Sprintf(`{"items":[{"id":"one"}],"next":%s}`, check.raw)
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				// A truthy invalid field must fail before this fallback is followed.
				w.Header().Set("Link", `<?marker=second>; rel="next"`)
				writeList(w, body)
			})
			spec.Paging.IgnoreFalseyNext = check.enabled
			spec.Paging.HTTPLink = true
			values, err := collectList(context.Background(), spec, nil)
			var proof *resource.ResponseError
			if len(values) != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Page") != "kept" || calls.Load() != 1 {
				t.Fatal(values, err, proof, calls.Load())
			}
			if !strings.Contains(err.Error(), "string") {
				t.Fatal(err)
			}
		})
	}
}
