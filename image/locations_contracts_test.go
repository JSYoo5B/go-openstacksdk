package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image"
	sdkimages "gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const locationsPrefix = "/reverse/locations/glance/v2/"
const locationsBase = "https://glance.invalid" + locationsPrefix
const locationsURL = "cinder://store/object1?literal=%2F#fragment"

type locationsTransport func(*http.Request) (*http.Response, error)

func (f locationsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type locationsBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *locationsBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type locationsReader func([]byte) (int, error)

func (f locationsReader) Read(p []byte) (int, error) { return f(p) }

func locationsWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"Content-Type": {"application/json"}, "X-Request-Id": {"actual-locations"},
	}, Body: body}
}

func locationsClient(transport locationsTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: locationsBase}
}

type locationsOptions struct {
	add []image.AddImageLocationOption
	get []image.GetImageLocationsOption
}

type locationsView struct {
	ImageID, URL string
	Locations    []*image.ImageLocation
	Body         []byte
	Header       http.Header
	StatusCode   int
}

func locationsCall(s *image.Service, ctx context.Context, add bool, ref resource.Ref, value string, options locationsOptions) (*locationsView, error) {
	if add {
		result, err := s.AddImageLocation(ctx, ref, value, options.add...)
		if result == nil {
			return nil, err
		}
		return &locationsView{ImageID: result.ImageID, URL: result.URL, Body: result.Body, Header: result.Header, StatusCode: result.StatusCode}, err
	}
	result, err := s.GetImageLocations(ctx, ref, options.get...)
	if result == nil {
		return nil, err
	}
	return &locationsView{ImageID: result.ImageID, Locations: result.Locations, Body: result.Body, Header: result.Header, StatusCode: result.StatusCode}, err
}

func locationsProof(t *testing.T, err error, code int, body []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, body) || proof.Header.Get("X-Request-Id") != "actual-locations" {
		t.Fatalf("accepted response evidence lost: %v %#v", err, proof)
	}
	return proof
}

func TestImageLocationsFixedRoutesAndPassiveData(t *testing.T) {
	t.Run("default Add and finite Get routes", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + locationsPrefix
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.10"
		cloud.Provider.SetToken("latest")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			raw, err := io.ReadAll(r.Body)
			if err != nil || r.URL.Path != locationsPrefix+"images/fixed/locations" || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != "latest" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
				t.Error(r.Method, r.URL, r.Header, string(raw), err)
			}
			w.Header().Set("X-Request-Id", "actual-locations")
			w.Header().Set("Location", "https://foreign.invalid/images/decoy")
			w.Header().Set("Link", "<https://foreign.invalid/next>; rel=\"next\"")
			switch r.Method {
			case http.MethodPost:
				var body map[string]json.RawMessage
				if err := json.Unmarshal(raw, &body); err != nil || len(body) != 2 || string(body["url"]) != "\"cinder://store/object1?literal=%2F#fragment\"" || string(body["validation_data"]) != "{}" {
					t.Error("default flat body changed", string(raw), err)
				}
				w.WriteHeader(202)
			case http.MethodGet:
				if len(raw) != 0 || r.ContentLength != 0 {
					t.Error("GET acquired a body", string(raw), r.ContentLength)
				}
				testcloud.JSON(w, 200, "[]")
			default:
				t.Error("discovery, status GET, fallback or URL fetch", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		service := image.New(client)
		ack, err := service.AddImageLocation(context.Background(), resource.ID("fixed"), locationsURL, image.WithAddImageLocationHeader("X-Option", "ordinary"))
		if err != nil || ack == nil || ack.ImageID != "fixed" || ack.URL != locationsURL || ack.StatusCode != 202 || len(ack.Body) != 0 || ack.Header.Get("X-Request-Id") != "actual-locations" {
			t.Fatal(ack, err)
		}
		value, err := service.GetImageLocations(context.Background(), resource.ID("fixed"), image.WithGetImageLocationsHeader("X-Option", "ordinary"))
		if err != nil || value == nil || value.ImageID != "fixed" || value.Locations == nil || len(value.Locations) != 0 || value.StatusCode != 200 || string(value.Body) != "[]" || value.Header.Get("Location") == "" || calls.Load() != 2 {
			t.Fatal(value, err, calls.Load())
		}
	})
	for _, literal := range []string{locationsURL, "rbd://pool/字%2F?x#fragment", "swift://store/container/object", "https://foreign.invalid/../images/other?project=else#literal", " Mixed Case %2F ", "\t"} {
		t.Run("passive literal "+literal, func(t *testing.T) {
			var calls atomic.Int32
			rawAck := []byte{0, 255, 'a', 'c', 'k'}
			body := &locationsBody{Reader: bytes.NewReader(rawAck)}
			wire := locationsWire(202, body)
			wire.Header.Set("Location", "https://foreign.invalid/images/other")
			client := locationsClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, err := io.ReadAll(r.Body)
				var input struct {
					URL            string
					ValidationData map[string]string
				}
				var fields map[string]json.RawMessage
				if err != nil || json.Unmarshal(raw, &fields) != nil || json.Unmarshal(fields["url"], &input.URL) != nil || json.Unmarshal(fields["validation_data"], &input.ValidationData) != nil || input.URL != literal || len(input.ValidationData) != 0 || len(fields) != 2 || r.Method != http.MethodPost || r.URL.String() != locationsBase+"images/fixed/locations" {
					t.Error(r.Method, r.URL, string(raw), input, err)
				}
				return wire, nil
			})
			value, err := image.New(client).AddImageLocation(context.Background(), resource.ID("fixed"), literal)
			if err != nil || value == nil || value.ImageID != "fixed" || value.URL != literal || !bytes.Equal(value.Body, rawAck) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), body.closes.Load())
			}
			wire.Header.Set("X-Request-Id", "changed")
			value.Body[0] = '!'
			if rawAck[0] != 0 || value.Header.Get("X-Request-Id") != "actual-locations" {
				t.Fatal("opaque acknowledgement aliases wire evidence", value)
			}
		})
	}
	t.Run("unknown algorithm and nonhex hash remain literal data", func(t *testing.T) {
		var calls atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, err := io.ReadAll(r.Body)
			var fields map[string]json.RawMessage
			var pair map[string]string
			if err != nil || json.Unmarshal(raw, &fields) != nil || json.Unmarshal(fields["validation_data"], &pair) != nil || len(pair) != 2 || pair["os_hash_algo"] != "Future:Algorithm" || pair["os_hash_value"] != " NOT HEX %2F " {
				t.Error(string(raw), pair, err)
			}
			return locationsWire(202, io.NopCloser(strings.NewReader("queued"))), nil
		})
		value, err := image.New(client).AddImageLocation(context.Background(), resource.ID("fixed"), locationsURL, image.WithImageLocationValidation("Future:Algorithm", " NOT HEX %2F "))
		if err != nil || value == nil || string(value.Body) != "queued" || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
	t.Run("canonical passive rows own raw precision and identities", func(t *testing.T) {
		raw := "[{\"url\":\"rbd://pool/字%2F?x#fragment\",\"metadata\":{\"store\":\"foreign/store\",\"large\":9007199254740993,\"fraction\":1.000000000000000000001,\"zero\":0,\"false\":false,\"null\":null,\"nested\":{\"array\":[1,true]}},\"id\":\"other\",\"image_id\":\"decoy\",\"status\":\"active\",\"next\":\"https://foreign.invalid/next\"},{\"url\":null,\"metadata\":null},{},{\"url\":\"\",\"metadata\":{}},{\"URL\":12,\"Metadata\":false}]"
		body := &locationsBody{Reader: strings.NewReader(raw)}
		wire := locationsWire(200, body)
		wire.Header.Set("Link", "<https://foreign.invalid/next>; rel=\"next\"")
		var calls atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodGet || r.URL.String() != locationsBase+"images/fixed/locations" || r.Body != nil {
				t.Error(r.Method, r.URL, r.Body)
			}
			return wire, nil
		})
		value, err := image.New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
		if err != nil || value == nil || value.ImageID != "fixed" || value.StatusCode != 200 || string(value.Body) != raw || len(value.Locations) != 5 || calls.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(value, err, calls.Load(), body.closes.Load())
		}
		first, null, missing, empty, decoy := value.Locations[0], value.Locations[1], value.Locations[2], value.Locations[3], value.Locations[4]
		if first.URL == nil || *first.URL != "rbd://pool/字%2F?x#fragment" || string(first.Metadata["large"]) != "9007199254740993" || string(first.Metadata["fraction"]) != "1.000000000000000000001" || string(first.Metadata["zero"]) != "0" || string(first.Metadata["false"]) != "false" || string(first.Metadata["null"]) != "null" || string(first.Metadata["nested"]) != "{\"array\":[1,true]}" || string(first.Body["id"]) != "\"other\"" || string(first.Body["next"]) == "" || null.URL != nil || null.Metadata != nil || string(null.Body["url"]) != "null" || missing.URL != nil || missing.Metadata != nil || empty.URL == nil || *empty.URL != "" || empty.Metadata == nil || len(empty.Metadata) != 0 || decoy.URL != nil || decoy.Metadata != nil || string(decoy.Body["URL"]) != "12" || string(decoy.Body["Metadata"]) != "false" {
			t.Fatal("canonical row data or presence changed", value.Locations)
		}
		if _, exists := missing.Body["url"]; exists {
			t.Fatal("missing URL acquired raw presence", missing)
		}
		value.Body[0] = '!'
		first.Body["metadata"][0] = '!'
		*first.URL = "caller"
		wire.Header.Set("X-Request-Id", "later")
		if string(first.Metadata["large"]) != "9007199254740993" || string(first.Body["url"]) != "\"rbd://pool/字%2F?x#fragment\"" || value.Header.Get("X-Request-Id") != "actual-locations" || string(empty.Body["metadata"]) != "{}" {
			t.Fatal("model aliases another projection", value)
		}
	})
}

func TestImageLocationsCanonicalModelsAndPreflight(t *testing.T) {
	for _, raw := range []string{"", "null", "{}", "{\"locations\":[]}", "true", "12", "\"row\"", "[null]", "[[]]", "[false]", "[12]", "[\"row\"]", "[{\"url\":12}]", "[{\"url\":false}]", "[{\"url\":{}}]", "[{\"url\":[]}]", "[{\"metadata\":12}]", "[{\"metadata\":false}]", "[{\"metadata\":[]}]", "[{\"url\":\"valid\"},{\"metadata\":false}]", "[{\"unknown\":\"" + string([]byte{255}) + "\"}]", "[{\"metadata\":{\"unknown\":\"" + string([]byte{255}) + "\"}}]", "["} {
		t.Run(fmt.Sprintf("strict GET %q", raw), func(t *testing.T) {
			body := &locationsBody{Reader: strings.NewReader(raw)}
			var calls atomic.Int32
			client := locationsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return locationsWire(200, body), nil
			})
			value, err := image.New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
			if value != nil || err == nil || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), body.closes.Load())
			}
			locationsProof(t, err, 200, []byte(raw))
		})
	}
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "invalid endpoint UTF8", "zero ref", "unsafe ID", "invalid UTF8 ID", "invalid UTF8 name", "nil context", "canceled", "source auth", "source bad value", "source aliases", "source version", "empty URL", "invalid UTF8 URL", "nil option", "callback error", "empty pair", "missing algorithm", "missing value", "invalid UTF8 pair", "provider replacement"} {
		t.Run(mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := locationsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return locationsWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			cause := errors.New("location preflight cause")
			var ctx context.Context = context.Background()
			ref, literal := resource.Name("needle"), locationsURL
			want := error(resource.ErrInvalidOption)
			wantCallbacks := int32(0)
			option := image.AddImageLocationOption(func(*image.AddImageLocationOpts) error { callbacks.Add(1); return nil })
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = locationsBase + "?query=bad"
			case "invalid endpoint UTF8":
				client.Endpoint = locationsBase + string([]byte{255}) + "/"
			case "zero ref":
				ref = resource.Ref{}
			case "unsafe ID":
				ref = resource.ID("../escape")
			case "invalid UTF8 ID":
				ref = resource.ID(string([]byte{255}))
			case "invalid UTF8 name":
				ref = resource.Name(string([]byte{255}))
			case "nil context":
				ctx = nil
			case "canceled":
				parent, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = parent, context.Canceled
			case "source auth":
				client.MoreHeaders = map[string]string{"x-auth-token": "foreign"}
			case "source bad value":
				client.MoreHeaders = map[string]string{"X-Source": "bad\nvalue"}
			case "source aliases":
				client.MoreHeaders = map[string]string{"X-Source": "a", "x-source": "b"}
			case "source version":
				client.MoreHeaders = map[string]string{"OpenStack-API-Version": "image 2.17"}
			case "empty URL":
				literal = ""
			case "invalid UTF8 URL":
				literal = string([]byte{255})
			case "nil option":
				option = nil
			case "callback error":
				want, wantCallbacks = cause, 1
				option = func(*image.AddImageLocationOpts) error { callbacks.Add(1); return cause }
			case "empty pair", "missing algorithm", "missing value", "invalid UTF8 pair":
				wantCallbacks = 1
				option = func(c *image.AddImageLocationOpts) error {
					callbacks.Add(1)
					c.ValidationData = &image.ImageLocationValidation{}
					switch mode {
					case "missing algorithm":
						c.ValidationData.OSHashValue = "value"
					case "missing value":
						c.ValidationData.OSHashAlgo = "algo"
					case "invalid UTF8 pair":
						c.ValidationData.OSHashAlgo, c.ValidationData.OSHashValue = "algo", string([]byte{255})
					}
					return nil
				}
			case "provider replacement":
				wantCallbacks = 1
				option = func(*image.AddImageLocationOpts) error {
					callbacks.Add(1)
					client.ProviderClient = &gophercloud.ProviderClient{}
					return nil
				}
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			value, err := service.AddImageLocation(ctx, ref, literal, option)
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
			if mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal("custom cancellation lost", err)
			}
		})
	}
	for _, add := range []bool{true, false} {
		for _, header := range []string{"X-Auth-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Accept", "Content-Type", "OpenStack-API-Version", "X-Openstack-Glance-Api-Version", "X-Openstack-Image-Size", "Bad Key"} {
			t.Run(fmt.Sprint("protected option ", add, " ", header), func(t *testing.T) {
				var calls atomic.Int32
				client := locationsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				options := locationsOptions{add: []image.AddImageLocationOption{image.WithAddImageLocationHeader(header, "foreign")}, get: []image.GetImageLocationsOption{image.WithGetImageLocationsHeader(header, "foreign")}}
				value, err := locationsCall(image.New(client), context.Background(), add, resource.Name("needle"), locationsURL, options)
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	}
	t.Run("GET validates context before callbacks", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("GET canceled")
		cancel(cause)
		var calls, callbacks atomic.Int32
		client := locationsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, cause })
		value, err := image.New(client).GetImageLocations(ctx, resource.ID("fixed"), func(*image.GetImageLocationsOpts) error { callbacks.Add(1); return nil })
		if value != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestImageLocationsPreparedOptionsAndExactNames(t *testing.T) {
	t.Run("Add helpers and retained callbacks snapshot payload and headers", func(t *testing.T) {
		headers := map[string]string{"X-Kept": "snapshot"}
		pair := &image.ImageLocationValidation{OSHashAlgo: "Future", OSHashValue: "hash1"}
		replacement := image.WithAddImageLocationOpts(image.AddImageLocationOpts{Headers: headers, ValidationData: pair})
		merged := map[string]string{"X-Merged": "snapshot", "x-source": "ordinary"}
		mergeOption := image.WithAddImageLocationHeaders(merged)
		headers["X-Kept"], pair.OSHashValue, merged["X-Merged"] = "caller", "caller", "caller"
		var calls, callbacks atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, err := io.ReadAll(r.Body)
			var fields map[string]json.RawMessage
			var data map[string]string
			if err != nil || json.Unmarshal(raw, &fields) != nil || json.Unmarshal(fields["validation_data"], &data) != nil || data["os_hash_algo"] != "Future" || data["os_hash_value"] != "hash1" || r.Header.Get("X-Discarded") != "" || r.Header.Get("X-Kept") != "snapshot" || r.Header.Get("X-Merged") != "snapshot" || r.Header.Get("X-Source") != "ordinary" || r.Header.Get("X-Retained") != "original" || r.Header.Get("X-Last") != "applied" {
				t.Error(string(raw), data, r.Header, err)
			}
			return locationsWire(202, io.NopCloser(strings.NewReader(""))), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "source"}
		options := []image.AddImageLocationOption{image.WithAddImageLocationHeader("X-Discarded", "discarded"), replacement, mergeOption, nil, nil}
		var retained *image.AddImageLocationOpts
		options[3] = func(c *image.AddImageLocationOpts) error {
			callbacks.Add(1)
			retained = c
			c.Headers["X-Retained"] = "original"
			options[4] = nil
			return nil
		}
		options[4] = func(c *image.AddImageLocationOpts) error {
			callbacks.Add(1)
			retained.ValidationData.OSHashValue = "late"
			retained.Headers["X-Retained"] = "late"
			c.Headers["X-Last"] = "applied"
			return nil
		}
		value, err := image.New(client).AddImageLocation(context.Background(), resource.ID("fixed"), locationsURL, options...)
		if err != nil || value == nil || calls.Load() != 1 || callbacks.Load() != 2 || client.MoreHeaders["X-Source"] != "source" {
			t.Fatal(value, err, calls.Load(), callbacks.Load(), client.MoreHeaders)
		}
	})
	t.Run("GET replacement header merge and retained callbacks", func(t *testing.T) {
		headers := map[string]string{"X-Kept": "snapshot"}
		option := image.WithGetImageLocationsOpts(image.GetImageLocationsOpts{Headers: headers})
		headers["X-Kept"] = "caller"
		var retained *image.GetImageLocationsOpts
		var calls, callbacks atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil || r.Header.Get("X-Discarded") != "" || r.Header.Get("X-Kept") != "snapshot" || r.Header.Get("X-Retained") != "original" || r.Header.Get("X-Merged") != "applied" {
				t.Error(r.Header, r.Body)
			}
			return locationsWire(200, io.NopCloser(strings.NewReader("[]"))), nil
		})
		options := []image.GetImageLocationsOption{image.WithGetImageLocationsHeader("X-Discarded", "discarded"), option, nil, nil}
		options[2] = func(c *image.GetImageLocationsOpts) error {
			callbacks.Add(1)
			retained = c
			c.Headers["X-Retained"] = "original"
			options[3] = nil
			return nil
		}
		options[3] = func(c *image.GetImageLocationsOpts) error {
			callbacks.Add(1)
			retained.Headers["X-Retained"] = "late"
			return image.WithGetImageLocationsHeaders(map[string]string{"X-Merged": "applied"})(c)
		}
		value, err := image.New(client).GetImageLocations(context.Background(), resource.ID("fixed"), options...)
		if err != nil || value == nil || callbacks.Load() != 2 || calls.Load() != 1 {
			t.Fatal(value, err, callbacks.Load(), calls.Load())
		}
	})
	t.Run("standard options reused in parallel", func(t *testing.T) {
		addOption := image.WithAddImageLocationOpts(image.AddImageLocationOpts{ValidationData: &image.ImageLocationValidation{OSHashAlgo: "Future", OSHashValue: "hash1"}, Headers: map[string]string{"X-Parallel": "owned"}})
		getOption := image.WithGetImageLocationsHeaders(map[string]string{"X-Parallel": "owned"})
		var calls atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Parallel") != "owned" {
				t.Error(r.Header)
			}
			if r.Method == http.MethodPost {
				return locationsWire(202, io.NopCloser(strings.NewReader(""))), nil
			}
			return locationsWire(200, io.NopCloser(strings.NewReader("[]"))), nil
		})
		service := image.New(client)
		var wait sync.WaitGroup
		for i := range 4 {
			wait.Add(1)
			go func(add bool) {
				defer wait.Done()
				value, err := locationsCall(service, context.Background(), add, resource.ID("fixed"), locationsURL, locationsOptions{add: []image.AddImageLocationOption{addOption}, get: []image.GetImageLocationsOption{getOption}})
				if err != nil || value == nil {
					t.Error(value, err)
				}
			}(i%2 == 0)
		}
		wait.Wait()
		if calls.Load() != 4 {
			t.Fatal(calls.Load())
		}
	})
	for _, add := range []bool{true, false} {
		for _, mode := range []string{"safe callback source changes", "all pages", "UUID-shaped Name", "missing", "ambiguous", "late malformed", "late HTTP404", "late read", "unsafe ID", "lookup and provider failure", "cancel lookup"} {
			t.Run(fmt.Sprint("Name ", add, " ", mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("Name/source cause")
				name := "needle"
				if mode == "UUID-shaped Name" {
					name = "b2173dd3-7ad6-4362-baa6-a68bce3565cb"
				}
				var calls, finalCalls atomic.Int32
				var bodies []*locationsBody
				client := locationsClient(nil)
				client.MoreHeaders = map[string]string{"X-Source": "captured"}
				client.HTTPClient.Transport = locationsTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "prepared" {
						t.Error("captured options lost", r.Header)
					}
					if r.URL.Path == locationsPrefix+"images/fixed/locations" {
						finalCalls.Add(1)
						if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "after-page" {
							t.Error(r.URL, r.Header)
						}
						if add {
							raw, err := io.ReadAll(r.Body)
							if err != nil || r.Method != http.MethodPost || !bytes.Contains(raw, []byte("\"os_hash_value\":\"hash1\"")) {
								t.Error(r.Method, string(raw), err)
							}
							return locationsWire(202, io.NopCloser(strings.NewReader(""))), nil
						}
						if r.Method != http.MethodGet || r.Body != nil {
							t.Error(r.Method, r.Body)
						}
						return locationsWire(200, io.NopCloser(strings.NewReader("[]"))), nil
					}
					if r.Method != http.MethodGet || r.URL.Path != locationsPrefix+"images" {
						t.Error("unexpected Name GET or fallback", r.Method, r.URL)
					}
					if mode == "lookup and provider failure" {
						client.ProviderClient = &gophercloud.ProviderClient{}
						return nil, cause
					}
					first := r.URL.Query().Get("marker") == ""
					if first && r.URL.Query().Get("name") != name {
						t.Error("Name kind/literal changed", r.URL)
					}
					payload, code := "{\"images\":[]}", 200
					if mode != "missing" {
						if first {
							payload = fmt.Sprintf("{\"images\":[{\"id\":\"fixed\",\"name\":%q,\"status\":\"deactivated\"}],\"next\":\"/v2/images?marker=second\"}", name)
						} else {
							payload = "{\"images\":[{\"id\":\"near\",\"name\":\"needle-suffix\"}]}"
						}
					}
					switch mode {
					case "UUID-shaped Name":
						payload = fmt.Sprintf("{\"images\":[{\"id\":\"fixed\",\"name\":%q}]}", name)
					case "ambiguous":
						if !first {
							payload = "{\"images\":[{\"id\":\"other\",\"name\":\"needle\"}]}"
						}
					case "late malformed":
						if !first {
							payload = "{\"images\":"
						}
					case "late HTTP404":
						if !first {
							payload, code = "list failure", 404
						}
					case "unsafe ID":
						payload = "{\"images\":[{\"id\":\"../escape\",\"name\":\"needle\"}]}"
					}
					body := &locationsBody{Reader: strings.NewReader(payload)}
					if mode == "late read" && !first {
						body.Reader = locationsReader(func(p []byte) (int, error) { return copy(p, "{\"images\":"), cause })
					}
					body.onClose = func() {
						client.SetToken("after-page")
						if mode == "cancel lookup" {
							cancel(cause)
						}
					}
					bodies = append(bodies, body)
					return locationsWire(code, body), nil
				})
				options := locationsOptions{add: []image.AddImageLocationOption{image.WithAddImageLocationHeader("X-Option", "prepared"), image.WithImageLocationValidation("Future", "hash1")}, get: []image.GetImageLocationsOption{image.WithGetImageLocationsHeader("X-Option", "prepared")}}
				if mode == "safe callback source changes" {
					change := func() {
						client.ResourceBase = "https://glance.invalid/later/v2/"
						client.MoreHeaders = map[string]string{"X-Source": "later"}
						client.SetToken("during-options")
					}
					options.add = append(options.add, func(*image.AddImageLocationOpts) error { change(); return nil })
					options.get = append(options.get, func(*image.GetImageLocationsOpts) error { change(); return nil })
				}
				value, err := locationsCall(image.New(client), ctx, add, resource.Name(name), locationsURL, options)
				success := mode == "all pages" || mode == "safe callback source changes" || mode == "UUID-shaped Name"
				if success {
					expected := int32(3)
					if mode == "UUID-shaped Name" {
						expected = 2
					}
					if err != nil || value == nil || value.ImageID != "fixed" || calls.Load() != expected || finalCalls.Load() != 1 {
						t.Fatal(value, err, calls.Load(), finalCalls.Load())
					}
				} else if value != nil || err == nil || finalCalls.Load() != 0 {
					t.Fatal(value, err, calls.Load(), finalCalls.Load())
				}
				if mode == "missing" && !errors.Is(err, resource.ErrNotFound) || mode == "ambiguous" && !errors.Is(err, resource.ErrAmbiguous) || (mode == "unsafe ID" || mode == "lookup and provider failure") && !errors.Is(err, resource.ErrInvalidOption) || mode == "late HTTP404" && !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(value, err)
				}
				if (mode == "late read" || mode == "lookup and provider failure" || mode == "cancel lookup") && !errors.Is(err, cause) {
					t.Fatal("Name custom cause lost", err)
				}
				for _, body := range bodies {
					if body.closes.Load() != 1 {
						t.Fatal("native Name body ownership changed", body.closes.Load())
					}
				}
			})
		}
	}
}

func TestImageLocationsAcceptedFailuresAndStatusPolicy(t *testing.T) {
	for _, add := range []bool{true, false} {
		for _, mode := range []string{"read", "Close", "read Close cancellation", "cancel only"} {
			t.Run(fmt.Sprint("accepted ", add, " ", mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("Close cause"), errors.New("cancel cause")
				code, raw := 200, []byte("[{\"url\":\"valid\"}]")
				if add {
					code, raw = 202, []byte{0, 255, 'a', 'c', 'k'}
				}
				body := &locationsBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "read") {
					body.Reader = locationsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					body.closeErr = closeCause
				}
				wire := locationsWire(code, body)
				body.onClose = func() {
					wire.Header.Set("X-Request-Id", "changed during Close")
					if strings.Contains(mode, "cancel") {
						cancel(cancelCause)
					}
				}
				var calls, retries atomic.Int32
				client := locationsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted replay")
				}
				value, err := locationsCall(image.New(client), ctx, add, resource.ID("fixed"), locationsURL, locationsOptions{})
				proof := locationsProof(t, err, code, raw)
				if calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(value, err, calls.Load(), retries.Load(), body.closes.Load())
				}
				if add {
					if value == nil || value.ImageID != "fixed" || value.URL != locationsURL || value.StatusCode != 202 || !bytes.Equal(value.Body, raw) || value.Header.Get("X-Request-Id") != "actual-locations" {
						t.Fatal(value, err)
					}
					value.Body[0] = '!'
					value.Header.Set("X-Request-Id", "caller")
					if proof.Body[0] != 0 || proof.Header.Get("X-Request-Id") != "actual-locations" || raw[0] != 0 {
						t.Fatal("acknowledgement aliases error evidence", value, proof)
					}
				} else if value != nil {
					t.Fatal("GET returned partial typed array after failure", value, err)
				}
				if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "Close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("accepted cause lost", err)
				}
			})
		}
		for _, code := range []int{200, 201, 202, 204, 206, 300, 400, 403, 404, 409, 429, 500} {
			expected := 200
			if add {
				expected = 202
			}
			if code == expected {
				continue
			}
			t.Run(fmt.Sprint("strict status ", add, " ", code), func(t *testing.T) {
				body := &locationsBody{Reader: strings.NewReader("raw status proof")}
				var calls atomic.Int32
				client := locationsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return locationsWire(code, body), nil })
				value, err := locationsCall(image.New(client), context.Background(), add, resource.ID("fixed"), locationsURL, locationsOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				method := http.MethodGet
				if add {
					method = http.MethodPost
				}
				if value != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{expected}) || native.Method != method || native.URL != locationsBase+"images/fixed/locations" || string(native.Body) != "raw status proof" || native.ResponseHeader.Get("X-Request-Id") != "actual-locations" || errors.As(err, &proof) || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(value, err, native, calls.Load(), body.closes.Load())
				}
			})
		}
	}
}

type locationsSnapshotMarshaler struct {
	body  []byte
	calls *atomic.Int32
}

func (m locationsSnapshotMarshaler) MarshalJSON() ([]byte, error) {
	if m.calls.Add(1) > 1 {
		return nil, errors.New("replacement marshaled again")
	}
	return append([]byte(nil), m.body...), nil
}

func TestImageLocationsProviderHooksAndExistingCompatibility(t *testing.T) {
	for _, add := range []bool{true, false} {
		t.Run(fmt.Sprint("configured live hooks ", add), func(t *testing.T) {
			var calls, reauth, backoff, retries atomic.Int32
			var bodies []*locationsBody
			client := locationsClient(nil)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff.Add(1)
				client.SetToken("backoff")
				return nil
			}
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) || !options.KeepResponseBody {
					return original
				}
				client.SetToken("retry")
				return nil
			}
			originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
			client.HTTPClient.Transport = locationsTransport(func(r *http.Request) (*http.Response, error) {
				n := int(calls.Add(1)) - 1
				codes, tokens := []int{401, 429, 503, 200}, []string{"initial", "reauth", "backoff", "retry"}
				if add {
					codes[3] = 202
				}
				if n >= len(codes) {
					return nil, errors.New("unexpected replay")
				}
				if r.URL.String() != locationsBase+"images/fixed/locations" || r.Header.Get("X-Auth-Token") != tokens[n] || r.Header.Get("X-Source") != "captured" {
					t.Error(r.URL, r.Header)
				}
				if add {
					raw, err := io.ReadAll(r.Body)
					if r.Method != http.MethodPost || err != nil || !bytes.Contains(raw, []byte("\"validation_data\":{}")) {
						t.Error(r.Method, string(raw), err)
					}
				} else if r.Method != http.MethodGet || r.Body != nil {
					t.Error(r.Method, r.Body)
				}
				body := &locationsBody{Reader: strings.NewReader("[]")}
				bodies = append(bodies, body)
				return locationsWire(codes[n], body), nil
			})
			value, err := locationsCall(image.New(client), context.Background(), add, resource.ID("fixed"), locationsURL, locationsOptions{})
			if err != nil || value == nil || calls.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
				t.Fatal(value, err, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	for _, change := range []string{"same serialized replacement", "stateful equivalent replacement", "changed JSONBody", "in-place RawMessage", "nil JSONBody", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody", "GET nil to null", "expanded OkCodes"} {
		t.Run("shared body guard "+change, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			add := change != "GET nil to null"
			callbackCause, readCause, closeCause, cancelCause := errors.New("callback"), errors.New("private read"), errors.New("private Close"), errors.New("private cancellation")
			var calls, retries, marshalCalls, borrowedReads atomic.Int32
			borrowed := &locationsBody{Reader: locationsReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*locationsBody
			var firstBody []byte
			inputPair := &image.ImageLocationValidation{OSHashAlgo: "Future", OSHashValue: "hash1"}
			option := image.WithAddImageLocationOpts(image.AddImageLocationOpts{ValidationData: inputPair})
			client := locationsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if n > 2 {
					return nil, errors.New("extra location replay")
				}
				if r.URL.String() != locationsBase+"images/fixed/locations" {
					t.Error(r.URL)
				}
				if add {
					raw, err := io.ReadAll(r.Body)
					if err != nil || r.Method != http.MethodPost {
						t.Error(string(raw), r.Method, err)
					}
					if n == 1 {
						firstBody = append([]byte(nil), raw...)
					} else if !bytes.Equal(raw, firstBody) {
						t.Error("retry serialized body changed", string(firstBody), string(raw))
					}
				} else if r.Method != http.MethodGet || r.Body != nil {
					t.Error(r.Method, r.Body)
				}
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "ack"
					if change == "expanded OkCodes" {
						code, raw = 200, "private200"
					}
				}
				body := &locationsBody{Reader: strings.NewReader(raw)}
				if change == "expanded OkCodes" && n == 2 {
					body.Reader = locationsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					body.closeErr = closeCause
					body.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, body)
				return locationsWire(code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) || !options.KeepResponseBody || options.JSONResponse != nil || options.RawBody != nil {
					t.Error(options, original)
				}
				inputPair.OSHashValue = "caller mutation"
				switch change {
				case "same serialized replacement":
					options.JSONBody = map[string]any{"url": locationsURL, "validation_data": map[string]string{"os_hash_algo": "Future", "os_hash_value": "hash1"}}
					return nil
				case "stateful equivalent replacement":
					options.JSONBody = locationsSnapshotMarshaler{body: firstBody, calls: &marshalCalls}
					return nil
				case "changed JSONBody":
					options.JSONBody = map[string]any{"url": "https://foreign.invalid/other", "validation_data": map[string]string{}}
				case "in-place RawMessage":
					raw, ok := options.JSONBody.(json.RawMessage)
					if !ok {
						t.Fatal("owned JSON body was not RawMessage", options.JSONBody)
					}
					index := bytes.Index(raw, []byte("hash1"))
					if index < 0 {
						t.Fatal(string(raw))
					}
					raw[index+4] = '2'
				case "nil JSONBody":
					options.JSONBody = nil
				case "KeepResponseBody":
					options.KeepResponseBody = false
				case "JSONResponse":
					options.JSONResponse = new(any)
				case "RawBody":
					options.RawBody = borrowed
				case "unsupported JSONBody":
					options.JSONBody = make(chan int)
				case "GET nil to null":
					options.JSONBody = json.RawMessage("null")
				case "expanded OkCodes":
					options.OkCodes = []int{202, 200}
					return nil
				}
				return callbackCause
			}
			originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
			value, err := locationsCall(image.New(client), ctx, add, resource.ID("fixed"), locationsURL, locationsOptions{add: []image.AddImageLocationOption{option}})
			safe := change == "same serialized replacement" || change == "stateful equivalent replacement"
			if safe {
				if err != nil || value == nil || value.StatusCode != 202 || calls.Load() != 2 {
					t.Fatal(value, err, calls.Load())
				}
				if change == "stateful equivalent replacement" && marshalCalls.Load() != 1 {
					t.Fatal("replacement re-marshaled during native replay", marshalCalls.Load())
				}
			} else if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{202}) || string(native.Body) != "private200" || native.ResponseHeader.Get("X-Request-Id") != "actual-locations" || errors.As(err, &proof) || calls.Load() != 2 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
					t.Fatal(value, err, native, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(err, &encoding) {
					t.Fatal("encoding cause lost", err)
				}
			}
			if retries.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
				t.Fatal("provider or borrowed body ownership changed", retries.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("transport callback and reauth nested missing are not suppressed", func(t *testing.T) {
		for _, mode := range []string{"transport", "callback", "reauth"} {
			cause := errors.New("nested missing cause")
			nested := errors.Join(gophercloud.ErrUnexpectedResponseCode{Method: http.MethodGet, URL: locationsBase, Expected: []int{200}, Actual: 404})
			var calls atomic.Int32
			client := locationsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return locationsWire(code, io.NopCloser(strings.NewReader("original failure"))), nil
			})
			if mode == "callback" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			value, err := image.New(client).GetImageLocations(context.Background(), resource.ID("fixed"))
			if value != nil || err == nil || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal("native direct reauth fields lost", err)
				}
			} else if !errors.Is(err, cause) || !errors.Is(err, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal("nested/original cause lost", err)
			}
		}
	})
	for _, mode := range []string{"foreign origin", "other path", "other query", "changed POST method", "identical target"} {
		t.Run("redirect "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*locationsBody
			target := locationsBase + "images/fixed/locations"
			client := locationsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != http.MethodPost || r.URL.String() != target {
					t.Error("redirect escaped fixed scope", r.Method, r.URL)
				}
				body := &locationsBody{Reader: strings.NewReader("")}
				bodies = append(bodies, body)
				if n == 2 {
					return locationsWire(202, body), nil
				}
				code, location := 307, target
				switch mode {
				case "foreign origin":
					location = "https://foreign.invalid/locations"
				case "other path":
					location = locationsBase + "images/other/locations"
				case "other query":
					code, location = 308, target+"?project=else"
				case "changed POST method":
					code = 302
				}
				wire := locationsWire(code, body)
				wire.Header.Set("Location", location)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			original := reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer()
			value, err := image.New(client).AddImageLocation(context.Background(), resource.ID("fixed"), locationsURL)
			if mode == "identical target" {
				if err != nil || value == nil || calls.Load() != 2 {
					t.Fatal(value, err, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if redirects.Load() != 1 || reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer() != original {
				t.Fatal("original redirect policy changed", redirects.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("native image metadata locations remain a separate projection", func(t *testing.T) {
		var calls atomic.Int32
		client := locationsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodGet || r.URL.String() != locationsBase+"images/fixed" || r.Body != nil {
				t.Error(r.Method, r.URL, r.Body)
			}
			return locationsWire(200, io.NopCloser(strings.NewReader("{\"id\":\"fixed\",\"name\":\"native\",\"locations\":[{\"url\":\"rbd://pool/object\",\"metadata\":{\"store\":\"backend\"}}]}"))), nil
		})
		value, err := sdkimages.New(client).Get(context.Background(), "fixed")
		if err != nil || value == nil || value.ID != "fixed" || value.Name != "native" || value.Properties["locations"] == nil || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}
