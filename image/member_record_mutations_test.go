package image

import (
	"bytes"
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

func memberRecordMutationWrite(service *Service, update bool, ctx context.Context, options ...ImageMemberRecordWriteOption) (*ImageMemberRecord, error) {
	if update {
		return service.UpdateImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, options...)
	}
	return service.AddImageMemberRecord(ctx, resource.ID(memberRecordParent), options...)
}

func memberRecordMutationCheckPayload(t *testing.T, req *http.Request, expected string) {
	t.Helper()
	actual := taskCorePayload(t, req)
	var want map[string]json.RawMessage
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	th.CheckDeepEquals(t, want, actual)
}

func memberRecordMutationSeed(fields map[string]json.RawMessage) *ImageMemberRecord {
	parent := "untrusted record parent"
	return &ImageMemberRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: http.Header{"X-Seed": {"old"}}, StatusCode: 500}}, Wire: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"wire decoy"`)}}}, ImageID: &parent, Envelope: json.RawMessage(`{"old":true}`), Header: http.Header{"X-Seed": {"old"}}, StatusCode: 500}
}

func TestAddImageMemberRecordOwnsFlatDeclaredBodyAndIgnoresUnknown(t *testing.T) {
	for _, test := range []struct {
		name, body, member string
		options            []ImageMemberRecordWriteOption
	}{
		{"empty constructor posts empty object", `{}`, `null`, nil},
		{"seven raw fields only", `{"id":false,"name":["raw",null],"member":"project","created_at":900719925474099312345,"status":{"future":true},"schema":false,"updated_at":null}`, `"project"`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttributes(map[string]any{"id": false, "name": []any{"raw", nil}, "member_id": "project", "created_at": json.RawMessage(`900719925474099312345`), "status": map[string]any{"future": true}, "schema": false, "updated_at": nil, "vendor": make(chan int), "location": map[string]any{"foreign": true}})}},
		{"canonical member wins wire", `{"member":null}`, `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttributes(map[string]any{"member": "wire", "member_id": nil})}},
		{"wire alias alone", `{"member":false}`, `false`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttribute("member", false)}},
		{"bulk replaces previous attrs", `{"status":null}`, `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordMemberID("discarded"), WithImageMemberRecordWriteAttributes(map[string]any{"status": nil})}},
		{"bulk clear posts empty object", `{}`, `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordMemberID("discarded"), WithImageMemberRecordWriteAttributes(nil)}},
		{"concrete helpers retain raw values", `{"member":[null,true],"status":"future state"}`, `[null,true]`, []ImageMemberRecordWriteOption{WithImageMemberRecordMemberID([]any{nil, true}), WithImageMemberRecordStatus("future state")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members" || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL)
				}
				memberRecordMutationCheckPayload(t, req, test.body)
				return taskCoreJSON(req, 201, `{}`), nil
			})
			got, err := memberRecordMutationWrite(New(client), false, context.Background(), test.options...)
			if got == nil || err != nil || calls != 1 || len(got.Resource.Body) != 9 || got.ImageID == nil || *got.ImageID != memberRecordParent || got.Wire == nil || string(got.Envelope) != `{}` {
				t.Fatal(got, err, calls)
			}
			th.AssertEquals(t, test.member, string(got.Resource.Body["member_id"]))
			th.AssertEquals(t, `"image-空白"`, string(got.Resource.Body["image_id"]))
			if _, present := got.Resource.Body["vendor"]; present {
				t.Fatal("unknown attribute leaked to view", got.Resource.Body)
			}
		})
	}
	t.Run("unknown marshaler captured once and ignored", func(t *testing.T) {
		marshals, calls := 0, 0
		marker := errors.New("unknown attribute encoder")
		option := WithImageMemberRecordWriteAttribute("vendor", imageRecordMarshalCallback(func() ([]byte, error) { marshals++; return nil, marker }))
		if marshals != 1 {
			t.Fatal("attribute was not captured at factory", marshals)
		}
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			memberRecordMutationCheckPayload(t, req, `{}`)
			return taskCoreJSON(req, 200, `{}`), nil
		})
		got, err := memberRecordMutationWrite(New(client), false, context.Background(), option)
		if got == nil || err != nil || calls != 1 || marshals != 1 {
			t.Fatal(got, err, calls, marshals)
		}
	})
}

func TestUpdateImageMemberRecordCreatesFreshBoundMemberAndRawBody(t *testing.T) {
	for _, test := range []struct {
		name, body, route, status string
		options                   []ImageMemberRecordWriteOption
	}{
		{"no attrs still commits bound member", `{"member":"selected"}`, "selected", `null`, nil},
		{"unknown only still commits bound member", `{"member":"selected"}`, "selected", `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttribute("vendor", make(chan int))}},
		{"future status", `{"member":"selected","status":"future"}`, "selected", `"future"`, []ImageMemberRecordWriteOption{WithImageMemberRecordStatus("future")}},
		{"null status", `{"member":"selected","status":null}`, "selected", `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordStatus(nil)}},
		{"list status", `{"member":"selected","status":[false,null]}`, "selected", `[false,null]`, []ImageMemberRecordWriteOption{WithImageMemberRecordStatus([]any{false, nil})}},
		{"literal descriptors", `{"member":"selected","name":false,"created_at":{"literal":true},"updated_at":900719925474099312345,"schema":[null]}`, "selected", `null`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttributes(map[string]any{"name": false, "created_at": map[string]any{"literal": true}, "updated_at": json.RawMessage(`900719925474099312345`), "schema": []any{nil}})}},
		{"id retarget excluded from body", `{"member":"selected","status":false}`, "after-한", `false`, []ImageMemberRecordWriteOption{WithImageMemberRecordWriteAttribute("id", "after-한"), WithImageMemberRecordStatus(false)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			seed := memberRecordMutationSeed(map[string]json.RawMessage{"id": json.RawMessage(`"selected"`), "member_id": json.RawMessage(`"record alias"`), "status": json.RawMessage(`"old status"`), "name": json.RawMessage(`"old name"`), "created_at": json.RawMessage(`"old date"`)})
			before := memberRecordValues(seed)
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/"+url.PathEscape(test.route) || req.URL.RawQuery != "" {
					t.Fatal(req.Method, req.URL)
				}
				memberRecordMutationCheckPayload(t, req, test.body)
				return taskCoreJSON(req, 299, `opaque accepted`), nil
			})
			got, err := New(client).UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed}, test.options...)
			if got == nil || err != nil || calls != 1 || got.Wire != nil || got.StatusCode != 299 || got == seed || got.Resource == seed.Resource {
				t.Fatal(got, err, calls)
			}
			th.AssertEquals(t, `"selected"`, string(got.Resource.Body["member_id"]))
			th.AssertEquals(t, test.status, string(got.Resource.Body["status"]))
			if test.name != "literal descriptors" {
				th.AssertEquals(t, `null`, string(got.Resource.Body["name"]))
				th.AssertEquals(t, `null`, string(got.Resource.Body["created_at"]))
			}
			th.CheckDeepEquals(t, before, memberRecordValues(seed))
			if seed.Header.Get("X-Seed") != "old" || got.Header.Get("X-Seed") != "" || *got.ImageID != memberRecordParent {
				t.Fatal("seed receipt/scope became mutation data", got, seed)
			}
		})
	}
}

func TestImageMemberRecordMutationUsesPresentIdentityBeforeAlias(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, test := range []struct {
			name, id, alias, route string
			missingID, bad         bool
		}{
			{"explicit id", `"actual"`, `"decoy"`, "actual", false, false},
			{"literal escaped identity", `"a /한:%?\\b"`, `"decoy"`, "a /한:%?\\b", false, false},
			{"missing id falls back", "", `"alias"`, "alias", true, false},
			{"present null blocks fallback", `null`, `"alias"`, "", false, true},
			{"present empty blocks fallback", `""`, `"alias"`, "", false, true},
			{"untyped id blocks fallback", `false`, `"alias"`, "", false, true},
		} {
			t.Run(fmt.Sprintf("remove=%v/%s", remove, test.name), func(t *testing.T) {
				fields := map[string]json.RawMessage{"member_id": json.RawMessage(test.alias)}
				if !test.missingID {
					fields["id"] = json.RawMessage(test.id)
				}
				seed := memberRecordMutationSeed(fields)
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					method := http.MethodPut
					if remove {
						method = http.MethodDelete
					}
					if req.Method != method || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/"+url.PathEscape(test.route) {
						t.Fatal(req.Method, req.URL)
					}
					if remove {
						if req.Body != nil {
							t.Fatal(req.Body)
						}
						return taskCoreJSON(req, 201, `opaque`), nil
					}
					memberRecordMutationCheckPayload(t, req, fmt.Sprintf(`{"member":%q}`, test.route))
					return taskCoreJSON(req, 200, `{}`), nil
				})
				service := New(client)
				var err error
				if remove {
					_, err = service.RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
				} else {
					_, err = service.UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
				}
				if test.bad {
					if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
						t.Fatal(err, calls)
					}
				} else if err != nil || calls != 1 {
					t.Fatal(err, calls)
				}
			})
		}
	}
}

func TestGetImageMemberRecordFreshConstructorAliasesAndAcceptedBodies(t *testing.T) {
	for _, test := range []struct {
		name             string
		code             int
		body, id, member string
		parsed           bool
	}{
		{"response member replaces fallback identity", 200, `{"member":"wire","status":[null,false],"image_id":"foreign","location":{"foreign":true}}`, `"wire"`, `"wire"`, true},
		{"explicit response id wins", 201, `{"id":"explicit","member":"wire"}`, `"explicit"`, `"wire"`, true},
		{"present null id wins", 299, `{"id":null,"member":"wire"}`, `null`, `"wire"`, true},
		{"empty object retains bound member", 300, `{}`, `"selected"`, `"selected"`, true},
		{"canonical alias later", 399, `{"member":"first","member_id":"last"}`, `"last"`, `"last"`, true},
		{"wire alias later", 203, `{"member_id":"first","member":"last"}`, `"last"`, `"last"`, true},
		{"empty accepted body", 204, "", `"selected"`, `"selected"`, false},
		{"invalid syntax accepted", 304, `{"broken":`, `"selected"`, `"selected"`, false},
		{"opaque accepted body", 202, "not JSON", `"selected"`, `"selected"`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			seed := memberRecordMutationSeed(map[string]json.RawMessage{"id": json.RawMessage(`"selected"`), "member_id": json.RawMessage(`"decoy"`), "status": json.RawMessage(`"old"`), "name": json.RawMessage(`"old name"`)})
			before := memberRecordValues(seed)
			calls, retries := 0, 0
			body := &taskCoreBody{reader: strings.NewReader(test.body)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/selected" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Option") != "owned" {
					t.Fatal(req.Method, req.URL, req.Body, req.Header)
				}
				return taskCoreHTTP(req, test.code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed}, WithImageMemberHeader("X-Option", "owned"))
			if got == nil || err != nil || calls != 1 || retries != 0 || body.closes != 1 || (got.Wire != nil) != test.parsed || got.StatusCode != test.code || string(got.Envelope) != test.body || len(got.Resource.Body) != 9 || got.ImageID == nil || *got.ImageID != memberRecordParent {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			th.AssertEquals(t, test.id, string(got.Resource.Body["id"]))
			th.AssertEquals(t, test.member, string(got.Resource.Body["member_id"]))
			th.AssertEquals(t, `null`, string(got.Resource.Body["name"]))
			th.CheckDeepEquals(t, before, memberRecordValues(seed))
			if test.name == "response member replaces fallback identity" {
				th.AssertEquals(t, `[null,false]`, string(got.Resource.Body["status"]))
			} else {
				th.AssertEquals(t, `null`, string(got.Resource.Body["status"]))
			}
		})
	}
}

func TestGetImageMemberRecordMalformedAcceptedBodyRetainsFreshSeedEvidence(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `"text"`, `false`, `42`, "{\"status\":\"\xff\"}"} {
		t.Run(fmt.Sprintf("body=%q", raw), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 201, raw), nil })
			got, err := New(client).GetImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"})
			if got == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || got.StatusCode != 201 || string(got.Envelope) != raw || got.Wire != nil || string(got.Resource.Body["member_id"]) != `"selected"` || string(got.Resource.Body["id"]) != `"selected"` {
				t.Fatal(got, err, calls)
			}
			taskCoreProof(t, err, 201, raw)
		})
	}
}

func TestImageMemberRecordFamilyRejectsNativeStatusesWithoutFakeReceipt(t *testing.T) {
	for _, operation := range []string{"add", "get", "update", "remove"} {
		for _, code := range []int{403, 404, 500} {
			t.Run(fmt.Sprintf("%s/%d", operation, code), func(t *testing.T) {
				const raw = `{"message":"rejected"}`
				calls := 0
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil })
				service := New(client)
				var got *ImageMemberRecord
				var ack *ImageMemberAcknowledgement
				var err error
				if operation == "get" {
					got, err = service.GetImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"})
				} else if operation == "remove" {
					ack, err = service.RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, WithRemoveImageMemberIgnoreMissing(false))
				} else {
					got, err = memberRecordMutationWrite(service, operation == "update", context.Background())
				}
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if got != nil || ack != nil || err == nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != raw || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &accepted) || calls != 1 || body.closes != 1 {
					t.Fatal(got, ack, err, native, calls, body.closes)
				}
			})
		}
	}
}

func TestImageMemberRecordWritesTranslateAcceptedStatusesAndPreserveDefaults(t *testing.T) {
	for _, update := range []bool{false, true} {
		for _, test := range []struct {
			code   int
			raw    string
			parsed bool
		}{
			{200, `{"member":"wire","status":[false,null],"created_at":900719925474099312345,"schema":{"passive":true},"image_id":"foreign","location":{"foreign":true},"vendor":1e400}`, true},
			{201, `{"id":null,"member":"wire"}`, true},
			{204, "", false}, {299, `{}`, true}, {300, `{"name":"server"}`, true}, {304, `{"broken":`, false}, {399, "opaque accepted", false},
		} {
			t.Run(fmt.Sprintf("update=%v/%d", update, test.code), func(t *testing.T) {
				calls, retries := 0, 0
				body := &taskCoreBody{reader: strings.NewReader(test.raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreHTTP(req, test.code, body), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				got, err := memberRecordMutationWrite(New(client), update, context.Background(), WithImageMemberRecordWriteAttribute("name", "submitted"), WithImageMemberRecordStatus(nil))
				if got == nil || err != nil || calls != 1 || retries != 0 || body.closes != 1 || (got.Wire != nil) != test.parsed || got.StatusCode != test.code || string(got.Envelope) != test.raw || len(got.Resource.Body) != 9 || got.Resource.Header.Get("X-Task-Proof") != "actual" || got.ImageID == nil || *got.ImageID != memberRecordParent {
					t.Fatal(got, err, calls, retries, body.closes)
				}
				th.AssertEquals(t, `"image-空白"`, string(got.Resource.Body["image_id"]))
				th.AssertEquals(t, `null`, string(got.Resource.Body["location"]))
				if test.code == 201 {
					th.AssertEquals(t, `null`, string(got.Resource.Body["id"]))
				}
				if test.code == 200 {
					th.AssertEquals(t, `[false,null]`, string(got.Resource.Body["status"]))
					th.AssertEquals(t, `"foreign"`, string(got.Wire.Body["image_id"]))
					th.AssertEquals(t, `1e400`, string(got.Wire.Body["vendor"]))
				}
			})
		}
	}
}

func TestImageMemberRecordWritesMalformedAcceptedBodiesKeepSubmittedReceipt(t *testing.T) {
	for _, update := range []bool{false, true} {
		for _, raw := range []string{`null`, `[]`, `"text"`, `false`, `42`, "{\"status\":\"\xff\"}"} {
			t.Run(fmt.Sprintf("update=%v/body=%q", update, raw), func(t *testing.T) {
				calls, retries := 0, 0
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, body), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				got, err := memberRecordMutationWrite(New(client), update, context.Background(), WithImageMemberRecordWriteAttribute("name", "submitted"))
				if got == nil || !errors.Is(err, resource.ErrInvalidOption) || got.StatusCode != 201 || string(got.Envelope) != raw || got.Header.Get("X-Task-Proof") != "actual" || string(got.Resource.Body["name"]) != `"submitted"` || got.Wire != nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, retries, body.closes)
				}
				taskCoreProof(t, err, 201, raw)
			})
		}
	}
}

func TestImageMemberRecordWriteOptionsAndLocationAreOwned(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("update=%v", update), func(t *testing.T) {
			headers := map[string]string{"X-Option": "factory"}
			values := map[string]any{"name": "factory name", "status": []any{false, nil}}
			attrs := []resource.ListOption{resource.WithFilters(values)}
			option := WithImageMemberRecordWriteOpts(ImageMemberRecordWriteOpts{Headers: headers, Attributes: attrs})
			headers["X-Option"], values["name"] = "caller changed", "caller changed"
			attrs[0] = resource.WithFilters(map[string]any{"name": "slice changed"})
			cloud := "captured"
			facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
			calls, callbacks, locations := 0, 0, 0
			var retained *ImageMemberRecordWriteOpts
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Fatal(req.Header)
				}
				want := `{"name":"factory name","status":[false,null]}`
				if update {
					want = `{"member":"selected","name":"factory name","status":[false,null]}`
				}
				memberRecordMutationCheckPayload(t, req, want)
				retained.Headers["X-Option"] = "retained changed"
				retained.Attributes[0] = resource.WithFilters(map[string]any{"name": "retained changed"})
				return taskCoreJSON(req, 200, `{}`), nil
			})
			client.MoreHeaders, client.Microversion = map[string]string{"X-Source": "captured"}, "2.10"
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				client.MoreHeaders["X-Source"] = "changed in callback"
				return facts, nil
			}})
			got, err := memberRecordMutationWrite(service, update, context.Background(), option, func(config *ImageMemberRecordWriteOpts) error {
				callbacks++
				retained = config
				cloud = "after option"
				facts.Project.ID[1] = 'X'
				client.SetToken("live")
				return WithImageMemberRecordWriteHeader("X-Final", "yes")(config)
			})
			if got == nil || err != nil || calls != 1 || callbacks != 1 || locations != 1 {
				t.Fatal(got, err, calls, callbacks, locations)
			}
			var location resource.CloudLocation
			if err := json.Unmarshal(got.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"token project"` {
				t.Fatal(location, err)
			}
		})
	}
}

func TestImageMemberRecordMutationsCaptureChildIDBeforeCallbacks(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprintf("remove=%v", remove), func(t *testing.T) {
			seed := memberRecordMutationSeed(map[string]json.RawMessage{"id": json.RawMessage(`"captured"`), "status": json.RawMessage(`"not submitted"`)})
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/captured" {
					t.Fatal("caller callback retargeted mutation", req.URL)
				}
				if !remove {
					memberRecordMutationCheckPayload(t, req, `{"member":"captured"}`)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				seed.Resource.Body["id"] = json.RawMessage(`"changed"`)
				return resource.CloudLocation{}, nil
			}})
			var err error
			if remove {
				_, err = service.RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
			} else {
				_, err = service.UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{Record: seed})
			}
			if err != nil || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}

func TestRemoveImageMemberRecordRetainsEveryAcceptedOpaqueAcknowledgement(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			raw := []byte{'o', 'p', 'a', 'q', 'u', 'e', 0xff}
			calls, retries := 0, 0
			body := &taskCoreBody{reader: bytes.NewReader(raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(memberRecordParent)+"/members/selected" || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal(req.Method, req.URL, req.Body)
				}
				return taskCoreHTTP(req, code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			ack, err := New(client).RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"})
			if ack == nil || err != nil || ack.ImageID != memberRecordParent || ack.MemberID != "selected" || ack.StatusCode != code || !bytes.Equal(ack.Body, raw) || ack.Header.Get("X-Task-Proof") != "actual" || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(ack, err, calls, retries, body.closes)
			}
			ack.Body[0] = '!'
			if raw[0] != 'o' {
				t.Fatal("ack body aliases source", raw)
			}
		})
	}
}

func TestRemoveImageMemberRecordMissingPolicyHonorsOnlyCleanFinal404(t *testing.T) {
	for _, mode := range []string{"default ignored", "false strict", "true ignored", "retry clean final", "retry hook cause", "read", "close", "cancel", "source", "nested transport404"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("member remove failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &taskCoreBody{reader: strings.NewReader(`{"message":"missing"}`)}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: marker}
			case "close":
				body.closeErr = marker
			case "cancel":
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: io.EOF, action: func() { cancel(marker) }}
			case "source":
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: io.EOF, action: func() { client.Endpoint = "https://foreign.test/" }}
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "nested transport404" {
					return nil, errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Method: req.Method, URL: "https://decoy.test/", Actual: 404})
				}
				if mode == "retry clean final" && calls == 1 {
					return taskCoreJSON(req, 503, `{"message":"retry"}`), nil
				}
				return taskCoreHTTP(req, 404, body), nil
			})
			if strings.HasPrefix(mode, "retry") {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
					retries++
					if mode == "retry hook cause" {
						return errors.Join(original, marker)
					}
					if count == 1 {
						return nil
					}
					return original
				}
			}
			options := []RemoveImageMemberOption{}
			if mode == "false strict" {
				options = append(options, WithRemoveImageMemberIgnoreMissing(false))
			}
			if mode == "true ignored" {
				options = append(options, WithRemoveImageMemberIgnoreMissing(true))
			}
			ack, err := New(client).RemoveImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, options...)
			clean := mode == "default ignored" || mode == "true ignored" || mode == "retry clean final"
			if ack != nil || clean && err != nil || !clean && err == nil {
				t.Fatal(ack, err, calls, retries)
			}
			wantCalls := 1
			if mode == "retry clean final" {
				wantCalls = 2
			}
			if calls != wantCalls || mode != "nested transport404" && body.closes != 1 {
				t.Fatal(calls, body.closes)
			}
			if !clean && mode != "false strict" && mode != "source" && !errors.Is(err, marker) {
				t.Fatal("handling cause was ignored", err)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if mode == "false strict" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"message":"missing"}` {
					t.Fatal(native, err)
				}
			}
		})
	}
}

func TestImageMemberRecordMutationsAcceptedPhysicalFailuresKeepPartialEvidence(t *testing.T) {
	for _, operation := range []string{"add", "get", "update", "remove"} {
		for _, mode := range []string{"read", "close", "cancel", "source drift", "read drift restored on Close", "outer drift restored on Close"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				const raw = `{"name":"actual"}`
				marker := errors.New("member accepted failure")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, retries := 0, 0
				outerInvalid := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if outerInvalid {
						return marker
					}
					return nil
				})
				var client *gophercloud.ServiceClient
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				action := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
				case "cancel":
					action = func() { cancel(marker) }
				case "source drift", "read drift restored on Close":
					action = func() { client.Endpoint = "https://foreign.test/" }
				case "outer drift restored on Close":
					action = func() { outerInvalid = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
				}
				selected := io.ReadCloser(body)
				if strings.Contains(mode, "restored") {
					selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerInvalid = false }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, selected), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				service := New(client)
				var err error
				if operation == "remove" {
					ack, failure := service.RemoveImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"})
					err = failure
					if ack == nil || ack.StatusCode != 201 || string(ack.Body) != raw || ack.Header.Get("X-Task-Proof") != "actual" {
						t.Fatal(ack, err)
					}
				} else {
					var got *ImageMemberRecord
					wantName := `"submitted"`
					if operation == "get" {
						got, err = service.GetImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"})
						wantName = `null`
					} else {
						got, err = memberRecordMutationWrite(service, operation == "update", ctx, WithImageMemberRecordWriteAttribute("name", "submitted"))
					}
					if got == nil || got.StatusCode != 201 || string(got.Envelope) != raw || got.Header.Get("X-Task-Proof") != "actual" || string(got.Resource.Body["name"]) != wantName {
						t.Fatal(got, err)
					}
				}
				if err == nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(err, calls, retries, body.closes)
				}
				if strings.Contains(mode, "source") || mode == "read drift restored on Close" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal("cause lost", err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted fault was treated as absence", err)
				}
				taskCoreProof(t, err, 201, raw)
			})
		}
	}
}

func TestImageMemberRecordMutationsFullPreflight(t *testing.T) {
	for _, update := range []bool{false, true} {
		for _, option := range []ImageMemberRecordWriteOption{nil, WithImageMemberRecordWriteHeader("X-Auth-Token", "foreign"), WithImageMemberRecordWriteHeader("X-Extra", "\n"), WithImageMemberRecordWriteAttribute("status", make(chan int)), WithImageMemberRecordWriteOpts(ImageMemberRecordWriteOpts{Attributes: []resource.ListOption{resource.WithQuery("status", "bad")}})} {
			t.Run(fmt.Sprintf("update=%v/option=%p", update, option), func(t *testing.T) {
				calls := 0
				service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
				got, err := memberRecordMutationWrite(service, update, context.Background(), option)
				if got != nil || err == nil || calls != 0 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, collision := range []string{"member", "member_id", "image_id"} {
		t.Run("update collision "+collision, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
			got, err := memberRecordMutationWrite(service, true, context.Background(), WithImageMemberRecordWriteAttribute(collision, "duplicate"))
			if got != nil || err == nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, input := range []ImageMemberRecordRequest{{}, {ID: "selected", Record: &ImageMemberRecord{}}, {Record: &ImageMemberRecord{}}, {ID: "."}, {ID: ".."}, {ID: "bad\n"}, {ID: string([]byte{0xff})}} {
		t.Run(fmt.Sprintf("child=%+v", input), func(t *testing.T) {
			calls, callbacks := 0, 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
			got, err := service.UpdateImageMemberRecord(context.Background(), resource.ID(memberRecordParent), input, func(*ImageMemberRecordWriteOpts) error { callbacks++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatal(got, err, calls, callbacks)
			}
			ack, err := service.RemoveImageMemberRecord(context.Background(), resource.ID(memberRecordParent), input)
			if ack != nil || err == nil || calls != 0 {
				t.Fatal(ack, err, calls)
			}
		})
	}
	for _, operation := range []string{"add", "get", "update", "remove"} {
		for _, mode := range []string{"nil context", "canceled", "nil service", "invalid parent"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				calls, callbacks := 0, 0
				ctx := context.Background()
				service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
				parent := resource.ID(memberRecordParent)
				if mode == "nil context" {
					ctx = nil
				}
				if mode == "canceled" {
					canceled, cancel := context.WithCancel(context.Background())
					cancel()
					ctx = canceled
				}
				if mode == "nil service" {
					service = nil
				}
				if mode == "invalid parent" {
					parent = resource.Ref{}
				}
				var err error
				if operation == "remove" {
					_, err = service.RemoveImageMemberRecord(ctx, parent, ImageMemberRecordRequest{ID: "selected"}, func(*RemoveImageMemberOpts) error { callbacks++; return nil })
				} else if operation == "get" {
					_, err = service.GetImageMemberRecord(ctx, parent, ImageMemberRecordRequest{ID: "selected"}, func(*ImageMemberOpts) error { callbacks++; return nil })
				} else if operation == "update" {
					_, err = service.UpdateImageMemberRecord(ctx, parent, ImageMemberRecordRequest{ID: "selected"}, func(*ImageMemberRecordWriteOpts) error { callbacks++; return nil })
				} else {
					_, err = service.AddImageMemberRecord(ctx, parent, func(*ImageMemberRecordWriteOpts) error { callbacks++; return nil })
				}
				if err == nil || calls != 0 || callbacks != 0 {
					t.Fatal(err, calls, callbacks)
				}
			})
		}
	}
}

func TestImageMemberRecordFamilyReusesExplicitParentNameLookup(t *testing.T) {
	for _, operation := range []string{"add", "get", "update", "remove"} {
		for _, missing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/missing=%v", operation, missing), func(t *testing.T) {
				calls, callbacks := 0, 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Header.Get("X-Call") != "parent" {
						t.Fatal(req.Header)
					}
					if calls == 1 {
						if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images" || req.URL.Query().Get("name") != "needle" {
							t.Fatal(req.Method, req.URL)
						}
						if missing {
							return taskCoreJSON(req, 200, `{"images":[]}`), nil
						}
						return taskCoreJSON(req, 200, `{"images":[{"id":"resolved-parent","name":"needle"}]}`), nil
					}
					if calls != 2 || missing {
						t.Fatal("unexpected discovery/mutation", calls, req.URL)
					}
					path, method := "/reverse/glance/v2/images/resolved-parent/members", http.MethodPost
					switch operation {
					case "get":
						method = http.MethodGet
						path += "/selected"
					case "update":
						method = http.MethodPut
						path += "/selected"
					case "remove":
						method = http.MethodDelete
						path += "/selected"
					}
					if req.Method != method || req.URL.Path != path || req.URL.RawQuery != "" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 201, `{}`), nil
				})
				service := New(client)
				var record *ImageMemberRecord
				var ack *ImageMemberAcknowledgement
				var err error
				parent := resource.Name("needle")
				writeOption := func(config *ImageMemberRecordWriteOpts) error {
					callbacks++
					return WithImageMemberRecordWriteHeader("X-Call", "parent")(config)
				}
				switch operation {
				case "add":
					record, err = service.AddImageMemberRecord(context.Background(), parent, writeOption)
				case "update":
					record, err = service.UpdateImageMemberRecord(context.Background(), parent, ImageMemberRecordRequest{ID: "selected"}, writeOption)
				case "get":
					record, err = service.GetImageMemberRecord(context.Background(), parent, ImageMemberRecordRequest{ID: "selected"}, func(config *ImageMemberOpts) error {
						callbacks++
						return WithImageMemberHeader("X-Call", "parent")(config)
					})
				case "remove":
					ack, err = service.RemoveImageMemberRecord(context.Background(), parent, ImageMemberRecordRequest{ID: "selected"}, func(config *RemoveImageMemberOpts) error {
						callbacks++
						return WithRemoveImageMemberHeader("X-Call", "parent")(config)
					})
				}
				if missing {
					if record != nil || ack != nil || err == nil || calls != 1 {
						t.Fatal("parent absence was swallowed", record, ack, err, calls)
					}
				} else {
					if err != nil || calls != 2 || record == nil && ack == nil {
						t.Fatal(record, ack, err, calls)
					}
					if record != nil && (record.ImageID == nil || *record.ImageID != "resolved-parent") {
						t.Fatal(record)
					}
					if ack != nil && ack.ImageID != "resolved-parent" {
						t.Fatal(ack)
					}
				}
				if callbacks != 1 {
					t.Fatal("options repeated across discovery", callbacks)
				}
			})
		}
	}
}

func TestImageMemberRecordFamilyStopsAfterOptionGuardFailure(t *testing.T) {
	for _, operation := range []string{"add", "get", "update", "remove"} {
		for _, mode := range []string{"cancel", "source", "binding", "outer"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				marker := errors.New("member option guard")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				invalidOuter := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if invalidOuter {
						return marker
					}
					return nil
				})
				calls, first, later := 0, 0, 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
				service := New(client)
				action := func() {
					first++
					switch mode {
					case "cancel":
						cancel(marker)
					case "source":
						client.Endpoint = "https://foreign.test/"
					case "binding":
						service.API = nil
					case "outer":
						invalidOuter = true
					}
				}
				var err error
				switch operation {
				case "add":
					_, err = service.AddImageMemberRecord(ctx, resource.ID(memberRecordParent), func(*ImageMemberRecordWriteOpts) error { action(); return nil }, func(*ImageMemberRecordWriteOpts) error { later++; return nil })
				case "update":
					_, err = service.UpdateImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, func(*ImageMemberRecordWriteOpts) error { action(); return nil }, func(*ImageMemberRecordWriteOpts) error { later++; return nil })
				case "get":
					_, err = service.GetImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, func(*ImageMemberOpts) error { action(); return nil }, func(*ImageMemberOpts) error { later++; return nil })
				case "remove":
					_, err = service.RemoveImageMemberRecord(ctx, resource.ID(memberRecordParent), ImageMemberRecordRequest{ID: "selected"}, func(*RemoveImageMemberOpts) error { action(); return nil }, func(*RemoveImageMemberOpts) error { later++; return nil })
				}
				if err == nil || calls != 0 || first != 1 || later != 0 {
					t.Fatal(err, calls, first, later)
				}
				if mode == "cancel" || mode == "outer" {
					if !errors.Is(err, marker) {
						t.Fatal("option cause lost", err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestImageMemberRecordWritesNativeRetryRetainsBodyAndLiveAuth(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(fmt.Sprintf("update=%v", update), func(t *testing.T) {
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls > 2 {
					t.Fatal("extra retry", calls)
				}
				want := `{"status":[null,false]}`
				if update {
					want = `{"member":"selected","status":[null,false]}`
				}
				memberRecordMutationCheckPayload(t, req, want)
				code := 503
				if calls == 2 {
					code = 203
					if req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("X-Retry") != "ordinary" {
						t.Fatal(req.Header)
					}
				}
				return taskCoreJSON(req, code, `{}`), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if count > 1 {
					return original
				}
				client.SetToken("retry token")
				options.MoreHeaders = map[string]string{"X-Retry": "ordinary"}
				return nil
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			got, err := memberRecordMutationWrite(New(client), update, context.Background(), WithImageMemberRecordStatus([]any{nil, false}))
			if got == nil || err != nil || got.StatusCode != 203 || calls != 2 || retries != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(got, err, calls, retries)
			}
		})
	}
}

func TestImageMemberRecordMutationResultsDoNotAliasRequestOrPhysicalResponse(t *testing.T) {
	const raw = `{"id":"returned","member":"wire","status":[null,false],"schema":{"large":900719925474099312345}}`
	header := http.Header{"X-Task-Proof": {"actual"}}
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		return &http.Response{Request: req, StatusCode: 201, Header: header, Body: io.NopCloser(strings.NewReader(raw))}, nil
	})
	got, err := New(client).AddImageMemberRecord(context.Background(), resource.ID(memberRecordParent), WithImageMemberRecordMemberID("submitted"))
	if got == nil || err != nil {
		t.Fatal(got, err)
	}
	viewBefore := memberRecordValues(got)
	got.Resource.Body["status"][0] = '!'
	got.Resource.Header.Set("X-Task-Proof", "view changed")
	got.Wire.Header.Set("X-Task-Proof", "wire changed")
	got.Header.Set("X-Task-Proof", "record changed")
	got.Envelope[0] = '!'
	*got.ImageID = "caller changed"
	if string(got.Wire.Body["status"]) != `[null,false]` || header.Get("X-Task-Proof") != "actual" || !reflect.DeepEqual(viewBefore["schema"], string(got.Resource.Body["schema"])) {
		t.Fatal("record channels alias", got)
	}
	got.Wire.Body["schema"][0] = '!'
	th.AssertEquals(t, `{"large":900719925474099312345}`, string(got.Resource.Body["schema"]))
}
