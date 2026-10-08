package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageRecordIdentityRejectsUnpairedSurrogatesBeforeHTTP(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		channel   int
	}{
		{"high surrogate from Resource", `"\uD800"`, 0},
		{"low surrogate from request attributes", `"\uDC00"`, 1},
		{"high followed by nonlow from option attributes", `"\uD800\u0041"`, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				t.Fatal("invalid raw identity selected a transport target", req.URL)
				return nil, nil
			})
			input := ImageRecordRequest{}
			options := []ImageRecordOption{}
			switch test.channel {
			case 0:
				input.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(test.raw)}}}
			case 1:
				input.Attributes = map[string]any{"id": json.RawMessage(test.raw)}
			case 2:
				options = append(options, WithImageRecordAttribute("id", json.RawMessage(test.raw)))
			}
			got, err := New(client).GetImageRecord(context.Background(), input, options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordIdentityPreservesValidUnicodeRoutes(t *testing.T) {
	for _, test := range []struct{ name, raw, target string }{
		{"surrogate pair", `"\uD83D\uDE80"`, "🚀"},
		{"literal replacement character", `"�"`, "�"},
		{"escaped replacement character", `"\uFFFD"`, "�"},
		{"literal backslash u text", `"\\uD800"`, `\uD800`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(test.target) || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("identity text was changed", req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, `{}`), nil
			})
			seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(test.raw)}}}
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed})
			if got == nil || err != nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
			th.AssertEquals(t, test.target, taskCoreText(t, got.Resource.Body["id"]))
			th.AssertEquals(t, test.raw, string(seed.Body["id"]))
		})
	}
}

func TestImageRecordIdentityFromAcceptedResponseCannotSelectNewTarget(t *testing.T) {
	for _, operation := range []string{"get", "update", "add tag", "remove tag", "wait status", "wait delete"} {
		t.Run(operation, func(t *testing.T) {
			const body = `{"id":"\uD800","status":"pending","tags":["tag"]}`
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
					t.Fatal("passive response identity selected another target", calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, body), nil
			})
			service := New(client)
			seed, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
			if seed == nil || err != nil || calls != 1 || string(seed.Resource.Body["id"]) != `"\uD800"` || string(seed.Wire.Body["id"]) != `"\uD800"` {
				t.Fatal("raw passive response must remain available", seed, err, calls)
			}
			switch operation {
			case "get":
				got, failure := service.GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed.Resource})
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "update":
				got, failure := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "changed"}})
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "add tag":
				got, failure := service.AddImageRecordTag(context.Background(), ImageRecordTagRequest{Record: seed}, "new tag")
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "remove tag":
				got, failure := service.RemoveImageRecordTag(context.Background(), ImageRecordTagRequest{Record: seed}, "tag")
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "wait status":
				got, failure := service.WaitForImageRecordStatus(context.Background(), seed, "active")
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "wait delete":
				got, failure := service.WaitForImageRecordDelete(context.Background(), seed)
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			}
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || string(seed.Resource.Body["id"]) != `"\uD800"` || string(seed.Envelope) != body {
				t.Fatal("identity reuse must fail without another HTTP request or seed mutation", err, calls, seed)
			}
		})
	}
}

func TestImageRecordUpdateIdentityRejectsSurrogateBeforeEquality(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		option    bool
	}{
		{"request high surrogate", `"\uD800"`, false},
		{"option low surrogate", `"\uDC00"`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodGet {
					t.Fatal("replacement-character equality hid an invalid supplied identity", req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, `{"id":"�","name":"before"}`), nil
			})
			service := New(client)
			seed, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
			if seed == nil || err != nil {
				t.Fatal(seed, err)
			}
			input := ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "after"}}
			options := []ImageRecordOption{}
			if test.option {
				options = append(options, WithImageRecordAttribute("id", json.RawMessage(test.raw)))
			} else {
				input.Attributes["id"] = json.RawMessage(test.raw)
			}
			got, err := service.UpdateImageRecord(context.Background(), input, options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || string(seed.Resource.Body["id"]) != `"�"` || string(seed.Resource.Body["name"]) != `"before"` {
				t.Fatal("invalid ID must be rejected before dirty comparison and PATCH", got, err, calls, seed)
			}
		})
	}
}

func TestImageMemberRecordIdentityFromAcceptedResponseCannotSelectNewTarget(t *testing.T) {
	for _, operation := range []string{"get", "update", "remove", "alternate member identity", "option route override"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			body := `{"id":"\uD800","member":"valid alternate"}`
			if operation == "alternate member identity" {
				body = `{"member":"\uDC00"}`
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if operation == "option route override" || calls != 1 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/fixed" {
					t.Fatal("invalid member identity selected a transport target", calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, body), nil
			})
			service := New(client)
			var err error
			var seed *ImageMemberRecord
			if operation != "option route override" {
				seed, err = service.GetImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "fixed"})
				if seed == nil || err != nil || string(seed.Envelope) != body {
					t.Fatal("raw passive member response must remain available", seed, err)
				}
			}
			switch operation {
			case "get", "alternate member identity":
				got, failure := service.GetImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "update":
				got, failure := service.UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed}, WithImageMemberRecordStatus("accepted"))
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "remove":
				got, failure := service.RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			case "option route override":
				got, failure := service.UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "fixed"}, WithImageMemberRecordWriteAttribute("id", json.RawMessage(`"\uD800\u0041"`)))
				err = failure
				if got != nil {
					t.Fatal(got)
				}
			}
			wantCalls := 1
			if operation == "option route override" {
				wantCalls = 0
			}
			if !errors.Is(err, resource.ErrInvalidOption) || calls != wantCalls {
				t.Fatal("invalid member route must fail before transport", err, calls)
			}
		})
	}
}

func TestImageRecordFindStrictPassiveIdentity(t *testing.T) {
	for _, family := range []string{"image", "member"} {
		for _, test := range []struct {
			name, field, raw string
			match            bool
		}{
			{"unpaired ID is unmatched", "id", `"\uD800"`, false},
			{"unpaired name is unmatched", "name", `"\uDC00"`, false},
			{"literal replacement ID matches", "id", `"�"`, true},
			{"escaped replacement name matches", "name", `"\uFFFD"`, true},
		} {
			t.Run(family+"/"+test.name, func(t *testing.T) {
				calls := 0
				row := `{"id":"other","member":"other","name":` + test.raw + `}`
				if test.field == "id" {
					row = `{"id":` + test.raw + `,"member":"other","name":"other"}`
				}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodGet || req.Body != nil {
						t.Fatal(req.Method, req.Body)
					}
					base := "/reverse/glance/v2/images"
					key := "images"
					if family == "member" {
						base += "/" + url.PathEscape(memberRecordParent) + "/members"
						key = "members"
					}
					if calls == 1 {
						if req.URL.EscapedPath() != base+"/"+url.PathEscape("�") || req.URL.RawQuery != "" {
							t.Fatal(req.URL)
						}
						return taskCoreJSON(req, 404, `{"message":"missing direct identity"}`), nil
					}
					if req.URL.EscapedPath() != base || req.URL.Query().Has("marker") || req.URL.Query().Has("limit") {
						t.Fatal("passive matching changed routing", req.URL)
					}
					if calls == 2 {
						return taskCoreJSON(req, 203, `{"`+key+`":[`+row+`],"next":null}`), nil
					}
					if family != "image" || calls != 3 || test.match || req.URL.Query().Get("os_hidden") != "True" {
						t.Fatal("unexpected discovery phase", calls, req.URL)
					}
					return taskCoreJSON(req, 203, `{"images":[],"next":null}`), nil
				})
				service := New(client)
				var err error
				matched := false
				if family == "image" {
					got, failure := service.FindImageRecord(context.Background(), "�")
					err, matched = failure, got != nil
					if got != nil {
						th.AssertEquals(t, test.raw, string(got.Resource.Body[test.field]))
					}
				} else {
					got, failure := service.FindImageMemberRecord(context.Background(), resource.ID(memberRecordParent), "�")
					err, matched = failure, got != nil
					if got != nil {
						th.AssertEquals(t, test.raw, string(got.Resource.Body[test.field]))
					}
				}
				wantCalls := 2
				if family == "image" && !test.match {
					wantCalls = 3
				}
				if err != nil || matched != test.match || calls != wantCalls {
					t.Fatal("passive malformed Unicode must not match U+FFFD", matched, err, calls)
				}
			})
		}
	}
}

func TestImageRecordPagingStrictRawMarker(t *testing.T) {
	for _, test := range []struct {
		name, family, row, target string
		invalid                   bool
	}{
		{"image high surrogate marker", "image", `{"id":"\uD800"}`, "", true},
		{"member low surrogate alternate marker", "member", `{"member":"\uDC00"}`, "", true},
		{"member explicit invalid ID wins valid alternate", "member", `{"id":"\uD800\u0041","member":"valid alternate"}`, "", true},
		{"image valid pair marker", "image", `{"id":"\uD83D\uDE80"}`, "🚀", false},
		{"member literal backslash u marker", "member", `{"member":"\\uD800"}`, `\uD800`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			key, base := "images", "/reverse/glance/v2/images"
			if test.family == "member" {
				key, base = "members", base+"/"+url.PathEscape(memberRecordParent)+"/members"
			}
			first := `{"` + key + `":[` + test.row + `]}`
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != base || req.Body != nil {
					t.Fatal(req.Method, req.URL)
				}
				if calls == 1 {
					th.AssertEquals(t, "limit=1", req.URL.RawQuery)
					return taskCoreJSON(req, 203, first), nil
				}
				if test.invalid || calls != 2 {
					t.Fatal("invalid wire marker selected a continuation target", calls, req.URL)
				}
				th.AssertEquals(t, test.target, req.URL.Query().Get("marker"))
				th.AssertEquals(t, url.Values{"limit": {"1"}, "marker": {test.target}}.Encode(), req.URL.RawQuery)
				return taskCoreJSON(req, 203, `{"`+key+`":[]}`), nil
			})
			service := New(client)
			var err error
			count := 0
			if test.family == "image" {
				got, failure := service.AllImageRecords(context.Background(), WithImageRecordListLimit(1))
				count, err = len(got), failure
			} else {
				got, failure := service.AllImageMemberRecords(context.Background(), resource.ID(memberRecordParent), WithImageMemberRecordListLimit(1))
				count, err = len(got), failure
			}
			if count != 1 {
				t.Fatal("consumed row was lost", count, err)
			}
			if test.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
					t.Fatal(err, calls)
				}
				taskCoreProof(t, err, 203, first)
			} else if err != nil || calls != 2 {
				t.Fatal("valid marker text was not retained", err, calls)
			}
		})
	}
}
