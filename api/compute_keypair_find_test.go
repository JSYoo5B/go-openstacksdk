package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback(t *testing.T) {
	for _, tc := range []struct {
		name, body, nameValue, kind, owner string
		status                             int
		failure, wire                      bool
	}{
		{"wrapped passive fields override seed", `{"keypair":{"id":17,"name":"actual","type":null,"user_id":null,"created_at":"now","fingerprint":"fp","private_key":"private","public_key":"public","deleted":"false","vendor":9007199254740993}}`, `"actual"`, `null`, `null`, 200, false, true},
		{"flat numeric alias stays numeric", `{"id":9007199254740993,"vendor":17}`, `9007199254740993`, `"ssh"`, `"owner+team"`, 201, false, true},
		{"null name overrides seed and response ID", `{"keypair":{"id":42,"name":null}}`, `null`, `"ssh"`, `"owner+team"`, 203, false, true},
		{"empty wrapped retains request seed", `{"keypair":{}}`, `"target"`, `"ssh"`, `"owner+team"`, 200, false, true},
		{"opaque accepted retains request seed", "opaque accepted", `"target"`, `"ssh"`, `"owner+team"`, 202, false, false},
		{"malformed accepted retains request seed", `{"keypair":`, `"target"`, `"ssh"`, `"owner+team"`, 201, false, false},
		{"invalid UTF8 accepted follows source fallback policy", "{\"keypair\":{\"name\":\"\xff\"}}", `"target"`, `"ssh"`, `"owner+team"`, 201, false, false},
		{"empty204 accepted retains request seed", "", `"target"`, `"ssh"`, `"owner+team"`, 204, false, false},
		{"valid null keypair is terminal processing failure", `{"keypair":null}`, "", "", "", 200, true, false},
		{"valid array root is terminal processing failure", `[]`, "", "", "", 200, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var gets, other atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != computeKeypairCreatePath+"/target" {
					other.Add(1)
					t.Error("direct success/fault performed fallback or discovery", r.URL)
				} else {
					gets.Add(1)
				}
				want := url.Values{"user_id": {"owner+team"}}
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || !reflect.DeepEqual(r.URL.Query(), want) || r.URL.RawQuery != want.Encode() {
					t.Error(r.URL, string(body), err)
				}
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				th.TestHeader(t, r, "Accept", "application/json")
				w.Header().Set("X-Find-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := keypairs.New(client).FindKeypair(context.Background(), "target", keypairs.WithKeypairFindUserID("owner+team"))
			if (err != nil) != tc.failure || value == nil || value.StatusCode != tc.status || string(value.Envelope) != tc.body || value.Header.Get("X-Find-Proof") != tc.name || gets.Load() != 1 || other.Load() != 0 || client.Microversion != "2.55" {
				t.Fatal(value, err, gets.Load(), other.Load())
			}
			if tc.failure {
				var proof *resource.ResponseError
				if value.Resource != nil || value.Wire != nil || !errors.As(err, &proof) || proof.StatusCode != tc.status || string(proof.Body) != tc.body || proof.Header.Get("X-Find-Proof") != tc.name || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted mapping fault lost receipt or became missing", value, err, proof)
				}
				return
			}
			if value.Resource == nil || len(value.Resource.Body) != 9 || string(value.Resource.Body["name"]) != tc.nameValue || string(value.Resource.Body["id"]) != tc.nameValue || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["user_id"]) != tc.owner || (value.Wire != nil) != tc.wire {
				t.Fatal("seeded source result differs", value)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal("unknown response became source attribute", value)
			}
			if tc.name == "wrapped passive fields override seed" && (string(value.Resource.Body["is_deleted"]) != `true` || string(value.Wire.Body["id"]) != `17` || string(value.Wire.Body["deleted"]) != `"false"` || string(value.Wire.Body["vendor"]) != `9007199254740993`) {
				t.Fatal("actual Wire was normalized", value)
			}
			if value.Wire != nil {
				if _, wrapped := value.Wire.Body["keypair"]; wrapped {
					t.Fatal("member Wire did not select actual envelope", value.Wire)
				}
				if tc.name == "empty wrapped retains request seed" && len(value.Wire.Body) != 0 {
					t.Fatal("request seed leaked into Wire", value.Wire)
				}
			}
			value.Resource.Body["id"][0] = 'x'
			if string(value.Resource.Body["name"]) != tc.nameValue {
				t.Fatal("logical alias bytes are shared", value.Resource)
			}
			value.Resource.Body["name"][0] = 'x'
			value.Resource.Header.Set("X-Find-Proof", "changed")
			if len(value.Envelope) > 0 {
				value.Envelope[0] = 'x'
			}
			value.Header.Set("X-Find-Proof", "changed")
			if value.Wire != nil {
				if value.Wire.Header.Get("X-Find-Proof") != tc.name {
					t.Fatal("headers alias actual Wire", value)
				}
				for _, raw := range value.Wire.Body {
					if !json.Valid(raw) {
						t.Fatal("view/envelope bytes alias actual Wire", value.Wire)
					}
				}
			}
		})
	}
	for _, identity := range []string{"key with space", "key/+?#%한글"} {
		t.Run("opaque member "+identity, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				want := computeKeypairCreatePath + "/" + url.PathEscape(identity)
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.EscapedPath() != want || r.RequestURI != want+"?user_id=owner%2Bteam" || !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner+team"}}) {
					t.Error("opaque name did not remain one escaped member", r.URL, r.RequestURI, want, string(body), err)
				}
				testcloud.JSON(w, 200, `{"keypair":{}}`)
			})
			value, err := keypairs.New(client).FindKeypair(context.Background(), identity, keypairs.WithKeypairFindUserID("owner+team"))
			seed, _ := json.Marshal(identity)
			if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["name"]) != string(seed) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, mode := range []string{"400", "403", "404", "empty owner omitted", "default missing", "strict missing", "explicit ignore wins", "duplicate on next page", "late403 after match", "numeric alias does not match string", "nested name wins outer alias", "unique waits for entire list", "409 has no fallback", "500 has no fallback"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var gets, lists, other atomic.Int32
			owner, identity := "owner+team", "target"
			if mode == "empty owner omitted" {
				owner = ""
			}
			if mode == "numeric alias does not match string" {
				identity = "7"
			}
			status := 404
			switch mode {
			case "400":
				status = 400
			case "403":
				status = 403
			case "409 has no fallback":
				status = 409
			case "500 has no fallback":
				status = 500
			}
			const rejected = `{"error":"original member rejection"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				want := url.Values{}
				if owner != "" {
					want.Set("user_id", owner)
				}
				if r.URL.Query().Has("marker") {
					want.Set("marker", "cursor")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("owner lost or name query invented", r.URL, string(body), err, want)
				}
				if r.URL.Path == computeKeypairCreatePath+"/"+identity {
					gets.Add(1)
					th.TestHeader(t, r, "X-Auth-Token", "test-token")
					cloud.Provider.SetToken("list-live")
					w.Header().Set("X-Find-Proof", "member original")
					testcloud.JSON(w, status, rejected)
					return
				}
				if r.URL.Path != computeKeypairCreatePath {
					other.Add(1)
					t.Error("find performed unrelated lookup", r.URL)
					w.WriteHeader(500)
					return
				}
				n := lists.Add(1)
				th.TestHeader(t, r, "X-Auth-Token", "list-live")
				w.Header().Set("X-Find-Proof", "list actual")
				if n > 2 {
					t.Error("find fallback is unbounded")
					testcloud.JSON(w, 200, `{"keypairs":[]}`)
					return
				}
				if n == 2 {
					if mode == "late403 after match" {
						w.Header().Set("X-Find-Proof", "late original")
						testcloud.JSON(w, 403, `{"error":"late list forbidden"}`)
					} else if mode == "duplicate on next page" {
						testcloud.JSON(w, 200, `{"keypairs":[{"id":"target"}]}`)
					} else {
						testcloud.JSON(w, 200, `{"keypairs":[{"name":"miss-later"}]}`)
					}
					return
				}
				rows := `{"keypair":{"name":"target","user_id":"response-owner"}}`
				switch mode {
				case "default missing", "strict missing", "explicit ignore wins":
					rows = `{"name":"miss"}`
				case "numeric alias does not match string":
					rows = `{"id":7}`
				case "nested name wins outer alias":
					rows = `{"id":"outer-miss","name":"outer-miss","keypair":{"name":"target"}}`
				}
				next := ""
				if mode == "duplicate on next page" || mode == "late403 after match" || mode == "unique waits for entire list" {
					next = `,"next":"` + computeKeypairCreatePath + `?marker=cursor"`
				}
				testcloud.JSON(w, 200, `{"keypairs":[`+rows+`]`+next+`}`)
			})
			options := []keypairs.KeypairFindOption{keypairs.WithKeypairFindUserID(owner)}
			if mode == "strict missing" {
				options = append(options, keypairs.WithKeypairFindIgnoreMissing(false))
			}
			if mode == "explicit ignore wins" {
				strict := false
				options = append(options, keypairs.WithKeypairFindOptions(keypairs.KeypairFindOpts{UserID: owner, IgnoreMissing: &strict}), keypairs.WithKeypairFindIgnoreMissing(true))
			}
			value, err := keypairs.New(client).FindKeypair(context.Background(), identity, options...)
			wantLists := int32(1)
			if mode == "duplicate on next page" || mode == "late403 after match" || mode == "unique waits for entire list" {
				wantLists = 2
			}
			if status == 409 || status == 500 {
				wantLists = 0
			}
			if gets.Load() != 1 || lists.Load() != wantLists || other.Load() != 0 || client.Microversion != "2.55" {
				t.Fatal(value, err, gets.Load(), lists.Load(), wantLists, other.Load())
			}
			if status == 409 || status == 500 {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != status || len(native.Expected) != 200 || native.Method != http.MethodGet || native.URL != cloud.Server.URL+computeKeypairCreatePath+"/target?user_id=owner%2Bteam" || string(native.Body) != rejected || native.ResponseHeader.Get("X-Find-Proof") != "member original" {
					t.Fatal("unrelated HTTP became list/missing", value, err, native)
				}
				return
			}
			switch mode {
			case "strict missing":
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.Is(err, resource.ErrNotFound) || errors.As(err, &native) {
					t.Fatal("logical absence retained suppressed member HTTP", value, err, native)
				}
			case "default missing", "explicit ignore wins", "numeric alias does not match string":
				if value != nil || err != nil {
					t.Fatal(value, err)
				}
			case "duplicate on next page":
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(value, err)
				}
			case "late403 after match":
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"late list forbidden"}` || native.ResponseHeader.Get("X-Find-Proof") != "late original" {
					t.Fatal("first match hid later failure", value, err, native)
				}
			default:
				if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.Resource.Body["name"]) != `"target"` || string(value.Resource.Body["id"]) != `"target"` || value.Header.Get("X-Find-Proof") != "list actual" {
					t.Fatal(value, err)
				}
				if _, wrapped := value.Wire.Body["keypair"]; !wrapped {
					t.Fatal("fallback did not retain actual list row Wire", value)
				}
			}
		})
	}
	t.Run("automatic version is discovered once across member and fallback", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
		var discoveries, gets, lists, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, http.MethodGet)
			if r.URL.Path == computeConsoleDiscoveryPath {
				discoveries.Add(1)
				th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
				th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 201, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`)
				return
			}
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.10")
			th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.10")
			th.TestHeader(t, r, "X-Find-Option", "owned")
			if !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner"}}) {
				t.Error(r.URL)
			}
			if r.URL.Path == computeKeypairCreatePath+"/target" {
				gets.Add(1)
				testcloud.JSON(w, 404, `{"error":"member missing"}`)
			} else if r.URL.Path == computeKeypairCreatePath {
				lists.Add(1)
				testcloud.JSON(w, 200, `{"keypairs":[{"keypair":{"name":"target"}}]}`)
			} else {
				t.Error(r.URL)
				w.WriteHeader(500)
			}
		})
		value, err := keypairs.New(client).FindKeypair(context.Background(), "target", keypairs.WithKeypairFindUserID("owner"), keypairs.WithKeypairFindHeader("X-Find-Option", "owned"), func(_ *request.Config[keypairs.KeypairFindOpts]) error { callbacks.Add(1); return nil })
		if err != nil || value == nil || value.Resource == nil || discoveries.Load() != 1 || gets.Load() != 1 || lists.Load() != 1 || callbacks.Load() != 1 || client.Microversion != "" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
			t.Fatal(value, err, discoveries.Load(), gets.Load(), lists.Load(), callbacks.Load(), client)
		}
	})
}

func TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors(t *testing.T) {
	t.Run("bulk pointers and callback headers are owned once for both phases", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		ignore, version := false, ""
		bulk := keypairs.WithKeypairFindOptions(keypairs.KeypairFindOpts{UserID: "owner+team", IgnoreMissing: &ignore, Microversion: &version})
		ignore = true
		version = "changed"
		shared := map[string]string{"X-Find-Trace": "owned", "Accept": "application/vendor+json"}
		var calls, first, last atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
			th.TestHeaderUnset(t, r, "OpenStack-API-Version")
			th.TestHeader(t, r, "X-Find-Trace", "owned")
			th.TestHeader(t, r, "Accept", "application/vendor+json")
			if !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner+team"}}) {
				t.Error(r.URL)
			}
			if r.URL.Path == computeKeypairCreatePath+"/target" {
				testcloud.JSON(w, 404, `{"error":"missing"}`)
			} else if r.URL.Path == computeKeypairCreatePath {
				testcloud.JSON(w, 200, `{"keypairs":[]}`)
			} else {
				t.Error(r.URL)
				w.WriteHeader(500)
			}
		})
		options := []keypairs.KeypairFindOption{bulk, func(c *request.Config[keypairs.KeypairFindOpts]) error {
			first.Add(1)
			shared["X-Find-Trace"] = "owned"
			c.Headers = shared
			return nil
		}, func(_ *request.Config[keypairs.KeypairFindOpts]) error {
			last.Add(1)
			shared["X-Find-Trace"] = "changed"
			return nil
		}}
		for repeat := 0; repeat < 2; repeat++ {
			value, err := keypairs.New(client).FindKeypair(context.Background(), "target", options...)
			if value != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal("factory pointer mutation changed absence policy", value, err)
			}
		}
		if calls.Load() != 4 || first.Load() != 2 || last.Load() != 2 || client.Microversion != "2.55" {
			t.Fatal(calls.Load(), first.Load(), last.Load(), client)
		}
	})
	for _, mode := range []string{"nil API", "nil context", "empty identity", "dot segment identity", "nil option", "cancel callback", "source callback", "outer guard", "source-owned blank version header"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			api := keypairs.New(client)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			identity := "target"
			cause := errors.New("outer keypair find cause")
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("preflight touched HTTP", r.URL)
				w.WriteHeader(500)
			})
			var opts []keypairs.KeypairFindOption
			expected := resource.ErrInvalidOption
			switch mode {
			case "nil API":
				api = nil
			case "nil context":
				ctx = nil
			case "empty identity":
				identity = ""
			case "dot segment identity":
				identity = ".."
			case "nil option":
				opts = append(opts, nil)
			case "cancel callback":
				expected = context.Canceled
				opts = append(opts, func(_ *request.Config[keypairs.KeypairFindOpts]) error { cancel(); return nil })
			case "source callback":
				opts = append(opts, func(_ *request.Config[keypairs.KeypairFindOpts]) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				})
			case "outer guard":
				expected = cause
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			case "source-owned blank version header":
				client.Microversion = ""
				client.MoreHeaders = map[string]string{"X-OpenStack-Nova-API-Version": ""}
			}
			opts = append(opts, func(_ *request.Config[keypairs.KeypairFindOpts]) error { later.Add(1); return nil })
			value, err := api.FindKeypair(ctx, identity, opts...)
			if value != nil || !errors.Is(err, expected) || calls.Load() != 0 || later.Load() != 0 {
				t.Fatal("preflight/callback guard ran later work", value, err, calls.Load(), later.Load())
			}
		})
	}
	for _, mode := range []string{"accepted member read404", "accepted member close404", "source after accepted member", "accepted discovery read404"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			path := computeKeypairCreatePath + "/target"
			body := `{"keypair":{"name":"actual"}}`
			status := 200
			discovery := mode == "accepted discovery read404"
			if discovery {
				client.Microversion = ""
				client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
				path = computeConsoleDiscoveryPath
				status = 201
				body = `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`
			}
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested", Body: []byte("nested accepted404"), ResponseHeader: http.Header{"X-Nested-Proof": {"original"}}}
			var track *payloadContractTracking
			if mode == "source after accepted member" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						client.ResourceBase = cloud.Server.URL + "/changed/"
					}
					return response, err
				})
			} else if mode == "accepted member close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else {
				track = payloadContractTrack(cloud, nested, nil)
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != path {
					t.Error("accepted terminal cause performed fallback/member", r.URL, path)
				}
				w.Header().Set("X-Find-Proof", mode)
				testcloud.JSON(w, status, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := keypairs.New(client).FindKeypair(context.Background(), "target")
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Find-Proof") != mode || errors.Is(err, resource.ErrNotFound) || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal(value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "source after accepted member" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.URL != "nested" || native.Actual != 404 || string(native.Body) != "nested accepted404" || native.ResponseHeader.Get("X-Nested-Proof") != "original" {
					t.Fatal(err, native)
				}
			}
			if discovery {
				if value != nil {
					t.Fatal("discovery receipt became member result", value)
				}
			} else if value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Find-Proof") != mode {
				t.Fatal("accepted member partial lost", value)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
	t.Run("native retry source mutation prevents second member and fallback", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls, retries atomic.Int32
		const body = `{"error":"original member unavailable"}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != computeKeypairCreatePath+"/target" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Find-Proof", "retry original")
			testcloud.JSON(w, 503, body)
		})
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, count uint) error {
			retries.Add(1)
			if count > 1 {
				return err
			}
			client.ResourceBase = cloud.Server.URL + "/changed/"
			return nil
		}
		value, err := keypairs.New(client).FindKeypair(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Find-Proof") != "retry original" || calls.Load() != 1 || retries.Load() != 1 {
			t.Fatal(value, err, native, calls.Load(), retries.Load())
		}
	})
	t.Run("live reauthentication keeps the fixed member and owner", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls, reauths atomic.Int32
		cloud.Provider.ReauthFunc = func(context.Context) error {
			if reauths.Add(1) > 1 {
				return errors.New("bounded keypair reauth exhausted")
			}
			cloud.Provider.SetToken("refreshed-find-auth")
			return nil
		}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
			if r.URL.Path != computeKeypairCreatePath+"/target" || !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner"}}) || r.ContentLength != 0 {
				t.Error(r.URL, r.ContentLength)
			}
			if n == 1 {
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				testcloud.JSON(w, 401, `{"error":"expired auth"}`)
				return
			}
			th.TestHeader(t, r, "X-Auth-Token", "refreshed-find-auth")
			testcloud.JSON(w, 200, `{"keypair":{"name":"actual"}}`)
		})
		value, err := keypairs.New(client).FindKeypair(context.Background(), "target", keypairs.WithKeypairFindUserID("owner"), keypairs.WithKeypairFindMicroversion("2.55"))
		if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["name"]) != `"actual"` || calls.Load() != 2 || reauths.Load() != 1 || client.Microversion != "2.55" {
			t.Fatal(value, err, calls.Load(), reauths.Load())
		}
	})
}
