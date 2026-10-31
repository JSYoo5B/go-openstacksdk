package request_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type parityOpts struct {
	Name string `json:"name"`
}

type parityCaps struct {
	body, query, headers bool
	arguments            []string
}

func (c parityCaps) request() request.Capabilities {
	return request.Capabilities{Body: c.body, Query: c.query, Headers: c.headers, Arguments: c.arguments}
}

// genericResult is what generated code produced before ApplyCalls: Apply
// followed by ValidateCapabilities.
func genericResult[T any](base T, caps parityCaps, options ...request.Option[T]) (request.Config[T], error) {
	cfg, err := request.Apply(base, options...)
	if err != nil {
		return cfg, err
	}
	return cfg, request.ValidateCapabilities(cfg, caps.body, caps.query, caps.headers, caps.arguments...)
}

func sameResult[T any](t *testing.T, name string, wantCfg request.Config[T], wantErr error, gotCfg request.Config[T], gotErr error) {
	t.Helper()
	if (wantErr == nil) != (gotErr == nil) {
		t.Fatalf("%s: generic error %v, ApplyCalls error %v", name, wantErr, gotErr)
	}
	if wantErr != nil {
		if wantErr.Error() != gotErr.Error() || !errors.Is(gotErr, resource.ErrInvalidOption) {
			t.Fatalf("%s: generic error %q, ApplyCalls error %q", name, wantErr, gotErr)
		}
		return
	}
	if !reflect.DeepEqual(wantCfg, gotCfg) {
		t.Fatalf("%s: generic config %#v, ApplyCalls config %#v", name, wantCfg, gotCfg)
	}
}

// TestApplyCallsMatchesGenericOptions pins that every shared call standing in
// for a generic option builds the same Config and fails with byte-identical
// errors, including ValidateCapabilities' checks.
func TestApplyCallsMatchesGenericOptions(t *testing.T) {
	type O = request.Option[parityOpts]
	all := parityCaps{body: true, query: true, headers: true, arguments: []string{"extra"}}
	cases := []struct {
		name    string
		caps    parityCaps
		generic []O
		shared  []request.Call
	}{
		{"none", all, nil, nil},
		{"field", all, []O{request.WithField[parityOpts]("color", map[string]any{"a": 1})}, []request.Call{request.Field("color", map[string]any{"a": 1})}},
		{"field overwrite", all, []O{request.WithField[parityOpts]("k", 1), request.WithField[parityOpts]("k", nil)}, []request.Call{request.Field("k", 1), request.Field("k", nil)}},
		{"field empty key", all, []O{request.WithField[parityOpts]("", 1)}, []request.Call{request.Field("", 1)}},
		{"field blank key", all, []O{request.WithField[parityOpts](" \t", 1)}, []request.Call{request.Field(" \t", 1)}},
		{"field marshal error", all, []O{request.WithField[parityOpts]("bad", func() {})}, []request.Call{request.Field("bad", func() {})}},
		{"field NaN", all, []O{request.WithField[parityOpts]("nan", math.NaN())}, []request.Call{request.Field("nan", math.NaN())}},
		{"field empty key before marshal error", all, []O{request.WithField[parityOpts]("", func() {})}, []request.Call{request.Field("", func() {})}},
		{"field unsupported", parityCaps{query: true, headers: true}, []O{request.WithField[parityOpts]("k", 1)}, []request.Call{request.Field("k", 1)}},
		{"header", all, []O{request.WithHeader[parityOpts]("x-trace-id", "v")}, []request.Call{request.Header("x-trace-id", "v")}},
		{"header empty value", all, []O{request.WithHeader[parityOpts]("X-A", "")}, []request.Call{request.Header("X-A", "")}},
		{"header empty key", all, []O{request.WithHeader[parityOpts]("", "v")}, []request.Call{request.Header("", "v")}},
		{"header space", all, []O{request.WithHeader[parityOpts]("X A", "v")}, []request.Call{request.Header("X A", "v")}},
		{"header colon", all, []O{request.WithHeader[parityOpts]("X:A", "v")}, []request.Call{request.Header("X:A", "v")}},
		{"header value newline", all, []O{request.WithHeader[parityOpts]("X-A", "v\r\nX-B: w")}, []request.Call{request.Header("X-A", "v\r\nX-B: w")}},
		{"header unsupported", parityCaps{body: true, query: true}, []O{request.WithHeader[parityOpts]("X-A", "v")}, []request.Call{request.Header("X-A", "v")}},
		{"query", all, []O{request.WithQuery[parityOpts]("name", "a b")}, []request.Call{request.Query("name", "a b")}},
		{"query replace", all, []O{request.WithQuery[parityOpts]("name", "a"), request.WithQuery[parityOpts]("name", "")}, []request.Call{request.Query("name", "a"), request.Query("name", "")}},
		{"query empty key", all, []O{request.WithQuery[parityOpts]("", "a")}, []request.Call{request.Query("", "a")}},
		{"query unsupported", parityCaps{body: true, headers: true}, []O{request.WithQuery[parityOpts]("name", "a")}, []request.Call{request.Query("name", "a")}},
		{"query values", all, []O{request.WithQueryValues[parityOpts]("tag", "a", "b")}, []request.Call{request.Query("tag", []string{"a", "b"})}},
		{"query values replace", all, []O{request.WithQuery[parityOpts]("tag", "x"), request.WithQueryValues[parityOpts]("tag", "a")}, []request.Call{request.Query("tag", "x"), request.Query("tag", []string{"a"})}},
		{"query values remove", all, []O{request.WithQuery[parityOpts]("tag", "x"), request.WithQueryValues[parityOpts]("tag")}, []request.Call{request.Query("tag", "x"), request.Query("tag", []string{})}},
		{"query nil values remove", all, []O{request.WithQuery[parityOpts]("tag", "x"), request.WithQueryValues[parityOpts]("tag")}, []request.Call{request.Query("tag", "x"), request.Query("tag", []string(nil))}},
		{"query values empty key", all, []O{request.WithQueryValues[parityOpts](" ", "a")}, []request.Call{request.Query(" ", []string{"a"})}},
		{"options", all, []O{request.WithOptions(parityOpts{Name: "n"})}, []request.Call{request.Custom(request.WithOptions(parityOpts{Name: "n"}))}},
		{"argument", all, []O{request.WithArgument[parityOpts]("extra", 3)}, []request.Call{request.Custom(request.WithArgument[parityOpts]("extra", 3))}},
		{"argument unsupported", all, []O{request.WithArgument[parityOpts]("other", 3)}, []request.Call{request.Custom(request.WithArgument[parityOpts]("other", 3))}},
		{"func literal", all, []O{func(c *request.Config[parityOpts]) error { c.Options.Name = "f"; return nil }}, []request.Call{request.Custom(func(c *request.Config[parityOpts]) error { c.Options.Name = "f"; return nil })}},
		{"func error", all, []O{func(*request.Config[parityOpts]) error { return resource.ErrInvalidOption }}, []request.Call{request.Custom(func(*request.Config[parityOpts]) error { return resource.ErrInvalidOption })}},
		{"nil option", all, []O{request.WithQuery[parityOpts]("a", "b"), nil}, []request.Call{request.Query("a", "b"), nil}},
		{"nil custom", all, []O{nil}, []request.Call{request.Custom[request.Config[parityOpts]](nil)}},
		{"apply error before capability error", parityCaps{}, []O{request.WithField[parityOpts]("k", 1), request.WithHeader[parityOpts]("", "v")}, []request.Call{request.Field("k", 1), request.Header("", "v")}},
		{"capability order", parityCaps{}, []O{request.WithHeader[parityOpts]("X-A", "v"), request.WithQuery[parityOpts]("q", "v"), request.WithField[parityOpts]("k", 1)}, []request.Call{request.Header("X-A", "v"), request.Query("q", "v"), request.Field("k", 1)}},
		{"mixed order", all, []O{request.WithOptions(parityOpts{Name: "a"}), request.WithField[parityOpts]("k", "v"), request.WithHeader[parityOpts]("x-a", "1"), request.WithQuery[parityOpts]("q", "1"), request.WithArgument[parityOpts]("extra", "e")}, []request.Call{request.Custom(request.WithOptions(parityOpts{Name: "a"})), request.Field("k", "v"), request.Header("x-a", "1"), request.Query("q", "1"), request.Custom(request.WithArgument[parityOpts]("extra", "e"))}},
	}
	for _, tc := range cases {
		wantCfg, wantErr := genericResult(parityOpts{Name: "base"}, tc.caps, tc.generic...)
		gotCfg, gotErr := request.ApplyCalls(parityOpts{Name: "base"}, tc.caps.request(), tc.shared...)
		sameResult(t, tc.name, wantCfg, wantErr, gotCfg, gotErr)
	}
}

func TestApplyCallsMatchesNilTypedOptions(t *testing.T) {
	type P = *parityOpts
	wantCfg, wantErr := genericResult[P](nil, parityCaps{})
	gotCfg, gotErr := request.ApplyCalls[P, request.Call](nil, request.Capabilities{})
	sameResult(t, "nil base", wantCfg, wantErr, gotCfg, gotErr)
	wantCfg, wantErr = genericResult(&parityOpts{}, parityCaps{}, request.WithOptions[P](nil))
	gotCfg, gotErr = request.ApplyCalls(&parityOpts{}, request.Capabilities{}, request.Custom(request.WithOptions[P](nil)))
	sameResult(t, "nil replacement", wantCfg, wantErr, gotCfg, gotErr)
}

// TestApplyCallsRejectsWhatNativeOperationsCannotCarry covers inputs without
// a generic counterpart: JSON query values, Custom over another state type,
// and SDK-owned controls.
func TestApplyCallsRejectsWhatNativeOperationsCannotCarry(t *testing.T) {
	all := request.Capabilities{Body: true, Query: true, Headers: true}
	for name, tc := range map[string]struct {
		caps request.Capabilities
		call request.Call
		want string
	}{
		"json query":            {all, request.Query("limit", 5), `invalid option: query "limit" needs a string or []string value for this operation`},
		"json query empty key":  {all, request.Query("", 5), "invalid option: empty query key"},
		"json query no support": {request.Capabilities{Body: true, Headers: true}, request.Query("limit", 5), "invalid option: query extensions are unsupported for this operation"},
		"foreign custom":        {all, request.Custom(func(*parityOpts) error { return nil }), "invalid option: Custom call option is unsupported by this operation"},
		"foreign config":        {all, request.Custom(func(*request.Config[struct{}]) error { return nil }), "invalid option: Custom call option is unsupported by this operation"},
		"headers":               {all, request.Headers(nil), "invalid option: Headers call option is unsupported by this operation"},
		"microversion":          {all, request.Microversion("2.1"), "invalid option: Microversion call option is unsupported by this operation"},
		"base path":             {all, request.BasePath("x"), "invalid option: BasePath call option is unsupported by this operation"},
		"ignore missing":        {all, request.IgnoreMissing(true), "invalid option: IgnoreMissing call option is unsupported by this operation"},
		"limit":                 {all, request.Limit(1), "invalid option: Limit call option is unsupported by this operation"},
		"marker":                {all, request.Marker("m"), "invalid option: Marker call option is unsupported by this operation"},
		"max items":             {all, request.MaxItems(1), "invalid option: MaxItems call option is unsupported by this operation"},
		"paginated":             {all, request.Paginated(true), "invalid option: Paginated call option is unsupported by this operation"},
	} {
		_, err := request.ApplyCalls(parityOpts{}, tc.caps, tc.call)
		if !errors.Is(err, resource.ErrInvalidOption) || err.Error() != tc.want {
			t.Fatalf("%s: %v", name, err)
		}
	}
	// Unsupported fields are reported before a JSON query, in
	// ValidateCapabilities' order.
	_, err := request.ApplyCalls[parityOpts, request.Call](parityOpts{}, request.Capabilities{}, request.Query("limit", 5), request.Field("k", 1))
	if err == nil || err.Error() != "invalid option: JSON field extensions are unsupported for this operation" {
		t.Fatal(err)
	}
}

// Generated operation types are aliases of a category, so a typed slice of
// that category is accepted by ApplyCalls without conversion.
func TestApplyCallsAcceptsCategorySlices(t *testing.T) {
	options := []request.NativeFieldHeaderCall{request.Field("k", 1), request.Header("x-a", "v"), request.Custom(request.WithOptions(parityOpts{Name: "n"}))}
	cfg, err := request.ApplyCalls(parityOpts{}, request.Capabilities{Body: true, Headers: true}, options...)
	if err != nil || cfg.Options.Name != "n" || string(cfg.Fields["k"]) != "1" || cfg.Headers["X-A"] != "v" {
		t.Fatal(cfg, err)
	}
}

// Unchecked is Apply alone: no capability or argument check after the calls.
func TestApplyCallsUncheckedMatchesApply(t *testing.T) {
	generic := []request.Option[parityOpts]{request.WithField[parityOpts]("k", 1), request.WithHeader[parityOpts]("x-a", "v"), request.WithQuery[parityOpts]("q", "v"), request.WithArgument[parityOpts]("any", 1)}
	shared := []request.Call{request.Field("k", 1), request.Header("x-a", "v"), request.Query("q", "v"), request.Custom(request.WithArgument[parityOpts]("any", 1))}
	wantCfg, wantErr := request.Apply(parityOpts{}, generic...)
	gotCfg, gotErr := request.ApplyCalls(parityOpts{}, request.Capabilities{Unchecked: true}, shared...)
	sameResult(t, "unchecked", wantCfg, wantErr, gotCfg, gotErr)
	_, err := request.ApplyCalls(parityOpts{}, request.Capabilities{Unchecked: true}, request.Query("limit", 5))
	if err == nil || err.Error() != `invalid option: query "limit" needs a string or []string value for this operation` {
		t.Fatal(err)
	}
}
