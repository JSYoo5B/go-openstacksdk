// Package testcloud provides an isolated HTTP fixture, never a live cloud.
package testcloud

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
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
	fake := testhelper.SetupHTTP()
	t.Cleanup(fake.Teardown)
	provider := &gophercloud.ProviderClient{HTTPClient: *fake.Server.Client()}
	provider.UseTokenLock()
	provider.SetToken("test-token")
	return &Cloud{Server: fake.Server, Mux: fake.Mux, Provider: provider}
}

func JSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
