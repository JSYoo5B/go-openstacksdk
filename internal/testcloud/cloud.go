// Package testcloud provides an isolated HTTP fixture, never a live cloud.
package testcloud

import (
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
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("test-token")
	return &Cloud{Server: server, Mux: mux, Provider: provider}
}

func JSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
