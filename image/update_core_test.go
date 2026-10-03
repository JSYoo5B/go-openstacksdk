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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func updateCorePayload(t *testing.T, req *http.Request) ([]ImagePatch, string) {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var changes []ImagePatch
	if err = json.Unmarshal(raw, &changes); err != nil || changes == nil {
		t.Fatal(string(raw), err)
	}
	return changes, string(raw)
}

func TestUpdateImageCoreFixedOrderedPatchAndMedia(t *testing.T) {
	for _, id := range []string{"이미지% ?#:parent", strings.Repeat("한", 300)} {
		calls := 0
		original := map[string]string{"Content-Type": "application/x-source", "Accept": "application/x-source", "X-Policy": "source"}
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "PATCH" || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id) || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Policy") != "option" || req.Header.Get("X-Auth-Token") != "first-token" {
				t.Fatal(req.Method, req.URL, req.Header)
			}
			changes, _ := updateCorePayload(t, req)
			expected := []ImagePatch{{Op: "replace", Path: "/name", Value: json.RawMessage(`"first"`)}, {Op: "remove", Path: "/custom"}, {Op: "add", Path: "/name", Value: json.RawMessage(`"last"`)}, {Op: "add", Path: "/locations/-", Value: json.RawMessage(`{"url":"foreign","metadata":{}}`)}}
			if !reflect.DeepEqual(changes, expected) {
				t.Fatal(changes)
			}
			return taskCoreJSON(req, 200, `{"id":"different","name":"fresh","created_at":"literal","protected":false,"size":9007199254740993,"locations":[{"url":"foreign","metadata":{"n":900719925474099312345}}],"custom":{"precise":1.000000000000000001},"links":false}`), nil
		})
		client.MoreHeaders = original
		value, err := New(client).UpdateImage(context.Background(), resource.ID(id), WithUpdateImageHeader("X-Policy", "option"), WithUpdateImageChanges(ImagePatch{Op: "replace", Path: "/name", Value: json.RawMessage(`"first"`)}, ImagePatch{Op: "remove", Path: "/custom"}, ImagePatch{Op: "add", Path: "/name", Value: json.RawMessage(`"last"`)}, ImagePatch{Op: "add", Path: "/locations/-", Value: json.RawMessage(`{"url":"foreign","metadata":{}}`)}))
		if err != nil || value == nil || *value.ID != "different" || *value.Name != "fresh" || *value.Size != 9007199254740993 || *value.CreatedAt != "literal" || value.StatusCode != 200 || value.Header.Get("X-Task-Proof") != "actual" || value.Links != nil || calls != 1 {
			t.Fatal(value, err, calls)
		}
		value.Properties["custom"][0] = '['
		value.Locations[0].Metadata["n"][0] = '8'
		if string(value.Body["custom"]) != `{"precise":1.000000000000000001}` || bytes.Contains(value.Body["locations"], []byte("800719")) {
			t.Fatal("typed/raw alias", value.Body)
		}
		if !reflect.DeepEqual(original, map[string]string{"Content-Type": "application/x-source", "Accept": "application/x-source", "X-Policy": "source"}) {
			t.Fatal("source headers changed", original)
		}
	}
}

func TestUpdateImageCoreSortedPropertiesAndEmptyPatch(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		changes, raw := updateCorePayload(t, req)
		if calls == 1 {
			expected := []ImagePatch{{Op: "add", Path: "/a~1b~0", Value: json.RawMessage("900719925474099312345")}, {Op: "add", Path: "/name", Value: json.RawMessage(`""`)}, {Op: "add", Path: "/properties", Value: json.RawMessage("[]")}, {Op: "add", Path: "/protected", Value: json.RawMessage("false")}, {Op: "add", Path: "/z", Value: json.RawMessage("null")}}
			if !reflect.DeepEqual(changes, expected) {
				t.Fatal(changes)
			}
		} else if raw != "[]" {
			t.Fatal("empty patch is not array", raw)
		}
		return taskCoreJSON(req, 200, `{"name":"fresh"}`), nil
	}))
	values := map[string]json.RawMessage{"z": json.RawMessage("null"), "a/b~": json.RawMessage("900719925474099312345"), "protected": json.RawMessage("false"), "properties": json.RawMessage("[]"), "name": json.RawMessage(`""`)}
	if v, e := service.SetImageProperties(context.Background(), resource.ID("id"), WithSetImagePropertiesProperties(values)); v == nil || e != nil || *v.Name != "fresh" {
		t.Fatal(v, e)
	}
	if v, e := service.UpdateImage(context.Background(), resource.ID("id")); v == nil || e != nil {
		t.Fatal(v, e)
	}
	if v, e := service.SetImageProperties(context.Background(), resource.ID("id")); v == nil || e != nil || calls != 3 {
		t.Fatal(v, e, calls)
	}
}

func TestUpdateImageCoreCompletePreflight(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID("."), resource.ID(".."), resource.ID("a/b"), resource.ID("a\\b"), resource.ID("a\n"), resource.ID(string([]byte{0xff})), resource.Name(string([]byte{0xff}))} {
		callbacks := 0
		if v, e := service.UpdateImage(context.Background(), ref, func(*UpdateImageOpts) error { callbacks++; return nil }); v != nil || e == nil || callbacks != 0 {
			t.Fatal(ref, v, e, callbacks)
		}
		if v, e := service.SetImageProperties(context.Background(), ref, func(*SetImagePropertiesOpts) error { callbacks++; return nil }); v != nil || e == nil || callbacks != 0 {
			t.Fatal(ref, v, e, callbacks)
		}
	}
	for _, change := range []ImagePatch{{Op: "ADD", Path: "/x", Value: json.RawMessage("1")}, {Op: "copy", Path: "/x"}, {Op: "add", Path: "x", Value: json.RawMessage("1")}, {Op: "add", Path: "/", Value: json.RawMessage("1")}, {Op: "add", Path: "/a//b", Value: json.RawMessage("1")}, {Op: "add", Path: "/a~2", Value: json.RawMessage("1")}, {Op: "add", Path: "/a~", Value: json.RawMessage("1")}, {Op: "add", Path: "/ x", Value: json.RawMessage("1")}, {Op: "add", Path: "/x\u001c", Value: json.RawMessage("1")}, {Op: "add", Path: "/x/\u2003b", Value: json.RawMessage("1")}, {Op: "add", Path: "/x"}, {Op: "replace", Path: "/x", Value: json.RawMessage{}}, {Op: "add", Path: "/x", Value: json.RawMessage("{")}, {Op: "add", Path: "/x", Value: json.RawMessage{'"', 0xff, '"'}}, {Op: "remove", Path: "/x", Value: json.RawMessage{}}, {Op: "remove", Path: "/x", Value: json.RawMessage("null")}} {
		if v, e := service.UpdateImage(context.Background(), resource.Name("Name"), WithUpdateImageChange(change)); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(change, v, e)
		}
	}
	for _, option := range []UpdateImageOption{nil, WithUpdateImageHeader("Accept", "other"), WithUpdateImageHeader("Content-Type", "other"), WithUpdateImageHeader("Authorization", "secret"), WithUpdateImageField("", 1), WithUpdateImageField(" x", 1), WithUpdateImageField(string([]byte{0xff}), 1)} {
		if v, e := service.UpdateImage(context.Background(), resource.Name("Name"), option); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(v, e)
		}
	}
	for _, option := range []SetImagePropertiesOption{nil, WithSetImagePropertiesHeader("Content-Type", "other"), WithSetImagePropertiesProperties(map[string]json.RawMessage{"x": {}}), WithSetImagePropertiesProperty("\u001fy", 1)} {
		if v, e := service.SetImageProperties(context.Background(), resource.Name("Name"), option); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(v, e)
		}
	}
	cause := errors.New("callback cause")
	if v, e := service.UpdateImage(context.Background(), resource.Name("Name"), func(*UpdateImageOpts) error { return cause }); v != nil || !errors.Is(e, cause) {
		t.Fatal(v, e)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	for _, ctx := range []context.Context{nil, ctx} {
		cb := 0
		if v, e := service.UpdateImage(ctx, resource.ID("id"), func(*UpdateImageOpts) error { cb++; return nil }); v != nil || e == nil || cb != 0 {
			t.Fatal(v, e, cb)
		}
	}
	var nilService *Service
	if v, e := nilService.UpdateImage(context.Background(), resource.ID("id")); v != nil || e == nil {
		t.Fatal(v, e)
	}
	if v, e := nilService.SetImageProperties(context.Background(), resource.ID("id")); v != nil || e == nil {
		t.Fatal(v, e)
	}
	if calls != 0 {
		t.Fatal("preflight reached HTTP", calls)
	}
}

func TestUpdateImageCoreExactNameAndSource(t *testing.T) {
	t.Run("captured headers before callback and fresh PATCH", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Fixed") != "option" || req.Header.Get("X-Auth-Token") != "later-token" {
				t.Fatal(req.Header)
			}
			switch calls {
			case 1:
				if req.Method != "GET" || req.Header.Get("Content-Type") != "application/x-source" || req.Header.Get("Accept") != "application/x-source" || req.URL.Query().Get("name") != "Parent" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, `{"images":[{"id":"lower","name":"parent"}],"next":"/v2/images?marker=next"}`), nil
			case 2:
				return taskCoreJSON(req, 200, `{"images":[{"id":"selected","name":"Parent"}]}`), nil
			case 3:
				if req.Method != "PATCH" || req.URL.Path != "/reverse/glance/v2/images/selected" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "application/json" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, `{"id":"passive"}`), nil
			}
			t.Fatal("extra request")
			return nil, nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured", "Content-Type": "application/x-source", "Accept": "application/x-source"}
		v, e := New(client).UpdateImage(context.Background(), resource.Name("Parent"), WithUpdateImageName("new"), WithUpdateImageHeader("X-Fixed", "option"), func(*UpdateImageOpts) error {
			client.MoreHeaders["X-Source"] = "changed"
			client.ProviderClient.SetToken("later-token")
			return nil
		})
		if e != nil || v == nil || *v.ID != "passive" || calls != 3 || client.MoreHeaders["Content-Type"] != "application/x-source" {
			t.Fatal(v, e, calls, client.MoreHeaders)
		}
	})
	for _, tc := range []struct {
		body  string
		cause error
	}{{`{"images":[]}`, resource.ErrNotFound}, {`{"images":[{"id":"one","name":"Parent"},{"id":"two","name":"Parent"}]}`, resource.ErrAmbiguous}} {
		calls := 0
		v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, tc.body), nil })).SetImageProperties(context.Background(), resource.Name("Parent"))
		if v != nil || !errors.Is(e, tc.cause) || calls != 1 {
			t.Fatal(v, e, calls)
		}
	}
	for _, field := range []string{"endpoint", "version", "provider"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			client.ResourceBase = client.Endpoint
			v, e := New(client).UpdateImage(context.Background(), resource.Name("Parent"), func(*UpdateImageOpts) error {
				switch field {
				case "endpoint":
					client.Endpoint = "https://glance.example/changed/"
				case "version":
					client.Microversion = "changed"
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				return nil
			})
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(v, e, calls)
			}
		})
	}
	t.Run("accepted source drift", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &taskCoreBody{reader: &taskCoreReader{body: "{}", err: io.EOF, action: func() { client.Endpoint = "https://glance.example/changed/" }}}
		client = taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 200, body), nil })
		client.ResourceBase = client.Endpoint
		v, e := New(client).SetImageProperties(context.Background(), resource.ID("id"))
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatal(v, e, body.closes)
		}
		taskCoreProof(t, e, 200, "{}")
	})
}

func TestUpdateImageCoreAcceptedFailureEvidence(t *testing.T) {
	for _, wire := range []string{"", `null`, `[]`, `{"name":42}`, `{"tags":[null]}`, `{"size":1.5}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		body := &taskCoreBody{reader: strings.NewReader(wire)}
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 200, body), nil })).UpdateImage(context.Background(), resource.ID("id"))
		if value != nil || err == nil || body.closes != 1 {
			t.Fatal(wire, value, err, body.closes)
		}
		taskCoreProof(t, err, 200, wire)
	}
	readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("custom cancel")
	ctx, cancel := context.WithCancelCause(context.Background())
	body := &taskCoreBody{reader: &taskCoreReader{body: "prefix", err: readErr, action: func() { cancel(cause) }}, closeErr: closeErr}
	calls := 0
	v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 200, body), nil })).SetImageProperties(ctx, resource.ID("id"))
	if v != nil || !errors.Is(e, readErr) || !errors.Is(e, closeErr) || !errors.Is(e, cause) || !errors.Is(e, context.Canceled) || calls != 1 || body.closes != 1 {
		t.Fatal(v, e, calls, body.closes)
	}
	taskCoreProof(t, e, 200, "prefix")
	for _, code := range []int{201, 202, 204, 206, 403, 404, 409, 413, 415, 429, 500} {
		v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, code, "native"), nil })).UpdateImage(context.Background(), resource.ID("id"))
		var proof *resource.ResponseError
		if v != nil || !gophercloud.ResponseCodeIs(e, code) || errors.As(e, &proof) {
			t.Fatal(code, v, e)
		}
	}
}

func TestUpdateImageCoreNativePrebodyOwnership(t *testing.T) {
	t.Run("configured retry retains body and explicit advanced headers", func(t *testing.T) {
		calls := 0
		var first string
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			_, raw := updateCorePayload(t, req)
			if calls == 1 {
				first = raw
				return taskCoreJSON(req, 503, "retry"), nil
			}
			if raw != first || req.Header.Get("Accept") != "application/advanced" || req.Method != "PATCH" || req.URL.Path != "/reverse/glance/v2/images/id" {
				t.Fatal(raw, first, req.Header, req.URL)
			}
			return taskCoreJSON(req, 200, "{}"), nil
		})
		client.ProviderClient.RetryFunc = func(ctx context.Context, m, u string, o *gophercloud.RequestOpts, e error, n uint) error {
			o.MoreHeaders["Accept"] = "application/advanced"
			return nil
		}
		v, e := New(client).UpdateImage(context.Background(), resource.ID("id"), WithUpdateImageField("x", json.Number("900719925474099312345")))
		if v == nil || e != nil || calls != 2 {
			t.Fatal(v, e, calls)
		}
	})
	t.Run("body replacement stopped before retry", func(t *testing.T) {
		calls := 0
		cause := errors.New("hook cause")
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 503, "retry"), nil })
		client.ProviderClient.RetryFunc = func(ctx context.Context, m, u string, o *gophercloud.RequestOpts, e error, n uint) error {
			o.JSONBody = json.RawMessage("[]")
			return cause
		}
		v, e := New(client).SetImageProperties(context.Background(), resource.ID("id"), WithSetImagePropertiesProperty("x", 1))
		if v != nil || !errors.Is(e, cause) || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls != 1 {
			t.Fatal(v, e, calls)
		}
	})
	t.Run("expanded status remains outside original200", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 503, "retry"), nil
			}
			return taskCoreJSON(req, 202, "expanded"), nil
		})
		client.ProviderClient.RetryFunc = func(ctx context.Context, m, u string, o *gophercloud.RequestOpts, e error, n uint) error {
			o.OkCodes = append(o.OkCodes, 202)
			return nil
		}
		v, e := New(client).UpdateImage(context.Background(), resource.ID("id"))
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "expanded" || calls != 2 {
			t.Fatal(v, e, native, calls)
		}
	})
}
