package blockstorage_test

import (
	"context"
	"encoding/json"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestSearchVolumesMappingDuplicateWireKeysKeepTextualAliasOrderAndRawEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("volumev3", "/duplicate-alias/v3/project/")
	wire := `{"volumes":[{"id":"id","imageRef":"first","image_id":"middle","imageRef":"last"}]}`
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /duplicate-alias/v3/project/volumes/detail", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, wire) })
	result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{})
	if err != nil || result == nil || calls.Load() != 1 || len(result.Volumes) != 1 || len(result.Pages) != 1 || string(result.Pages[0].Body) != wire {
		t.Fatal(result, err, calls.Load())
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(result.Value, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || string(rows[0]["image_id"]) != `"last"` {
		t.Fatal("textual last consumed alias lost", string(result.Value))
	}
	if string(result.Volumes[0].Body["imageRef"]) != `"last"` || string(result.Volumes[0].Body["image_id"]) != `"middle"` {
		t.Fatal("original raw map aliases rewritten", result.Volumes[0].Body)
	}
	// The normalized image_id follows textual members, not a reserialized map
	// whose imageRef key sorts before image_id. These are distinct owned views.
	result.Volumes[0].Body["imageRef"] = json.RawMessage(`"caller"`)
	if string(rows[0]["image_id"]) != `"last"` || string(result.Pages[0].Body) != wire {
		t.Fatal("raw/normalized/page evidence aliases")
	}
}
