package calls

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type state struct {
	Headers      map[string]string
	Microversion *string
	BasePath     *string
	Ignore       *bool
	Keywords     []string
}

func keyword(s *state, key string, raw json.RawMessage, err error) error {
	if err != nil {
		return err
	}
	s.Keywords = append(s.Keywords, key+"="+string(raw))
	return nil
}

func fullSpec() Spec[state] {
	header, headers := HeaderMap(func(s *state) *map[string]string { return &s.Headers })
	return Spec[state]{
		Header: header, Headers: headers,
		Microversion:  Pointer(func(s *state) **string { return &s.Microversion }),
		BasePath:      Pointer(func(s *state) **string { return &s.BasePath }),
		IgnoreMissing: Pointer(func(s *state) **bool { return &s.Ignore }),
		Query:         JSONQuery(keyword),
		Paging:        JSONPaging(keyword),
	}
}

func run(t *testing.T, spec Spec[state], calls ...request.ListBasePathCall) (state, error) {
	t.Helper()
	var s state
	for _, apply := range Translate(calls, spec) {
		if apply == nil {
			return s, errors.New("nil")
		}
		if err := apply(&s); err != nil {
			return s, err
		}
	}
	return s, nil
}

func TestTranslateAppliesInCallerOrder(t *testing.T) {
	s, err := run(t, fullSpec(),
		request.Header("A", "1"),
		request.Headers(map[string]string{"B": "2"}),
		request.Header("C", "3"),
		request.Microversion("1.1"), request.Microversion("1.2"),
		request.BasePath("/x"),
		request.Query("q", "text"), request.Query("n", 5),
		request.Limit(2), request.Marker("m"), request.MaxItems(3), request.Paginated(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Headers, map[string]string{"B": "2", "C": "3"}) {
		t.Fatalf("headers = %v", s.Headers)
	}
	if *s.Microversion != "1.2" || *s.BasePath != "/x" {
		t.Fatalf("state = %+v", s)
	}
	want := []string{`q="text"`, `n=5`, `limit=2`, `marker="m"`, `max_items=3`, `paginated=false`}
	if !reflect.DeepEqual(s.Keywords, want) {
		t.Fatalf("keywords = %v", s.Keywords)
	}
}

func TestTranslateNilAndCustom(t *testing.T) {
	custom := request.Custom(func(s *state) error { s.Keywords = append(s.Keywords, "custom"); return nil })
	out := Translate([]request.ListBasePathCall{nil, request.Custom[state](nil), custom, request.Custom(func(*int) error { return nil })}, fullSpec())
	if out[0] != nil || out[1] != nil {
		t.Fatal("nil call or nil Custom must stay nil")
	}
	var s state
	if err := out[2](&s); err != nil || len(s.Keywords) != 1 {
		t.Fatalf("custom = %v %v", err, s.Keywords)
	}
	if err := out[3](&s); !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "Custom call option is unsupported") {
		t.Fatalf("foreign Custom = %v", err)
	}
}

func TestTranslateUnsupportedControls(t *testing.T) {
	for _, call := range []request.ListBasePathCall{request.Header("A", "1"), request.Headers(nil), request.Microversion("1"), request.BasePath("/"), request.Query("k", "v"), request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(true)} {
		_, err := run(t, Spec[state]{}, call)
		entry, _ := request.Entry(call)
		if err == nil || err.Error() != request.UnsupportedCall(entry).Error() {
			t.Fatalf("%s: err = %v", entry.Kind, err)
		}
	}
}

func TestTranslateCopiesHeaders(t *testing.T) {
	input := map[string]string{"A": "1"}
	call := request.Headers(input)
	input["A"] = "changed"
	s, err := run(t, fullSpec(), call)
	if err != nil || s.Headers["A"] != "1" {
		t.Fatalf("headers = %v %v", s.Headers, err)
	}
	s.Headers["A"] = "mutated"
	again, _ := run(t, fullSpec(), call)
	if again.Headers["A"] != "1" {
		t.Fatal("Headers call shares its map")
	}
}

type opts struct{ N int }

func TestApplyConfigMatchesRequestApply(t *testing.T) {
	spec := Spec[request.Config[opts]]{Header: ConfigHeader[opts], Field: ConfigField[opts], Query: ConfigQuery[opts]}
	got, err := ApplyConfig[opts, request.WriteCall](opts{N: 1}, spec,
		request.Header("x-a", "1"), request.Field("f", []int{1}),
		request.Custom(request.WithOptions(opts{N: 2})))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := request.Apply(opts{N: 1}, request.WithHeader[opts]("x-a", "1"), request.WithField[opts]("f", []int{1}), request.WithOptions(opts{N: 2}))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	_, err = ApplyConfig[opts, request.WriteCall](opts{}, spec, nil)
	_, wantErr := request.Apply(opts{}, nil)
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("nil call = %v", err)
	}
	_, err = ApplyConfig(opts{}, spec, request.Field(" ", 1))
	_, wantErr = request.Apply(opts{}, request.WithField[opts](" ", 1))
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("empty field = %v", err)
	}
	_, err = ApplyConfig(opts{}, spec, request.Field("f", func() {}))
	_, wantErr = request.Apply(opts{}, request.WithField[opts]("f", func() {}))
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("bad field = %v want %v", err, wantErr)
	}
	_, err = ApplyConfig[opts, request.WriteCall](opts{}, spec, request.Microversion("1"))
	if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "Microversion call option is unsupported") {
		t.Fatalf("microversion = %v", err)
	}
}

func TestConfigQueryForms(t *testing.T) {
	spec := Spec[request.Config[opts]]{Query: ConfigQuery[opts]}
	got, err := ApplyConfig[opts, request.ListCall](opts{}, spec, request.Query("a", "1"), request.Query("b", []string{"x", "y"}), request.Query("b", []string{}))
	if err != nil || got.Query.Get("a") != "1" || got.Query.Has("b") {
		t.Fatalf("query = %v %v", got.Query, err)
	}
	_, err = ApplyConfig(opts{}, spec, request.Query("n", 1))
	if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), `query "n" needs a string`) {
		t.Fatalf("json query = %v", err)
	}
	_, err = ApplyConfig(opts{}, spec, request.Query(" ", "v"))
	_, wantErr := request.Apply(opts{}, request.WithQuery[opts](" ", "v"))
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("empty key = %v", err)
	}
}

func TestMergeHeaderUsesTheOperationMapOption(t *testing.T) {
	merge := func(values map[string]string) request.GetCall {
		return request.Custom(func(s *state) error {
			if _, bad := values["Bad"]; bad {
				return resource.ErrInvalidOption
			}
			if s.Headers == nil {
				s.Headers = map[string]string{}
			}
			for k, v := range values {
				s.Headers[k] = v
			}
			return nil
		})
	}
	spec := Spec[state]{Header: MergeHeader[state](merge)}
	var s state
	for _, apply := range Translate([]request.GetCall{merge(map[string]string{"A": "1"}), request.Header("B", "2"), request.Header("A", "3")}, spec) {
		if err := apply(&s); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(s.Headers, map[string]string{"A": "3", "B": "2"}) {
		t.Fatal(s.Headers)
	}
	if err := Translate([]request.GetCall{request.Header("Bad", "x")}, spec)[0](&s); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	other := Spec[state]{Header: MergeHeader[state](func(map[string]string) request.GetCall { return request.Header("X", "y") })}
	if err := Translate([]request.GetCall{request.Header("A", "1")}, other)[0](&s); !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "Header call option is unsupported") {
		t.Fatal(err)
	}
}

func TestOptionsKeepNilAndOrder(t *testing.T) {
	spec := Spec[request.Config[opts]]{Header: ConfigHeader[opts]}
	options := Options[opts, request.GetCall](nil, spec)
	if len(options) != 0 {
		t.Fatal(options)
	}
	options = Options([]request.GetCall{request.Header("x-a", "1"), nil, request.Custom[request.Config[opts]](nil), request.Header("x-a", "2"), request.Microversion("1")}, spec)
	if len(options) != 5 || options[1] != nil || options[2] != nil {
		t.Fatalf("options = %v", options)
	}
	got, err := request.Apply(opts{}, options[0], options[3])
	if err != nil || got.Headers["X-A"] != "2" {
		t.Fatalf("got %+v %v", got.Headers, err)
	}
	if err := options[4](&got); !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "Microversion call option is unsupported") {
		t.Fatalf("microversion = %v", err)
	}
}
