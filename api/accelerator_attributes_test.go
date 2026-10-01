package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/attributes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const attributeUUID1 = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const attributeUUID2 = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
const attributeUUID3 = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

func TestAcceleratorAttributeReadAndKeyIsNotName(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/attributes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("deployable_id") != "17" || r.URL.Query().Get("key") != "same" {
			t.Errorf("query: %s", r.URL)
		}
		if r.URL.Query().Get("marker") == "second" {
			testcloud.JSON(w, 200, fmt.Sprintf(`{"attributes":[{"uuid":%q,"deployable_id":17,"key":"same","value":"second"}]}`, attributeUUID2))
			return
		}
		w.Header().Set("X-Request-Id", "req-attribute-list")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"attributes":[{"uuid":%q,"id":1234567890123456789,"deployable_id":17,"key":"same","value":"","created_at":"2026-09-01 01:02:03+00:00","updated_at":null,"future":{"n":1234567890123456789}}],"next":"?deployable_id=17&key=same&marker=second"}`, attributeUUID1))
	})
	cloud.Mux.HandleFunc("/v2/attributes/"+attributeUUID1, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Request-Id", "req-attribute-get")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":%q,"deployable_id":17,"key":"same","value":"first","future":false}`, attributeUUID2))
	})
	a := attributes.New(cloud.Client("accelerator", "/v2"))
	for _, action := range []func() error{
		func() error { _, err := a.Find(context.Background(), resource.Name("same")); return err },
		func() error { _, err := a.ResolveID(context.Background(), resource.Name("same")); return err },
		func() error { return a.Delete(context.Background(), resource.Name("same")) },
		func() error { return a.WaitDeleted(context.Background(), resource.Name("same")) },
		func() error { _, err := a.Wait(context.Background(), resource.ID(attributeUUID1), "ready"); return err },
	} {
		if err := action(); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsupported name/status made HTTP request")
	}
	values, err := a.All(context.Background(), resource.WithQuery("deployable_id", "17"), resource.WithQuery("key", "same"))
	if err != nil || len(values) != 2 || values[0].Key != values[1].Key || values[0].UUID == values[1].UUID {
		t.Fatalf("duplicate keys: %+v/%v", values, err)
	}
	value := values[0]
	if value.ID.String() != "1234567890123456789" || value.Header.Get("X-Request-Id") != "req-attribute-list" || value.StatusCode != 200 || value.CreatedAt == nil || string(value.Body["updated_at"]) != "null" || string(value.Body["future"]) != `{"n":1234567890123456789}` {
		t.Fatalf("metadata: %+v", value)
	}
	got, err := a.Find(context.Background(), resource.ID(attributeUUID1))
	if err != nil || got.UUID != attributeUUID2 || got.Header.Get("X-Request-Id") != "req-attribute-get" || string(got.Body["future"]) != "false" {
		t.Fatalf("get: %+v/%v", got, err)
	}
	if id, err := a.ResolveID(context.Background(), resource.ID(attributeUUID1)); err != nil || id != attributeUUID1 {
		t.Fatalf("resolve ID: %q/%v", id, err)
	}
}

func TestAcceleratorAttributeCreateFlatBodyAndSnapshots(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/attributes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("X-Attribute-Extension") != "enabled" {
			t.Errorf("request: %s/%v", r.Method, r.Header)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if string(body["deployable_id"]) != "0" || string(body["key"]) != `""` || string(body["value"]) != `""` || string(body["future"]) != `{"enabled":false,"tags":["original"]}` {
			t.Errorf("flat body/snapshot: %s", body)
		}
		if _, ok := body["attribute"]; ok {
			t.Error("attribute request must not have an envelope")
		}
		w.Header().Set("X-Request-Id", "req-create-attribute")
		testcloud.JSON(w, 201, fmt.Sprintf(`{"uuid":%q,"deployable_id":0,"key":"","value":"","future":null}`, attributeUUID1))
	})
	opts := attributes.CreateOpts{}
	typed := attributes.WithCreateOptions(opts)
	opts.DeployableID, opts.Key, opts.Value = 42, "changed", "changed"
	extension := map[string]any{"enabled": false, "tags": []string{"original"}}
	field := attributes.WithCreateField("future", extension)
	extension["tags"].([]string)[0] = "changed"
	a := attributes.New(cloud.Client("accelerator", "/v2"))
	for i := 0; i < 2; i++ {
		value, err := a.Create(context.Background(), opts, typed, field, attributes.WithCreateHeader("X-Attribute-Extension", "enabled"))
		if err != nil || value.UUID != attributeUUID1 || value.StatusCode != 201 || value.Header.Get("X-Request-Id") != "req-create-attribute" || string(value.Body["future"]) != "null" {
			t.Fatalf("create: %+v/%v", value, err)
		}
	}
	if calls.Load() != 2 || opts.DeployableID != 42 || extension["tags"].([]string)[0] != "changed" {
		t.Fatal("caller options changed")
	}
}

func TestAcceleratorAttributeSingletonResponseShapes(t *testing.T) {
	object := fmt.Sprintf(`{"uuid":%q,"deployable_id":17,"key":"k","value":"v","future":false}`, attributeUUID1)
	for name, body := range map[string]string{"flat": object, "singular": `{"attribute":` + object + `}`, "plural": `{"attributes":[` + object + `]}`} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2/attributes", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 201, body) })
			value, err := attributes.New(cloud.Client("accelerator", "/v2")).Create(context.Background(), attributes.CreateOpts{DeployableID: 17, Key: "k", Value: "v"})
			if err != nil || value.UUID != attributeUUID1 || string(value.Body["future"]) != "false" || value.StatusCode != 201 {
				t.Fatalf("singleton: %+v/%v", value, err)
			}
		})
	}
	for name, body := range map[string]string{
		"null": `null`, "array": `[` + object + `]`, "scalar": `17`, "missing UUID": `{"key":"k"}`,
		"null singular": `{"attribute":null}`, "array singular": `{"attribute":[]}`,
		"empty plural": `{"attributes":[]}`, "multiple plural": `{"attributes":[` + object + `,` + object + `]}`,
		"null plural": `{"attributes":null}`, "object plural": `{"attributes":` + object + `}`,
		"null item": `{"attributes":[null]}`, "two envelopes": `{"attribute":` + object + `,"attributes":[` + object + `]}`,
		"wrong deployable type": fmt.Sprintf(`{"uuid":%q,"deployable_id":"17"}`, attributeUUID1),
	} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v2/attributes", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 201, body) })
			if value, err := attributes.New(cloud.Client("accelerator", "/v2")).Create(context.Background(), attributes.CreateOpts{}); err == nil || value != nil {
				t.Fatalf("invalid singleton accepted: %+v/%v", value, err)
			}
		})
	}
}

func TestAcceleratorAttributeValidationAndHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/attributes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Request-Id", "req-attribute-denied")
		testcloud.JSON(w, 403, `{"error":"forbidden"}`)
	})
	a := attributes.New(cloud.Client("accelerator", "/v2"))
	for _, key := range []string{"deployable_id", "key", "value"} {
		if _, err := a.Create(context.Background(), attributes.CreateOpts{}, attributes.WithCreateField(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("core field %s: %v", key, err)
		}
	}
	for _, header := range []string{"X-AUTH-TOKEN", "OpenStack-API-Version", "CONTENT-TYPE", "Host"} {
		if _, err := a.Create(context.Background(), attributes.CreateOpts{}, attributes.WithCreateHeader(header, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("core header %s: %v", header, err)
		}
	}
	if _, err := a.Create(context.Background(), attributes.CreateOpts{}, attributes.WithCreateField("future", make(chan int))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.Get(context.Background(), "not-uuid"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := a.Delete(context.Background(), resource.ID("one,two")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Create(ctx, attributes.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid inputs made HTTP requests")
	}
	_, err := a.Create(context.Background(), attributes.CreateOpts{DeployableID: 17, Key: "k", Value: "v"})
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 || response.ResponseHeader.Get("X-Request-Id") != "req-attribute-denied" || string(response.Body) != `{"error":"forbidden"}` {
		t.Fatalf("HTTP cause lost: %v", err)
	}
}

func TestAcceleratorAttributeWaitDeletedStableIDAndContext(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, deletes atomic.Int32
	cloud.Mux.HandleFunc("/v2/attributes/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/attributes/"+attributeUUID1 {
			t.Errorf("response UUID changed poll target: %s", r.URL.Path)
		}
		if r.Method == "DELETE" {
			deletes.Add(1)
			testcloud.JSON(w, 404, `{"error":"gone"}`)
			return
		}
		if gets.Add(1) > 1 {
			testcloud.JSON(w, 404, `{"error":"gone"}`)
			return
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":%q,"deployable_id":17,"key":"same","value":"v"}`, attributeUUID2))
	})
	a := attributes.New(cloud.Client("accelerator", "/v2"))
	if err := a.WaitDeleted(context.Background(), resource.ID(attributeUUID1), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil || gets.Load() != 2 {
		t.Fatalf("wait: %v/%d", err, gets.Load())
	}
	if err := a.Delete(context.Background(), resource.ID(attributeUUID1)); err != nil {
		t.Fatal(err)
	}
	if err := a.Delete(context.Background(), resource.ID(attributeUUID1), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.WaitDeleted(ctx, resource.ID(attributeUUID1)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if gets.Load() != 2 || deletes.Load() != 2 {
		t.Fatal("canceled waiter sent HTTP")
	}
	cloud.Mux.HandleFunc("/v2/pending/attributes/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, fmt.Sprintf(`{"uuid":%q,"deployable_id":17,"key":"pending"}`, attributeUUID1))
	})
	pending := attributes.New(cloud.Client("accelerator", "/v2/pending"))
	if err := pending.WaitDeleted(context.Background(), resource.ID(attributeUUID1), resource.WithTimeout(15*time.Millisecond), resource.WithPollInterval(time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait timeout: %v", err)
	}
	cloud.Mux.HandleFunc("/v2/cancel/attributes/", func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	if _, err := attributes.New(cloud.Client("accelerator", "/v2/cancel")).Get(ctx, attributeUUID1); !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("in-flight cancellation: %v", err)
	}
}
