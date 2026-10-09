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

func imageRecordActionCall(service *Service, action string, ctx context.Context, input ImageRecordActionRequest, options ...ImageRecordActionOption) (*ImageRecordActionResult, error) {
	if action == "reactivate" {
		return service.ReactivateImageRecord(ctx, input, options...)
	}
	return service.DeactivateImageRecord(ctx, input, options...)
}

func TestImageRecordActionsLiteralRoutesAndOpaqueAcceptedStatuses(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		for _, test := range []struct {
			code int
			raw  string
		}{
			{200, `null`}, {201, `[]`}, {204, ""}, {299, `{"id":"foreign","status":"inferred decoy"}`}, {300, "not JSON"}, {304, `false`}, {399, "opaque\xff"},
		} {
			t.Run(fmt.Sprintf("%s/%d", action, test.code), func(t *testing.T) {
				const id = "name-like /한:%?\\b"
				calls, locations, retries := 0, 0, 0
				cloud := "action location"
				body := &taskCoreBody{reader: strings.NewReader(test.raw)}
				var actualHeader http.Header
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/actions/"+action || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "first-token" {
						t.Fatal("action must send one fixed bodyless POST", req.Method, req.URL, req.Body, req.Header)
					}
					reply := taskCoreHTTP(req, test.code, body)
					reply.Header.Set("OpenStack-image-import-methods", "ACK methods stay passive")
					actualHeader = reply.Header
					return reply, nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
				got, err := imageRecordActionCall(service, action, context.Background(), ImageRecordActionRequest{ID: id})
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 1 || locations != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, locations, retries, body.closes)
				}
				ack, record := got.Acknowledgement, got.Record
				if ack.ImageID != id || ack.Action != action || ack.StatusCode != test.code || string(ack.Body) != test.raw || ack.Header.Get("X-Task-Proof") != "actual" || len(record.Resource.Body) != 65 || len(record.ImportMethods) != 0 || record.Resource.StatusCode != 0 || record.StatusCode != 0 || record.Wire != nil || record.Envelope != nil || len(record.Header) != 0 || string(record.Resource.Body["status"]) != "null" {
					t.Fatal("opaque action receipt overwrote local model or invented fetch/status", got)
				}
				th.AssertEquals(t, id, taskCoreText(t, record.Resource.Body["id"]))
				var location resource.CloudLocation
				if err := json.Unmarshal(record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != cloud {
					t.Fatal(location, err)
				}
				ack.Header.Set("X-Task-Proof", "changed")
				if len(ack.Body) != 0 {
					ack.Body[0] = '!'
				}
				if actualHeader.Get("X-Task-Proof") != "actual" || len(record.Header) != 0 {
					t.Fatal("ack channels alias response or model", got)
				}
			})
		}
	}
}

func TestImageRecordActionsProjectPrivateCurrentBodyAndPreserveFetchReceipts(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		t.Run(action, func(t *testing.T) {
			const fetched = `{"id":"fixed","status":"ACTIVE","protected":"false","size":"04","tags":"one","vendor":{"precise":900719925474099312345}}`
			var handler taskCoreTransport
			calls, locations := 0, 0
			cloud := "fetch location"
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
			seed := imageRecordUpdateFetched(t, service, fetched, &handler)
			seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
			seed.Resource.Body["size"] = json.RawMessage(`"public invalid descriptor"`)
			seed.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
			created, updated := "passive created", "passive updated"
			seed.Resource.CreatedAt, seed.Resource.UpdatedAt = &created, &updated
			seed.Resource.Links = []resource.Link{{Href: "https://foreign.test/passive", Rel: "self"}}
			before := cloneImageRecord(seed)
			cloud = "action location"
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/actions/"+action || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal(req.Method, req.URL, req.Body)
				}
				reply := taskCoreJSON(req, 299, `{"id":"ACK decoy","status":"DEACTIVATED","size":null}`)
				reply.Header.Set("OpenStack-image-import-methods", "must not restore")
				return reply, nil
			}
			got, err := imageRecordActionCall(service, action, context.Background(), ImageRecordActionRequest{Record: seed})
			if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 2 || locations != 2 || got.Record == seed || got.Record.Resource == seed.Resource || got.Record.Wire == seed.Wire {
				t.Fatal(got, err, calls, locations)
			}
			value := got.Record
			for key, want := range map[string]string{"id": `"fixed"`, "status": `"ACTIVE"`, "size": "4", "is_protected": "true", "tags": `["one"]`} {
				th.AssertEquals(t, want, string(value.Resource.Body[key]))
			}
			if value.StatusCode != 203 || value.Resource.StatusCode != 203 || !reflect.DeepEqual(value.Header, seed.Header) || !reflect.DeepEqual(value.Wire, seed.Wire) || string(value.Envelope) != fetched || !reflect.DeepEqual(value.bodyState, seed.bodyState) || len(value.ImportMethods) != 0 || !reflect.DeepEqual(before, seed) {
				t.Fatal("action changed raw baseline, prior receipt, import reset or input", value, seed)
			}
			if value.Resource.CreatedAt == nil || value.Resource.UpdatedAt == nil || value.Resource.CreatedAt == seed.Resource.CreatedAt || value.Resource.UpdatedAt == seed.Resource.UpdatedAt || *value.Resource.CreatedAt != created || *value.Resource.UpdatedAt != updated || !reflect.DeepEqual(value.Resource.Links, seed.Resource.Links) {
				t.Fatal("passive metadata lost or aliased", value.Resource)
			}
			var location resource.CloudLocation
			if err := json.Unmarshal(value.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "action location" {
				t.Fatal(location, err)
			}
			value.bodyState.current["size"][1] = 'X'
			value.bodyState.original["size"][1] = 'X'
			value.Wire.Body["id"][1] = 'X'
			value.Header.Set("X-Task-Proof", "changed")
			value.Resource.Header.Set("X-Task-Proof", "changed")
			value.Envelope[0] = '!'
			*value.Resource.CreatedAt, *value.Resource.UpdatedAt = "changed", "changed"
			value.Resource.Links[0].Href = "changed"
			if !reflect.DeepEqual(before, seed) {
				t.Fatal("action output aliases input", seed)
			}
		})
	}
}

func TestImageRecordActionsKeepPendingRawChangesForExplicitCommit(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		t.Run(action, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				switch calls {
				case 1, 3:
					if req.Method != http.MethodPatch {
						t.Fatal("action cleaned pending body", req.Method)
					}
					imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"pending"}]`)
					if calls == 1 {
						return taskCoreJSON(req, 299, "pending opaque PATCH response"), nil
					}
					return taskCoreJSON(req, 200, `{}`), nil
				case 2:
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/actions/"+action || req.Body != nil {
						t.Fatal(req.Method, req.URL, req.Body)
					}
					return taskCoreJSON(req, 203, `{"name":"must not overlay","status":"must not infer"}`), nil
				default:
					t.Fatal("unexpected implicit replay", calls)
					return nil, nil
				}
			})
			service := New(client)
			pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "pending"}})
			if pending == nil || err != nil {
				t.Fatal(pending, err)
			}
			got, err := imageRecordActionCall(service, action, context.Background(), ImageRecordActionRequest{Record: pending})
			if got == nil || got.Record == nil || err != nil || got.Record.StatusCode != 299 || got.Record.Wire != nil || string(got.Record.Envelope) != "pending opaque PATCH response" || !reflect.DeepEqual(got.Record.bodyState, pending.bodyState) || string(got.Record.Resource.Body["name"]) != `"pending"` || string(got.Record.Resource.Body["status"]) != "null" {
				t.Fatal(got, err)
			}
			committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			if committed == nil || err != nil || calls != 3 {
				t.Fatal(committed, err, calls)
			}
		})
	}
}

func TestImageRecordActionsStrictRawUnicodeIdentity(t *testing.T) {
	for _, test := range []struct {
		name, action, raw, target string
		invalid                   bool
	}{
		{"high surrogate", "deactivate", `"\uD800"`, "", true},
		{"low surrogate", "reactivate", `"\uDC00"`, "", true},
		{"high followed by nonlow", "deactivate", `"\uD800\u0041"`, "", true},
		{"valid pair", "reactivate", `"\uD83D\uDE80"`, "🚀", false},
		{"actual replacement", "deactivate", `"�"`, "�", false},
		{"escaped replacement", "reactivate", `"\uFFFD"`, "�", false},
		{"literal backslash u", "deactivate", `"\\uD800"`, `\uD800`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 203, `{"id":`+test.raw+`}`), nil
				}
				if test.invalid || calls != 2 || req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(test.target)+"/actions/"+test.action || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("raw identity selected a wrong action target", calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 203, "opaque"), nil
			})
			service := New(client)
			seed, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
			if seed == nil || err != nil {
				t.Fatal(seed, err)
			}
			seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
			seed.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
			got, err := imageRecordActionCall(service, test.action, context.Background(), ImageRecordActionRequest{Record: seed})
			if test.invalid {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
					t.Fatal(got, err, calls)
				}
			} else {
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 2 || got.Acknowledgement.ImageID != test.target {
					t.Fatal(got, err, calls)
				}
				th.AssertEquals(t, test.target, taskCoreText(t, got.Record.Resource.Body["id"]))
			}
		})
	}
}

func TestImageRecordActionsRejectUnsafeSelectorsAndDescriptorInputsBeforeHTTP(t *testing.T) {
	cases := []struct {
		name    string
		input   ImageRecordActionRequest
		options []ImageRecordActionOption
	}{
		{"missing identity", ImageRecordActionRequest{}, nil},
		{"both forms", ImageRecordActionRequest{ID: "fixed", Record: &ImageRecord{}}, nil},
		{"empty Record", ImageRecordActionRequest{Record: &ImageRecord{}}, nil},
		{"handcrafted Resource without raw ownership", ImageRecordActionRequest{Record: &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"fixed"`)}}}}}, nil},
		{"blank literal", ImageRecordActionRequest{ID: " "}, nil},
		{"dot segment", ImageRecordActionRequest{ID: ".."}, nil},
		{"control in literal", ImageRecordActionRequest{ID: "bad\n"}, nil},
		{"invalid UTF8 literal", ImageRecordActionRequest{ID: string([]byte{0xff})}, nil},
		{"nil option", ImageRecordActionRequest{ID: "fixed"}, []ImageRecordActionOption{nil}},
		{"owned auth header", ImageRecordActionRequest{ID: "fixed"}, []ImageRecordActionOption{WithImageRecordActionHeader("X-Auth-Token", "foreign")}},
		{"invalid ordinary header", ImageRecordActionRequest{ID: "fixed"}, []ImageRecordActionOption{WithImageRecordActionHeaders(map[string]string{"X-Extra": "\n"})}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				t.Fatal("preflight performed HTTP", req.URL)
				return nil, nil
			}))
			got, err := imageRecordActionCall(service, "deactivate", context.Background(), test.input, test.options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, field := range []string{"is_hw_vif_multiqueue_enabled", "instance_type_rxtx_factor"} {
		t.Run("private descriptor/"+field, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			seed.bodyState.current[field] = json.RawMessage(`"invalid descriptor"`)
			before := cloneImageRecord(seed)
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("invalid private descriptor reached action", req.URL)
				return nil, nil
			}
			got, err := imageRecordActionCall(service, "reactivate", context.Background(), ImageRecordActionRequest{Record: seed})
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || !reflect.DeepEqual(before, seed) {
				t.Fatal(got, err, calls, seed)
			}
		})
	}
	for _, mode := range []string{"nil context", "canceled context", "nil service", "location cause"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("action preflight cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, callbacks := 0, 0
			service := NewWithDependencies(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }), Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				if mode == "location cause" {
					return resource.CloudLocation{}, marker
				}
				return resource.CloudLocation{}, nil
			}})
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled context":
				cancel(marker)
			case "nil service":
				service = nil
			}
			got, err := imageRecordActionCall(service, "reactivate", ctx, ImageRecordActionRequest{ID: "fixed"}, func(*ImageRecordActionOpts) error { callbacks++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatal(got, err, calls, callbacks)
			}
			if (mode == "canceled context" || mode == "location cause") && !errors.Is(err, marker) {
				t.Fatal("preflight cause lost", err)
			}
			if mode == "canceled context" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordActionsSnapshotInputsOptionsHeadersAndLocation(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		t.Run(action, func(t *testing.T) {
			var handler taskCoreTransport
			calls, callbacks, locations := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			seed := imageRecordUpdateFetched(t, New(client), `{"id":"fixed","name":"captured seed"}`, &handler)
			headers := map[string]string{"X-Option": "factory"}
			factory := WithImageRecordActionOpts(ImageRecordActionOpts{Headers: headers})
			headers["X-Option"] = "caller changed"
			cloud := "captured location"
			facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"captured project"`)}}
			var options []ImageRecordActionOption
			var retained *ImageRecordActionOpts
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/actions/"+action || req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				retained.Headers["X-Option"] = "retained changed"
				return taskCoreJSON(req, 203, "opaque"), nil
			}
			client.MoreHeaders, client.Microversion = map[string]string{"X-Source": "captured"}, "2.10"
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				seed.bodyState.current["id"] = json.RawMessage(`"location changed ID"`)
				seed.bodyState.current["name"] = json.RawMessage(`"location changed name"`)
				client.MoreHeaders["X-Source"] = "location changed"
				options[0] = WithImageRecordActionHeader("X-Option", "location changed option")
				return facts, nil
			}})
			options = []ImageRecordActionOption{factory, func(config *ImageRecordActionOpts) error {
				callbacks++
				retained = config
				cloud = "option changed location"
				facts.Project.ID[1] = 'X'
				client.SetToken("live")
				return WithImageRecordActionHeader("X-Final", "yes")(config)
			}}
			got, err := imageRecordActionCall(service, action, context.Background(), ImageRecordActionRequest{Record: seed}, options...)
			if got == nil || got.Record == nil || err != nil || calls != 2 || callbacks != 1 || locations != 1 || string(got.Record.Resource.Body["id"]) != `"fixed"` || string(got.Record.Resource.Body["name"]) != `"captured seed"` {
				t.Fatal(got, err, calls, callbacks, locations)
			}
			var location resource.CloudLocation
			if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured location" || string(location.Project.ID) != `"captured project"` {
				t.Fatal(location, err)
			}
		})
	}
	for _, mode := range []string{"full options replace", "bulk headers merge"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				wantDiscarded := ""
				if mode == "bulk headers merge" {
					wantDiscarded = "old"
				}
				if req.Header.Get("X-Discarded") != wantDiscarded || req.Header.Get("X-Retained") != "new" {
					t.Fatal("ordinary header helper semantics changed", req.Header)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			options := []ImageRecordActionOption{WithImageRecordActionHeader("X-Discarded", "old")}
			if mode == "full options replace" {
				options = append(options, WithImageRecordActionOpts(ImageRecordActionOpts{Headers: map[string]string{"X-Retained": "new"}}))
			} else {
				options = append(options, WithImageRecordActionHeaders(map[string]string{"X-Retained": "new"}))
			}
			got, err := imageRecordActionCall(New(client), "deactivate", context.Background(), ImageRecordActionRequest{ID: "fixed"}, options...)
			if got == nil || err != nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordActionsStopAfterGuardFailureInOptions(t *testing.T) {
	for _, mode := range []string{"cancel", "source", "binding", "outer", "callback cause"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("action option cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return marker
				}
				return nil
			})
			calls, callbacks, later := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			service := New(client)
			got, err := imageRecordActionCall(service, "reactivate", ctx, ImageRecordActionRequest{ID: "fixed"}, func(*ImageRecordActionOpts) error {
				callbacks++
				switch mode {
				case "cancel":
					cancel(marker)
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "binding":
					service.API = nil
				case "outer":
					outerInvalid = true
				case "callback cause":
					return marker
				}
				return nil
			}, func(*ImageRecordActionOpts) error { later++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "source" || mode == "binding" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal("option/guard cause lost", err)
			}
		})
	}
}

func TestImageRecordActionsAcceptedPhysicalFailuresReturnOnlyPartialAcknowledgement(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		for _, mode := range []string{"read", "close", "cancel", "source drift restored on Close", "outer drift restored on Close"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				const raw = "already accepted"
				marker := errors.New("action accepted handling cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				outerInvalid := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if outerInvalid {
						return marker
					}
					return nil
				})
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				beforeRead := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
				case "cancel":
					beforeRead = func() { cancel(marker) }
				case "source drift restored on Close":
					beforeRead = func() { client.Endpoint = "https://foreign.test/" }
				case "outer drift restored on Close":
					beforeRead = func() { outerInvalid = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: beforeRead}
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
				got, err := imageRecordActionCall(New(client), action, ctx, ImageRecordActionRequest{ID: "fixed"})
				if got == nil || got.Record != nil || got.Acknowledgement == nil || got.Acknowledgement.StatusCode != 201 || got.Acknowledgement.Action != action || string(got.Acknowledgement.Body) != raw || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, retries, body.closes)
				}
				if strings.HasPrefix(mode, "source") {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal("accepted failure cause lost", err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted close404 became missing", err)
				}
				taskCoreProof(t, err, 201, raw)
			})
		}
	}
}

func TestImageRecordActionsNativeRejectionsIncludeMissingAndNeverAcknowledge(t *testing.T) {
	for _, action := range []string{"deactivate", "reactivate"} {
		for _, code := range []int{400, 403, 404, 500, 599} {
			t.Run(fmt.Sprintf("%s/%d", action, code), func(t *testing.T) {
				calls, retries := 0, 0
				body := &taskCoreBody{reader: strings.NewReader("actual rejection")}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				got, err := imageRecordActionCall(New(client), action, context.Background(), ImageRecordActionRequest{ID: "fixed"})
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if got != nil || err == nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "actual rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &accepted) || calls != 1 || retries != 1 || body.closes != 1 || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("native status was swallowed or acknowledged", got, err, native, calls, retries, body.closes)
				}
			})
		}
	}
}

func TestImageRecordActionsNativeRetryPreservesFixedScopeLiveAuthAndOwnership(t *testing.T) {
	for _, mode := range []string{"successful retry", "expanded OkCodes rejection", "changed body ownership", "source change", "canceled retry", "hook cause", "clean final404 is still rejection"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("action retry cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/actions/deactivate" || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("retry moved action scope or body", req.Method, req.URL, req.Body)
				}
				if calls == 1 {
					return taskCoreJSON(req, 503, "initial rejection"), nil
				}
				if calls != 2 || req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("X-Retry") != "ordinary" {
					t.Fatal(calls, req.Header)
				}
				if mode == "expanded OkCodes rejection" {
					return taskCoreJSON(req, 418, "actual rejected status"), nil
				}
				if mode == "clean final404 is still rejection" {
					return taskCoreJSON(req, 404, "final missing"), nil
				}
				return taskCoreJSON(req, 203, "actual accepted status"), nil
			})
			client.RetryFunc = func(_ context.Context, method, endpoint string, options *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if method != http.MethodPost || endpoint != "https://glance.example/reverse/glance/v2/images/fixed/actions/deactivate" {
					t.Fatal(method, endpoint)
				}
				if count > 1 {
					return original
				}
				client.SetToken("retry token")
				options.MoreHeaders = map[string]string{"X-Retry": "ordinary"}
				switch mode {
				case "expanded OkCodes rejection":
					options.OkCodes = append(options.OkCodes, 418)
				case "changed body ownership":
					options.JSONBody = map[string]bool{"must_not_send": true}
				case "source change":
					client.Endpoint = "https://foreign.test/"
				case "canceled retry":
					cancel(marker)
				case "hook cause":
					return errors.Join(original, marker)
				}
				return nil
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			got, err := imageRecordActionCall(New(client), "deactivate", ctx, ImageRecordActionRequest{ID: "fixed"})
			wantCalls, wantRetries := 1, 1
			if mode == "successful retry" || mode == "expanded OkCodes rejection" || mode == "clean final404 is still rejection" {
				wantCalls = 2
			}
			if mode == "clean final404 is still rejection" {
				wantRetries = 2
			}
			if calls != wantCalls || retries != wantRetries || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(calls, retries)
			}
			if mode == "successful retry" {
				if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StatusCode != 203 || string(got.Acknowledgement.Body) != "actual accepted status" {
					t.Fatal(got, err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			wantCode, wantBody := 503, "initial rejection"
			if mode == "expanded OkCodes rejection" {
				wantCode, wantBody = 418, "actual rejected status"
			}
			if mode == "clean final404 is still rejection" {
				wantCode, wantBody = 404, "final missing"
			}
			if got != nil || err == nil || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal(got, err, native)
			}
			if (mode == "changed body ownership" || mode == "source change") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if (mode == "canceled retry" || mode == "hook cause") && !errors.Is(err, marker) {
				t.Fatal("retry cause lost", err)
			}
			if mode == "canceled retry" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
