package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type novaIPFixture struct {
	cloud   *testcloud.Cloud
	service *compute.Service
	fail    string
	mu      sync.Mutex
	events  []string
	target  string
	gets    int
	rows    string
	status  string
}

func novaIPRow(id, address, pool, instance string) string {
	return fmt.Sprintf(`{"id":%s,"ip":%q,"pool":%q,"instance_id":%s,"fixed_ip":null,"owner":"owner","vendor_number":9007199254740993}`, id, address, pool, instance)
}

func (f *novaIPFixture) event(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, value)
}
func (f *novaIPFixture) trace() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func newNovaIPFixture(t *testing.T, fail string) *novaIPFixture {
	t.Helper()
	f := &novaIPFixture{cloud: testcloud.New(t), fail: fail, status: "BUILD"}
	client := f.cloud.Client("compute", "/v2.1")
	client.Microversion = "2.35"
	f.service = compute.New(client, compute.Dependencies{})
	f.rows = novaIPRow("9007199254740993", dispatchAddresses["a"], "public", `"foreign"`) + "," + novaIPRow("2", dispatchAddresses["b"], "public", "null") + "," + novaIPRow("3", dispatchAddresses["c"], "public", "null")
	f.cloud.Mux.HandleFunc("GET /v2.1/os-floating-ip-pools", func(w http.ResponseWriter, r *http.Request) {
		f.event("pools")
		testcloud.JSON(w, 200, `{"floating_ip_pools":[{"name":"public"},{"name":"later"}]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/os-floating-ips", func(w http.ResponseWriter, r *http.Request) {
		f.event("list")
		if r.URL.RawQuery != "" {
			t.Error("Nova lookup pushed down filters", r.URL)
		}
		if f.fail == "list204" {
			w.WriteHeader(204)
			return
		}
		rows := f.rows
		if f.fail == "missing" {
			rows = novaIPRow("9007199254740993", dispatchAddresses["a"], "public", `"foreign"`)
		}
		w.Header().Set("X-Proof", "Nova-list")
		testcloud.JSON(w, 200, `{"floating_ips":[`+rows+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/os-floating-ips/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		f.event("get:" + id)
		if id == "29" && f.fail == "compatHTTP" {
			http.Error(w, "compat lookup denied", 403)
			return
		}
		address, instance := dispatchAddresses["b"], "null"
		if id == "9007199254740993" {
			address, instance = dispatchAddresses["a"], `"foreign"`
		}
		if id == "3" {
			address = dispatchAddresses["c"]
		}
		if id == "29" {
			address = dispatchAddresses["pool"]
		}
		row := novaIPRow(id, address, "public", instance)
		if f.fail == "getDrift" {
			row = strings.Replace(row, address, "198.51.100.99", 1)
		}
		w.Header().Set("X-Proof", "Nova-get-"+id)
		testcloud.JSON(w, 200, `{"floating_ip":`+row+`}`)
	})
	f.cloud.Mux.HandleFunc("POST /v2.1/os-floating-ips", func(w http.ResponseWriter, r *http.Request) {
		f.event("allocate")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]any{"pool": "public"}) {
			t.Error(body, err)
		}
		pool := "public"
		if f.fail == "allocatedWrongPool" {
			pool = "unexpected"
		}
		w.Header().Set("X-Proof", "Nova-allocated")
		testcloud.JSON(w, 200, `{"floating_ip":`+novaIPRow("29", dispatchAddresses["pool"], pool, "null")+`}`)
	})
	f.cloud.Mux.HandleFunc("POST /v2.1/servers/server/action", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		action := body["addFloatingIp"]
		if len(body) != 1 || len(action) != 2 || action["fixed_address"] != "10.0.0.10" {
			t.Error("incorrect owned action", body)
		}
		address := action["address"]
		f.event("attach:" + address)
		if f.fail == "actionHTTP" && address == dispatchAddresses["b"] {
			w.Header().Set("X-Proof", "Nova-denied")
			http.Error(w, "second attach denied", 403)
			return
		}
		f.mu.Lock()
		f.target = address
		f.mu.Unlock()
		w.Header().Set("X-Proof", "Nova-attached-"+address)
		w.WriteHeader(202)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		f.event("raw")
		f.mu.Lock()
		f.gets++
		target := f.target
		f.mu.Unlock()
		addresses := autoFixed
		if target != "" {
			addresses = fmt.Sprintf(`{"private":[{"version":4,"addr":"10.0.0.10","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":%q,"OS-EXT-IPS:type":"floating"}]}`, target)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"server","status":%q,"addresses":%s}}`, f.status, addresses))
	})
	return f
}

func novaIPOptions(extra ...compute.AutomaticFloatingIPOption) []compute.ServerIPOption {
	common := []compute.AutomaticFloatingIPOption{compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNova), compute.WithAddressReachability(false)), compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10")), compute.WithAutomaticIPPollInterval(time.Millisecond)}
	return []compute.ServerIPOption{compute.WithServerIPAutomaticOptions(append(common, extra...)...)}
}

func TestNovaServerIPOrderedAsyncAndRawOnlyObservation(t *testing.T) {
	for _, wait := range []bool{false, true} {
		for _, status := range []string{"BUILD", "ERROR"} {
			t.Run(fmt.Sprintf("%t-%s", wait, status), func(t *testing.T) {
				f := newNovaIPFixture(t, "")
				f.status = status
				server := automaticServer(t, "null")
				server.Status = "SHUTOFF"
				options := append(novaIPOptions(), compute.WithServerIPWait(wait))
				result, err := f.service.AddIPList(context.Background(), server, []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["a"]}, options...)
				if err != nil || result == nil || result.Decision.Backend != compute.FloatingIPNova || result.Assignment != nil || result.NovaAssignment == nil || len(result.Attempts) != 3 || len(result.Decision.NovaSelections) != 3 || result.Observed != wait {
					t.Fatal(result, err, f.trace())
				}
				var expected []string
				for _, pair := range [][2]string{{"9007199254740993", dispatchAddresses["a"]}, {"2", dispatchAddresses["b"]}, {"9007199254740993", dispatchAddresses["a"]}} {
					expected = append(expected, "list", "get:"+pair[0], "attach:"+pair[1])
					if wait {
						expected = append(expected, "raw")
					}
				}
				if !reflect.DeepEqual(f.trace(), expected) {
					t.Fatal(f.trace(), expected)
				}
				for _, attempt := range result.Attempts {
					a := attempt.NovaAssignment
					if attempt.Assignment != nil || !attempt.Completed || attempt.Observed != wait || a == nil || !a.Reused || a.Allocated || !a.ActionAccepted || a.ActionResponse.StatusCode != 202 || len(a.ActionResponse.Body) != 0 || a.FloatingIP.StatusCode != 200 || string(a.FloatingIP.Body["vendor_number"]) != "9007199254740993" {
						t.Fatal(attempt)
					}
				}
				// The raw allocation model remains its pre-action association.
				if instance := result.Attempts[0].NovaAssignment.FloatingIP.InstanceID; instance == nil || *instance != "foreign" {
					t.Fatal(instance)
				}
			})
		}
	}
}

func TestNovaServerIPPoolReuseAndFreshAllocation(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		f := newNovaIPFixture(t, "")
		options := novaIPOptions(compute.WithFloatingIPPool(resource.Name("public")), compute.WithFloatingIPAddresses("ignored bad address"), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(reuse)))
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null"), Network: resource.ID("ignored/auto")}, options...)
		if err != nil || result == nil || result.Mode != compute.ServerIPPool || result.Assignment != nil || len(result.Attempts) != 1 || !result.Attempts[0].Completed || result.Observed || result.NovaAssignment.Reused != reuse || result.NovaAssignment.Allocated == reuse {
			t.Fatal(result, err, f.trace())
		}
		want := []string{"list", "get:2", "attach:" + dispatchAddresses["b"]}
		if !reuse {
			want = []string{"allocate", "get:29", "get:29", "attach:" + dispatchAddresses["pool"]}
		}
		if !reflect.DeepEqual(f.trace(), want) {
			t.Fatal(f.trace(), want)
		}
		if !reuse && (result.NovaAssignment.AllocationResponse.StatusCode != 200 || result.NovaAssignment.AllocationResponse.Header.Get("X-Proof") != "Nova-allocated") {
			t.Fatal(result.NovaAssignment)
		}
	}
}

func TestNovaServerIPSecondFailureKeepsHistoryAndCurrentModel(t *testing.T) {
	for _, wait := range []bool{false, true} {
		for _, fail := range []string{"missing", "actionHTTP"} {
			f := newNovaIPFixture(t, fail)
			result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]}, append(novaIPOptions(), compute.WithServerIPWait(wait))...)
			if err == nil || result == nil || len(result.Attempts) != 2 || !result.Attempts[0].Completed || result.Attempts[0].Observed != wait || result.Attempts[1].Completed || result.Attempts[1].Error == nil || result.Observed {
				t.Fatal(result, err, f.trace())
			}
			if fail == "missing" {
				if !errors.Is(err, resource.ErrNotFound) || result.NovaAssignment != result.Attempts[0].NovaAssignment {
					t.Fatal(result, err)
				}
			} else if result.NovaAssignment.FloatingIP.ID != "2" || result.NovaAssignment.ActionAccepted || result.NovaAssignment.ActionResponse != nil {
				t.Fatal(result, err)
			}
			if strings.Contains(strings.Join(f.trace(), ","), "get:3") || f.gets != map[bool]int{false: 0, true: 1}[wait] {
				t.Fatal(f.trace(), f.gets)
			}
		}
	}
}

func TestNovaServerIPAllocationAndCompatibilityFailureKeepPartial(t *testing.T) {
	for _, fail := range []string{"compatHTTP", "allocatedWrongPool", "getDrift"} {
		f := newNovaIPFixture(t, fail)
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, novaIPOptions(compute.WithFloatingIPPool(resource.Name("public")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false)))...)
		if err == nil || result == nil || result.NovaAssignment == nil || !result.NovaAssignment.Allocated || result.NovaAssignment.FloatingIP == nil || result.NovaAssignment.FloatingIP.ID != "29" || result.NovaAssignment.AllocationResponse.Header.Get("X-Proof") != "Nova-allocated" || result.NovaAssignment.ActionAccepted || len(result.Attempts) != 1 || result.Attempts[0].Completed {
			t.Fatal(result, err, f.trace())
		}
		if strings.Contains(strings.Join(f.trace(), ","), "attach:") {
			t.Fatal(f.trace())
		}
		if fail == "getDrift" {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Proof") != "Nova-get-29" {
				t.Fatal(err)
			}
			if result.NovaAssignment.FloatingIP.Address != dispatchAddresses["pool"] || result.NovaAssignment.FloatingIP.Pool != "public" {
				t.Fatal("drift response replaced original allocation", result.NovaAssignment)
			}
		}
		if fail == "allocatedWrongPool" && (result.NovaAssignment.FloatingIP.Pool != "unexpected" || !strings.Contains(string(result.NovaAssignment.AllocationResponse.Body), "unexpected")) {
			t.Fatal("actual allocation fields were replaced", result.NovaAssignment)
		}
	}
}

func TestNovaServerIPListAbsenceAndAssociationMetadataCannotAllocate(t *testing.T) {
	for _, fail := range []string{"list204", "missingInstance", "emptyInstance"} {
		f := newNovaIPFixture(t, fail)
		if fail == "missingInstance" {
			f.rows = strings.Replace(novaIPRow("2", dispatchAddresses["b"], "public", "null"), `"instance_id":null,`, "", 1)
		}
		if fail == "emptyInstance" {
			f.rows = novaIPRow("2", dispatchAddresses["b"], "public", `""`)
		}
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, novaIPOptions(compute.WithFloatingIPPool(resource.Name("public")))...)
		if fail == "emptyInstance" {
			if err != nil || !result.NovaAssignment.Allocated || !strings.Contains(strings.Join(f.trace(), ","), "allocate") {
				t.Fatal(result, err, f.trace())
			}
			continue
		}
		if err == nil || result.NovaAssignment != nil || !reflect.DeepEqual(f.trace(), []string{"list"}) {
			t.Fatal(result, err, f.trace())
		}
		if fail == "list204" {
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("204 was treated as absence", err)
			}
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Proof") != "Nova-list" {
				t.Fatal(err)
			}
		}
	}
}

func TestNovaServerIPVersionAndNeutronSelectorsFailBeforeHTTP(t *testing.T) {
	for _, scenario := range []string{"2.36", "2.44", "latest", "port", "NAT", "project"} {
		f := newNovaIPFixture(t, "")
		var extra []compute.AutomaticFloatingIPOption
		switch scenario {
		case "port":
			extra = append(extra, compute.WithAutomaticEnsureOptions(network.WithEnsurePort(resource.ID("port"))))
		case "NAT":
			extra = append(extra, compute.WithAutomaticEnsureOptions(network.WithEnsureNATDestination(resource.ID("private"))))
		case "project":
			extra = append(extra, compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner")))
		default:
			f.service.RawClient().Microversion = scenario
		}
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"]}, novaIPOptions(extra...)...)
		if err == nil || result == nil || len(result.Attempts) != 0 || len(f.trace()) != 0 || result.NovaAssignment != nil {
			t.Fatal(scenario, result, err, f.trace())
		}
		if scenario != "latest" && !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(scenario, err)
		}
	}
}

func TestNovaServerIPLocalSelectionIgnoresUnrelatedNullAddresses(t *testing.T) {
	f := newNovaIPFixture(t, "")
	f.rows = novaIPRow("9007199254740993", dispatchAddresses["a"], "public", `"foreign"`) + `,{"id":5,"ip":null,"pool":"other","instance_id":null}`
	result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"]}, novaIPOptions()...)
	if err != nil || result == nil || result.NovaAssignment.FloatingIP.ID != "9007199254740993" || !result.Attempts[0].Completed {
		t.Fatal(result, err, f.trace())
	}
	f = newNovaIPFixture(t, "")
	f.rows = `{"id":5,"ip":null,"pool":"other","instance_id":null},` + novaIPRow("2", dispatchAddresses["b"], "public", "null")
	result, err = f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "null")}, novaIPOptions(compute.WithFloatingIPPool(resource.Name("public")))...)
	if err != nil || result.NovaAssignment.FloatingIP.ID != "2" || result.NovaAssignment.Allocated {
		t.Fatal(result, err, f.trace())
	}
}
