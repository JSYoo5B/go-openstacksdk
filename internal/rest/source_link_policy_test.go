package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestSourceLinkPolicyRootPresenceAndDictionaryIgnoreAreOptIn(t *testing.T) {
	for _, tc := range []struct {
		name, fields              string
		source, dictInterop, fail bool
		wantMarker                string
		calls                     int
	}{
		{"Source dict next ignored", `"links":{"next":"?marker=dict"}`, true, true, false, "", 1},
		{"Source root empty shadows plural", `"links":[],"items_links":[{"rel":"next","href":"?marker=plural"}]`, true, false, false, "", 1},
		{"Source root dict shadows plural", `"links":{"next":"?marker=dict"},"items_links":[{"rel":"next","href":"?marker=plural"}]`, true, true, false, "", 1},
		{"Source plural only", `"items_links":[{"rel":"next","href":"?marker=plural"}]`, true, false, false, "plural", 2},
		{"Source root wins conflicting plural", `"links":[{"rel":"next","href":"?marker=root"}],"items_links":[{"rel":"next","href":"?marker=plural"}]`, true, false, false, "root", 2},
		{"default dictionaries stay strict", `"links":{"next":"?marker=dict"}`, false, false, true, "", 1},
		{"default dictionary interoperability", `"links":{"next":"?marker=dict"}`, false, true, false, "dict", 2},
		{"default conflicting roots unchanged", `"links":[{"rel":"next","href":"?marker=root"}],"items_links":[{"rel":"next","href":"?marker=plural"}]`, false, false, true, "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				index := calls.Add(1)
				if index == 1 {
					writeList(w, `{"items":[{"id":"first"}],`+tc.fields+`}`)
				} else {
					if r.URL.Query().Get("marker") != tc.wantMarker {
						t.Error(r.URL)
					}
					writeList(w, `{"items":[]}`)
				}
			})
			spec.Paging.SourceLinkPolicy, spec.Paging.DictionaryLinks = tc.source, tc.dictInterop
			rows, err := collectList(context.Background(), spec, nil)
			if (err != nil) != tc.fail || int(calls.Load()) != tc.calls || len(rows) != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if tc.fail {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestSourceLinkPolicyFirstLiteralCandidateAndTopNextFallback(t *testing.T) {
	for _, tc := range []struct {
		name, fields, want string
		fail               bool
	}{
		{"first literal next", `"links":[{"rel":"next","href":"?marker=first"},{"rel":"next","href":"?marker=second"}],"next":true`, "first", false},
		{"upper relation ignored", `"links":[{"rel":"NEXT","href":"?marker=upper"}],"next":"?marker=top"`, "top", false},
		{"missing href continues", `"links":[{"rel":"next"},{"rel":"next","href":"?marker=second"}]`, "second", false},
		{"false first href breaks then top", `"links":[{"rel":"next","href":false},{"rel":"next","href":"?marker=second"}],"next":"?marker=top"`, "top", false},
		{"null first href breaks without later candidate", `"links":[{"rel":"next","href":null},{"rel":"next","href":"?marker=second"}]`, "", false},
		{"empty object href falsey", `"links":[{"rel":"next","href":{}}],"next":"?marker=top"`, "top", false},
		{"loaded float href underflows", `"links":[{"rel":"next","href":1e-400}],"next":"?marker=top"`, "top", false},
		{"truthy nonstring href errors", `"links":[{"rel":"next","href":[1]}],"next":"?marker=top"`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					writeList(w, `{"items":[{"id":"one"}],`+tc.fields+`}`)
				} else {
					if r.URL.Query().Get("marker") != tc.want {
						t.Error(r.URL)
					}
					writeList(w, `{"items":[]}`)
				}
			})
			spec.Paging.SourceLinkPolicy = true
			rows, err := collectList(context.Background(), spec, nil)
			expected := 1
			if tc.want != "" {
				expected = 2
			}
			if (err != nil) != tc.fail || calls != expected || len(rows) != 1 {
				t.Fatal(rows, err, calls)
			}
			if tc.fail {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestSourceLinkPolicyFiniteJSONShapesPreserveActualPageProof(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		fail bool
	}{
		{`{}`, false}, {`{"next":"?marker=ignored","self":{"x":1}}`, false}, {`{"rel":"next","href":"?marker=ignored"}`, false},
		{`[]`, false}, {`""`, false}, {`null`, true}, {`false`, true}, {`0`, true}, {`"wrong"`, true},
		{`[null]`, true}, {`[false]`, true}, {`[1]`, true}, {`["x"]`, true}, {`[[]]`, true}, {`[{}]`, false},
		{`[{"rel":null,"href":true}]`, false}, {`[{"rel":[],"href":true}]`, false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			calls := 0
			body := `{"items":[{"id":"one"}],"links":` + tc.raw + `}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls++; writeList(w, body) })
			spec.Paging.SourceLinkPolicy = true
			rows, err := collectList(context.Background(), spec, nil)
			if (err != nil) != tc.fail || calls != 1 || len(rows) != 1 {
				t.Fatal(rows, err, calls)
			}
			if tc.fail {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 || proof.Header.Get("X-Page") != "kept" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestSourceLinkPolicyFalseyTopNextAndTruthyNonstringErrors(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `-0.0`, `1e-400`, `""`, `[]`, `{}`, `true`, `1`, `1e400`, `[1]`, `{"a":1}`} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			body := `{"items":[{"id":"one"}],"next":` + raw + `}`
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls++; writeList(w, body) })
			spec.Paging.SourceLinkPolicy = true
			truthy, err := cloudfilter.PythonJSONTruthy(json.RawMessage(raw))
			if err != nil {
				t.Fatal(err)
			}
			rows, err := collectList(context.Background(), spec, nil)
			if (err != nil) != truthy || calls != 1 || len(rows) != 1 {
				t.Fatal(rows, err, calls)
			}
			if truthy {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 {
					t.Fatal(err, proof)
				}
			}
		})
	}
}

func TestSourceLinkPolicyHeaderInteropCustomKeysAndScopeRemainExplicit(t *testing.T) {
	for _, tc := range []struct {
		name, body, header string
		keys               []string
		http, fail         bool
		want               string
	}{
		{"Source default ignores header", `"links":{}`, `<?marker=header>; rel="next"`, nil, false, false, ""},
		{"Header interoperability stays optin", `"links":{}`, `<?marker=header>; rel="next"`, nil, true, false, "header"},
		{"Header conflict keeps existing guard", `"links":[{"rel":"next","href":"?marker=body"}]`, `<?marker=header>; rel="next"`, nil, true, true, ""},
		{"custom present key wins", `"custom":[],"links":[{"rel":"next","href":"?marker=body"}]`, ``, []string{"custom", "links"}, false, false, ""},
		{"custom absent falls to present", `"links":[{"rel":"next","href":"?marker=body"}]`, ``, []string{"custom", "links"}, false, false, "body"},
		{"foreign scope stays forbidden", `"links":[{"rel":"next","href":"https://foreign.invalid/v1/items?marker=x"}]`, ``, nil, false, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					if tc.header != "" {
						w.Header().Set("Link", tc.header)
					}
					writeList(w, `{"items":[{"id":"one"}],`+tc.body+`}`)
				} else {
					if r.URL.Query().Get("marker") != tc.want {
						t.Error(r.URL)
					}
					writeList(w, `{"items":[]}`)
				}
			})
			spec.Paging.SourceLinkPolicy, spec.Paging.HTTPLink, spec.Paging.LinkKeys = true, tc.http, tc.keys
			rows, err := collectList(context.Background(), spec, nil)
			wantCalls := 1
			if tc.want != "" {
				wantCalls = 2
			}
			if (err != nil) != tc.fail || calls != wantCalls || len(rows) != 1 {
				t.Fatal(rows, err, calls)
			}
			if tc.fail && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestSourceLinkPolicyTruthyCallbackOwnsRawAndControlStopsBeforeLinks(t *testing.T) {
	body := `{"items":[{"id":"one"}],"links":null,"next":true}`
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { writeList(w, body) })
	spec.Paging.SourceLinkPolicy = true
	for _, control := range []ListControl{{SinglePage: true}, {MaxItems: 1}} {
		rows, err := collectControlledList(context.Background(), spec, nil, control)
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	}
	raw := json.RawMessage(`"?marker=next"`)
	fields := map[string]json.RawMessage{"next": raw}
	var observed []string
	truthy := func(value json.RawMessage) (bool, error) {
		observed = append(observed, string(value))
		if len(value) > 0 {
			value[0] = 'X'
		}
		return len(observed) > 1, nil
	}
	link, err := sourceJSONContinuation(fields, []string{"links"}, "next", truthy)
	if err != nil || link != "?marker=next" {
		t.Fatal(link, err)
	}
	if string(raw) != `"?marker=next"` || !reflect.DeepEqual(observed, []string{"", `"?marker=next"`}) {
		t.Fatal(string(raw), observed, err)
	}
	// A callback error remains a handling error, with actual page proof in List.
	cause := errors.New("owned truthiness failed")
	body = `{"items":[{"id":"one"}],"next":false}`
	spec.Paging.NextTruthy = func(json.RawMessage) (bool, error) { return false, cause }
	_, err = collectList(context.Background(), spec, url.Values{})
	var proof *resource.ResponseError
	if !errors.Is(err, cause) || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 {
		t.Fatal(fmt.Sprint(err), proof)
	}
}
