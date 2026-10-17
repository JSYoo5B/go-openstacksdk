package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudPortUpdateLookupOnlyAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"empty", `{"id":"p","name":"old"}`, `{}`, ``, true},
		{"same", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"old","admin_state_up":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"new","admin_state_up":true}`, `{"port":{"name":"new"}}`, false},
		{"raw-string-bool", `{"id":"p","admin_state_up":"false"}`, `{"admin_state_up":true}`, `{"port":{"admin_state_up":true}}`, false},
		{"null", `{"id":"p","name":"old"}`, `{"name":null}`, `{"port":{"name":null}}`, false},
		{"missing-null", `{"id":"p"}`, `{"device_id":null}`, `{"port":{"device_id":null}}`, false},
		{"explicit-false-empty", `{"id":"p","admin_state_up":true,"security_groups":["sg"]}`, `{"admin_state_up":false,"security_groups":[]}`, `{"port":{"admin_state_up":false,"security_groups":[]}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports/p": {{200, `{"port":` + tc.body + `}`}}, "PUT /v2.0/ports/p": {{200, `{"port":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdatePortRecord(context.Background(), "p", network.WithPortUpdateJSON(json.RawMessage(tc.attrs)))
			if err != nil || result == nil || result.Lookup == nil || result.NoOp != tc.noOp {
				t.Fatal(result, err)
			}
			got := calls()
			wantCount := 2
			if tc.noOp {
				wantCount = 1
			}
			if len(got) != wantCount {
				t.Fatal(got)
			}
			if tc.noOp {
				if result.Port != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Port == nil || result.Port.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudPortUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports/target": {{404, `{}`}}, "GET /v2.0/ports": {{200, `{"ports":` + rows + `}`}}, "PUT /v2.0/ports/actual": {{200, `{"port":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdatePortRecord(context.Background(), "target", network.WithPortUpdateName("new"))
			if found {
				if err != nil || result.Port == nil || len(calls()) != 3 {
					t.Fatal(result, err, calls())
				}
			} else {
				if !errors.Is(err, resource.ErrNotFound) || result.Lookup != nil || len(calls()) != 2 {
					t.Fatal(result, err, calls())
				}
			}
		})
	}
}
func TestCloudPortUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"name":"owned"}`)
	option := network.WithPortUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports/p": {{200, `{"port":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdatePortRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.PortUpdateOption{{nil}, {network.WithPortUpdateAttribute("is_admin_state_up", true)}, {network.WithPortUpdateAttribute("if_revision", 0)}, {network.WithPortUpdateAttribute("name", make(chan int))}, {network.WithPortUpdateJSON(json.RawMessage(`[]`))}, {network.WithPortUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdatePortRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdatePortRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdatePortRecord(context.Background(), "p", func(*network.PortUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudPortUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports/p": {{200, `{"port":{"id":"p","name":"old"}}`}}, "PUT /v2.0/ports/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdatePortRecord(context.Background(), "p", network.WithPortUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Port != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}
