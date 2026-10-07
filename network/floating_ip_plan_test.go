package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const plannedPort = `{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}`

func plannedPortRead(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"port":`+plannedPort+`}`) })
}

func TestFloatingIPPlanDefersOwnerAndPreservesConcreteSelection(t *testing.T) {
	for _, reuse := range []bool{true, false} {
		t.Run(fmt.Sprint(reuse), func(t *testing.T) {
			cloud := testcloud.New(t)
			var reads, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.URL.Query().Get("device_id") != "server" || r.URL.Query().Get("network_id") != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"ports":[`+plannedPort+`]}`)
			})
			plannedPortRead(t, cloud)
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				if got := floatingIPBody(t, r); !reflect.DeepEqual(got, map[string]any{"floating_network_id": "external", "port_id": "port", "fixed_ip_address": "10.0.0.10"}) {
					t.Error(got)
				}
				respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureReuse(reuse))
			if err != nil || reads.Load() != 1 || writes.Load() != 0 {
				t.Fatalf("err=%v reads=%d writes=%d", err, reads.Load(), writes.Load())
			}
			want := network.FloatingIPSelection{ServerID: "server", NetworkID: "external", PortNetworkID: "private", PortID: "port", FixedIPv4: "10.0.0.10"}
			if plan.Selection() != want {
				t.Fatal(plan.Selection())
			}
			copy := plan.Selection()
			copy.PortID = "replacement"
			result, err := ips.EnsurePrepared(context.Background(), plan)
			if reuse {
				if result != nil || !errors.Is(err, resource.ErrUnsupported) || writes.Load() != 0 {
					t.Fatalf("result=%+v err=%v writes=%d", result, err, writes.Load())
				}
			} else if err != nil || result == nil || !result.Allocated || result.Reused || result.FloatingIP.PortID != "port" || writes.Load() != 1 || reads.Load() != 1 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestFloatingIPPlanTypedAbsenceDoesNotHideSelectionOrHTTPErrors(t *testing.T) {
	for _, scenario := range []string{"no ports", "foreign ports", "no fixed match", "IPv6 only", "wrong NAT", "ambiguity", "port404", "late403", "no external"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("marker") != "" {
					http.Error(w, "late denied", 403)
					return
				}
				rows := plannedPort
				switch scenario {
				case "no ports":
					rows = ""
				case "foreign ports":
					rows = strings.ReplaceAll(plannedPort, `"server"`, `"other"`)
				case "IPv6 only":
					rows = strings.ReplaceAll(plannedPort, "10.0.0.10", "2001:db8::1")
				case "ambiguity":
					rows = strings.ReplaceAll(plannedPort, `{"ip_address":"10.0.0.10"}`, `{"ip_address":"10.0.0.10"},{"ip_address":"10.0.0.11"}`)
				}
				links := ""
				if scenario == "late403" {
					links = fmt.Sprintf(`,"ports_links":[{"rel":"next","href":%q}]`, cloud.Server.URL+"/v2.0/ports?marker=late")
				}
				testcloud.JSON(w, 200, `{"ports":[`+rows+`]`+links+`}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports/missing", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "missing port", 404) })
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"networks":[]}`) })
			cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"routers":[]}`) })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			request := ensureFloatingRequest()
			var options []network.EnsureFloatingIPOption
			switch scenario {
			case "no fixed match":
				options = append(options, network.WithEnsureFixedAddress("10.0.0.99"))
			case "wrong NAT":
				options = append(options, network.WithEnsureNATDestination(resource.ID("other")))
			case "port404":
				options = append(options, network.WithEnsurePort(resource.ID("missing")))
			case "no external":
				request.Network = resource.Ref{}
			}
			plan, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.PrepareEnsure(context.Background(), request, options...)
			if err == nil || plan.Selection() != (network.FloatingIPSelection{}) {
				t.Fatalf("plan=%+v err=%v", plan.Selection(), err)
			}
			var absence *network.FloatingIPPlanUnavailableError
			isAbsence := errors.As(err, &absence)
			want := map[string]network.FloatingIPPlanAbsence{"no ports": network.NoFloatingIPServerPorts, "foreign ports": network.NoFloatingIPServerPorts, "no fixed match": network.NoFloatingIPFixedMatch, "no external": network.NoFloatingIPExternalNetwork}[scenario]
			if isAbsence != (want != "") || isAbsence && absence.Reason != want {
				t.Fatalf("absence=%+v err=%v", absence, err)
			}
			if scenario == "ambiguity" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if scenario == "late403" || scenario == "port404" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || !strings.Contains(string(native.Body), map[string]string{"late403": "late denied", "port404": "missing port"}[scenario]) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFloatingIPPlanRechecksPortWithoutReselectionOrMutation(t *testing.T) {
	for _, field := range []string{"id", "device_id", "network_id", "fixed_ips", "404"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
				if field == "404" {
					http.Error(w, "gone", 404)
					return
				}
				var body map[string]any
				_ = json.Unmarshal([]byte(plannedPort), &body)
				body[field] = "changed"
				if field == "fixed_ips" {
					body[field] = []any{map[string]string{"ip_address": "10.0.0.99"}}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"port": body})
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureReuse(false))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(context.Background(), plan)
			if err == nil || result != nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if field != "404" {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || len(accepted.Body) == 0 {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFloatingIPPlanReusesAfterAllPagesAndWaitsForActualACTIVE(t *testing.T) {
	for _, scenario := range []string{"free revision0", "attached wins", "wait identity drift", "wait ERROR", "late list403"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			var lists, updates, gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("marker") == "" {
					w.Header().Set("Link", "<"+cloud.Server.URL+"/v2.0/floatingips?marker=next>; rel=\"next\"")
					testcloud.JSON(w, 200, `{"floatingips":[]}`)
					return
				}
				if scenario == "late list403" {
					http.Error(w, "late IP denied", 403)
					return
				}
				attached := ""
				if scenario == "attached wins" {
					attached = `,{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"ACTIVE"}`
				}
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"free","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.11","revision_number":0}`+attached+`]}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/free", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				if lists.Load() != 2 || r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(lists.Load(), r.Header)
				}
				if got := floatingIPBody(t, r); !reflect.DeepEqual(got, map[string]any{"port_id": "port", "fixed_ip_address": "10.0.0.10"}) {
					t.Error(got)
				}
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "DOWN", map[string]any{"id": "free"})
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
				n := gets.Add(1)
				status := "DOWN"
				if n > 1 {
					status = "ACTIVE"
				}
				extra := map[string]any{"id": r.PathValue("id")}
				if scenario == "wait identity drift" {
					extra["port_id"] = "other"
				}
				if scenario == "wait ERROR" {
					status = "ERROR"
				}
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", status, extra)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected allocation/deletion %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureProject("owner"), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond)))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(context.Background(), plan)
			if scenario == "late list403" {
				if err == nil || result != nil || updates.Load() != 0 {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				return
			}
			if result == nil || !result.Reused || result.Allocated {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if scenario == "wait identity drift" || scenario == "wait ERROR" {
				if err == nil || result.FloatingIP.Status != "DOWN" {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				return
			}
			if err != nil || result.FloatingIP.Status != "ACTIVE" || gets.Load() != 2 {
				t.Fatalf("result=%+v err=%v gets=%d", result, err, gets.Load())
			}
			if scenario == "attached wins" && updates.Load() != 0 {
				t.Fatal(updates.Load())
			}
		})
	}
}

type planTransport func(*http.Request) (*http.Response, error)

func (f planTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type planCloseBody struct {
	io.Reader
	close func() error
}

func (b planCloseBody) Close() error {
	if b.close != nil {
		return b.close()
	}
	return nil
}

func TestFloatingIPPlanAcceptedAllocationFailuresKeepEvidenceAndNoRetry(t *testing.T) {
	for _, scenario := range []string{"malformed", "close", "source close", "cancel close"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			client := cloud.Client("network", "/v2.0")
			cause := errors.New("accepted close failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var posts atomic.Int32
			original := http.DefaultTransport
			client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "POST" {
					return original.RoundTrip(r)
				}
				posts.Add(1)
				payload := `{"floatingip":{"id":"fip","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"DOWN"}}`
				if scenario == "malformed" {
					payload = `{"floatingip":`
				}
				body := planCloseBody{Reader: strings.NewReader(payload), close: func() error {
					switch scenario {
					case "close":
						return cause
					case "source close":
						client.Endpoint = cloud.Server.URL + "/changed/"
					case "cancel close":
						cancel(cause)
					}
					return nil
				}}
				return &http.Response{StatusCode: 201, Header: http.Header{"X-Plan-Evidence": {"accepted"}}, Body: body, Request: r}, nil
			})
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				t.Error("accepted failure retried")
				return nil
			}
			ips := network.New(client).FloatingIPs
			plan, err := ips.PrepareEnsure(ctx, ensureFloatingRequest(), network.WithEnsureReuse(false))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(ctx, plan)
			var evidence *resource.ResponseError
			if err == nil || result == nil || !result.Allocated || result.Reused || posts.Load() != 1 || !errors.As(err, &evidence) || evidence.StatusCode != 201 || evidence.Header.Get("X-Plan-Evidence") != "accepted" || len(evidence.Body) == 0 {
				t.Fatalf("result=%+v err=%v evidence=%+v posts=%d", result, err, evidence, posts.Load())
			}
			if scenario == "malformed" {
				if result.FloatingIP != nil {
					t.Fatal(result, err)
				}
			} else if result.FloatingIP == nil || result.FloatingIP.ID != "fip" || result.FloatingIP.PortID != "port" || result.FloatingIP.Status != "DOWN" {
				t.Fatal(result, err)
			}
			if scenario == "close" && !errors.Is(err, cause) || scenario == "cancel close" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}

func TestFloatingIPPlanSourceGuardStopsMutationRetryAndRejectsForeignPlans(t *testing.T) {
	cloud := testcloud.New(t)
	ensurePortFixture(t, cloud)
	plannedPortRead(t, cloud)
	client := cloud.Client("network", "/v2.0")
	var posts atomic.Int32
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); http.Error(w, "original503", 503) })
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		client.Endpoint = cloud.Server.URL + "/replacement/"
		return nil
	}
	ips := network.New(client).FloatingIPs
	plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureReuse(false))
	if err != nil {
		t.Fatal(err)
	}
	for _, foreign := range []network.FloatingIPPlan{{}, plan} {
		result, err := network.New(client).FloatingIPs.EnsurePrepared(context.Background(), foreign)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	}
	result, err := ips.EnsurePrepared(context.Background(), plan)
	var native gophercloud.ErrUnexpectedResponseCode
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || posts.Load() != 1 {
		t.Fatalf("result=%+v err=%v posts=%d", result, err, posts.Load())
	}
	client.Endpoint = cloud.Server.URL + "/v2.0/"
	if _, err := ips.EnsurePrepared(context.Background(), plan); !errors.Is(err, resource.ErrInvalidOption) || posts.Load() != 1 {
		t.Fatal(err, posts.Load())
	}
}
