package attributestags_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonTagCall struct{ method, path, body, query string }

func pythonTagWire(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"Content-Type": {"application/json"}, "X-Tag-Proof": {"actual"}})
}
func pythonTagCloud(t *testing.T, reply func(*http.Request) *http.Response) (*attributestags.API, *gophercloud.ServiceClient, *[]pythonTagCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]pythonTagCall{}
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		body := ""
		if req.Body != nil {
			raw, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = string(raw)
		}
		*calls = append(*calls, pythonTagCall{req.Method, req.URL.EscapedPath(), body, req.URL.RawQuery})
		if req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return attributestags.New(client), client, calls
}
func pythonTagTarget() attributestags.TaggedResource {
	return attributestags.NewTaggedResource(attributestags.NetworkResource, "n1")
}

func TestPythonTagsKindsRoutesAndBulkCapabilities(t *testing.T) {
	cases := []struct {
		kind attributestags.ResourceKind
		path string
		bulk bool
	}{
		{attributestags.NetworkResource, "networks", true}, {attributestags.PortResource, "ports", true},
		{attributestags.SubnetResource, "subnets", true}, {attributestags.SubnetPoolResource, "subnetpools", true},
		{attributestags.RouterResource, "routers", true}, {attributestags.FloatingIPResource, "floatingips", true},
		{attributestags.SecurityGroupResource, "security-groups", true}, {attributestags.SecurityGroupRuleResource, "security-group-rules", true},
		{attributestags.QoSPolicyResource, "policies", false}, {attributestags.TrunkResource, "trunks", false},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(200, `{"tags":[]}`) })
			target := attributestags.NewTaggedResource(tc.kind, "id")
			options := []attributestags.TagOption{}
			if tc.kind == attributestags.SecurityGroupRuleResource {
				got, err := api.GetTags(context.Background(), target)
				if got != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
					t.Fatal("stock27 lacks SGR tag controller", got, err, *calls)
				}
				options = append(options, attributestags.WithSourceSecurityGroupRuleTags())
			}
			got, err := api.GetTags(context.Background(), target, options...)
			if err != nil || got == nil || !got.ObservedTags || got.Resource.Kind() != tc.kind || got.Resource.ID() != "id" || !reflect.DeepEqual(got.Resource.Tags(), []string{}) || len(*calls) != 1 || (*calls)[0].path != "/neutron/v2.0/"+tc.path+"/id/tags" {
				t.Fatal(got, err, *calls)
			}
			got, err = api.AddTags(context.Background(), target, []string{"a"}, options...)
			if tc.bulk {
				if err != nil || got == nil || len(*calls) != 2 || (*calls)[1].method != http.MethodPost || (*calls)[1].body != `{"tags":["a"]}` || !reflect.DeepEqual(got.Resource.Tags(), []string{"a"}) {
					t.Fatal(got, err, *calls)
				}
			} else if got != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 1 {
				t.Fatal(got, err, *calls)
			}
			// All ten Source-tagged kinds have replacement routes, with SGR explicitly
			// tested as compatibility-only and QoS using its stock tag collection.
			got, err = api.SetTags(context.Background(), target, []string{}, options...)
			if err != nil || got == nil || (*calls)[len(*calls)-1].path != "/neutron/v2.0/"+tc.path+"/id/tags" || (*calls)[len(*calls)-1].body != `{"tags":[]}` {
				t.Fatal(got, err, *calls)
			}
		})
	}
}

func TestPythonTagsCacheTransitionsDoNotInventResponseState(t *testing.T) {
	api, _, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/tags") {
			return pythonTagWire(200, `{"unrelated":true}`)
		}
		return pythonTagWire(204, `opaque acknowledgement`)
	})
	original := pythonTagTarget().WithCachedTags([]string{"a", "a", "b"})
	got, err := api.GetTags(context.Background(), original)
	if err != nil || got == nil || got.ObservedTags || !reflect.DeepEqual(got.Resource.Tags(), []string{"a", "a", "b"}) {
		t.Fatal(got, err)
	}
	got, err = api.AddTag(context.Background(), original, "a")
	if err != nil || got == nil || got.ObservedTags || !reflect.DeepEqual(got.Resource.Tags(), []string{"a", "a", "b", "a"}) || string(got.Body) != `opaque acknowledgement` {
		t.Fatal(got, err)
	}
	got, err = api.RemoveTag(context.Background(), original, "a")
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{"a", "b"}) {
		t.Fatal(got, err)
	}
	got, err = api.RemoveTag(context.Background(), original, "not-cached")
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), original.Tags()) {
		t.Fatal(got, err)
	}
	got, err = api.CheckTag(context.Background(), original, "a")
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), original.Tags()) {
		t.Fatal(got, err)
	}
	got, err = api.SetTags(context.Background(), original, nil)
	if err != nil || got == nil || got.Resource.Tags() != nil {
		t.Fatal(got, err)
	}
	got, err = api.AddTags(context.Background(), original, nil)
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{}) {
		t.Fatal(got, err)
	}
	got, err = api.AddTags(context.Background(), original, []string{"new"})
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{"new"}) {
		t.Fatal("bulk-add local cache is supplied list, not union", got, err)
	}
	got, err = api.RemoveAllTags(context.Background(), original)
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{}) {
		t.Fatal(got, err)
	}
	base := "/neutron/v2.0/networks/n1/tags"
	want := []pythonTagCall{
		{http.MethodGet, base, "", ""}, {http.MethodPut, base + "/a", "", ""},
		{http.MethodDelete, base + "/a", "", ""}, {http.MethodDelete, base + "/not-cached", "", ""},
		{http.MethodGet, base + "/a", "", ""}, {http.MethodPut, base, `{"tags":null}`, ""},
		{http.MethodPost, base, `{"tags":[]}`, ""}, {http.MethodPost, base, `{"tags":["new"]}`, ""},
		{http.MethodDelete, base, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(original.Tags(), []string{"a", "a", "b"}) {
		t.Fatal(*calls, original.Tags())
	}
	// Tags getters return snapshots as well; consumers cannot mutate the cache.
	fetched := original.Tags()
	fetched[0] = "changed"
	if original.Tags()[0] != "a" {
		t.Fatal(original.Tags())
	}
}

func TestPythonTagsFetchPresenceNullAndDecodeEvidence(t *testing.T) {
	cases := []struct {
		body           string
		tags           []string
		observed, fail bool
	}{
		{`{}`, []string{"cached"}, false, false}, {`{"tags":[]}`, []string{}, true, false},
		{`{"tags":null}`, nil, true, false}, {`{"tags":["a","a",""]}`, []string{"a", "a", ""}, true, false},
		{`{"tags":[7]}`, []string{"cached"}, false, true}, {`{"tags":[null]}`, []string{"cached"}, false, true},
		{`{"tags":"a"}`, []string{"cached"}, false, true}, {`null`, []string{"cached"}, false, true},
		{`[]`, []string{"cached"}, false, true}, {`not JSON`, []string{"cached"}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(203, tc.body) })
			got, err := api.GetTags(context.Background(), pythonTagTarget().WithCachedTags([]string{"cached"}))
			if got == nil || len(*calls) != 1 || got.StatusCode != 203 || string(got.Body) != tc.body || got.Header.Get("X-Tag-Proof") != "actual" || got.ObservedTags != tc.observed || !reflect.DeepEqual(got.Resource.Tags(), tc.tags) {
				t.Fatal(got, err, *calls)
			}
			if tc.fail {
				var receipt *resource.ResponseError
				if err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 203 || string(receipt.Body) != tc.body || receipt.Header.Get("X-Tag-Proof") != "actual" {
					t.Fatal(err, receipt)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
	api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(200, `{}`) })
	got, err := api.GetTags(context.Background(), pythonTagTarget())
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{}) || got.ObservedTags || len(*calls) != 1 {
		t.Fatal(got, err, *calls)
	}
}

func TestPythonTagsStrictMissingAndAcceptedCodes(t *testing.T) {
	actions := []struct {
		name string
		call func(*attributestags.API) (*attributestags.TagResult, error)
	}{
		{"GetTags", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.GetTags(context.Background(), pythonTagTarget())
		}},
		{"SetTags", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.SetTags(context.Background(), pythonTagTarget(), []string{})
		}},
		{"AddTags", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.AddTags(context.Background(), pythonTagTarget(), []string{})
		}},
		{"AddTag", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.AddTag(context.Background(), pythonTagTarget(), "a")
		}},
		{"RemoveTag", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.RemoveTag(context.Background(), pythonTagTarget(), "a")
		}},
		{"RemoveAllTags", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.RemoveAllTags(context.Background(), pythonTagTarget())
		}},
		{"CheckTag", func(a *attributestags.API) (*attributestags.TagResult, error) {
			return a.CheckTag(context.Background(), pythonTagTarget(), "a")
		}},
	}
	for _, action := range actions {
		for _, status := range []int{200, 201, 203, 204, 299, 300, 399, 400, 404, 409, 503} {
			t.Run(action.name+fmt.Sprint(status), func(t *testing.T) {
				api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(status, `{"tags":[]}`) })
				got, err := action.call(api)
				if len(*calls) != 1 {
					t.Fatal(*calls)
				}
				if status >= 200 && status < 400 {
					if err != nil || got == nil || got.StatusCode != status || got.Header.Get("X-Tag-Proof") != "actual" {
						t.Fatal(got, err)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					var operation *resource.OperationError
					if got != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"tags":[]}` || native.ResponseHeader.Get("X-Tag-Proof") != "actual" || !errors.As(err, &operation) || operation.Operation != action.name {
						t.Fatal(got, err, native, operation)
					}
				}
			})
		}
	}
}

func TestPythonTagsLiteralSegmentsExtensionsAndSourceHeaders(t *testing.T) {
	var response *http.Response
	api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Test") != "caller" || req.Header.Get("X-Source") != "before" {
			t.Error(req.Header)
		}
		response = pythonTagWire(200, `{"tags":["server"]}`)
		return response
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	target := attributestags.NewTaggedResource(attributestags.PortResource, "a/b%2Fz")
	values := map[string]any{"zero": 0, "null": nil, "false": false, "empty": []string{}}
	field := attributestags.WithTagField("future", values)
	values["zero"] = 7
	got, err := api.SetTags(context.Background(), target, []string{"duplicate", "duplicate", ""}, field, attributestags.WithTagHeader("X-Test", "caller"))
	if err != nil || got == nil || !reflect.DeepEqual(got.Resource.Tags(), []string{"duplicate", "duplicate", ""}) || got.ObservedTags || len(*calls) != 1 || (*calls)[0].path != "/neutron/v2.0/ports/a%2Fb%252Fz/tags" || (*calls)[0].body != `{"future":{"empty":[],"false":false,"null":null,"zero":0},"tags":["duplicate","duplicate",""]}` || string(got.Body) != `{"tags":["server"]}` {
		t.Fatal(got, err, *calls)
	}
	response.Header.Set("X-Tag-Proof", "changed")
	if got.Header.Get("X-Tag-Proof") != "actual" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "before"}) {
		t.Fatal(got.Header, client.MoreHeaders)
	}
	got, err = api.AddTag(context.Background(), target, "x/y%2Fz", attributestags.WithTagHeader("X-Test", "caller"))
	if err != nil || got == nil || len(*calls) != 2 || (*calls)[1].path != "/neutron/v2.0/ports/a%2Fb%252Fz/tags/x%2Fy%252Fz" || (*calls)[1].body != "" {
		t.Fatal(got, err, *calls)
	}
}

func TestPythonTagsPreflightAndSourceCapture(t *testing.T) {
	api, client, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(200, `{"tags":[]}`) })
	count := 0
	option := func(c *request.Config[attributestags.TagOpts]) error { count++; return nil }
	var nilAPI *attributestags.API
	if got, err := nilAPI.GetTags(context.Background(), pythonTagTarget(), option); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	if got, err := api.GetTags(nil, pythonTagTarget(), option); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	cause := errors.New("preflight cancellation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if got, err := api.GetTags(ctx, pythonTagTarget(), option); got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(got, err)
	}
	for _, id := range []string{"", ".", "..", "bad\n", "\xff"} {
		if got, err := api.GetTags(context.Background(), attributestags.NewTaggedResource(attributestags.NetworkResource, id), option); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, got, err)
		}
	}
	if got, err := api.GetTags(context.Background(), attributestags.NewTaggedResource("address-group", "id"), option); got != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(got, err)
	}
	if count != 0 || len(*calls) != 0 {
		t.Fatal(count, *calls)
	}
	options := []attributestags.TagOption{nil, attributestags.WithSourceSecurityGroupRuleTags(), attributestags.WithTagField("tags", nil), attributestags.WithTagHeader("X-Auth-Token", "foreign"), attributestags.WithTagHeader("Content-Length", "1"), attributestags.WithTagHeader("X-Test", "bad\n"), request.WithQuery[attributestags.TagOpts]("x", "1"), request.WithArgument[attributestags.TagOpts]("x", 1)}
	for i, option := range options {
		if got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{}, option); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(i, got, err)
		}
	}
	if got, err := api.AddTag(context.Background(), pythonTagTarget(), "a", attributestags.WithTagField("future", false)); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	if got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{"\xff"}); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	got, err := api.GetTags(context.Background(), pythonTagTarget(), func(c *request.Config[attributestags.TagOpts]) error {
		client.ResourceBase = "https://changed.invalid/v2.0/"
		return nil
	})
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(got, err, *calls)
	}
}

type pythonTagFaultBody struct {
	reader                io.Reader
	readErr, closeErr     error
	afterRead, afterClose func()
}

func (b *pythonTagFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if b.afterRead != nil {
		b.afterRead()
		b.afterRead = nil
	}
	if b.readErr != nil {
		err = b.readErr
		b.readErr = nil
	}
	return n, err
}
func (b *pythonTagFaultBody) Close() error {
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}
func TestPythonTagsPhysicalFaultsRetainAcceptedAndRejectedProof(t *testing.T) {
	for _, status := range []int{204, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var client *gophercloud.ServiceClient
			readErr, closeErr, cause := errors.New("physical read"), errors.New("physical close"), errors.New("cancel cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			original := ""
			api, source, calls := pythonTagCloud(t, func(*http.Request) *http.Response {
				response := pythonTagWire(status, "")
				response.Body = &pythonTagFaultBody{reader: strings.NewReader("partial-wire"), readErr: readErr, closeErr: closeErr, afterRead: func() { client.ResourceBase = "https://changed.invalid/"; cancel(cause) }, afterClose: func() { client.ResourceBase = original }}
				return response
			})
			client = source
			original = client.ResourceBase
			input := pythonTagTarget().WithCachedTags([]string{"unchanged"})
			got, err := api.CheckTag(ctx, input, "a")
			for _, target := range []error{readErr, closeErr, context.Canceled, cause, resource.ErrInvalidOption} {
				if !errors.Is(err, target) {
					t.Fatal(target, got, err)
				}
			}
			if len(*calls) != 1 || !reflect.DeepEqual(input.Tags(), []string{"unchanged"}) {
				t.Fatal(*calls, input.Tags())
			}
			if status < 400 {
				var receipt *resource.ResponseError
				if got == nil || got.StatusCode != 204 || string(got.Body) != "partial-wire" || !reflect.DeepEqual(got.Resource.Tags(), input.Tags()) || !errors.As(err, &receipt) || receipt.StatusCode != 204 || string(receipt.Body) != "partial-wire" || receipt.Header.Get("X-Tag-Proof") != "actual" {
					t.Fatal(got, err, receipt)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "partial-wire" || native.ResponseHeader.Get("X-Tag-Proof") != "actual" {
					t.Fatal(got, err, native)
				}
			}
		})
	}
}

func TestPythonTagsRetryOwnershipCleanRetryAndDeniedCause(t *testing.T) {
	for _, change := range []bool{false, true} {
		t.Run(fmt.Sprint(change), func(t *testing.T) {
			attempts := 0
			api, client, calls := pythonTagCloud(t, func(*http.Request) *http.Response {
				attempts++
				if attempts == 1 {
					return pythonTagWire(503, "original-failure")
				}
				return pythonTagWire(200, `{"tags":["actual"]}`)
			})
			callbacks := 0
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
				callbacks++
				if change {
					options.JSONBody = map[string]any{"tags": []string{"changed"}}
				}
				return nil
			}
			got, err := api.SetTags(context.Background(), pythonTagTarget(), []string{"caller"})
			if callbacks != 1 {
				t.Fatal(callbacks)
			}
			if change {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original-failure" || attempts != 1 {
					t.Fatal(got, err, native, attempts)
				}
			} else if err != nil || got == nil || got.StatusCode != 200 || !reflect.DeepEqual(got.Resource.Tags(), []string{"caller"}) || string(got.Body) != `{"tags":["actual"]}` || attempts != 2 || (*calls)[0].body != (*calls)[1].body {
				t.Fatal(got, err, attempts, *calls)
			}
		})
	}
	api, client, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(404, "gone") })
	cause := errors.New("callback denied")
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return cause }
	got, err := api.RemoveAllTags(context.Background(), pythonTagTarget())
	var native gophercloud.ErrUnexpectedResponseCode
	if got != nil || !errors.Is(err, cause) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "gone" || len(*calls) != 1 {
		t.Fatal(got, err, native, *calls)
	}
}

func TestPythonTagsNullCacheFailureKeepsAcknowledgement(t *testing.T) {
	api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(200, `{"tags":null}`) })
	fetched, err := api.GetTags(context.Background(), pythonTagTarget())
	if err != nil || fetched == nil || fetched.Resource.Tags() != nil {
		t.Fatal(fetched, err)
	}
	for _, add := range []bool{true, false} {
		var got *attributestags.TagResult
		if add {
			got, err = api.AddTag(context.Background(), fetched.Resource, "a")
		} else {
			got, err = api.RemoveTag(context.Background(), fetched.Resource, "a")
		}
		var receipt *resource.ResponseError
		if got == nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != `{"tags":null}` || receipt.Header.Get("X-Tag-Proof") != "actual" || got.Resource.Tags() != nil {
			t.Fatal(got, err, receipt)
		}
	}
	if len(*calls) != 3 {
		t.Fatal(*calls)
	}
}

func TestPythonTagsNoContentFetchAndMutationBodies(t *testing.T) {
	api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(204, "") })
	got, err := api.GetTags(context.Background(), pythonTagTarget().WithCachedTags([]string{"cache"}))
	var receipt *resource.ResponseError
	if got == nil || err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 204 || string(receipt.Body) != "" || got.ObservedTags || !reflect.DeepEqual(got.Resource.Tags(), []string{"cache"}) {
		t.Fatal(got, err, receipt)
	}
	got, err = api.SetTags(context.Background(), pythonTagTarget(), []string{"input"})
	if err != nil || got == nil || got.StatusCode != 204 || string(got.Body) != "" || got.ObservedTags || !reflect.DeepEqual(got.Resource.Tags(), []string{"input"}) || len(*calls) != 2 {
		t.Fatal(got, err, *calls)
	}
}

func TestPythonTagsCallerSlicesAndRetainedOptionsAreSnapshots(t *testing.T) {
	tags := []string{"input"}
	cached := []string{"cache"}
	target := pythonTagTarget().WithCachedTags(cached)
	cached[0] = "caller mutation"
	var retained *request.Config[attributestags.TagOpts]
	attempts := 0
	api, client, calls := pythonTagCloud(t, func(req *http.Request) *http.Response {
		attempts++
		if req.Header.Get("X-Test") != "original" {
			t.Error(req.Header)
		}
		if retained != nil {
			retained.Fields["future"] = []byte(`99`)
			retained.Headers["X-Test"] = "changed after request"
		}
		if attempts == 1 {
			return pythonTagWire(503, "first")
		}
		return pythonTagWire(200, `{"tags":["actual"]}`)
	})
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
	got, err := api.SetTags(context.Background(), target, tags, attributestags.WithTagField("future", false), func(c *request.Config[attributestags.TagOpts]) error {
		retained = c
		c.Headers["X-Test"] = "original"
		tags[0] = "mutated inside option"
		return nil
	})
	want := `{"future":false,"tags":["input"]}`
	if err != nil || got == nil || attempts != 2 || len(*calls) != 2 || (*calls)[0].body != want || (*calls)[1].body != want || !reflect.DeepEqual(got.Resource.Tags(), []string{"input"}) || !reflect.DeepEqual(target.Tags(), []string{"cache"}) || got.ObservedTags {
		t.Fatal(got, err, *calls, target.Tags())
	}
}

func TestPythonTagsSecurityGroupRuleSourceCapabilityIsExplicit(t *testing.T) {
	target := attributestags.NewTaggedResource(attributestags.SecurityGroupRuleResource, "r1")
	actions := []struct {
		name string
		call func(*attributestags.API, ...attributestags.TagOption) (*attributestags.TagResult, error)
	}{
		{"GetTags", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.GetTags(context.Background(), target, o...)
		}},
		{"SetTags", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.SetTags(context.Background(), target, []string{}, o...)
		}},
		{"AddTags", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.AddTags(context.Background(), target, []string{}, o...)
		}},
		{"AddTag", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.AddTag(context.Background(), target, "a", o...)
		}},
		{"RemoveTag", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.RemoveTag(context.Background(), target, "a", o...)
		}},
		{"RemoveAllTags", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.RemoveAllTags(context.Background(), target, o...)
		}},
		{"CheckTag", func(a *attributestags.API, o ...attributestags.TagOption) (*attributestags.TagResult, error) {
			return a.CheckTag(context.Background(), target, "a", o...)
		}},
	}
	for _, action := range actions {
		t.Run(action.name, func(t *testing.T) {
			api, _, calls := pythonTagCloud(t, func(*http.Request) *http.Response { return pythonTagWire(200, `{"tags":[]}`) })
			got, err := action.call(api)
			if got != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
				t.Fatal("stock27 capability failed before HTTP", got, err, *calls)
			}
			got, err = action.call(api, attributestags.WithSourceSecurityGroupRuleTags())
			if err != nil || got == nil || len(*calls) != 1 || !strings.HasPrefix((*calls)[0].path, "/neutron/v2.0/security-group-rules/r1/tags") {
				t.Fatal("explicit Source route profile", got, err, *calls)
			}
		})
	}
}
