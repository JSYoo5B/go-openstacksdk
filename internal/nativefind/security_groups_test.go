package nativefind_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeSecurityGroupsPayload(id, next string) string {
	links := "[]"
	if next != "" {
		links = fmt.Sprintf(`[{"rel":"next","href":%q}]`, next)
	}
	body := fmt.Sprintf(`{"security_groups":[{"id":%q,"name":"literal-name","description":"owned native model","stateful":false,"tenant_id":"tenant","project_id":"project","tags":["one","two"],"revision_number":7,"created_at":"2026-10-01T10:11:12Z","updated_at":"2026-10-02T10:11:12Z","security_group_rules":[{"id":"rule","security_group_id":%q,"direction":"ingress","ethertype":"IPv4","port_range_min":0,"port_range_max":0}]}],"security_groups_links":%s}`, id, id, links)
	if id == "first" {
		body = strings.ReplaceAll(body, "10:11:12Z", "10:11:12")
	}
	return body
}

func TestNativeSecurityGroupsPreserveQueriesModelsAndLiveSource(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/unused")
	client.ResourceBase = cloud.Server.URL + "/reverse/v2/"
	client.MoreHeaders = map[string]string{"X-Source": "retained"}
	client.Microversion = "2.9"
	provider := client.ProviderClient
	endpoint, base := client.Endpoint, client.ResourceBase
	input := url.Values{"name": {"a&b /한글"}, "fields": {"id", "name"}, "tag": {"one", "two"}, "nil": nil, "empty": {}, "vendor": {""}, "status": {"raw"}}
	frozen := input.Encode()
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		query := r.URL.Query()
		marker := query.Get("marker")
		query.Del("marker")
		if query.Encode() != frozen || r.Header.Get("X-Source") != "retained" || r.Header.Get("OpenStack-API-Version") != "network 2.9" {
			t.Error(r.URL, r.Header)
		}
		wantToken := "test-token"
		if call > 1 {
			wantToken = "renewed"
		}
		if r.Header.Get("X-Auth-Token") != wantToken {
			t.Error(r.Header)
		}
		id, next := "second", ""
		if marker == "" {
			id = "first"
			next = cloud.Server.URL + "/reverse/v2/security-groups?" + frozen + "&marker=next"
		} else if marker != "next" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, nativeSecurityGroupsPayload(id, next))
	})
	originalTransport := provider.HTTPClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	var middleware atomic.Int32
	provider.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		middleware.Add(1)
		return originalTransport.RoundTrip(r)
	})
	sequence := nativefind.IterateSecurityGroups(context.Background(), client, input, resource.ListControl{})
	input["fields"][0] = "mutated"
	input["name"] = []string{"changed"}
	delete(input, "status")
	var ids []string
	for value, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
		if value.Name != "literal-name" || value.Stateful || value.ProjectID != "project" || value.TenantID != "tenant" || value.RevisionNumber != 7 || !reflect.DeepEqual(value.Tags, []string{"one", "two"}) || len(value.Rules) != 1 || value.Rules[0].SecGroupID != value.ID || value.CreatedAt.Year() != 2026 || value.UpdatedAt.Day() != 2 {
			t.Fatal(value)
		}
		if value.ID == "first" {
			provider.SetToken("renewed")
		}
	}
	if !reflect.DeepEqual(ids, []string{"first", "second"}) || calls.Load() != 2 || middleware.Load() != 2 {
		t.Fatal(ids, calls.Load(), middleware.Load())
	}
	if client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != "2.9" || client.MoreHeaders["X-Source"] != "retained" {
		t.Fatal("source changed", client)
	}
}

func TestNativeSecurityGroupsLocalControlsStopBeforeContinuation(t *testing.T) {
	for _, control := range []resource.ListControl{{MaxItems: 1}, {SinglePage: true}} {
		t.Run(fmt.Sprint(control), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "" {
					t.Error("local cap added wire limit", r.URL)
				}
				testcloud.JSON(w, 200, `{"security_groups":[{"id":"first"},{"id":"second"}],"security_groups_links":[{"rel":"next","href":"http://[bad"}]}`)
			})
			count := 0
			for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, control) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			want := 1
			if control.SinglePage {
				want = 2
			}
			if count != want || calls.Load() != 1 {
				t.Fatal(count, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/v2")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, nativeSecurityGroupsPayload("row", "http://[bad"))
	})
	for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{}) {
		if err != nil {
			break
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
}

func TestNativeSecurityGroupsCodesAndWholePageErrors(t *testing.T) {
	for _, status := range []int{200, 204, 300, 203, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2")
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
				if status == 204 {
					w.WriteHeader(status)
					return
				}
				testcloud.JSON(w, status, nativeSecurityGroupsPayload("row", ""))
			})
			count := 0
			var result error
			for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{}) {
				if err != nil {
					result = err
					break
				}
				count++
			}
			if status == 200 || status == 300 {
				if result != nil || count != 1 {
					t.Fatal(count, result)
				}
			} else if status == 204 {
				if result != nil || count != 0 {
					t.Fatal(count, result)
				}
			} else if !gophercloud.ResponseCodeIs(result, status) || count != 0 {
				t.Fatal(count, result)
			}
		})
	}
	for _, body := range []string{`{"security_groups":[{"id":"valid"},{"id":"bad","stateful":"wrong"}]}`, `{"security_groups":[{"id":"valid"},{"id":"bad","security_group_rules":[{"port_range_min":"wrong"}]}]}`, `{"security_groups":[{"id":"bad","created_at":"invalid"}]}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2")
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			count := 0
			var result error
			for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{MaxItems: 1}) {
				if err != nil {
					result = err
					break
				}
				count++
			}
			if result == nil || count != 0 {
				t.Fatal(count, result)
			}
		})
	}
}

func TestNativeSecurityGroupsContinuationAndFailuresAreTerminal(t *testing.T) {
	for _, failure := range []string{"http", "decode", "cycle"} {
		t.Run(failure, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, nativeSecurityGroupsPayload("first", cloud.Server.URL+"/v2/security-groups?marker=next"))
					return
				}
				switch failure {
				case "http":
					testcloud.JSON(w, 403, `{"forbidden":"later"}`)
				case "decode":
					testcloud.JSON(w, 200, `{"security_groups":[{"stateful":"bad"}]}`)
				case "cycle":
					testcloud.JSON(w, 200, nativeSecurityGroupsPayload("second", cloud.Server.URL+"/v2/security-groups?marker=next"))
				}
			})
			count := 0
			var result error
			for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{}) {
				if err != nil {
					result = err
					break
				}
				count++
			}
			if result == nil || calls.Load() != 2 || count < 1 {
				t.Fatal(count, calls.Load(), result)
			}
			if failure == "http" && !gophercloud.ResponseCodeIs(result, 403) {
				t.Fatal(result)
			}
		})
	}
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/v2")
	sentinel := errors.New("original transport")
	var calls atomic.Int32
	client.ProviderClient.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { calls.Add(1); return nil, sentinel })
	var result error
	for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{}) {
		result = err
		break
	}
	if !errors.Is(result, sentinel) || calls.Load() != 1 {
		t.Fatal(calls.Load(), result)
	}
}

func TestNativeSecurityGroupsInvalidSourceContextAndControlBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	valid := cloud.Client("network", "/v2")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, nativeSecurityGroupsPayload("row", ""))
	})
	for _, client := range []*gophercloud.ServiceClient{nil, {}, {ProviderClient: valid.ProviderClient, Endpoint: "relative/"}, {ProviderClient: valid.ProviderClient, Endpoint: "http://user:pass@example.invalid/v2/"}, {ProviderClient: valid.ProviderClient, Endpoint: cloud.Server.URL + "/v2/?query=yes"}} {
		var result error
		for _, err := range nativefind.IterateSecurityGroups(context.Background(), client, nil, resource.ListControl{}) {
			result = err
			break
		}
		if !errors.Is(result, resource.ErrInvalidOption) {
			t.Fatal(client, result)
		}
	}
	for _, control := range []resource.ListControl{{MaxItems: -1}} {
		var result error
		for _, err := range nativefind.IterateSecurityGroups(context.Background(), valid, nil, control) {
			result = err
			break
		}
		if !errors.Is(result, resource.ErrInvalidOption) {
			t.Fatal(result)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, input := range []context.Context{nil, ctx} {
		var result error
		for _, err := range nativefind.IterateSecurityGroups(input, valid, nil, resource.ListControl{}) {
			result = err
			break
		}
		if input == nil {
			if !errors.Is(result, resource.ErrInvalidOption) {
				t.Fatal(result)
			}
		} else if !errors.Is(result, context.Canceled) {
			t.Fatal(result)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	client := cloud.Client("network", "/v2")
	client.ProviderClient.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { cancel(); return nil, r.Context().Err() })
	var result error
	for _, err := range nativefind.IterateSecurityGroups(ctx, client, nil, resource.ListControl{}) {
		result = err
		break
	}
	if !errors.Is(result, context.Canceled) {
		t.Fatal(result)
	}
}

func TestNativeSecurityGroupsConcurrentReuseHasOwnedRowsAndQuery(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/v2")
	cloud.Mux.HandleFunc("GET /v2/security-groups", func(w http.ResponseWriter, r *http.Request) {
		if !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, nativeSecurityGroupsPayload("row", ""))
	})
	query := url.Values{"fields": {"id", "name"}}
	sequence := nativefind.IterateSecurityGroups(context.Background(), client, query, resource.ListControl{})
	query["fields"][0] = "changed"
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for value, err := range sequence {
				if err != nil {
					t.Error(err)
					return
				}
				if len(value.Tags) != 2 || value.Tags[0] != "one" {
					t.Error(value)
				}
				value.Tags[0] = "private"
			}
		}()
	}
	wg.Wait()
}
