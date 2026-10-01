package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringNodeAdoptDefaultsFlatBodyAndIncidentalLocation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	response := `{"node":{"id":"adopted-node","physical_id":"physical-resource","name":"auto-name","profile_id":"auto-profile","role":null,"cluster_id":null,"created_at":null,"updated_at":null,"index":9007199254740993,"status":"ACTIVE","metadata":{"big":9007199254740993},"future":false}}`
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/adopt", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"identity":"physical/name with ?#%","type":"os.nova.server-1.0"}` || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(string(body), r.URL, r.Header)
		}
		if calls.Add(1) == 2 {
			w.Header().Set("Location", "https://foreign.example/arbitrary?trace=1#fragment")
		}
		w.Header().Set("X-Request-ID", "adopted-request")
		testcloud.JSON(w, 200, response)
	})
	client := cloud.Client("clustering", "/reverse/senlin/v1")
	client.Microversion = "1.7"
	for index := range 2 {
		value, err := nodes.New(client).Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical/name with ?#%", Type: "os.nova.server-1.0"})
		if err != nil || value == nil || value.ID != "adopted-node" || value.PhysicalID != "physical-resource" || value.Name != "auto-name" || value.Role != nil || value.ClusterID != nil || value.CreatedAt != nil || value.UpdatedAt != nil || value.Index == nil || value.Index.String() != "9007199254740993" || value.Status != "ACTIVE" || value.Operation != nil {
			t.Fatal(value, err)
		}
		if value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "adopted-request" || string(value.Body["future"]) != "false" || string(value.UserMetadata["big"]) != "9007199254740993" {
			t.Fatal(value)
		}
		if index == 1 && value.Header.Get("Location") != "https://foreign.example/arbitrary?trace=1#fragment" {
			t.Fatal("incidental Location was parsed or lost", value.Header)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("adopt performed a lookup, follow or resend", calls.Load())
	}
}

func TestClusteringNodeAdoptPresenceSnapshotsAndIndependentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/adopt", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		expected := `{"identity":"physical","metadata":{"big":9007199254740993,"off":false},"name":"","overrides":{},"role":null,"snapshot":false,"type":"os.nova.server-1.0","vendor":{"zero":0}}`
		if calls.Add(1) == 3 {
			expected = `{"identity":"physical","metadata":null,"name":null,"overrides":null,"role":"","snapshot":null,"type":"os.nova.server-1.0"}`
		}
		if string(body) != expected || r.Header.Get("X-Vendor") != "retained" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("X-Request-ID", "owned-response")
		testcloud.JSON(w, 200, `{"node":{"id":"adopted","metadata":{"big":9007199254740993},"future":null}}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.7"
	api := nodes.New(client)
	metadata := json.RawMessage(`{"big":9007199254740993,"off":false}`)
	snapshot := nodes.WithAdoptOptions(nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0", Metadata: metadata})
	metadata[2] = 'X'
	vendor := map[string]any{"zero": 0}
	extension := nodes.WithAdoptField("vendor", vendor)
	vendor["zero"] = 1
	for range 2 {
		value, err := api.Adopt(context.Background(), nodes.AdoptOpts{}, snapshot, nodes.WithAdoptName(""), nodes.WithAdoptRoleNull(), nodes.WithAdoptSnapshot(false), nodes.WithAdoptOverrides(map[string]any{}), extension, nodes.WithAdoptHeader("X-Vendor", "retained"))
		if err != nil || value.ID != "adopted" || value.Operation != nil || string(value.UserMetadata["big"]) != "9007199254740993" || value.Header.Get("X-Request-ID") != "owned-response" {
			t.Fatal(value, err)
		}
		value.UserMetadata["big"][0] = 'X'
		value.Header.Set("X-Request-ID", "consumer-change")
	}
	if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"}, nodes.WithAdoptNameNull(), nodes.WithAdoptRole(""), nodes.WithAdoptSnapshotNull(), nodes.WithAdoptMetadata(nil), nodes.WithAdoptOverrides(nil), nodes.WithAdoptHeader("X-Vendor", "retained")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeAdoptPreviewFourFieldsNullDefaultsAndExactVersion(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	versions := []string{`1.0`, `9007199254740993`, `"1.0"`, `null`}
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/adopt-preview", func(w http.ResponseWriter, r *http.Request) {
		index := int(calls.Add(1)) - 1
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"identity":"physical/name with ?#%","overrides":null,"snapshot":null,"type":"os.nova.server-1.0"}` || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" {
			t.Error(string(body), r.URL, r.Header)
		}
		w.Header().Set("Location", "https://foreign.example/incidental")
		w.Header().Set("X-Request-ID", "preview-request")
		testcloud.JSON(w, 200, ` {"node_preview":{"type":"os.nova.server","version":`+versions[index]+`,"properties":{"big":9007199254740993,"fraction":1.234567890123456789,"off":false},"future":null},"outer_extension":false} `)
	})
	client := cloud.Client("clustering", "/reverse/senlin/v1")
	client.Microversion = "1.7"
	api := nodes.New(client)
	for _, version := range versions {
		value, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical/name with ?#%", Type: "os.nova.server-1.0"})
		if err != nil || value == nil || value.Spec.Type != "os.nova.server" || string(value.Spec.Version) != version || string(value.Spec.Properties["big"]) != "9007199254740993" || string(value.Spec.Properties["fraction"]) != "1.234567890123456789" || string(value.Spec.Properties["off"]) != "false" {
			t.Fatal(value, err)
		}
		if value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "preview-request" || value.Header.Get("Location") != "https://foreign.example/incidental" || string(value.Body["outer_extension"]) != "false" || string(value.Spec.Body["future"]) != "null" || !strings.HasPrefix(string(value.RawBody), " {") || !strings.HasSuffix(string(value.RawBody), "} ") {
			t.Fatal(value)
		}
		value.Spec.Properties["big"][0] = 'X'
		value.Spec.Version[0] = 'X'
		if strings.Contains(string(value.RawBody), "X") || strings.Contains(string(value.Body["node_preview"]), "X") {
			t.Fatal("typed preview shares mutable raw evidence")
		}
	}
	if calls.Load() != int32(len(versions)) {
		t.Fatal("preview created a node, followed Location or resent", calls.Load())
	}
}

func TestClusteringNodeAdoptPreviewSnapshotsFalseNullAndEmptyObject(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /senlin/v1/nodes/adopt-preview", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		expected := `{"identity":"physical","overrides":{"big":9007199254740993,"off":false},"snapshot":false,"type":"os.nova.server-1.0"}`
		switch calls.Add(1) {
		case 3:
			expected = `{"identity":"physical","overrides":{},"snapshot":true,"type":"os.nova.server-1.0"}`
		case 4:
			expected = `{"identity":"physical","overrides":null,"snapshot":null,"type":"os.nova.server-1.0"}`
		}
		if string(body) != expected || r.Header.Get("X-Vendor") != "retained" {
			t.Error(string(body), r.Header)
		}
		testcloud.JSON(w, 200, `{"node_preview":{"type":"os.nova.server","properties":{}}}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.7"
	api := nodes.New(client)
	overrides := json.RawMessage(`{"big":9007199254740993,"off":false}`)
	option := nodes.WithAdoptPreviewOptions(nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0", Overrides: overrides, Snapshot: request.Present(false)})
	overrides[2] = 'X'
	for range 2 {
		value, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{}, option, nodes.WithAdoptPreviewHeader("X-Vendor", "retained"))
		if err != nil || value.Spec.Version != nil || string(value.Body["node_preview"]) != `{"type":"os.nova.server","properties":{}}` {
			t.Fatal(value, err)
		}
	}
	for _, options := range [][]nodes.AdoptPreviewOption{
		{nodes.WithAdoptPreviewOverrides(map[string]any{}), nodes.WithAdoptPreviewSnapshot(true)},
		{nodes.WithAdoptPreviewOverrides(nil), nodes.WithAdoptPreviewSnapshotNull()},
	} {
		options = append(options, nodes.WithAdoptPreviewHeader("X-Vendor", "retained"))
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"}, options...); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 4 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeAdoptVersionTypeRecheckAndCanceledOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid adoption reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := nodes.New(client)
	for _, version := range []string{"", "1.6", "latest"} {
		client.Microversion = version
		if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"}); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	for _, change := range []string{"version", "type", "header", "context"} {
		for _, preview := range []bool{false, true} {
			client.Microversion, client.Type, client.MoreHeaders = "1.7", "clustering", nil
			ctx, cancel := context.WithCancel(context.Background())
			changeSource := func() {
				switch change {
				case "version":
					client.Microversion = "1.6"
				case "type":
					client.Type = "compute"
				case "header":
					client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.99"}
				case "context":
					cancel()
				}
			}
			var err error
			if preview {
				_, err = api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"}, func(config *request.Config[nodes.AdoptPreviewOpts]) error { changeSource(); return nil })
			} else {
				_, err = api.Adopt(ctx, nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"}, func(config *request.Config[nodes.AdoptOpts]) error { changeSource(); return nil })
			}
			cancel()
			want := resource.ErrInvalidOption
			if change == "version" {
				want = resource.ErrUnsupported
			} else if change == "context" {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatal(change, preview, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeAdoptInputCapabilityAndOwnedFieldPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid adoption reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.7"
	api := nodes.New(client)
	for _, identity := range []string{"", "  "} {
		if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: identity, Type: "os.nova.server-1.0"}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: identity, Type: "os.nova.server-1.0"}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"identity", "type", "name", "role", "snapshot", "metadata", "overrides"} {
		if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"}, nodes.WithAdoptField(key, nil)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`[]`), json.RawMessage(`false`), json.RawMessage(`{`)} {
		for _, value := range []nodes.AdoptOpts{
			{Identity: "physical", Type: "os.nova.server-1.0", Metadata: raw},
			{Identity: "physical", Type: "os.nova.server-1.0", Overrides: raw},
		} {
			if _, err := api.Adopt(context.Background(), value); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0", Overrides: raw}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, option := range []nodes.AdoptOption{nil, nodes.WithAdoptHeader("X-Auth-Token", "foreign"), nodes.WithAdoptHeader("oPeNsTaCk-ApI-vErSiOn", "clustering 1.99"), request.WithQuery[nodes.AdoptOpts]("vendor", "unsupported"), request.WithArgument[nodes.AdoptOpts]("vendor", false), nodes.WithAdoptField("vendor", json.RawMessage(`{`))} {
		if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, option := range []nodes.AdoptPreviewOption{nil, nodes.WithAdoptPreviewHeader("X-Auth-Token", "foreign"), nodes.WithAdoptPreviewHeader("OpenStack-API-Version", "clustering 1.99"), request.WithField[nodes.AdoptPreviewOpts]("vendor", false), request.WithField[nodes.AdoptPreviewOpts]("name", "ignored-by-Python"), request.WithField[nodes.AdoptPreviewOpts]("snapshot", false), request.WithQuery[nodes.AdoptPreviewOpts]("vendor", "unsupported"), request.WithArgument[nodes.AdoptPreviewOpts]("vendor", false)} {
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, api := range []*nodes.API{nil, nodes.New(nil), nodes.New(&gophercloud.ServiceClient{})} {
		if _, err := api.Adopt(context.Background(), nodes.AdoptOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Adopt(ctx, nodes.AdoptOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.AdoptPreview(ctx, nodes.AdoptPreviewOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringNodeAdoptMalformedAcceptedResponsesRetainEvidence(t *testing.T) {
	for _, preview := range []bool{false, true} {
		key, path := "node", "adopt"
		if preview {
			key, path = "node_preview", "adopt-preview"
		}
		fixtures := []string{"", `{`, `{}`, `null`, `[]`, `{"` + key + `":null}`, `{"` + key + `":[]}`}
		if preview {
			fixtures = append(fixtures, `{"node_preview":{"type":false}}`, `{"node_preview":{"properties":[]}}`)
		} else {
			fixtures = append(fixtures, `{"node":{"index":"invalid-number"}}`)
		}
		for _, body := range fixtures {
			t.Run(path+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("POST /senlin/v1/nodes/"+path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "accepted-malformed")
					testcloud.JSON(w, 200, body)
				})
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.7"
				api := nodes.New(client)
				var err error
				if preview {
					var value *nodes.AdoptPreviewResult
					value, err = api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"})
					if value != nil {
						t.Fatal(value)
					}
				} else {
					var value *nodes.Node
					value, err = api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"})
					if value != nil {
						t.Fatal(value)
					}
				}
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "accepted-malformed" || calls.Load() != 1 {
					t.Fatal(err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringNodeAdoptHTTPFailuresRetainNativeErrors(t *testing.T) {
	for _, preview := range []bool{false, true} {
		path := "adopt"
		if preview {
			path = "adopt-preview"
		}
		for _, code := range []int{201, 202, 400, 404, 409, 503} {
			t.Run(path+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				body := `{"error":{"message":"original-error"}}`
				cloud.Mux.HandleFunc("POST /senlin/v1/nodes/"+path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "failed-request")
					testcloud.JSON(w, code, body)
				})
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.7"
				api := nodes.New(client)
				var err error
				if preview {
					_, err = api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"})
				} else {
					_, err = api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"})
				}
				var original gophercloud.ErrUnexpectedResponseCode
				var evidence *resource.ResponseError
				if !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &original) || string(original.Body) != body || original.ResponseHeader.Get("X-Request-ID") != "failed-request" || errors.As(err, &evidence) || calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
			})
		}
	}
}

type nodeAdoptTransportFunc func(*http.Request) (*http.Response, error)

func (f nodeAdoptTransportFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type nodeAdoptReadErrorBody struct {
	reader io.Reader
	cause  error
	closed bool
}

func (b *nodeAdoptReadErrorBody) Read(buffer []byte) (int, error) {
	n, err := b.reader.Read(buffer)
	if err == io.EOF {
		return n, b.cause
	}
	return n, err
}
func (b *nodeAdoptReadErrorBody) Close() error { b.closed = true; return nil }

func TestClusteringNodeAdoptTransportAndAcceptedReadErrorsPreserveCauses(t *testing.T) {
	for _, preview := range []bool{false, true} {
		for _, accepted := range []bool{false, true} {
			t.Run(strings.Join([]string{map[bool]string{false: "adopt", true: "preview"}[preview], map[bool]string{false: "transport", true: "read"}[accepted]}, "/"), func(t *testing.T) {
				cause := errors.New("original-source-failure")
				partial := `{"partial":`
				body := &nodeAdoptReadErrorBody{reader: strings.NewReader(partial), cause: cause}
				var calls atomic.Int32
				provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: nodeAdoptTransportFunc(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if !accepted {
						return nil, cause
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"accepted-read"}}, Body: body, Request: r}, nil
				})}}
				client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://senlin.example/v1/", Type: "clustering", Microversion: "1.7"}
				api := nodes.New(client)
				var err error
				if preview {
					_, err = api.AdoptPreview(context.Background(), nodes.AdoptPreviewOpts{Identity: "physical", Type: "os.nova.server-1.0"})
				} else {
					_, err = api.Adopt(context.Background(), nodes.AdoptOpts{Identity: "physical", Type: "os.nova.server-1.0"})
				}
				var evidence *resource.ResponseError
				if !errors.Is(err, cause) || calls.Load() != 1 || errors.As(err, &evidence) != accepted {
					t.Fatal(err, evidence, calls.Load())
				}
				if accepted && (evidence.StatusCode != 200 || string(evidence.Body) != partial || evidence.Header.Get("X-Request-ID") != "accepted-read" || !body.closed) {
					t.Fatal(evidence, body.closed)
				}
			})
		}
	}
}
