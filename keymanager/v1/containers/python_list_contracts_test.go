package containers_test

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
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordListAPI(t *testing.T, reply func(*http.Request) (*http.Response, error)) (*containers.API, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	client := nativeContainerClient(cloud)
	calls := &[]string{}
	cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		if req.URL.Path != "/barbican/v1/containers" {
			t.Fatal("unexpected member/payload/discovery request", req.URL)
		}
		return reply(req)
	})
	return containers.New(client), calls
}
func recordListField(t *testing.T, value *resource.RawResource, key, want string) {
	t.Helper()
	if value == nil || string(value.Body[key]) != want {
		t.Fatalf("field %s got %#v want %s", key, value, want)
	}
}
func TestPythonContainerListRecordsDefaultsAndOwnedResourceEvidence(t *testing.T) {
	for _, version := range []string{"", "1.1"} {
		t.Run("version-"+version, func(t *testing.T) {
			body := `{"containers":[{"id":null,"container_ref":"https://passive.example/v1/containers/c1%2Fraw?x=1","name":false,"status":0,"secret_refs":{},"consumers":false,"created":null,"updated":"unparsed","unknown":{"n":9007199254740993}}]}`
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
				return nativeContainerWire(199, body), nil
			})
			api.RawClient().Microversion = version
			api.RawClient().MoreHeaders = map[string]string{"aCcEpT": "configured", "X-Source": "original"}
			values, err := api.AllRecords(context.Background())
			if err != nil || len(values) != 1 || len(*calls) != 1 {
				t.Fatal(values, err, *calls)
			}
			value := values[0]
			if value.RequestID != "" || value.StatusCode != 199 || value.Resource.StatusCode != 199 || value.Wire.StatusCode != 199 || len(value.Resource.Body) != 11 || string(value.Envelope) != body {
				t.Fatal(value)
			}
			recordListField(t, value.Resource, "id", `null`)
			recordListField(t, value.Resource, "container_id", `"c1%2Fraw"`)
			recordListField(t, value.Resource, "secret_refs", `[{}]`)
			recordListField(t, value.Resource, "consumers", `[false]`)
			recordListField(t, value.Resource, "name", `false`)
			recordListField(t, value.Resource, "type", `null`)
			recordListField(t, value.Resource, "created_at", `null`)
			recordListField(t, value.Resource, "updated_at", `"unparsed"`)
			recordListField(t, value.Resource, "location", "null")
			if string(value.Wire.Body["unknown"]) != `{"n":9007199254740993}` || value.Resource.Body["unknown"] != nil {
				t.Fatal(value)
			}
			original := string(value.Wire.Body["container_ref"])
			value.Resource.Body["container_ref"][0] = 'x'
			value.Header.Set("X-Changed", "yes")
			value.Envelope[0] = 'x'
			if string(value.Wire.Body["container_ref"]) != original || value.Wire.Header.Get("X-Changed") != "" || value.Resource.Header.Get("X-Changed") != "" || api.RawClient().MoreHeaders["aCcEpT"] != "configured" || api.RawClient().Microversion != version {
				t.Fatal("owned channels alias", value)
			}

		})
	}
}
func TestPythonContainerListRecordsQueryClassificationAndRawCap(t *testing.T) {
	body := `{"containers":[{"id":"first","name":"other","type":true},{"id":"second","name":"wanted","type":1}]}`
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeContainerWire(200, body), nil })
	query := map[string]json.RawMessage{"limit": json.RawMessage(`0`), "name": json.RawMessage(`"wanted"`), "type": json.RawMessage(`true`), "offset": json.RawMessage(`0`), "unknown": json.RawMessage(`invalid ignored`)}
	option := containers.WithRecordListOpts(containers.RecordListOpts{Query: query, MaxItems: 2})
	query["name"] = json.RawMessage(`"changed"`)
	values, err := api.AllRecords(context.Background(), option)
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	recordListField(t, values[0].Resource, "id", `"second"`)
	requested, _ := url.ParseQuery(strings.SplitN((*calls)[0], "?", 2)[1])
	if !reflect.DeepEqual(requested, url.Values{"limit": {"2"}}) {
		t.Fatal("wrong declared server query", requested)
	}
	*calls = nil
	values, err = api.AllRecords(context.Background(), option, containers.WithRecordListMaxItems(1))
	if err != nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal("cap counted matches instead of raw rows", values, err, *calls)
	}
	requested, _ = url.ParseQuery(strings.SplitN((*calls)[0], "?", 2)[1])
	if requested.Get("limit") != "1" {
		t.Fatal("missing falsey-limit hint", requested)
	}
}
func TestPythonContainerListRecordsConstructorControlsCapsAndSelf(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		for _, raw := range []string{"null", "false"} {
			t.Run(key+"/"+raw, func(t *testing.T) {
				body := `{"containers":[{"id":"first"},{"id":"second","` + key + `":` + raw + `}]}`
				api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeContainerWire(200, body), nil })
				values, err := api.AllRecords(context.Background(), containers.WithRecordListMaxItems(1))
				if err != nil || len(values) != 1 || len(*calls) != 1 {
					t.Fatal("cap consumed trailing collision", values, err, *calls)
				}
				*calls = nil
				values, err = api.AllRecords(context.Background())
				var proof *resource.ResponseError
				if values != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Container-Proof") != "actual" || len(*calls) != 1 {
					t.Fatal("collision lost whole-page proof", values, err, proof, *calls)
				}
			})
		}
	}
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
		return nativeContainerWire(200, `{"containers":{"id":"row","self":false}}`), nil
	})
	values, err := api.AllRecords(context.Background())
	if err != nil || len(values) != 1 || len(*calls) != 1 || string(values[0].Wire.Body["self"]) != "false" || values[0].Resource.Body["self"] != nil {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonContainerListRecordsRawCapsBeforeDescriptorAndLatePageErrors(t *testing.T) {
	body := `{"containers":[{"id":"first"},{"id":"second","container_ref":"relative"}],"next":"?offset=2"}`
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeContainerWire(200, body), nil })
	values, err := api.AllRecords(context.Background(), containers.WithRecordListMaxItems(1))
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
	api.RawClient().ProviderClient.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.URL.String())
		if len(*calls) == 1 {
			return nativeContainerWire(200, `{"containers":[{"id":"first"}],"next":"?offset=1"}`), nil
		}
		return nativeContainerWire(503, `{"failure":"late actual"}`), nil
	})
	values, err = api.AllRecords(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	if values != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"failure":"late actual"}` || native.ResponseHeader.Get("X-Container-Proof") != "actual" || len(*calls) != 2 {
		t.Fatal(values, err, native, *calls)
	}
}
func TestPythonContainerListRecordsOffsetBodyHTTPLinkAndMarkerPaging(t *testing.T) {
	for _, mode := range []string{"offset-body", "offset-http", "marker-short"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				q := req.URL.Query()
				if q.Get("offset") == "" && q.Get("marker") == "" {
					response := nativeContainerWire(200, `{"containers":[{"container_ref":"https://passive.example/v1/containers/first"}]}`)
					if mode == "offset-body" {
						response = nativeContainerWire(200, `{"containers":[{"id":"first"}],"next":"/v1/containers?offset=1&limit=2"}`)
					}
					if mode == "offset-http" {
						response.Header.Set("Link", `</v1/containers?offset=1&limit=2>; rel="next"`)
					}
					return response, nil
				}
				if mode == "marker-short" {
					if q.Get("marker") != "https://passive.example/v1/containers/first" || q.Get("limit") != "2" {
						t.Fatal(q)
					}
				} else if q.Get("offset") != "1" || q.Get("limit") != "2" {
					t.Fatal(q)
				}
				return nativeContainerWire(200, `{"containers":[]}`), nil
			})
			var options []containers.RecordListOption
			if mode == "marker-short" {
				options = append(options, containers.WithRecordListQuery("limit", 2))
			}
			values, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(values) != 1 || len(*calls) != 2 {
				t.Fatal(values, err, *calls)
			}
			*calls = nil
			values, err = api.AllRecords(context.Background(), append(options, containers.WithRecordListPaginated(false))...)
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
func TestPythonContainerListRecordsContinuationScopeAndCycles(t *testing.T) {
	for _, next := range []string{"https://foreign.example/v1/containers?offset=1", "?offset=0", "?offset=-1", "?offset=1&type=changed", "?marker=same"} {
		t.Run(next, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"containers": []any{map[string]any{"id": "first"}}, "next": next})
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeContainerWire(200, string(body)), nil })
			options := []containers.RecordListOption{}
			if next == "?marker=same" {
				options = append(options, containers.WithRecordListQuery("marker", "same"))
			}
			values, err := api.AllRecords(context.Background(), options...)
			var proof *resource.ResponseError
			if values != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != string(body) || len(*calls) != 1 {
				t.Fatal(values, err, proof, *calls)
			}
		})
	}
}
func TestPythonContainerListRecordsLazyOwnedHeadersVersionAndPreflight(t *testing.T) {
	api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Accept") != "application/custom" || req.Header.Get("X-Selected") != "owned" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" {
			t.Fatal(req.Header)
		}
		return nativeContainerWire(200, `{"containers":[]}`), nil
	})
	headers := map[string]string{"accept": "application/custom", "X-Selected": "owned"}
	option := containers.WithRecordListHeaders(headers)
	headers["X-Selected"] = "changed"
	var callbacks int
	stream := api.ListRecords(context.Background(), option, containers.WithRecordListMicroversion("1.1"), func(c *containers.RecordListOpts) error { callbacks++; return nil })
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
	for _, option := range []containers.RecordListOption{nil, containers.WithRecordListMaxItems(-1), containers.WithRecordListQuery("headers", nil), containers.WithRecordListQuery("limit", -1), containers.WithRecordListHeaders(map[string]string{"X-Auth-Token": "forbidden"})} {
		*calls = nil
		values, err := api.AllRecords(context.Background(), option)
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(values, err, *calls)
		}
	}
	*calls = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("option cancellation")
	values, err := api.AllRecords(ctx, func(c *containers.RecordListOpts) error { cancel(cause); return nil })
	if values != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(*calls) != 0 {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonContainerListRecordsAcceptedSchemasAndNativeFailures(t *testing.T) {
	for _, body := range []string{`{"containers":[null]}`, `{"containers":null}`, `{"containers":[1]}`, `[]`, `not json`, ``} {
		t.Run(body, func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) { return nativeContainerWire(204, body), nil })
			values, err := api.AllRecords(context.Background())
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body || proof.Header.Get("X-Container-Proof") != "actual" || len(*calls) != 1 {
				t.Fatal(values, err, proof, *calls)
			}
		})
	}
	for _, status := range []int{400, 403, 404, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, calls := recordListAPI(t, func(req *http.Request) (*http.Response, error) {
				return nativeContainerWire(status, `{"native":"original"}`), nil
			})
			values, err := api.AllRecords(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodGet || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || !strings.HasSuffix(native.URL, "/barbican/v1/containers") || string(native.Body) != `{"native":"original"}` || native.ResponseHeader.Get("X-Container-Proof") != "actual" || len(*calls) != 1 {
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
