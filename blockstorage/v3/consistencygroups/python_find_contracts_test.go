package consistencygroups_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderConsistencyGroupFindRecordDirectUntypedProjectionAndLiteralIdentity(t *testing.T) {
	for _, body := range []string{`{"consistencygroup":{"id":null,"name":false,"volume_types":"one","cls":false,"unknown":9007199254740993}}`, `{"consistencygroup":{"id":9007199254740993}}`, `{"consistencygroup":{}}`, `{bad`} {
		api, calls := cgAPI(t, "3.90", func(*http.Request) *http.Response { return cgWire(200, body) })
		value, err := api.FindRecord(context.Background(), "literal/id %2F")
		if err != nil || value == nil || value.Resource == nil || len(*calls) != 1 || (*calls)[0].escaped != cgBase+"/literal%2Fid%20%252F" || (*calls)[0].query != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.90" || string(value.Envelope) != body || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, *calls)
		}
		if body == `{bad` || body == `{"consistencygroup":{}}` {
			cgField(t, value, "id", `"literal/id %2F"`)
		}
		if strings.Contains(body, `"cls"`) {
			cgField(t, value, "id", `null`)
			cgField(t, value, "name", `false`)
			cgField(t, value, "volume_types", `["one"]`)
			if string(value.Wire.Body["cls"]) != `false` || value.Resource.Body["cls"] != nil || string(value.Wire.Body["unknown"]) != `9007199254740993` {
				t.Fatal(value)
			}
		}
	}
}

func TestPythonCinderConsistencyGroupFindRecordCleanFallbackHasNoServerNameQuery(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, details := range []bool{true, false} {
			t.Run(http.StatusText(code)+"/"+map[bool]string{true: "detail", false: "summary"}[details], func(t *testing.T) {
				path := cgBase
				if details {
					path += "/detail"
				}
				api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
					if r.URL.Path == cgBase+"/target" {
						return cgWire(code, `{"member":"unavailable"}`)
					}
					if r.URL.Path != path || r.URL.Query().Has("name") {
						t.Fatal("Group has no declared server name query", r.URL)
					}
					if r.URL.Query().Get("offset") == "2" {
						return cgWire(202, `{"consistencygroups":[{"id":"found","name":"target","group_type":null}]}`)
					}
					return cgWire(200, `{"consistencygroups":[{"id":"other","name":"other"}],"next":"?offset=2"}`)
				})
				options := []consistencygroups.RecordFindOption{}
				if !details {
					options = append(options, consistencygroups.WithRecordFindDetails(false))
				}
				value, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || value == nil || len(*calls) != 3 || value.StatusCode != 202 || (*calls)[0].query != "" || (*calls)[1].query != "" || (*calls)[2].query != "offset=2" {
					t.Fatal(value, err, *calls)
				}
				cgField(t, value, "id", `"found"`)
			})
		}
	}
	api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == cgBase+"/target" {
			return cgWire(404, `{"member":false}`)
		}
		return cgWire(200, `{"consistencygroups":[{"id":"target","name":"different"},{"id":false,"name":false},{"id":null,"name":"other"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "target")
	if err != nil || len(*calls) != 2 || value == nil {
		t.Fatal(value, err, *calls)
	}
	cgField(t, value, "name", `"different"`)
}

func TestPythonCinderConsistencyGroupFindRecordMissingRequiresSuccessfulList(t *testing.T) {
	for _, ignore := range []bool{true, false} {
		api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == cgBase+"/missing" {
				return cgWire(403, `{"member":"denied"}`)
			}
			return cgWire(200, `{"consistencygroups":[]}`)
		})
		options := []consistencygroups.RecordFindOption{}
		if !ignore {
			options = append(options, consistencygroups.WithRecordFindIgnoreMissing(false))
		}
		value, err := api.FindRecord(context.Background(), "missing", options...)
		if value != nil || len(*calls) != 2 || (err == nil) != ignore || !ignore && !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal(value, err, *calls)
		}
	}
	for _, code := range []int{403, 404, 503} {
		api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == cgBase+"/missing" {
				return cgWire(404, `{"member":"missing"}`)
			}
			return cgWire(code, `{"list":"actual"}`)
		})
		value, err := api.FindRecord(context.Background(), "missing")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || len(*calls) != 2 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"list":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, *calls)
		}
	}
}

func TestPythonCinderConsistencyGroupFindRecordLateDuplicatesKeepAcceptedPage(t *testing.T) {
	for _, secondID := range []string{"second", "first"} {
		page := `{"consistencygroups":[{"id":"` + secondID + `","name":"target"}],"proof":"duplicate-page"}`
		api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == cgBase+"/target" {
				return cgWire(404, `{"member":false}`)
			}
			if r.URL.Query().Get("marker") == "next" {
				return cgWire(202, page)
			}
			return cgWire(200, `{"consistencygroups":[{"id":"first","name":"target"}],"next":"?marker=next"}`)
		})
		value, err := api.FindRecord(context.Background(), "target")
		var ambiguous *resource.AmbiguousError
		var proof *resource.ResponseError
		if value != nil || len(*calls) != 3 || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"first", secondID}) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != page || proof.Header.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, ambiguous, proof, *calls)
		}
	}
}

func TestPythonCinderConsistencyGroupFindRecordLateFailureDoesNotReturnFirstMatch(t *testing.T) {
	for _, physical := range []bool{false, true} {
		readCause, closeCause := errors.New("late read"), errors.New("late close")
		faulty := &cgFaultBody{text: `{"late":"partial"}`, readErr: readCause, closeErr: closeCause}
		api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
			if r.URL.Path == cgBase+"/target" {
				return cgWire(404, `{"member":false}`)
			}
			if r.URL.Query().Has("marker") {
				reply := cgWire(503, `{"late":"actual"}`)
				if physical {
					reply.Body = faulty
				}
				return reply
			}
			return cgWire(200, `{"consistencygroups":[{"id":"first","name":"target"}],"next":"?marker=next"}`)
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

func TestPythonCinderConsistencyGroupFindRecordAcceptedErrorsAndListConstructorAreTerminal(t *testing.T) {
	for _, stage := range []string{"member", "list"} {
		body := `{"consistencygroup":null}`
		if stage == "list" {
			body = `{"consistencygroups":[{"id":"first","name":"target"},{"id":"second","cls":null}]}`
		}
		api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
			if stage == "list" && r.URL.Path == cgBase+"/target" {
				return cgWire(404, `{"member":false}`)
			}
			return cgWire(202, body)
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

func TestPythonCinderConsistencyGroupFindRecordRejectedReadCloseNeverFallsBack(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause, closeCause := errors.New("member read"), errors.New("member close")
		faulty := &cgFaultBody{text: `{"member":"partial"}`, readErr: readCause, closeErr: closeCause}
		api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response { reply := cgWire(code, ""); reply.Body = faulty; return reply })
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

func TestPythonCinderConsistencyGroupFindRecordOptionsOwnBothPhaseSnapshots(t *testing.T) {
	details := false
	headers := map[string]string{"X-Policy": "snapshot"}
	option := consistencygroups.WithRecordFindOpts(consistencygroups.RecordFindOpts{Details: &details, Headers: headers})
	details = true
	headers["X-Policy"] = "changed"
	var held *consistencygroups.RecordFindOpts
	api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == cgBase+"/target" {
			if held != nil {
				held.Headers["X-Policy"] = "retained-mutated"
				*held.Details = true
			}
			return cgWire(404, `{"member":false}`)
		}
		return cgWire(200, `{"consistencygroups":[{"id":"found","name":"target"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "target", option, func(v *consistencygroups.RecordFindOpts) error { held = v; return nil })
	if err != nil || value == nil || len(*calls) != 2 || (*calls)[1].path != cgBase {
		t.Fatal(value, err, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("X-Policy") != "snapshot" {
			t.Fatal(call)
		}
	}
}

func TestPythonCinderConsistencyGroupFindRecordPreflightAndStickyOriginalSource(t *testing.T) {
	for _, invoke := range []func(*consistencygroups.API) error{
		func(api *consistencygroups.API) error { _, err := api.FindRecord(nil, "id"); return err },
		func(api *consistencygroups.API) error {
			_, err := (*consistencygroups.API)(nil).FindRecord(context.Background(), "id")
			return err
		},
		func(api *consistencygroups.API) error {
			_, err := api.FindRecord(context.Background(), " ")
			return err
		},
		func(api *consistencygroups.API) error {
			_, err := api.FindRecord(context.Background(), "id", nil)
			return err
		},
		func(api *consistencygroups.API) error {
			_, err := api.FindRecord(context.Background(), "id", consistencygroups.WithRecordFindMicroversion("bad"))
			return err
		},
		func(api *consistencygroups.API) error {
			_, err := api.FindRecord(context.Background(), "id", consistencygroups.WithRecordFindHeaders(map[string]string{"Authorization": "forged"}))
			return err
		},
		func(api *consistencygroups.API) error {
			_, err := api.FindRecord(context.Background(), "id", consistencygroups.WithRecordFindHeaders(map[string]string{"OpenStack-API-Version": "volume 3.38"}))
			return err
		},
	} {
		api, calls := cgAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
		if err := invoke(api); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response { t.Fatal("mutated source HTTP"); return nil })
	cause := errors.New("option failure")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err := api.FindRecord(ctx, "id", func(*consistencygroups.RecordFindOpts) error {
		api.RawClient().ResourceBase = "https://other.example/v3/project/"
		cancel(cause)
		return cause
	})
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

func TestPythonCinderConsistencyGroupFindRecordGetOverrideFallbackOriginalVersionAndHeaders(t *testing.T) {
	for _, override := range []string{"3.90", ""} {
		api, calls := cgAPI(t, "3.13", func(r *http.Request) *http.Response {
			if r.URL.Path == cgBase+"/target" {
				return cgWire(404, `{"member":false}`)
			}
			return cgWire(200, `{"consistencygroups":[{"id":"found","name":"target"}]}`)
		})
		original := map[string]string{"OpenStack-API-Version": "volume 3.13", "X-OpenStack-Volume-API-Version": "3.13", "Accept": "application/source-custom", "X-Original": "kept"}
		api.RawClient().MoreHeaders = original
		expected := make(map[string]string, len(original))
		for key, value := range original {
			expected[key] = value
		}
		value, err := api.FindRecord(context.Background(), "target", consistencygroups.WithRecordFindMicroversion(override))
		wantGet := override
		if wantGet == "" {
			wantGet = "3.13"
		}
		if err != nil || value == nil || len(*calls) != 2 || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume "+wantGet || (*calls)[0].headers.Get("X-OpenStack-Volume-API-Version") != wantGet || (*calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.13" || (*calls)[1].headers.Get("X-OpenStack-Volume-API-Version") != "3.13" || (*calls)[0].headers.Get("Accept") != "application/source-custom" || (*calls)[1].headers.Get("Accept") != "application/json" || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.13" {
			t.Fatal(value, err, *calls, api.RawClient())
		}
		for _, call := range *calls {
			if call.headers.Get("X-Original") != "kept" {
				t.Fatal(call)
			}
		}
	}
	api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == cgBase+"/target" {
			return cgWire(404, `{"member":false}`)
		}
		return cgWire(200, `{"consistencygroups":[{"id":"found","name":"target"}]}`)
	})
	api.RawClient().MoreHeaders = map[string]string{"Accept": "source"}
	value, err := api.FindRecord(context.Background(), "target", consistencygroups.WithRecordFindHeaders(map[string]string{"accept": "per-call"}))
	if err != nil || value == nil || len(*calls) != 2 {
		t.Fatal(value, err, *calls)
	}
	for _, call := range *calls {
		if call.headers.Get("Accept") != "per-call" {
			t.Fatal(call)
		}
	}
}

func TestPythonCinderConsistencyGroupFindRecordAcceptedTransientSourceMutationKeepsReceipt(t *testing.T) {
	var api *consistencygroups.API
	faulty := &cgFaultBody{text: `{"consistencygroup":{"id":"actual"}}`}
	api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response { reply := cgWire(200, ""); reply.Body = faulty; return reply })
	original := api.RawClient().ResourceBase
	faulty.afterRead = func() { api.RawClient().ResourceBase = "https://other.example/v3/project/" }
	faulty.afterClose = func() { api.RawClient().ResourceBase = original }
	value, err := api.FindRecord(context.Background(), "target")
	var proof *resource.ResponseError
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != faulty.text || proof.Header.Get("X-Request-Id") != "actual" || len(*calls) != 1 || faulty.closed != 1 {
		t.Fatal(value, err, proof, *calls)
	}
}

func TestPythonCinderConsistencyGroupFindRecordNativeRetryOwnsVersionAndPreservesOrdinaryHeaders(t *testing.T) {
	for _, mode := range []string{"mutate_version", "add_nova", "reserved_auth", "omit_owned_version"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response { return cgWire(503, `{"retry":"actual"}`) })
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
	api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response {
		attempt++
		if attempt == 1 {
			return cgWire(503, `{"retry":"first"}`)
		}
		return cgWire(200, `{"consistencygroup":{"id":"actual"}}`)
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

func TestPythonCinderConsistencyGroupFindRecordCleanRetryOriginalKeepsFallback(t *testing.T) {
	api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Path == cgBase+"/target" {
			return cgWire(404, `{"member":"original"}`)
		}
		return cgWire(200, `{"consistencygroups":[{"id":"found","name":"target"}]}`)
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
	cgField(t, value, "id", `"found"`)
}

func TestPythonCinderConsistencyGroupFindRecordAcceptedPhysicalCausesCannotBecomeFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		readCause := gophercloud.ErrUnexpectedResponseCode{Actual: code, Body: []byte(`{"nested":"not-physical-status"}`)}
		closeCause := errors.New("accepted close")
		faulty := &cgFaultBody{text: `{"consistencygroup":{"id":"partial"}}`, readErr: readCause, closeErr: closeCause}
		api, calls := cgAPI(t, "3.38", func(*http.Request) *http.Response { reply := cgWire(202, ""); reply.Body = faulty; return reply })
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

func TestPythonCinderConsistencyGroupFindRecordFirstCustomHeaderOptionOwnsDefaultAndNilMaps(t *testing.T) {
	for _, mode := range []string{"default", "nil_opts_headers", "nil_header_option"} {
		t.Run(mode, func(t *testing.T) {
			var held *consistencygroups.RecordFindOpts
			api, calls := cgAPI(t, "3.38", func(r *http.Request) *http.Response {
				if held != nil {
					held.Headers["X-Custom"] = "retained-mutated"
				}
				if r.URL.Path == cgBase+"/target" {
					return cgWire(404, `{"member":"fallback"}`)
				}
				return cgWire(200, `{"consistencygroups":[{"id":"found","name":"target"}]}`)
			})
			original := map[string]string{"X-Original": "source", "Accept": "application/source"}
			expected := map[string]string{"X-Original": "source", "Accept": "application/source"}
			api.RawClient().MoreHeaders = original
			for _, value := range []string{"first", "second"} {
				options := []consistencygroups.RecordFindOption{}
				switch mode {
				case "nil_opts_headers":
					options = append(options, consistencygroups.WithRecordFindOpts(consistencygroups.RecordFindOpts{Headers: nil}))
				case "nil_header_option":
					options = append(options, consistencygroups.WithRecordFindHeaders(nil))
				}
				options = append(options, func(cfg *consistencygroups.RecordFindOpts) error {
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

func TestPythonCinderConsistencyGroupFindNextUsesPythonBinary64TruthinessAndActualPage(t *testing.T) {
	for _, test := range []struct {
		next   string
		truthy bool
	}{
		{`1e-400`, false}, {`-1e-400`, false}, {`-0.0`, false}, {`0`, false}, {`null`, false},
		{`1e-300`, true}, {`1e400`, true}, {`-1e400`, true},
	} {
		t.Run(test.next, func(t *testing.T) {
			body := `{"consistencygroups":[{"id":"first","name":"data"}],"next":` + test.next + `}`
			api, calls := cgAPI(t, "", func(r *http.Request) *http.Response {
				if r.URL.Path == cgBase+"/data" {
					return cgWire(404, `{"member":"actual missing"}`)
				}
				if r.URL.Path != cgBase+"/detail" || r.URL.RawQuery != "" {
					t.Fatal("unexpected continuation", r.URL)
				}
				return cgWire(203, body)
			})
			value, err := api.FindRecord(context.Background(), "data")
			if len(*calls) != 2 || (err != nil) != test.truthy {
				t.Fatal(test, value, err, *calls)
			}
			if test.truthy {
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual" {
					t.Fatal("terminal next error returned candidate or lost actual page", test, value, err, proof)
				}
			} else {
				cgField(t, value, "id", `"first"`)
				if value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "actual" {
					t.Fatal(test, value)
				}
			}
		})
	}
}
