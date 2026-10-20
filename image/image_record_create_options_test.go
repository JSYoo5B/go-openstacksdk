package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageRecordCreateOptionsKeepOmittedValuesForSourceOrderedDefaults(t *testing.T) {
	got, err := prepareImageRecordCreateOptions(context.Background(), nil, nil)
	th.AssertNoErr(t, err)
	if got.Container != nil || got.MD5 != nil || got.SHA256 != nil || got.DiskFormat != nil || got.ContainerFormat != nil || got.Tags != nil || got.DisableVendorAgent != nil || got.AllowDuplicates != nil || got.Wait != nil || got.Timeout != nil || got.ValidateChecksum != nil || got.UseImport != nil || got.Size != nil || got.Meta != nil || got.Attributes != nil || got.Import.Method != nil || got.Import.Stores != nil || got.optionContext != nil || got.optionCheck != nil {
		t.Fatal("preface/default decisions moved into capture", got)
	}
}

func TestImageRecordCreateRawHelpersPreserveSourceValuesWithoutEagerEnumOrRangeChecks(t *testing.T) {
	for _, test := range []struct {
		name     string
		option   ImageRecordCreateOption
		expected string
		raw      func(ImageRecordCreateOpts) json.RawMessage
	}{
		{"MD5", WithImageRecordCreateMD5(nil), "null", func(c ImageRecordCreateOpts) json.RawMessage { return c.MD5 }},
		{"SHA256", WithImageRecordCreateSHA256(""), `""`, func(c ImageRecordCreateOpts) json.RawMessage { return c.SHA256 }},
		{"DiskFormat", WithImageRecordCreateDiskFormat(false), "false", func(c ImageRecordCreateOpts) json.RawMessage { return c.DiskFormat }},
		{"ContainerFormat", WithImageRecordCreateContainerFormat(0), "0", func(c ImageRecordCreateOpts) json.RawMessage { return c.ContainerFormat }},
		{"Tags", WithImageRecordCreateTags([]any{}), "[]", func(c ImageRecordCreateOpts) json.RawMessage { return c.Tags }},
		{"DisableVendorAgent", WithImageRecordCreateDisableVendorAgent(map[string]any{"why": []any{}}), `{"why":[]}`, func(c ImageRecordCreateOpts) json.RawMessage { return c.DisableVendorAgent }},
		{"AllowDuplicates", WithImageRecordCreateAllowDuplicates(false), "false", func(c ImageRecordCreateOpts) json.RawMessage { return c.AllowDuplicates }},
		{"Wait", WithImageRecordCreateWait([]any{}), "[]", func(c ImageRecordCreateOpts) json.RawMessage { return c.Wait }},
		{"Timeout", WithImageRecordCreateTimeout(json.RawMessage(`1e9999`)), "1e9999", func(c ImageRecordCreateOpts) json.RawMessage { return c.Timeout }},
		{"ValidateChecksum", WithImageRecordCreateValidateChecksum("truthy"), `"truthy"`, func(c ImageRecordCreateOpts) json.RawMessage { return c.ValidateChecksum }},
		{"UseImport", WithImageRecordCreateUseImport(-1), "-1", func(c ImageRecordCreateOpts) json.RawMessage { return c.UseImport }},
		{"Size", WithImageRecordCreateSize(true), "true", func(c ImageRecordCreateOpts) json.RawMessage { return c.Size }},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{test.option})
			th.AssertNoErr(t, err)
			if string(test.raw(got)) != test.expected || got.Import.Method != nil {
				t.Fatal(got, test.expected)
			}
		})
	}
}

func TestImageRecordCreateWholeOptionsCaptureNestedRawMapsStoresAndReuseIndependently(t *testing.T) {
	container := ""
	size := json.RawMessage(`-900719925474099312345`)
	attrs := map[string]any{"vendor": map[string]any{"precise": json.RawMessage(`900719925474099312345`)}}
	meta := map[string]any{"size": "do not convert"}
	headers := map[string]string{"X-Note": "captured"}
	swift := map[string]string{"X-Note": "swift"}
	stores := []ImageRecordImportStore{{RawID: json.RawMessage(`null`)}, {Attributes: map[string]any{}}}
	option := WithImageRecordCreateOpts(ImageRecordCreateOpts{Container: &container, Size: size, Attributes: attrs, Meta: meta, Headers: headers, SwiftHeaders: swift, Import: ImageRecordImportOpts{Stores: stores}})
	container = "changed"
	size[0] = '!'
	attrs["vendor"].(map[string]any)["precise"] = false
	meta["size"] = "changed"
	headers["X-Note"] = "changed"
	swift["X-Note"] = "changed"
	stores[0].RawID[0] = '!'
	stores[1].Attributes["id"] = "changed"
	got, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{option})
	th.AssertNoErr(t, err)
	if got.Container == nil || *got.Container != "" || string(got.Size) != `-900719925474099312345` || string(got.Attributes["vendor"].(json.RawMessage)) != `{"precise":900719925474099312345}` || string(got.Meta["size"].(json.RawMessage)) != `"do not convert"` || got.Headers["X-Note"] != "captured" || got.SwiftHeaders["X-Note"] != "swift" || got.Import.Method != nil || len(got.Import.Stores) != 2 || string(got.Import.Stores[0].identity) != "null" || string(got.Import.Stores[1].identity) != "null" {
		t.Fatal(got)
	}
	*got.Container = "changed"
	got.Size[0] = '!'
	got.Attributes["vendor"].(json.RawMessage)[0] = '!'
	got.Meta["size"].(json.RawMessage)[0] = '!'
	got.Import.Stores[0].identity[0] = '!'
	got.Headers["X-Note"] = "changed"
	again, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{option})
	th.AssertNoErr(t, err)
	if *again.Container != "" || string(again.Size) != `-900719925474099312345` || string(again.Import.Stores[0].identity) != "null" || again.Headers["X-Note"] != "captured" {
		t.Fatal("helper reuse shared mutable bytes", again)
	}
}

func TestImageRecordCreateOptionsMergeReplaceAndEmptyStorePresenceRemainDistinct(t *testing.T) {
	for _, mode := range []string{"merge", "whole replace", "meta replace", "present empty stores", "omitted stores"} {
		t.Run(mode, func(t *testing.T) {
			options := []ImageRecordCreateOption{WithImageRecordCreateAttribute("a", 1), WithImageRecordCreateHeader("x-note", "first"), WithImageRecordCreateSwiftHeader("x-note", "first"), WithImageRecordCreateMeta(map[string]any{"a": 1})}
			switch mode {
			case "merge":
				options = append(options, WithImageRecordCreateAttributes(map[string]any{"b": nil}), WithImageRecordCreateHeader("X-NOTE", "last"), WithImageRecordCreateSwiftHeaders(map[string]string{"X-NOTE": "last"}))
			case "whole replace":
				options = append(options, WithImageRecordCreateOpts(ImageRecordCreateOpts{Attributes: map[string]any{"b": nil}}))
			case "meta replace":
				options = append(options, WithImageRecordCreateMeta(map[string]any{"b": nil}))
			case "present empty stores":
				options = append(options, WithImageRecordCreateOpts(ImageRecordCreateOpts{Import: ImageRecordImportOpts{Stores: []ImageRecordImportStore{}}}))
			case "omitted stores":
				options = append(options, WithImageRecordCreateOpts(ImageRecordCreateOpts{}))
			}
			got, err := prepareImageRecordCreateOptions(context.Background(), nil, options)
			th.AssertNoErr(t, err)
			switch mode {
			case "merge":
				if len(got.Attributes) != 2 || string(got.Attributes["b"].(json.RawMessage)) != "null" || got.Headers["X-Note"] != "last" || got.SwiftHeaders["X-Note"] != "last" {
					t.Fatal(got)
				}
			case "whole replace":
				if len(got.Attributes) != 1 || got.Attributes["a"] != nil || len(got.Headers) != 0 || len(got.SwiftHeaders) != 0 || got.Meta != nil {
					t.Fatal(got)
				}
			case "meta replace":
				if len(got.Meta) != 1 || got.Meta["a"] != nil || string(got.Meta["b"].(json.RawMessage)) != "null" {
					t.Fatal(got)
				}
			case "present empty stores":
				if got.Import.Stores == nil || len(got.Import.Stores) != 0 || got.Import.Method != nil {
					t.Fatal("present empty selection defaulted or lost", got.Import)
				}
			case "omitted stores":
				if got.Import.Stores != nil || got.Import.Method != nil {
					t.Fatal(got.Import)
				}
			}
		})
	}
}

func TestImageRecordCreateJSONFactoriesMarshalOnceAndKeepEncodingCauses(t *testing.T) {
	calls := 0
	value := imageRecordMarshalCallback(func() ([]byte, error) { calls++; return []byte(`{"precise":900719925474099312345}`), nil })
	option := WithImageRecordCreateTags(value)
	if calls != 1 {
		t.Fatal(calls)
	}
	for index := 0; index < 2; index++ {
		got, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{option})
		th.AssertNoErr(t, err)
		if calls != 1 || string(got.Tags) != `{"precise":900719925474099312345}` {
			t.Fatal(got, calls)
		}
	}
	for _, mode := range []string{"marshal cause", "unsupported type", "lone surrogate", "invalid Go UTF8"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("create JSON marshal")
			var value any
			switch mode {
			case "marshal cause":
				value = imageRecordMarshalCallback(func() ([]byte, error) { return nil, marker })
			case "unsupported type":
				value = make(chan int)
			case "lone surrogate":
				value = json.RawMessage(`"\ud800"`)
			case "invalid Go UTF8":
				value = string([]byte{255})
			}
			_, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{WithImageRecordCreateSize(value)})
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "marshal cause" && !errors.Is(err, marker) {
				t.Fatal(err)
			}
			if mode == "unsupported type" {
				var unsupported *json.UnsupportedTypeError
				if !errors.As(err, &unsupported) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestImageRecordCreateNestedImportCallbacksRunOnceAndStopAfterEachGuard(t *testing.T) {
	for _, mode := range []string{"captured callbacks", "source", "cancel", "callback cause", "whole replacement guard"} {
		t.Run(mode, func(t *testing.T) {
			calls, first, later := 0, 0, 0
			marker := errors.New("create nested import")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("foundation dispatched")
				return nil, nil
			})
			p, err := New(client).captureImageRecord(ctx)
			th.AssertNoErr(t, err)
			var retained *ImageRecordImportOpts
			options := []ImageRecordImportOption{func(c *ImageRecordImportOpts) error {
				first++
				retained = c
				c.URI = json.RawMessage(`"without method"`)
				switch mode {
				case "source", "whole replacement guard":
					client.Endpoint = "https://foreign.test/"
				case "cancel":
					cancel(marker)
				case "callback cause":
					return marker
				}
				return nil
			}, func(*ImageRecordImportOpts) error { later++; return nil }}
			option := WithImageRecordCreateImportOptions(options...)
			options[0] = func(*ImageRecordImportOpts) error { t.Fatal("uncaptured nested option slice"); return nil }
			var outer ImageRecordCreateOption = option
			if mode == "whole replacement guard" {
				outer = func(c *ImageRecordCreateOpts) error {
					if err := WithImageRecordCreateOpts(ImageRecordCreateOpts{})(c); err != nil {
						return err
					}
					return option(c)
				}
			}
			got, err := prepareImageRecordCreateOptions(p.ctx, p.check, []ImageRecordCreateOption{outer})
			if calls != 0 || first != 1 {
				t.Fatal(got, err, calls, first)
			}
			if mode == "captured callbacks" {
				th.AssertNoErr(t, err)
				if later != 1 || got.Import.Method != nil || string(got.Import.URI) != `"without method"` {
					t.Fatal(got, later)
				}
				retained.URI[0] = '!'
				if string(got.Import.URI) != `"without method"` {
					t.Fatal("retained candidate aliases prepared import")
				}
				return
			}
			if later != 0 {
				t.Fatal(later)
			}
			expected := marker
			if mode == "source" || mode == "whole replacement guard" {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordCreateOptionAndMarshalerGuardsStopLaterCallbacks(t *testing.T) {
	for _, mode := range []string{"source", "binding", "cancel", "outer", "callback cause", "marshaler source", "marshaler cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls, later, marshaled := 0, 0, 0
			marker := errors.New("create option guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("foundation dispatched")
				return nil, nil
			})
			service := New(client)
			p, err := service.captureImageRecord(ctx)
			th.AssertNoErr(t, err)
			change := func() {
				switch mode {
				case "source", "marshaler source":
					client.Endpoint = "https://foreign.test/"
				case "binding":
					service.API = nil
				case "cancel", "marshaler cancel":
					cancel(marker)
				case "outer":
					outerBad = true
				}
			}
			first := func(c *ImageRecordCreateOpts) error {
				if strings.HasPrefix(mode, "marshaler") {
					c.Attributes = map[string]any{"a": imageRecordMarshalCallback(func() ([]byte, error) { marshaled++; change(); return []byte(`1`), nil }), "z": imageRecordMarshalCallback(func() ([]byte, error) { t.Fatal("later value marshaled after failed guard"); return nil, nil })}
					return nil
				}
				change()
				if mode == "callback cause" {
					return marker
				}
				return nil
			}
			_, err = prepareImageRecordCreateOptions(p.ctx, p.check, []ImageRecordCreateOption{first, func(*ImageRecordCreateOpts) error { later++; return nil }})
			if err == nil || later != 0 || calls != 0 {
				t.Fatal(err, later, calls)
			}
			expected := marker
			if mode == "source" || mode == "binding" || mode == "marshaler source" {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
			if strings.HasPrefix(mode, "marshaler") && marshaled != 1 {
				t.Fatal(marshaled)
			}
		})
	}
}

func TestImageRecordCreateHeaderNamespacesRejectOwnedControlsAndInvalidOptions(t *testing.T) {
	for _, test := range []struct {
		name   string
		option ImageRecordCreateOption
	}{
		{"nil create option", nil}, {"nil nested option", WithImageRecordCreateImportOptions(nil)},
		{"auth", WithImageRecordCreateHeader("X-Auth-Token", "spoof")}, {"media", WithImageRecordCreateHeader("Content-Type", "spoof")},
		{"framing", WithImageRecordCreateSwiftHeader("Content-Length", "1")}, {"Swift TTL", WithImageRecordCreateSwiftHeader("x-delete-after", "1")},
		{"Swift autocreate", WithImageRecordCreateSwiftHeader("x-object-meta-x-sdk-autocreated", "false")},
		{"newline", WithImageRecordCreateHeaders(map[string]string{"X-Note": "bad\nvalue"})},
		{"container UTF8", WithImageRecordCreateContainer(string([]byte{255}))},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := prepareImageRecordCreateOptions(context.Background(), nil, []ImageRecordCreateOption{test.option})
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}
