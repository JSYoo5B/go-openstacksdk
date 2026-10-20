package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const snapshotPagerContractCollection = "https://cinder.example/v3/project/snapshots/detail"

func snapshotPagerContractFields(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func snapshotPagerContractQuery(t *testing.T, target string) url.Values {
	t.Helper()
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		t.Fatal(err)
	}
	return query
}

func TestSnapshotPagerBodyLinksPrecedePluralNextAndHeaderWithoutDictionaryRepair(t *testing.T) {
	header := http.Header{"Link": {`<` + snapshotPagerContractCollection + `?marker=header>; rel="next"`}}
	for _, tc := range []struct{ body, want string }{
		{`{"links":[{"rel":"next","href":"?marker=body"}],"snapshots_links":[{"rel":"next","href":"?marker=plural"}],"next":"?marker=top"}`, `"?marker=body"`},
		{`{"snapshots_links":[{"rel":"next","href":"?marker=plural"}],"next":"?marker=top"}`, `"?marker=plural"`},
		{`{"links":[],"snapshots_links":[{"rel":"next","href":"?marker=ignored"}],"next":"?marker=top"}`, `"?marker=top"`},
		{`{"links":{"next":"https://evil.example/no-repair"},"next":"?marker=top"}`, `"?marker=top"`},
		{`{"links":{"rel":"next","href":"?marker=not-one-item"},"next":"?marker=top"}`, `"?marker=top"`},
		{`{"links":{"next":null},"snapshots_links":[{"rel":"next","href":"?marker=ignored"}]}`, `"` + snapshotPagerContractCollection + `?marker=header"`},
		{`{"links":"","next":"?marker=top"}`, `"?marker=top"`},
		{`{"links":[{"rel":"Next","href":"?marker=case-miss"},{"rel":true,"href":"?marker=type-miss"}],"next":"?marker=top"}`, `"?marker=top"`},
	} {
		t.Run(tc.body, func(t *testing.T) {
			fields := snapshotPagerContractFields(t, tc.body)
			next, err := snapshotNext(fields, header)
			if err != nil || string(next) != tc.want {
				t.Fatal(string(next), err, tc.want)
			}
		})
	}
}

func TestSnapshotPagerConsumesMalformedPrefixButStopsAtFirstPresentHrefEvenFalsey(t *testing.T) {
	for _, body := range []string{
		`{"links":null,"next":"?marker=ignored"}`,
		`{"links":false,"next":"?marker=ignored"}`,
		`{"links":0,"next":"?marker=ignored"}`,
		`{"links":"x","next":"?marker=ignored"}`,
		`{"links":[null,{"rel":"next","href":"?marker=ignored"}]}`,
		`{"links":[false,{"rel":"next","href":"?marker=ignored"}]}`,
		`{"links":[[],{"rel":"next","href":"?marker=ignored"}]}`,
	} {
		next, err := snapshotNext(snapshotPagerContractFields(t, body), nil)
		if err == nil || next != nil {
			t.Fatal("consumed malformed prefix hidden", string(next), err)
		}
	}
	for _, href := range []string{"null", "false", "0", `""`, "[]", "{}"} {
		fields := snapshotPagerContractFields(t, `{"links":[{"rel":"next","href":`+href+`},null,{"rel":"next","href":"?marker=unused"}],"next":"?marker=top"}`)
		next, err := snapshotNext(fields, http.Header{"Link": {"unused malformed header"}})
		if err != nil || string(next) != `"?marker=top"` {
			t.Fatal(string(next), err)
		}
	}
	fields := snapshotPagerContractFields(t, `{"links":[{"rel":"next"},{"rel":"next","href":"?marker=used"},null],"next":true}`)
	next, err := snapshotNext(fields, http.Header{"Link": {"unused malformed header"}})
	if err != nil || string(next) != `"?marker=used"` {
		t.Fatal(string(next), err)
	}
	// A truthy nonstring is selected first; URI conversion is a later stage.
	next, err = snapshotNext(snapshotPagerContractFields(t, `{"links":[{"rel":"next","href":true},null],"next":"?marker=unused"}`), nil)
	if err != nil || string(next) != "true" {
		t.Fatal(string(next), err)
	}
	if target, query, exists, err := snapshotContinuation(snapshotPagerContractCollection, next, snapshotPagerContractCollection, nil, nil, nil); err == nil || target != "" || query != nil || exists {
		t.Fatal("selected nonstring link was not rejected before GET", target, query, exists, err)
	}
}

func TestSnapshotPagerHTTPLinkRepairSupportsQuotedCommasLastNextAndLazyValidation(t *testing.T) {
	header := http.Header{"Link": {
		`<?marker=first>; rel="next"; title="quoted,comma", <?marker=previous>; rel="prev"`,
		`<?marker=last>; rel="previous NEXT"`,
	}}
	next, err := snapshotNext(snapshotPagerContractFields(t, `{"links":[],"next":false}`), header)
	if err != nil || string(next) != `"?marker=last"` {
		t.Fatal(string(next), err)
	}
	for _, bad := range []string{"malformed", string([]byte{'<', '?', 0xff, '>', ';', ' ', 'r', 'e', 'l', '=', '"', 'n', 'e', 'x', 't', '"'}), "<?marker=bad\n>; rel=next"} {
		headers := http.Header{"Link": {bad}}
		next, err := snapshotNext(snapshotPagerContractFields(t, `{"next":"?marker=body"}`), headers)
		if err != nil || string(next) != `"?marker=body"` {
			t.Fatal("unused Link consumed", string(next), err)
		}
		if next, err := snapshotNext(nil, headers); err == nil || next != nil {
			t.Fatal("used invalid Link accepted", string(next), err)
		}
	}
	next, err = snapshotNext(nil, http.Header{"Link": {"", " \t "}})
	if err != nil || next != nil {
		t.Fatal(string(next), err)
	}
}

func TestSnapshotPagerContinuationKeepsRawQueryAndDropsOnlyOldPaginationAndAdvertisedBlanks(t *testing.T) {
	query := map[string]json.RawMessage{
		"marker": json.RawMessage(`"old"`), "limit": json.RawMessage("7"),
		"name": json.RawMessage(`"nightly"`), "status": json.RawMessage(`"available"`),
		"all_tenants": json.RawMessage("true"), "precise": json.RawMessage("9007199254740993"),
		"container": json.RawMessage(`[false,[1,2],null]`), "preserved_blank": json.RawMessage(`""`),
	}
	target, merged, exists, err := snapshotContinuation(
		snapshotPagerContractCollection+"?marker=old&limit=7&name=nightly&status=available&all_tenants=True&precise=9007199254740993&container=False&container=1&container=2&preserved_blank=",
		json.RawMessage(`"?marker=new&marker=second&limit=&name=&status=ready&tag=&tag=a&tag=b&unused="`),
		snapshotPagerContractCollection, query, json.RawMessage("7"), json.RawMessage(`"last"`))
	if err != nil || !exists {
		t.Fatal(target, merged, exists, err)
	}
	for key, want := range map[string]string{
		"marker": `["new","second"]`, "name": `"nightly"`, "status": `["ready"]`, "tag": `["a","b"]`,
		"all_tenants": "true", "precise": "9007199254740993", "container": `[false,[1,2],null]`, "preserved_blank": `""`,
	} {
		if string(merged[key]) != want {
			t.Fatal(key, string(merged[key]), want)
		}
	}
	if _, ok := merged["limit"]; ok {
		t.Fatal("old limit survived advertised blank", merged)
	}
	if _, ok := merged["unused"]; ok {
		t.Fatal("advertised wholly blank key survived", merged)
	}
	physical := snapshotPagerContractQuery(t, target)
	if !reflect.DeepEqual(physical["marker"], []string{"new", "second"}) || !reflect.DeepEqual(physical["container"], []string{"False", "1", "2"}) || physical.Get("all_tenants") != "True" || physical.Get("precise") != "9007199254740993" {
		t.Fatal("query reparse/coercion changed source values", physical)
	}
	if !physical.Has("preserved_blank") || physical.Get("name") != "nightly" || physical.Get("status") != "ready" {
		t.Fatal(physical)
	}
}

func TestSnapshotPagerTruthyInitialLimitSynthesizesShortPageAndUsesRawLastID(t *testing.T) {
	query := map[string]json.RawMessage{"limit": json.RawMessage("3"), "status": json.RawMessage(`"available"`)}
	target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection+"?limit=3&status=available", nil, snapshotPagerContractCollection, query, json.RawMessage("3"), json.RawMessage(`{"foreign":1,"keys":2}`))
	if err != nil || !exists || string(merged["marker"]) != `{"foreign":1,"keys":2}` || string(merged["limit"]) != "3" {
		t.Fatal(target, merged, exists, err)
	}
	if !reflect.DeepEqual(snapshotPagerContractQuery(t, target)["marker"], []string{"foreign", "keys"}) {
		t.Fatal(target)
	}
	// The reader supplies the last consumed raw row, including filtered-out
	// rows. No page size/accepted-row counter is part of this pager helper.
	for _, limit := range []string{"null", "false", "0", `""`, "[]", "{}"} {
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, nil, snapshotPagerContractCollection, query, json.RawMessage(limit), nil)
		if err != nil || exists || target != "" || string(merged["status"]) != `"available"` {
			t.Fatal(target, merged, exists, err)
		}
		if _, ok := merged["limit"]; ok {
			t.Fatal("old limit not discarded", merged)
		}
	}
	for _, tc := range []struct{ marker, lastID string }{
		{"null", "null"}, {"false", "0"}, {`"same"`, `"same"`}, {`[true,{"n":1}]`, `[1,{"n":true}]`},
	} {
		query := map[string]json.RawMessage{"marker": json.RawMessage(tc.marker), "limit": json.RawMessage("1")}
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, nil, snapshotPagerContractCollection, query, json.RawMessage("1"), json.RawMessage(tc.lastID))
		if !errors.Is(err, resource.ErrPaginationCycle) || target != "" || merged != nil || exists {
			t.Fatal(target, merged, exists, err)
		}
	}
	if target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, nil, snapshotPagerContractCollection, nil, json.RawMessage("1"), nil); !errors.Is(err, resource.ErrPaginationCycle) || target != "" || merged != nil || exists {
		t.Fatal("source absent marker/ID equality was optimized away", target, merged, exists, err)
	}
}

func TestSnapshotPagerAdvertisedMarkerEqualityKeepsParseQSArrayShape(t *testing.T) {
	for _, tc := range []struct {
		previous  string
		wantCycle bool
	}{
		{`["same"]`, true}, {`"same"`, false}, {"null", false},
	} {
		query := map[string]json.RawMessage{"marker": json.RawMessage(tc.previous)}
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, json.RawMessage(`"?marker=same"`), snapshotPagerContractCollection, query, nil, nil)
		if tc.wantCycle {
			if !errors.Is(err, resource.ErrPaginationCycle) || target != "" || merged != nil || exists {
				t.Fatal(target, merged, exists, err)
			}
		} else if err != nil || !exists || string(merged["marker"]) != `["same"]` {
			t.Fatal(target, merged, exists, err)
		}
	}
	// An advertised link with no marker does not trigger source marker equality.
	target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection+"?marker=old", json.RawMessage(`"?offset=2"`), snapshotPagerContractCollection, map[string]json.RawMessage{"marker": json.RawMessage(`"old"`)}, json.RawMessage("5"), nil)
	if err != nil || !exists || string(merged["offset"]) != `["2"]` {
		t.Fatal(target, merged, exists, err)
	}
	if _, ok := merged["marker"]; ok {
		t.Fatal("old marker retained", merged)
	}
}

func TestSnapshotPagerPageKeyCanonicalizesQueryButRejectsOriginEscapedPathAndTraversal(t *testing.T) {
	left, err := snapshotPageKey(snapshotPagerContractCollection+"?b=two&a=one&a=three", snapshotPagerContractCollection)
	if err != nil {
		t.Fatal(err)
	}
	right, err := snapshotPageKey("HTTPS://CINDER.EXAMPLE/v3/project/snapshots/detail?a=one&a=three&b=two", snapshotPagerContractCollection)
	if err != nil || left != right {
		t.Fatal(left, right, err)
	}
	other, err := snapshotPageKey(snapshotPagerContractCollection+"?a=three&a=one&b=two", snapshotPagerContractCollection)
	if err != nil || left == other {
		t.Fatal("repeated value order lost", left, other, err)
	}
	for _, href := range []string{
		"https://evil.example/v3/project/snapshots/detail?marker=new", "http://cinder.example/v3/project/snapshots/detail", "https://user@cinder.example/v3/project/snapshots/detail",
		"/v3/project/other", "/v3/project/snapshots/%64etail", "../snapshots/detail", "/v3/project/snapshots/%2e%2e/snapshots/detail", "?marker=new#", "?marker=new#fragment",
		"/v3/project/snapshots/de%00tail", "/v3/project/snapshots/de%FFtail", "/v3/project/snapshots/de%5ctail", "mailto:bad", "?marker=%ZZ", "?marker=%FF", "?marker=bad\n",
	} {
		raw, _ := json.Marshal(href)
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, raw, snapshotPagerContractCollection, nil, nil, nil)
		if err == nil || target != "" || merged != nil || exists {
			t.Fatal("unsafe next accepted", href, target, merged, exists, err)
		}
	}
	for _, href := range []string{"?marker=once", "/v3/project/snapshots/detail?marker=once", "detail?marker=once"} {
		raw, _ := json.Marshal(href)
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection+"?name=retained", raw, snapshotPagerContractCollection, map[string]json.RawMessage{"name": json.RawMessage(`"retained"`)}, nil, nil)
		if err != nil || !exists || string(merged["marker"]) != `["once"]` || len(snapshotPagerContractQuery(t, target)["marker"]) != 1 {
			t.Fatal(target, merged, exists, err)
		}
	}
}

func TestSnapshotPagerOwnsInputsAndReturnsAtomicFailuresWithoutAnInventedResource(t *testing.T) {
	fields := snapshotPagerContractFields(t, `{"links":[{"rel":"next","href":"?marker=new"}]}`)
	before := bytes.Clone(fields["links"])
	next, err := snapshotNext(fields, nil)
	if err != nil || !bytes.Equal(fields["links"], before) {
		t.Fatal(string(next), err)
	}
	fields["links"][0] = '!'
	if string(next) != `"?marker=new"` {
		t.Fatal("next borrowed body storage", string(next))
	}
	query := map[string]json.RawMessage{"name": json.RawMessage(`"stable"`), "limit": json.RawMessage("2")}
	limit, lastID := json.RawMessage("2"), json.RawMessage(`"last"`)
	target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, nil, snapshotPagerContractCollection, query, limit, lastID)
	if err != nil || !exists {
		t.Fatal(target, merged, exists, err)
	}
	query["name"][1], limit[0], lastID[1] = 'X', '9', 'X'
	if string(merged["name"]) != `"stable"` || string(merged["limit"]) != "2" || string(merged["marker"]) != `"last"` {
		t.Fatal("continuation borrowed caller values", merged)
	}
	for _, bad := range []json.RawMessage{json.RawMessage{}, json.RawMessage(`invalid`), json.RawMessage(`{} {}`), json.RawMessage{0xff}, json.RawMessage("true"), json.RawMessage(`["next"]`)} {
		target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, bad, snapshotPagerContractCollection, nil, nil, nil)
		if err == nil || target != "" || merged != nil || exists {
			t.Fatal(target, merged, exists, err)
		}
	}
	badQuery := map[string]json.RawMessage{"name": json.RawMessage(`invalid`)}
	if target, merged, exists, err := snapshotContinuation(snapshotPagerContractCollection, json.RawMessage(`"?marker=next"`), snapshotPagerContractCollection, badQuery, nil, nil); err == nil || target != "" || merged != nil || exists || string(badQuery["name"]) != "invalid" {
		t.Fatal("query encoder failure not atomic", target, merged, exists, err)
	}
}
