package cinderrequest

import (
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"strings"
	"sync"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type faultsState struct {
	mu    sync.Mutex
	cause error
}

func (f *faultsState) add(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	f.cause = errors.Join(f.cause, err)
	f.mu.Unlock()
}

func (f *faultsState) error() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cause
}

type observedBody struct {
	io.ReadCloser
	faults *faultsState
}

func (b *observedBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err != io.EOF {
		b.faults.add(err)
	}
	return n, err
}

func (b *observedBody) Close() error {
	err := b.ReadCloser.Close()
	b.faults.add(err)
	return err
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func versionHeader(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "openstack-api-version", "x-openstack-volume-api-version":
		return key, true
	}
	return "", false
}
