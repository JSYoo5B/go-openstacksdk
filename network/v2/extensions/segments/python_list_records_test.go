package segments_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	sdk "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func listRecordsFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*sdk.API, *int) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new(int)
	cloud.Mux.HandleFunc("/neutron/v2.0/", func(w http.ResponseWriter, r *http.Request) {
		*calls++
		w.Header().Set("X-Request-Id", "list-actual")
		reply(w, r)
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return sdk.New(client), calls
}
func TestPythonOwnedListDescriptorsAndIndependentWire(t *testing.T) {
	const raw = `{"segments":[{"id":"keep","name":"segment","segmentation_id":"٣","x_unknown":9007199254740993}]}`
	api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/segments" || r.URL.RawQuery != "" || (r.Body != nil && r.Body != http.NoBody) || r.Header.Get("Accept") != "application/json" {
			t.Error(r.Method, r.URL, r.Header)
		}
		testcloud.JSON(w, 203, raw)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || *calls != 1 {
		t.Fatal(rows, err, *calls)
	}
	row := rows[0]
	for key, want := range map[string]string{"segmentation_id": "3"} {
		if string(row.Resource.Body[key]) != want {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if string(row.Resource.Body["x_unknown"]) != "9007199254740993" || string(row.Wire.Body["x_unknown"]) != "9007199254740993" || row.StatusCode != 203 || string(row.Envelope) != raw || row.Header.Get("X-Request-Id") != "list-actual" || row.Resource.Body["tags"] != nil || row.Resource.Body["revision_number"] != nil {
		t.Fatal(row)
	}
	row.Resource.Body["name"][1] = 'X'
	row.Header.Set("X-Request-Id", "consumer")
	if string(row.Wire.Body["name"]) == string(row.Resource.Body["name"]) || string(row.Envelope) != raw || row.Wire.Header.Get("X-Request-Id") != "list-actual" {
		t.Fatal(row)
	}
}
func TestPythonOwnedListServerQueriesAndLocalBodyFilters(t *testing.T) {
	api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("network_type") != "vlan" || r.URL.Query().Has("id") || r.URL.Query().Has("unknown") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"segments":[{"id":"skip","name":"segment","segmentation_id":"٣","x_unknown":9007199254740993},{"id":"keep","name":"segment","segmentation_id":"٣","x_unknown":9007199254740993}]}`)
	})
	rows, err := api.AllRecords(context.Background(), sdk.WithRecordListQuery("network_type", "vlan"), sdk.WithRecordListQuery("id", json.RawMessage(`"keep"`)), sdk.WithRecordListQuery("unknown", func() {}))
	if err != nil || len(rows) != 1 || *calls != 1 || string(rows[0].Resource.Body["id"]) != `"keep"` {
		t.Fatal(rows, err, *calls)
	}
}
func TestPythonOwnedListPagingLinksRawCapsAndEarlyBreak(t *testing.T) {
	for _, kind := range []string{"body", "http", "next"} {
		t.Run(kind, func(t *testing.T) {
			api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("marker") == "tail" {
					testcloud.JSON(w, 200, `{"segments":[{"id":"two"}]}`)
					return
				}
				suffix := ``
				if kind == "body" {
					suffix = `,"segments_links":[{"rel":"next","href":"?marker=tail"}]`
				}
				if kind == "next" {
					suffix = `,"next":"?marker=tail"`
				}
				if kind == "http" {
					w.Header().Set("Link", `<?marker=tail>; rel="next"`)
				}
				testcloud.JSON(w, 200, `{"segments":[{"id":"one"}]`+suffix+`}`)
			})
			rows, err := api.AllRecords(context.Background())
			if err != nil || len(rows) != 2 || *calls != 2 {
				t.Fatal(rows, err, *calls)
			}
			before := *calls
			for row, err := range api.ListRecords(context.Background()) {
				if err != nil || row == nil {
					t.Fatal(row, err)
				}
				break
			}
			if *calls != before+1 {
				t.Fatal(*calls)
			}
		})
	}
	api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"segments":[{"id":"skip"},{"id":"keep"}],"next":"?marker=tail"}`)
	})
	rows, err := api.AllRecords(context.Background(), sdk.WithRecordListMaxItems(1))
	if err != nil || len(rows) != 1 || *calls != 1 {
		t.Fatal(rows, err, *calls)
	}
}
func TestPythonOwnedListProxyKeepsGeneratorAndExpressionsUnread(t *testing.T) {
	api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"segments":[{"id":"one"}]}`) })
	for _, expression := range []string{"[0]", "@", "to_array(@)"} {
		result, err := api.ProxyListRecords(context.Background(), sdk.WithRecordListJMESPath(expression))
		if err != nil || result == nil || *calls != 0 {
			t.Fatal(result, err, *calls)
		}
		if expression == "[0]" {
			if string(result.JSON) != "null" {
				t.Fatal(result)
			}
		} else if result.JSON != nil {
			t.Fatal(result)
		}
	}
	if _, err := api.ProxyListRecords(context.Background(), sdk.WithRecordListJMESPath("length(@)")); err == nil || *calls != 0 {
		t.Fatal(err, *calls)
	}
	if _, err := api.AllRecords(context.Background(), sdk.WithRecordListJMESPath("@")); !errors.Is(err, resource.ErrInvalidOption) || *calls != 0 {
		t.Fatal(err, *calls)
	}
	result, err := api.ProxyListRecords(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	generator, ok := result.Value.(*sdk.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if *calls != 1 {
		t.Fatal(*calls)
	}
}
func TestPythonOwnedListOptionsSnapshotRestorationAndPreflight(t *testing.T) {
	headers := map[string]string{"X-Owned": "original"}
	raw := json.RawMessage(`"vlan"`)
	option := sdk.WithRecordListOpts(sdk.RecordListOpts{Headers: headers, Query: map[string]json.RawMessage{"network_type": raw}})
	headers["X-Owned"] = "changed"
	raw[1] = 'X'
	api, calls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Owned") != "original" || r.URL.Query().Get("network_type") != "vlan" {
			t.Error(r.Header, r.URL)
		}
		testcloud.JSON(w, 200, `{"segments":[]}`)
	})
	if _, err := api.AllRecords(context.Background(), option); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []sdk.RecordListOption{nil, sdk.WithRecordListQuery("session", func() {}), sdk.WithRecordListBasePath("https://foreign.invalid/path"), sdk.WithRecordListMaxItems(-1)} {
		if _, err := api.AllRecords(context.Background(), bad); err == nil || *calls != 1 {
			t.Fatal(err, *calls)
		}
	}
	restored, restoredCalls := listRecordsFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("network_type") != "restored" || r.Header.Get("X-Raw") != "proof" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"segments":[]}`)
	})
	if _, err := restored.AllRecords(context.Background(), sdk.WithRecordListQuery("network_type", "old"), sdk.WithRecordListConflictingAttrs(json.RawMessage(`{"network_type":"restored","headers":{"X-Raw":"proof"}}`))); err != nil || *restoredCalls != 1 {
		t.Fatal(err, *restoredCalls)
	}
}
func TestPythonOwnedListPhysicalErrorsKeepNativeProofAndNoReplay(t *testing.T) {
	for _, code := range []int{200, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/neutron/v2.0/")
			calls, closed, retries := 0, 0, 0
			readErr, closeErr := errors.New("list read"), errors.New("list close")
			client.ProviderClient.HTTPClient.Transport = listRecordsTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"fault-proof"}}, Body: &listRecordsFaultBody{Reader: strings.NewReader(`{"segments":[{"id":"partial"}]}`), readErr: readErr, closeErr: closeErr, closed: &closed}}, nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			rows, err := sdk.New(client).AllRecords(context.Background())
			if rows != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || calls != 1 || closed != 1 || retries != 0 {
				t.Fatal(rows, err, calls, closed, retries)
			}
			if code < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "fault-proof" {
					t.Fatal(err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "fault-proof" {
					t.Fatal(err, native)
				}
			}
		})
	}
}

type listRecordsTransport func(*http.Request) (*http.Response, error)

func (f listRecordsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil {
		response.Request = r
	}
	return response, err
}

type listRecordsFaultBody struct {
	io.Reader
	readErr, closeErr error
	closed            *int
}

func (b *listRecordsFaultBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if n > 0 {
		return n, b.readErr
	}
	return n, err
}
func (b *listRecordsFaultBody) Close() error { *b.closed++; return b.closeErr }
