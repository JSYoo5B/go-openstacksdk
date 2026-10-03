package containers

import "net/http"

// ContainerResponse preserves an opaque lifecycle acknowledgement. A clean,
// explicitly allowed 404 has IgnoredMissing true; failed handling never does.
type ContainerResponse struct {
	Body           []byte
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}
