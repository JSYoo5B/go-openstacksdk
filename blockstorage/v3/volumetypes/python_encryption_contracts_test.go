package volumetypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonEncryptionAttributesPreservePresenceAndResponseTypes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []volumetypes.EncryptionAttributeOption
		body    string
	}{
		{"empty", nil, `{"encryption":{}}`},
		{"partial", []volumetypes.EncryptionAttributeOption{volumetypes.WithEncryptionAttribute("provider", "luks")}, `{"encryption":{"provider":"luks"}}`},
		{"null zero false empty", []volumetypes.EncryptionAttributeOption{volumetypes.WithEncryptionAttribute("cipher", nil), volumetypes.WithEncryptionAttribute("key_size", 0), volumetypes.WithEncryptionAttribute("deleted", false), volumetypes.WithEncryptionAttribute("control_location", "")}, `{"encryption":{"cipher":null,"control_location":"","deleted":false,"key_size":0}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response {
				return pythonTypeWire(200, `{"encryption":{"encryption_id":"e-1","key_size":"0256","provider":{"vendor":"raw"},"unknown":9007199254740993}}`)
			})
			value, err := api.CreateEncryptionWithAttributes(context.Background(), "vt-1", tc.options...)
			if err != nil || value == nil || string(value.Resource.Body["key_size"]) != `"0256"` || string(value.Resource.Body["provider"]) != `{"vendor":"raw"}` || string(value.Wire.Body["unknown"]) != "9007199254740993" || string(value.Resource.Body["id"]) != `"e-1"` {
				t.Fatal(value, err)
			}
			if _, exists := value.Resource.Body["unknown"]; exists {
				t.Fatal("unknown projected to Source view")
			}
			if len(*calls) != 1 || (*calls)[0].body != tc.body || (*calls)[0].path != pythonTypeBase+"/vt-1/encryption" {
				t.Fatal(*calls)
			}
			if value.StatusCode != 200 || string(value.Envelope) == "" || value.Header.Get("Content-Type") == "" {
				t.Fatal(value)
			}
		})
	}
}
func TestPythonEncryptionOptionsSnapshotInputsAndIgnoreUnknown(t *testing.T) {
	raw := json.RawMessage(`{"nested":[0,null]}`)
	fields := map[string]json.RawMessage{"provider": raw, "extension": json.RawMessage(`false`)}
	headers := map[string]string{"X-Profile": "before"}
	option := volumetypes.WithEncryptionAttributeOpts(volumetypes.EncryptionAttributeOpts{Fields: fields, Headers: headers})
	raw[0] = '['
	fields["provider"] = json.RawMessage(`"changed"`)
	headers["X-Profile"] = "after"
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Profile") != "before" {
			t.Fatal(req.Header)
		}
		return pythonTypeWire(204, "")
	})
	value, err := api.CreateEncryptionWithAttributes(context.Background(), "vt-1", option)
	if err != nil || value == nil || string(value.Resource.Body["provider"]) != `{"nested":[0,null]}` || value.StatusCode != 204 {
		t.Fatal(value, err)
	}
	if len(*calls) != 1 || (*calls)[0].body != `{"encryption":{"provider":{"nested":[0,null]}}}` {
		t.Fatal(*calls)
	}
}
func TestPythonEncryptionUpdateFetchesAndSendsOnlyDirtyFields(t *testing.T) {
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonTypeWire(200, pythonTypeEncryption)
		}
		return pythonTypeWire(200, `{"encryption":{"key_size":512}}`)
	})
	value, err := api.UpdateEncryptionFromType(context.Background(), "vt-1", volumetypes.WithEncryptionAttribute("key_size", 512), volumetypes.WithEncryptionAttribute("cipher", nil))
	if err != nil || value == nil || string(value.Resource.Body["provider"]) != `"luks"` || string(value.Resource.Body["cipher"]) != "null" {
		t.Fatal(value, err)
	}
	want := []pythonTypeCall{{http.MethodGet, pythonTypeBase + "/vt-1/encryption", "", ""}, {http.MethodPut, pythonTypeBase + "/vt-1/encryption/enc-1", "", `{"encryption":{"cipher":null,"key_size":512}}`}}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatal(*calls)
	}
}
func TestPythonEncryptionUpdateEqualEmptyAndIDOnlySkipPut(t *testing.T) {
	for _, options := range [][]volumetypes.EncryptionAttributeOption{nil, {volumetypes.WithEncryptionAttribute("key_size", 256), volumetypes.WithEncryptionAttribute("deleted", 0)}, {volumetypes.WithEncryptionAttribute("id", "other")}} {
		api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response { return pythonTypeWire(200, pythonTypeEncryption) })
		value, err := api.UpdateEncryptionFromType(context.Background(), "vt-1", options...)
		if err != nil || value == nil || len(*calls) != 1 || (*calls)[0].method != http.MethodGet {
			t.Fatal(value, err, *calls)
		}
	}
}
func TestPythonEncryptionRecordUpdateSnapshotsCallerAndBody(t *testing.T) {
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return pythonTypeWire(200, pythonTypeEncryption)
		}
		return pythonTypeWire(204, "")
	})
	value, err := api.GetEncryptionRecord(context.Background(), "vt-1")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := api.UpdateEncryptionRecord(context.Background(), value, volumetypes.WithEncryptionAttribute("key_size", 0))
	if err != nil || updated == nil || string(updated.Resource.Body["key_size"]) != "0" || string(value.Resource.Body["key_size"]) != "256" || len(*calls) != 2 || (*calls)[1].body != `{"encryption":{"key_size":0}}` {
		t.Fatal(updated, value, err, *calls)
	}
	updated.Resource.Body["provider"][0] = '['
	if string(value.Resource.Body["provider"]) != `"luks"` {
		t.Fatal("record shares caller bytes")
	}
}
func TestPythonEncryptionPreflightGuardsOptionsAndMalformedJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []volumetypes.EncryptionAttributeOption
	}{
		{"nil", []volumetypes.EncryptionAttributeOption{nil}},
		{"malformed", []volumetypes.EncryptionAttributeOption{volumetypes.WithEncryptionAttributeOpts(volumetypes.EncryptionAttributeOpts{Fields: map[string]json.RawMessage{"provider": json.RawMessage(`{"bad"`)}})}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response { t.Fatal("HTTP before preflight"); return nil })
			_, err := api.CreateEncryptionWithAttributes(context.Background(), "vt-1", tc.options...)
			if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
		})
	}
	api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response { t.Fatal("changed source reached HTTP"); return nil })
	_, err := api.CreateEncryptionWithAttributes(context.Background(), "vt-1", func(*volumetypes.EncryptionAttributeOpts) error {
		api.RawClient().Microversion = "3.50"
		return errors.New("option failed")
	})
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

type encryptionBrokenBody struct {
	io.Reader
	read, close error
}

func (b encryptionBrokenBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF {
		return n, b.read
	}
	return n, err
}
func (b encryptionBrokenBody) Close() error { return b.close }
func TestPythonEncryptionAcceptedPhysicalFailuresKeepEvidence(t *testing.T) {
	readFailure, closeFailure := errors.New("encryption read"), errors.New("encryption close")
	api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response {
		response := pythonTypeWire(200, "")
		response.Header.Set("X-Request-Id", "encryption-proof")
		response.Body = encryptionBrokenBody{Reader: strings.NewReader(`{"encryption":{"encryption_id":"e-1"}}`), read: readFailure, close: closeFailure}
		return response
	})
	value, err := api.CreateEncryptionWithAttributes(context.Background(), "vt-1", volumetypes.WithEncryptionAttribute("provider", "luks"))
	var proof *resource.ResponseError
	if !errors.Is(err, readFailure) || !errors.Is(err, closeFailure) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "encryption-proof" || len(proof.Body) == 0 || value == nil || len(*calls) != 1 {
		t.Fatal(value, err, proof, *calls)
	}
}
