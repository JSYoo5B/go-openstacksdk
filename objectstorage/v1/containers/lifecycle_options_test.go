package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestContainerLifecycleOptionsSnapshotsAndOverwrite(t *testing.T) {
	var calls atomic.Int32
	headers, metadata := map[string]string{"x-call": "factory"}, map[string]string{"Old": "old"}
	full := WithCreateContainerOpts(CreateContainerOpts{Headers: headers, Metadata: metadata})
	values := map[string]string{"Final": "literal"}
	replace := WithCreateContainerMetadata(values)
	headers["x-call"], metadata["Old"], values["Final"] = "changed", "changed", "changed"
	a, _ := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("X-Call") != "last" || r.Header.Get("X-Container-Meta-Final") != "literal" || r.Header.Get("X-Container-Meta-Old") != "" || r.Header.Get("X-Stage") != "kept" {
			t.Errorf("owned headers%v", r.Header)
		}
		return lifecycleWire(r, 201, io.NopCloser(strings.NewReader(""))), nil
	}))
	var retained *CreateContainerOpts
	callbacks := 0
	first := func(c *CreateContainerOpts) error {
		callbacks++
		if c.Headers == nil || c.Metadata == nil {
			t.Fatal("maps not initialized")
		}
		retained = c
		c.Headers["X-Stage"] = "kept"
		return nil
	}
	second := func(c *CreateContainerOpts) error {
		callbacks++
		retained.Headers["X-Stage"] = "outside"
		retained.Metadata["Final"] = "outside"
		return nil
	}
	_, err := a.CreateContainer(context.Background(), "name", full, WithCreateContainerHeader("X-Call", "last"), replace, first, second)
	if err != nil || callbacks != 2 || calls.Load() != 1 {
		t.Fatalf("snapshot err%v callbacks%d", err, callbacks)
	}
}

func TestContainerLifecycleOptionsReplacementAndDefaults(t *testing.T) {
	var calls atomic.Int32
	a, _ := lifecycleAPI(lifecycleStatic(404, "missing", &calls))
	value := false
	factory := WithDeleteContainerOpts(DeleteContainerOpts{Headers: map[string]string{"x-call": "old"}, IgnoreMissing: &value})
	value = true
	r, err := a.DeleteContainer(context.Background(), "name", factory, WithDeleteContainerHeader("X-Call", "new"))
	if r != nil || err == nil {
		t.Fatalf("factory pointer not owned %+v %v", r, err)
	}
	r, err = a.DeleteContainer(context.Background(), "name", factory, WithDeleteContainerIgnoreMissing(true))
	if err != nil || r == nil || !r.IgnoredMissing {
		t.Fatalf("last pointer %+v %v", r, err)
	}
	r, err = a.DeleteContainer(context.Background(), "name", WithDeleteContainerIgnoreMissing(false), WithDeleteContainerOpts(DeleteContainerOpts{}))
	if err != nil || r == nil || !r.IgnoredMissing {
		t.Fatalf("full reset %+v %v", r, err)
	}
	var retained *DeleteContainerOpts
	_, err = a.DeleteContainer(context.Background(), "name", WithDeleteContainerIgnoreMissing(false), func(c *DeleteContainerOpts) error { retained = c; return nil }, func(c *DeleteContainerOpts) error { *retained.IgnoreMissing = true; return nil })
	if err == nil {
		t.Fatal("retained pointer changed current config")
	}
	a, _ = lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Discard") != "" || r.Header.Get("X-Container-Meta-Discard") != "" {
			t.Error("full config failed replacement")
		}
		return lifecycleWire(r, 202, io.NopCloser(strings.NewReader(""))), nil
	}))
	_, err = a.CreateContainer(context.Background(), "name", WithCreateContainerOpts(CreateContainerOpts{Headers: map[string]string{"X-Discard": "old"}, Metadata: map[string]string{"Discard": "old"}}), WithCreateContainerOpts(CreateContainerOpts{}))
	if err != nil {
		t.Fatal(err)
	}
}

func TestContainerLifecycleOptionsInvalidInputs(t *testing.T) {
	var calls atomic.Int32
	a, _ := lifecycleAPI(lifecycleStatic(201, "", &calls))
	for _, values := range []map[string]string{{"": "bad"}, {"Foo": "one", "foo": "two"}, {"X-Container-Meta-Foo": "bad"}, {"K": "bad"}, {"K": "a\n"}, {"K": string([]byte{255})}} {
		_, err := a.CreateContainer(context.Background(), "name", WithCreateContainerMetadata(values))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("metadata%q err%v", values, err)
		}
	}
	for _, values := range []map[string]string{{"X-Call": "one", "x-call": "two"}, {"X-Container-Meta-K": "bad"}, {"X-Remove-Container-Meta-K": "bad"}, {"Authorization": "bad"}, {"X-Newest": "true"}, {"X-Call": "a\r"}} {
		_, err := a.CreateContainer(context.Background(), "name", WithCreateContainerHeaders(values))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("headers%q err%v", values, err)
		}
	}
	_, err := a.CreateContainer(context.Background(), "name", nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil create option")
	}
	_, err = a.DeleteContainer(context.Background(), "name", nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil delete option")
	}
	cause := errors.New("callback failed")
	callbacks := 0
	_, err = a.DeleteContainer(context.Background(), "name", func(c *DeleteContainerOpts) error { callbacks++; c.Headers["X-Call"] = "safe"; return cause }, func(*DeleteContainerOpts) error { callbacks++; return nil })
	if !errors.Is(err, cause) || callbacks != 1 || calls.Load() != 0 {
		t.Fatalf("error once%v callbacks%d calls%d", err, callbacks, calls.Load())
	}
}

func TestContainerLifecycleOptionsParallelReuse(t *testing.T) {
	var calls atomic.Int32
	a, _ := lifecycleAPI(lifecycleTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		code := 201
		if r.Header.Get("X-Call") != "shared" {
			t.Error("shared ordinary header lost")
		}
		if r.Method == "DELETE" {
			code = 404
			if r.Header.Get("X-Container-Meta-K") != "" {
				t.Error("create metadata leaked")
			}
		} else if r.Header.Get("X-Container-Meta-K") != "V" {
			t.Error("shared metadata lost")
		}
		return lifecycleWire(r, code, io.NopCloser(strings.NewReader("actual"))), nil
	}))
	create := WithCreateContainerOpts(CreateContainerOpts{Headers: map[string]string{"X-Call": "shared"}, Metadata: map[string]string{"K": "V"}})
	deleteOption := WithDeleteContainerOpts(DeleteContainerOpts{Headers: map[string]string{"X-Call": "shared"}})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := a.CreateContainer(context.Background(), "name", create)
			if err != nil || r == nil || r.StatusCode != 201 {
				t.Errorf("parallel create%v", err)
			}
			r, err = a.DeleteContainer(context.Background(), "name", deleteOption)
			if err != nil || r == nil || !r.IgnoredMissing {
				t.Errorf("parallel delete%v", err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 16 {
		t.Fatalf("parallel requests%d", calls.Load())
	}
}
