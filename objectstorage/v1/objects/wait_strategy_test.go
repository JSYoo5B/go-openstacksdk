package objects_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestObjectWaitStrategy(t *testing.T) {
	var mu sync.Mutex
	var seen []time.Time
	server := testcloud.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, time.Now())
		n := len(seen)
		mu.Unlock()
		w.Header().Set("Content-Length", "0")
		if n < 4 {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer server.Close()
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("wait-token")
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "object-store", Endpoint: server.URL + "/v1/AUTH_wait/"}
	api := objects.New(client)
	interval := time.Millisecond
	valid := resource.FixedWait(time.Millisecond)
	for name, option := range map[string]objects.ObjectWaitOption{
		"invalid": objects.WithObjectWaitStrategy(resource.LinearWait(time.Millisecond, -time.Millisecond, 0)),
		"zero":    objects.WithObjectWaitStrategy(resource.WaitStrategy{}),
		"both":    objects.WithObjectWaitOpts(objects.ObjectWaitOpts{Interval: &interval, WaitStrategy: &valid}),
	} {
		if _, err := api.WaitForDelete(context.Background(), "c", "o", option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(seen) != 0 {
		t.Fatal("HTTP before validation", len(seen))
	}
	ms := time.Millisecond
	result, err := api.WaitForDelete(context.Background(), "c", "o", objects.WithObjectWaitPollInterval(time.Hour), objects.WithObjectWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
	if err != nil || result == nil || len(seen) != 4 {
		t.Fatal(result, err, len(seen))
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := seen[i+1].Sub(seen[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
