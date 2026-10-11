package secrets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordListAPI(t *testing.T, reply func(*http.Request) (*http.Response, error)) (*secrets.API, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	client := nativeSecretClient(cloud)
	calls := &[]string{}
	cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		if req.URL.Path != "/barbican/v1/secrets" {
			t.Fatal("unexpected member/payload/discovery request", req.URL)
		}
		return reply(req)
	})
	return secrets.New(client), calls
}
func recordListField(t *testing.T, value *resource.RawResource, key, want string) {
	t.Helper()
	if value == nil || string(value.Body[key]) != want {
		t.Fatalf("field %s got %#v want %s", key, value, want)
	}
}
func TestPythonSecretListRecordsDefaultsAndOwnedResourceEvidence(t *testing.T) {
	for _, version := range []string{"", "1.1"} {
		t.Run("version-"+version, func(t *testing.T) {
			body := `{"secrets":[{"id":null,"secret_ref":"https://passive.example/v1/secrets/s1%2Fraw?x=1","name":false,"status":0,"algorithm":[false],"bit_length":9007199254740993,"content_types":false,"created":null,"updated":"unparsed","unknown":{"n":9007199254740993}}]}`
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Auth-Token") != "test-token" {
					t.Fatal(req)
				}
				wantVersion := ""
				if version != "" {
					wantVersion = "key-manager " + version
				}
				if req.Header.Get("OpenStack-API-Version") != wantVersion {
					t.Fatal(req.Header)
				}
				return nativeSecretWire(199, body), nil
			})
			api.RawClient().Microversion = version
			api.RawClient().MoreHeaders = map[string]string{"aCcEpT": "configured", "X-Source": "original"}
			values, err := api.AllRecords(context.Background())
			if err != nil || len(values) != 1 || len(*calls) != 1 {
				t.Fatal(values, err, *calls)
			}
			value := values[0]
			if value.RequestID != "" || value.StatusCode != 199 || value.Resource.StatusCode != 199 || value.Wire.StatusCode != 199 || len(value.Resource.Body) != 17 || string(value.Envelope) != body {
				t.Fatal(value)
			}
			recordListField(t, value.Resource, "id", `null`)
			recordListField(t, value.Resource, "secret_id", `"s1%2Fraw"`)
			recordListField(t, value.Resource, "bit_length", `9007199254740993`)
			recordListField(t, value.Resource, "content_types", `{}`)
			recordListField(t, value.Resource, "payload", `null`)
			recordListField(t, value.Resource, "name", `false`)
			recordListField(t, value.Resource, "expires_at", `null`)
			recordListField(t, value.Resource, "created_at", `null`)
			recordListField(t, value.Resource, "updated_at", `"unparsed"`)
			recordListField(t, value.Resource, "location", "null")
			if string(value.Wire.Body["unknown"]) != `{"n":9007199254740993}` || value.Resource.Body["unknown"] != nil {
				t.Fatal(value)
			}
			original := string(value.Wire.Body["secret_ref"])
			value.Resource.Body["secret_ref"][0] = 'x'
			value.Header.Set("X-Changed", "yes")
			value.Envelope[0] = 'x'
			if string(value.Wire.Body["secret_ref"]) != original || value.Wire.Header.Get("X-Changed") != "" || value.Resource.Header.Get("X-Changed") != "" || api.RawClient().MoreHeaders["aCcEpT"] != "configured" || api.RawClient().Microversion != version {
				t.Fatal("owned channels alias", value)
			}
			if value.Payload != nil {
				t.Fatal("list fetched payload", value)
			}
		})
	}
}
func TestPythonSecretListRecordsQueryClassificationAndRawCap(t *testing.T) {
	body := `{"secrets":[{"id":"first","name":"untrusted","bit_length":0,"status":"ACTIVE"},{"id":"second","name":"untrusted","bit_length":true,"status":"ACTIVE"}]}`
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeSecretWire(200, body), nil })
	query := map[string]json.RawMessage{"limit": json.RawMessage(`0`), "name": json.RawMessage(`["server",null,"second"]`), "algorithm": json.RawMessage(`null`), "alg": json.RawMessage(`"ignored"`), "acl_only": json.RawMessage(`false`), "bit_length": json.RawMessage(`1`), "status": json.RawMessage(`"ACTIVE"`), "offset": json.RawMessage(`0`), "unknown": json.RawMessage(`invalid ignored`)}
	option := secrets.WithRecordListOpts(secrets.RecordListOpts{Query: query, MaxItems: 2})
	query["name"] = json.RawMessage(`"changed"`)
	values, err := api.AllRecords(context.Background(), option)
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	recordListField(t, values[0].Resource, "id", `"second"`)
	requested, _ := url.ParseQuery(strings.SplitN((*calls)[0], "?", 2)[1])
	if !reflect.DeepEqual(requested, url.Values{"limit": {"2"}, "name": {"server", "second"}, "acl_only": {"False"}}) {
		t.Fatal("wrong declared server query", requested)
	}
	*calls = nil
	values, err = api.AllRecords(context.Background(), option, secrets.WithRecordListMaxItems(1))
	if err != nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal("cap counted matches instead of raw rows", values, err, *calls)
	}
	requested, _ = url.ParseQuery(strings.SplitN((*calls)[0], "?", 2)[1])
	if requested.Get("limit") != "1" {
		t.Fatal("missing falsey-limit hint", requested)
	}
}
func TestPythonSecretListRecordsConstructorControlsCapsAndSelf(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		for _, raw := range []string{"null", "false"} {
			t.Run(key+"/"+raw, func(t *testing.T) {
				body := `{"secrets":[{"id":"first"},{"id":"second","` + key + `":` + raw + `}]}`
				api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeSecretWire(200, body), nil })
				values, err := api.AllRecords(context.Background(), secrets.WithRecordListMaxItems(1))
				if err != nil || len(values) != 1 || len(*calls) != 1 {
					t.Fatal("cap consumed trailing collision", values, err, *calls)
				}
				*calls = nil
				values, err = api.AllRecords(context.Background())
				var proof *resource.ResponseError
				if values != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Secret-Proof") != "actual" || len(*calls) != 1 {
					t.Fatal("collision lost whole-page proof", values, err, proof, *calls)
				}
			})
		}
	}
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
		return nativeSecretWire(200, `{"secrets":{"id":"row","self":false}}`), nil
	})
	values, err := api.AllRecords(context.Background())
	if err != nil || len(values) != 1 || len(*calls) != 1 || string(values[0].Wire.Body["self"]) != "false" || values[0].Resource.Body["self"] != nil {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonSecretListRecordsRawCapsBeforeDescriptorAndLatePageErrors(t *testing.T) {
	body := `{"secrets":[{"id":"first"},{"id":"second","secret_ref":"relative"}],"next":"?offset=2"}`
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeSecretWire(200, body), nil })
	values, err := api.AllRecords(context.Background(), secrets.WithRecordListMaxItems(1))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	*calls = nil
	values, err = api.AllRecords(context.Background())
	var proof *resource.ResponseError
	if values != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
		t.Fatal(values, err, proof, *calls)
	}
	*calls = nil
	api.RawClient().ProviderClient.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.URL.String())
		if len(*calls) == 1 {
			return nativeSecretWire(200, `{"secrets":[{"id":"first"}],"next":"?offset=1"}`), nil
		}
		return nativeSecretWire(503, `{"failure":"late actual"}`), nil
	})
	values, err = api.AllRecords(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	if values != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"failure":"late actual"}` || native.ResponseHeader.Get("X-Secret-Proof") != "actual" || len(*calls) != 2 {
		t.Fatal(values, err, native, *calls)
	}
}
func TestPythonSecretListRecordsOffsetBodyHTTPLinkAndMarkerPaging(t *testing.T) {
	for _, mode := range []string{"offset-body", "offset-http", "marker-short"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				q := req.URL.Query()
				if q.Get("offset") == "" && q.Get("marker") == "" {
					response := nativeSecretWire(200, `{"secrets":[{"secret_ref":"https://passive.example/v1/secrets/first"}]}`)
					if mode == "offset-body" {
						response = nativeSecretWire(200, `{"secrets":[{"id":"first"}],"next":"/v1/secrets?offset=1&limit=2"}`)
					}
					if mode == "offset-http" {
						response.Header.Set("Link", `</v1/secrets?offset=1&limit=2>; rel="next"`)
					}
					return response, nil
				}
				if mode == "marker-short" {
					if q.Get("marker") != "https://passive.example/v1/secrets/first" || q.Get("limit") != "2" {
						t.Fatal(q)
					}
				} else if q.Get("offset") != "1" || q.Get("limit") != "2" {
					t.Fatal(q)
				}
				return nativeSecretWire(200, `{"secrets":[]}`), nil
			})
			var options []secrets.RecordListOption
			if mode == "marker-short" {
				options = append(options, secrets.WithRecordListQuery("limit", 2))
			}
			values, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(values) != 1 || len(*calls) != 2 {
				t.Fatal(values, err, *calls)
			}
			*calls = nil
			values, err = api.AllRecords(context.Background(), append(options, secrets.WithRecordListPaginated(false))...)
			if err != nil || len(values) != 1 || len(*calls) != 1 {
				t.Fatal(values, err, *calls)
			}
			*calls = nil
			for value, err := range api.ListRecords(context.Background(), options...) {
				if err != nil {
					t.Fatal(err)
				}
				value.Resource.Body["id"] = json.RawMessage(`"changed-public-id"`)
				break
			}
			if len(*calls) != 1 {
				t.Fatal("break fetched a continuation", *calls)
			}
			*calls = nil
			for value, err := range api.ListRecords(context.Background(), options...) {
				if err != nil {
					t.Fatal(err)
				}
				value.Resource.Body["id"] = json.RawMessage(`"changed-public-id"`)
			}
			if len(*calls) != 2 {
				t.Fatal("owned wire cursor changed by consumer", *calls)
			}
		})
	}
}
func TestPythonSecretListRecordsContinuationScopeAndCycles(t *testing.T) {
	for _, next := range []string{"https://foreign.example/v1/secrets?offset=1", "?offset=0", "?offset=-1", "?offset=1&type=changed", "?marker=same"} {
		t.Run(next, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"secrets": []any{map[string]any{"id": "first"}}, "next": next})
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeSecretWire(200, string(body)), nil })
			options := []secrets.RecordListOption{}
			if next == "?marker=same" {
				options = append(options, secrets.WithRecordListQuery("marker", "same"))
			}
			values, err := api.AllRecords(context.Background(), options...)
			var proof *resource.ResponseError
			if values != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != string(body) || len(*calls) != 1 {
				t.Fatal(values, err, proof, *calls)
			}
		})
	}
}
func TestPythonSecretListRecordsLazyOwnedHeadersVersionAndPreflight(t *testing.T) {
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Accept") != "application/custom" || req.Header.Get("X-Selected") != "owned" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" {
			t.Fatal(req.Header)
		}
		return nativeSecretWire(200, `{"secrets":[]}`), nil
	})
	headers := map[string]string{"accept": "application/custom", "X-Selected": "owned"}
	option := secrets.WithRecordListHeaders(headers)
	headers["X-Selected"] = "changed"
	var callbacks int
	stream := api.ListRecords(context.Background(), option, secrets.WithRecordListMicroversion("1.1"), func(c *secrets.RecordListOpts) error { callbacks++; return nil })
	if len(*calls) != 0 || callbacks != 0 {
		t.Fatal("eager iterator")
	}
	for _, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(*calls) != 1 || callbacks != 1 || api.RawClient().Microversion != "" {
		t.Fatal(*calls, callbacks, api.RawClient())
	}
	for _, option := range []secrets.RecordListOption{nil, secrets.WithRecordListMaxItems(-1), secrets.WithRecordListQuery("headers", nil), secrets.WithRecordListQuery("limit", -1), secrets.WithRecordListHeaders(map[string]string{"X-Auth-Token": "forbidden"})} {
		*calls = nil
		values, err := api.AllRecords(context.Background(), option)
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(values, err, *calls)
		}
	}
	*calls = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("option cancellation")
	values, err := api.AllRecords(ctx, func(c *secrets.RecordListOpts) error { cancel(cause); return nil })
	if values != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(*calls) != 0 {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonSecretListRecordsAcceptedSchemasAndNativeFailures(t *testing.T) {
	for _, body := range []string{`{"secrets":[null]}`, `{"secrets":null}`, `{"secrets":[1]}`, `[]`, `not json`, ``} {
		t.Run(body, func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeSecretWire(204, body), nil })
			values, err := api.AllRecords(context.Background())
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body || proof.Header.Get("X-Secret-Proof") != "actual" || len(*calls) != 1 {
				t.Fatal(values, err, proof, *calls)
			}
		})
	}
	for _, status := range []int{400, 403, 404, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				return nativeSecretWire(status, `{"native":"original"}`), nil
			})
			values, err := api.AllRecords(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodGet || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || !strings.HasSuffix(native.URL, "/barbican/v1/secrets") || string(native.Body) != `{"native":"original"}` || native.ResponseHeader.Get("X-Secret-Proof") != "actual" || len(*calls) != 1 {
				t.Fatal(values, err, native, *calls)
			}
		})
	}
	cause := errors.New("list transport cause")
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nil, cause })
	values, err := api.AllRecords(context.Background())
	if values != nil || !errors.Is(err, cause) || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
}

func TestPythonSecretListRecordsPhysicalFaultsStickySourceAndNilRetryNeverReplay(t *testing.T) {
	for _, status := range []int{200, 404, 429, 500, 503} {
		for _, mixed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/sticky-cancel-%v", status, mixed), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("existing fixture list read"), errors.New("existing fixture list close"), errors.New("list custom cancellation")
				body := `{"secrets":[],"failure":"original physical bytes"}`
				var api *secrets.API
				var calls *[]string
				original := ""
				closes := 0
				api, calls = recordListAPI(t, func(req *http.Request) (*http.Response, error) {
					response := nativeSecretWire(status, body)
					response.Body = &secretFindRecordFaultBody{data: []byte(body), readErr: readCause, closeErr: closeCause,
						read: func() {
							if mixed {
								api.RawClient().ResourceBase = original + "temporary/"
							}
						},
						close: func() {
							closes++
							if mixed {
								api.RawClient().ResourceBase = original
								cancel(cancelCause)
							}
						},
					}
					return response, nil
				})
				original = api.RawClient().ResourceBase
				retryCalls := 0
				hook := func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
					retryCalls++
					return nil
				}
				api.RawClient().RetryFunc = hook
				values, err := api.AllRecords(ctx)
				if values != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || len(*calls) != 1 || closes != 1 || retryCalls > 1 || reflect.ValueOf(api.RawClient().RetryFunc).Pointer() != reflect.ValueOf(hook).Pointer() {
					t.Fatal(values, err, *calls, closes, retryCalls)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Secret-Proof") != "actual" {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || native.Method != http.MethodGet || native.URL != original+"secrets" || string(native.Body) != body || native.ResponseHeader.Get("X-Secret-Proof") != "actual" {
						t.Fatal(err, native)
					}
				}
				if mixed && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || api.RawClient().ResourceBase != original) {
					t.Fatal("sticky or custom cancellation lost", err)
				}
			})
		}
	}
}

func TestPythonSecretListRecordsExactDecimalNumericBoundary(t *testing.T) {
	for _, filter := range []string{`{"count":9007199254740993.0}`, `{"count":9007199254740992.0}`} {
		t.Run(filter, func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				return nativeSecretWire(200, `{"secrets":[{"id":"s1","content_types":{"count":9007199254740992}}]}`), nil
			})
			values, err := api.AllRecords(context.Background(), secrets.WithRecordListQuery("content_types", json.RawMessage(filter)))
			want := 0
			if filter == `{"count":9007199254740992.0}` {
				want = 1
			}
			if err != nil || len(values) != want || len(*calls) != 1 {
				t.Fatal(values, err, *calls)
			}
		})
	}
}
