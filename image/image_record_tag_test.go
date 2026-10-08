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

func imageRecordTagInvoke(service *Service, remove bool, ctx context.Context, input ImageRecordTagRequest, tag string, options ...ImageMutationOption) (*ImageRecordTagResult, error) {
	if remove {
		return service.RemoveImageRecordTag(ctx, input, tag, options...)
	}
	return service.AddImageRecordTag(ctx, input, tag, options...)
}

func TestImageRecordTagOwnedSeedListSemanticsAndReceipts(t *testing.T) {
	for _, test := range []struct {
		name, tags, tag, want string
		remove, missing       bool
	}{
		{"append duplicate", `["a","a"]`, "a", `["a","a","a"]`, false, false},
		{"remove first duplicate", `["a","b","a"]`, "a", `["b","a"]`, true, false},
		{"absent removal is noop", `["b","b"]`, "a", `["b","b"]`, true, false},
		{"remove exact spelling only", `["A","a ","a","a"]`, "a", `["A","a ","a"]`, true, false},
		{"preserve untyped members", `[null,false,900719925474099312345,{"n":1.00000000000000000001},"a"]`, "a", `[null,false,900719925474099312345,{"n":1.00000000000000000001}]`, true, false},
		{"scalar string coerced before append", `"a"`, "a", `["a","a"]`, false, false},
		{"scalar string coerced before remove", `"a"`, "a", `[]`, true, false},
		{"scalar bool retained on append", `false`, "a", `[false,"a"]`, false, false},
		{"object coerced remains on nonmatch", `{"n":900719925474099312345}`, "a", `[{"n":900719925474099312345}]`, true, false},
		{"lone high surrogate is not replacement character", `["\uD800"]`, "\ufffd", `["\uD800"]`, true, false},
		{"lone low surrogate is not replacement character", `["\uDC00"]`, "\ufffd", `["\uDC00"]`, true, false},
		{"replacement character removes actual replacement only", `["\uD800","\ufffd","\uDC00"]`, "\ufffd", `["\uD800","\uDC00"]`, true, false},
		{"valid surrogate pair matches Unicode scalar", `["\uD83D\uDE00","keep"]`, "😀", `["keep"]`, true, false},
		{"escaped backslash is literal text", `["\\uD800","keep"]`, `\uD800`, `["keep"]`, true, false},
		{"missing default append", `[]`, "a", `["a"]`, false, true},
		{"missing default remove", `[]`, "a", `[]`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
			seed.Resource.Body["tags"] = json.RawMessage(test.tags)
			seed.Resource.Body["location"] = json.RawMessage(`{"cloud":"seed location"}`)
			if test.missing {
				delete(seed.Resource.Body, "tags")
			}
			before := imageRecordValues(seed.Resource)
			calls := 0
			var wireHeader http.Header
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				wantMethod := http.MethodPut
				if test.remove {
					wantMethod = http.MethodDelete
				}
				if req.Method != wantMethod || req.URL.EscapedPath() != "/reverse/glance/v2/images/original/tags/"+url.PathEscape(test.tag) || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("fixed single mutation", req.Method, req.URL, req.Body)
				}
				response := taskCoreJSON(req, 203, "opaque acknowledgement")
				wireHeader = response.Header
				return response, nil
			})
			got, err := imageRecordTagInvoke(New(client), test.remove, context.Background(), ImageRecordTagRequest{Record: seed}, test.tag)
			if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 1 || got.Record == seed || got.Record.Resource == seed.Resource || got.Record.Wire == seed.Wire {
				t.Fatal(got, err, calls)
			}
			ack := got.Acknowledgement
			if ack.ImageID != "original" || ack.Tag != test.tag || ack.StatusCode != 203 || string(ack.Body) != "opaque acknowledgement" || ack.Header.Get("X-Task-Proof") != "actual" {
				t.Fatal(ack)
			}
			want := imageRecordValues(seed.Resource)
			want["tags"] = test.want
			th.CheckDeepEquals(t, want, imageRecordValues(got.Record.Resource))
			th.CheckDeepEquals(t, before, imageRecordValues(seed.Resource))
			if got.Record.StatusCode != seed.StatusCode || !reflect.DeepEqual(got.Record.Header, seed.Header) || !reflect.DeepEqual(got.Record.Resource.Header, seed.Resource.Header) || !reflect.DeepEqual(got.Record.Wire, seed.Wire) || string(got.Record.Envelope) != string(seed.Envelope) || !reflect.DeepEqual(got.Record.ImportMethods, seed.ImportMethods) {
				t.Fatal("mutation replaced fetch receipt", got.Record, seed)
			}
			got.Record.Resource.Body["tags"][0] = '!'
			got.Record.Resource.Body["checksum"][1] = 'X'
			got.Record.Resource.Header.Set("X-Seed", "changed")
			got.Record.Wire.Body["wire_only"][0] = '0'
			got.Record.Wire.Header.Set("X-Seed", "changed")
			got.Record.Header.Set("X-Seed", "changed")
			got.Record.Envelope[0] = '!'
			got.Record.ImportMethods[0] = "changed"
			ack.Body[0] = '!'
			ack.Header.Set("X-Task-Proof", "changed")
			if string(seed.Resource.Body["checksum"]) != `"seed checksum"` || string(seed.Wire.Body["wire_only"]) != "900719925474099312345" || seed.Header.Get("X-Seed") != "original receipt" || string(seed.Envelope) != `{"seed":900719925474099312345}` || seed.ImportMethods[0] != "seed method" || wireHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal("owned channels alias", seed, ack)
			}
		})
	}
}

func TestImageRecordTagLiteralConstructorBoundsAndCurrentLocation(t *testing.T) {
	for _, tag := range []string{" ", " a /b\\c %2F?#한 ", strings.Repeat("界", 256), "é", "e\u0301"} {
		for _, remove := range []bool{false, true} {
			t.Run(fmt.Sprintf("tag=%q/remove=%v", tag, remove), func(t *testing.T) {
				const id = "a /한:%?\\b"
				cloud := "captured"
				locations, calls := 0, 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					method := http.MethodPut
					if remove {
						method = http.MethodDelete
					}
					if req.Method != method || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/tags/"+url.PathEscape(tag) || req.URL.RawQuery != "" || req.Body != nil {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 204, "opaque"), nil
				})
				service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
					locations++
					return resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}, nil
				}})
				got, err := imageRecordTagInvoke(service, remove, context.Background(), ImageRecordTagRequest{ID: id}, tag)
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 1 || locations != 1 {
					t.Fatal(got, err, calls, locations)
				}
				value := got.Record
				want := imageRecordDefaults("null")
				encodedID, _ := json.Marshal(id)
				want["id"] = string(encodedID)
				want["location"] = string(value.Resource.Body["location"])
				want["tags"] = "[]"
				if !remove {
					encodedTag, _ := json.Marshal(tag)
					want["tags"] = "[" + string(encodedTag) + "]"
				}
				th.CheckDeepEquals(t, want, imageRecordValues(value.Resource))
				if value.Wire != nil || len(value.Envelope) != 0 || len(value.Header) != 0 || value.StatusCode != 0 || value.Resource.StatusCode != 0 || len(value.Resource.Header) != 0 || value.ImportMethods == nil || len(value.ImportMethods) != 0 {
					t.Fatal("constructor fabricated fetched receipt", value)
				}
				var location resource.CloudLocation
				if err := json.Unmarshal(value.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"token project"` {
					t.Fatal(location, err)
				}
				if got.Acknowledgement.ImageID != id || got.Acknowledgement.Tag != tag {
					t.Fatal(got.Acknowledgement)
				}
			})
		}
	}
}

func TestImageRecordTagAcceptsOpaque200Through399(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299, 300, 399} {
		for _, remove := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/remove=%v", code, remove), func(t *testing.T) {
				raw := []byte{'o', 'p', 'a', 'q', 'u', 'e', 0xff}
				calls, retries := 0, 0
				body := &taskCoreBody{reader: bytes.NewReader(raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				got, err := imageRecordTagInvoke(New(client), remove, context.Background(), ImageRecordTagRequest{ID: "original"}, "tag")
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StatusCode != code || !bytes.Equal(got.Acknowledgement.Body, raw) || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, retries, body.closes)
				}
			})
		}
	}
}

func TestImageRecordTagNullLocalListFailsAfterAcknowledgement(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`{"broken":`), json.RawMessage([]byte{'"', 0xff, '"'})} {
			t.Run(fmt.Sprintf("remove=%v/raw=%q", remove, raw), func(t *testing.T) {
				seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
				seed.Resource.Body["tags"] = append(json.RawMessage(nil), raw...)
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreJSON(req, 201, "already acknowledged"), nil
				})
				got, err := imageRecordTagInvoke(New(client), remove, context.Background(), ImageRecordTagRequest{Record: seed}, "tag")
				if got == nil || got.Record != nil || got.Acknowledgement == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || got.Acknowledgement.StatusCode != 201 || string(got.Acknowledgement.Body) != "already acknowledged" {
					t.Fatal(got, err, calls)
				}
				taskCoreProof(t, err, 201, "already acknowledged")
				if !bytes.Equal(raw, seed.Resource.Body["tags"]) {
					t.Fatal("failed descriptor mutated seed", seed)
				}
			})
		}
	}
	t.Run("unrelated raw fields remain passive", func(t *testing.T) {
		seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
		seed.Resource.Body["checksum"] = json.RawMessage(`{"broken":`)
		seed.Wire.Body["wire_only"] = json.RawMessage([]byte{0xff})
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 204, "ack"), nil })
		got, err := New(client).AddImageRecordTag(context.Background(), ImageRecordTagRequest{Record: seed}, "tag")
		if got == nil || got.Record == nil || err != nil || string(got.Record.Resource.Body["checksum"]) != `{"broken":` || !bytes.Equal(got.Record.Wire.Body["wire_only"], []byte{0xff}) || string(got.Record.Resource.Body["tags"]) != `["tag"]` {
			t.Fatal(got, err)
		}
	})
}

func TestImageRecordTagSourceHeaderSeedAndOptionSnapshots(t *testing.T) {
	seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
	seed.Resource.Body["tags"] = json.RawMessage(`["before"]`)
	seed.Resource.Body["location"] = json.RawMessage(`{"cloud":"seed location"}`)
	cloud := "current"
	headers := map[string]string{"X-Option": "factory snapshot"}
	option := WithImageMutationOpts(ImageMutationOpts{Headers: headers})
	headers["X-Option"] = "caller changed"
	calls, locations, callbacks := 0, 0, 0
	var retained *ImageMutationOpts
	var client *gophercloud.ServiceClient
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPut || req.URL.Path != "/reverse/glance/v2/images/original/tags/after" || req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory snapshot" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		retained.Headers["X-Option"] = "retained changed"
		return taskCoreJSON(req, 204, "opaque"), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	client.Microversion = "2.10"
	options := []ImageMutationOption{option, func(value *ImageMutationOpts) error {
		callbacks++
		retained = value
		client.SetToken("live")
		return WithImageMutationHeader("X-Final", "yes")(value)
	}}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "getter changed"
		seed.Resource.Body["id"][1] = 'X'
		seed.Resource.Body["tags"][2] = 'X'
		options[0] = nil
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	got, err := service.AddImageRecordTag(context.Background(), ImageRecordTagRequest{Record: seed}, "after", options...)
	if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 1 || locations != 1 || callbacks != 1 || string(got.Record.Resource.Body["tags"]) != `["before","after"]` || string(got.Record.Resource.Body["location"]) != `{"cloud":"seed location"}` || got.Acknowledgement.ImageID != "original" {
		t.Fatal(got, err, calls, locations, callbacks)
	}
}

func TestImageRecordTagAcceptedReadCloseAndStickyGuardAcknowledgements(t *testing.T) {
	const raw = "acknowledged partial body"
	for _, remove := range []bool{false, true} {
		for _, mode := range []string{"read", "close", "cancel", "source restored on Close", "outer restored on Close"} {
			t.Run(fmt.Sprintf("remove=%v/%s", remove, mode), func(t *testing.T) {
				marker := errors.New("tag acknowledgement body failure")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				invalidOuter := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if invalidOuter {
						return marker
					}
					return nil
				})
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				action := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = marker
				case "cancel":
					action = func() { cancel(marker) }
				case "source restored on Close":
					action = func() { client.Endpoint = "https://foreign.test/" }
				case "outer restored on Close":
					action = func() { invalidOuter = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
				}
				selected := io.ReadCloser(body)
				if strings.Contains(mode, "restored") {
					selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; invalidOuter = false }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, selected), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
				got, err := imageRecordTagInvoke(New(client), remove, ctx, ImageRecordTagRequest{Record: seed}, "tag")
				if got == nil || got.Record != nil || got.Acknowledgement == nil || err == nil || got.Acknowledgement.StatusCode != 201 || string(got.Acknowledgement.Body) != raw || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, retries, body.closes)
				}
				if mode == "source restored on Close" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal(err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				proof := taskCoreProof(t, err, 201, raw)
				proof.Body[0] = '!'
				proof.Header.Set("X-Task-Proof", "proof changed")
				if string(got.Acknowledgement.Body) != raw || got.Acknowledgement.Header.Get("X-Task-Proof") != "actual" || string(seed.Resource.Body["tags"]) != "[]" {
					t.Fatal("proof aliases ack or mutated seed", got, seed)
				}
			})
		}
	}
}

func TestImageRecordTagNativeRejectionsRetryAndLegacy204(t *testing.T) {
	for _, code := range []int{400, 403, 404, 500, 503} {
		for _, remove := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/remove=%v", code, remove), func(t *testing.T) {
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreJSON(req, code, "native tag rejection"), nil
				})
				got, err := imageRecordTagInvoke(New(client), remove, context.Background(), ImageRecordTagRequest{ID: "original"}, "tag")
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native tag rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 1 {
					t.Fatal(got, err, native, calls)
				}
			})
		}
	}
	t.Run("native RetryFunc remains authoritative", func(t *testing.T) {
		calls, hooks := 0, 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 503, "retry original"), nil
			}
			return taskCoreJSON(req, 204, "final ack"), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			return nil
		}
		seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
		seed.Resource.Body["tags"] = json.RawMessage(`["a"]`)
		got, err := New(client).AddImageRecordTag(context.Background(), ImageRecordTagRequest{Record: seed}, "a")
		if got == nil || got.Record == nil || err != nil || calls != 2 || hooks != 1 || string(got.Record.Resource.Body["tags"]) != `["a","a"]` || string(got.Acknowledgement.Body) != "final ack" {
			t.Fatal(got, err, calls, hooks)
		}
	})
	t.Run("legacy strict204 remains", func(t *testing.T) {
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 201, "opaque"), nil })
		if got, err := New(client).AddTag(context.Background(), resource.ID("original"), "tag"); got != nil || err == nil {
			t.Fatal(got, err)
		}
		client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
			t.Fatal("legacy slash tag reached HTTP")
			return nil, nil
		})
		if got, err := New(client).RemoveTag(context.Background(), resource.ID("original"), "a/b"); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(got, err)
		}
	})
}

func TestImageRecordTagPreflightAndGuardedCallbacks(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   ImageRecordTagRequest
		tag     string
		options []ImageMutationOption
	}{
		{"missing identity", ImageRecordTagRequest{}, "tag", nil},
		{"ambiguous identity", ImageRecordTagRequest{ID: "original", Record: imageRecordWaitSeed(json.RawMessage(`"pending"`))}, "tag", nil},
		{"missing Resource", ImageRecordTagRequest{Record: &ImageRecord{}}, "tag", nil},
		{"untyped record id", ImageRecordTagRequest{Record: &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`false`)}}}}}, "tag", nil},
		{"unsafe image identity", ImageRecordTagRequest{ID: "bad\n"}, "tag", nil},
		{"empty tag", ImageRecordTagRequest{ID: "original"}, "", nil}, {"dot tag", ImageRecordTagRequest{ID: "original"}, ".", nil}, {"dotdot tag", ImageRecordTagRequest{ID: "original"}, "..", nil}, {"control tag", ImageRecordTagRequest{ID: "original"}, "bad\n", nil}, {"invalid UTF8 tag", ImageRecordTagRequest{ID: "original"}, string([]byte{0xff}), nil},
		{"nil option", ImageRecordTagRequest{ID: "original"}, "tag", []ImageMutationOption{nil}},
		{"owned auth header", ImageRecordTagRequest{ID: "original"}, "tag", []ImageMutationOption{WithImageMutationHeader("X-Auth-Token", "foreign")}},
		{"header newline", ImageRecordTagRequest{ID: "original"}, "tag", []ImageMutationOption{WithImageMutationHeaders(map[string]string{"X-Extra": "\n"})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			got, err := New(client).AddImageRecordTag(context.Background(), test.input, test.tag, test.options...)
			if got != nil || err == nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		calls, callbacks := 0, 0
		service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }))
		got, err := service.RemoveImageRecordTag(ctx, ImageRecordTagRequest{ID: "original"}, "tag", func(*ImageMutationOpts) error { callbacks++; return nil })
		if got != nil || err == nil || calls != 0 || callbacks != 0 {
			t.Fatal(got, err, calls, callbacks)
		}
	}
	var nilService *Service
	if got, err := nilService.AddImageRecordTag(context.Background(), ImageRecordTagRequest{ID: "original"}, "tag"); got != nil || err == nil {
		t.Fatal(got, err)
	}
	for _, mode := range []string{"cancel", "source drift", "service binding", "outer drift"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("tag option cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			invalidOuter := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if invalidOuter {
					return marker
				}
				return nil
			})
			calls, later := 0, 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			service := New(client)
			got, err := service.AddImageRecordTag(ctx, ImageRecordTagRequest{ID: "original"}, "tag", func(*ImageMutationOpts) error {
				switch mode {
				case "cancel":
					cancel(marker)
				case "source drift":
					client.ResourceBase = "https://foreign.test/"
				case "service binding":
					service.API = nil
				case "outer drift":
					invalidOuter = true
				}
				return nil
			}, func(*ImageMutationOpts) error { later++; return nil })
			if got != nil || err == nil || calls != 0 || later != 0 {
				t.Fatal(got, err, calls, later)
			}
			if mode == "cancel" || mode == "outer drift" {
				if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordTagRejectedPhysicalFaults(t *testing.T) {
	// The owned profile observes rejected bodies through the shared REST wrapper;
	// faults retain the original native HTTP evidence and never establish an ACK.
	const raw = `{"message":"physical rejection"}`
	for _, code := range []int{400, 404, 503} {
		for _, mode := range []string{"read", "close", "cancel", "binding"} {
			t.Run(fmt.Sprintf("%d/%s", code, mode), func(t *testing.T) {
				marker := errors.New("rejected tag physical failure")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls := 0
				var service *Service
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = marker
				case "cancel":
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { cancel(marker) }}
				case "binding":
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { service.API = nil }}
				}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil })
				service = New(client)
				seed := imageRecordWaitSeed(json.RawMessage(`"pending"`))
				got, err := service.RemoveImageRecordTag(ctx, ImageRecordTagRequest{Record: seed}, "tag")
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || err == nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodDelete || native.URL != "https://glance.example/reverse/glance/v2/images/original/tags/tag" || string(native.Body) != raw || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 1 || body.closes != 1 {
					t.Fatal(got, err, native, calls, body.closes)
				}
				if mode == "binding" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal("physical cause was discarded", err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if errors.Is(err, resource.ErrNotFound) {
					t.Fatal("rejected tag converted to logical absence", err)
				}
				th.AssertEquals(t, "[]", string(seed.Resource.Body["tags"]))
			})
		}
	}
}
