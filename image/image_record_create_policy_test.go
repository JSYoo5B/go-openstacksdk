package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageCreatePolicyDefaultsRetainExplicitRawAndServicePresence(t *testing.T) {
	got, err := PrepareImageCreatePolicy()
	th.AssertNoErr(t, err)
	if string(got.ImageFormat) != `"qcow2"` || string(got.UseTasks) != "false" || got.DisableVendorAgent == nil || len(got.DisableVendorAgent) != 0 || got.ObjectStoreEnabled != nil {
		t.Fatal(got)
	}
	for _, test := range []struct {
		name                          string
		format, tasks                 any
		enabled                       bool
		expectedFormat, expectedTasks string
	}{{"null", nil, nil, false, "null", "null"}, {"falsey", 0, []any{}, false, "0", "[]"}, {"raw truthy", map[string]any{"future": true}, "yes", true, `{"future":true}`, `"yes"`}} {
		t.Run(test.name, func(t *testing.T) {
			got, err := PrepareImageCreatePolicy(WithImageCreatePolicyFormat(test.format), WithImageCreatePolicyTasks(test.tasks), WithImageCreatePolicyObjectStoreEnabled(test.enabled))
			th.AssertNoErr(t, err)
			if string(got.ImageFormat) != test.expectedFormat || string(got.UseTasks) != test.expectedTasks || got.ObjectStoreEnabled == nil || *got.ObjectStoreEnabled != test.enabled {
				t.Fatal(got)
			}
		})
	}
}

func TestImageCreatePolicyFactoriesCaptureReplaceAndReuseWithoutAliases(t *testing.T) {
	var options []ImageCreatePolicyOption
	options = []ImageCreatePolicyOption{func(*ImageCreatePolicy) error {
		options[1] = func(*ImageCreatePolicy) error { t.Fatal("uncaptured policy option slice"); return nil }
		return nil
	}, WithImageCreatePolicyFormat("vhd")}
	fromSlice, err := PrepareImageCreatePolicy(options...)
	th.AssertNoErr(t, err)
	th.AssertEquals(t, `"vhd"`, string(fromSlice.ImageFormat))
	format := json.RawMessage(`"vhd"`)
	enabled := false
	vendor := map[string]any{"nested": []any{json.RawMessage(`900719925474099312345`)}}
	option := WithImageCreatePolicyOpts(ImageCreatePolicy{ImageFormat: format, DisableVendorAgent: vendor, ObjectStoreEnabled: &enabled, CloudName: "captured"})
	format[0] = '!'
	enabled = true
	vendor["nested"].([]any)[0] = false
	got, err := PrepareImageCreatePolicy(WithImageCreatePolicyTasks(true), WithImageCreatePolicyVendorAgent(map[string]any{"discarded": true}), option)
	th.AssertNoErr(t, err)
	if string(got.ImageFormat) != `"vhd"` || string(got.UseTasks) != "false" || *got.ObjectStoreEnabled || got.CloudName != "captured" || len(got.DisableVendorAgent) != 1 || string(got.DisableVendorAgent["nested"].(json.RawMessage)) != `[900719925474099312345]` {
		t.Fatal(got)
	}
	got.ImageFormat[0] = '!'
	*got.ObjectStoreEnabled = true
	got.DisableVendorAgent["nested"].(json.RawMessage)[0] = '!'
	again, err := PrepareImageCreatePolicy(option, WithImageCreatePolicyVendorAgent(map[string]any{"replacement": nil}))
	th.AssertNoErr(t, err)
	if string(again.ImageFormat) != `"vhd"` || *again.ObjectStoreEnabled || len(again.DisableVendorAgent) != 1 || string(again.DisableVendorAgent["replacement"].(json.RawMessage)) != "null" {
		t.Fatal(again)
	}
}

func TestImageCreatePolicyGuardedCopyStopsMarshalersAndKeepsCausalChain(t *testing.T) {
	for _, mode := range []string{"source", "cancel", "marshal cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, marshaled, later := 0, 0, 0
			marker := errors.New("create policy capture")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("policy dispatched"); return nil, nil })
			p, err := New(client).captureImageRecord(ctx)
			th.AssertNoErr(t, err)
			policy := ImageCreatePolicy{DisableVendorAgent: map[string]any{"a": imageRecordMarshalCallback(func() ([]byte, error) {
				marshaled++
				switch mode {
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "cancel":
					cancel(marker)
				case "marshal cause":
					return nil, marker
				}
				return []byte(`true`), nil
			}), "z": imageRecordMarshalCallback(func() ([]byte, error) { later++; return []byte(`true`), nil })}}
			_, err = copyImageCreatePolicy(p.ctx, p.check, policy)
			if calls != 0 || marshaled != 1 || later != 0 {
				t.Fatal(err, calls, marshaled, later)
			}
			expected := marker
			if mode == "source" {
				expected = resource.ErrInvalidOption
			}
			if !errors.Is(err, expected) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageCreatePolicyRejectsInvalidEncodingAndNilOptionWithoutFormatEnums(t *testing.T) {
	for _, test := range []struct {
		name   string
		option ImageCreatePolicyOption
	}{{"nil option", nil}, {"Go UTF8", WithImageCreatePolicyFormat(string([]byte{255}))}, {"surrogate", WithImageCreatePolicyTasks(json.RawMessage(`"\ud800"`))}, {"vendor encoding", WithImageCreatePolicyVendorAgent(map[string]any{"a": make(chan int)})}, {"cloud UTF8", WithImageCreatePolicyOpts(ImageCreatePolicy{CloudName: string([]byte{255})})}} {
		t.Run(test.name, func(t *testing.T) {
			_, err := PrepareImageCreatePolicy(test.option)
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageCreatePolicyRawVendorOverrideDefersShapeChecksAndTypedHelperClearsIt(t *testing.T) {
	for _, raw := range []string{`null`, `[["vendor",true]]`, `42`} {
		t.Run(raw, func(t *testing.T) {
			value := json.RawMessage(raw)
			option := WithImageCreatePolicyOpts(ImageCreatePolicy{RawVendorAgent: value})
			value[0] = '!'
			got, err := PrepareImageCreatePolicy(option)
			th.AssertNoErr(t, err)
			if string(got.RawVendorAgent) != raw || got.DisableVendorAgent == nil {
				t.Fatal(got)
			}
			got.RawVendorAgent[0] = '!'
			again, err := PrepareImageCreatePolicy(option)
			th.AssertNoErr(t, err)
			if string(again.RawVendorAgent) != raw {
				t.Fatal("raw vendor snapshot aliases", again)
			}
			typed, err := PrepareImageCreatePolicy(option, WithImageCreatePolicyVendorAgent(map[string]any{"vendor": nil}))
			th.AssertNoErr(t, err)
			if typed.RawVendorAgent != nil || string(typed.DisableVendorAgent["vendor"].(json.RawMessage)) != "null" {
				t.Fatal(typed)
			}
		})
	}
}

func TestImageCreatePolicyRawObjectStoreOverrideDefersAvailabilityChecksAndTypedHelperClearsIt(t *testing.T) {
	for _, raw := range []string{`null`, `"configured"`, `false`} {
		t.Run(raw, func(t *testing.T) {
			value := json.RawMessage(raw)
			option := WithImageCreatePolicyOpts(ImageCreatePolicy{RawObjectStoreEnabled: value})
			value[0] = '!'
			got, err := PrepareImageCreatePolicy(option)
			th.AssertNoErr(t, err)
			if string(got.RawObjectStoreEnabled) != raw || got.ObjectStoreEnabled != nil {
				t.Fatal(got)
			}
			got.RawObjectStoreEnabled[0] = '!'
			again, err := PrepareImageCreatePolicy(option)
			th.AssertNoErr(t, err)
			if string(again.RawObjectStoreEnabled) != raw {
				t.Fatal("raw availability snapshot aliases", again)
			}
			typed, err := PrepareImageCreatePolicy(option, WithImageCreatePolicyObjectStoreEnabled(false))
			th.AssertNoErr(t, err)
			if typed.RawObjectStoreEnabled != nil || typed.ObjectStoreEnabled == nil || *typed.ObjectStoreEnabled {
				t.Fatal(typed)
			}
		})
	}
}
