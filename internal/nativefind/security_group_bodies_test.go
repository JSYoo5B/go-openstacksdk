package nativefind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestNativeSecurityGroupBodiesOwnQueryRowsAndLiveSource(t *testing.T) {
	cloud := testcloud.New(t)
	c := cloud.Client("network", "/unused")
	c.ResourceBase = cloud.Server.URL + "/reverse/v2/"
	c.MoreHeaders = map[string]string{"X-Source": "kept"}
	c.Microversion = "2.9"
	q := url.Values{"fields": {"id", "security_group_rules"}, "shared": {"false"}, "status": {"wire-only"}, "name": nil, "vendor": {"", "one", "two"}}
	frozen := q.Encode()
	var calls, middleware atomic.Int32
	base := c.ProviderClient.HTTPClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.ProviderClient.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { middleware.Add(1); return base.RoundTrip(r) })
	cloud.Mux.HandleFunc("GET /reverse/v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		query := r.URL.Query()
		marker := query.Get("marker")
		query.Del("marker")
		if query.Encode() != frozen || r.Header.Get("X-Source") != "kept" || r.Header.Get("OpenStack-API-Version") != "network 2.9" {
			t.Error(r.URL, r.Header)
		}
		id, next, token := "first", cloud.Server.URL+"/reverse/v2/security-groups?"+frozen+"&marker=next", "test-token"
		if marker != "" {
			id, next, token = "second", "", "renewed"
		}
		if r.Header.Get("X-Auth-Token") != token {
			t.Error(r.Header)
		}
		row := fmt.Sprintf(`{"id":%q,"name":"group","revision_number":7,"security_group_rules":[{"id":"rule","port_range_min":80,"port_range_max":80,"vendor":{"n":9007199254740993}},null],"created_at":"2025-01-02T03:04:05+00:00","updated_at":"2025-01-03T03:04:05+00:00"}`, id)
		testcloud.JSON(w, 200, `{"security_groups":[`+row+`],"security_groups_links":[{"rel":"next","href":`+fmt.Sprintf("%q", next)+`}]}`)
	})
	seq := nativefind.IterateSecurityGroupBodies(context.Background(), c, q, resource.ListControl{})
	q["fields"][0] = "changed"
	q["shared"] = []string{"true"}
	q["name"] = []string{"invented"}
	var records []*resource.BodyRecord[groups.SecGroup]
	for record, e := range seq {
		if e != nil {
			t.Fatal(e)
		}
		if record.Value.RevisionNumber != 7 || len(record.Value.Rules) != 2 || record.Value.Rules[1].ID != "" || record.Value.CreatedAt.IsZero() || string(record.Fields["created_at"]) != `"2025-01-02T03:04:05+00:00"` || !strings.Contains(string(record.Fields["security_group_rules"]), "9007199254740993") {
			t.Fatal(record)
		}
		records = append(records, record)
		if record.Value.ID == "first" {
			c.ProviderClient.SetToken("renewed")
		}
	}
	if len(records) != 2 || calls.Load() != 2 || middleware.Load() != 2 || records[1].Value.ID != "second" {
		t.Fatal(records, calls.Load(), middleware.Load())
	}
	records[0].Fields["security_group_rules"][0] = '!'
	records[0].Value.Rules[0].ID = "changed"
	if !json.Valid(records[1].Fields["security_group_rules"]) || records[1].Value.Rules[0].ID != "rule" {
		t.Fatal("rows share backing values")
	}
	if c.Endpoint != cloud.Server.URL+"/unused/" || c.ResourceBase != cloud.Server.URL+"/reverse/v2/" || c.MoreHeaders["X-Source"] != "kept" || c.ProviderClient != cloud.Provider {
		t.Fatal("source changed", c)
	}
}

func TestNativeSecurityGroupBodiesControlsAndWholePageDecoding(t *testing.T) {
	for _, mode := range []string{"cap", "first-page", "break", "null-row", "empty-object", "nested-port", "nested-time", "group-bool", "group-time"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := cloud.Client("network", "/v2")
			var calls atomic.Int32
			rows := `{"id":"first","security_group_rules":[null]},null,{}`
			nativeFailure := false
			switch mode {
			case "null-row":
				rows = `null`
			case "empty-object":
				rows = `{}`
			case "nested-port":
				rows = `{"id":"first"},{"id":"bad","security_group_rules":[{"port_range_min":"80"}]}`
				nativeFailure = true
			case "nested-time":
				rows = `{"id":"first"},{"id":"bad","security_group_rules":[{"created_at":"2025-01-02T03:04:05","updated_at":"2025-01-03T03:04:05Z"}]}`
				nativeFailure = true
			case "group-bool":
				rows = `{"id":"first"},{"id":"bad","stateful":"false"}`
				nativeFailure = true
			case "group-time":
				rows = `{"id":"first"},{"id":"bad","created_at":"bad-time"}`
				nativeFailure = true
			}
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Has("limit") {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"security_groups":[`+rows+`],"security_groups_links":[{"rel":"next","href":"http://[bad"}]}`)
			})
			control := resource.ListControl{MaxItems: 1}
			if mode == "first-page" {
				control = resource.ListControl{SinglePage: true}
			}
			seen := 0
			var terminal error
			for record, e := range nativefind.IterateSecurityGroupBodies(context.Background(), c, nil, control) {
				if e != nil {
					if record != nil {
						t.Fatal(record, e)
					}
					terminal = e
					break
				}
				seen++
				if mode == "null-row" && record.Fields != nil {
					t.Fatal(record)
				}
				if mode == "empty-object" && (record.Fields == nil || len(record.Fields) != 0) {
					t.Fatal(record)
				}
				if mode == "break" {
					break
				}
			}
			if nativeFailure {
				if terminal == nil || seen != 0 {
					t.Fatal(seen, terminal)
				}
				if mode == "nested-port" || mode == "group-bool" {
					var cause *json.UnmarshalTypeError
					if !errors.As(terminal, &cause) {
						t.Fatal(terminal)
					}
				}
			} else {
				want := 1
				if mode == "first-page" {
					want = 3
				}
				if terminal != nil || seen != want {
					t.Fatal(seen, terminal)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestNativeSecurityGroupBodiesNativeContinuationCodesAndErrors(t *testing.T) {
	for _, mode := range []string{"complete", "late-http", "cycle", "foreign", "wrong-links", "204", "json-204", "201", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud, foreign := testcloud.New(t), testcloud.New(t)
			c := cloud.Client("network", "/v2")
			var calls, followed atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { followed.Add(1); w.WriteHeader(500) })
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if mode == "204" || mode == "json-204" {
					if mode == "json-204" {
						w.Header().Set("Content-Type", "application/json")
					}
					w.WriteHeader(204)
					return
				}
				if mode == "201" {
					w.WriteHeader(201)
					return
				}
				if r.URL.Query().Has("marker") {
					if mode == "late-http" {
						w.WriteHeader(404)
						return
					}
					testcloud.JSON(w, 200, `{"security_groups":[{"id":"second"}]}`)
					return
				}
				next := cloud.Server.URL + "/v2/security-groups?marker=next"
				if mode == "cycle" {
					next = cloud.Server.URL + r.URL.String()
				}
				if mode == "foreign" {
					next = foreign.Server.URL + "/foreign"
				}
				if mode == "wrong-links" {
					testcloud.JSON(w, 300, `{"security_groups":[{"id":"first"}],"next":`+fmt.Sprintf("%q", next)+`}`)
					return
				}
				testcloud.JSON(w, 300, `{"security_groups":[{"id":"first"}],"security_groups_links":[{"rel":"next","href":`+fmt.Sprintf("%q", next)+`}]}`)
			})
			seen := 0
			var terminal error
			for record, e := range nativefind.IterateSecurityGroupBodies(ctx, c, nil, resource.ListControl{}) {
				if e != nil {
					terminal = e
					break
				}
				if record == nil {
					t.Fatal(record, e)
				}
				seen++
				if mode == "cancel" {
					cancel()
				}
			}
			switch mode {
			case "complete":
				if seen != 2 || terminal != nil {
					t.Fatal(seen, terminal)
				}
			case "wrong-links":
				if seen != 1 || terminal != nil {
					t.Fatal(seen, terminal)
				}
			case "204":
				if seen != 0 || terminal != nil {
					t.Fatal(seen, terminal)
				}
			default:
				if terminal == nil {
					t.Fatal(seen, terminal)
				}
				if mode == "cycle" && !errors.Is(terminal, resource.ErrPaginationCycle) || mode == "cancel" && !errors.Is(terminal, context.Canceled) || mode == "json-204" && !errors.Is(terminal, io.EOF) || mode == "201" && !gophercloud.ResponseCodeIs(terminal, 201) {
					t.Fatal(terminal)
				}
			}
			wantCalls := int32(1)
			if mode == "complete" || mode == "late-http" {
				wantCalls = 2
			}
			if calls.Load() != wantCalls || (mode == "foreign" && followed.Load() != 1) {
				t.Fatal(calls.Load(), followed.Load())
			}
		})
	}
}

func TestNativeSecurityGroupBodiesLazyPreflightAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	c := cloud.Client("network", "/v2")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "security_group_rules"}) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"security_groups":[{"id":"one","security_group_rules":[{"id":"rule"}]}]}`)
	})
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		ctx context.Context
		c   *gophercloud.ServiceClient
	}{{nil, c}, {canceled, c}, {context.Background(), nil}, {context.Background(), &gophercloud.ServiceClient{}}, {context.Background(), cloud.Client("network", "/v2?invalid=true")}} {
		seq := nativefind.IterateSecurityGroupBodies(tc.ctx, tc.c, nil, resource.ListControl{})
		if calls.Load() != 0 {
			t.Fatal("eager HTTP")
		}
		seen := 0
		for v, e := range seq {
			seen++
			if v != nil || e == nil {
				t.Fatal(v, e)
			}
		}
		if seen != 1 || calls.Load() != 0 {
			t.Fatal(seen, calls.Load())
		}
	}
	q := url.Values{"fields": {"id", "security_group_rules"}}
	seq := nativefind.IterateSecurityGroupBodies(context.Background(), c, q, resource.ListControl{})
	q["fields"][0] = "changed"
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seen := 0
			for v, e := range seq {
				if e != nil || v == nil || v.Value.ID != "one" {
					t.Error(v, e)
					return
				}
				seen++
				v.Fields["id"][0] = '!'
				v.Value.Rules[0].ID = "changed"
			}
			if seen != 1 {
				t.Error(seen)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 6 {
		t.Fatal(calls.Load())
	}
}
