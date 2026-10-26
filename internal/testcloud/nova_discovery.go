package testcloud

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// AdvertiseNova serves a Nova v2.1 version document at the discovery path
// (ending in a slash) with the given advertised bounds. Empty bounds serve a
// clean 404, as a cloud without version discovery does. It returns a counter
// of discovery requests.
func (c *Cloud) AdvertiseNova(path, minimum, maximum string) *atomic.Int32 {
	var count atomic.Int32
	c.Mux.HandleFunc("GET "+path+"{$}", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if minimum == "" && maximum == "" {
			http.NotFound(w, r)
			return
		}
		JSON(w, 200, NovaVersionDocument(c.Server.URL+path, minimum, maximum))
	})
	return &count
}

// NovaVersionDocument is a single CURRENT v2.1 version document.
func NovaVersionDocument(self, minimum, maximum string) string {
	return fmt.Sprintf(`{"version":{"id":"v2.1","status":"CURRENT","min_version":%q,"version":%q,"links":[{"rel":"self","href":%q}]}}`, minimum, maximum, self)
}
