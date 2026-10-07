package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageMembersCoreFixedRoutesAndActualEvidence(t *testing.T) {
	const imageID, memberID = "image-한글", "project-한글"
	for _, operation := range []string{"add", "get", "update", "find", "remove", "list", "all"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			body := &deleteCoreBody{reader: strings.NewReader(`{"image_id":"passive-image","member_id":"passive-member","status":"extension","created_at":"not-a-time","unknown":9007199254740993123}`)}
			headers := http.Header{"X-Actual": {"captured"}, "Location": {"https://foreign.test/not-a-target"}}
			client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				path := "/reverse/glance/v2/images/" + url.PathEscape(imageID) + "/members"
				method, code := http.MethodGet, http.StatusOK
				var wantBody map[string]string
				switch operation {
				case "add":
					method, wantBody = http.MethodPost, map[string]string{"member": memberID}
				case "update":
					method, wantBody = http.MethodPut, map[string]string{"status": "accepted"}
					path += "/" + url.PathEscape(memberID)
				case "get", "find":
					path += "/" + url.PathEscape(memberID)
				case "remove":
					method, code = http.MethodDelete, http.StatusNoContent
					path += "/" + url.PathEscape(memberID)
					body.reader = bytes.NewReader([]byte{'r', 255})
				case "list", "all":
					body.reader = strings.NewReader(`{"members":[{"member_id":"first"},{"member_id":"second"}],"next":{},"links":false}`)
				}
				if r.Method != method || r.URL.EscapedPath() != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "before" {
					t.Fatalf("%s %s %v", r.Method, r.URL, r.Header)
				}
				if wantBody != nil {
					var actual map[string]string
					if err := json.NewDecoder(r.Body).Decode(&actual); err != nil || !reflect.DeepEqual(actual, wantBody) {
						t.Fatal(actual, err)
					}
				} else if r.Body != nil {
					t.Fatal("unexpected body")
				}
				return deleteCoreHTTP(code, body, headers), nil
			})
			service := New(client)
			var value *ImageMember
			var err error
			switch operation {
			case "add":
				value, err = service.AddImageMember(context.Background(), resource.ID(imageID), memberID)
			case "get":
				value, err = service.GetImageMember(context.Background(), resource.ID(imageID), memberID)
			case "update":
				value, err = service.UpdateImageMember(context.Background(), resource.ID(imageID), memberID, "accepted")
			case "find":
				value, err = service.FindImageMember(context.Background(), resource.ID(imageID), memberID)
			case "remove":
				ack, failure := service.RemoveImageMember(context.Background(), resource.ID(imageID), memberID)
				err = failure
				if ack == nil || ack.ImageID != imageID || ack.MemberID != memberID || ack.StatusCode != 204 || !bytes.Equal(ack.Body, []byte{'r', 255}) || ack.Header.Get("X-Actual") != "captured" {
					t.Fatal(ack, err)
				}
				headers.Set("X-Actual", "wire changed")
				if ack.Header.Get("X-Actual") != "captured" {
					t.Fatal("header alias")
				}
			case "list":
				count := 0
				for row, failure := range service.ListImageMembers(context.Background(), resource.ID(imageID)) {
					if failure != nil || row.StatusCode != 200 {
						t.Fatal(row, failure)
					}
					count++
				}
				if count != 2 {
					t.Fatal(count)
				}
			case "all":
				rows, failure := service.AllImageMembers(context.Background(), resource.ID(imageID))
				err = failure
				if len(rows) != 2 {
					t.Fatal(rows, err)
				}
			}
			if err != nil || calls != 1 || body.closes != 1 {
				t.Fatal(err, calls, body.closes)
			}
			if value != nil {
				if *value.ImageID != "passive-image" || *value.MemberID != "passive-member" || *value.Status != "extension" || *value.CreatedAt != "not-a-time" || value.StatusCode != 200 || string(value.Body["unknown"]) != "9007199254740993123" {
					t.Fatal(value)
				}
				headers.Set("X-Actual", "wire changed")
				if value.Header.Get("X-Actual") != "captured" {
					t.Fatal("header alias")
				}
			}
		})
	}
}

func TestImageMembersCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	data := []byte(`{"image_id":null,"member_id":"","status":"future","schema":"https://foreign.test/schema","created_at":"2026-10-03T12:34:56.123456789+09:00","updated_at":null,"links":false,"Member_ID":22,"extension":9007199254740993123456789}`)
	var value ImageMember
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if value.ImageID != nil || value.MemberID == nil || *value.MemberID != "" || value.UpdatedAt != nil || value.Links != nil || *value.CreatedAt != "2026-10-03T12:34:56.123456789+09:00" || string(value.Body["extension"]) != "9007199254740993123456789" {
		t.Fatal(value)
	}
	data[0] = 'x'
	if string(value.Body["links"]) != "false" {
		t.Fatal("input aliases raw body")
	}
	*value.MemberID = "typed change"
	if string(value.Body["member_id"]) != `""` {
		t.Fatal("typed pointer aliases raw body")
	}
	for _, invalid := range []string{"null", "[]", "", `{"created_at":3}`, `{"updated_at":[]}`, `{"image_id":false}`, `{"member_id":{}}`, `{"status":2}`, `{"schema":[]}`, string([]byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'})} {
		before := value
		if err := json.Unmarshal([]byte(invalid), &value); err == nil || !reflect.DeepEqual(value, before) {
			t.Fatalf("%q err=%v value=%+v", invalid, err, value)
		}
	}
	for _, invalid := range []string{`{"member_id":3}`, `{"updated_at":false}`, "null", "[]", string([]byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'})} {
		body := &deleteCoreBody{reader: strings.NewReader(invalid)}
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			return deleteCoreHTTP(200, body, http.Header{"X-Actual": {"decode"}}), nil
		})
		member, err := New(client).GetImageMember(context.Background(), resource.ID("image"), "member")
		var proof *resource.ResponseError
		if member != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != invalid || proof.Header.Get("X-Actual") != "decode" || body.closes != 1 {
			t.Fatal(member, err, proof, body.closes)
		}
	}
}

func TestImageMembersCoreFiniteLazyListAndConsumedRows(t *testing.T) {
	for _, cap := range []int{0, 1, 3, 20} {
		calls, callbacks := 0, 0
		client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.RawQuery != "" || r.Header.Get("X-Policy") != "owned" {
				t.Fatal(r.URL, r.Header)
			}
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"members": [{"member_id":"a"},{"member_id":"b"}],"next":"https://foreign.test/next","links":false,"schema":[]}`)), nil), nil
		})
		seq := New(client).ListImageMembers(context.Background(), resource.ID("image"), WithListImageMembersMaxItems(cap), func(config *ListImageMembersOpts) error {
			callbacks++
			config.Headers["X-Policy"] = "owned"
			return nil
		})
		if calls != 0 || callbacks != 0 {
			t.Fatal("not lazy")
		}
		for repeat := 0; repeat < 2; repeat++ {
			count := 0
			for value, err := range seq {
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				count++
			}
			want := 2
			if cap == 1 {
				want = 1
			}
			if count != want {
				t.Fatal(cap, count)
			}
		}
		if calls != 2 || callbacks != 2 {
			t.Fatal(calls, callbacks)
		}
	}
	for _, badRow := range []string{"null", "[]", "1", `{"member_id":7}`} {
		payload := `{"members":[{"member_id":"valid"},` + badRow + `]}`
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(payload)), http.Header{"X-Actual": {"whole"}}), nil
		})
		service := New(client)
		rows, err := service.AllImageMembers(context.Background(), resource.ID("image"), WithListImageMembersMaxItems(1))
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
		for row, failure := range service.ListImageMembers(context.Background(), resource.ID("image")) {
			if row == nil || failure != nil {
				t.Fatal(row, failure)
			}
			break
		}
		rows, err = service.AllImageMembers(context.Background(), resource.ID("image"))
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != payload || proof.Header.Get("X-Actual") != "whole" {
			t.Fatal(rows, err, proof)
		}
	}
	for _, payload := range []string{`{"members":[]}`, `{"members":null}`, `{"members":{}}`, `{}`, `[]`, string([]byte{'{', '"', 'm', 'e', 'm', 'b', 'e', 'r', 's', '"', ':', '[', ']', ',', '"', 'x', '"', ':', '"', 255, '"', '}'})} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(payload)), nil), nil
		})
		rows, err := New(client).AllImageMembers(context.Background(), resource.ID("image"))
		if payload == `{"members":[]}` {
			if err != nil || rows == nil || len(rows) != 0 {
				t.Fatal(rows, err)
			}
		} else {
			var proof *resource.ResponseError
			if rows != nil || !errors.As(err, &proof) || string(proof.Body) != payload {
				t.Fatal(rows, err)
			}
		}
	}
}

func TestImageMembersCoreCompletePreflightAndParentMissing(t *testing.T) {
	calls, callbacks := 0, 0
	client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("HTTP on invalid input")
		return nil, nil
	})
	service := New(client)
	option := func(*ImageMemberOpts) error { callbacks++; return nil }
	for _, id := range []string{"", ".", "..", "a/b", "a%2Fb", "a?x", "a#x", "a b", "a:b", "a\\b", string([]byte{255})} {
		for _, action := range []string{"add", "get", "update", "find", "remove"} {
			var err error
			switch action {
			case "add":
				_, err = service.AddImageMember(context.Background(), resource.Name("parent"), id, option)
			case "get":
				_, err = service.GetImageMember(context.Background(), resource.Name("parent"), id, option)
			case "update":
				_, err = service.UpdateImageMember(context.Background(), resource.Name("parent"), id, "accepted", option)
			case "find":
				_, err = service.FindImageMember(context.Background(), resource.Name("parent"), id)
			case "remove":
				_, err = service.RemoveImageMember(context.Background(), resource.Name("parent"), id)
			}
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(action, id, err)
			}
		}
	}
	for _, status := range []string{"", "Accepted", " accepted", "accepted ", "unknown"} {
		if _, err := service.UpdateImageMember(context.Background(), resource.Name("parent"), "member", status, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(status, err)
		}
	}
	if _, err := service.GetImageMember(nil, resource.ID("image"), "member", option); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := service.AllImageMembers(context.Background(), resource.Name("parent"), WithListImageMembersMaxItems(-1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 || callbacks != 0 {
		t.Fatal(calls, callbacks)
	}
	for _, mode := range []string{"logical", "wire404"} {
		calls = 0
		client = deleteCoreClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if !strings.HasSuffix(r.URL.Path, "/images") {
				t.Fatal("child request after missing parent", r.URL)
			}
			if mode == "wire404" {
				return deleteCoreHTTP(404, io.NopCloser(strings.NewReader("parent missing")), nil), nil
			}
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[]}`)), nil), nil
		})
		service = New(client)
		if value, err := service.FindImageMember(context.Background(), resource.Name("missing"), "member"); value != nil || err == nil {
			t.Fatal(value, err)
		}
		if value, err := service.RemoveImageMember(context.Background(), resource.Name("missing"), "member"); value != nil || err == nil {
			t.Fatal(value, err)
		}
		if calls != 2 {
			t.Fatal(calls)
		}
	}
}

func TestImageMembersCoreOwnedBodyFailuresAndPhysical404(t *testing.T) {
	readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("close cause"), errors.New("cancel cause")
	for _, operation := range []string{"get", "find404", "remove404", "remove204"} {
		ctx, cancel := context.WithCancelCause(context.Background())
		calls, retries := 0, 0
		body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) { cancel(cancelCause); return copy(buffer, "partial"), readCause }), closeErr: closeCause}
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			code := 200
			if strings.HasSuffix(operation, "404") {
				code = 404
			}
			if operation == "remove204" {
				code = 204
			}
			return deleteCoreHTTP(code, body, http.Header{"X-Actual": {"failed"}}), nil
		})
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		service := New(client)
		var err error
		var ack *ImageMemberAcknowledgement
		if operation == "get" {
			value, failure := service.GetImageMember(ctx, resource.ID("image"), "member")
			err = failure
			if value != nil {
				t.Fatal(value)
			}
		} else if operation == "find404" {
			value, failure := service.FindImageMember(ctx, resource.ID("image"), "member")
			err = failure
			if value != nil {
				t.Fatal(value)
			}
		} else {
			ack, err = service.RemoveImageMember(ctx, resource.ID("image"), "member")
		}
		var proof *resource.ResponseError
		if !errors.As(err, &proof) || string(proof.Body) != "partial" || proof.Header.Get("X-Actual") != "failed" || calls != 1 || retries != 0 || body.closes != 1 {
			t.Fatal(operation, ack, err, proof, calls, retries, body.closes)
		}
		for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled} {
			if !errors.Is(err, cause) {
				t.Fatal("lost cause", operation, cause, err)
			}
		}
		if operation == "remove204" {
			if ack == nil || ack.StatusCode != 204 || string(ack.Body) != "partial" {
				t.Fatal(ack)
			}
			proof.Body[0] = 'X'
			proof.Header.Set("X-Actual", "changed")
			if string(ack.Body) != "partial" || ack.Header.Get("X-Actual") != "failed" {
				t.Fatal("ack/proof alias")
			}
		} else if ack != nil {
			t.Fatal(ack)
		}
		cancel(nil)
	}
	for _, operation := range []string{"find", "remove"} {
		for _, strict := range []bool{false, true} {
			retries := 0
			stop := errors.New("stop retry")
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
				return deleteCoreHTTP(404, io.NopCloser(strings.NewReader("missing")), nil), nil
			})
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return stop
			}
			var err error
			if operation == "find" {
				value, failure := New(client).FindImageMember(context.Background(), resource.ID("image"), "member", WithFindImageMemberIgnoreMissing(!strict))
				err = failure
				if value != nil {
					t.Fatal(value)
				}
			} else {
				value, failure := New(client).RemoveImageMember(context.Background(), resource.ID("image"), "member", WithRemoveImageMemberIgnoreMissing(!strict))
				err = failure
				if value != nil {
					t.Fatal(value)
				}
			}
			if strict {
				if !errors.Is(err, stop) || retries != 1 || !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(operation, err, retries)
				}
			} else if err != nil || retries != 0 {
				t.Fatal(operation, err, retries)
			}
		}
	}
}

func TestImageMembersCoreCapturedSourceAndRequestOwnership(t *testing.T) {
	var client *gophercloud.ServiceClient
	var retained *ImageMemberOpts
	client = deleteCoreClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.EscapedPath() != "/reverse/glance/v2/images/image/members/member" || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-A") != "owned" || r.Header.Get("X-Auth-Token") != "after" {
			t.Fatal(r.URL, r.Header)
		}
		retained.Headers["X-A"] = "late"
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{}`)), nil), nil
	})
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	value, err := New(client).GetImageMember(context.Background(), resource.ID("image"), "member", func(config *ImageMemberOpts) error {
		retained = config
		config.Headers["X-A"] = "owned"
		client.MoreHeaders["X-Snapshot"] = "changed"
		client.ResourceBase = "https://example.test/later/"
		client.ProviderClient.SetToken("after")
		return nil
	})
	if value == nil || err != nil {
		t.Fatal(value, err)
	}
	for _, mode := range []string{"provider-before", "provider-between-rows", "rawbody-retry", "expanded-status"} {
		calls := 0
		client = deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			if mode == "provider-between-rows" {
				return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"members":[{}, {"member_id":7}]}`)), nil), nil
			}
			if calls == 1 {
				return deleteCoreHTTP(503, io.NopCloser(strings.NewReader("original")), nil), nil
			}
			return deleteCoreHTTP(202, io.NopCloser(strings.NewReader("unexpected")), http.Header{"X-Actual": {"202"}}), nil
		})
		service := New(client)
		if mode == "provider-before" {
			_, err = service.GetImageMember(context.Background(), resource.ID("image"), "member", func(*ImageMemberOpts) error { client.ProviderClient = &gophercloud.ProviderClient{}; return nil })
			if calls != 0 || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(mode, err, calls)
			}
			continue
		}
		if mode == "provider-between-rows" {
			rows := 0
			for row, failure := range service.ListImageMembers(context.Background(), resource.ID("image")) {
				if failure == nil {
					rows++
					client.ProviderClient = &gophercloud.ProviderClient{}
				} else {
					var proof *resource.ResponseError
					if row != nil || !errors.Is(failure, resource.ErrInvalidOption) || !errors.As(failure, &proof) || !strings.Contains(string(proof.Body), `"member_id":7`) {
						t.Fatal(row, failure)
					}
				}
			}
			if rows != 1 || calls != 1 {
				t.Fatal(rows, calls)
			}
			continue
		}
		client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			if mode == "rawbody-retry" {
				opts.RawBody = strings.NewReader("replaced")
			} else {
				opts.OkCodes = append(opts.OkCodes, 202)
			}
			return nil
		}
		value, err = service.AddImageMember(context.Background(), resource.ID("image"), "member")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || err == nil || !errors.As(err, &native) {
			t.Fatal(mode, value, err)
		}
		if mode == "rawbody-retry" {
			if calls != 1 || native.Actual != 503 || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err, calls)
			}
		} else if calls != 2 || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "unexpected" {
			t.Fatal(native, calls)
		}
	}
}
