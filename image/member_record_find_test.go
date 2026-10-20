package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestFindImageMemberRecordSeededGetAndTolerance(t *testing.T) {
	const selected = "project 空 白"
	for _, test := range []struct {
		name         string
		code         int
		raw          string
		parsed, good bool
		id, member   string
	}{
		{"seed survives absent id", 200, `{"member":"response member","created_at":false}`, true, true, `"project 空 白"`, `"response member"`},
		{"response id replaces seed", 201, `{"id":"wire id","member":12}`, true, true, `"wire id"`, "12"},
		{"explicit null id", 299, `{"id":null,"member":"alias"}`, true, true, "null", `"alias"`},
		{"accepted redirect passive links", 300, `{"name":"wire","self":"https://foreign.test/","schema":{"passive":true}}`, true, true, `"project 空 白"`, "null"},
		{"accepted end", 399, `{}`, true, true, `"project 空 白"`, "null"},
		{"empty accepted", 204, "", false, true, `"project 空 白"`, "null"},
		{"invalid syntax tolerated", 200, `{"member":!}`, false, true, `"project 空 白"`, "null"},
		{"opaque tolerated", 304, "not JSON", false, true, `"project 空 白"`, "null"},
		{"parsed null", 200, `null`, true, false, "", ""},
		{"parsed list", 200, `[]`, true, false, "", ""},
		{"parsed string", 200, `"text"`, true, false, "", ""},
		{"parsed number", 200, `1`, true, false, "", ""},
		{"parsed boolean", 200, `false`, true, false, "", ""},
		{"invalid UTF8", 200, "{\"member\":\"\xff\"}", true, false, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls, retries := 0, 0
			body := &deleteCoreBody{reader: strings.NewReader(test.raw)}
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				want := "/reverse/glance/v2/images/" + url.PathEscape(memberRecordParent) + "/members/" + url.PathEscape(selected)
				if req.Method != http.MethodGet || req.URL.EscapedPath() != want || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("fixed direct Get", req.Method, req.URL, req.Body)
				}
				return deleteCoreHTTP(test.code, body, http.Header{"X-Member-Proof": {"actual"}, "Content-Type": {"text/plain"}}), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), selected)
			if test.good {
				if err != nil || got == nil || got.Resource == nil || (got.Wire != nil) != test.parsed || got.ImageID == nil || *got.ImageID != memberRecordParent || got.StatusCode != test.code || got.Resource.StatusCode != test.code || got.Header.Get("X-Member-Proof") != "actual" || string(got.Envelope) != test.raw || len(got.Resource.Body) != 9 {
					t.Fatal(got, err)
				}
				th.AssertEquals(t, test.id, string(got.Resource.Body["id"]))
				th.AssertEquals(t, test.member, string(got.Resource.Body["member_id"]))
				if test.parsed {
					if got.Wire.StatusCode != test.code || got.Wire.Header.Get("X-Member-Proof") != "actual" {
						t.Fatal("wire receipt", got.Wire)
					}
					var actual map[string]json.RawMessage
					if err := json.Unmarshal([]byte(test.raw), &actual); err != nil {
						t.Fatal(err)
					}
					if _, exists := actual["id"]; !exists {
						if _, injected := got.Wire.Body["id"]; injected {
							t.Fatal("seed injected into Wire", got.Wire)
						}
					}
				}
			} else {
				if got != nil || err == nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
				memberRecordProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted direct Get replay/fallback", calls, retries, body.closes)
			}
		})
	}
}

func TestFindImageMemberRecordCompatibleFallbackAndUniqueness(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, mode := range []string{"alternate member", "explicit id", "inherited name", "null id name", "same row id and name", "later page unique", "duplicate second row", "duplicate later page", "zero ignored", "zero strict", "candidate before later failure", "slash name"} {
			t.Run(fmt.Sprintf("%d/%s", code, mode), func(t *testing.T) {
				selected := "selected"
				if mode == "slash name" {
					selected = "name/空 白:member"
				}
				calls := 0
				var getBody *deleteCoreBody
				options := []FindImageMemberRecordOption{}
				if mode == "zero strict" {
					options = append(options, WithFindImageMemberRecordIgnoreMissing(false))
				}
				first := `{"member":"other"}`
				second := ""
				switch mode {
				case "alternate member":
					first = `{"member":"selected"}`
				case "explicit id":
					first = `{"id":"selected","member":"other"}`
				case "inherited name":
					first = `{"id":"other","name":"selected"}`
				case "null id name":
					first = `{"id":null,"member":"other","name":"selected"}`
				case "same row id and name":
					first = `{"id":"selected","name":"selected"}`
				case "later page unique":
					second = `{"member":"selected"}`
				case "duplicate second row":
					first = `{"member":"selected"},{"id":"second","name":"selected"}`
				case "duplicate later page":
					first = `{"member":"selected"}`
					second = `{"id":"second","name":"selected"}`
				case "candidate before later failure":
					first = `{"member":"selected"}`
					second = "broken"
				case "slash name":
					encoded, _ := json.Marshal(selected)
					first = `{"name":` + string(encoded) + `}`
				}
				client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodGet || req.Body != nil {
						t.Fatal("fallback method", req.Method, req.Body)
					}
					if calls == 1 {
						if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/"+url.PathEscape(selected) || req.URL.RawQuery != "" {
							t.Fatal("direct identity route", req.URL)
						}
						getBody = &deleteCoreBody{reader: strings.NewReader("native missing or forbidden")}
						return deleteCoreHTTP(code, getBody, http.Header{"X-Member-Proof": {"actual"}}), nil
					}
					if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members" || req.URL.Query().Get("name") != "" {
						t.Fatal("Source member list has no name query", req.URL)
					}
					if calls == 2 {
						suffix := ""
						if second != "" || mode == "duplicate second row" {
							suffix = `,"next":"?marker=second"`
						}
						return memberRecordJSON(200, `{"members":[`+first+`]`+suffix+`}`), nil
					}
					if calls != 3 || req.URL.Query().Get("marker") != "second" {
						t.Fatal("wrong follow-up", calls, req.URL)
					}
					if mode == "duplicate second row" {
						t.Fatal("ambiguity should stop at second matching row")
					}
					if second == "broken" {
						return memberRecordJSON(200, `{"members":[null]}`), nil
					}
					return memberRecordJSON(203, `{"members":[`+second+`]}`), nil
				})
				got, err := New(client).FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), selected, options...)
				wantCalls := 2
				if second != "" {
					wantCalls = 3
				}
				if calls != wantCalls || getBody == nil || getBody.closes != 1 {
					t.Fatal("fallback calls", calls, wantCalls, getBody)
				}
				switch mode {
				case "zero ignored":
					if got != nil || err != nil {
						t.Fatal(got, err)
					}
				case "zero strict":
					var missing *resource.NotFoundError
					if got != nil || !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &missing) || missing.Cause != nil {
						t.Fatal(got, err, missing)
					}
				case "duplicate second row", "duplicate later page":
					if got != nil || !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatal(got, err)
					}
				case "candidate before later failure":
					if got != nil || err == nil {
						t.Fatal(got, err)
					}
					memberRecordProof(t, err, 200, `{"members":[null]}`)
				default:
					if got == nil || err != nil || got.ImageID == nil || *got.ImageID != memberRecordParent {
						t.Fatal(got, err)
					}
					if mode == "later page unique" && got.StatusCode != 203 {
						t.Fatal("candidate receipt", got)
					}
				}
			})
		}
	}
	t.Run("present null id suppresses member identity", func(t *testing.T) {
		calls := 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return memberRecordJSON(404, `{}`), nil
			}
			return memberRecordJSON(200, `{"members":[{"id":null,"member":"selected"},{"id":false,"member":"selected"},{"id":1,"member":"selected"},{"member":["selected"]}]}`), nil
		})
		got, err := New(client).FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "selected")
		if got != nil || err != nil || calls != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestFindImageMemberRecordNativeHooksAndTerminalFailures(t *testing.T) {
	for _, mode := range []string{"native 500", "nested transport404", "accepted read404", "accepted Close404", "rejected read404", "rejected Close404", "retry hook cause", "retry succeeds before list", "retry clean final fallback"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			cause := errors.New("native handling failure")
			nested := gophercloud.ErrUnexpectedResponseCode{Method: "GET", URL: "https://decoy.test/", Actual: 404, Expected: []int{200}, Body: []byte("decoy")}
			var bodies []*deleteCoreBody
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "nested transport404" {
					return nil, errors.Join(cause, nested)
				}
				if mode == "retry succeeds before list" && calls == 2 {
					return memberRecordJSON(200, `{"member":"recovered"}`), nil
				}
				if mode == "retry clean final fallback" && calls == 3 {
					if strings.HasSuffix(req.URL.Path, "/selected") {
						t.Fatal("third request was not LIST", req.URL)
					}
					return memberRecordJSON(200, `{"members":[{"member":"selected"}]}`), nil
				}
				code := 404
				switch mode {
				case "native 500":
					code = 500
				case "accepted read404", "accepted Close404":
					code = 201
				}
				body := &deleteCoreBody{reader: strings.NewReader(`{"member":"actual"}`)}
				switch mode {
				case "accepted read404", "rejected read404":
					body.reader = deleteCoreReader(func(buf []byte) (int, error) { return copy(buf, `{"member":"actual"}`), errors.Join(cause, nested) })
				case "accepted Close404", "rejected Close404":
					body.closeErr = errors.Join(cause, nested)
				}
				bodies = append(bodies, body)
				return deleteCoreHTTP(code, body, http.Header{"X-Member-Proof": {"actual"}}), nil
			})
			if strings.HasPrefix(mode, "retry") {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, retry uint) error {
					retries++
					switch mode {
					case "retry hook cause":
						return errors.Join(err, cause)
					case "retry succeeds before list":
						if retries == 1 {
							return nil
						}
						return err
					case "retry clean final fallback":
						if retries == 1 {
							return nil
						}
						return err
					}
					return err
				}
			}
			got, err := New(client).FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "selected")
			if mode == "retry succeeds before list" || mode == "retry clean final fallback" {
				wantCalls, wantRetries := 2, 1
				if mode == "retry clean final fallback" {
					wantCalls, wantRetries = 3, 2
				}
				if got == nil || err != nil || calls != wantCalls || retries != wantRetries {
					t.Fatal(got, err, calls, retries)
				}
			} else {
				if got != nil || err == nil || calls != 1 {
					t.Fatal("terminal error fell back", got, err, calls)
				}
				if mode != "native 500" && !errors.Is(err, cause) {
					t.Fatal("cause lost", err)
				}
				if strings.HasPrefix(mode, "accepted") {
					memberRecordProof(t, err, 201, `{"member":"actual"}`)
					if errors.Is(err, resource.ErrNotFound) {
						t.Fatal("nested404 misclassified", err)
					}
				}
				if mode == "native 500" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 500 || native.URL != memberRecordPrefix+url.PathEscape(memberRecordParent)+"/members/selected" || string(native.Body) != `{"member":"actual"}` || native.ResponseHeader.Get("X-Member-Proof") != "actual" {
						t.Fatal(native, err)
					}
				}
			}
			for _, body := range bodies {
				if body.closes != 1 {
					t.Fatal("native body closes", body.closes)
				}
			}
		})
	}
}

func TestFindImageMemberRecordOptionsLocationAndFallbackSnapshot(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "captured"
	headers := map[string]string{"X-Option": "factory snapshot"}
	ignore := false
	option := WithFindImageMemberRecordOpts(FindImageMemberRecordOpts{Headers: headers, IgnoreMissing: &ignore})
	headers["X-Option"] = "caller changed"
	ignore = true
	var retained *FindImageMemberRecordOpts
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory snapshot" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" {
			t.Fatal("Find snapshot", req.Header)
		}
		retained.Headers["X-Option"] = "retained changed"
		*retained.IgnoreMissing = true
		if calls == 1 {
			return memberRecordJSON(403, `{}`), nil
		}
		return memberRecordJSON(200, `{"members":[]}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	location := resource.CloudLocation{Cloud: &cloud}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "changed by location"
		return location, nil
	}})
	got, err := service.FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "selected", option, func(value *FindImageMemberRecordOpts) error {
		callbacks++
		retained = value
		cloud = "changed"
		client.MoreHeaders["X-Source"] = "changed"
		client.SetToken("live")
		return WithFindImageMemberRecordHeader("X-Final", "yes")(value)
	})
	if got != nil || !errors.Is(err, resource.ErrNotFound) || calls != 2 || callbacks != 1 || locations != 1 {
		t.Fatal(got, err, calls, callbacks, locations)
	}
	t.Run("same captured location in list candidate", func(t *testing.T) {
		calls = 0
		locations = 0
		cloud = "captured"
		client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				cloud = "changed by GET"
				return memberRecordJSON(404, `{}`), nil
			}
			return memberRecordJSON(200, `{"members":[{"member":"selected"}]}`), nil
		})
		service = NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
		got, err := service.FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "selected")
		var captured resource.CloudLocation
		if got == nil || err != nil || locations != 1 || calls != 2 {
			t.Fatal(got, err, calls, locations)
		}
		if err := json.Unmarshal(got.Resource.Body["location"], &captured); err != nil || captured.Cloud == nil || *captured.Cloud != "captured" {
			t.Fatal(captured, err)
		}
	})
}

func TestFindImageMemberRecordPreflightAndLegacyBoundaries(t *testing.T) {
	for _, selected := range []string{"", " ", ".", "..", "\n", "bad\x00id", string([]byte{0xff})} {
		t.Run(fmt.Sprintf("invalid %q", selected), func(t *testing.T) {
			calls, callbacks, locations := 0, 0, 0
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
			got, err := service.FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), selected, func(*FindImageMemberRecordOpts) error { callbacks++; return nil })
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 || locations != 0 {
				t.Fatal(got, err, calls, callbacks, locations)
			}
		})
	}
	for _, option := range []FindImageMemberRecordOption{nil, WithFindImageMemberRecordHeader("X-Auth-Token", "foreign"), WithFindImageMemberRecordHeader("X-Extra", "\n")} {
		t.Run(fmt.Sprintf("option %p", option), func(t *testing.T) {
			calls := 0
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			got, err := New(client).FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "selected", option)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("legacy direct Find keeps finite absence policy", func(t *testing.T) {
		calls := 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return memberRecordJSON(404, "legacy missing"), nil
		})
		got, err := New(client).FindImageMember(context.Background(), resource.ID(memberRecordParent), "selected")
		if got != nil || err != nil || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("legacy direct Find403 has no list fallback", func(t *testing.T) {
		calls := 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return memberRecordJSON(403, "legacy forbidden"), nil
		})
		got, err := New(client).FindImageMember(context.Background(), resource.ID(memberRecordParent), "selected")
		var native gophercloud.ErrUnexpectedResponseCode
		if got != nil || !errors.As(err, &native) || native.Actual != 403 || calls != 1 {
			t.Fatal(got, err, native, calls)
		}
	})
}
