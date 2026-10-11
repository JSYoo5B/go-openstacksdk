package addressgroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonAddressActionsRoutesInputsAndEmptyValues(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_group":{"id":"returned","name":false,"description":0,"addresses":"one","project_id":null,"tenant_id":"other","created_at":"not-native-time","x_new":99999999999999999999999,"self":"ignored"}}`)
	})
	values := []any{nil, []any{}, false, 0, map[string]any{"port_id": "child", "segmentation_id": 0, "extension": map[string]any{"null": nil, "false": false, "zero": 0, "empty": []any{}}}}
	for _, value := range values {
		record, err := api.AddAddressesRecord(context.Background(), "r1", value)
		if err != nil || record == nil {
			t.Fatal(record, err)
		}
	}
	if len(calls) != len(values) {
		t.Fatal(calls)
	}
	for index, value := range values {
		raw, _ := json.Marshal(map[string]any{"addresses": value})
		if calls[index].method != "PUT" || calls[index].path != "/neutron/v2.0/address-groups/r1/add_addresses" || calls[index].query != "" || calls[index].body != string(raw) {
			t.Fatal(index, calls)
		}
	}
	_, err := api.RemoveAddressesRecord(context.Background(), "r1", []any{})
	if err != nil || calls[len(calls)-1].path != "/neutron/v2.0/address-groups/r1/remove_addresses" || calls[len(calls)-1].body != `{"addresses":[]}` {
		t.Fatal(err, calls)
	}
	for _, value := range []any{nil, false, 0} {
		_, err := api.RemoveAddressesRecord(context.Background(), "r1", value)
		if err != nil {
			t.Fatal(value, err)
		}
	}
}
func TestPythonAddressActionsSourceProjectionAndActualWire(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		r := nativeGroupWire(200, `{"address_group":{"id":"returned","name":false,"description":0,"addresses":"one","project_id":null,"tenant_id":"other","created_at":"not-native-time","x_new":99999999999999999999999,"self":"ignored"}}`)
		r.Header.Set("X-Request-Id", "wire-proof")
		return r
	})
	seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"name": json.RawMessage(`"cached"`), "x_cached": json.RawMessage(`{"false":false}`)}}}
	record, err := api.AddAddressesRecord(context.Background(), "r1", []any{}, addressgroups.WithAddressActionSeed(seed))
	if err != nil || record == nil || string(record.Resource.Body["id"]) != `"returned"` || string(record.Resource.Body["name"]) != "false" || string(record.Resource.Body["project_id"]) != "null" || string(record.Resource.Body["tenant_id"]) != `"other"` || string(record.Resource.Body["x_cached"]) != `{"false":false}` || record.Header.Get("X-Request-Id") != "wire-proof" || record.StatusCode != 200 || record.Wire == nil || string(record.Wire.Body["created_at"]) != `"not-native-time"` || string(record.Resource.Body["addresses"]) != `["one"]` || string(record.Resource.Body["description"]) != "0" || string(record.Resource.Body["x_new"]) != "99999999999999999999999" || record.Resource.Body["self"] != nil {
		t.Fatal(record, err)
	}
	record.Resource.Body["name"] = json.RawMessage(`"consumer"`)
	record.Header.Set("X-Request-Id", "consumer")
	if string(record.Wire.Body["name"]) != "false" || record.Wire.Header.Get("X-Request-Id") != "wire-proof" || string(seed.Body["name"]) != `"cached"` {
		t.Fatal(record, seed)
	}
}
func TestPythonAddressActionsBodylessSuccessKeepsSourcePolicy(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(204, "") })
	record, err := api.AddAddressesRecord(context.Background(), "r1", nil)
	if err != nil || record == nil || record.Wire != nil || record.StatusCode != 204 || string(record.Resource.Body["id"]) != `"r1"` || string(record.Resource.Body["addresses"]) != "null" {
		t.Fatal(record, err)
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}
func TestPythonAddressActionsSnapshotsHeaderPolicyAndInvalidOptions(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(200, `{}`) })
	value := map[string]any{"false": false}
	headers := map[string]string{"X-Owned": "old"}
	headerOption := addressgroups.WithAddressActionHeaders(headers)
	headers["Authorization"] = "bad"
	seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"name": json.RawMessage(`"old"`)}}}
	seedOption := addressgroups.WithAddressActionSeed(seed)
	seed.Body["name"] = json.RawMessage(`"changed"`)
	var retained *addressgroups.AddressActionOpts
	record, err := api.AddAddressesRecord(context.Background(), "r1", value, headerOption, seedOption, func(v *addressgroups.AddressActionOpts) error { retained = v; value["false"] = true; return nil }, func(*addressgroups.AddressActionOpts) error {
		retained.Seed.Body["name"] = json.RawMessage(`"retained"`)
		retained.Headers["Authorization"] = "bad"
		return nil
	})
	if err != nil || record == nil || string(record.Resource.Body["name"]) != `"old"` || calls[0].body != `{"addresses":{"false":false}}` {
		t.Fatal(record, err, calls)
	}
	for _, test := range []struct {
		name string
		call func() error
	}{
		{"nil option", func() error { _, err := api.AddAddressesRecord(context.Background(), "r1", nil, nil); return err }},
		{"bad target", func() error { _, err := api.AddAddressesRecord(context.Background(), "..", nil); return err }},
		{"protected header", func() error {
			_, err := api.AddAddressesRecord(context.Background(), "r1", nil, addressgroups.WithAddressActionHeaders(map[string]string{"X-Auth-Token": "bad"}))
			return err
		}},
		{"bad seed", func() error {
			_, err := api.AddAddressesRecord(context.Background(), "r1", nil, addressgroups.WithAddressActionSeed(&resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"name": json.RawMessage(`invalid`)}}}))
			return err
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}
func TestPythonAddressActionsCanceledOrChangedOptionsAndMarshalAreTerminal(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(200, `{}`) })
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cause"), errors.New("option")
	_, err := api.AddAddressesRecord(ctx, "r1", nil, func(*addressgroups.AddressActionOpts) error { cancel(cause); return callback })
	if len(calls) != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) {
		t.Fatal(err, calls)
	}
	client := api.RawClient()
	original := client.ResourceBase
	defer func() { client.ResourceBase = original }()
	_, err = api.AddAddressesRecord(context.Background(), "r1", nil, func(*addressgroups.AddressActionOpts) error { client.ResourceBase += "changed/"; return callback })
	if len(calls) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callback) {
		t.Fatal(err, calls)
	}
	client.ResourceBase = original
	_, err = api.AddAddressesRecord(context.Background(), "r1", actionChangingJSON(func() { client.ResourceBase += "changed/" }))
	if len(calls) != 0 || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err, calls)
	}
}
func TestPythonAddressActionsPhysicalAcceptedAndRejectedErrorsRetainProof(t *testing.T) {
	for _, operation := range []string{"add", "remove"} {
		for _, status := range []int{200, 409, 503} {
			for _, kind := range []string{"read", "close", "read+close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, kind), func(t *testing.T) {
					var calls []nativeGroupCall
					api, cloud := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(200, `{}`) })
					attempts := 0
					readErr, closeErr, cancelErr := errors.New("read"), errors.New("close"), errors.New("cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					client := api.RawClient()
					original := client.ResourceBase
					defer func() { client.ResourceBase = original }()
					body := &actionFaultBody{data: []byte("actual body"), read: io.EOF}
					switch kind {
					case "read":
						body.read = readErr
					case "close":
						body.close = closeErr
					case "read+close":
						body.read = readErr
						body.close = closeErr
					case "cancel":
						body.onClose = func() { cancel(cancelErr) }
					case "source":
						body.onRead = func() { client.ResourceBase += "changed/" }
						body.onClose = func() { client.ResourceBase = original }
					}
					cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(*http.Request) (*http.Response, error) {
						attempts++
						return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"actual"}}, Body: body}, nil
					})
					var record *addressgroups.AddressActionRecord
					var err error
					if operation == "remove" {
						record, err = api.RemoveAddressesRecord(ctx, "r1", []any{})
					} else {
						record, err = api.AddAddressesRecord(ctx, "r1", []any{})
					}
					if attempts != 1 || err == nil {
						t.Fatal(record, err, attempts)
					}
					if status == 200 {
						var receipt *resource.ResponseError
						if record == nil || !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != "actual body" || receipt.Header.Get("X-Request-Id") != "actual" {
							t.Fatal(record, err, receipt)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != "actual body" || native.ResponseHeader.Get("X-Request-Id") != "actual" {
							t.Fatal(err, native)
						}
					}
					switch kind {
					case "read", "read+close":
						if !errors.Is(err, readErr) {
							t.Fatal(err)
						}
					case "cancel":
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelErr) {
							t.Fatal(err)
						}
					case "source":
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(err)
						}
					}
					if kind == "close" || kind == "read+close" {
						if !errors.Is(err, closeErr) {
							t.Fatal(err)
						}
					}
				})
			}
		}
	}
}
func TestPythonAddressActionsRetryMutationKeepsOriginalRejection(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(409, `{"conflict":"actual"}`) })
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.JSONBody = map[string]any{"addresses": []any{"changed"}}
		return nil
	}
	_, err := api.AddAddressesRecord(context.Background(), "r1", []any{})
	var native gophercloud.ErrUnexpectedResponseCode
	if len(calls) != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 409 || string(native.Body) != `{"conflict":"actual"}` {
		t.Fatal(err, native, calls)
	}
}
func TestPythonAddressActionsCleanSourceStatusPolicyAndEscapedID(t *testing.T) {
	var calls []nativeGroupCall
	status := 203
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(status, `{}`) })
	record, err := api.AddAddressesRecord(context.Background(), "folder/resource%", nil)
	if err != nil || record == nil || record.StatusCode != 203 || len(calls) != 1 {
		t.Fatal(record, err, calls)
	}
	// The response fixture records decoded paths; physical URL escapes exactly once.
	var escaped string
	api.RawClient().ProviderClient.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
		escaped = req.URL.EscapedPath()
		return nativeGroupWire(299, `{}`), nil
	})
	record, err = api.AddAddressesRecord(context.Background(), "folder/resource%", nil)
	if err != nil || record.StatusCode != 299 || escaped != "/neutron/v2.0/address-groups/folder%2Fresource%25/add_addresses" {
		t.Fatal(record, err, escaped)
	}
	status = 404
	api, _ = nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(status, `{"missing":"actual"}`) })
	_, err = api.AddAddressesRecord(context.Background(), "r1", nil)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 404 {
		t.Fatal(err, native)
	}
}

type actionFaultBody struct {
	data            []byte
	read, close     error
	onRead, onClose func()
}

func (b *actionFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
		}
		return n, nil
	}
	return 0, b.read
}
func (b *actionFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

type actionChangingJSON func()

func (f actionChangingJSON) MarshalJSON() ([]byte, error) { f(); return []byte(`[]`), nil }

var _ io.ReadCloser = (*actionFaultBody)(nil)
