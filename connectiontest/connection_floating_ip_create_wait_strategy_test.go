package connectiontest_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestFloatingIPCreateWaitStrategyPacesNeutronPolls(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	var mu sync.Mutex
	var polls []time.Time
	state.reply = func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "networks/net") {
			return 200, `{"network":{"id":"net"}}`
		}
		if r.Method == "POST" {
			return 201, `{"floatingip":{"id":"new"}}`
		}
		mu.Lock()
		polls = append(polls, time.Now())
		n := len(polls)
		mu.Unlock()
		if n < 4 {
			return 200, `{"floatingips":[{"id":"new","status":"DOWN","port_id":"port"}]}`
		}
		return 200, `{"floatingips":[{"id":"new","status":"ACTIVE","port_id":"port"}]}`
	}
	ms := time.Millisecond
	result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true),
		compute.WithFloatingIPCreateWaitTimeout(5*time.Second), compute.WithFloatingIPCreateWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
	if err != nil || !result.Waited || len(polls) != 4 {
		t.Fatal(result, err, len(polls))
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := polls[i+1].Sub(polls[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
