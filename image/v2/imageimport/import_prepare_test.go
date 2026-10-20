package imageimport

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPrepareImportOptionsFreezesJSONAndSourceConflictsWithoutRequests(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.test/v2/", Type: "image"}
	api := New(client)
	var calls atomic.Int32
	var captured *ImportOpts
	prepared, err := api.PrepareImportOptions(context.Background(), func(value *ImportOpts) error {
		calls.Add(1)
		value.Stores = []string{"fast"}
		value.Fields = map[string]any{"vendor": map[string]any{"large": json.Number("9007199254740993")}}
		captured = value
		return nil
	})
	if err != nil || calls.Load() != 1 || prepared.Method != GlanceDirectMethod {
		t.Fatalf("prepared=%+v calls=%d err=%v", prepared, calls.Load(), err)
	}
	captured.Stores[0] = "after"
	captured.Fields["vendor"].(map[string]any)["large"] = 1
	if prepared.Stores[0] != "fast" || string(prepared.Fields["vendor"].(json.RawMessage)) != `{"large":9007199254740993}` {
		t.Fatalf("snapshot changed: %+v", prepared)
	}
	client.MoreHeaders = map[string]string{"x-image-meta-store": "legacy"}
	_, err = api.PrepareImportOptions(context.Background(), WithImportStores("other"))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("source conflict err=%v", err)
	}
	prepared, err = api.PrepareImportOptions(context.Background(), WithImportStore("explicit"))
	if err != nil || *prepared.Store != "explicit" || client.MoreHeaders["x-image-meta-store"] != "legacy" {
		t.Fatalf("explicit overlay prepared=%+v source=%v err=%v", prepared, client.MoreHeaders, err)
	}
}
