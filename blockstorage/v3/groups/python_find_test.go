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
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderGroupFindRecordDirectUntypedProjectionAndLiteralIdentity(t *testing.T) {
	for _, body := range []string{`{"group":{"id":null,"name":false,"volumes":"one","cls":false,"unknown":9007199254740993}}`, `{"group":{"id":9007199254740993}}`, `{"group":{}}`, `{bad`} {
		api, calls := groupAPI(t, "3.90", func(*http.Request) *http.Response { return groupWire(200, body) })
		value, err := api.FindRecord(context.Background(), "literal/id %2F")
		if err != nil || value == nil || value.Resource == nil || len(*calls) != 1 || (*calls)[0].escaped != groupBase+"/literal%2Fid%20%252F" || (*calls)[0].query != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.90" || string(value.Envelope) != body || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, *calls)
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
				if err != nil || value == nil || len(*calls) != 3 || value.StatusCode != 202 || (*calls)[0].query != "" || (*calls)[1].query != "" || (*calls)[2].query != "offset=2" {
					t.Fatal(value, err, *calls)
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
	if err != nil || len(*calls) != 2 || value == nil {
		t.Fatal(value, err, *calls)
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
		if value != nil || len(*calls) != 2 || (err == nil) != ignore || !ignore && !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal(value, err, *calls)
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
		if value != nil || len(*calls) != 2 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"list":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, *calls)
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
		if value != nil || len(*calls) != 3 || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"first", secondID}) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != page || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, ambiguous, proof, *calls)
		}
	}
}

func TestPythonCinderGroupFindRecordLateFailureDoesNotReturnFirstMatch(t *testing.T) {
	for _, physical := range []bool{false, true} {
		readCause, closeCause := errors.New("late read"), errors.New("late close")
		faulty := &groupFaultBody{text: `{"late":"partial"}`, readErr: readCause, closeErr: closeCause}
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
			want = faulty.text
		}
		if value != nil || len(*calls) != 3 || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != want || native.ResponseHeader.Get("X-Request-Id") != "actual" || physical && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || faulty.closed != 1) {
			t.Fatal(value, err, *calls)
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
		if value != nil || len(*calls) != wantCalls || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, proof, *calls)
		}
	}
}

func TestPythonCinderGroupFindRecordRejectedReadCloseNeverFallsBack(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause, closeCause := errors.New("member read"), errors.New("member close")
		faulty := &groupFaultBody{text: `{"member":"partial"}`, readErr: readCause, closeErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(code, ""); reply.Body = faulty; return reply })
		hooks := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			return nil
		}
		value, err := api.FindRecord(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != code || string(native.Body) != faulty.text || native.ResponseHeader.Get("X-Request-Id") != "actual" || len(*calls) != 1 || faulty.closed != 1 || hooks != 0 {
			t.Fatal(value, err, *calls, faulty.closed, hooks)
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
	if err != nil || value == nil || len(*calls) != 2 || (*calls)[1].path != groupBase {
		t.Fatal(value, err, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("X-Policy") != "snapshot" {
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
		if err := invoke(api); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, *calls)
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
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || len(*calls) != 0 {
		t.Fatal(err, *calls)
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
		if err != nil || value == nil || len(*calls) != 2 || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume "+wantGet || (*calls)[0].headers.Get("X-OpenStack-Volume-API-Version") != wantGet || (*calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.13" || (*calls)[1].headers.Get("X-OpenStack-Volume-API-Version") != "3.13" || (*calls)[0].headers.Get("Accept") != "application/source-custom" || (*calls)[1].headers.Get("Accept") != "application/json" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.13" || len(value.Discovery) != 0 {
			t.Fatal(value, err, *calls, api.RawClient())
		}
		for _, call := range *calls {
			if call.headers.Get("X-Original") != "kept" {
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
	if err != nil || value == nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("Accept") != "per-call" {
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
		if err != nil || value == nil || len(*calls) != 3 || len(value.Discovery) != 1 || value.Discovery[0].StatusCode != 200 || value.StatusCode != 202 || (*calls)[2].headers.Get("OpenStack-API-Version") != "volume 3.38" || api.RawClient().Microversion != "" {
			t.Fatal(value, err, *calls)
		}
		if override {
			if (*calls)[0].path != groupBase+"/target" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.99" || (*calls)[1].path != "/cinder/v3/" {
				t.Fatal(*calls)
			}
		} else if (*calls)[0].path != "/cinder/v3/" || (*calls)[1].path != groupBase+"/target" || (*calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.38" {
			t.Fatal(*calls)
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
	if value != nil || len(*calls) != 1 || (*calls)[0].path != "/cinder/v3/" || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"discovery":"denied"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
		t.Fatal(value, err, *calls)
	}
}

func TestPythonCinderGroupFindRecordAcceptedTransientSourceMutationKeepsReceipt(t *testing.T) {
	var api *groups.API
	faulty := &groupFaultBody{text: `{"group":{"id":"actual"}}`}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(200, ""); reply.Body = faulty; return reply })
	original := api.RawClient().ResourceBase
	faulty.afterRead = func() { api.RawClient().ResourceBase = "https://other.example/v3/project/" }
	faulty.afterClose = func() { api.RawClient().ResourceBase = original }
	value, err := api.FindRecord(context.Background(), "target")
	var proof *resource.ResponseError
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != faulty.text || proof.Header.Get("X-Request-Id") != "actual" || len(*calls) != 1 || faulty.closed != 1 {
		t.Fatal(value, err, proof, *calls)
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
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"retry":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" || len(*calls) != 1 {
				t.Fatal(value, err, *calls)
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
	if err != nil || value == nil || len(*calls) != 2 || (*calls)[1].headers.Get("X-Retry") != "allowed" || (*calls)[1].headers.Get("X-Auth-Token") != "live-refreshed" {
		t.Fatal(value, err, *calls)
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
	if err != nil || value == nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	query, _ := url.ParseQuery((*calls)[1].query)
	if len(query) != 0 {
		t.Fatal("Group name match remains local", query)
	}
	groupField(t, value, "id", `"found"`)
}

func TestPythonCinderGroupFindRecordAcceptedPhysicalCausesCannotBecomeFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause := gophercloud.ErrUnexpectedResponseCode{Actual: code, Body: []byte(`{"nested":"not-physical-status"}`)}
		closeCause := errors.New("accepted close")
		faulty := &groupFaultBody{text: `{"group":{"id":"partial"}}`, readErr: readCause, closeErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { reply := groupWire(202, ""); reply.Body = faulty; return reply })
		hooks := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			return nil
		}
		value, err := api.FindRecord(context.Background(), "target")
		var proof *resource.ResponseError
		if value != nil || !errors.Is(err, closeCause) || !gophercloud.ResponseCodeIs(err, code) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != faulty.text || proof.Header.Get("X-Request-Id") != "actual" || len(*calls) != 1 || faulty.closed != 1 || hooks != 0 {
			t.Fatal(value, err, proof, *calls, faulty.closed, hooks)
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
				start := len(*calls)
				result, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || result == nil || len(*calls) != start+2 || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.38" {
					t.Fatal(result, err, *calls, api.RawClient().MoreHeaders)
				}
				for _, call := range (*calls)[start:] {
					if call.headers.Get("X-Custom") != value || call.headers.Get("X-Original") != "source" {
						t.Fatal(call, value)
					}
				}
				if (*calls)[start].headers.Get("Accept") != "application/source" || (*calls)[start+1].headers.Get("Accept") != "application/json" {
					t.Fatal((*calls)[start:])
				}
			}
		})
	}
}
