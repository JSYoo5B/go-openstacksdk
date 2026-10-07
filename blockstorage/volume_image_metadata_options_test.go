package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder(t *testing.T) {
	ctx := vsaContext(t)
	raw := json.RawMessage(`{"nested":[false,null,9007199254740993]}`)
	original := string(raw)
	values := map[string]json.RawMessage{"payload": raw}
	whole := blockstorage.WithVolumeImageMetadataOptions(blockstorage.VolumeImageMetadataOpts{Metadata: values})
	rawFactory := blockstorage.WithVolumeImageMetadataRaw(values)
	strings := map[string]string{"literal": "owned"}
	stringFactory := blockstorage.WithVolumeImageMetadata(strings)
	raw[0] = '['
	delete(values, "payload")
	strings["literal"] = "caller changed"
	for _, factory := range []blockstorage.VolumeImageMetadataOption{whole, rawFactory} {
		for i := 0; i < 2; i++ {
			got, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, factory)
			if err != nil || len(got.Metadata) != 1 || string(got.Metadata["payload"]) != original {
				t.Fatal(got, err)
			}
			got.Metadata["payload"][0] = '['
			delete(got.Metadata, "payload")
		}
	}
	for i := 0; i < 2; i++ {
		got, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, stringFactory)
		if err != nil || string(got.Metadata["literal"]) != `"owned"` {
			t.Fatal(got, err)
		}
		got.Metadata["literal"][0] = '['
	}
	got, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx,
		blockstorage.WithVolumeImageMetadataValue("discarded", "first"),
		blockstorage.WithVolumeImageMetadataRaw(map[string]json.RawMessage{"keep": json.RawMessage(`false`)}),
		blockstorage.WithVolumeImageMetadataValue("literal / ?#\n\x00", ""),
		blockstorage.WithVolumeImageMetadataValue("keep", "last"))
	if err != nil || len(got.Metadata) != 2 || string(got.Metadata["keep"]) != `"last"` || string(got.Metadata["literal / ?#\n\x00"]) != `""` {
		t.Fatal("replace and merge order changed", got, err)
	}
	for _, replacement := range []blockstorage.VolumeImageMetadataOption{
		blockstorage.WithVolumeImageMetadataOptions(blockstorage.VolumeImageMetadataOpts{}),
		blockstorage.WithVolumeImageMetadata(nil), blockstorage.WithVolumeImageMetadataRaw(nil),
	} {
		got, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, stringFactory, replacement)
		if err != nil || got.Metadata == nil || len(got.Metadata) != 0 {
			t.Fatal("nil replacement must produce owned empty metadata", got, err)
		}
		got.Metadata["changed result"] = json.RawMessage(`true`)
	}
	for i := 0; i < 2; i++ {
		empty, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx)
		if err != nil || empty.Metadata == nil || len(empty.Metadata) != 0 {
			t.Fatal(empty, err)
		}
		empty.Metadata["changed result"] = json.RawMessage(`true`)
	}
	keys := []string{"b", "", "b"}
	keyFactory := blockstorage.WithVolumeImageMetadataDeleteKeys(keys...)
	wholeKeys := blockstorage.WithVolumeImageMetadataDeleteOptions(blockstorage.VolumeImageMetadataDeleteOpts{Keys: &keys})
	keys[0] = "caller changed"
	keys = nil
	for _, factory := range []blockstorage.VolumeImageMetadataDeleteOption{keyFactory, wholeKeys} {
		for i := 0; i < 2; i++ {
			got, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, factory)
			if err != nil || got.Keys == nil || !reflect.DeepEqual(*got.Keys, []string{"b", "", "b"}) {
				t.Fatal(got, err)
			}
			(*got.Keys)[0] = "returned changed"
			*got.Keys = nil
		}
	}
	all, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, keyFactory, blockstorage.WithVolumeImageMetadataDeleteAll())
	if err != nil || all.Keys != nil {
		t.Fatal("all intent did not replace explicit keys", all, err)
	}
	var nilKeys []string
	for _, factory := range []blockstorage.VolumeImageMetadataDeleteOption{
		blockstorage.WithVolumeImageMetadataDeleteKeys(),
		blockstorage.WithVolumeImageMetadataDeleteKeys(nilKeys...),
		blockstorage.WithVolumeImageMetadataDeleteOptions(blockstorage.VolumeImageMetadataDeleteOpts{Keys: &nilKeys}),
	} {
		empty, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, blockstorage.WithVolumeImageMetadataDeleteAll(), factory)
		if err != nil || empty.Keys == nil || len(*empty.Keys) != 0 {
			t.Fatal("explicit none was changed into all", empty, err)
		}
	}
}

func TestVolumeImageMetadataOriginalsOwnCallbackSliceIntermediateMapsAndKeySlices(t *testing.T) {
	for _, family := range []string{"set", "delete"} {
		t.Run(family, func(t *testing.T) {
			var order []int
			replaced := 0
			if family == "set" {
				raw := json.RawMessage(`{"nested":[false,null,9007199254740993]}`)
				var first, second *blockstorage.VolumeImageMetadataOpts
				var options []blockstorage.VolumeImageMetadataOption
				options = []blockstorage.VolumeImageMetadataOption{
					func(next *blockstorage.VolumeImageMetadataOpts) error {
						order = append(order, 1)
						next.Metadata = map[string]json.RawMessage{"payload": raw}
						first = next
						options[1] = func(*blockstorage.VolumeImageMetadataOpts) error {
							replaced++
							return errors.New("replacement must not run")
						}
						return nil
					},
					func(next *blockstorage.VolumeImageMetadataOpts) error {
						order = append(order, 2)
						raw[0] = '['
						delete(first.Metadata, "payload")
						if string(next.Metadata["payload"]) != `{"nested":[false,null,9007199254740993]}` {
							t.Fatal("borrowed previous map or member bytes", next)
						}
						second = next
						return nil
					},
					func(next *blockstorage.VolumeImageMetadataOpts) error {
						order = append(order, 3)
						second.Metadata["payload"][0] = '['
						delete(second.Metadata, "payload")
						if string(next.Metadata["payload"]) != `{"nested":[false,null,9007199254740993]}` {
							t.Fatal("borrowed intermediate map or member bytes", next)
						}
						return nil
					},
				}
				got, err := blockstorage.PrepareVolumeImageMetadataOptions(vsaContext(t), options...)
				if err != nil || string(got.Metadata["payload"]) != `{"nested":[false,null,9007199254740993]}` {
					t.Fatal(got, err)
				}
				first.Metadata["late"] = json.RawMessage(`true`)
				if len(got.Metadata) != 1 {
					t.Fatal("returned policy borrowed callback map", got)
				}
			} else {
				keys := []string{"b", "", "b"}
				var first, second *blockstorage.VolumeImageMetadataDeleteOpts
				var options []blockstorage.VolumeImageMetadataDeleteOption
				options = []blockstorage.VolumeImageMetadataDeleteOption{
					func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
						order = append(order, 1)
						next.Keys = &keys
						first = next
						options[1] = func(*blockstorage.VolumeImageMetadataDeleteOpts) error {
							replaced++
							return errors.New("replacement must not run")
						}
						return nil
					},
					func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
						order = append(order, 2)
						keys[0] = "caller changed"
						*first.Keys = nil
						if next.Keys == nil || !reflect.DeepEqual(*next.Keys, []string{"b", "", "b"}) {
							t.Fatal("borrowed previous keys pointer or backing array", next)
						}
						second = next
						return nil
					},
					func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
						order = append(order, 3)
						(*second.Keys)[0] = "retained changed"
						*second.Keys = nil
						if next.Keys == nil || !reflect.DeepEqual(*next.Keys, []string{"b", "", "b"}) {
							t.Fatal("borrowed intermediate keys pointer or backing array", next)
						}
						return nil
					},
				}
				got, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(vsaContext(t), options...)
				if err != nil || got.Keys == nil || !reflect.DeepEqual(*got.Keys, []string{"b", "", "b"}) {
					t.Fatal(got, err)
				}
				*first.Keys = []string{"late"}
				if !reflect.DeepEqual(*got.Keys, []string{"b", "", "b"}) {
					t.Fatal("returned policy borrowed callback pointer", got)
				}
			}
			if replaced != 0 || !reflect.DeepEqual(order, []int{1, 2, 3}) {
				t.Fatal(order, replaced)
			}
		})
	}
}

func TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys(t *testing.T) {
	bad := string([]byte{0xff})
	cases := []struct {
		name string
		raw  map[string]json.RawMessage
	}{
		{"invalid key", map[string]json.RawMessage{bad: json.RawMessage(`true`)}},
		{"malformed value", map[string]json.RawMessage{"key": json.RawMessage(`{]`)}},
		{"invalid UTF8 value", map[string]json.RawMessage{"key": json.RawMessage([]byte{'"', 0xff, '"'})}},
		{"nil raw member", map[string]json.RawMessage{"key": nil}},
	}
	for _, tc := range cases {
		t.Run("set/"+tc.name, func(t *testing.T) {
			got, err := blockstorage.PrepareVolumeImageMetadataOptions(vsaContext(t), blockstorage.WithVolumeImageMetadataRaw(tc.raw))
			if !errors.Is(err, resource.ErrInvalidOption) || got.Metadata != nil {
				t.Fatal(got, err)
			}
			got, err = blockstorage.PrepareVolumeImageMetadataOptions(vsaContext(t), blockstorage.WithVolumeImageMetadataRaw(tc.raw), blockstorage.WithVolumeImageMetadataRaw(map[string]json.RawMessage{"": json.RawMessage(`null`), "literal / ?#\n\x00": json.RawMessage(`9007199254740993`)}))
			if err != nil || len(got.Metadata) != 2 || string(got.Metadata[""]) != `null` || string(got.Metadata["literal / ?#\n\x00"]) != `9007199254740993` {
				t.Fatal("overwritten invalid metadata was validated too early", got, err)
			}
		})
	}
	for _, invalid := range []blockstorage.VolumeImageMetadataOption{
		blockstorage.WithVolumeImageMetadata(map[string]string{"key": bad}),
		blockstorage.WithVolumeImageMetadataValue("key", bad),
	} {
		got, err := blockstorage.PrepareVolumeImageMetadataOptions(vsaContext(t), invalid)
		if !errors.Is(err, resource.ErrInvalidOption) || got.Metadata != nil {
			t.Fatal("string factory silently replaced invalid UTF8", got, err)
		}
		got, err = blockstorage.PrepareVolumeImageMetadataOptions(vsaContext(t), invalid, blockstorage.WithVolumeImageMetadataValue("key", "repaired"))
		if err != nil || string(got.Metadata["key"]) != `"repaired"` {
			t.Fatal("later value did not repair invalid string", got, err)
		}
	}
	for _, replacement := range []blockstorage.VolumeImageMetadataDeleteOption{
		blockstorage.WithVolumeImageMetadataDeleteKeys("", "literal / ?#\n\x00", ""),
		blockstorage.WithVolumeImageMetadataDeleteKeys(),
		blockstorage.WithVolumeImageMetadataDeleteAll(),
	} {
		got, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(vsaContext(t), blockstorage.WithVolumeImageMetadataDeleteKeys(bad), replacement)
		if err != nil {
			t.Fatal("overwritten invalid key was validated too early", got, err)
		}
	}
	failed, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(vsaContext(t), blockstorage.WithVolumeImageMetadataDeleteKeys("valid", bad))
	if !errors.Is(err, resource.ErrInvalidOption) || failed.Keys != nil {
		t.Fatal(failed, err)
	}
	all, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(vsaContext(t))
	if err != nil || all.Keys != nil {
		t.Fatal("nil keys default changed", all, err)
	}
}

func TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause(t *testing.T) {
	for _, family := range []string{"set", "delete"} {
		for _, kind := range []string{"nil context", "already canceled", "nil callback", "callback error", "callback cancel", "callback error and cancel"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				callbackCause, cancelCause := errors.New("metadata prepare callback"), errors.New("metadata prepare custom cancel")
				first, later, wantFirst := 0, 0, 1
				if kind == "nil context" {
					selected = nil
					wantFirst = 0
				}
				if kind == "already canceled" {
					cancel(cancelCause)
					wantFirst = 0
				}
				if kind == "nil callback" {
					wantFirst = 0
				}
				callback := func() error {
					first++
					if kind == "callback cancel" || kind == "callback error and cancel" {
						cancel(cancelCause)
					}
					if kind == "callback error" || kind == "callback error and cancel" {
						return callbackCause
					}
					return nil
				}
				var err error
				if family == "set" {
					options := []blockstorage.VolumeImageMetadataOption{
						func(next *blockstorage.VolumeImageMetadataOpts) error {
							next.Metadata = map[string]json.RawMessage{"key": json.RawMessage(`null`)}
							return callback()
						},
						func(*blockstorage.VolumeImageMetadataOpts) error { later++; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					got, cause := blockstorage.PrepareVolumeImageMetadataOptions(selected, options...)
					err = cause
					if got.Metadata != nil {
						t.Fatal("failed set Prepare published metadata", got, err)
					}
				} else {
					options := []blockstorage.VolumeImageMetadataDeleteOption{
						func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
							keys := []string{"key"}
							next.Keys = &keys
							return callback()
						},
						func(*blockstorage.VolumeImageMetadataDeleteOpts) error { later++; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					got, cause := blockstorage.PrepareVolumeImageMetadataDeleteOptions(selected, options...)
					err = cause
					if got.Keys != nil {
						t.Fatal("failed delete Prepare published keys", got, err)
					}
				}
				if err == nil || first != wantFirst || later != 0 {
					t.Fatal(err, first, later)
				}
				if (kind == "nil context" || kind == "nil callback") && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if (kind == "already canceled" || kind == "callback cancel" || kind == "callback error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("lost context identities", err)
				}
				if (kind == "callback error" || kind == "callback error and cancel") && !errors.Is(err, callbackCause) {
					t.Fatal("lost callback identity", err)
				}
			})
		}
	}
}
