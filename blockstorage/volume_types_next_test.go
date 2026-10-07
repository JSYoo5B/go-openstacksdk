package blockstorage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

func volumeTypesPaginationFixture(t *testing.T, body, current string, header http.Header) (*rest.Response, *url.URL, map[string]json.RawMessage) {
	t.Helper()
	fields, err := attachmentObject([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(current)
	if err != nil {
		t.Fatal(err)
	}
	return &rest.Response{Body: json.RawMessage(body), Header: header, StatusCode: http.StatusOK}, target, fields
}

func TestVolumeTypesNextSourcePrecedenceAndFirstConsumedHref(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	cases := []struct{ name, body, want string }{
		{"top links presence suppresses plural", `{"links":[],"volume_types_links":[{"rel":"next","href":"?marker=plural"}]}`, ""},
		{"dict iteration cannot combine rel href", `{"links":{"rel":"next","href":"?marker=wrong"},"next":"?marker=top"}`, base + "?marker=top"},
		{"plural source key", `{"volume_types_links":[{"rel":"next","href":"?marker=plural"}]}`, base + "?marker=plural"},
		{"first source next wins and tail unused", `{"links":[{"rel":"next","href":"?marker=first"},false,{"rel":"next","href":"?marker=second"}]}`, base + "?marker=first"},
		{"missing href continues", `{"links":[{"rel":"next"},{"rel":"next","href":"?marker=second"}]}`, base + "?marker=second"},
		{"falsey first href stops before malformed tail", `{"links":[{"rel":"next","href":null},false,{"rel":"next","href":"?marker=wrong"}],"next":"?marker=top"}`, base + "?marker=top"},
		{"exact source relation", `{"links":[{"rel":"NEXT","href":"?marker=wrong"},{"rel":true,"href":"?marker=wrong"}],"next":"?marker=top"}`, base + "?marker=top"},
		{"empty string iterable", `{"links":"","next":"?marker=top"}`, base + "?marker=top"},
		{"unconsumed malformed plural", `{"links":[],"volume_types_links":false,"next":"?marker=top"}`, base + "?marker=top"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, tc.body, base, nil)
			next, err := volumeTypesNext(response, current, fields)
			if err != nil {
				t.Fatal(err)
			}
			got := ""
			if next != nil {
				got = next.String()
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVolumeTypesNextConsumedShapesAndFalseyValues(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	for _, raw := range []string{"null", "false", "true", "0", "1", `"x"`, `[null]`, `[1]`, `[[]]`} {
		t.Run("bad-links-"+raw, func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, `{"links":`+raw+`,"next":"?marker=top","volume_type_links":[{"rel":"next","href":"?marker=native"}]}`, base, nil)
			if _, err := volumeTypesNext(response, current, fields); err == nil {
				t.Fatal("consumed noniterable or nonobject links must fail before fallback")
			}
		})
	}
	for _, raw := range []string{"null", "false", "0", "-0.00e4", `""`, "[]", "{}"} {
		t.Run("falsey-href-"+raw, func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, `{"links":[{"rel":"next","href":`+raw+`}],"next":"?marker=top"}`, base, nil)
			next, err := volumeTypesNext(response, current, fields)
			if err != nil || next == nil || next.Query().Get("marker") != "top" {
				t.Fatalf("next=%v, err=%v", next, err)
			}
		})
	}
	for _, raw := range []string{"true", "1", "1e-9999", "[0]", `{"x":null}`} {
		t.Run("truthy-href-"+raw, func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, `{"links":[{"rel":"next","href":`+raw+`}],"next":"?marker=top"}`, base, nil)
			if _, err := volumeTypesNext(response, current, fields); err == nil {
				t.Fatal("truthy nonstring href must fail")
			}
		})
	}
}

func TestVolumeTypesNextBodyThenHTTPThenNativeCompatibility(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	header := http.Header{"Link": {`<?marker=http>; rel="next"`}}
	cases := []struct {
		name, body, marker string
		header             http.Header
	}{
		{"body next wins", `{"next":"?marker=body","volume_type_links":[{"rel":"next","href":"?marker=native"}]}`, "body", header},
		{"HTTP wins over native", `{"next":false,"volume_type_links":[{"rel":"next","href":"?marker=native"}]}`, "http", header},
		{"native singular last matching next", `{"volume_type_links":[{"rel":"next","href":"?marker=first"},{"rel":"NEXT","href":"?marker=wrong"},{"rel":"next","href":"?marker=last"}]}`, "last", nil},
		{"native late empty next clears first", `{"volume_type_links":[{"rel":"next","href":"?marker=first"},{"rel":"next","href":""}]}`, "", nil},
		{"native null links empty", `{"volume_type_links":null}`, "", nil},
		{"unused native malformed ignored", `{"next":"?marker=body","volume_type_links":false}`, "body", nil},
		{"unused malformed HTTP ignored", `{"next":"?marker=body"}`, "body", http.Header{"Link": {"malformed"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, tc.body, base, tc.header)
			next, err := volumeTypesNext(response, current, fields)
			if err != nil {
				t.Fatal(err)
			}
			if tc.marker == "" {
				if next != nil {
					t.Fatalf("unexpected next %v", next)
				}
				return
			}
			if next == nil || next.Query().Get("marker") != tc.marker {
				t.Fatalf("next=%v, want marker %q", next, tc.marker)
			}
		})
	}
	for _, body := range []string{`{"volume_type_links":false}`, `{"volume_type_links":[{"rel":"next","href":1}]}`} {
		response, current, fields := volumeTypesPaginationFixture(t, body, base, nil)
		if _, err := volumeTypesNext(response, current, fields); err == nil {
			t.Fatal("consumed invalid native links must fail")
		}
	}
	response, current, fields := volumeTypesPaginationFixture(t, `{"volume_type_links":[{"rel":"next","href":"?marker=native"}]}`, base, http.Header{"Link": {"malformed"}})
	if _, err := volumeTypesNext(response, current, fields); err == nil {
		t.Fatal("consumed invalid HTTP must not fall through to native")
	}
}

func TestVolumeTypesNextHTTPQuotedLinksUseLastNextAcrossHeaders(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	header := http.Header{"Link": {
		`<https://cinder.example/v3/project/types?marker=a,b>; rel="next"; title="a,b;c", <?marker=unused>; rel="prev"`,
		`<?marker=last>; rel="prev NEXT"; title="quoted \"comma,\""`,
	}}
	response, current, fields := volumeTypesPaginationFixture(t, `{}`, base, header)
	next, err := volumeTypesNext(response, current, fields)
	if err != nil || next == nil || next.Query().Get("marker") != "last" {
		t.Fatalf("next=%v, err=%v", next, err)
	}
	response.Header = http.Header{"Link": {" ", `<?marker=prev>; rel="prev"`}}
	next, err = volumeTypesNext(response, current, fields)
	if err != nil || next != nil {
		t.Fatalf("unexpected next=%v, err=%v", next, err)
	}
}

func TestVolumeTypesNextQueryMergeDropsBlankValuesAndOwnsInputs(t *testing.T) {
	const currentURL = "https://cinder.example/v3/project/types?is_public=none&marker=old&limit=5&foo=keep&z=old"
	response, current, fields := volumeTypesPaginationFixture(t, `{"next":"?marker=new&foo=&limit=&blank=&repeat=a&repeat=&repeat=b&flag&z=new&is_public=false"}`, currentURL, http.Header{"X-Evidence": {"original"}})
	bodyBefore := bytes.Clone(response.Body)
	fieldsBefore := make(map[string]json.RawMessage)
	for key, raw := range fields {
		fieldsBefore[key] = bytes.Clone(raw)
	}
	headersBefore := response.Header.Clone()
	next, err := volumeTypesNext(response, current, fields)
	if err != nil {
		t.Fatal(err)
	}
	want := url.Values{"is_public": {"false"}, "marker": {"new"}, "foo": {"keep"}, "z": {"new"}, "repeat": {"a", "b"}}
	if !reflect.DeepEqual(next.Query(), want) {
		t.Fatalf("got %v, want %v", next.Query(), want)
	}
	if current.String() != currentURL || !bytes.Equal(response.Body, bodyBefore) || !reflect.DeepEqual(fields, fieldsBefore) || !reflect.DeepEqual(response.Header, headersBefore) {
		t.Fatal("resolver mutated its inputs")
	}
	response, current, fields = volumeTypesPaginationFixture(t, `{"next":"?marker=second"}`, "https://cinder.example/v3/project/types?is_public=none&marker=first&limit=2", nil)
	next, err = volumeTypesNext(response, current, fields)
	if err != nil || next.Query().Get("is_public") != "none" || next.Query().Has("limit") {
		t.Fatalf("omitted filter must persist and old limit disappear: next=%v err=%v", next, err)
	}
}

func TestVolumeTypesNextChecksOriginalHrefBeforeURLSerialization(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	for _, href := range []string{"?marker=x#", "?marker=x#fragment", "https://user@cinder.example/v3/project/types", "mailto:opaque", "?marker=%", "?marker=a;b", "?marker=\n", string([]byte{0xff})} {
		t.Run(strings.ReplaceAll(href, "\n", "newline"), func(t *testing.T) {
			response, current, fields := volumeTypesPaginationFixture(t, `{}`, base, nil)
			// Invalid UTF-8 cannot be represented by Marshal without repair;
			// inject it in the consumed raw string to test original evidence.
			if strings.Contains(href, string([]byte{0xff})) {
				fields["next"] = append(append([]byte{'"'}, []byte(href)...), '"')
			} else {
				fields["next"], _ = json.Marshal(href)
			}
			if _, err := volumeTypesNext(response, current, fields); err == nil {
				t.Fatal("unsafe or malformed advertised URL must fail")
			}
		})
	}
}

func TestVolumeTypesNextRequiresReaderOriginPathAndCycleValidation(t *testing.T) {
	const base = "https://cinder.example/v3/project/types"
	initial, _ := url.Parse(base)
	reader := &preparedGetVolumes{initialURL: initial, cinder: attachSource{client: gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://cinder.example/v3/project/", Type: "volumev3"}}}
	for _, href := range []string{"https://other.example/v3/project/types?marker=next", "/v3/project/volumes?marker=next", "/v3/project/%74ypes?marker=next"} {
		encoded, _ := json.Marshal(href)
		response, current, fields := volumeTypesPaginationFixture(t, `{"next":`+string(encoded)+`}`, base, nil)
		next, err := volumeTypesNext(response, current, fields)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reader.continuation(current, next.String()); err == nil {
			t.Fatalf("mandatory caller guard accepted %v", next)
		}
	}
	response, current, fields := volumeTypesPaginationFixture(t, `{"next":"?marker=b&repeat=2&repeat=1"}`, base+"?is_public=none&marker=a", nil)
	next, err := volumeTypesNext(response, current, fields)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := reader.continuation(current, next.String())
	if err != nil {
		t.Fatal(err)
	}
	key, err := reader.pageKey(checked)
	if err != nil {
		t.Fatal(err)
	}
	equivalent, _ := url.Parse(base + "?repeat=1&is_public=none&repeat=2&marker=b")
	otherKey, err := reader.pageKey(equivalent)
	if err != nil || key != otherKey {
		t.Fatalf("cycle identity differs: %q / %q err=%v", key, otherKey, err)
	}
}
