package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

// pyManilaBStub is a canned response for one "METHOD /path" key.
type pyManilaBStub struct {
	status int
	body   string
	// probe, when set, answers requests that carry an offset query.
	probe string
	// bodies answer successive requests in order; the last body repeats.
	bodies []string
}

// pyManilaBCall records one request received by the fixture.
type pyManilaBCall struct {
	method, path, query, body, version string
}

// pyManilaBCloud records every request and answers from stubs keyed by
// "METHOD /v2/path". Unknown routes answer 599 and fail the test.
type pyManilaBCloud struct {
	t      *testing.T
	cloud  *testcloud.Cloud
	mu     sync.Mutex
	calls  []pyManilaBCall
	stubs  map[string]pyManilaBStub
	counts map[string]int
}

func newPyManilaBCloud(t *testing.T, stubs map[string]pyManilaBStub) *pyManilaBCloud {
	t.Helper()
	c := &pyManilaBCloud{t: t, cloud: testcloud.New(t), stubs: stubs, counts: map[string]int{}}
	c.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.calls = append(c.calls, pyManilaBCall{r.Method, r.URL.Path, r.URL.RawQuery, string(data), r.Header.Get("X-Openstack-Manila-Api-Version")})
		key := r.Method + " " + r.URL.Path
		stub, ok := c.stubs[key]
		if ok && len(stub.bodies) > 0 {
			index := c.counts[key]
			if index >= len(stub.bodies) {
				index = len(stub.bodies) - 1
			}
			c.counts[key]++
			stub.body = stub.bodies[index]
		}
		c.mu.Unlock()
		if !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(599)
			return
		}
		if stub.probe != "" && r.URL.Query().Has("offset") {
			testcloud.JSON(w, 200, stub.probe)
			return
		}
		if stub.body == "" {
			w.WriteHeader(stub.status)
			return
		}
		testcloud.JSON(w, stub.status, stub.body)
	})
	return c
}

func (c *pyManilaBCloud) client(version string) *gophercloud.ServiceClient {
	client := c.cloud.Client("shared-file-system", "/v2")
	client.Microversion = version
	return client
}

func (c *pyManilaBCloud) snapshot() []pyManilaBCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]pyManilaBCall(nil), c.calls...)
}

// expect asserts the full request sequence: method, path, raw query, JSON body
// (empty means no body) and selected microversion header.
func (c *pyManilaBCloud) expect(want ...pyManilaBCall) {
	c.t.Helper()
	got := c.snapshot()
	if len(got) != len(want) {
		c.t.Fatalf("requests=%+v want %d", got, len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		gq, _ := url.ParseQuery(g.query)
		wq, _ := url.ParseQuery(w.query)
		if g.method != w.method || g.path != w.path || !reflect.DeepEqual(gq, wq) || g.version != w.version {
			c.t.Errorf("request %d = %+v want %+v", i, g, w)
		}
		if !pyManilaBJSONEqual(g.body, w.body) {
			c.t.Errorf("request %d body=%s want %s", i, g.body, w.body)
		}
	}
}

func pyManilaBJSONEqual(got, want string) bool {
	if got == "" || want == "" {
		return got == want
	}
	var a, b any
	if json.Unmarshal([]byte(got), &a) != nil || json.Unmarshal([]byte(want), &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
