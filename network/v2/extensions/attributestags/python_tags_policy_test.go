package attributestags_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonTagPolicyRejected(t *testing.T, result *attributestags.TagResult, err error, calls *[]pythonTagCall) {
	t.Helper()
	var original gophercloud.ErrUnexpectedResponseCode
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &original) || original.Actual != 503 || string(original.Body) != "original-503-proof" || original.ResponseHeader.Get("X-Tag-Proof") != "actual" || len(*calls) != 1 {
		t.Fatal(result, err, original, *calls)
	}
	if original.Method != http.MethodPut || original.URL == "" || len(original.Expected) != 200 || original.Expected[0] != 200 || original.Expected[199] != 399 || (*calls)[0].body != `{"tags":["caller"]}` {
		t.Fatal(original, *calls)
	}
}

func TestPythonTagsRetryCannotChangeSelectedOrAbsentVersionHeaders(t *testing.T) {
	cases := []struct {
		name, version string
		change        func(*gophercloud.RequestOpts)
	}{
		{"selected_value", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network changed" }},
		{"selected_delete", "2.0", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "OpenStack-API-Version") }},
		{"selected_case_alias", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["openstack-api-version"] = "network changed" }},
		{"absent_generic_added", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network 9.9" }},
		{"absent_nova_added", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99" }},
		{"absent_volume_added", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["x-openstack-volume-api-version"] = "3.99" }},
		{"selected_omit", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"openstack-api-version"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
				if tc.version == "" && len(req.Header.Values("OpenStack-API-Version")) != 0 {
					t.Error("absent selected version was emitted", req.Header)
				}
				if tc.version != "" && req.Header.Get("OpenStack-API-Version") != "network "+tc.version {
					t.Error("selected version changed", req.Header)
				}
				return pythonTagWire(503, "original-503-proof")
			})
			client.Microversion = tc.version
			callbacks := 0
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				tc.change(o)
				return nil
			}
			got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{"caller"})
			pythonTagPolicyRejected(t, got, err, calls)
			if callbacks != 1 || client.Microversion != tc.version {
				t.Fatal(callbacks, client.Microversion)
			}
		})
	}
}

func TestPythonTagsRetryCannotAddReservedHeaders(t *testing.T) {
	for _, header := range []string{"authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade"} {
		t.Run(header, func(t *testing.T) {
			api, client, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(503, "original-503-proof") })
			callbacks := 0
			cause := errors.New("native callback cause")
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				o.MoreHeaders[header] = "foreign"
				if header == "authorization" {
					return cause
				}
				return nil
			}
			got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{"caller"})
			pythonTagPolicyRejected(t, got, err, calls)
			if callbacks != 1 || header == "authorization" && !errors.Is(err, cause) {
				t.Fatal(callbacks, err)
			}
		})
	}
}

func TestPythonTagsRetryOwnsBulkJSONContentTypeAndOmissions(t *testing.T) {
	cases := []struct {
		name, sourceType, optionType string
		omit                         bool
	}{
		{"default_changed", "", "", false},
		{"source_changed", "application/vnd.source+json", "", false},
		{"option_changed", "application/vnd.source+json", "application/vnd.option+json", false},
		{"default_omitted", "", "", true},
		{"source_omitted", "application/vnd.source+json", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.sourceType
			if want == "" {
				want = "application/json"
			}
			if tc.optionType != "" {
				want = tc.optionType
			}
			api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
				if req.Header.Get("Content-Type") != want {
					t.Error(req.Header, want)
				}
				return pythonTagWire(503, "original-503-proof")
			})
			if tc.sourceType != "" {
				client.MoreHeaders = map[string]string{"Content-Type": tc.sourceType}
			}
			callbacks := 0
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				if tc.omit {
					o.OmitHeaders = []string{"content-type"}
				} else {
					o.MoreHeaders["Content-Type"] = "application/octet-stream"
				}
				return nil
			}
			options := []attributestags.TagOption{}
			if tc.optionType != "" {
				options = append(options, attributestags.WithTagHeader("Content-Type", tc.optionType))
			}
			got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{"caller"}, options...)
			pythonTagPolicyRejected(t, got, err, calls)
			if callbacks != 1 {
				t.Fatal(callbacks)
			}
		})
	}
	// Bulk POST uses the same body policy; ordinary retry headers remain native.
	attempts := 0
	api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
		attempts++
		if req.Header.Get("Content-Type") != "application/vnd.tags+json" {
			t.Error(req.Header)
		}
		if attempts == 1 {
			return pythonTagWire(503, "original")
		}
		if req.Header.Get("X-Retry") != "second" {
			t.Error(req.Header)
		}
		return pythonTagWire(200, "bulk-ack")
	})
	client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
		o.MoreHeaders["X-Retry"] = "second"
		return nil
	}
	got, err := api.AddTags(context.Background(), pythonTagTarget(), []string{"caller"}, attributestags.WithTagHeader("Content-Type", "application/vnd.tags+json"))
	if err != nil || got == nil || attempts != 2 || len(*calls) != 2 || (*calls)[0].method != http.MethodPost || (*calls)[0].body != `{"tags":["caller"]}` || (*calls)[0].body != (*calls)[1].body || string(got.Body) != "bulk-ack" || !reflect.DeepEqual(got.Resource.Tags(), []string{"caller"}) {
		t.Fatal(got, err, *calls)
	}
}

func TestPythonTagsOrdinaryRetryHeadersAndLiveTokenKeepNativePolicy(t *testing.T) {
	t.Run("bodyless_ordinary_headers", func(t *testing.T) {
		attempts := 0
		api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
			attempts++
			if attempts == 1 {
				return pythonTagWire(503, "original")
			}
			if req.Header.Get("X-Retry") != "second" || req.Header.Get("Content-Type") != "application/native-bodyless" {
				t.Error(req.Header)
			}
			return pythonTagWire(200, `{"tags":["actual"]}`)
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.MoreHeaders["X-Retry"] = "second"
			o.MoreHeaders["Content-Type"] = "application/native-bodyless"
			o.MoreHeaders["X-Auth-Token"] = "forged"
			o.OmitHeaders = []string{"Authorization", "X-OpenStack-Nova-API-Version"}
			return nil
		}
		got, err := api.GetTags(context.Background(), pythonTagTarget())
		if err != nil || got == nil || attempts != 2 || len(*calls) != 2 || !got.ObservedTags || !reflect.DeepEqual(got.Resource.Tags(), []string{"actual"}) || (*calls)[0].body != "" || (*calls)[1].body != "" {
			t.Fatal(got, err, *calls)
		}
	})
	t.Run("live_token", func(t *testing.T) {
		cloud := testcloud.New(t)
		seen := []string{}
		cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Header.Get("X-Auth-Token"))
			if len(seen) == 1 {
				return pythonTagWire(503, "original"), nil
			}
			return pythonTagWire(200, `{"tags":["actual"]}`), nil
		})
		client := cloud.Client("network", "/neutron/v2.0/")
		client.Microversion = "2.0"
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			cloud.Provider.SetToken("fresh-token")
			o.MoreHeaders["X-Auth-Token"] = "forged"
			o.OmitHeaders = []string{"X-Auth-Token"}
			return nil
		}
		got, err := attributestags.New(client).GetTags(context.Background(), pythonTagTarget())
		if err != nil || got == nil || !reflect.DeepEqual(seen, []string{"test-token", "fresh-token"}) || !reflect.DeepEqual(got.Resource.Tags(), []string{"actual"}) || cloud.Provider.Token() != "fresh-token" {
			t.Fatal(got, err, seen, cloud.Provider.Token())
		}
	})
}
