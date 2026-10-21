package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const (
	r2SmallSegment = "aaaaaaaa-1111-4111-8111-111111111111"
	r2SmallHost    = "bbbbbbbb-2222-4222-8222-222222222222"
	r2SmallNotif   = "cccccccc-3333-4333-8333-333333333333"
)

type r2SmallPages struct {
	mu      sync.Mutex
	queries []url.Values
}

func (p *r2SmallPages) record(r *http.Request) url.Values {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queries = append(p.queries, r.URL.Query())
	return r.URL.Query()
}

func (p *r2SmallPages) markers() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, q := range p.queries {
		out = append(out, q.Get("marker")+"|"+q.Get("limit"))
	}
	return out
}

func r2SmallPageHandler(t *testing.T, pages *r2SmallPages, key string, rows map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := pages.record(r)
		body, ok := rows[q.Get("marker")]
		if !ok {
			t.Errorf("unexpected marker %q", q.Get("marker"))
			body = `[]`
		}
		testcloud.JSON(w, 200, `{"`+key+`":`+body+`}`)
	}
}

func TestR2SmallInstanceHASegmentsMarkerFallbackUsesWireID(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	client.Microversion = "1.2"
	pages := &r2SmallPages{}
	cloud.Mux.HandleFunc("/v1/tenant/segments", r2SmallPageHandler(t, pages, "segments", map[string]string{
		"":  `[{"uuid":"` + r2SmallSegment + `","id":1,"name":"a"},{"uuid":"` + r2SmallHost + `","id":2,"name":"b"}]`,
		"2": `[{"uuid":"` + r2SmallNotif + `","id":3,"name":"c"}]`,
		"3": `[]`,
	}))
	limit := 2
	api := segments.New(client)
	all, err := api.AllMarkerFallback(context.Background(), segments.WithListOptions(segments.ListOpts{Limit: &limit}))
	if err != nil || len(all) != 3 || all[2].UUID != r2SmallNotif {
		t.Fatalf("all=%v err=%v", all, err)
	}
	if got := pages.markers(); !reflect.DeepEqual(got, []string{"|2", "2|2", "3|2"}) {
		t.Fatalf("queries=%v", got)
	}
}

func TestR2SmallInstanceHAMarkerFallbackNeedsExplicitLimitAndPrefersLinks(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	pages := &r2SmallPages{}
	cloud.Mux.HandleFunc("/v1/tenant/notifications", func(w http.ResponseWriter, r *http.Request) {
		q := pages.record(r)
		if q.Get("marker") == "8" {
			testcloud.JSON(w, 200, `{"notifications":[]}`)
			return
		}
		if q.Get("marker") == "link" {
			testcloud.JSON(w, 200, `{"notifications":[{"notification_uuid":"`+r2SmallNotif+`","id":8}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"notifications":[{"notification_uuid":"`+r2SmallSegment+`","id":7}],"notifications_links":[{"rel":"next","href":"?marker=link&limit=1"}]}`)
	})
	limit := 1
	values, err := notifications.New(client).AllMarkerFallback(context.Background(), notifications.WithListOptions(notifications.ListOpts{Limit: &limit}))
	if err != nil || len(values) != 2 {
		t.Fatalf("values=%d err=%v", len(values), err)
	}
	if got := pages.markers(); len(got) != 3 || got[0] != "|1" || got[1] != "link|1" || got[2] != "8|1" {
		t.Fatalf("queries=%v", got)
	}
	// Without a limit, a single page without a next link ends the list.
	pages = &r2SmallPages{}
	cloud.Mux.HandleFunc("/v1/tenant/segments", func(w http.ResponseWriter, r *http.Request) {
		pages.record(r)
		testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+r2SmallSegment+`","id":1}]}`)
	})
	if values, err := segments.New(client).AllMarkerFallback(context.Background()); err != nil || len(values) != 1 || len(pages.queries) != 1 {
		t.Fatalf("values=%v err=%v queries=%v", values, err, pages.queries)
	}
}

func TestR2SmallInstanceHAHostsAndVMovesMarkerFallbackStringAndMissingIDs(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	client.Microversion = "1.3"
	pages := &r2SmallPages{}
	cloud.Mux.HandleFunc("/v1/tenant/segments/"+r2SmallSegment+"/hosts", r2SmallPageHandler(t, pages, "hosts", map[string]string{
		"":           `[{"uuid":"` + r2SmallHost + `","id":"host-row-1","name":"h1"}]`,
		"host-row-1": `[]`,
	}))
	limit := 1
	scope, err := hosts.New(client).InSegment(context.Background(), resource.ID(r2SmallSegment))
	if err != nil {
		t.Fatal(err)
	}
	values, err := scope.AllMarkerFallback(context.Background(), hosts.WithListOptions(hosts.ListOpts{Limit: &limit}))
	if err != nil || len(values) != 1 || values[0].SegmentID != r2SmallSegment {
		t.Fatalf("values=%v err=%v", values, err)
	}
	cloud.Mux.HandleFunc("/v1/tenant/notifications/"+r2SmallNotif+"/vmoves", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"vmoves":[{"uuid":"`+r2SmallHost+`","name":"no-id"}]}`)
	})
	vscope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(r2SmallNotif))
	if err != nil {
		t.Fatal(err)
	}
	_, err = vscope.AllMarkerFallback(context.Background(), vmoves.WithListOptions(vmoves.ListOpts{Limit: &limit}))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing wire id must fail instead of repeating the page: %v", err)
	}
}

func TestR2SmallInstanceHAMarkerFallbackBreakAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	var calls int
	cloud.Mux.HandleFunc("/v1/tenant/segments", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+r2SmallSegment+`","id":1}]}`)
	})
	limit := 1
	for _, err := range segments.New(client).ListMarkerFallback(context.Background(), segments.WithListOptions(segments.ListOpts{Limit: &limit})) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if calls != 1 {
		t.Fatalf("break must not fetch the fallback page: %d", calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := segments.New(client).AllMarkerFallback(ctx); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestR2SmallInstanceHASegmentAttributesPreserveNullFalseAndOmission(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	client.Microversion = "1.2"
	var bodies []map[string]json.RawMessage
	var methods []string
	handler := func(status int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			methods = append(methods, r.Method)
			var body map[string]map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body["segment"])
			testcloud.JSON(w, status, `{"segment":{"uuid":"`+r2SmallSegment+`","id":1,"description":null}}`)
		}
	}
	cloud.Mux.HandleFunc("/v1/tenant/segments", handler(202))
	cloud.Mux.HandleFunc("/v1/tenant/segments/"+r2SmallSegment, handler(200))
	api := segments.New(client)
	_, err := api.CreateAttributes(context.Background(), segments.Attributes{
		Name: request.Present("a"), Description: request.Null[string](), Enabled: request.Present(false),
	}, segments.WithAttributeField("custom", 1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.UpdateAttributes(context.Background(), resource.ID(r2SmallSegment), segments.Attributes{Description: request.Null[string](), ServiceType: request.Present("")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(methods, []string{"POST", "PUT"}) {
		t.Fatal(methods)
	}
	if string(bodies[0]["description"]) != "null" || string(bodies[0]["enabled"]) != "false" || string(bodies[0]["custom"]) != "1" || bodies[0]["recovery_method"] != nil || len(bodies[0]) != 4 {
		t.Fatalf("create=%v", bodies[0])
	}
	if string(bodies[1]["description"]) != "null" || string(bodies[1]["service_type"]) != `""` || len(bodies[1]) != 2 {
		t.Fatalf("update=%v", bodies[1])
	}
}

func TestR2SmallInstanceHASegmentAttributesRejectBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	var calls int
	cloud.Mux.HandleFunc("/v1/tenant/segments", func(w http.ResponseWriter, r *http.Request) { calls++ })
	api := segments.New(client)
	cases := []struct {
		name  string
		attrs segments.Attributes
		opts  []segments.AttributeOption
	}{
		{"empty", segments.Attributes{}, nil},
		{"protected id", segments.Attributes{Name: request.Present("a")}, []segments.AttributeOption{segments.WithAttributeField("uuid", "x")}},
		{"core collision", segments.Attributes{Name: request.Present("a")}, []segments.AttributeOption{segments.WithAttributeField("description", "x")}},
		{"reserved header", segments.Attributes{Name: request.Present("a")}, []segments.AttributeOption{segments.WithAttributeHeader("X-Auth-Token", "x")}},
		{"enabled below 1.2", segments.Attributes{Enabled: request.Present(true)}, nil},
	}
	for _, tc := range cases {
		if _, err := api.CreateAttributes(context.Background(), tc.attrs, tc.opts...); err == nil {
			t.Errorf("%s: want error", tc.name)
		}
	}
	if calls != 0 {
		t.Fatalf("HTTP sent for rejected input: %d", calls)
	}
}

func TestR2SmallInstanceHAHostUpdateAttributes(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1/tenant")
	var body map[string]map[string]json.RawMessage
	cloud.Mux.HandleFunc("/v1/tenant/segments/"+r2SmallSegment+"/hosts/"+r2SmallHost, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method=%s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		testcloud.JSON(w, 200, `{"host":{"uuid":"`+r2SmallHost+`","name":"h","on_maintenance":false}}`)
	})
	scope, err := hosts.New(client).InSegment(context.Background(), resource.ID(r2SmallSegment))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scope.UpdateAttributes(context.Background(), resource.ID(r2SmallHost), hosts.UpdateAttributes{
		Name: request.Null[string](), OnMaintenance: request.Present(false),
		ControlAttributes: request.Present(json.RawMessage(`{"ssh":{"port":22}}`)),
	})
	if err != nil || value.UUID != r2SmallHost || value.StatusCode != 200 {
		t.Fatalf("value=%v err=%v", value, err)
	}
	h := body["host"]
	if string(h["name"]) != "null" || string(h["on_maintenance"]) != "false" || string(h["control_attributes"]) != `{"ssh":{"port":22}}` || len(h) != 3 {
		t.Fatalf("body=%v", h)
	}
	if _, err := scope.UpdateAttributes(context.Background(), resource.ID(r2SmallHost), hosts.UpdateAttributes{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty update must fail before HTTP: %v", err)
	}
	if _, err := scope.UpdateAttributes(context.Background(), resource.ID(r2SmallHost), hosts.UpdateAttributes{Reserved: request.Present(true)}, hosts.WithAttributeField("segment_id", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
}
