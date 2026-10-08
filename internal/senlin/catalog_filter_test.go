package senlin_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestBodyFilterCatalogFacadesUseRawFieldsAndOwnNamespaces(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error("catalog Body filter reached wire", r.URL)
		}
		if r.URL.Path == "/senlin/services" {
			testcloud.JSON(w, 200, `{"services":[{"id":"one","name":"wanted","updated_at":"opaque-time","disabled_reason":null},{"id":"two","name":"other","disabled_reason":"disabled"}]}`)
			return
		}
		key := "profile_types"
		if r.URL.Path == "/senlin/policy-types" {
			key = "policy_types"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"name":"wanted","schema":{"n":9007199254740993},"support_status":[false,{"reason":"ok"}]},{"name":"other","NAME":"wanted","schema":{"n":9007199254740994}}]}`, key))
	})
	client := cloud.Client("clustering", "/senlin")
	client.Microversion = "1.7"
	svc := services.New(client)
	rows, err := svc.All(context.Background(), services.WithListFilter("name", "wanted"), services.WithListFilter("updated_at", "opaque-time"), services.WithListFilter("disabled_reason", nil))
	if err != nil || len(rows) != 1 || rows[0].ID != "one" {
		t.Fatal("raw-only service fields were not wired", rows, err)
	}
	profiles := profiletypes.New(client)
	types, err := profiles.All(context.Background(), profiletypes.WithListFilter("name", "wanted"), profiletypes.WithListFilter("schema", json.RawMessage(`{"n":9.007199254740993e15}`)), profiletypes.WithListFilter("support_status", []any{false, map[string]any{"reason": "ok"}}))
	if err != nil || len(types) != 1 || types[0].Name != "wanted" {
		t.Fatal(types, err)
	}
	if types, err := profiles.All(context.Background(), profiletypes.WithListFilter("id", "wanted")); err != nil || len(types) != 0 {
		t.Fatal("raw id invented a name fallback", types, err)
	}
	policies := policytypes.New(client)
	pol, err := policies.All(context.Background(), policytypes.WithListFilter("name", "wanted"))
	if err != nil || len(pol) != 1 || pol[0].Name != "wanted" {
		t.Fatal("casefold typed name overrode raw Body filter", pol, err)
	}
	if pol, err := policies.All(context.Background(), policytypes.WithListFilter("id", "wanted")); err != nil || len(pol) != 0 {
		t.Fatal("raw id invented a name fallback", pol, err)
	}
	before := calls.Load()
	if rows, err := svc.All(context.Background(), profiletypes.WithListFilter("schema", map[string]any{})); !errors.Is(err, resource.ErrInvalidOption) || rows != nil {
		t.Fatal("foreign catalog option accepted", rows, err)
	}
	if rows, err := svc.All(context.Background(), services.WithListFilter("topic", "go-extra")); !errors.Is(err, resource.ErrInvalidOption) || rows != nil {
		t.Fatal("Go extra became a source Body filter", rows, err)
	}
	if types, err := profiles.All(context.Background(), profiletypes.WithListFilter("version", "go-extra")); !errors.Is(err, resource.ErrInvalidOption) || types != nil {
		t.Fatal(types, err)
	}
	if pol, err := policies.All(context.Background(), policytypes.WithListQuery("name", "wire")); !errors.Is(err, resource.ErrInvalidOption) || pol != nil {
		t.Fatal(pol, err)
	}
	client.Microversion = "1.6"
	if rows, err := svc.All(context.Background(), services.WithListFilter("name", "wanted")); !errors.Is(err, resource.ErrUnsupported) || rows != nil || calls.Load() != before {
		t.Fatal("catalog filter bypassed gate or preflight", rows, err, calls.Load(), before)
	}
}
