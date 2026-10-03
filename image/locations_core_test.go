package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestImageLocationsCoreFixedRoutesAndPassiveData(t *testing.T) {
	const literalURL = " cinder://store/中文%2F?next=/images/elsewhere#fragment "
	for _, supplied := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "literal validation"}[supplied], func(t *testing.T) {
			raw := []byte{'a', 'c', 'k', 255}
			body := &deleteCoreBody{reader: bytes.NewReader(raw)}
			var calls int
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed-%E4%B8%AD%E6%96%87/locations" || req.URL.RawQuery != "" || req.Header.Get("X-Extra") != "owned" {
					t.Fatal("request changed", req.Method, req.URL, req.Header)
				}
				var fields map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				var gotURL string
				var validation map[string]string
				if len(fields) != 2 || json.Unmarshal(fields["url"], &gotURL) != nil || gotURL != literalURL || json.Unmarshal(fields["validation_data"], &validation) != nil || validation == nil {
					t.Fatal("incorrect flat request", string(fields["url"]), string(fields["validation_data"]))
				}
				want := map[string]string{}
				if supplied {
					want = map[string]string{"os_hash_algo": " Future-Algorithm ", "os_hash_value": " Non-Hex AA "}
				}
				if !reflect.DeepEqual(validation, want) {
					t.Fatal("validation normalized", validation)
				}
				return deleteCoreHTTP(202, body, http.Header{"Location": {"https://foreign.test/decoy"}}), nil
			})
			options := []AddImageLocationOption{WithAddImageLocationHeader("X-Extra", "owned")}
			if supplied {
				options = append(options, WithImageLocationValidation(" Future-Algorithm ", " Non-Hex AA "))
			}
			ack, err := New(client).AddImageLocation(context.Background(), resource.ID("fixed-中文"), literalURL, options...)
			if err != nil || ack == nil || ack.ImageID != "fixed-中文" || ack.URL != literalURL || ack.StatusCode != 202 || !bytes.Equal(ack.Body, raw) || calls != 1 || body.closes != 1 {
				t.Fatalf("ack=%+v err=%v calls=%d closes=%d", ack, err, calls, body.closes)
			}
		})
	}

	raw := []byte(`[{"url":"rbd://literal/%2F?next=other#fragment","metadata":{"store":"fast","large":9007199254740993,"null":null},"id":"decoy","next":"https://foreign.test"},{"url":null,"metadata":null},{"URL":"decoy","Metadata":{},"url":"","metadata":{}}]`)
	var calls int
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images/fixed/locations" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatal("GET changed", req.Method, req.URL, req.Body)
		}
		return deleteCoreHTTP(200, io.NopCloser(bytes.NewReader(raw)), http.Header{"Link": {`<https://foreign.test/page>; rel="next"`}}), nil
	})
	result, err := New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
	if err != nil || result == nil || result.ImageID != "fixed" || result.StatusCode != 200 || calls != 1 || len(result.Locations) != 3 || !bytes.Equal(result.Body, raw) {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
	}
	first := result.Locations[0]
	if first.URL == nil || *first.URL != "rbd://literal/%2F?next=other#fragment" || string(first.Metadata["large"]) != "9007199254740993" || string(first.Body["id"]) != `"decoy"` || result.Locations[1].URL != nil || result.Locations[1].Metadata != nil || result.Locations[2].URL == nil || *result.Locations[2].URL != "" || result.Locations[2].Metadata == nil {
		t.Fatal("passive fields lost", result.Locations)
	}
	metadataBody := append([]byte(nil), first.Body["metadata"]...)
	first.Metadata["large"][0] = '1'
	if !bytes.Equal(first.Body["metadata"], metadataBody) || !bytes.Equal(result.Body, raw) {
		t.Fatal("metadata aliases row or root bytes")
	}
	result.Body[0] = '!'
	first.Body["url"][1] = '!'
	if *first.URL != "rbd://literal/%2F?next=other#fragment" || raw[0] != '[' {
		t.Fatal("typed URL or wire bytes alias raw result")
	}
}

func TestImageLocationsCoreStrictArrayAndCanonicalSchema(t *testing.T) {
	bad := [][]byte{nil, []byte(`null`), []byte(`{}`), []byte(`{"locations":[]}`), []byte(`1`), []byte(`[null]`), []byte(`[[]]`), []byte(`["url"]`), []byte(`[{"url":false}]`), []byte(`[{"url":1}]`), []byte(`[{"metadata":[]}]`), []byte(`[{"metadata":"bad"}]`), []byte(`[{},`), []byte("[{\"unknown\":\"bad\xff\"}]")}
	for index, raw := range bad {
		t.Run(string(rune('A'+index)), func(t *testing.T) {
			body := &deleteCoreBody{reader: bytes.NewReader(raw)}
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
				return deleteCoreHTTP(200, body, http.Header{"X-Actual": {"owned"}}), nil
			})
			result, err := New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
			var proof *resource.ResponseError
			if result != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Actual") != "owned" || body.closes != 1 {
				t.Fatalf("raw=%q result=%+v err=%v proof=%+v closes=%d", raw, result, err, proof, body.closes)
			}
		})
	}
	for _, raw := range []string{`[]`, `[{}]`, `[{"url":null,"metadata":null}]`, `[{"url":"","metadata":{"false":false,"zero":0,"fraction":1.00000000000000001,"array":[null]}}]`} {
		rows, err := decodeImageLocations([]byte(raw))
		if err != nil || rows == nil {
			t.Fatalf("valid raw=%s rows=%v err=%v", raw, rows, err)
		}
	}
	value := ImageLocation{}
	if err := json.Unmarshal([]byte(`{"url":"before","metadata":{"store":"owned"}}`), &value); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"url":"changed","metadata":true}`), &value); err == nil || *value.URL != "before" || string(value.Metadata["store"]) != `"owned"` {
		t.Fatal("failed decode partially updated model", value, err)
	}
}

func TestImageLocationsCoreExactNameAndPreparedPayload(t *testing.T) {
	var retained *AddImageLocationOpts
	var client *gophercloud.ServiceClient
	var calls int
	client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Snapshot") != "captured" || req.Header.Get("X-Extra") != "owned" || req.Header.Get("X-Auth-Token") != "latest" {
			t.Fatal("prepared headers/auth changed", req.Header)
		}
		switch calls {
		case 1:
			retained.Headers["X-Extra"] = "late"
			retained.ValidationData.OSHashValue = "late"
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"near","name":"Exact"}],"next":"/v2/images?marker=second&name=exact"}`)), nil), nil
		case 2:
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"chosen","name":"exact"}]}`)), nil), nil
		case 3:
			if req.Method != http.MethodPost || req.URL.Path != "/reverse/glance/v2/images/chosen/locations" {
				t.Fatal("chosen route changed", req.Method, req.URL)
			}
			var fields struct {
				URL            string            `json:"url"`
				ValidationData map[string]string `json:"validation_data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&fields); err != nil || fields.URL != "swift://literal" || fields.ValidationData["os_hash_value"] != "owned" {
				t.Fatal("prepared body changed", fields, err)
			}
			return deleteCoreHTTP(202, http.NoBody, nil), nil
		default:
			t.Fatal("extra request", req.URL)
			return nil, nil
		}
	})
	client.MoreHeaders = map[string]string{"X-Snapshot": "captured"}
	ack, err := New(client).AddImageLocation(context.Background(), resource.Name("exact"), "swift://literal", func(config *AddImageLocationOpts) error {
		retained = config
		config.Headers["X-Extra"] = "owned"
		config.ValidationData = &ImageLocationValidation{OSHashAlgo: "algo", OSHashValue: "owned"}
		client.ResourceBase = "https://example.test/later/"
		client.MoreHeaders["X-Snapshot"] = "late"
		client.ProviderClient.SetToken("latest")
		return nil
	})
	if err != nil || ack == nil || ack.ImageID != "chosen" || calls != 3 {
		t.Fatalf("ack=%+v err=%v calls=%d", ack, err, calls)
	}

	for _, failed := range []bool{false, true} {
		calls := 0
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path != "/reverse/glance/v2/images" {
				t.Fatal("strict Name lookup issued locations GET", req.URL)
			}
			if calls == 1 && failed {
				return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"found","name":"exact"}],"next":"/v2/images?marker=next"}`)), nil), nil
			}
			if failed {
				return deleteCoreHTTP(503, http.NoBody, nil), nil
			}
			return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[]}`)), nil), nil
		})
		result, err := New(client).GetImageLocations(context.Background(), resource.Name("exact"))
		if result != nil || err == nil || calls != map[bool]int{false: 1, true: 2}[failed] {
			t.Fatalf("lateFailure=%v result=%+v err=%v calls=%d", failed, result, err, calls)
		}
	}
}

func TestImageLocationsCoreCompletePreflight(t *testing.T) {
	for _, badURL := range []string{"", "bad\xff"} {
		var calls, options int
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			return deleteCoreHTTP(202, http.NoBody, nil), nil
		})
		result, err := New(client).AddImageLocation(context.Background(), resource.Name("exact"), badURL, func(*AddImageLocationOpts) error { options++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || options != 0 {
			t.Fatalf("url=%q result=%v err=%v calls=%d options=%d", badURL, result, err, calls, options)
		}
	}
	for _, pair := range []*ImageLocationValidation{{}, {OSHashAlgo: "algo"}, {OSHashValue: "value"}, {OSHashAlgo: "bad\xff", OSHashValue: "value"}, {OSHashAlgo: "algo", OSHashValue: "bad\xff"}} {
		calls := 0
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			return deleteCoreHTTP(202, http.NoBody, nil), nil
		})
		result, err := New(client).AddImageLocation(context.Background(), resource.Name("exact"), "literal", WithAddImageLocationOpts(AddImageLocationOpts{ValidationData: pair}))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("pair=%+v result=%v err=%v calls=%d", pair, result, err, calls)
		}
	}
	custom := errors.New("caller cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	var calls int
	client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
		calls++
		return deleteCoreHTTP(200, http.NoBody, nil), nil
	})
	for _, selected := range []context.Context{nil, ctx} {
		result, err := New(client).GetImageLocations(selected, resource.ID("fixed"), nil)
		if result != nil || err == nil || calls != 0 || selected != nil && (!errors.Is(err, custom) || !errors.Is(err, context.Canceled)) {
			t.Fatalf("result=%v err=%v calls=%d", result, err, calls)
		}
	}
	for _, mutate := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://foreign.test/v2/" }} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("retargeted request"); return nil, nil })
		result, err := New(client).GetImageLocations(context.Background(), resource.ID("fixed"), func(*GetImageLocationsOpts) error { mutate(client); return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("source change result=%v err=%v", result, err)
		}
	}
}

func TestImageLocationsCoreAcceptedFailuresKeepOwnedProof(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failure"), errors.New("close failure"), errors.New("caller stopped")
	for _, add := range []bool{false, true} {
		for _, mode := range []string{"read", "close", "all", "context"} {
			t.Run(map[bool]string{false: "Get", true: "Add"}[add]+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := []byte(`[]`)
				header := http.Header{"X-Actual": {"before"}}
				body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) {
					header.Set("X-Actual", "read mutation")
					if mode == "all" || mode == "context" {
						cancel(customCause)
					}
					if mode == "read" || mode == "all" {
						return copy(buffer, raw), readCause
					}
					return copy(buffer, raw), io.EOF
				})}
				if mode == "close" || mode == "all" {
					body.closeErr = closeCause
				}
				var calls int
				client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
					calls++
					return deleteCoreHTTP(map[bool]int{false: 200, true: 202}[add], body, header), nil
				})
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					t.Fatal("accepted body replay")
					return nil
				}
				var ack *AddImageLocationResult
				var get *ImageLocationsResult
				var err error
				if add {
					ack, err = New(client).AddImageLocation(ctx, resource.ID("fixed"), "literal")
				} else {
					get, err = New(client).GetImageLocations(ctx, resource.ID("fixed"))
				}
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != map[bool]int{false: 200, true: 202}[add] || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Actual") != "before" || get != nil || add && ack == nil || calls != 1 || body.closes != 1 {
					t.Fatalf("ack=%+v get=%+v err=%v proof=%+v calls=%d closes=%d", ack, get, err, proof, calls, body.closes)
				}
				if mode == "read" || mode == "all" {
					if !errors.Is(err, readCause) {
						t.Fatal("read cause lost", err)
					}
				}
				if mode == "close" || mode == "all" {
					if !errors.Is(err, closeCause) {
						t.Fatal("close cause lost", err)
					}
				}
				if mode == "all" || mode == "context" {
					if !errors.Is(err, customCause) || !errors.Is(err, context.Canceled) {
						t.Fatal("context causes lost", err)
					}
				}
				if ack != nil {
					proof.Body[0] = '!'
					proof.Header.Set("X-Actual", "proof mutation")
					if !bytes.Equal(ack.Body, raw) || ack.Header.Get("X-Actual") != "before" {
						t.Fatal("ack aliases error proof")
					}
				}
			})
		}
	}
}

func TestImageLocationsCoreStrictStatusesAndRetryOwnership(t *testing.T) {
	for _, add := range []bool{false, true} {
		for _, status := range []int{200, 202, 204, 404, 409} {
			if status == map[bool]int{false: 200, true: 202}[add] {
				continue
			}
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) { return deleteCoreHTTP(status, http.NoBody, nil), nil })
			var err error
			if add {
				result, failure := New(client).AddImageLocation(context.Background(), resource.ID("fixed"), "literal")
				if result != nil {
					t.Fatal("unexpected Add status accepted", status)
				}
				err = failure
			} else {
				result, failure := New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
				if result != nil {
					t.Fatal("unexpected Get status accepted", status)
				}
				err = failure
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{map[bool]int{false: 200, true: 202}[add]}) {
				t.Fatalf("status=%d err=%v native=%+v", status, err, native)
			}
		}
	}
	for _, add := range []bool{false, true} {
		calls := 0
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			return deleteCoreHTTP(503, http.NoBody, nil), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
			if add {
				options.JSONBody = json.RawMessage(`{"url":"retargeted","validation_data":{}}`)
			} else {
				options.JSONBody = json.RawMessage(`null`)
			}
			return nil
		}
		var err error
		if add {
			_, err = New(client).AddImageLocation(context.Background(), resource.ID("fixed"), "literal")
		} else {
			_, err = New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 {
			t.Fatalf("ownership add=%v err=%v calls=%d", add, err, calls)
		}
	}
	var calls int
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return deleteCoreHTTP(503, http.NoBody, nil), nil
		}
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader("opaque unexpected body")), nil), nil
	})
	client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
		options.OkCodes = append(options.OkCodes, 200)
		return nil
	}
	ack, err := New(client).AddImageLocation(context.Background(), resource.ID("fixed"), "literal")
	var native gophercloud.ErrUnexpectedResponseCode
	if ack != nil || !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{202}) || string(native.Body) != "opaque unexpected body" || calls != 2 {
		t.Fatalf("expanded status ack=%v err=%v native=%+v calls=%d", ack, err, native, calls)
	}
}
