package groups_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderGroupFindRecordDirectUntypedProjectionAndLiteralIdentity(t *testing.T) {
	for _, body := range []string{`{"group":{"id":null,"name":false,"volumes":"one","cls":false,"unknown":9007199254740993}}`, `{"group":{"id":9007199254740993}}`, `{"group":{}}`, `{bad`} {
		api, calls := groupAPI(t, "3.90", func(*http.Request) *http.Response { return groupWire(200, body) })
		value, err := api.FindRecord(context.Background(), "literal/id %2F")
		if err != nil || value == nil || value.Resource == nil || calls.Len() != 1 || calls.Calls()[0].EscapedPath != groupBase+"/literal%2Fid%20%252F" || calls.Calls()[0].RawQuery != "" || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.90" || string(value.Envelope) != body || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, calls.Calls())
		}
		if body == `{bad` || body == `{"group":{}}` {
			groupField(t, value, "id", `"literal/id %2F"`)
		}
		if strings.Contains(body, `"cls"`) {
			groupField(t, value, "id", `null`)
			groupField(t, value, "name", `false`)
			groupField(t, value, "volumes", `["one"]`)
			if string(value.Wire.Body["cls"]) != `false` || value.Resource.Body["cls"] != nil || string(value.Wire.Body["unknown"]) != `9007199254740993` {
				t.Fatal(value)
			}
		}
	}
}

func TestPythonCinderGroupFindRecordCleanFallbackHasNoServerNameQuery(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(http.StatusText(code)+"/"+map[bool]string{true: "detail", false: "summary"}[details], func(t *testing.T) {
				path := groupBase
				if details {
					path += "/detail"
				}
				api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
					if r.URL.Path == groupBase+"/target" {
						return groupWire(code, `{"member":"unavailable"}`)
					}
					if r.URL.Path != path || r.URL.Query().Has("name") {
						t.Fatal("Group has no declared server name query", r.URL)
					}
					if r.URL.Query().Get("offset") == "2" {
						return groupWire(202, `{"groups":[{"id":"found","name":"target","group_type":null}]}`)
					}
					return groupWire(200, `{"groups":[{"id":"other","name":"other"}],"next":"?offset=2"}`)
				})
				options := []groups.RecordFindOption{}
				if !details {
					options = append(options, groups.WithRecordFindDetails(false))
				}
				value, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || value == nil || calls.Len() != 3 || value.StatusCode != 202 || calls.Calls()[0].RawQuery != "" || calls.Calls()[1].RawQuery != "" || calls.Calls()[2].RawQuery != "offset=2" {
					t.Fatal(value, err, calls.Calls())
				}
				groupField(t, value, "id", `"found"`)
			})
		}
	}
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == groupBase+"/target" {
			return groupWire(404, `{"member":false}`)
		}
		return groupWire(200, `{"groups":[{"id":"target","name":"different"},{"id":false,"name":false},{"id":null,"name":"other"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "target")
	if err != nil || calls.Len() != 2 || value == nil {
		t.Fatal(value, err, calls.Calls())
	}
	groupField(t, value, "name", `"different"`)
}

func TestPythonCinderGroupFindRecordMissingRequiresSuccessfulList(t *testing.T) {
	for _, ignore := range []bool{true, false} {
		api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == groupBase+"/missing" {
				return groupWire(403, `{"member":"denied"}`)
			}
			return groupWire(200, `{"groups":[]}`)
		})
		options := []groups.RecordFindOption{}
		if !ignore {
			options = append(options, groups.WithRecordFindIgnoreMissing(false))
		}
		value, err := api.FindRecord(context.Background(), "missing", options...)
		if value != nil || calls.Len() != 2 || (err == nil) != ignore || !ignore && !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal(value, err, calls.Calls())
		}
	}
	for _, code := range []int{403, 404, 503} {
		api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == groupBase+"/missing" {
				return groupWire(404, `{"member":"missing"}`)
			}
			return groupWire(code, `{"list":"actual"}`)
		})
		value, err := api.FindRecord(context.Background(), "missing")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || calls.Len() != 2 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"list":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, calls.Calls())
		}
	}
}

func TestPythonCinderGroupFindRecordLateDuplicatesKeepAcceptedPage(t *testing.T) {
	for _, secondID := range []string{"second", "first"} {
		page := `{"groups":[{"id":"` + secondID + `","name":"target"}],"proof":"duplicate-page"}`
		api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == groupBase+"/target" {
				return groupWire(404, `{"member":false}`)
			}
			if r.URL.Query().Get("marker") == "next" {
				return groupWire(202, page)
			}
			return groupWire(200, `{"groups":[{"id":"first","name":"target"}],"next":"?marker=next"}`)
		})
		value, err := api.FindRecord(context.Background(), "target")
		var ambiguous *resource.AmbiguousError
		var proof *resource.ResponseError
		if value != nil || calls.Len() != 3 || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"first", secondID}) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != page || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, ambiguous, proof, calls.Calls())
		}
	}
}

func TestPythonCinderGroupFindRecordLateFailureDoesNotReturnFirstMatch(t *testing.T) {
	for _, physical := range []bool{false, true} {
		readCause, closeCause := errors.New("late read"), errors.New("late close")
		faulty := &testcloud.Body{Data: `{"late":"partial"}`, ReadErr: readCause, CloseErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == groupBase+"/target" {
				return groupWire(404, `{"member":false}`)
			}
			if r.URL.Query().Has("marker") {
				reply := groupWire(503, `{"late":"actual"}`)
				if physical {
					reply.Body = faulty
				}
				return reply
			}
			return groupWire(200, `{"groups":[{"id":"first","name":"target"}],"next":"?marker=next"}`)
		})
		value, err := api.FindRecord(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		want := `{"late":"actual"}`
		if physical {
			want = faulty.Data
		}
		if value != nil || calls.Len() != 3 || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != want || native.ResponseHeader.Get("X-Request-Id") != "actual" || physical && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || faulty.Closes() != 1) {
			t.Fatal(value, err, calls.Calls())
		}
	}
}

func TestPythonCinderGroupFindRecordAcceptedErrorsAndListConstructorAreTerminal(t *testing.T) {
	for _, stage := range []string{"member", "list"} {
		body := `{"group":null}`
		if stage == "list" {
			body = `{"groups":[{"id":"first","name":"target"},{"id":"second","cls":null}]}`
		}
		api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
			if stage == "list" && r.URL.Path == groupBase+"/target" {
				return groupWire(404, `{"member":false}`)
			}
			return groupWire(202, body)
		})
		value, err := api.FindRecord(context.Background(), "target")
		var proof *resource.ResponseError
		wantCalls := 1
		if stage == "list" {
			wantCalls = 2
		}
		if value != nil || calls.Len() != wantCalls || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, proof, calls.Calls())
		}
	}
}

func TestPythonCinderGroupFindRecordRejectedReadCloseNeverFallsBack(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause, closeCause := errors.New("member read"), errors.New("member close")
		faulty := &testcloud.Body{Data: `{"member":"partial"}`, ReadErr: readCause, CloseErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(code, ""); reply.Body = faulty; return reply })
		hooks := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			return nil
		}
		value, err := api.FindRecord(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != code || string(native.Body) != faulty.Data || native.ResponseHeader.Get("X-Request-Id") != "actual" || calls.Len() != 1 || faulty.Closes() != 1 || hooks != 0 {
			t.Fatal(value, err, calls.Calls(), faulty.Closes(), hooks)
		}
	}
}

func TestPythonCinderGroupFindRecordOptionsOwnBothPhaseSnapshots(t *testing.T) {
	details := false
	headers := map[string]string{"X-Policy": "snapshot"}
	option := groups.WithRecordFindOpts(groups.RecordFindOpts{Details: &details, Headers: headers})
	details = true
	headers["X-Policy"] = "changed"
	var held *groups.RecordFindOpts
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == groupBase+"/target" {
			if held != nil {
				held.Headers["X-Policy"] = "retained-mutated"
				*held.Details = true
			}
			return groupWire(404, `{"member":false}`)
		}
		return groupWire(200, `{"groups":[{"id":"found","name":"target"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "target", option, func(v *groups.RecordFindOpts) error { held = v; return nil })
	if err != nil || value == nil || calls.Len() != 2 || calls.Calls()[1].Path != groupBase {
		t.Fatal(value, err, calls.Calls())
	}
	for _, call := range calls.Calls() {
		if call.Header.Get("X-Policy") != "snapshot" {
			t.Fatal(call)
		}
	}
}

func TestPythonCinderGroupFindRecordPreflightAndStickyOriginalSource(t *testing.T) {
	for _, invoke := range []func(*groups.API) error{
		func(api *groups.API) error { _, err := api.FindRecord(nil, "id"); return err },
		func(api *groups.API) error {
			_, err := (*groups.API)(nil).FindRecord(context.Background(), "id")
			return err
		},
		func(api *groups.API) error { _, err := api.FindRecord(context.Background(), " "); return err },
		func(api *groups.API) error { _, err := api.FindRecord(context.Background(), "id", nil); return err },
		func(api *groups.API) error {
			_, err := api.FindRecord(context.Background(), "id", groups.WithRecordFindMicroversion("bad"))
			return err
		},
		func(api *groups.API) error {
			_, err := api.FindRecord(context.Background(), "id", groups.WithRecordFindHeaders(map[string]string{"Authorization": "forged"}))
			return err
		},
		func(api *groups.API) error {
			_, err := api.FindRecord(context.Background(), "id", groups.WithRecordFindHeaders(map[string]string{"OpenStack-API-Version": "volume 3.38"}))
			return err
		},
	} {
		api, calls := groupAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
		if err := invoke(api); !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(err, calls.Calls())
		}
	}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { t.Fatal("mutated source HTTP"); return nil })
	cause := errors.New("option failure")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err := api.FindRecord(ctx, "id", func(*groups.RecordFindOpts) error {
		api.RawClient().ResourceBase = "https://other.example/v3/project/"
		cancel(cause)
		return cause
	})
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls.Len() != 0 {
		t.Fatal(err, calls.Calls())
	}
}

func TestPythonCinderGroupFindRecordGetOverrideFallbackOriginalVersionAndHeaders(t *testing.T) {
	for _, override := range []string{"3.90", ""} {
		api, calls := groupAPI(t, "3.13", func(r *http.Request) *http.Response {
			if r.URL.Path == groupBase+"/target" {
				return groupWire(404, `{"member":false}`)
			}
			return groupWire(200, `{"groups":[{"id":"found","name":"target"}]}`)
		})
		original := map[string]string{"OpenStack-API-Version": "volume 3.13", "X-OpenStack-Volume-API-Version": "3.13", "Accept": "application/source-custom", "X-Original": "kept"}
		api.RawClient().MoreHeaders = original
		expected := make(map[string]string, len(original))
		for key, value := range original {
			expected[key] = value
		}
		value, err := api.FindRecord(context.Background(), "target", groups.WithRecordFindMicroversion(override))
		wantGet := override
		if wantGet == "" {
			wantGet = "3.13"
		}
		if err != nil || value == nil || calls.Len() != 2 || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume "+wantGet || calls.Calls()[0].Header.Get("X-OpenStack-Volume-API-Version") != wantGet || calls.Calls()[1].Header.Get("OpenStack-API-Version") != "volume 3.13" || calls.Calls()[1].Header.Get("X-OpenStack-Volume-API-Version") != "3.13" || calls.Calls()[0].Header.Get("Accept") != "application/source-custom" || calls.Calls()[1].Header.Get("Accept") != "application/json" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.13" || len(value.Discovery) != 0 {
			t.Fatal(value, err, calls.Calls(), api.RawClient())
		}
		for _, call := range calls.Calls() {
			if call.Header.Get("X-Original") != "kept" {
				t.Fatal(call)
			}
		}
	}
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == groupBase+"/target" {
			return groupWire(404, `{"member":false}`)
		}
		return groupWire(200, `{"groups":[{"id":"found","name":"target"}]}`)
	})
	api.RawClient().MoreHeaders = map[string]string{"Accept": "source"}
	value, err := api.FindRecord(context.Background(), "target", groups.WithRecordFindHeaders(map[string]string{"accept": "per-call"}))
	if err != nil || value == nil || calls.Len() != 2 {
		t.Fatal(value, err, calls.Calls())
	}
	for _, call := range calls.Calls() {
		if call.Header.Get("Accept") != "per-call" {
			t.Fatal(call)
		}
	}
}

func TestPythonCinderGroupFindRecordDiscoveryEvidenceFollowsEachVersionPhase(t *testing.T) {
	for _, override := range []bool{false, true} {
		api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
			if r.URL.Path == "/cinder/v3/" {
				return groupWire(200, `{"version":{"id":"v3.0","version":"3.90","min_version":"3.0"}}`)
			}
			if r.URL.Path == groupBase+"/target" {
				return groupWire(404, `{"member":false}`)
			}
			return groupWire(202, `{"groups":[{"id":"found","name":"target"}]}`)
		})
		options := []groups.RecordFindOption{}
		if override {
			options = append(options, groups.WithRecordFindMicroversion("3.99"))
		}
		value, err := api.FindRecord(context.Background(), "target", options...)
		if err != nil || value == nil || calls.Len() != 3 || len(value.Discovery) != 1 || value.Discovery[0].StatusCode != 200 || value.StatusCode != 202 || calls.Calls()[2].Header.Get("OpenStack-API-Version") != "volume 3.38" || api.RawClient().Microversion != "" {
			t.Fatal(value, err, calls.Calls())
		}
		if override {
			if calls.Calls()[0].Path != groupBase+"/target" || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.99" || calls.Calls()[1].Path != "/cinder/v3/" {
				t.Fatal(calls.Calls())
			}
		} else if calls.Calls()[0].Path != "/cinder/v3/" || calls.Calls()[1].Path != groupBase+"/target" || calls.Calls()[1].Header.Get("OpenStack-API-Version") != "volume 3.38" {
			t.Fatal(calls.Calls())
		}
		value.Discovery[0].Envelope[0] = 'X'
		if string(value.Envelope) != `{"groups":[{"id":"found","name":"target"}]}` || value.Wire.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value)
		}
	}
}

func TestPythonCinderGroupFindRecordDiscoveryDenialCannotBecomeMemberFallback(t *testing.T) {
	api, calls := groupAPI(t, "", func(*http.Request) *http.Response { return groupWire(403, `{"discovery":"denied"}`) })
	value, err := api.FindRecord(context.Background(), "target")
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || calls.Len() != 1 || calls.Calls()[0].Path != "/cinder/v3/" || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"discovery":"denied"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
		t.Fatal(value, err, calls.Calls())
	}
}

func TestPythonCinderGroupFindRecordAcceptedTransientSourceMutationKeepsReceipt(t *testing.T) {
	var api *groups.API
	faulty := &testcloud.Body{Data: `{"group":{"id":"actual"}}`}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(200, ""); reply.Body = faulty; return reply })
	original := api.RawClient().ResourceBase
	faulty.OnRead = func() { api.RawClient().ResourceBase = "https://other.example/v3/project/" }
	faulty.OnClose = func() { api.RawClient().ResourceBase = original }
	value, err := api.FindRecord(context.Background(), "target")
	var proof *resource.ResponseError
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != faulty.Data || proof.Header.Get("X-Request-Id") != "actual" || calls.Len() != 1 || faulty.Closes() != 1 {
		t.Fatal(value, err, proof, calls.Calls())
	}
}

func TestPythonCinderGroupFindRecordNativeRetryOwnsVersionAndPreservesOrdinaryHeaders(t *testing.T) {
	for _, mode := range []string{"mutate_version", "add_nova", "reserved_auth", "omit_owned_version"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(503, `{"retry":"actual"}`) })
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, err error, _ uint) error {
				switch mode {
				case "mutate_version":
					opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.99"
				case "add_nova":
					opts.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99"
				case "reserved_auth":
					opts.MoreHeaders["Authorization"] = "forged"
				case "omit_owned_version":
					opts.OmitHeaders = append(opts.OmitHeaders, "OpenStack-API-Version")
				}
				return nil
			}
			value, err := api.FindRecord(context.Background(), "target")
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"retry":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" || calls.Len() != 1 {
				t.Fatal(value, err, calls.Calls())
			}
		})
	}
	attempt := 0
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		attempt++
		if attempt == 1 {
			return groupWire(503, `{"retry":"first"}`)
		}
		return groupWire(200, `{"group":{"id":"actual"}}`)
	})
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, err error, _ uint) error {
		opts.MoreHeaders["X-Retry"] = "allowed"
		api.RawClient().ProviderClient.SetToken("live-refreshed")
		return nil
	}
	value, err := api.FindRecord(context.Background(), "target")
	if err != nil || value == nil || calls.Len() != 2 || calls.Calls()[1].Header.Get("X-Retry") != "allowed" || calls.Calls()[1].Header.Get("X-Auth-Token") != "live-refreshed" {
		t.Fatal(value, err, calls.Calls())
	}
}

func TestPythonCinderGroupFindRecordCleanRetryOriginalKeepsFallback(t *testing.T) {
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == groupBase+"/target" {
			return groupWire(404, `{"member":"original"}`)
		}
		return groupWire(200, `{"groups":[{"id":"found","name":"target"}]}`)
	})
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
		return err
	}
	value, err := api.FindRecord(context.Background(), "target")
	if err != nil || value == nil || calls.Len() != 2 {
		t.Fatal(value, err, calls.Calls())
	}
	query, _ := url.ParseQuery(calls.Calls()[1].RawQuery)
	if len(query) != 0 {
		t.Fatal("Group name match remains local", query)
	}
	groupField(t, value, "id", `"found"`)
}

func TestPythonCinderGroupFindRecordAcceptedPhysicalCausesCannotBecomeFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause := gophercloud.ErrUnexpectedResponseCode{Actual: code, Body: []byte(`{"nested":"not-physical-status"}`)}
		closeCause := errors.New("accepted close")
		faulty := &testcloud.Body{Data: `{"group":{"id":"partial"}}`, ReadErr: readCause, CloseErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(202, ""); reply.Body = faulty; return reply })
		hooks := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			return nil
		}
		value, err := api.FindRecord(context.Background(), "target")
		var proof *resource.ResponseError
		if value != nil || !errors.Is(err, closeCause) || !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != faulty.Data || proof.Header.Get("X-Request-Id") != "actual" || calls.Len() != 1 || faulty.Closes() != 1 || hooks != 0 {
			t.Fatal(value, err, proof, calls.Calls(), faulty.Closes(), hooks)
		}
	}
}

func TestPythonCinderGroupFindRecordFirstCustomHeaderOptionOwnsDefaultAndNilMaps(t *testing.T) {
	for _, mode := range []string{"default", "nil_opts_headers", "nil_header_option"} {
		t.Run(mode, func(t *testing.T) {
			var held *groups.RecordFindOpts
			api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
				if held != nil {
					held.Headers["X-Custom"] = "retained-mutated"
				}
				if r.URL.Path == groupBase+"/target" {
					return groupWire(404, `{"member":"fallback"}`)
				}
				return groupWire(200, `{"groups":[{"id":"found","name":"target"}]}`)
			})
			original := map[string]string{"X-Original": "source", "Accept": "application/source"}
			expected := map[string]string{"X-Original": "source", "Accept": "application/source"}
			api.RawClient().MoreHeaders = original
			for _, value := range []string{"first", "second"} {
				options := []groups.RecordFindOption{}
				switch mode {
				case "nil_opts_headers":
					options = append(options, groups.WithRecordFindOpts(groups.RecordFindOpts{Headers: nil}))
				case "nil_header_option":
					options = append(options, groups.WithRecordFindHeaders(nil))
				}
				options = append(options, func(cfg *groups.RecordFindOpts) error {
					if cfg.Headers == nil || len(cfg.Headers) != 0 {
						t.Fatal("first custom option requires a fresh empty owned map", cfg.Headers)
					}
					cfg.Headers["X-Custom"] = value
					held = cfg
					return nil
				})
				start := calls.Len()
				result, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || result == nil || calls.Len() != start+2 || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.38" {
					t.Fatal(result, err, calls.Calls(), api.RawClient().MoreHeaders)
				}
				for _, call := range calls.Calls()[start:] {
					if call.Header.Get("X-Custom") != value || call.Header.Get("X-Original") != "source" {
						t.Fatal(call, value)
					}
				}
				if calls.Calls()[start].Header.Get("Accept") != "application/source" || calls.Calls()[start+1].Header.Get("Accept") != "application/json" {
					t.Fatal(calls.Calls()[start:])
				}
			}
		})
	}
}
