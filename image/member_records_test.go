package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const memberRecordParent = "image-空白"
const memberRecordPrefix = "https://example.test/reverse/glance/v2/images/"

// Byte delivery and close counting reuse the existing image fixture. Only the
// missing Close callback is supplied here for source-boundary assertions.
type memberRecordCloseBody struct {
	*deleteCoreBody
	after func()
}

func (body *memberRecordCloseBody) Close() error {
	err := body.deleteCoreBody.Close()
	if body.after != nil {
		body.after()
	}
	return err
}
func memberRecordJSON(status int, raw string) *http.Response {
	return deleteCoreHTTP(status, &deleteCoreBody{reader: strings.NewReader(raw)}, http.Header{"X-Member-Proof": {"actual"}})
}
func memberRecordProof(t *testing.T, err error, code int, raw string) *resource.ResponseError {
	t.Helper()
	var receipt *resource.ResponseError
	if !errors.As(err, &receipt) || receipt.StatusCode != code || string(receipt.Body) != raw || receipt.Header.Get("X-Member-Proof") != "actual" {
		t.Fatalf("actual receipt code=%d body=%q err=%v receipt=%+v", code, raw, err, receipt)
	}
	return receipt
}
func memberRecordValues(record *ImageMemberRecord) map[string]string {
	values := make(map[string]string)
	if record != nil && record.Resource != nil {
		for key, raw := range record.Resource.Body {
			values[key] = string(raw)
		}
	}
	return values
}

func TestImageMemberRecordsProjectionAndOwnedReceipts(t *testing.T) {
	const row = `{"member":false,"created_at":900719925474099312345,"updated_at":{"literal":true},"status":["future",null],"schema":false,"image_id":"wire parent","location":{"cloud":"wire"},"self":"https://foreign.test/","vendor":1e400}`
	const raw = `{"members":[` + row + `,{}],"next":false}`
	calls := 0
	header := http.Header{"X-Member-Proof": {"actual"}}
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "before" || req.Header.Get("Accept") != "application/json" {
			t.Fatal("default list route", req.Method, req.URL, req.Header)
		}
		return deleteCoreHTTP(203, &deleteCoreBody{reader: strings.NewReader(raw)}, header), nil
	})
	rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent))
	if err != nil || len(rows) != 2 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	parent, _ := json.Marshal(memberRecordParent)
	want := map[string]string{"id": "false", "name": "null", "member_id": "false", "created_at": "900719925474099312345", "updated_at": `{"literal":true}`, "status": `["future",null]`, "schema": "false", "image_id": string(parent), "location": "null"}
	th.CheckDeepEquals(t, want, memberRecordValues(rows[0]))
	for _, record := range rows {
		if record.ImageID == nil || *record.ImageID != memberRecordParent || record.StatusCode != 203 || record.Resource.StatusCode != 203 || record.Wire.StatusCode != 203 || record.Header.Get("X-Member-Proof") != "actual" || record.Resource.Header.Get("X-Member-Proof") != "actual" || record.Wire.Header.Get("X-Member-Proof") != "actual" || string(record.Envelope) != raw {
			t.Fatal("page provenance", record)
		}
	}
	for _, key := range []string{"id", "name", "member_id", "created_at", "updated_at", "status", "schema", "location"} {
		th.AssertEquals(t, "null", string(rows[1].Resource.Body[key]))
	}
	th.AssertEquals(t, `"wire parent"`, string(rows[0].Wire.Body["image_id"]))
	th.AssertEquals(t, `{"cloud":"wire"}`, string(rows[0].Wire.Body["location"]))
	th.AssertEquals(t, "1e400", string(rows[0].Wire.Body["vendor"]))
	th.AssertEquals(t, `"https://foreign.test/"`, string(rows[0].Wire.Body["self"]))
	if _, exists := rows[0].Resource.Body["self"]; exists {
		t.Fatal("self became source field")
	}
	rows[0].Resource.Body["member_id"][0] = '0'
	rows[0].Resource.Header.Set("X-Member-Proof", "view changed")
	rows[0].Wire.Header.Set("X-Member-Proof", "wire changed")
	rows[0].Header.Set("X-Member-Proof", "record changed")
	rows[0].Envelope[0] = '!'
	*rows[0].ImageID = "changed parent"
	if string(rows[0].Wire.Body["member"]) != "false" || rows[1].Header.Get("X-Member-Proof") != "actual" || string(rows[1].Envelope) != raw || *rows[1].ImageID != memberRecordParent || header.Get("X-Member-Proof") != "actual" {
		t.Fatal("caller mutation crossed owned channels")
	}
}

func TestImageMemberRecordsIdentityAliasesAndLocation(t *testing.T) {
	for _, test := range []struct{ name, row, id, member string }{
		{"wire alias", `{"member":"alias"}`, `"alias"`, `"alias"`},
		{"canonical alias", `{"member_id":"canonical"}`, `"canonical"`, `"canonical"`},
		{"wire alias later", `{"member_id":"first","member":"last"}`, `"last"`, `"last"`},
		{"canonical alias later", `{"member":"first","member_id":"last"}`, `"last"`, `"last"`},
		{"duplicate first position final value", `{"member":"first","member_id":"canonical","member":"last"}`, `"canonical"`, `"canonical"`},
		{"present null id", `{"id":null,"member":"alias"}`, "null", `"alias"`},
		{"present empty id", `{"id":"","member":"alias"}`, `""`, `"alias"`},
		{"untyped identity", `{"member":[900719925474099312345]}`, "[900719925474099312345]", "[900719925474099312345]"},
		{"absent alternate identity", `{"name":"inherited name"}`, "null", "null"},
		{"only unknown location", `{"location":{"server":"foreign"}}`, "null", "null"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls, locations := 0, 0
			cloud := "captured"
			location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return memberRecordJSON(200, `{"members":[`+test.row+`]}`), nil
			})
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }})
			rows, err := service.AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent), func(*ImageMemberRecordListOpts) error {
				cloud = "after option"
				location.Project.ID[1] = 'X'
				return nil
			})
			if err != nil || len(rows) != 1 || calls != 1 || locations != 1 {
				t.Fatal(rows, err, calls, locations)
			}
			th.AssertEquals(t, test.id, string(rows[0].Resource.Body["id"]))
			th.AssertEquals(t, test.member, string(rows[0].Resource.Body["member_id"]))
			var captured resource.CloudLocation
			if err := json.Unmarshal(rows[0].Resource.Body["location"], &captured); err != nil || captured.Cloud == nil || *captured.Cloud != "captured" || string(captured.Project.ID) != `"token project"` {
				t.Fatal("location snapshot", captured, err)
			}
		})
	}
}

func TestImageMemberRecordsLazyRepeatedOptionSnapshots(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "first"
	headers := map[string]string{"X-Option": "factory snapshot"}
	filters := map[string]json.RawMessage{"member_id": json.RawMessage(`"selected"`)}
	paginated := false
	factory := WithImageMemberRecordListOpts(ImageMemberRecordListOpts{Headers: headers, Filters: filters, Paginated: &paginated, Limit: 2, Marker: "start"})
	var retained *ImageMemberRecordListOpts
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		wantSource := "first"
		if calls == 2 {
			wantSource = "second"
		}
		if req.Header.Get("X-Source") != wantSource || req.Header.Get("X-Option") != "factory snapshot" || req.URL.Query().Get("limit") != "2" || req.URL.Query().Get("marker") != "start" || len(req.URL.Query()) != 2 {
			t.Fatal("snapshot", req.URL, req.Header)
		}
		retained.Headers["X-Option"] = "retained mutation"
		retained.Filters["member_id"][1] = 'X'
		*retained.Paginated = true
		return memberRecordJSON(200, `{"members":[{"member":"selected"}],"next":"https://foreign.test/never"}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "changed by location"
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	stream := service.ListImageMemberRecords(context.Background(), resource.ID(memberRecordParent), factory, func(value *ImageMemberRecordListOpts) error {
		callbacks++
		retained = value
		cloud = "changed by callback"
		client.MoreHeaders["X-Source"] = "changed by callback"
		return nil
	})
	headers["X-Option"] = "caller mutation"
	filters["member_id"][1] = 'X'
	paginated = true
	if calls != 0 || callbacks != 0 || locations != 0 {
		t.Fatal("eager iterator", calls, callbacks, locations)
	}
	for iteration := 0; iteration < 2; iteration++ {
		wantCloud := "first"
		if iteration == 1 {
			client.MoreHeaders["X-Source"] = "second"
			cloud = "second"
			wantCloud = "second"
		}
		count := 0
		for row, err := range stream {
			if err != nil {
				t.Fatal(err)
			}
			count++
			var got resource.CloudLocation
			if err := json.Unmarshal(row.Resource.Body["location"], &got); err != nil || got.Cloud == nil || *got.Cloud != wantCloud {
				t.Fatal("per-iteration location", got, err)
			}
		}
		if count != 1 || calls != iteration+1 || callbacks != iteration+1 || locations != iteration+1 {
			t.Fatal(count, calls, callbacks, locations)
		}
	}
}

func TestImageMemberRecordsPagingAndRawConsumption(t *testing.T) {
	for _, mode := range []string{"next", "plural links", "links", "HTTP Link", "marker fallback", "initial server limit only", "single page", "raw cap filter", "early break"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			next := "/v2/images/" + url.PathEscape(memberRecordParent) + "/members?marker=second"
			first := `{"members":[{"member":"first"},{"member":"last"}]`
			extra := `,"next":"` + next + `"`
			header := http.Header{"X-Member-Proof": {"actual"}}
			options := []ImageMemberRecordListOption{}
			switch mode {
			case "plural links":
				extra = `,"members_links":[{"rel":"next","href":"` + next + `"}]`
			case "links":
				extra = `,"links":[{"rel":"next","href":"` + next + `"}]`
			case "HTTP Link":
				extra = ""
				header.Set("Link", "<"+next+">; rel=\"next\"")
			case "marker fallback":
				extra = ""
				options = append(options, WithImageMemberRecordListLimit(2))
			case "initial server limit only":
				extra = `,"limit":2`
			case "single page":
				options = append(options, WithImageMemberRecordListPaginated(false))
			case "raw cap filter":
				options = append(options, WithImageMemberRecordListMaxItems(2), WithImageMemberRecordListFilter("member_id", "last"))
			}
			first += extra + `}`
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if mode == "marker fallback" || mode == "raw cap filter" {
						th.AssertEquals(t, "2", req.URL.Query().Get("limit"))
					} else {
						th.AssertEquals(t, "", req.URL.RawQuery)
					}
					return deleteCoreHTTP(200, &deleteCoreBody{reader: strings.NewReader(first)}, header), nil
				}
				if calls != 2 {
					t.Fatal("unexpected third page", req.URL)
				}
				if mode == "marker fallback" {
					if req.URL.Query().Get("marker") != "last" || req.URL.Query().Get("limit") != "2" {
						t.Fatal("wire last member marker", req.URL)
					}
				} else if req.URL.Query().Get("marker") != "second" {
					t.Fatal("continuation route", req.URL)
				}
				return memberRecordJSON(200, `{"members":[]}`), nil
			})
			count := 0
			for _, err := range New(client).ListImageMemberRecords(context.Background(), resource.ID(memberRecordParent), options...) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				if mode == "early break" {
					break
				}
			}
			wantCalls, wantCount := 2, 2
			switch mode {
			case "initial server limit only", "single page":
				wantCalls = 1
			case "raw cap filter", "early break":
				wantCalls = 1
				wantCount = 1
			}
			if calls != wantCalls || count != wantCount {
				t.Fatal("paging/consumption", calls, count, wantCalls, wantCount)
			}
		})
	}
	t.Run("partial collector retains first page", func(t *testing.T) {
		calls := 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return memberRecordJSON(200, `{"members":[{"member":"kept"}],"next":"?marker=second"}`), nil
			}
			return memberRecordJSON(200, `{"members":[null]}`), nil
		})
		rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent))
		if len(rows) != 1 || err == nil || calls != 2 || string(rows[0].Resource.Body["member_id"]) != `"kept"` {
			t.Fatal(rows, err, calls)
		}
		memberRecordProof(t, err, 200, `{"members":[null]}`)
	})
	t.Run("unused malformed row and next stay passive", func(t *testing.T) {
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			return memberRecordJSON(200, `{"members":[{"member":"kept"},null],"next":"https://foreign.test/unsafe"}`), nil
		})
		count := 0
		for _, err := range New(client).ListImageMemberRecords(context.Background(), resource.ID(memberRecordParent)) {
			if err != nil {
				t.Fatal(err)
			}
			count++
			break
		}
		th.AssertEquals(t, 1, count)
	})
}

func TestImageMemberRecordsEnvelopeFailuresAndActualStatus(t *testing.T) {
	for _, test := range []struct {
		code  int
		raw   string
		good  bool
		count int
	}{
		{200, `{"members":[]}`, true, 0}, {201, `{"members":{}}`, true, 1}, {299, `{"members":[{"member":"id"}]}`, true, 1}, {300, `{"members":[{}]}`, true, 1}, {399, `{"members":[]}`, true, 0},
		{204, "", false, 0}, {200, "not JSON", false, 0}, {200, `{}`, false, 0}, {200, `null`, false, 0}, {200, `[]`, false, 0}, {200, `{"members":null}`, false, 0}, {200, `{"members":false}`, false, 0}, {200, `{"members":[null]}`, false, 0}, {200, `{"members":[1]}`, false, 0}, {200, `{"members":[{"connection":null}]}`, false, 0}, {200, `{"members":[{"microversion":"2"}]}`, false, 0}, {200, `{"members":[{"_synchronized":true}]}`, false, 0}, {200, "{\"members\":[{\"unknown\":\"\xff\"}]}", false, 0},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &deleteCoreBody{reader: strings.NewReader(test.raw)}
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return deleteCoreHTTP(test.code, body, http.Header{"X-Member-Proof": {"actual"}}), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent))
			if test.good {
				if err != nil || len(rows) != test.count {
					t.Fatal(rows, err)
				}
			} else {
				if err == nil || len(rows) != 0 {
					t.Fatal(rows, err)
				}
				memberRecordProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted replay", calls, retries, body.closes)
			}
		})
	}
}

func TestImageMemberRecordsReadCloseAndStickyGuardEvidence(t *testing.T) {
	const raw = `{"members":[{"member":"actual"}]}`
	for _, mode := range []string{"read", "close", "cancel", "source drift", "read drift restored on Close", "outer drift restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("owned body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &deleteCoreBody{reader: strings.NewReader(raw)}
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return cause
				}
				return nil
			})
			readAction := func() {}
			switch mode {
			case "read":
				body.reader = deleteCoreReader(func(buf []byte) (int, error) { return copy(buf, raw), cause })
			case "close":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			case "cancel":
				readAction = func() { cancel(cause) }
			case "source drift", "read drift restored on Close":
				readAction = func() { client.Endpoint = "https://foreign.test/" }
			case "outer drift restored on Close":
				readAction = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = deleteCoreReader(func(buf []byte) (int, error) { readAction(); return copy(buf, raw), io.EOF })
			}
			selectedBody := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selectedBody = &memberRecordCloseBody{deleteCoreBody: body, after: func() { client.Endpoint = "https://example.test/catalog/"; outerInvalid = false }}
			}
			client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return deleteCoreHTTP(201, selectedBody, http.Header{"X-Member-Proof": {"actual"}}), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			rows, err := New(client).AllImageMemberRecords(ctx, resource.ID(memberRecordParent))
			if len(rows) != 0 || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(rows, err, calls, retries, body.closes)
			}
			if strings.Contains(mode, "source") || mode == "read drift restored on Close" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal("body/outer cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("nested404 was interpreted as absence", err)
			}
			proof := memberRecordProof(t, err, 201, raw)
			proof.Body[0] = '!'
			proof.Header.Set("X-Member-Proof", "caller changed")
		})
	}
}

func TestImageMemberRecordsPreflightAndExtensionFilters(t *testing.T) {
	for _, test := range []struct {
		name   string
		parent resource.Ref
		opts   []ImageMemberRecordListOption
	}{
		{"empty image", resource.ID(""), nil}, {"unsafe image", resource.ID("bad/id"), nil},
		{"nil option", resource.ID(memberRecordParent), []ImageMemberRecordListOption{nil}},
		{"negative limit", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListLimit(-1)}},
		{"negative cap", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListMaxItems(-1)}},
		{"bad marker", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListMarker("\n")}},
		{"owned token", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListHeader("X-Auth-Token", "foreign")}},
		{"header newline", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListHeader("X-Extra", "\n")}},
		{"broken declared filter", resource.ID(memberRecordParent), []ImageMemberRecordListOption{WithImageMemberRecordListOpts(ImageMemberRecordListOpts{Filters: map[string]json.RawMessage{"status": json.RawMessage(`{`)}})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return memberRecordJSON(200, `{"members":[]}`), nil
			})
			rows, err := New(client).AllImageMemberRecords(context.Background(), test.parent, test.opts...)
			if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	for _, test := range []struct {
		name  string
		row   string
		field string
		value any
		want  int
	}{
		{"canonical alias filter", `[{"member":"selected"},{"member":"other"}]`, "member_id", "selected", 1},
		{"identity null stays authoritative", `[{"id":null,"member":"selected"},{"member":"selected"}]`, "id", "selected", 1},
		{"inherited name", `[{"name":"same"},{"member":"same"}]`, "name", "same", 1},
		{"arbitrary nested field", `[{"status":{"kept":1,"extra":2}},{"status":null}]`, "status", map[string]any{"kept": 1}, 1},
		{"exact huge number", `[{"created_at":900719925474099312345},{"created_at":900719925474099312346}]`, "created_at", json.RawMessage(`900719925474099312345`), 1},
		{"nullable default", `[{}, {"schema":null}, {"schema":false}]`, "schema", nil, 2},
		{"unknown stays passive", `[{}]`, "vendor", json.RawMessage(`{`), 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.RawQuery != "" {
					t.Fatal("local filter became API query", req.URL)
				}
				return memberRecordJSON(200, `{"members":`+test.row+`}`), nil
			})
			rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent), WithImageMemberRecordListFilter(test.field, test.value))
			if err != nil || len(rows) != test.want {
				t.Fatal(rows, err)
			}
		})
	}
	t.Run("option source drift stops later callback", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return memberRecordJSON(200, `{"members":[]}`), nil
		})
		rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent), func(*ImageMemberRecordListOpts) error { client.Endpoint = "https://foreign.test/"; return nil }, func(*ImageMemberRecordListOpts) error {
			callbacks++
			client.Endpoint = "https://example.test/catalog/"
			return nil
		})
		if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatal(rows, err, calls, callbacks)
		}
	})
	t.Run("canceled source skips location and options", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		callbacks, locations := 0, 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) { t.Fatal("HTTP before preflight"); return nil, nil })
		service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
		rows, err := service.AllImageMemberRecords(ctx, resource.ID(memberRecordParent), func(*ImageMemberRecordListOpts) error { callbacks++; return nil })
		if len(rows) != 0 || !errors.Is(err, context.Canceled) || locations != 0 || callbacks != 0 {
			t.Fatal(rows, err, locations, callbacks)
		}
	})
	t.Run("empty collector is allocated", func(t *testing.T) {
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) { return memberRecordJSON(200, `{"members":[]}`), nil })
		rows, err := New(client).AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent))
		if err != nil || rows == nil || len(rows) != 0 {
			t.Fatal(rows, err)
		}
	})
}

func TestImageMemberRecordsPreserveLegacyFiniteMemberList(t *testing.T) {
	calls := 0
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return memberRecordJSON(200, `{"members":[{"member_id":"legacy","image_id":"wire","status":"future"}],"next":"https://foreign.test/never"}`), nil
	})
	rows, err := New(client).AllImageMembers(context.Background(), resource.ID(memberRecordParent))
	if err != nil || len(rows) != 1 || calls != 1 || rows[0].MemberID == nil || *rows[0].MemberID != "legacy" {
		t.Fatal(rows, err, calls)
	}
	if !reflect.DeepEqual(rows[0].Body["member_id"], json.RawMessage(`"legacy"`)) {
		t.Fatal("legacy raw fields changed", rows[0])
	}
}
