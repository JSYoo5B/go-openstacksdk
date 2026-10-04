package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func vboWire(t *testing.T, req *http.Request, readonly bool, token string) {
	t.Helper()
	want := `{"os-update_readonly_flag":{"readonly":true}}`
	if !readonly {
		want = `{"os-update_readonly_flag":{"readonly":false}}`
	}
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
	}
	if err != nil || string(body) != want || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/literal/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != "volume 3.80" || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.80" || req.ContentLength != int64(len(want)) || len(req.TransferEncoding) != 0 {
		t.Error("readonly option changed literal flag, scope, framing, captured header or version", req.Method, req.URL, string(body), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}
func vboReply(w http.ResponseWriter) {
	w.Header().Set("X-Proof", "readonly option")
	w.WriteHeader(203)
	_, _ = w.Write([]byte("opaque acknowledgement"))
}
func vboCompleted(t *testing.T, result *blockstorage.VolumeActionResult, err error) {
	t.Helper()
	if err != nil || result == nil || !result.Completed || result.VolumeID != "literal" || result.Microversion != "3.80" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "readonly option" || string(result.Applied.Body) != "opaque acknowledgement" {
		t.Fatal(result, err)
	}
}

func TestVolumeReadonlyOptionsDefaultsFalseReplacementAndFactoryCopies(t *testing.T) {
	no := false
	cases := []struct {
		name    string
		options []blockstorage.VolumeReadonlyOption
		want    bool
	}{
		{"no options", nil, true},
		{"empty complete config", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{})}, true},
		{"explicit false", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonly(false)}, false},
		{"explicit true", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonly(true)}, true},
		{"false complete config", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{Readonly: &no})}, false},
		{"replacement restores default", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonly(false), blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{})}, true},
		{"later false wins", []blockstorage.VolumeReadonlyOption{blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{}), blockstorage.WithVolumeReadonly(false)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prepared, err := blockstorage.PrepareVolumeReadonlyOptions(vsaContext(t), tc.options...)
			if err != nil || prepared.Readonly == nil || *prepared.Readonly != tc.want {
				t.Fatal(prepared, err, tc.want)
			}
			*prepared.Readonly = !tc.want
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.80")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vboWire(t, req, tc.want, "test-token")
				vboReply(w)
			})
			result, err := blockstorage.SetVolumeReadonly(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, tc.options...)
			vboCompleted(t, result, err)
			if calls.Load() != 1 {
				t.Fatal("readonly looked up, discovered or replayed", calls.Load())
			}
		})
	}
	t.Run("factory owns caller and every prepared result", func(t *testing.T) {
		flag := false
		value := blockstorage.VolumeReadonlyOpts{Readonly: &flag}
		factory := blockstorage.WithVolumeReadonlyOptions(value)
		flag = true
		value.Readonly = nil
		for i := 0; i < 2; i++ {
			prepared, err := blockstorage.PrepareVolumeReadonlyOptions(vsaContext(t), factory)
			if err != nil || prepared.Readonly == nil || *prepared.Readonly {
				t.Fatal("factory borrowed caller or previous result", prepared, err)
			}
			*prepared.Readonly = true
			prepared.Readonly = nil
		}
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.80")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			vboWire(t, req, false, "test-token")
			vboReply(w)
		})
		result, err := blockstorage.SetVolumeReadonly(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, factory)
		vboCompleted(t, result, err)
		if calls.Load() != 1 || !flag {
			t.Fatal(calls.Load(), flag)
		}
	})
}

func TestVolumeReadonlyOriginalsOwnCallbackSliceAndRetainedPointersThroughHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.80")
	var order []int
	var calls, replacements atomic.Int32
	var firstRetained, secondRetained *blockstorage.VolumeReadonlyOpts
	var options []blockstorage.VolumeReadonlyOption
	flag := false
	options = []blockstorage.VolumeReadonlyOption{
		func(next *blockstorage.VolumeReadonlyOpts) error {
			order = append(order, 1)
			firstRetained = next
			next.Readonly = &flag
			options[1] = func(*blockstorage.VolumeReadonlyOpts) error {
				replacements.Add(1)
				return errors.New("caller replaced original callback")
			}
			client.MoreHeaders["x-source"] = "later ordinary header"
			cloud.Provider.SetToken("option-live")
			return nil
		},
		func(next *blockstorage.VolumeReadonlyOpts) error {
			order = append(order, 2)
			flag = true
			*firstRetained.Readonly = true
			if next.Readonly == nil || *next.Readonly {
				t.Error("later callback borrowed retained input", next)
			}
			secondRetained = next
			return nil
		},
		func(next *blockstorage.VolumeReadonlyOpts) error {
			order = append(order, 3)
			*secondRetained.Readonly = true
			if next.Readonly == nil || *next.Readonly {
				t.Error("next callback borrowed previous config", next)
			}
			return nil
		},
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		vboWire(t, req, false, "option-live")
		vboReply(w)
	})
	result, err := blockstorage.SetVolumeReadonly(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, options...)
	vboCompleted(t, result, err)
	if !reflect.DeepEqual(order, []int{1, 2, 3}) || calls.Load() != 1 || replacements.Load() != 0 || client.MoreHeaders["x-source"] != "later ordinary header" || client.Microversion != "3.80" {
		t.Fatal(order, calls.Load(), replacements.Load(), client)
	}
}

func TestVolumeReadonlyPrepareStopsNilContextCancellationNilCallbackAndErrors(t *testing.T) {
	for _, kind := range []string{"nil context", "already canceled", "nil callback", "callback error", "callback cancel", "callback error and cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(vsaContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("readonly prepare callback"), errors.New("readonly prepare custom cancellation")
			var selected context.Context = ctx
			first, later := 0, 0
			options := []blockstorage.VolumeReadonlyOption{
				func(next *blockstorage.VolumeReadonlyOpts) error {
					first++
					flag := false
					next.Readonly = &flag
					if kind == "callback cancel" || kind == "callback error and cancel" {
						cancel(cancelCause)
					}
					if kind == "callback error" || kind == "callback error and cancel" {
						return callbackCause
					}
					return nil
				},
				func(*blockstorage.VolumeReadonlyOpts) error { later++; return nil },
			}
			wantFirst := 1
			switch kind {
			case "nil context":
				selected = nil
				wantFirst = 0
			case "already canceled":
				cancel(cancelCause)
				wantFirst = 0
			case "nil callback":
				options[0] = nil
				wantFirst = 0
			}
			prepared, err := blockstorage.PrepareVolumeReadonlyOptions(selected, options...)
			if err == nil || prepared.Readonly != nil || first != wantFirst || later != 0 {
				t.Fatal(prepared, err, first, later)
			}
			if (kind == "nil context" || kind == "nil callback") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if (kind == "already canceled" || kind == "callback cancel" || kind == "callback error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("prepare cancellation cause lost", err)
			}
			if (kind == "callback error" || kind == "callback error and cancel") && !errors.Is(err, callbackCause) {
				t.Fatal("prepare callback cause lost", err)
			}
		})
	}
}

func TestVolumeReadonlyDirectPreflightAndStickySourceErrorsStopLaterOriginals(t *testing.T) {
	for _, kind := range []string{"nil client", "nil provider", "wrong role and unsafe ID", "unsafe ID", "nil context", "already canceled", "nil callback", "source change", "joined callback source and cancel"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.80")
			original := *client
			ctx, cancel := context.WithCancelCause(vsaContext(t))
			defer cancel(nil)
			var selected context.Context = ctx
			id := "literal"
			callbackCause, cancelCause := errors.New("readonly original callback"), errors.New("readonly original custom cancellation")
			var calls, first, later atomic.Int32
			options := []blockstorage.VolumeReadonlyOption{
				func(next *blockstorage.VolumeReadonlyOpts) error {
					first.Add(1)
					flag := false
					next.Readonly = &flag
					if kind == "source change" || kind == "joined callback source and cancel" {
						client.ResourceBase += "changed/"
					}
					if kind == "joined callback source and cancel" {
						cancel(cancelCause)
						return callbackCause
					}
					return nil
				},
				func(*blockstorage.VolumeReadonlyOpts) error { later.Add(1); *client = original; return nil },
			}
			wantFirst := int32(0)
			want := resource.ErrInvalidOption
			switch kind {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong role and unsafe ID":
				client.Type = "compute"
				id = "a/b"
				want = resource.ErrUnsupported
			case "unsafe ID":
				id = "a/b"
			case "nil context":
				selected = nil
			case "already canceled":
				cancel(cancelCause)
				want = context.Canceled
			case "nil callback":
				options[0] = nil
			case "source change", "joined callback source and cancel":
				wantFirst = 1
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				t.Error("local readonly failure reached HTTP", req.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.SetVolumeReadonly(selected, client, blockstorage.VolumeActionRequest{VolumeID: id}, options...)
			var proof *resource.ResponseError
			vsaOperation(t, err, "SetVolumeReadonly")
			if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || first.Load() != wantFirst || later.Load() != 0 {
				t.Fatal(result, err, proof, calls.Load(), first.Load(), later.Load())
			}
			if (kind == "already canceled" || kind == "joined callback source and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("direct cancellation cause lost", err)
			}
			if kind == "joined callback source and cancel" && !errors.Is(err, callbackCause) {
				t.Fatal("direct callback cause lost", err)
			}
		})
	}
}
