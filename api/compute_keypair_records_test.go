package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/keypairs"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, row, nameValue, kind, deleted string
		singleton                           bool
	}{
		{"nested fields win while outer fields remain", `{"name":"outer","fingerprint":"outer-fp","vendor":9007199254740993,"keypair":{"name":"inner","deleted":"false","user_id":"response-owner"}}`, `"inner"`, `"ssh"`, `true`, false},
		{"flat numeric ID alias", `{"id":42,"deleted":0}`, `42`, `"ssh"`, `false`, false},
		{"null name wins ID and null type remains null", `{"id":42,"name":null,"type":null,"deleted":null}`, `null`, `null`, `null`, false},
		{"empty name wins ID", `{"id":42,"name":"","deleted":[]}`, `""`, `"ssh"`, `false`, false},
		{"missing fields project null", `{}`, `null`, `"ssh"`, `null`, false},
		{"object envelope is a singleton", `{"keypair":{"name":"singleton","type":"x509"}}`, `"singleton"`, `"x509"`, `null`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" || r.ContentLength != 0 {
					t.Error(r.URL, r.ContentLength, string(body), err)
				}
				w.Header().Set("X-Record-Proof", tc.name)
				rows := "[" + tc.row + "]"
				if tc.singleton {
					rows = tc.row
				}
				testcloud.JSON(w, 200, `{"keypairs":`+rows+`}`)
			})
			var values []*keypairs.KeypairRecord
			var failure error
			for value, err := range keypairs.New(client).ListRecords(context.Background()) {
				if err != nil {
					failure = err
					break
				}
				values = append(values, value)
			}
			if failure != nil || len(values) != 1 || calls.Load() != 1 {
				t.Fatal(values, failure, calls.Load())
			}
			value := values[0]
			if value == nil || value.Resource == nil || value.Wire == nil || len(value.Resource.Body) != 9 || string(value.Resource.Body["name"]) != tc.nameValue || string(value.Resource.Body["id"]) != tc.nameValue || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["is_deleted"]) != tc.deleted || string(value.Envelope) != tc.row || value.StatusCode != 200 || value.Header.Get("X-Record-Proof") != tc.name {
				t.Fatal("source projection or actual row evidence changed", value)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal(value.Resource)
			}
			if tc.name == "nested fields win while outer fields remain" && (string(value.Resource.Body["fingerprint"]) != `"outer-fp"` || string(value.Wire.Body["name"]) != `"outer"` || string(value.Wire.Body["vendor"]) != `9007199254740993` || string(value.Resource.Body["user_id"]) != `"response-owner"`) {
				t.Fatal("nested merge changed actual outer Wire", value)
			}
			value.Resource.Body["name"][0] = 'x'
			value.Resource.Header.Set("X-Record-Proof", "changed")
			value.Envelope[0] = 'x'
			value.Header.Set("X-Record-Proof", "changed")
			if value.Wire.Header.Get("X-Record-Proof") != tc.name {
				t.Fatal("record/view headers alias Wire", value.Wire)
			}
			for _, raw := range value.Wire.Body {
				if !json.Valid(raw) {
					t.Fatal("view/envelope bytes alias Wire", value.Wire)
				}
			}
		})
	}
	for key, want := range map[string]any{"id": "key", "created_at": "now", "is_deleted": true, "fingerprint": "fp", "name": "key", "private_key": "private", "public_key": "public", "type": "ssh", "user_id": "owner+team", "limit": 2, "marker": "before"} {
		t.Run("semantic "+key, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				expected := url.Values{}
				if key == "user_id" {
					expected.Set(key, "owner+team")
				}
				if key == "limit" {
					expected.Set(key, "2")
				}
				if key == "marker" {
					expected.Set(key, "before")
				}
				if r.URL.Path != computeKeypairCreatePath || !reflect.DeepEqual(r.URL.Query(), expected) {
					t.Error(r.URL, expected)
				}
				testcloud.JSON(w, 200, `{"keypairs":[{"keypair":{"name":"key","created_at":"now","deleted":"false","fingerprint":"fp","private_key":"private","public_key":"public","user_id":"server-owner"}},{"name":"miss","created_at":"old","deleted":false,"fingerprint":"other","private_key":"other","public_key":"other","type":"x509"}]}`)
			})
			count := 0
			var failure error
			for value, err := range keypairs.New(client).ListRecords(context.Background(), keypairs.WithKeypairListFilter(key, want), keypairs.WithKeypairListPaginated(false)) {
				if err != nil {
					failure = err
					break
				}
				count++
				if value.Resource == nil {
					t.Fatal(value)
				}
			}
			wantCount := 1
			if key == "user_id" || key == "limit" || key == "marker" {
				wantCount = 2
			}
			if failure != nil || count != wantCount || calls.Load() != 1 {
				t.Fatal(count, wantCount, failure, calls.Load())
			}
		})
	}
	t.Run("lazy reusable options own maps pointers and callback carriers", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		api := keypairs.New(client)
		var calls, callbacks atomic.Int32
		filters := map[string]any{"name": "original", "deleted": func() {}, "unknown": func() {}}
		filter := keypairs.WithKeypairListFilters(filters)
		filters["name"] = "changed"
		page, version := false, "2.55"
		bulk := keypairs.WithKeypairListOptions(keypairs.KeypairListOpts{UserID: "owner+team", Paginated: &page, Microversion: &version})
		page = true
		version = "changed"
		shared := map[string]string{"X-Trace": "owned"}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-Trace", "owned")
			if !reflect.DeepEqual(r.URL.Query(), url.Values{"user_id": {"owner+team"}, "vendor": {"last"}}) {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"keypairs":[{"name":"original"},{"name":"changed"}]}`)
		})
		seq := api.ListRecords(context.Background(), filter, bulk, keypairs.WithKeypairListQuery("vendor", "first"), keypairs.WithKeypairListQuery("vendor", "last"), func(c *request.Config[keypairs.KeypairListOpts]) error {
			callbacks.Add(1)
			shared["X-Trace"] = "owned"
			c.Headers = shared
			return nil
		}, func(_ *request.Config[keypairs.KeypairListOpts]) error {
			callbacks.Add(1)
			shared["X-Trace"] = "changed"
			return nil
		})
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("iterator is not lazy")
		}
		for repeat := 0; repeat < 2; repeat++ {
			count := 0
			for value, err := range seq {
				if err != nil || value == nil || string(value.Resource.Body["name"]) != `"original"` {
					t.Fatal(value, err)
				}
				count++
			}
			if count != 1 {
				t.Fatal(count)
			}
		}
		if calls.Load() != 2 || callbacks.Load() != 4 || client.Microversion != "2.55" {
			t.Fatal(calls.Load(), callbacks.Load(), client)
		}
	})
}

func TestComputeKeypairRecordsContinuationLogicalMarkersCapsAndLateErrors(t *testing.T) {
	for _, mode := range []string{"dictionary", "links", "keypairs_links", "next", "HTTP Link", "initial limit fallback", "server limit only", "single page", "empty page stops", "late403", "break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				want := url.Values{"user_id": {"owner"}}
				if mode == "initial limit fallback" {
					want.Set("limit", "5")
				}
				if n > 1 {
					want.Set("marker", "wire-last")
					if mode == "server limit only" {
						want.Set("limit", "25")
					}
					if n == 3 {
						want.Set("marker", "second")
					}
					th.TestHeader(t, r, "X-Auth-Token", "next-token")
				}
				if !reflect.DeepEqual(r.URL.Query(), want) || r.URL.Path != computeKeypairCreatePath {
					t.Error(r.URL, want)
				}
				if n > 3 {
					t.Error("unbounded pager")
					testcloud.JSON(w, 200, `{"keypairs":[]}`)
					return
				}
				if n == 3 {
					testcloud.JSON(w, 200, `{"keypairs":[],"next":"`+computeKeypairCreatePath+`?marker=do-not-follow"}`)
					return
				}
				if n == 2 {
					if mode == "late403" {
						w.Header().Set("X-Page-Proof", "late")
						testcloud.JSON(w, 403, `{"error":"late forbidden"}`)
						return
					}
					testcloud.JSON(w, 200, `{"keypairs":[{"keypair":{"name":"second"}}]}`)
					return
				}
				cloud.Provider.SetToken("next-token")
				next := cloud.Server.URL + computeKeypairCreatePath + "?marker=wire-last"
				if mode == "server limit only" {
					next += "&limit=25"
				}
				continuation := ""
				switch mode {
				case "dictionary", "single page", "late403", "break":
					continuation = `,"links":{"next":"` + next + `"}`
				case "links", "server limit only", "empty page stops":
					continuation = `,"links":[{"rel":"next","href":"` + next + `"}]`
				case "keypairs_links":
					continuation = `,"keypairs_links":[{"rel":"next","href":"` + next + `"}]`
				case "next":
					continuation = `,"next":"` + next + `"`
				case "HTTP Link":
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
				}
				rows := `{"keypair":{"name":"first"}},{"id":"not-marker","keypair":{"name":"wire-last","type":"x509"}}`
				if mode == "empty page stops" {
					rows = ""
				}
				testcloud.JSON(w, 200, `{"keypairs":[`+rows+`]`+continuation+`}`)
			})
			options := []keypairs.KeypairListOption{keypairs.WithKeypairListUserID("owner"), keypairs.WithKeypairListFilter("type", "ssh")}
			if mode == "initial limit fallback" {
				options = append(options, keypairs.WithKeypairListLimit(5))
			}
			if mode == "single page" {
				options = append(options, keypairs.WithKeypairListPaginated(false))
			}
			var names []string
			var failure error
			for value, err := range keypairs.New(client).ListRecords(context.Background(), options...) {
				if err != nil {
					failure = err
					break
				}
				var name string
				if err := json.Unmarshal(value.Resource.Body["name"], &name); err != nil {
					t.Fatal(err)
				}
				names = append(names, name)
				value.Resource.Body["name"][0] = 'x'
				if mode == "break" {
					break
				}
			}
			want := []string{"first", "second"}
			wantCalls := int32(2)
			if mode == "initial limit fallback" {
				wantCalls = 3
			}
			if mode == "single page" || mode == "break" || mode == "late403" {
				want = []string{"first"}
				wantCalls = 1
				if mode == "late403" {
					wantCalls = 2
				}
			}
			if mode == "empty page stops" {
				want = nil
				wantCalls = 1
			}
			if !reflect.DeepEqual(names, want) || calls.Load() != wantCalls {
				t.Fatal(names, want, failure, calls.Load(), wantCalls)
			}
			if mode == "late403" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &native) || native.Actual != 403 || string(native.Body) != `{"error":"late forbidden"}` || native.ResponseHeader.Get("X-Page-Proof") != "late" {
					t.Fatal(failure, native)
				}
			} else if failure != nil {
				t.Fatal(failure)
			}
		})
	}
	for _, cap := range []int{1, 2, 0} {
		t.Run("raw cap "+strconv.Itoa(cap), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			const body = `{"keypairs":[{"name":"drop","type":"x509"},{"name":"hit"},false]}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if cap > 0 && r.URL.Query().Get("limit") != strconv.Itoa(cap) {
					t.Error(r.URL)
				}
				w.Header().Set("X-Cap-Proof", "actual")
				testcloud.JSON(w, 200, body)
			})
			count := 0
			var failure error
			for _, err := range keypairs.New(client).ListRecords(context.Background(), keypairs.WithKeypairListMaxItems(cap), keypairs.WithKeypairListFilter("type", "ssh")) {
				if err != nil {
					failure = err
					break
				}
				count++
			}
			want := 1
			if cap == 1 {
				want = 0
			}
			if count != want || calls.Load() != 1 {
				t.Fatal(count, want, failure, calls.Load())
			}
			if cap == 0 {
				var proof *resource.ResponseError
				if !errors.As(failure, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Cap-Proof") != "actual" {
					t.Fatal(failure, proof)
				}
			} else if failure != nil {
				t.Fatal("decoded beyond physical cap", failure)
			}
		})
	}
}

func TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding(t *testing.T) {
	for _, mode := range []string{"auto2.10", "explicit empty", "negative limit", "nil option", "query collision", "base_path control", "jmespath control", "source callback", "outer guard"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = "2.55"
			ctx := context.Background()
			cause := errors.New("outer keypair list cause")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path == computeConsoleDiscoveryPath {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`)
					return
				}
				if r.URL.Path != computeKeypairCreatePath {
					t.Error(r.URL)
				}
				if mode == "explicit empty" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.10")
				}
				testcloud.JSON(w, 200, `{"keypairs":[]}`)
			})
			var opts []keypairs.KeypairListOption
			switch mode {
			case "auto2.10":
				client.Microversion = ""
			case "explicit empty":
				opts = append(opts, keypairs.WithKeypairListMicroversion(""))
			case "negative limit":
				opts = append(opts, keypairs.WithKeypairListLimit(-1))
			case "nil option":
				opts = append(opts, nil)
			case "query collision":
				opts = append(opts, keypairs.WithKeypairListUserID("same"), keypairs.WithKeypairListFilter("user_id", "same"))
			case "base_path control":
				opts = append(opts, keypairs.WithKeypairListQuery("base_path", "/other"))
			case "jmespath control":
				opts = append(opts, keypairs.WithKeypairListFilter("jmespath_filters", "name"))
			case "source callback":
				opts = append(opts, func(_ *request.Config[keypairs.KeypairListOpts]) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				})
			case "outer guard":
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			}
			var failure error
			for _, err := range keypairs.New(client).ListRecords(ctx, opts...) {
				if err != nil {
					failure = err
					break
				}
			}
			if mode == "auto2.10" || mode == "explicit empty" {
				want := int32(1)
				if mode == "auto2.10" {
					want = 2
				}
				if failure != nil || calls.Load() != want {
					t.Fatal(failure, calls.Load())
				}
			} else {
				expected := resource.ErrInvalidOption
				if mode == "base_path control" || mode == "jmespath control" {
					expected = resource.ErrUnsupported
				}
				if mode == "outer guard" {
					expected = cause
				}
				if !errors.Is(failure, expected) || calls.Load() != 0 {
					t.Fatal(failure, calls.Load())
				}
			}
		})
	}
	for _, mode := range []string{"null nested row", "accepted empty204", "accepted read404", "source after page"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			body := `{"keypairs":[{"keypair":{"name":"actual"}}]}`
			status := 200
			if mode == "null nested row" {
				body = `{"keypairs":[{"keypair":null}]}`
			}
			if mode == "accepted empty204" {
				body = ""
				status = 204
			}
			var track *payloadContractTracking
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested", Body: []byte("nested404")}
			if mode == "accepted read404" {
				track = payloadContractTrack(cloud, nested, nil)
			}
			if mode == "source after page" {
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
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				w.Header().Set("X-Page-Proof", mode)
				testcloud.JSON(w, status, body)
			})
			count := 0
			var failure error
			for _, err := range keypairs.New(client).ListRecords(context.Background()) {
				if err != nil {
					failure = err
					break
				}
				count++
			}
			var proof *resource.ResponseError
			if count != 0 || calls.Load() != 1 || !errors.As(failure, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Page-Proof") != mode {
				t.Fatal(count, failure, proof, calls.Load())
			}
			if mode == "accepted read404" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &native) || native.URL != "nested" || errors.Is(failure, resource.ErrNotFound) {
					t.Fatal(failure, native)
				}
				physical := track.last(t)
				if physical.closes.Load() != 1 {
					t.Fatal(physical)
				}
			}
			if mode == "source after page" && !errors.Is(failure, resource.ErrInvalidOption) {
				t.Fatal(failure)
			}
		})
	}
}

func TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI(t *testing.T) {
	full := &keypairs.KeyPair{Name: "native", Fingerprint: "fp", PublicKey: "public", PrivateKey: "private", UserID: "response-owner", Type: "x509"}
	for _, tc := range []struct {
		name, body         string
		status             int
		want               *keypairs.KeyPair
		failure, typeError bool
	}{
		{"wrapped six strings", `{"keypairs":[{"keypair":{"name":"native","fingerprint":"fp","public_key":"public","private_key":"private","user_id":"response-owner","type":"x509","vendor":42}}],"next":"/must-not-follow"}`, 200, full, false, false},
		{"missing type remains native empty", `{"keypairs":[{"keypair":{"name":"native"}}],"links":[{"rel":"next","href":"/must-not-follow"}]}`, 200, &keypairs.KeyPair{Name: "native"}, false, false},
		{"null type remains native empty", `{"keypairs":[{"keypair":{"name":"native","type":null}}]}`, 200, &keypairs.KeyPair{Name: "native"}, false, false},
		{"null wrapped row remains native zero model", `{"keypairs":[{"keypair":null}]}`, 200, &keypairs.KeyPair{}, false, false},
		{"204 empty is native empty page", "", 204, nil, false, false},
		{"native pager also accepts 300", `{"keypairs":[{"keypair":{"name":"native"}}]}`, 300, &keypairs.KeyPair{Name: "native"}, false, false},
		{"201 is outside native pager codes", `{"keypairs":[]}`, 201, nil, true, false},
		{"forbidden preserves native HTTP", `{"error":"native owner forbidden"}`, 403, nil, true, false},
		{"known field type error stops before yielding partial rows", `{"keypairs":[{"keypair":{"name":7,"fingerprint":"retained"}}]}`, 200, nil, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Native-Source": "selected"}
			cloud.Provider.SetToken("native-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "native-live")
				th.TestHeader(t, r, "X-Native-Source", "selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				want := url.Values{"user_id": {"raw+owner"}, "limit": {"25"}}
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.Path != computeKeypairCreatePath || !reflect.DeepEqual(r.URL.Query(), want) || r.URL.RawQuery != want.Encode() {
					t.Error(r.URL, string(body), err, want)
				}
				w.Header().Set("X-Native-Proof", tc.name)
				w.Header().Set("Link", "</must-not-follow>; rel=\"next\"")
				if tc.status == 204 {
					w.WriteHeader(204)
				} else {
					testcloud.JSON(w, tc.status, tc.body)
				}
			})
			var values []*keypairs.KeyPair
			var failure error
			for value, err := range keypairs.New(client).List(context.Background(), keypairs.WithListOptions(keypairs.ListOpts{UserID: "typed-owner"}), keypairs.WithListQuery("user_id", "raw+owner"), keypairs.WithListQuery("limit", "25")) {
				if err != nil {
					failure = err
					break
				}
				values = append(values, value)
			}
			wantCount := 0
			if tc.want != nil {
				wantCount = 1
			}
			if (failure != nil) != tc.failure || len(values) != wantCount || calls.Load() != 1 || wantCount == 1 && !reflect.DeepEqual(values[0], tc.want) || client.Microversion != "2.55" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
				t.Fatal(values, failure, calls.Load(), tc.want, client)
			}
			if tc.status == 201 || tc.status == 403 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &native) || native.Actual != tc.status || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) || native.Method != http.MethodGet || native.URL != cloud.Server.URL+computeKeypairCreatePath+"?limit=25&user_id=raw%2Bowner" || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Native-Proof") != tc.name {
					t.Fatal("native List HTTP evidence changed", failure, native)
				}
			} else if tc.typeError {
				var typed *json.UnmarshalTypeError
				if !errors.As(failure, &typed) {
					t.Fatal("known-field type cause lost", failure)
				}
			}
		})
	}
}
