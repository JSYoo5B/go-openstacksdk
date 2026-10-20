package servers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeServerAddressesSinglePageStreams(t *testing.T) {
	ctx := context.Background()
	t.Run("all networks and one network", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeServerCall
		nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path == "/nova/v2.1/servers/s1/ips" {
				return nativeServerWire(200, `{"addresses":{"net-a":[{"addr":"10.0.0.2","version":4}],"net-b":[{"addr":"fd00::2","version":6}]}}`)
			}
			return nativeServerWire(200, `{"net-a":[{"addr":"10.0.0.2","version":4},{"addr":"10.0.0.3","version":4}]}`)
		})
		api := servers.New(nativeServerClient(cloud))
		var pages []map[string][]servers.Address
		for value, err := range api.ListAddresses(ctx, "s1") {
			if err != nil {
				t.Fatal(err)
			}
			pages = append(pages, value)
		}
		want := map[string][]servers.Address{"net-a": {{Version: 4, Address: "10.0.0.2"}}, "net-b": {{Version: 6, Address: "fd00::2"}}}
		if len(pages) != 1 || !reflect.DeepEqual(pages[0], want) {
			t.Fatal(pages)
		}
		var rows []string
		for value, err := range api.ListAddressesByNetwork(ctx, "s1", "net-a") {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, value.Address)
		}
		if !reflect.DeepEqual(rows, []string{"10.0.0.2", "10.0.0.3"}) {
			t.Fatal(rows)
		}
		wantCalls := []nativeServerCall{
			{http.MethodGet, "/nova/v2.1/servers/s1/ips", "", ""},
			{http.MethodGet, "/nova/v2.1/servers/s1/ips/net-a", "", ""},
		}
		if !reflect.DeepEqual(calls, wantCalls) {
			t.Fatal(calls)
		}
	})
	t.Run("empty objects yield nothing", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			if req.URL.Path == "/nova/v2.1/servers/s1/ips" {
				return nativeServerWire(200, `{"addresses":{}}`), nil
			}
			return nativeServerWire(200, `{"net-a":[]}`), nil
		})
		api := servers.New(nativeServerClient(cloud))
		for value, err := range api.ListAddresses(ctx, "s1") {
			t.Fatal(value, err)
		}
		for value, err := range api.ListAddressesByNetwork(ctx, "s1", "net-a") {
			t.Fatal(value, err)
		}
		if requests.Load() != 2 {
			t.Fatal(requests.Load())
		}
	})
	t.Run("bodyless 204 surfaces the native EOF", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeServerWire(204, ""), nil
		})
		api := servers.New(nativeServerClient(cloud))
		// The pager decodes JSON before IsEmpty sees the 204 status.
		var errs []error
		for _, err := range api.ListAddresses(ctx, "s1") {
			errs = append(errs, err)
		}
		for _, err := range api.ListAddressesByNetwork(ctx, "s1", "net-a") {
			errs = append(errs, err)
		}
		if len(errs) != 2 || !errors.Is(errs[0], io.EOF) || !errors.Is(errs[1], io.EOF) || requests.Load() != 2 {
			t.Fatal(errs, requests.Load())
		}
	})
	t.Run("native pager status", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeServerWire(404, `{}`), nil
		})
		api := servers.New(nativeServerClient(cloud))
		for _, seq := range []func(func(any, error) bool){
			func(yield func(any, error) bool) {
				for value, err := range api.ListAddresses(ctx, "s1") {
					if !yield(value, err) {
						return
					}
				}
			},
			func(yield func(any, error) bool) {
				for value, err := range api.ListAddressesByNetwork(ctx, "s1", "net-a") {
					if !yield(value, err) {
						return
					}
				}
			},
		} {
			var errs []error
			for _, err := range seq {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if len(errs) != 1 || !errors.As(errs[0], &native) || native.Actual != 404 || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
				t.Fatal(errs)
			}
		}
		if requests.Load() != 2 {
			t.Fatal(requests.Load())
		}
	})
}

func TestNativeServerConsoleOutput(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls []nativeServerCall
	nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
		return nativeServerWire(200, `{"output":"line1\nline2"}`)
	})
	api := servers.New(nativeServerClient(cloud))
	output, err := api.ShowConsoleOutput(ctx, "s1", servers.ShowConsoleOutputOpts{Length: 50}, servers.WithShowConsoleOutputField("x_extension", 1))
	if err != nil || output != "line1\nline2" {
		t.Fatal(output, err)
	}
	if _, err := api.ShowConsoleOutput(ctx, "s1", servers.ShowConsoleOutputOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []nativeServerCall{
		{http.MethodPost, nativeServerActionPath, "", `{"os-getConsoleOutput":{"length":50,"x_extension":1}}`},
		// Without a length every line is requested through an empty action object.
		{http.MethodPost, nativeServerActionPath, "", `{"os-getConsoleOutput":{}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				return nativeServerWire(code, `{"output":"x"}`), nil
			})
			_, err := servers.New(nativeServerClient(cloud)).ShowConsoleOutput(ctx, "s1", servers.ShowConsoleOutputOpts{})
			nativeServerOperation(t, err, "ShowConsoleOutput")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || requests.Load() != 1 {
				t.Fatal(err)
			}
		})
	}
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeServerWire(200, `{}`), nil
		})
		api := servers.New(nativeServerClient(cloud))
		for name, err := range map[string]error{
			"length collision": func() error {
				_, err := api.ShowConsoleOutput(ctx, "s1", servers.ShowConsoleOutputOpts{}, servers.WithShowConsoleOutputField("length", 1))
				return err
			}(),
			"nil option": func() error {
				_, err := api.ShowConsoleOutput(ctx, "s1", servers.ShowConsoleOutputOpts{}, nil)
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeServerOperation(t, err, "ShowConsoleOutput")
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

func TestNativeServerWaitForStatus(t *testing.T) {
	waitAPI := func(t *testing.T, reply func(n int32) *http.Response) (*servers.API, *atomic.Int32) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			n := requests.Add(1)
			if req.Method != http.MethodGet || req.URL.Path != "/nova/v2.1/servers/s1" {
				t.Error(req.Method, req.URL.Path)
			}
			return reply(n), nil
		})
		return servers.New(nativeServerClient(cloud)), &requests
	}
	t.Run("first GET matches without a tick", func(t *testing.T) {
		api, requests := waitAPI(t, func(int32) *http.Response {
			return nativeServerWire(203, `{"server":{"id":"s1","status":"ACTIVE"}}`)
		})
		if err := api.WaitForStatus(context.Background(), "s1", "ACTIVE"); err != nil || requests.Load() != 1 {
			t.Fatal(err, requests.Load())
		}
	})
	t.Run("polls each second until the exact status", func(t *testing.T) {
		api, requests := waitAPI(t, func(n int32) *http.Response {
			if n == 1 {
				return nativeServerWire(200, `{"server":{"id":"s1","status":"BUILD"}}`)
			}
			return nativeServerWire(200, `{"server":{"id":"s1","status":"ACTIVE"}}`)
		})
		started := time.Now()
		if err := api.WaitForStatus(context.Background(), "s1", "ACTIVE"); err != nil || requests.Load() != 2 || time.Since(started) < 900*time.Millisecond {
			t.Fatal(err, requests.Load(), time.Since(started))
		}
	})
	t.Run("ERROR is not terminal and the context ends the wait", func(t *testing.T) {
		api, requests := waitAPI(t, func(int32) *http.Response {
			return nativeServerWire(200, `{"server":{"id":"s1","status":"ERROR"}}`)
		})
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()
		err := api.WaitForStatus(ctx, "s1", "ACTIVE")
		nativeServerOperation(t, err, "WaitForStatus")
		if !errors.Is(err, context.DeadlineExceeded) || requests.Load() != 2 {
			t.Fatal(err, requests.Load())
		}
	})
	t.Run("GET error stops the wait", func(t *testing.T) {
		api, requests := waitAPI(t, func(int32) *http.Response { return nativeServerWire(404, `{}`) })
		err := api.WaitForStatus(context.Background(), "s1", "ACTIVE")
		nativeServerOperation(t, err, "WaitForStatus")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || !reflect.DeepEqual(native.Expected, []int{200, 203}) || requests.Load() != 1 {
			t.Fatal(err)
		}
	})
}
