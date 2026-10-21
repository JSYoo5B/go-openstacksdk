package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type r2icCall struct {
	Method, Path, RawQuery, Body, Version string
}

type r2icRec struct {
	mu    sync.Mutex
	calls []r2icCall
}

func (r *r2icRec) all() []r2icCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]r2icCall(nil), r.calls...)
}

// r2icServe answers every request of a pattern from a script of statuses and
// bodies (the last entry repeats) and records the calls.
func r2icServe(t *testing.T, cloud *testcloud.Cloud, pattern string, script ...any) *r2icRec {
	t.Helper()
	rec := &r2icRec{}
	n := 0
	cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.calls = append(rec.calls, r2icCall{r.Method, r.URL.Path, r.URL.RawQuery, string(data), r.Header.Get("X-OpenStack-Ironic-API-Version")})
		i := n
		n++
		rec.mu.Unlock()
		if i*2+1 >= len(script) {
			i = len(script)/2 - 1
		}
		status, body := script[i*2].(int), script[i*2+1].(string)
		if body == "" {
			w.WriteHeader(status)
			return
		}
		testcloud.JSON(w, status, body)
	})
	return rec
}

func r2icClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	c := cloud.Client("baremetal", "/v1")
	c.Microversion = version
	return c
}

// r2icDiscovery serves the version document of the Ironic root.
func r2icDiscovery(t *testing.T, cloud *testcloud.Cloud, min, max string) *r2icRec {
	t.Helper()
	body := `{"version":{"id":"v1","min_version":"` + min + `","version":"` + max + `","status":"CURRENT","links":[{"rel":"self","href":"` + cloud.Server.URL + `/v1/"}]}}`
	rec := r2icServe(t, cloud, "GET /v1/{$}", 200, body)
	return rec
}

func r2icQuery(raw string) url.Values { v, _ := url.ParseQuery(raw); return v }

func r2icHas(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

type rawJSON string

func r2icRaw(in map[string]rawJSON) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(in))
	for k, v := range in {
		out[k] = json.RawMessage(v)
	}
	return out
}
