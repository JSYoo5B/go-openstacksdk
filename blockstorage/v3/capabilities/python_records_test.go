package capabilities_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderCapabilitiesWholeBodyProjectionAndEscapedHost(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/cinder/v3/capabilities/host%2Fbackend%25" || r.URL.RawQuery != "" {
			t.Error(r.Method, r.URL)
		}
		w.Header().Set("X-Request-Id", "capabilities-actual")
		testcloud.JSON(w, 203, `{"display_name":"backend","properties":false,"replication_targets":{"backend":"remote"},"x_unknown":9007199254740993}`)
	})
	api := capabilities.New(cloud.Client("volumev3", "/cinder/v3/"))
	row, err := api.GetRecord(context.Background(), "host/backend%")
	if err != nil || row == nil || row.Resource == nil || row.Wire == nil || calls != 1 || string(row.Resource.Body["id"]) != `"host/backend%"` || string(row.Resource.Body["properties"]) != "{}" || string(row.Resource.Body["replication_targets"]) != `[{"backend":"remote"}]` || row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "9007199254740993" || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "capabilities-actual" {
		t.Fatal(row, err, calls)
	}
}
func TestPythonCinderCapabilitiesOwnedHeadersAndPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Owned") != "original" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{}`)
	})
	api := capabilities.New(cloud.Client("volumev3", "/cinder/v3/"))
	headers := map[string]string{"X-Owned": "original"}
	option := capabilities.WithHeaders(headers)
	headers["X-Owned"] = "changed"
	if _, err := api.GetRecord(context.Background(), "host", option); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "..", "bad\n"} {
		if _, err := api.GetRecord(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.GetRecord(context.Background(), "host", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestPythonCinderCapabilities404AndMalformedBodyPreserveProof(t *testing.T) {
	for _, code := range []int{203, 404} {
		cloud := testcloud.New(t)
		calls := 0
		raw := `{"properties":"²"}`
		if code == 404 {
			raw = `{"error":"missing backend"}`
		}
		cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("X-Request-Id", "actual")
			testcloud.JSON(w, code, raw)
		})
		row, err := capabilities.New(cloud.Client("volumev3", "/cinder/v3/")).GetRecord(context.Background(), "host")
		if code == 404 {
			var native gophercloud.ErrUnexpectedResponseCode
			if row != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual" {
				t.Fatal(row, err, native)
			}
		} else if err != nil || row == nil {
			t.Fatal(row, err)
		}
		if calls != 1 {
			t.Fatal(calls)
		}
	}
}

func TestPythonCinderCapabilitiesPhysical404KeepsReadCloseWithoutReplay(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("volumev3", "/cinder/v3/")
	readCause, closeCause := errors.New("capabilities read"), errors.New("capabilities close")
	calls, closed, retries := 0, 0, 0
	client.ProviderClient.HTTPClient.Transport = capabilityTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 404, Header: http.Header{"X-Request-Id": {"physical-capability"}}, Body: &capabilityFaultBody{readCause: readCause, closeCause: closeCause, closed: &closed}}, nil
	})
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	row, err := capabilities.New(client).GetRecord(context.Background(), "host")
	var native gophercloud.ErrUnexpectedResponseCode
	if row != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "partial capability" || native.ResponseHeader.Get("X-Request-Id") != "physical-capability" || calls != 1 || closed != 1 || retries != 0 {
		t.Fatal(row, err, native, calls, closed, retries)
	}
}

type capabilityTransport func(*http.Request) (*http.Response, error)

func (f capabilityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil {
		response.Request = r
	}
	return response, err
}

type capabilityFaultBody struct {
	readCause, closeCause error
	closed                *int
	read                  bool
}

func (b *capabilityFaultBody) Read(p []byte) (int, error) {
	if b.read {
		return 0, b.readCause
	}
	b.read = true
	return copy(p, "partial capability"), b.readCause
}
func (b *capabilityFaultBody) Close() error { *b.closed++; return b.closeCause }
