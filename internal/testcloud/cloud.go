// Package testcloud provides an isolated HTTP fixture, never a live cloud.
package testcloud

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

type Cloud struct {
	Server   *httptest.Server
	Mux      *http.ServeMux
	Provider *gophercloud.ProviderClient
}

func (c *Cloud) Client(service, path string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: c.Provider, Type: service, Endpoint: gophercloud.NormalizeURL(c.Server.URL + path)}
}

func New(t *testing.T) *Cloud {
	t.Helper()
	mux := http.NewServeMux()
	server := NewServer(mux)
	t.Cleanup(server.Close)
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("test-token")
	return &Cloud{Server: server, Mux: mux, Provider: provider}
}

// NewServer starts an httptest server whose idle connections close without
// lingering. Each fixture server is short-lived, and the default close leaves
// a TIME_WAIT socket per connection; across the full suite that exhausts local
// ephemeral ports. Connections serving a request keep the default close, so a
// handler that truncates its body still produces a clean EOF, not a reset.
func NewServer(handler http.Handler) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		tcp, ok := conn.(*net.TCPConn)
		if !ok {
			return
		}
		switch state {
		case http.StateIdle:
			_ = tcp.SetLinger(0)
		case http.StateActive:
			_ = tcp.SetLinger(-1)
		}
	}
	server.Start()
	return server
}

func JSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
