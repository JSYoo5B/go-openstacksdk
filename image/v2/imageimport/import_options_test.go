package imageimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"gophercloudsdk/resource"
)

func TestImportOptionsOwnedSnapshotsReplacementAndConcurrentReuse(t *testing.T) {
	store, enabled := "old", false
	extension := map[string]any{"number": json.Number("9007199254740993"), "nested": []any{"before"}}
	value := ImportOpts{Store: &store, AllStoresMustSucceed: &enabled, Headers: map[string]string{"x-note": "before"},
		Fields: map[string]any{"vendor": extension}, MethodFields: map[string]any{"extra": []string{"before"}}}
	option := WithImportOpts(value)
	store, enabled, value.Headers["x-note"] = "after", true, "after"
	extension["nested"].([]any)[0] = "after"
	value.MethodFields["extra"].([]string)[0] = "after"
	var captured *ImportOpts
	config, err := parseImportOpts([]ImportOption{option, func(value *ImportOpts) error { captured = value; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	captured.Store = nil
	captured.Headers["x-note"] = "captured"
	captured.Fields["vendor"] = "captured"
	if *config.Store != "old" || *config.AllStoresMustSucceed || config.Headers["X-Note"] != "before" {
		t.Fatalf("lost owned scalar/header snapshots: %#v", config)
	}
	body, err := json.Marshal(importBody(config))
	if err != nil || !strings.Contains(string(body), `9007199254740993`) || strings.Contains(string(body), "after") || strings.Contains(string(body), "captured") {
		t.Fatalf("lost extension snapshot: %s, %v", body, err)
	}
	replaced, err := parseImportOpts([]ImportOption{option, WithImportOpts(ImportOpts{}), WithImportAllStores(false), WithImportURI("unused"), WithImportURI("")})
	if err != nil || replaced.Store != nil || len(replaced.Fields) != 0 || replaced.AllStores == nil || *replaced.AllStores || replaced.Method != GlanceDirectMethod {
		t.Fatalf("replacement/last-wins: %#v, %v", replaced, err)
	}
	var group sync.WaitGroup
	errorsCh := make(chan error, 12)
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			cfg, err := parseImportOpts([]ImportOption{option})
			if err != nil || cfg.Store == nil || *cfg.Store != "old" || cfg.Headers["X-Note"] != "before" {
				errorsCh <- fmt.Errorf("reuse: %#v, %v", cfg, err)
				return
			}
			cfg.Headers["X-Note"], cfg.Fields["vendor"] = "private", "private"
		}()
	}
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
}

func TestImportOptionsMethodBodyAndOwnedNamespaces(t *testing.T) {
	cases := []struct {
		name string
		opts []ImportOption
		want string
	}{
		{"default", nil, `{"method":{"name":"glance-direct"}}`},
		{"false-root", []ImportOption{WithImportAllStores(false), WithImportAllStoresMustSucceed(false)}, `{"all_stores":false,"all_stores_must_succeed":false,"method":{"name":"glance-direct"}}`},
		{"remote-default-interface", []ImportOption{WithImportMethod(GlanceDownloadMethod), WithImportRemoteRegion("region"), WithImportRemoteImageID("not-a-route-id")}, `{"method":{"glance_image_id":"not-a-route-id","glance_region":"region","name":"glance-download"}}`},
		{"unknown-method", []ImportOption{WithImportMethod("operator-method"), WithImportMethodField("vendor", json.Number("9007199254740993")), WithImportField("root", nil)}, `{"method":{"name":"operator-method","vendor":9007199254740993},"root":null}`},
		{"plural-stores", []ImportOption{WithImportStores("one", "two")}, `{"method":{"name":"glance-direct"},"stores":["one","two"]}`},
		{"singular-no-body", []ImportOption{WithImportStore("one")}, `{"method":{"name":"glance-direct"}}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseImportOpts(test.opts)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(importBody(config))
			if err != nil || string(body) != test.want {
				t.Fatalf("body %s, want %s: %v", body, test.want, err)
			}
		})
	}
	invalid := [][]ImportOption{
		{nil}, {WithImportMethod(WebDownloadMethod)}, {WithImportURI("url")},
		{WithImportMethod(GlanceDownloadMethod), WithImportRemoteRegion("one")},
		{WithImportRemoteImageID("one")}, {WithImportRemoteServiceInterface("public")},
		{WithImportStore("one"), WithImportStores("two")}, {WithImportStore("one"), WithImportAllStores(true)},
		{WithImportStores("one"), WithImportAllStores(true)}, {WithImportField("AllStores", false)},
		{WithImportField("METHOD", map[string]any{})}, {WithImportMethodField("remote_region", "x")},
		{WithImportMethodField("Glance-Image-ID", "x")}, {WithImportField("bad", func() {})},
		{WithImportOpts(ImportOpts{Fields: map[string]any{"stores": nil}})},
		{WithImportStore("one\r\nX-Auth-Token: other")}, {WithImportURI(string([]byte{255}))},
		{WithImportStore("")}, {WithImportStores("valid", "")},
	}
	for index, opts := range invalid {
		if _, err := parseImportOpts(opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("invalid case %d: %v", index, err)
		}
	}
}

func TestImportOptionsHeaderAuthorityAndCanonicalLastWins(t *testing.T) {
	config, err := parseImportOpts([]ImportOption{WithImportHeaders(map[string]string{"x-note": "first", "X-NOTE": "first"}), WithImportHeader("X-Note", "last")})
	if err != nil || len(config.Headers) != 1 || config.Headers["X-Note"] != "last" {
		t.Fatalf("headers: %#v, %v", config.Headers, err)
	}
	for _, key := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Type", "Content-Length", "Transfer-Encoding", "Connection", "OpenStack-API-Version", "X-OpenStack-Glance-API-Version", "X-Image-Meta-Store"} {
		if _, err := parseImportOpts([]ImportOption{WithImportHeader(key, "blocked")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("caller authority %s: %v", key, err)
		}
	}
	for _, value := range []map[string]string{
		{"X-Note": "one", "x-note": "two"}, {"invalid key": "one"}, {"X-Note": "one\nother"},
	} {
		if _, err := parseImportOpts([]ImportOption{WithImportHeaders(value)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("invalid headers %#v: %v", value, err)
		}
	}
	if _, err := importHeaders(map[string]string{"openstack-api-version": "image 2.8"}, true, "2.8"); err != nil {
		t.Fatal(err)
	}
	if _, err := importHeaders(map[string]string{"openstack-api-version": "image 2.7"}, true, "2.8"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
