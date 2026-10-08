package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestFindImageRecordDefaultHiddenPhases(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, explicitName := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/nameOverride=%t", code, explicitName), func(t *testing.T) {
				selected := "literal/空 白%?#"
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodGet || req.Body != nil {
						t.Fatal(req.Method, req.Body)
					}
					switch calls {
					case 1:
						if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(selected) || req.URL.RawQuery != "" {
							t.Fatal(req.URL)
						}
						return taskCoreJSON(req, code, `{"message":"direct missing"}`), nil
					case 2:
						want := selected
						if explicitName {
							want = "caller name"
						}
						if req.URL.Query().Get("name") != want || req.URL.Query().Has("os_hidden") {
							t.Fatal("visible query", req.URL)
						}
						return taskCoreJSON(req, 203, `{"images":[{"id":"other","name":"other"}],"next":null}`), nil
					case 3:
						q := req.URL.Query()
						if q.Get("os_hidden") != "True" || !explicitName && q.Has("name") || explicitName && q.Get("name") != "caller name" {
							t.Fatal("hidden query", req.URL)
						}
						name, _ := json.Marshal(selected)
						return taskCoreJSON(req, 206, `{"images":[{"id":"passive","name":`+string(name)+`,"extension":{"precise":9007199254740993}}],"next":null}`), nil
					default:
						t.Fatal("unexpected request", req.URL)
						return nil, nil
					}
				})
				opts := []FindImageRecordOption{}
				if explicitName {
					opts = append(opts, WithFindImageRecordListOptions(WithImageRecordListFilter("name", "caller name")))
				}
				got, err := New(client).FindImageRecord(context.Background(), selected, opts...)
				if err != nil || got == nil || calls != 3 || imageRecordText(got, "id") != "passive" || got.StatusCode != 206 || len(got.Resource.Body) != 65 {
					t.Fatal(got, err, calls)
				}
				if string(got.Resource.Body["properties"]) != `{"extension":{"precise":9007199254740993}}` {
					t.Fatal(got.Resource.Body)
				}
			})
		}
	}
}
func TestFindImageRecordAcceptedGetAndAbsence(t *testing.T) {
	for _, test := range []struct {
		name, raw       string
		code            int
		strict, missing bool
	}{
		{"direct empty object", `{}`, 200, false, false},
		{"direct explicit null", `{"id":null}`, 399, false, false},
		{"direct malformed tolerated", `opaque`, 204, false, false},
		{"complete missing default", `{}`, 404, false, true},
		{"complete missing strict", `{}`, 403, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, test.code, test.raw), nil
				}
				return taskCoreJSON(req, 200, `{"images":[],"next":null}`), nil
			})
			opts := []FindImageRecordOption{}
			if test.strict {
				opts = append(opts, WithFindImageRecordIgnoreMissing(false))
			}
			got, err := New(client).FindImageRecord(context.Background(), "selected", opts...)
			if test.missing {
				if calls != 3 || got != nil || test.strict && !errors.Is(err, resource.ErrNotFound) || !test.strict && err != nil {
					t.Fatal(got, err, calls)
				}
			} else {
				if calls != 1 || got == nil || err != nil {
					t.Fatal(got, err, calls)
				}
				want := "{}"
				if test.raw == "opaque" {
					want = "null"
				}
				if string(got.Resource.Body["properties"]) != want {
					t.Fatal(got.Resource.Body)
				}
			}
		})
	}
}
func TestFindImageRecordErrorsNeverBecomeHiddenSearch(t *testing.T) {
	for _, mode := range []string{"duplicate", "lateHTTP", "lateModel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 404, `{}`), nil
				}
				if calls == 2 {
					if mode == "duplicate" {
						return taskCoreJSON(req, 200, `{"images":[{"id":"first","name":"selected"},{"id":"second","name":"selected"}]}`), nil
					}
					return taskCoreJSON(req, 200, `{"images":[{"id":"first","name":"selected"}],"next":"/v2/images?marker=tail"}`), nil
				}
				if calls != 3 || req.URL.Query().Get("marker") != "tail" || req.URL.Query().Get("name") != "selected" || req.URL.Query().Has("os_hidden") {
					t.Fatal("late page", req.URL)
				}
				if mode == "lateHTTP" {
					return taskCoreJSON(req, 503, `{"message":"late failure"}`), nil
				}
				return taskCoreJSON(req, 200, `{"images":[{"instance_type_rxtx_factor":"bad float"}]}`), nil
			})
			got, err := New(client).FindImageRecord(context.Background(), "selected")
			wantCalls := 3
			if mode == "duplicate" {
				wantCalls = 2
			}
			if got != nil || err == nil || calls != wantCalls {
				t.Fatal(got, err, calls)
			}
			if mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if mode == "lateHTTP" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"message":"late failure"}` || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
					t.Fatal("native rejected-page evidence", err, native)
				}
			}
		})
	}
}
func TestFindImageRecordRejectedOwnershipFaultsAreTerminal(t *testing.T) {
	for _, mode := range []string{"read", "wrappedEOF", "close", "hook"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("ownership marker")
			calls := 0
			body := &taskCoreBody{reader: strings.NewReader(`{"message":"missing"}`)}
			if mode == "read" {
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: marker}
			}
			if mode == "wrappedEOF" {
				body.reader = &taskCoreReader{body: `{"message":"missing"}`, err: errors.Join(io.EOF, marker)}
			}
			if mode == "close" {
				body.closeErr = marker
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 404, body), nil })
			if mode == "hook" {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					return errors.Join(err, marker)
				}
			}
			got, err := New(client).FindImageRecord(context.Background(), "selected")
			if got != nil || !errors.Is(err, marker) || calls != 1 || body.closes != 1 {
				t.Fatal(got, err, calls, body.closes)
			}
		})
	}
}
func TestFindImageRecordOptionsAndLocationOwnAllPhases(t *testing.T) {
	cloud := "captured"
	locations, callbacks, calls := 0, 0, 0
	headers := map[string]string{"X-Option": "snapshot"}
	ignore := false
	option := WithFindImageRecordOpts(FindImageRecordOpts{Headers: headers, IgnoreMissing: &ignore})
	headers["X-Option"] = "changed"
	ignore = true
	var retained *FindImageRecordOpts
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "before" || req.Header.Get("X-Option") != "snapshot" || req.Header.Get("X-Auth-Token") != "live" {
			t.Fatal(req.Header)
		}
		retained.Headers["X-Option"] = "retained"
		*retained.IgnoreMissing = true
		if calls == 1 {
			return taskCoreJSON(req, 403, `{}`), nil
		}
		if req.URL.Query().Get("limit") != "2" {
			t.Fatal(req.URL)
		}
		return taskCoreJSON(req, 200, `{"images":[]}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "later"
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	got, err := service.FindImageRecord(context.Background(), "selected", option, func(v *FindImageRecordOpts) error {
		callbacks++
		retained = v
		cloud = "changed"
		client.SetToken("live")
		return nil
	}, WithFindImageRecordListOptions(WithImageRecordListLimit(2)))
	if got != nil || !errors.Is(err, resource.ErrNotFound) || calls != 3 || callbacks != 1 || locations != 1 {
		t.Fatal(got, err, calls, callbacks, locations)
	}
}
func TestFindImageRecordNestedCallbackSourceChangeStopsBeforeHTTP(t *testing.T) {
	calls, callbacks := 0, 0
	client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
	got, err := New(client).FindImageRecord(context.Background(), "selected", WithFindImageRecordListOptions(
		func(*ImageRecordListOpts) error {
			callbacks++
			client.Endpoint = "https://changed.test/v2/"
			return nil
		},
		func(*ImageRecordListOpts) error { callbacks++; return nil },
	))
	if got != nil || err == nil || calls != 0 || callbacks != 1 {
		t.Fatal(got, err, calls, callbacks)
	}
}
